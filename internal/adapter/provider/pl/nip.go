package pl

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// NIP is the Polish tax identification number.
//
// # Where the rules came from
//
// Read at the country sheet the Ministry of Finance of Poland publishes on the
// European TIN Portal, which carries its copyright notice, rather than recalled
// or taken from somebody's validator. The sheet does something the description
// of the PESEL never did: it works one number out by hand, digit by digit. That
// example is quoted below and stands as a test, so the weights here are checked
// against an official answer instead of resting on prose alone.
//
// The number is ten digits. The first three are the code of the tax office that
// issued it, the next six are a serial and the tenth is a check digit over the
// first nine with the weights 6 5 7 2 3 4 5 6 7, modulo eleven.
//
// # The hole modulo eleven leaves, which is a property of the scheme
//
// Eleven residues and ten digits do not fit. The source says so outright: the
// check digit is the remainder, and if the remainder is ten the number is not a
// valid one. So roughly one nine digit body in eleven can never begin a NIP at
// all, whatever is written after it. Two consequences run through this file.
// Generation never draws such a body, by choosing the ninth digit only among
// those that leave a remainder a digit can hold. Recomputing refuses, with a
// reason, rather than nudging some other digit to make the arithmetic work,
// because that would be a second change nobody asked for.
//
// # What the scheme does not carry
//
// No date, no gender, so nothing in a record can be made to agree with it and
// this provider claims neither capability. No reserved range either, and that
// absence is the expensive one: no source names a block of tax numbers that is
// never issued, and the first three digits are the code of a real office. Every
// well formed NIP could therefore belong to a real company. Inventing a block
// that merely looks free is the mistake this project already made once, when a
// Social Security range that looked obviously spare turned out to be the space
// of a real taxpayer identifier. The defence here is the same as for the PESEL:
// a dataset carrying these has to be marked synthetic in its own metadata.
type NIP struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider     = NIP{}
	_ provider.CheckDigited = NIP{}
)

const (
	// nipLength is the number of digits and nipCheckPosition is the index of
	// the check digit. nipBodyLength is what the check digit is computed over.
	nipLength        = 10
	nipCheckPosition = 9
	nipBodyLength    = nipCheckPosition
	// nipModulus is the eleven the source divides by, and nipNoCheckDigit is
	// the remainder no digit can hold.
	nipModulus      = 11
	nipNoCheckDigit = 10

	// nipOfficeDigits is how many digits each place of the office code is
	// drawn from, one to nine, and nipOfficeCodes is how many codes that makes
	// over its three places.
	//
	// # Why no nought, when the scheme's arithmetic does not care
	//
	// The first three digits are the code of the tax office, and two secondary
	// sources read on 2026-10-21 (infor.pl and fakturownia.pl, both describing
	// the NIP to their readers) say such a code is made of the digits one to
	// nine, one of them adding that a nought has been allowed in the second
	// place in exceptional cases since 2014. No official list of codes was
	// found. Before that day a code was drawn from all thousand, and 368 of 2000
	// numbers carried a nought in the first or third place, which an
	// application checking the office would refuse although the check digit
	// was right. Drawing from the strictest reading costs nothing, because a
	// number valid under it is valid under the looser one as well.
	//
	// 🔴 Validate is deliberately not narrowed the same way. It reads numbers
	// out of somebody else's file, a real NIP with a nought in the second place
	// exists by the second source, and refusing it on the strength of two
	// secondary sources would be a defect reported in a value that has none.
	nipOfficeDigits = 9
	nipOfficeCodes  = nipOfficeDigits * nipOfficeDigits * nipOfficeDigits
	// nipSerialLength is how many digits are drawn after the office and before
	// the ninth, which is chosen, and nipSerialCodes is how many values they
	// can take.
	nipSerialLength = 5
	nipSerialCodes  = 100000
)

// nipWeights are applied to the first nine digits, in the order the source
// lists them as C1 to C9.
var nipWeights = [nipBodyLength]int{6, 5, 7, 2, 3, 4, 5, 6, 7}

// nipGroupings are the two ways Poland writes the same ten digits.
//
// The source knows neither of them: it says ten numeric characters and stops.
// Accepting both is this project's decision, and the reason is that in the take
// it and break it mode the column arrives from somebody else's file, where a
// tax number on an invoice is nearly always written with separators. Refusing
// those would report a defect in a value that has none.
//
// The emitted form is the bare ten digits, which is what the catalog examples
// carry, and the form a value arrives in survives the corruption untouched.
var nipGroupings = [][]int{
	{3, 3, 2, 2}, // 123-456-32-18
	{3, 2, 2, 3}, // 123-45-63-218
}

