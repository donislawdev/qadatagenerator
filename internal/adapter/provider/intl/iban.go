// Package intl holds the providers for numbering schemes that belong to no one
// country.
//
// The identifier of such a provider carries no country prefix, because a
// provider belongs to a country and these do not. That is the same rule the
// country packages follow from the other side.
package intl

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// IBAN is an international bank account number.
//
// # One scheme, two questions, and only one of them has a source we can read
//
// The arithmetic over the check digits is the same everywhere, and it was read
// in a normative act. The length and the national part are a property of each
// country, and the only register that holds all of them is published under
// terms this project cannot accept. So this provider answers the first question
// for every country and the second one for Poland alone, and it says which it
// is doing rather than letting a caller assume the answer covers both.
//
// # Where the arithmetic came from
//
// Regulation 5/2002 of the President of the National Bank of Poland, Dz. Urz.
// NBP 2002 no. 8 item 26, annexes 1 and 2, states the whole of it in words: the
// four leading characters move to the end, the letters become numbers, the
// result is divided by ninety seven, a remainder of one means the check number
// is correct, and the check number itself is ninety eight minus that remainder.
// A normative act is outside copyright under article 4 point 1 of the Polish
// copyright act, so the wording is quoted freely in the independent validator.
//
// That act is repealed, and the two that replaced it dropped those annexes:
// 15/2010 and 7/2017 carry only the check digit of the settlement number. The
// shape below therefore comes from the act in force and the arithmetic from the
// one that is not, which is the same split the statistical number met from the
// other side.
//
// The act states the rule for Poland. Applying it to every other country is our
// generalisation, and it rests on a measurement rather than on that act: 89
// example account numbers from 89 countries, taken from the ISO 13616 register
// on 2026-08-04, leave exactly one modulus, one rotation and one letter offset
// that close all of them. The register itself is not shipped here and no example
// from it is stored, for the reason the fuzz seeds give.
//
// # Where the shape came from
//
// Regulation 7/2017, in force, section 13: the national number is twenty six
// digits, of which the first two are the check number, the next eight the
// settlement number and the last sixteen the account. Section 14: the
// international form is twenty eight characters, the country code PL and then
// those twenty six. Annex 2 gives the settlement number its own check digit,
// modulo ten under the weights three nine seven one repeated.
//
// # Ethics, and this scheme has no harbour either
//
// No block of this space is set aside as never issued. The country codes ISO
// 3166 leaves for private use are not spare: one of them is already carrying a
// live country in this very register, which is the same trap a Social Security
// range looked like once. Inside Poland the settlement number names a real bank,
// and a number that satisfies the act's rule may name one that exists.
//
// So every value produced here could resemble a real account, and the defence is
// the one the routing number already uses: the dataset carrying these has to say
// it is synthetic, which the tool does from the binary.
type IBAN struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider     = IBAN{}
	_ provider.CheckDigited = IBAN{}
)

const (
	// ibanCountryLength and ibanCheckLength are the two leading fields every
	// account number has, whatever country it belongs to.
	ibanCountryLength = 2
	ibanCheckLength   = 2
	// ibanPrefixLength is the two together, which is also how many characters
	// move to the end before the arithmetic.
	ibanPrefixLength = ibanCountryLength + ibanCheckLength
	// ibanShortest is the shortest string that can be one at all: the two
	// fields above and at least one character of a national part.
	//
	// There is deliberately no longest. A ceiling would be a rule about every
	// country, and the only document holding those is the one this provider
	// exists to avoid depending on. A value longer than any real account number
	// is answered as unchecked rather than faulted, which is true.
	ibanShortest = ibanPrefixLength + 1
	// ibanModulus, ibanRotation and ibanLetterBase are the three constants of
	// the arithmetic. See the type comment for where each one comes from.
	ibanModulus    = 97
	ibanRotation   = ibanPrefixLength
	ibanLetterBase = 10
	// ibanValidRemainder is what the division leaves for a correct value.
	ibanValidRemainder = 1
	// ibanSubtrahend is what the remainder is taken away from to get the check
	// number.
	ibanSubtrahend = 98
	// ibanLowestCheckNumber and ibanHighestCheckNumber follow from the two above
	// rather than from a separate rule. A remainder modulo ninety seven runs
	// from nought to ninety six, so ninety eight minus it runs from two to
	// ninety eight. Nothing reaches 00, 01 or 99.
	//
	// 🔴 Both ends are needed and the second one was missing until a fuzzer
	// found it. The division alone does not catch either end, because a check
	// number and that number plus ninety seven leave the same remainder: 00
	// pairs with 97, 01 with 98, and 99 with 02. Guarding only the low end
	// looked complete and let one impossible value through, which is this
	// project's recurring shape of a rule that is true where it was checked.
	ibanLowestCheckNumber  = ibanSubtrahend - (ibanModulus - 1)
	ibanHighestCheckNumber = ibanSubtrahend
)

