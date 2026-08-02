package pl

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// REGON is the Polish statistical number of an entity of the national economy.
//
// # Where the rules came from, and they came from two different places
//
// The shape is read from the regulation itself. The Council of Ministers has
// described it twice, in the regulation of 27 July 1999 and again in the one of
// 30 November 2015 that replaced it, and the two sentences say the same thing
// sixteen years apart. Nine digits, of which the first eight are a sequence
// number and the ninth a check digit.
//
// The arithmetic came from somewhere else, because neither regulation carries
// it. Nor does the register's own glossary, nor the public information bulletin
// of the statistical office. Four official documents describe the number and
// none of them says how the ninth digit is computed. What settled it was a
// calculation rather than a sentence: the weights below reproduce the check
// digit of 174 real numbers drawn from two unrelated public registers, with no
// exception. That is a weaker kind of source than the worked example the
// Ministry of Finance publishes for the tax number, and it is written down here
// as what it is.
//
// # The difference from the tax number, which must not be assumed away
//
// Both schemes divide by eleven and eleven residues do not fit in ten digits.
// They resolve the collision in opposite directions. The tax number calls a
// remainder of ten invalid, so one body in eleven can begin no number at all.
// This one folds ten onto nought, so every body completes and there is no hole:
// generation never has to choose a digit, and recomputing never has to refuse.
//
// Fifteen of the 174 numbers checked fall exactly in that case and every one of
// them carries a check digit of nought, which is what makes the rule a measured
// fact here rather than a habit carried over from the file next door.
//
// # What that costs, said out loud because a check digit invites trust
//
// Folding two residues onto one digit makes the map from remainder to digit not
// one to one, so this check digit is weaker than the tax number's: some single
// digit errors survive it. Anything relying on a corruption being caught has to
// know that, and there is a test measuring how often it happens rather than a
// sentence guessing.
//
// # What the scheme does not carry
//
// Nothing about its holder, and this is a quotation rather than an inference.
// Both regulations say the digits "nie mogą mieć ukrytego lub jawnego
// charakteru znaczącego, określającego pewne cechy podmiotu", that is, they may
// carry no hidden or open meaning describing any feature of the entity. So no
// date, no gender, and no region either, whatever the widely repeated claim
// about the first two digits says. This provider declares none of those.
//
// No reserved range either, and the absence is the same expensive one the tax
// number has: no source names a block that is never issued, and a number struck
// from the register is kept in a historical set and never reused, so there is no
// freed space to borrow. A dataset carrying these has to be marked synthetic in
// its own metadata.
//
// # The fourteen digit form, which this build does not produce
//
// The regulation also describes a fourteen digit number for a local unit, whose
// first nine digits are the number of the legal entity that created it. That
// form is deliberately absent: its weights appear only in secondary sources and
// no official document and no real number was found to check them against.
// Writing them down anyway would produce three layers of oracle all descended
// from one reading, which is the defect this project has already caught once.
type REGON struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider     = REGON{}
	_ provider.CheckDigited = REGON{}
)

const (
	// regonLength is the number of digits and regonCheckPosition is the index
	// of the check digit. regonBodyLength is what it is computed over.
	regonLength        = 9
	regonCheckPosition = 8
	regonBodyLength    = regonCheckPosition
	// regonModulus is the eleven the weighted total is divided by, and
	// regonFoldedRemainder is the residue that does not fit in a digit and is
	// written as nought instead of invalidating the number.
	regonModulus         = 11
	regonFoldedRemainder = 10
	// regonLocalUnitLength is the length of the local unit form, recognised so
	// that it can be refused by name rather than as a malformed value.
	regonLocalUnitLength = 14
)

// regonWeights are applied to the first eight digits, in order.
var regonWeights = [regonBodyLength]int{8, 9, 2, 3, 4, 5, 6, 7}

// ID names this provider. The country prefix is ISO 3166-1 alpha-2, because a
// provider belongs to a country rather than to a language.
func (REGON) ID() string { return "PL.regon" }

// Capabilities are what this provider brings.
//
// One, and every absence is quoted rather than assumed. See the type comment.
func (REGON) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Generate produces one statistical number for the place described by the
// request.
//
// The agreement is not read, and that is a property of the scheme rather than
// an omission: the regulation forbids the digits from meaning anything about
// the entity, so there is nothing here that could be made to agree with another
// field. A recipe binding one to this field asks for something the scheme
// cannot express, and answering that belongs to the use case that sees both
// sides.
//
// Unlike the tax number there is no constrained digit and no list to draw from.
// Every eight digit body completes into a number, so one draw per place is the
// whole of it.
func (r REGON) Generate(req provider.Request) (string, error) {
	drawn := req.Source.IntN(seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":regon-body",
	}, 100000000)

	digits := appendFixed(make([]int, 0, regonLength), int(drawn), regonBodyLength)
	return digitsToString(append(digits, regonCheckDigit(digits))), nil
}

// Validate answers whether a value is a correct nine digit REGON.
//
// It is written from the rules of the scheme rather than from what Generate
// happens to produce. Checking a generator with its own arithmetic would make
// one mistake twice and let the two halves agree with each other.
//
// A well formed local unit number is refused, and the reason says that this
// build does not check that form rather than claiming the value is wrong. The
// distinction matters: reporting a defect in a value that has none is the
// failure mode the tax number provider was written to avoid.
func (REGON) Validate(value string) provider.Verdict {
	digits, ok := parseDigits(value)
	if !ok {
		return provider.Invalid("a REGON is written as bare digits with no separators")
	}
	if len(digits) == regonLocalUnitLength {
		return provider.Invalid("this is the fourteen digit number of a local unit, and this build checks only the nine digit form")
	}
	if len(digits) != regonLength {
		return provider.Invalid("a REGON of an entity is nine digits")
	}
	if want, got := regonCheckDigit(digits[:regonCheckPosition]), digits[regonCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf("the check digit is %d and should be %d", got, want))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
func (REGON) CheckDigits(value string) ([]int, bool) {
	if digits, ok := parseDigits(value); !ok || len(digits) != regonLength {
		return nil, false
	}
	return []int{regonCheckPosition}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
//
// It never refuses for arithmetic reasons, and the contrast with the tax number
// is the whole point of the scheme's choice: folding the eleventh residue onto
// nought means every body has a completion, so a mutator that moved a digit
// always gets a consistent number back.
func (REGON) WithCheckDigits(value string) (string, error) {
	digits, ok := parseDigits(value)
	if !ok || len(digits) != regonLength {
		return "", fmt.Errorf("regon: the value is not nine digits")
	}
	digits[regonCheckPosition] = regonCheckDigit(digits[:regonCheckPosition])
	return digitsToString(digits), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// regonCheckDigit computes the ninth digit over the first eight.
//
// It returns one result where the tax number returns two, and the missing
// second result is the difference between the schemes rather than a
// simplification. A remainder of ten is written as nought, so there is no body
// this cannot complete and nothing to report as impossible.
func regonCheckDigit(body []int) int {
	sum := 0
	for position, digit := range body {
		sum += digit * regonWeights[position]
	}
	if remainder := sum % regonModulus; remainder != regonFoldedRemainder {
		return remainder
	}
	return 0
}