// ID names this provider. The country prefix is ISO 3166-1 alpha-2, because a
// provider belongs to a country rather than to a language.
func (NIP) ID() string { return "PL.nip" }

// Capabilities are what this provider brings.
//
// One, and the absences are as deliberate as the presence. See the type comment
// for why there is no date, no gender and no reserved range.
// Mask is ten digits with no punctuation. The scheme is also written in
// groups, and this build does not produce it that way, so the mask does not
// claim it does.
func (NIP) Mask() string { return strings.Repeat(string(provider.MaskDigit), nipLength) }

func (NIP) Capabilities() []capability.Name {
	return []capability.Name{capability.IdentifierCheckDigit}
}

// Generate produces one tax number for the place described by the request.
//
// The agreement is not read, which is a property of the scheme rather than an
// omission: a NIP encodes nothing about its holder, so there is nothing in it
// that could be made to agree with a date of birth or a gender. A recipe
// binding one to this field is asking for something the scheme cannot express,
// and answering that belongs to the use case that can see both sides.
func (n NIP) Generate(req provider.Request) (string, error) {
	draw := func(purpose string, bound int64) int {
		return int(req.Source.IntN(seed.Coordinate{
			Entity:  req.At.Entity,
			Row:     req.At.Row,
			Field:   req.At.Field,
			Purpose: req.At.Purpose + purpose,
		}, bound))
	}

	// The office comes first, then five digits drawn as one number, then the
	// ninth is chosen only from the values that leave a remainder a digit can
	// hold. There is no retry loop: the choice is made from a list that is never
	// empty, so the answer stays one draw per place rather than an unknown
	// number of them.
	digits := make([]int, 0, nipLength)
	office := draw(":nip-office", nipOfficeCodes)
	for divisor := nipOfficeCodes / nipOfficeDigits; divisor > 0; divisor /= nipOfficeDigits {
		digits = append(digits, 1+office/divisor%nipOfficeDigits)
	}
	digits = appendFixed(digits, draw(":nip-body", nipSerialCodes), nipSerialLength)
	allowed := nipAllowedNinthDigits(digits)
	digits = append(digits, allowed[draw(":nip-ninth", int64(len(allowed)))])

	check, ok := nipCheckDigit(digits)
	if !ok {
		// Unreachable by construction, and it is here rather than assumed. A
		// silent wrong value from an identifier generator is the failure this
		// whole file exists to make impossible.
		return "", fmt.Errorf("nip: %s leaves a remainder of %d, which the ninth digit was chosen to avoid",
			digitsToString(digits), nipNoCheckDigit)
	}
	return digitsToString(append(digits, check)), nil
}