// The Polish form, from regulation 7/2017 sections 13 and 14.
const (
	polishCountry = "PL"
	// polishSettlementLength and polishAccountLength are the two parts of the
	// national number after its own check number.
	polishSettlementLength = 8
	polishAccountLength    = 16
	polishNationalLength   = ibanCheckLength + polishSettlementLength + polishAccountLength
	polishLength           = ibanCountryLength + polishNationalLength
	// polishSettlementModulus is the modulus of the settlement number's own
	// check digit, which is a different arithmetic from the one above.
	polishSettlementModulus = 10
)

// polishSettlementWeights are applied to the settlement number, from annex 2 of
// regulation 7/2017.
//
// The eighth position is the check digit and its weight is one, so the digit
// that closes the number is the plain complement of the total over the first
// seven. Holding eight weights here rather than seven is deliberate and is the
// opposite of the routing number, where the unused ninth entry was dead: here
// every entry is read, because verification runs over all eight.
var polishSettlementWeights = [polishSettlementLength]int{3, 9, 7, 1, 3, 9, 7, 1}

// The reasons a value is refused. They are constants so that a test can compare
// against the same text the user sees, and none of them repeats the value it
// was given: a reason that quotes unparsed input is a message that grows with
// whatever somebody put in a file.
const (
	ibanShapeReason = "an account number is two letters naming the country, two check digits and " +
		"then the national part, in capitals, optionally written in groups of four"
	ibanImpossibleCheckReason = "no calculation produces these check digits: the rule takes a " +
		"remainder modulo ninety seven away from ninety eight, so they run from 02 to 98"
	ibanCheckReason = "the check digits do not agree with the rest of the account number"
	// ibanUncheckedCountry carries the country code, which is two letters that
	// were already checked to be letters, so it cannot carry anything else.
	ibanUncheckedCountry = "the check digits agree, and the length and national structure of an " +
		"account number for %s are not checked by this build. Only the Polish form was read at a " +
		"primary source, so a value from anywhere else is passed over rather than approved"
	polishLengthReason = "a Polish account number is twenty eight characters, the country code and " +
		"twenty six digits"
	polishDigitsReason = "a Polish account number carries digits after the country code, and this " +
		"one carries a letter"
	polishSettlementReason = "the settlement number does not close under its own check digit, so it " +
		"is not one the central bank could have assigned"
)

// ID names this provider. There is no country prefix because the scheme belongs
// to no country.
func (IBAN) ID() string { return "iban" }

// Capabilities are what this provider brings.
//
// One, and the two that are absent both mean something. There is no reserved
// range to declare, so a report reading the registry learns that a dataset
// carrying these has to be marked synthetic. There is no unassigned target
// either, which is the stronger absence: the routing number could at least offer
// a block that is spare today, and this scheme cannot even do that.
func (IBAN) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Generate produces one Polish account number for the place described by the
// request.
//
// It is Polish because that is the only country whose length and national
// structure this build read at a primary source, and producing a value whose
// correctness we could not check would be the defect this tool sells.
//
// The agreement is not read. An account number encodes nothing about a person,
// so there is nothing in it that could be made to agree with a date of birth.
func (i IBAN) Generate(req provider.Request) (string, error) {
	draw := func(purpose string, bound int64) int64 {
		return req.Source.IntN(seed.Coordinate{
			Entity:  req.At.Entity,
			Row:     req.At.Row,
			Field:   req.At.Field,
			Purpose: req.At.Purpose + purpose,
		}, bound)
	}

	national := make([]byte, 0, polishSettlementLength+polishAccountLength)
	national = appendSettlement(national, draw(":iban-settlement", 10000000))
	national = appendFixedDigits(national, draw(":iban-account", 10000000000000000), polishAccountLength)

	check, ok := checkNumber(polishCountry, string(national))
	if !ok {
		// Unreachable: everything appended above is a digit. Answered rather
		// than ignored, because a value leaving here with the wrong check
		// digits would be exactly the silent defect this tool sells.
		return "", fmt.Errorf("iban: the generated national part did not divide")
	}
	return polishCountry + check + string(national), nil
}

