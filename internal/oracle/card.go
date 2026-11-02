package oracle

import "strings"

// The rule this file implements.
//
// A card number carries a check digit computed by the formula of Luhn. No
// first party source was found that states it for every card: the standard
// that numbers cards, ISO/IEC 7812, is sold rather than published, and the PCI
// Security Standards Council, in FAQ 1137 of July 2012, calls it the algorithm
// most often used to validate a primary account number. So the rule is anchored
// on the numbers instead. Eight numbers published as test numbers by at least
// two of four payment providers, read on 2026-11-02, all satisfy it, which a
// value of the same length does one time in ten.
//
// The FAQ words its first step so that a doubled five stays ten. Read that way,
// 5555555555554444 fails, and three of the four providers publish it, so this
// file does not follow that wording. It asks the question the FAQ asks last,
// whether a total divides by ten, and builds the total by adding the individual
// digits of every product. The provider computes the digit and takes nine off a
// product above nine instead. Two operations for one rule, so a fold written the
// wrong way round in one is not written the same way in the other.
//
// The shape of an American Express number is the one quotation here.

// cardAmexShapeAsPublished is the sentence of the American Express merchant
// guide that gives the shape of its numbers.
//
// American Express Global Merchant Services, "Getting Started" (United
// Kingdom), the page on preventing fraud, under card security features. Read
// from the page itself on 2026-11-02 rather than from a summary of it.
const cardAmexShapeAsPublished = "Card Account numbers are embossed and begin with ‘34’ or ‘37’ and consist of 15 digits in a four-six-five formation."

// cardCheckedLength is the form every other number this build writes takes.
//
// Ours rather than a quotation: six of the eight published numbers are sixteen
// digits, and a number of another length is a form this build passes over.
const cardCheckedLength = 16

// CardFinding is one reason a value is not a valid card number of a form this
// build checks. An empty list means the value is valid.
type CardFinding string

// CheckCard reports every rule a value breaks.
//
// # What it deliberately does not answer
//
// It asks about the two forms this build writes, sixteen digits and the
// American Express fifteen. A number of another length is said to be outside
// those forms and nothing more, because the provider passes such a value over
// rather than calling it wrong, and an oracle with an opinion about it would
// disagree with the provider about a value neither of them calls wrong.
//
// It also says nothing about whether a number is one of the published test
// numbers. The provider does not refuse other numbers either, so neither does
// this.
func CheckCard(value string) []CardFinding {
	prefixes, length, ok := cardAmexShape()
	if !ok {
		// Loud rather than guessed. The form below is read out of the sentence,
		// and a sentence that no longer carries it supports nothing.
		return []CardFinding{"the American Express sentence no longer gives two prefixes and a length"}
	}
	for at := 0; at < len(value); at++ {
		if value[at] < '0' || value[at] > '9' {
			return []CardFinding{"the value carries a character that is not a digit"}
		}
	}
	if !cardIsCheckedForm(value, prefixes, length) {
		return []CardFinding{"the value is neither sixteen digits nor an American Express number"}
	}
	if CardWeightedTotal(value)%10 != 0 {
		return []CardFinding{"the total of the weighted digits does not divide by ten"}
	}
	return nil
}

// CardWeightedTotal is the total whose division by ten decides a card number.
//
// Counted from the right, the check digit weighs one, the digit before it two,
// and so on alternately. A product of two digits is added as its digits, so ten
// adds one and eighteen adds nine.
//
// Exported so that a test can hold the published Mastercard number to it, which
// is the one the literal wording of the FAQ refuses.
func CardWeightedTotal(value string) int {
	total := 0
	for back := 0; back < len(value); back++ {
		product := int(value[len(value)-1-back]-'0') * (1 + back%2)
		total += product/10 + product%10
	}
	return total
}

// cardIsCheckedForm reports whether a value of digits is sixteen long, or has
// the length and one of the beginnings American Express gives.
func cardIsCheckedForm(value string, prefixes []string, length int) bool {
	if len(value) == cardCheckedLength {
		return true
	}
	if len(value) != length {
		return false
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

// cardAmexShape reads the two prefixes and the length out of the quoted
// sentence, so neither is typed here a second time.
//
// The prefixes are the runs of digits between quotation marks, and the length
// is the run before the word digits.
func cardAmexShape() (prefixes []string, length int, ok bool) {
	for _, quoted := range strings.Split(cardAmexShapeAsPublished, "‘")[1:] {
		closing := strings.Index(quoted, "’")
		if closing <= 0 {
			return nil, 0, false
		}
		prefixes = append(prefixes, quoted[:closing])
	}
	before, _, found := strings.Cut(cardAmexShapeAsPublished, " digits")
	if !found {
		return nil, 0, false
	}
	words := strings.Fields(before)
	for _, digit := range words[len(words)-1] {
		if digit < '0' || digit > '9' {
			return nil, 0, false
		}
		length = length*10 + int(digit-'0')
	}
	return prefixes, length, len(prefixes) == 2 && length > 0
}
