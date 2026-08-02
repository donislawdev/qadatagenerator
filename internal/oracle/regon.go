package oracle

import "strings"

// regonWeightsAsPublished is the weight table written as a sentence, with the
// position of each weight spelled out next to it.
//
// The provider stores eight bare numbers in an array. This keeps them apart and
// labelled, so a weight typed into the wrong slot there does not get typed into
// the same wrong slot here, and a table that lost a position is caught rather
// than quietly shortened.
const regonWeightsAsPublished = "P1 8 P2 9 P3 2 P4 3 P5 4 P6 5 P7 6 P8 7"

// regonDigitForRemainder maps each of the eleven possible remainders onto the
// digit that is written.
//
// The provider says this with a condition. This says it with a table, on
// purpose. The two shapes disagree if either is wrong, whereas a second copy of
// the same condition would agree with the first even when both are mistaken.
// The interesting slot is the last one: a remainder of eleven residues does not
// fit in ten digits, and this scheme resolves that by writing nought rather
// than by calling the number invalid, which is the opposite of what the Polish
// tax number does with the same collision.
var regonDigitForRemainder = [11]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0}

// The rule this file implements.
//
// Neither the regulation of 30 November 2015 nor the one of 27 July 1999 it
// replaced states the arithmetic, and neither does the glossary of the
// statistical office nor its public information bulletin. All four describe the
// shape only: nine digits, of which the first eight are a sequence number and
// the ninth is a check digit, carrying no hidden or open meaning about the
// entity.
//
// So the arithmetic below rests on a measurement instead of a quotation. The
// weights reproduce the check digit of 174 real numbers taken from two
// unrelated public registers without a single exception, and fifteen of those
// numbers land on the folded remainder and all fifteen carry nought. That is
// written here rather than dressed up as a citation, because a validator whose
// provenance is overstated is worse than one whose limits are known.

// REGONFinding is one reason a value is not a valid REGON. An empty list means
// the value is valid.
type REGONFinding string

// CheckREGON reports every rule a value breaks.
func CheckREGON(value string) []REGONFinding {
	digits, ok := regonDigits(value)
	if !ok {
		return []REGONFinding{"the value is not nine bare digits"}
	}

	weights := regonPublishedWeights()
	if len(weights) != 8 {
		// Loud rather than truncated. A weight table that lost a position would
		// otherwise agree with a provider that lost the same one.
		return []REGONFinding{"the published weight table did not come apart into eight weights"}
	}

	total := 0
	for position, weight := range weights {
		total += digits[position] * weight
	}

	var findings []REGONFinding
	if want := regonDigitForRemainder[total%11]; digits[8] != want {
		findings = append(findings, "the check digit does not agree with the first eight digits")
	}
	return findings
}

// regonPublishedWeights takes the sentence apart into the eight weights,
// reading the number that follows each position label.
func regonPublishedWeights() []int {
	fields := strings.Fields(regonWeightsAsPublished)
	weights := make([]int, 0, len(fields)/2)
	for index := 1; index < len(fields); index += 2 {
		if len(fields[index]) != 1 || fields[index][0] < '0' || fields[index][0] > '9' {
			return nil
		}
		weights = append(weights, int(fields[index][0]-'0'))
	}
	return weights
}

// regonDigits reads the nine digits, refusing anything else.
//
// Separators are not accepted in any position. The tax number is written with
// hyphens on invoices and this one is not, so there is no second shape to
// carry.
func regonDigits(value string) ([]int, bool) {
	if len(value) != 9 {
		return nil, false
	}
	digits := make([]int, 0, 9)
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return nil, false
		}
		digits = append(digits, int(value[index]-'0'))
	}
	return digits, true
}
