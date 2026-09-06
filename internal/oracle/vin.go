package oracle

import "strings"

// The rule this file implements.
//
// 49 CFR 565, the requirements the United States places on the vehicle
// identification number of every vehicle subject to its safety standards. It is
// a work of the federal government and carries no copyright under 17 U.S.C.
// 105, so the sentences below are held as the regulation writes them.
//
// Unlike the statistical number, nothing here rests on a measurement. The
// regulation states the shape in 565.13, the arithmetic in 565.15(c) and works
// an example in Table VI, so all three questions a source can be asked are
// answered by one document.

// vinAlphabetAsPublished is the set of 565.13(g), written as the regulation
// writes it.
//
// The provider holds the same set as a plain string of characters. This holds
// the brackets and the two groups, so a character dropped from one of them is
// not dropped from the other in the same slip of a finger.
const vinAlphabetAsPublished = "[ABCDEFGHJKLMNPRSTUVWXYZ] [0123456789]"

// vinValuesAsPublished is Table III, with each letter written next to its value.
//
// The provider stores twenty six numbers in an array, positional and unlabelled,
// where the value of a letter is found by counting. This spells the pairs out,
// so a value typed into the wrong slot there is not typed into the same wrong
// slot here, and a letter that fell out of the table is caught rather than
// silently shifting every letter after it.
const vinValuesAsPublished = "A 1 B 2 C 3 D 4 E 5 F 6 G 7 H 8 " +
	"J 1 K 2 L 3 M 4 N 5 P 7 R 9 " +
	"S 2 T 3 U 4 V 5 W 6 X 7 Y 8 Z 9"

// vinWeightsAsPublished is Table IV, with the position spelled out beside each
// weight.
//
// Same reason as the values above, and one more: the eighth weight is ten while
// every other is a single digit, and the ninth is nought. A table read as a run
// of single characters would lose both.
const vinWeightsAsPublished = "P1 8 P2 7 P3 6 P4 5 P5 4 P6 3 P7 2 P8 10 P9 0 " +
	"P10 9 P11 8 P12 7 P13 6 P14 5 P15 4 P16 3 P17 2"

// vinDigitForRemainder maps each of the eleven possible remainders onto the
// character that is written, from Table V.
//
// The provider says this with a condition and an arithmetic shortcut from the
// remainder to a byte. This says it with a table, on purpose. The two shapes
// disagree if either is wrong, whereas a second copy of the same condition
// would agree with the first even when both are mistaken. The interesting slot
// is the last one: eleven residues do not fit in ten digits, and this scheme
// resolves that with a letter, where the tax number refuses the number and the
// statistical number writes nought.
var vinDigitForRemainder = [11]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'X'}

// VINFinding is one reason a value is not a valid vehicle identification
// number. An empty list means the value is valid.
type VINFinding string

// CheckVIN reports every rule a value breaks.
//
// It answers the same question as the provider's Validate and deliberately not
// a wider one. The model year of Table VII is not checked here, because the
// provider does not enforce it either and an oracle that asked more would
// disagree with a value neither of them calls wrong.
func CheckVIN(value string) []VINFinding {
	alphabet := vinPublishedAlphabet()
	if len(alphabet) != 33 {
		// Loud rather than truncated. An alphabet that lost a character would
		// otherwise quietly refuse values the regulation allows.
		return []VINFinding{"the published alphabet did not come apart into thirty three characters"}
	}
	if len(value) != 17 {
		return []VINFinding{"a vehicle identification number is seventeen characters"}
	}
	for at := 0; at < len(value); at++ {
		if !strings.ContainsRune(alphabet, rune(value[at])) {
			return []VINFinding{"the value carries a character this scheme does not use"}
		}
	}

	values := vinPublishedValues()
	weights := vinPublishedWeights()
	if len(weights) != 17 {
		return []VINFinding{"the published weight table did not come apart into seventeen weights"}
	}

	total := 0
	for at := 0; at < len(value); at++ {
		total += vinAssigned(value[at], values) * weights[at]
	}

	var findings []VINFinding
	if want := vinDigitForRemainder[total%11]; value[8] != want {
		findings = append(findings, "the check digit does not agree with the other sixteen characters")
	}
	return findings
}

// vinAssigned is the value of one character: its own for a digit, the published
// one for a letter.
func vinAssigned(symbol byte, values map[byte]int) int {
	if symbol >= '0' && symbol <= '9' {
		return int(symbol - '0')
	}
	return values[symbol]
}

// vinPublishedAlphabet takes the two bracketed groups apart into the characters
// they hold.
func vinPublishedAlphabet() string {
	var built strings.Builder
	for _, symbol := range vinAlphabetAsPublished {
		if symbol != '[' && symbol != ']' && symbol != ' ' {
			built.WriteRune(symbol)
		}
	}
	return built.String()
}

// vinPublishedValues reads the letter and value pairs out of the sentence.
//
// A letter absent from the sentence is absent from the map, and a map answers
// nought for what it does not hold. That is safe here for the same reason it is
// safe in the provider: the regulation assigns nothing lower than one, so
// nought cannot be confused with an assigned value. The three letters with no
// value are I, O and Q, and a value carrying one of them was already refused by
// the alphabet above.
func vinPublishedValues() map[byte]int {
	fields := strings.Fields(vinValuesAsPublished)
	values := make(map[byte]int, len(fields)/2)
	for index := 0; index+1 < len(fields); index += 2 {
		letter, written := fields[index], fields[index+1]
		if len(letter) != 1 || len(written) != 1 {
			continue
		}
		if written[0] < '0' || written[0] > '9' {
			continue
		}
		values[letter[0]] = int(written[0] - '0')
	}
	return values
}

// vinPublishedWeights takes the sentence apart into the seventeen weights,
// reading the number that follows each position label.
func vinPublishedWeights() []int {
	fields := strings.Fields(vinWeightsAsPublished)
	weights := make([]int, 0, len(fields)/2)
	for index := 1; index < len(fields); index += 2 {
		weight := 0
		for at := 0; at < len(fields[index]); at++ {
			symbol := fields[index][at]
			if symbol < '0' || symbol > '9' {
				return nil
			}
			weight = weight*10 + int(symbol-'0')
		}
		weights = append(weights, weight)
	}
	return weights
}
