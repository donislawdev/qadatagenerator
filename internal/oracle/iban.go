package oracle

import (
	"strconv"
	"strings"
)

// The international bank account number, checked against a normative act that
// states the whole rule in words.
//
// # Why this one is a quotation where the routing number was a measurement
//
// The arithmetic of this scheme lives in ISO 7064 and ISO 13616, which are sold
// rather than published, and that is where the search would have stopped. It did
// not, because a source is rarely one: regulation 5/2002 of the President of the
// National Bank of Poland restates the whole calculation in Polish, and a
// normative act is outside copyright under article 4 point 1 of the Polish
// copyright act, so it may be quoted without limit.
//
// The act is repealed and the two that replaced it dropped these annexes. That
// costs nothing here. A rule does not stop being the rule it was when the
// document stating it is superseded, and the successor did not change it, it
// merely stopped repeating it.
//
// # Why this validator uses the other method of the two the act gives
//
// The act offers a simplified method and a polynomial method and calls them
// equivalent. The provider implements the first: divide, subtract the remainder
// from ninety eight. This implements the second: multiply each position by its
// weight and check the total.
//
// That is a stronger kind of independence than this project usually gets. The
// two are not one description written twice by one hand. They are two methods
// the source itself set out, so agreement between them means the arithmetic is
// right and not merely that it was copied consistently.
//
// # The one thing the act gets wrong, which must not be copied
//
// It calls twenty five and twenty one the decimal ASCII values of P and L. They
// are not: ASCII puts P at eighty and L at seventy six. They are the letter
// values of ISO 13616, where A is ten and Z is thirty five, which puts the
// sixteenth letter at twenty five and the twelfth at twenty one.
//
// The numbers are right and the name for them is wrong. Anybody taking the
// justification literally and computing ASCII for another country would break
// every value. It is quoted as it stands, because a corrected quotation is not a
// quotation, and it is contradicted here rather than left to be believed.
const (
	// ibanRearrangementSentence is annex 2, section 2 point 1 of regulation
	// 5/2002 of the President of the National Bank of Poland, Dz. Urz. NBP 2002
	// no. 8 item 26. It is the sentence that fixes how many characters move.
	//
	// In English: the two letter country code PL is added to the right of the
	// national number and the check number CC then moves from the start to the
	// end, while in the international form the first four characters, PLCC,
	// move from the start to the end.
	ibanRearrangementSentence = "do NRB z prawej strony dodaje się 2-literowy kod Polski (PL), " +
		"a następnie liczbę kontrolną CC przenosi się z początku na koniec, zaś w IBAN " +
		"cztery pierwsze znaki, tj. „PLCC\", przenosi się z początku na koniec"

	// ibanLetterValuesSentence is annex 2, section 2 point 2 of the same act.
	// It is where the letter values come from, under the wrong name for them.
	//
	// In English: in the resulting string the letters PL are replaced by numeric
	// characters, putting in their place the corresponding decimal values of the
	// ASCII code, that is P=25, L=21.
	ibanLetterValuesSentence = "w otrzymanym ciągu litery „PL\" zamienia się na znaki numeryczne, " +
		"podstawiając w ich miejsce odpowiadające im dziesiętne wartości kodu ASCII, tj. P=25, L=21"

	// ibanRemainderSentence is annex 2, section 2 point 4 of the same act, and
	// it is the accepting condition.
	//
	// In English: a remainder of one from the division means the check number is
	// verified as correct.
	ibanRemainderSentence = "reszta z dzielenia równa 1 oznacza, że liczba kontrolna jest " +
		"zweryfikowana jako poprawna"

	// ibanCheckNumberSentence is annex 1, section 2 point 5 of the same act. It
	// is the calculating half, and it is what puts a floor under the check
	// number.
	//
	// In English: the value of the check number CC is computed by subtracting NN
	// from 98, CC=98-NN, and a result below ten is padded with a leading zero.
	//
	// A remainder modulo ninety seven is at most ninety six, so the smallest
	// check number any calculation reaches is two. The pair 00 and 01 is
	// therefore impossible, and the division alone does not catch it: 01 leaves
	// the same remainder as 98 and 00 the same as 97. This is derived from the
	// sentence rather than written down as a rule of its own.
	ibanCheckNumberSentence = "oblicza się wartość liczby kontrolnej CC, odejmując NN od 98: " +
		"CC=98-NN, przy czym w przypadku wyniku mniejszego od 10 uzupełnia się go o zero wiodące"

	// nrbLengthSentence is section 13 of regulation 7/2017, the act in force,
	// Dz. Urz. NBP 2017 item 3, consolidated at 2019 item 9.
	//
	// In English: the national number is unique and consists of twenty six
	// digits.
	//
	// Only the opening clause is quoted. What the twenty six are divided into is
	// a fact about a method rather than an expression of one, so it is stated in
	// our own words below, next to the constants that hold it.
	nrbLengthSentence = "NRB jest unikatowy i składa się z dwudziestu sześciu cyfr"
)

