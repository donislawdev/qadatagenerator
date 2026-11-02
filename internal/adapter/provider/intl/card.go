package intl

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// Card is the number on a payment card, and this build writes only numbers
// that payment providers publish for testing.
//
// # Where the numbers come from, and why from two places at once
//
// A number drawn at random that satisfies the check digit can be somebody's
// card, so the handbook lets a card number into a dataset only when a payment
// provider publishes it as a test number. DEC-268 narrows that further. The
// test documentation of Stripe, Braintree, Adyen and Checkout.com was read on
// 2026-11-02, and a number is here only when at least two of those four
// publish it as a test number. Eight numbers pass, of six brands.
//
// Two reasons for asking two. No number here comes out of one provider's
// table, so this list is not a copy of anybody's list. And two providers
// saying the same thing about one number is better evidence that it is a test
// number than one provider saying it.
//
// # The check digit, and a source that words it wrongly
//
// The last digit is computed by the formula of Luhn. The PCI Security
// Standards Council names it, in FAQ 1137 of July 2012, as the algorithm most
// often used to validate a primary account number. Most often is not always,
// and that is why Validate below approves fewer forms than exist. The eight
// numbers here all satisfy it, which a value of the same length does one time
// in ten, so eight in a row by chance happen about once in a hundred million.
//
// The same FAQ says to subtract nine from a doubled value greater than ten.
// Read literally, a doubled five stays ten, and 5555555555554444, which three of
// the four providers publish, then fails. The published numbers decide against
// the words. This file subtracts nine from anything above nine, and the
// independent validator adds the digits of each product instead, which is a
// third way of saying the same rule.
//
// # Why Validate approves two forms and passes over the others
//
// A validator that approved any number of digits with a correct check digit
// would approve every equipment identity as well, because an IMEI is fifteen
// digits under the same formula. The mode that reads somebody else's file
// refuses to choose between two schemes accepting a whole column, so a column
// of IMEI values would stop being recognised the day this file arrived. That
// was worked out before this was written rather than measured after.
//
// So Valid is given to the two forms this build writes: sixteen digits, and
// fifteen beginning 34 or 37, which American Express gives in its merchant
// guide as the shape of its numbers. Any other length up to nineteen digits is
// a card number too, the providers publish fourteen and nineteen digit ones, and
// such a value is Unsupported: recognised and not checked, so neither approved
// nor faulted, and the corruption engine leaves it alone rather than claiming a
// rejection nobody verified.
//
// Validate is written from those rules rather than from the list. A test
// number of some other provider in somebody's file is accepted and can be
// damaged, because calling it wrong would report a defect in a value that has
// none.
//
// # There is no safe harbour
//
// A test number works in its provider's test system, and nothing in any of the
// four documents says that no issuer has given the same number to a real card.
// The list narrows what this build writes to numbers providers call test
// numbers, and that is all it claims, so identifier.reserved-range is not
// declared.
type Card struct{}

// Contract checks. There is deliberately no RangeAware here, for the reason
// given above.
var (
	_ provider.Provider     = Card{}
	_ provider.CheckDigited = Card{}
	_ provider.Shaped       = Card{}
)

// cardTestNumber is one number this build writes, with the providers that
// publish it as a test number.
type cardTestNumber struct {
	number      string
	brand       string
	publishedBy []string
}

// The four providers whose test documentation was read on 2026-11-02.
const (
	cardStripe    = "Stripe"
	cardBraintree = "Braintree"
	cardAdyen     = "Adyen"
	cardCheckout  = "Checkout.com"
)

// cardPublished are the numbers this build writes, in the order they are
// drawn from.
//
// 🔴 A number joins only with two providers beside it. One provider is a number
// out of one table, and copying a provider's table is what DEC-268 was decided
// against. The providers are data rather than a comment so that the test beside
// this file can hold every number to that.
var cardPublished = []cardTestNumber{
	{"4242424242424242", "Visa", []string{cardStripe, cardCheckout}},
	{"4111111111111111", "Visa", []string{cardBraintree, cardAdyen}},
	{"5555555555554444", "Mastercard", []string{cardStripe, cardBraintree, cardAdyen}},
	{"378282246310005", "American Express", []string{cardStripe, cardBraintree}},
	{"371449635398431", "American Express", []string{cardStripe, cardBraintree}},
	{"6011111111111117", "Discover", []string{cardStripe, cardCheckout}},
	{"3530111333300000", "JCB", []string{cardBraintree, cardCheckout}},
	{"6243030000000001", "UnionPay", []string{cardBraintree, cardAdyen}},
}

const (
	// cardLength is the form most card numbers take.
	cardLength = 16
	// cardAmexLength is the form American Express gives its numbers.
	cardAmexLength = 15
	// cardLongest is the longest number the four providers publish, and none
	// of them publishes a longer one. A value longer than this is refused
	// rather than passed over.
	cardLongest = 19
	// cardModulus is the divisor of the Luhn formula.
	cardModulus = 10
)

// cardAmexPrefixes begin every American Express number, as its merchant guide
// writes them.
var cardAmexPrefixes = []string{"34", "37"}