// Validate says whether a value is a correct account number.
//
// It answers three different questions and the order they are asked in is part
// of the answer. A Polish value of the wrong length is told so, rather than
// being told its check digits disagree, which they also would: reporting the
// arithmetic when the shape is wrong names the symptom instead of the fault.
func (i IBAN) Validate(value string) provider.Verdict {
	parsed, ok := parseIBAN(value)
	if !ok {
		return provider.Invalid(ibanShapeReason)
	}

	polish := parsed.country() == polishCountry
	if polish {
		if fault := polishShapeFault(parsed); fault != "" {
			return provider.Invalid(fault)
		}
	}
	if fault := checkNumberFault(parsed); fault != "" {
		return provider.Invalid(fault)
	}
	if !polish {
		return provider.Unsupported(fmt.Sprintf(ibanUncheckedCountry, parsed.country()))
	}
	if !settlementCloses(parsed.settlement()) {
		return provider.Invalid(polishSettlementReason)
	}
	return provider.Valid()
}

// CheckDigits says where the check digits sit, as offsets into the value as it
// was written.
//
// They are the third and fourth characters of the number, which is not always
// the third and fourth byte: a value written with spaces carries them further
// along. Answering about the number and returning offsets into something else
// is the mistake this signature is shaped to avoid.
func (IBAN) CheckDigits(value string) ([]int, bool) {
	parsed, ok := parseIBAN(value)
	if !ok {
		return nil, false
	}
	return []int{parsed.offsets[ibanCountryLength], parsed.offsets[ibanCountryLength+1]}, true
}