// ibanPolynomialWeights is the weight table of annex 2, section 3 of regulation
// 5/2002, written with the position of each weight beside it.
//
// # This table is the source doing the arithmetic itself
//
// It is not a description of a rule. It is thirty numbers the act computed, and
// they are the powers of ten modulo ninety seven. Reproducing all thirty forces
// the modulus: no other modulus produces this table, which is checked by a test
// rather than asserted here.
//
// That makes it the strongest kind of anchor this project recognises, and it is
// the reason the modulus of this scheme is not a measurement the way the routing
// number weights are.
//
// The positions are numbered from the right, so P1 is the last character of the
// rearranged value.
const ibanPolynomialWeights = "P1 1 P2 10 P3 3 P4 30 P5 9 P6 90 P7 27 P8 76 P9 81 P10 34 " +
	"P11 49 P12 5 P13 50 P14 15 P15 53 P16 45 P17 62 P18 38 P19 89 P20 17 " +
	"P21 73 P22 51 P23 25 P24 56 P25 75 P26 71 P27 31 P28 19 P29 93 P30 57"

// nrbSettlementWeights is the weight table of annex 2 to regulation 7/2017, the
// act in force, for the check digit of the settlement number.
//
// It is a different arithmetic from everything above: modulo ten rather than
// ninety seven, over the eight digits that name the bank. The provider holds
// eight bare numbers in an array and this keeps them labelled, so a weight typed
// into the wrong slot there is not typed into the same wrong slot here.
const nrbSettlementWeights = "P1 3 P2 9 P3 7 P4 1 P5 3 P6 9 P7 7 P8 1"

// The lengths of the Polish form, from sections 13 and 14 of the act in force,
// stated in our own words because a division into fields is a method rather
// than an expression.
const (
	nrbLength              = 26
	nrbSettlementLength    = 8
	nrbAccountLength       = 16
	ibanPolishLength       = 2 + nrbLength
	ibanPolishCountry      = "PL"
	ibanRotationFromSource = 4
	ibanModulusFromSource  = 97
	ibanAcceptedRemainder  = 1
)

// IBANFinding is one reason a value is not a well formed account number. An
// empty list means it is well formed as far as this validator can tell, which
// for a country other than Poland is less than it sounds.
type IBANFinding string

// CheckIBAN reports every rule a value breaks.
//
// It answers about the check digits for any country and about the length and
// the national structure for Poland alone, which is the same division the
// provider makes and for the same reason: the register holding every country's
// length is published under terms this project cannot accept.
func CheckIBAN(value string) []IBANFinding {
	bare, ok := ibanBare(value)
	if !ok {
		return []IBANFinding{"the value is not a country code, two check digits and a national part"}
	}

	var findings []IBANFinding
	if strings.HasPrefix(bare, ibanPolishCountry) {
		findings = append(findings, polishShapeFindings(bare)...)
	}
	if len(findings) > 0 {
		return findings
	}

	floor, ceiling, ok := IBANCheckNumberRangeFromSource()
	if !ok {
		return []IBANFinding{"the sentence stating how the check number is computed no longer carries it"}
	}
	if written := int(bare[2]-'0')*10 + int(bare[3]-'0'); written < floor || written > ceiling {
		return []IBANFinding{"the check number is outside the range the calculation reaches"}
	}

	weights := ibanWeights(len(bare) + ibanLetterCount(bare))
	if weights == nil {
		return []IBANFinding{"the weight table did not come apart into positions"}
	}
	if ibanWeightedTotal(bare, weights)%ibanModulusFromSource != ibanAcceptedRemainder {
		findings = append(findings, "the weighted total leaves a remainder other than one, "+
			"so the check number is not verified as correct")
	}
	return findings
}