// ID names this provider.
//
// No country prefix, because a card number is issued under an international
// numbering rather than by a country. The catalog published this name before
// this file existed, in the applies_to of both checksum entries, so it is read
// from there rather than chosen here.
func (Card) ID() string { return "card" }

// Masks are the two forms, fifteen digits and sixteen, because the numbers this
// build writes take both.
func (Card) Masks() []string {
	digits := func(n int) string { return strings.Repeat(string(provider.MaskDigit), n) }
	return []string{digits(cardAmexLength), digits(cardLength)}
}

// Capabilities are what this provider brings.
//
// One. The check digit is what the catalog asks of this scheme. There is no
// reserved range for the reason set out on the type, and no claim about dates
// or gender, because a card number encodes an issuer and an account and
// neither is a value a recipe holds in another column.
func (Card) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Published returns the numbers this build writes, in a copy the caller may
// keep.
//
// The registry of generators draws from this rather than through Generate, so
// that a unique column can count the numbers and a limit on length can choose
// among them. Both read this one list, so the two cannot part company.
func (Card) Published() []string {
	numbers := make([]string, 0, len(cardPublished))
	for _, published := range cardPublished {
		numbers = append(numbers, published.number)
	}
	return numbers
}

// Generate picks one of the published numbers.
//
// The agreement is not read. Nothing in this scheme is bound to another column
// of the row.
//
// The draw is made at the address of the request itself, which is the draw the
// registry of generators makes from Published, so this and a recipe produce
// the same number for the same row.
func (Card) Generate(req provider.Request) (string, error) {
	return cardPublished[req.Source.IntN(req.At, int64(len(cardPublished)))].number, nil
}

// Validate answers whether a value is a correct card number of a form this
// build checks.
func (Card) Validate(value string) provider.Verdict {
	if value == "" {
		return provider.Invalid("an empty value is not a card number")
	}
	if at, strange := cardStrangeCharacter(value); strange {
		return provider.Invalid(fmt.Sprintf(
			"position %d of %s is not a digit, and a card number is digits throughout",
			at+1, provider.Show(value)))
	}
	if len(value) > cardLongest {
		return provider.Invalid(fmt.Sprintf(
			"%s is %d digits, and no card number this build knows of is longer than %d",
			provider.Show(value), len(value), cardLongest))
	}
	if !cardChecked(value) {
		return provider.Unsupported(fmt.Sprintf(
			"%s is %d digits, and this build checks a card number of %d digits, or of %d beginning "+
				"%s for American Express. Other forms exist and are passed over rather than called wrong",
			provider.Show(value), len(value), cardLength, cardAmexLength,
			strings.Join(cardAmexPrefixes, " or ")))
	}
	last := len(value) - 1
	if want := cardCheckDigit(value); value[last] != want {
		return provider.Invalid(fmt.Sprintf(
			"the check digit is %s and should be %s",
			provider.Show(value[last:]), provider.Show(string(want))))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits, in a form this build checks.
func (Card) CheckDigits(value string) ([]int, bool) {
	if !cardChecked(value) || cardHasStrangeCharacter(value) {
		return nil, false
	}
	return []int{len(value) - 1}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
//
// It never refuses for arithmetic reasons, because ten residues fit in ten
// digits. It refuses a form this build does not check, because a check digit
// recomputed there would be one nobody verified.
func (Card) WithCheckDigits(value string) (string, error) {
	if !cardChecked(value) {
		return "", fmt.Errorf("card: %s is not a form this build checks", provider.Show(value))
	}
	if cardHasStrangeCharacter(value) {
		return "", fmt.Errorf(
			"card: %s carries a character that is not a digit, so no check digit completes it",
			provider.Show(value))
	}
	rebuilt := []byte(value)
	rebuilt[len(rebuilt)-1] = cardCheckDigit(value)
	return string(rebuilt), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// cardChecked reports whether a value has the length and the beginning of a
// form this build checks. It does not look at whether the rest is digits.
func cardChecked(value string) bool {
	switch len(value) {
	case cardLength:
		return true
	case cardAmexLength:
		for _, prefix := range cardAmexPrefixes {
			if strings.HasPrefix(value, prefix) {
				return true
			}
		}
	}
	return false
}

// cardCheckDigit computes the last digit over the ones before it, which the
// caller has established are digits.
//
// Walking from the digit next to the check digit, every second one starting
// with that one is doubled, and nine is taken off a doubled value above nine.
func cardCheckDigit(value string) byte {
	sum := 0
	body := len(value) - 1
	for back := 0; back < body; back++ {
		digit := int(value[body-1-back] - '0')
		if back%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return byte('0' + (cardModulus-sum%cardModulus)%cardModulus)
}

// cardStrangeCharacter returns the first position holding something that is not
// a digit, and whether there is one.
func cardStrangeCharacter(value string) (int, bool) {
	for at := 0; at < len(value); at++ {
		if !isDigit(value[at]) {
			return at, true
		}
	}
	return 0, false
}

// cardHasStrangeCharacter is the same question without the position.
func cardHasStrangeCharacter(value string) bool {
	_, strange := cardStrangeCharacter(value)
	return strange
}