// WithCheckDigits returns the value with its check digits recomputed, in the
// shape it arrived in.
//
// It works for a value from any country, unlike Validate, and the difference is
// not an inconsistency. Recomputing needs the arithmetic, which is the same
// everywhere. Approving needs the length, which is not.
//
// It never refuses for arithmetic reasons. Every national part has a completion
// because the rule is a subtraction rather than a lookup, which is the
// difference from the Polish tax number.
func (IBAN) WithCheckDigits(value string) (string, error) {
	parsed, ok := parseIBAN(value)
	if !ok {
		return "", fmt.Errorf("iban: the value is not an account number this build can read")
	}
	check, ok := checkNumber(parsed.country(), parsed.national())
	if !ok {
		return "", fmt.Errorf("iban: the national part of the value did not divide")
	}
	return parsed.withCheckNumber(check), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// checkNumber returns the two check digits for a national part, as text, and
// whether the input could be divided at all.
//
// This is annex 1 of the repealed regulation: the national part, the country
// code and two zeros, letters turned into numbers, the remainder modulo ninety
// seven taken away from ninety eight. The independent validator states the
// verifying half instead, and over a table of weights rather than a division,
// so the two do not restate one another.
//
// The country is a parameter rather than the Polish one, because the arithmetic
// belongs to the scheme and not to Poland. Only the shape rules below are
// Polish.
func checkNumber(country, national string) (string, bool) {
	remainder, ok := ibanRemainder(national + country + "00")
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%02d", ibanSubtrahend-remainder), true
}

// checkNumberFault reports why the check digits are wrong, or an empty string.
func checkNumberFault(parsed ibanValue) string {
	if written := parsed.checkNumber(); written < ibanLowestCheckNumber ||
		written > ibanHighestCheckNumber {
		return ibanImpossibleCheckReason
	}
	remainder, ok := ibanRemainder(parsed.rearranged())
	if !ok || remainder != ibanValidRemainder {
		return ibanCheckReason
	}
	return ""
}

// ibanRemainder divides the rearranged value by ninety seven.
//
// The number has thirty digits or more, so it is reduced as it is read rather
// than held. A letter contributes two digits, which is why it advances the
// running value by a hundred: every letter value from ten to thirty five is two
// digits wide.
func ibanRemainder(rearranged string) (int, bool) {
	remainder := 0
	for index := 0; index < len(rearranged); index++ {
		symbol := rearranged[index]
		switch {
		case isDigit(symbol):
			remainder = (remainder*10 + int(symbol-'0')) % ibanModulus
		case isUpperLetter(symbol):
			remainder = (remainder*100 + int(symbol-'A') + ibanLetterBase) % ibanModulus
		default:
			return 0, false
		}
	}
	return remainder, true
}

// settlementCloses reports whether the settlement number satisfies its own
// check digit, from annex 2 of the regulation in force.
//
// A number that fails this was never assigned by the central bank, which is a
// different defect from the check digits of the account number as a whole, and
// it is reported separately for that reason.
func settlementCloses(settlement string) bool {
	if len(settlement) != polishSettlementLength {
		return false
	}
	total := 0
	for position, weight := range polishSettlementWeights {
		symbol := settlement[position]
		if !isDigit(symbol) {
			return false
		}
		total += int(symbol-'0') * weight
	}
	return total%polishSettlementModulus == 0
}

// appendSettlement appends seven drawn digits and the check digit that closes
// them, so that pillar one produces a settlement number the central bank's own
// rule accepts.
//
// The weight of the last position is one, so the closing digit is the plain
// complement of the total over the first seven and there is never a body that
// cannot be closed.
func appendSettlement(into []byte, drawn int64) []byte {
	body := appendFixedDigits(make([]byte, 0, polishSettlementLength-1), drawn,
		polishSettlementLength-1)

	total := 0
	for position, digit := range body {
		total += int(digit-'0') * polishSettlementWeights[position]
	}
	closing := (polishSettlementModulus - total%polishSettlementModulus) % polishSettlementModulus

	into = append(into, body...)
	return append(into, byte('0'+closing))
}

// appendFixedDigits appends a value as a fixed number of digits, most
// significant first.
func appendFixedDigits(into []byte, value int64, width int) []byte {
	divisor := int64(1)
	for remaining := 1; remaining < width; remaining++ {
		divisor *= 10
	}
	for ; divisor > 0; divisor /= 10 {
		into = append(into, byte('0'+value/divisor%10))
	}
	return into
}

// --- the value, kept together with the shape it arrived in ---------------------

// ibanValue is a parsed account number in its bare form, together with where
// each of its characters sat in the text it arrived in.
//
// # Why the offsets are kept rather than a flag saying it was grouped
//
// The regulation says the spaces are removed before the arithmetic and says
// nothing about where they were, so any spacing is a way of writing the same
// number and all of them have to be read. That leaves the question of how to
// write one back, and a flag cannot answer it: regrouping a value into fours
// because it happened to contain a space would be a second change nobody asked
// for, and a corruption is supposed to change one thing.
//
// Holding the offsets answers it exactly. A rebuilt value is the text that
// arrived with two characters replaced, so the spacing survives whatever it was.
type ibanValue struct {
	original string
	text     string
	offsets  []int
}

func (v ibanValue) country() string  { return v.text[:ibanCountryLength] }
func (v ibanValue) national() string { return v.text[ibanPrefixLength:] }

// settlement is the eight digits naming the bank, which only a Polish value
// has. It is empty for anything shorter than the Polish form.
func (v ibanValue) settlement() string {
	if len(v.text) != polishLength {
		return ""
	}
	return v.text[ibanPrefixLength : ibanPrefixLength+polishSettlementLength]
}

// checkNumber reads the two check digits as a number, which is what makes the
// pair 00 and 01 answerable.
func (v ibanValue) checkNumber() int {
	return int(v.text[ibanCountryLength]-'0')*10 + int(v.text[ibanCountryLength+1]-'0')
}

// rearranged moves the four leading characters to the end, which is the whole
// of the rearrangement the regulation describes.
func (v ibanValue) rearranged() string {
	return v.text[ibanRotation:] + v.text[:ibanRotation]
}

// withCheckNumber returns the text this value arrived as, carrying different
// check digits, with the spacing it had.
func (v ibanValue) withCheckNumber(check string) string {
	rebuilt := []byte(v.original)
	rebuilt[v.offsets[ibanCountryLength]] = check[0]
	rebuilt[v.offsets[ibanCountryLength+1]] = check[1]
	return string(rebuilt)
}

// parseIBAN reads a value in any spacing and refuses everything else.
//
// It answers about the shape every account number shares and says nothing about
// length, because length belongs to a country and this function does not know
// which countries exist.
func parseIBAN(value string) (ibanValue, bool) {
	bare := make([]byte, 0, len(value))
	offsets := make([]int, 0, len(value))
	for index := 0; index < len(value); index++ {
		if value[index] == ' ' {
			continue
		}
		bare = append(bare, value[index])
		offsets = append(offsets, index)
	}
	if len(bare) < ibanShortest || !ibanShapeHolds(bare) {
		return ibanValue{}, false
	}
	return ibanValue{original: value, text: string(bare), offsets: offsets}, true
}

// ibanShapeHolds checks the two letters, the two digits and the national part
// every account number shares.
func ibanShapeHolds(bare []byte) bool {
	for index, symbol := range bare {
		switch {
		case index < ibanCountryLength && !isUpperLetter(symbol):
			return false
		case index >= ibanCountryLength && index < ibanPrefixLength && !isDigit(symbol):
			return false
		case index >= ibanPrefixLength && !isDigit(symbol) && !isUpperLetter(symbol):
			return false
		}
	}
	return true
}

func isDigit(symbol byte) bool { return symbol >= '0' && symbol <= '9' }

func isUpperLetter(symbol byte) bool { return symbol >= 'A' && symbol <= 'Z' }

// polishShapeFault reports why a Polish value has the wrong shape, or an empty
// string.
//
// This is the half that exists for one country only, and the reason it exists
// for one country only is written on the type.
func polishShapeFault(parsed ibanValue) string {
	if len(parsed.text) != polishLength {
		return polishLengthReason
	}
	for index := ibanCountryLength; index < len(parsed.text); index++ {
		if !isDigit(parsed.text[index]) {
			return polishDigitsReason
		}
	}
	return ""
}