// IBANCheckedInFull reports whether this validator checked everything about a
// value or only its check digits.
//
// A caller that treats an empty finding list as approval would approve an
// account number of any length from any country, which is exactly the silence
// this validator exists not to produce.
func IBANCheckedInFull(value string) bool {
	bare, ok := ibanBare(value)
	return ok && strings.HasPrefix(bare, ibanPolishCountry)
}

// polishShapeFindings reports what is wrong with the shape of a Polish value.
func polishShapeFindings(bare string) []IBANFinding {
	if len(bare) != ibanPolishLength {
		return []IBANFinding{"a Polish account number is not twenty eight characters long"}
	}
	for index := 2; index < len(bare); index++ {
		if bare[index] < '0' || bare[index] > '9' {
			return []IBANFinding{"a Polish account number carries a character that is not a digit"}
		}
	}
	settlement := bare[4 : 4+nrbSettlementLength]
	if !settlementCloses(settlement) {
		return []IBANFinding{"the settlement number does not close under its own check digit"}
	}
	return nil
}

// settlementCloses applies the weights of the act in force to the eight digits
// naming the bank.
func settlementCloses(settlement string) bool {
	weights := labelledWeights(nrbSettlementWeights, nrbSettlementLength)
	if weights == nil || len(settlement) != nrbSettlementLength {
		return false
	}
	total := 0
	for position, weight := range weights {
		total += int(settlement[position]-'0') * weight
	}
	return total%10 == 0
}

// ibanWeightedTotal applies the weights to the rearranged value, reading it
// from the right so that position one is the last character.
//
// A letter occupies two positions because it expands into two digits, which is
// what makes the table of thirty fit a Polish value of twenty eight characters.
func ibanWeightedTotal(bare string, weights []int) int {
	rearranged := bare[ibanRotationFromSource:] + bare[:ibanRotationFromSource]
	total, position := 0, 0
	for index := len(rearranged) - 1; index >= 0; index-- {
		symbol := rearranged[index]
		if symbol >= '0' && symbol <= '9' {
			total += int(symbol-'0') * weights[position]
			position++
			continue
		}
		value := int(symbol-'A') + ibanLetterValue('A')
		total += (value % 10) * weights[position]
		total += (value / 10) * weights[position+1]
		position += 2
	}
	return total
}

// ibanWeights returns as many weights as a value of this many digits needs.
//
// The first thirty are compared against the table the act published, by a test.
// Beyond thirty they continue as the same powers, because the act stops at the
// length of a Polish value and this scheme has longer ones. That continuation is
// ours and it is the reason the register of anchors carries a second row for
// this provider.
func ibanWeights(digits int) []int {
	published := labelledWeights(ibanPolynomialWeights, 30)
	if published == nil {
		return nil
	}
	weights := make([]int, 0, digits)
	for position := 0; position < digits; position++ {
		if position < len(published) {
			weights = append(weights, published[position])
			continue
		}
		weights = append(weights, weights[position-1]*10%ibanModulusFromSource)
	}
	return weights
}

// ibanLetterValue returns the value of a letter under the offset the act fixes
// for P and L.
//
// It is derived from the quoted sentence rather than written down, so that a
// change to the quotation changes the arithmetic instead of leaving a number
// beside a sentence that no longer supports it.
func ibanLetterValue(letter byte) int {
	forP, forL, ok := IBANLetterValuesFromSource()
	if !ok {
		return 0
	}
	// Two letters, one offset. Taking both and requiring them to agree is what
	// makes a misread of either one visible.
	if forP-int('P'-'A') != forL-int('L'-'A') {
		return 0
	}
	return int(letter-'A') + forP - int('P'-'A')
}

