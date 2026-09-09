package intl

import (
	"strings"

	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// IMEI is the fifteen digit identity of a piece of mobile equipment.
//
// # Where the rules come from
//
// ETSI TS 123 003 version 19.7.0, which is the transposition of 3GPP TS 23.003,
// "Numbering, addressing and identification". The shape is clause 6.2.1 and the
// arithmetic is annex B, which is normative and works an example of itself.
//
// # Why the newest text is not the one cited
//
// The newest text is version 20.0.0 of 2026-06-26, from the 3GPP archive, and
// it was read as well. It says of itself that it has not been through any
// approval process by the 3GPP Organizational Partners, that it shall not be
// implemented, and that anybody implementing the system should obtain the
// specification from the Partners' publications offices. ETSI is one of those
// offices and its transposition carries no such disclaimer, so for a tool that
// implements the rule the ETSI publication is the one with standing and the
// 3GPP copy is the newer draft.
//
// Both were read and annex B is word for word the same in each, so the
// arithmetic below is not in doubt either way. That is the whole reason the
// newer one was fetched: not to cite it, but to know that the rule had not
// moved since the published version.
//
// Version was established before anything was quoted, and it mattered twice
// over. A search offered version 16.6.0 of April 2021 as the newest, four
// releases behind, and the previous scheme in this build was nearly written
// from a specification five years stale.
//
// Two readings of one text is not two sources, so this counts as one document,
// and it is named that way in the anchor register.
//
// # The example the specification works, and the number in the middle of it
//
// Annex B.3 takes the fourteen digits 2 6 0 5 3 1 7 9 3 1 1 3 8 3 apart, prints
// the total 53 it reaches at step 2, and only then arrives at the check digit
// 7. A worked example that gives an answer and no working lets a wrong sum and
// a wrong fold cancel each other out. This one pins both halves, which is the
// same footing the article number stands on.
//
// # Why the arithmetic is written twice and never the same way twice
//
// The specification labels the digits D14 down to D1 and doubles the odd
// labelled ones, then tells the reader to subtract the total from the next
// higher number ending in nought. This file walks the value from its right hand
// end and folds a remainder, which is the compact way to say it. The
// independent validator follows the labels and the sentence instead. Two
// shapes, so a fold written the wrong way round in one is not written the same
// wrong way round in the other.
//
// # There is no safe harbour, and the reason is worth reading before it is
// changed
//
// Clause 6.3 has the type allocation code issued by the GSM Association acting
// as Global Decimal Administrator, which is the same shape as the routing
// number and the vehicle number: allocated by a body, not left free.
//
// GSMA TS.06, version 28.1, does describe a test space. Its table 1 lists the
// reporting body identifier 00 as "Test ME" rather than as a commercial
// allocator, and equipment carrying a test identity cannot be supplied to the
// market. That is worth something and this scheme generates there, see Generate
// below. It is still not a harbour in the sense docs/05 uses the word: the
// format in clause 9.1 of that document is 00, then 44 for the allocating body,
// then a manufacturer code that is unique to a single manufacturer. Codes are
// issued inside the block, so a value there can coincide with a real prototype.
// A harbour means nothing real can sit where we write, and that is not true
// here, so identifier.reserved-range is not declared.
//
// # A caution about one word in the source
//
// The specification uses TAC for two unrelated things. In clause 6 it is the
// Type Allocation Code of eight digits, which is this scheme. In clause 19 it
// is the Tracking Area Code of two octets, which is part of an address and has
// nothing to do with equipment. The reserved values named in the second sense
// are not reserved identities, and reading them as such would invent a harbour
// this scheme does not have.
type IMEI struct{}

// Contract checks. There is deliberately no RangeAware here, for the reason
// given above.
var (
	_ provider.Provider     = IMEI{}
	_ provider.CheckDigited = IMEI{}
)

const (
	// imeiLength is fifteen digits, fourteen carrying meaning and one
	// checking them.
	imeiLength = 15
	// imeiBodyLength is what the check digit is computed over.
	imeiBodyLength = 14
	// imeiCheckPosition is the last position, counted from zero.
	imeiCheckPosition = 14
	// imeiModulus is the divisor of the Luhn formula.
	imeiModulus = 10
	// imeiSoftwareLength is the length of the sixteen digit form carrying a
	// software version number. It is here so that form can be recognised and
	// named rather than called malformed.
	imeiSoftwareLength = 16
)

// imeiDigits is the alphabet of this scheme, which is numeric throughout.
//
// Spelled out rather than shared with the article number, for the reason that
// one gives for not sharing with the vehicle number: the two are the same ten
// characters for unrelated reasons, and binding them would let a change to one
// scheme move another silently.
const imeiDigits = "0123456789"

// imeiTestBlock is the reporting body identifier GSMA TS.06 table 1 gives to
// test equipment rather than to a commercial allocator.
//
// This is where generation happens. It is not a safe harbour and the type
// comment says why, so this constant is named for what the source calls it
// instead of being called a reserved range.
const imeiTestBlock = "00"

// imeiDrawnLength and imeiDrawnSpace are the digits between the test block and
// the check digit.
//
// Twelve digits in one draw rather than twelve draws, for the reason the
// article number gives: a draw needs a coordinate of its own and twelve of them
// for one value would cost more than the value is worth. A million million sits
// comfortably inside an int64.
const (
	imeiDrawnLength = imeiBodyLength - len(imeiTestBlock)
	imeiDrawnSpace  = 1_000_000_000_000
)

// ID names this provider.
//
// No country prefix, because equipment identities are allocated globally by one
// body rather than by a country. The catalog published this name before this
// file existed, in the applies_to of both checksum entries, so it is read from
// there rather than chosen here.
func (IMEI) ID() string { return "imei" }

// Capabilities are what this provider brings.
//
// One. The check digit is what the catalog asks of this scheme. There is no
// reserved range for the reason set out on the type, and no claim about dates
// or gender, because an equipment identity encodes a model and a serial number
// and neither is a value a recipe holds in another column.
// Mask is fifteen digits. The sixteen digit form carrying a software version
// is recognised and never generated, so it is not what this describes.
func (IMEI) Mask() string { return strings.Repeat(string(provider.MaskDigit), imeiLength) }

func (IMEI) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Generate produces one equipment identity.
//
// The agreement is not read. Nothing in this scheme is bound to another column
// of the row.
//
// # Why the value starts inside the test block
//
// This scheme has no safe harbour, so the honest default would be the one the
// vehicle number takes: draw anywhere well formed and let the dataset be marked
// synthetic. Starting inside the block GSMA reserves for test equipment costs
// nothing and buys something real, because equipment carrying a test identity
// cannot be supplied to the market. A value drawn here therefore cannot be a
// device somebody owns, even though it can still collide with a prototype.
//
// # Why the two digits after the block are drawn rather than fixed
//
// Clause 9.1 of TS.06 spells a test identity as the block, then 44 naming the
// allocating body, then a manufacturer code. Fixing those two digits would make
// every value this tool writes claim a named allocating body and a manufacturer
// code that may well be issued. Drawing them means the value sits in test space
// without asserting who allocated it, which claims less and is the safer of the
// two.
func (i IMEI) Generate(req provider.Request) (string, error) {
	value := make([]byte, 0, imeiLength)
	value = append(value, imeiTestBlock...)
	value = appendFrom(value, imeiDigits,
		req.Source.IntN(imeiAt(req, "imei-body"), imeiDrawnSpace), imeiDrawnLength)
	// A place for the check digit, so the value is fifteen characters before
	// the arithmetic reads it.
	value = append(value, '0')

	digit, ok := imeiCheckDigit(value)
	if !ok {
		// Unreachable while the draws come from digits, and loud rather than
		// silent if that ever stops being true.
		return "", fmt.Errorf("imei: the drawn characters do not form a value this scheme can complete")
	}
	value[imeiCheckPosition] = digit
	return string(value), nil
}

// Validate answers whether a value is a correct fifteen digit equipment
// identity.
//
// It is written from the rules of the specification rather than from what
// Generate happens to produce, and it is deliberately wider: a well formed
// identity outside the test block is somebody's real value and calling it wrong
// would be a defect reported against data that has none. That asymmetry is the
// shape the social security number, the vehicle number and the article number
// already have.
//
// # The sixteen digit form is recognised and not checked
//
// Clause 6.2.2 defines a sixteen digit form carrying a software version number,
// and the note under clause 6.2.1 says the check digit is not applied to it. So
// a sixteen digit value is a form of this scheme that has nothing to check.
// Calling it invalid would report a defect that is not there, and calling it
// valid would approve something nobody verified. Unsupported is the verdict
// that says which of those it is, and it also stops the corruption engine
// aiming a check digit failure at a value that has no check digit.
func (IMEI) Validate(value string) provider.Verdict {
	if len(value) == imeiSoftwareLength && !imeiHasStrangeCharacter(value) {
		return provider.Unsupported(
			"this is the sixteen digit form carrying a software version number, " +
				"and the specification does not apply a check digit to it, so there is nothing here to check")
	}
	if len(value) != imeiLength {
		return provider.Invalid(fmt.Sprintf(
			"an equipment identity is fifteen digits and %s is %d",
			provider.Show(value), len(value)))
	}
	if at, ok := imeiStrangeCharacter(value); ok {
		return provider.Invalid(fmt.Sprintf(
			"position %d of %s is not a digit, and this scheme is numeric throughout",
			at+1, provider.Show(value)))
	}
	want, ok := imeiCheckDigit([]byte(value))
	if !ok {
		return provider.Invalid(fmt.Sprintf(
			"no check digit completes %s", provider.Show(value)))
	}
	if got := value[imeiCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf(
			"the check digit is %s and should be %s",
			provider.Show(string(got)), provider.Show(string(want))))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
func (IMEI) CheckDigits(value string) ([]int, bool) {
	if len(value) != imeiLength {
		return nil, false
	}
	return []int{imeiCheckPosition}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
//
// It never refuses for arithmetic reasons. Every fourteen digit body has a
// completion, because ten residues fit in ten digits, so a mutator that moved a
// digit always gets a consistent value back. That is the article number's
// property rather than the vehicle number's, and it comes from the modulus
// being ten rather than eleven.
func (IMEI) WithCheckDigits(value string) (string, error) {
	if len(value) != imeiLength {
		return "", fmt.Errorf("imei: %s is not fifteen digits", provider.Show(value))
	}
	rebuilt := []byte(value)
	digit, ok := imeiCheckDigit(rebuilt)
	if !ok {
		return "", fmt.Errorf(
			"imei: %s carries a character that is not a digit, so no check digit completes it",
			provider.Show(value))
	}
	rebuilt[imeiCheckPosition] = digit
	return string(rebuilt), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// imeiCheckDigit computes the last digit over the fourteen before it.
//
// The specification labels those fourteen D14 down to D1 and doubles the odd
// labelled ones. D1 is the digit next to the check digit, so walking from the
// right and doubling every second character starting with the first is the same
// instruction said positionally. The digits of a doubled value are added rather
// than the value itself, which for a number between ten and eighteen is the
// value less nine.
func imeiCheckDigit(value []byte) (byte, bool) {
	if len(value) != imeiLength {
		return 0, false
	}
	sum := 0
	for back := 0; back < imeiBodyLength; back++ {
		symbol := value[imeiBodyLength-1-back]
		if !isDigit(symbol) {
			return 0, false
		}
		digit := int(symbol - '0')
		if back%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	// The next higher number ending in nought, said as a fold. A total already
	// ending in nought stays where it is, which is how a check digit of nought
	// arises and is the case the specification names out loud.
	return byte('0' + (imeiModulus-sum%imeiModulus)%imeiModulus), true
}

// imeiStrangeCharacter returns the first position holding something that is not
// a digit, and whether there is one.
func imeiStrangeCharacter(value string) (int, bool) {
	for at := 0; at < len(value); at++ {
		if !isDigit(value[at]) {
			return at, true
		}
	}
	return 0, false
}

// imeiHasStrangeCharacter is the same question without the position, for the
// caller that only needs to know whether a value is numeric throughout.
func imeiHasStrangeCharacter(value string) bool {
	_, strange := imeiStrangeCharacter(value)
	return strange
}

// imeiAt is the address the draw is derived from.
//
// It carries its own purpose, so this value cannot come out of the same number
// as another field of the same row, and the value of a row does not depend on
// the order the rows were asked for.
func imeiAt(req provider.Request, purpose string) seed.Coordinate {
	return seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":" + purpose,
	}
}