// Validate answers whether a value is a correct NIP.
//
// It is written from the rules of the scheme rather than from what Generate
// happens to produce. Checking a generator with its own arithmetic would make
// one mistake twice and let the two agree with each other.
func (NIP) Validate(value string) provider.Verdict {
	parsed, ok := parseNIP(value)
	if !ok {
		return provider.Invalid("a NIP is ten digits, written 1234563218, 123-456-32-18 or 123-45-63-218")
	}
	want, ok := nipCheckDigit(parsed.digits[:nipCheckPosition])
	if !ok {
		return provider.Invalid("the first nine digits leave a remainder of ten, and ten is not a digit, so no NIP starts this way")
	}
	if got := parsed.digits[nipCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf("the check digit is %d and should be %d", got, want))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
//
// It is the last character in all three accepted forms, so the offset follows
// the shape the value was written in rather than a fixed position that would be
// wrong for two of them.
func (NIP) CheckDigits(value string) ([]int, bool) {
	if _, ok := parseNIP(value); !ok {
		return nil, false
	}
	return []int{len(value) - 1}, true
}

// WithCheckDigits returns the value with its check digit recomputed, in the
// shape it arrived in.
//
// It refuses when the first nine digits are ones no NIP can start with, and
// that refusal is an answer about applicability rather than a guard on the
// method. A mutator that moved a digit into that hole has produced something
// the scheme cannot make consistent again, and the manifest has to say so
// instead of receiving a value quietly patched somewhere else.
func (NIP) WithCheckDigits(value string) (string, error) {
	parsed, ok := parseNIP(value)
	if !ok {
		return "", fmt.Errorf("nip: %s is not ten digits", provider.Show(value))
	}
	check, ok := nipCheckDigit(parsed.digits[:nipCheckPosition])
	if !ok {
		return "", fmt.Errorf("nip: the first nine digits of %s leave a remainder of %d, and %d is not a digit, so no check digit makes this a NIP",
			provider.Show(value), nipNoCheckDigit, nipNoCheckDigit)
	}
	parsed.digits[nipCheckPosition] = check
	return parsed.String(), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// nipCheckDigit computes the tenth digit over the first nine, and says whether
// one exists at all.
//
// The second result is the whole difference between this scheme and a modulo
// ten one. The source states it plainly: the check digit is the remainder, and
// a remainder of ten means the number is not valid. A version returning only
// the digit would have to invent an answer for one body in eleven.
func nipCheckDigit(body []int) (int, bool) {
	sum := 0
	for position, digit := range body {
		sum += digit * nipWeights[position]
	}
	if remainder := sum % nipModulus; remainder != nipNoCheckDigit {
		return remainder, true
	}
	return 0, false
}

// nipAllowedNinthDigits returns the values the ninth digit may take so that a
// check digit exists, in ascending order.
//
// At most one value is ever excluded, so the answer holds nine or ten of them
// and is never empty. The arithmetic reason is that the ninth weight is seven,
// which is invertible modulo eleven, so exactly one residue is forbidden and it
// is a digit only some of the time. It is worked out by trying all ten rather
// than by inverting anything, because a modular inverse written down wrongly
// looks exactly like one written down correctly.
func nipAllowedNinthDigits(firstEight []int) []int {
	body := make([]int, nipBodyLength)
	copy(body, firstEight)

	allowed := make([]int, 0, 10)
	for digit := 0; digit <= 9; digit++ {
		body[nipBodyLength-1] = digit
		if _, ok := nipCheckDigit(body); ok {
			allowed = append(allowed, digit)
		}
	}
	return allowed
}

// --- the number, kept together with the shape it arrived in --------------------

// nipValue is a parsed NIP.
//
// It keeps the grouping the value was written with so that a corruption hands
// back the shape it was given. Reformatting on the way through would be a
// second change nobody asked for, and it would show in a diff as though the
// corruption had done more than it did.
type nipValue struct {
	digits   [nipLength]int
	grouping []int // nil when the value was written as ten bare digits
}

// String writes the number out in the shape it arrived in.
func (v nipValue) String() string {
	if v.grouping == nil {
		return digitsToString(v.digits[:])
	}
	var built strings.Builder
	built.Grow(nipLength + len(v.grouping) - 1)
	at := 0
	for index, width := range v.grouping {
		if index > 0 {
			built.WriteByte('-')
		}
		for offset := 0; offset < width; offset++ {
			built.WriteByte(byte('0' + v.digits[at+offset]))
		}
		at += width
	}
	return built.String()
}

// parseNIP reads any of the three accepted ways of writing the number and
// refuses everything else.
func parseNIP(value string) (nipValue, bool) {
	if digits, ok := parseDigits(value); ok && len(digits) == nipLength {
		return newNIPValue(digits, nil), true
	}
	for _, grouping := range nipGroupings {
		if digits, ok := parseGrouped(value, grouping); ok {
			return newNIPValue(digits, grouping), true
		}
	}
	return nipValue{}, false
}

func newNIPValue(digits []int, grouping []int) nipValue {
	parsed := nipValue{grouping: grouping}
	copy(parsed.digits[:], digits)
	return parsed
}

// parseGrouped reads the digits of one hyphenated form, and refuses a value
// whose separators sit anywhere other than between the named groups.
func parseGrouped(value string, grouping []int) ([]int, bool) {
	digits := make([]int, 0, nipLength)
	at := 0
	for index, width := range grouping {
		if index > 0 {
			if at >= len(value) || value[at] != '-' {
				return nil, false
			}
			at++
		}
		part, ok := parseDigits(sliceOf(value, at, width))
		if !ok || len(part) != width {
			return nil, false
		}
		digits = append(digits, part...)
		at += width
	}
	if at != len(value) {
		return nil, false
	}
	return digits, true
}

// sliceOf returns width bytes from at, or the empty string when the value is
// too short. Asking for the substring directly would panic, and a value from
// somebody else's file is not allowed to do that.
func sliceOf(value string, at, width int) string {
	if at < 0 || at+width > len(value) {
		return ""
	}
	return value[at : at+width]
}