// IBANLetterValuesFromSource pulls the two letter values out of the quoted
// sentence, so that a test can ask whether they are the ASCII values the act
// claims or the letter values it actually used.
func IBANLetterValuesFromSource() (int, int, bool) {
	forP, okP := valueAfter(ibanLetterValuesSentence, "P=")
	forL, okL := valueAfter(ibanLetterValuesSentence, "L=")
	return forP, forL, okP && okL
}

// IBANCheckNumberRangeFromSource derives both ends of the range a check number
// can occupy, from the sentence that states the calculation.
//
// A remainder modulo ninety seven runs from nought to ninety six, so the
// subtraction runs from the subtrahend down to the subtrahend less ninety six.
// Both ends matter: a check number and that number plus ninety seven leave the
// same remainder, so the division alone lets an impossible value through at
// either end.
//
// It is derived rather than written down so that the two numbers it rests on,
// the subtrahend and the modulus, come from the source on every run. Bounds
// typed in by hand would go on being right after the sentence beside them
// changed.
func IBANCheckNumberRangeFromSource() (int, int, bool) {
	subtrahend, ok := valueAfter(ibanCheckNumberSentence, "CC=")
	if !ok {
		return 0, 0, false
	}
	return subtrahend - (ibanModulusFromSource - 1), subtrahend, true
}

// IBANSourceSaysFourCharactersMove reports whether the quoted sentence still
// fixes the rotation at four, so the constant is compared against the source
// rather than against itself.
func IBANSourceSaysFourCharactersMove() bool {
	return strings.Contains(ibanRearrangementSentence, "cztery pierwsze znaki")
}

// IBANSourceAcceptsRemainderOne reports whether the quoted sentence still makes
// one the accepting remainder.
func IBANSourceAcceptsRemainderOne() bool {
	return strings.Contains(ibanRemainderSentence, "reszta z dzielenia równa 1")
}

// IBANPublishedWeights returns the thirty weights the act printed, for the test
// that reproduces them as powers of ten.
func IBANPublishedWeights() []int { return labelledWeights(ibanPolynomialWeights, 30) }

// NRBLengthFromSource pulls the length out of the quoted sentence of the act in
// force, so that twenty six is compared against the source and not against a
// number somebody typed twice.
func NRBLengthFromSource() bool {
	return strings.Contains(nrbLengthSentence, "dwudziestu sześciu cyfr")
}

// valueAfter reads the number written immediately after a label.
func valueAfter(sentence, label string) (int, bool) {
	at := strings.Index(sentence, label)
	if at < 0 {
		return 0, false
	}
	rest := sentence[at+len(label):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	number, err := strconv.Atoi(rest[:end])
	if err != nil {
		return 0, false
	}
	return number, true
}

// labelledWeights takes a table apart into the weights, reading the number that
// follows each position label.
//
// It returns nothing rather than a short list when the table is not the length
// it should be. A table that lost a position would otherwise agree with a
// provider that lost the same one.
func labelledWeights(table string, want int) []int {
	fields := strings.Fields(table)
	if len(fields) != want*2 {
		return nil
	}
	weights := make([]int, 0, want)
	for index := 1; index < len(fields); index += 2 {
		number, err := strconv.Atoi(fields[index])
		if err != nil {
			return nil
		}
		weights = append(weights, number)
	}
	return weights
}

// ibanBare removes the spaces of the printed form and checks the shape every
// account number shares, whatever country it belongs to.
func ibanBare(value string) (string, bool) {
	bare := strings.ReplaceAll(value, " ", "")
	if len(bare) < 5 {
		return "", false
	}
	for index := 0; index < len(bare); index++ {
		symbol := bare[index]
		digit := symbol >= '0' && symbol <= '9'
		letter := symbol >= 'A' && symbol <= 'Z'
		switch {
		case index < 2 && !letter:
			return "", false
		case index >= 2 && index < 4 && !digit:
			return "", false
		case index >= 4 && !digit && !letter:
			return "", false
		}
	}
	return bare, true
}

// ibanLetterCount counts the letters, each of which becomes two digits.
func ibanLetterCount(bare string) int {
	letters := 0
	for index := 0; index < len(bare); index++ {
		if bare[index] >= 'A' && bare[index] <= 'Z' {
			letters++
		}
	}
	return letters
}
