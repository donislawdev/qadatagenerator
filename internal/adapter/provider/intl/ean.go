package intl

import (
	"strings"

	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// EAN is the thirteen digit article number, the GTIN-13 of the GS1 system.
//
// # Where the rules come from
//
// The GS1 General Specifications, the standard GS1 publishes and keeps current.
// Release 26.0 was read at ref.gs1.org on 2026-09-06. Unlike the vehicle number,
// no detour through a regulation was needed: GS1 publishes this one itself.
//
// A second and independent reading of the same arithmetic comes from the ISBN
// Users' Manual of the International ISBN Agency, because an ISBN-13 is a
// GTIN-13 carrying the prefix 978 or 979. Two first party documents state the
// same rule in different words and each works its own example, which is a
// firmer footing than any scheme here has had so far.
//
// # The weights are anchored at the right, and that matters more than it looks
//
// Table 7-8 lays the GTIN-8, GTIN-12, GTIN-13, GTIN-14 and the longer fields
// against a single row of multipliers, all of them aligned to the right edge.
// The character immediately left of the check digit always weighs three. For
// the twelve digits of this scheme that reads one, three, one, three and so on
// from the left, and for a seventeen digit body it reads three, one, three.
//
// Taking the weights from the left instead would give the right answer here and
// the wrong answer for the longer fields, so the mistake would be invisible in
// this file and would surface only when some later scheme reused the reasoning.
// That is why the rule is written down as the standard states it rather than as
// this one scheme happens to need it.
//
// # The residue needs no special case
//
// Three schemes in this build fold the eleventh residue of a modulus eleven sum
// and each answers differently, so a reader arriving from those may expect a
// fourth answer here. There is none to give. The modulus is ten, ten residues
// fit in ten digits, and the only wrinkle is that a computed ten is written as
// nought. The ISBN manual names that case out loud rather than leaving it to be
// inferred.
//
// # What this generates, and the narrower promise it keeps
//
// GS1 sets the prefix 952 aside for demonstrations and examples of its own
// system, in both the prefix table and the eight digit prefix table, and it
// puts every worked example in its current specification inside that prefix.
// A number there cannot belong to a real article, because no company is issued
// a prefix from it. So this scheme has a safe harbour, generation stays inside
// it, and it is the second scheme here to manage that after the social security
// number.
//
// Validate is deliberately wider and accepts any well formed thirteen digit
// article number. That asymmetry is the shape the social security number and
// the vehicle number already have: generate only where nothing can be claimed,
// but do not call somebody else's real value wrong.
//
// # A caution for whoever revisits the harbour
//
// The prefix 952 was ordinary company prefix space as recently as release
// 21.0.1 of January 2021, and the release that carved it out for demonstrations
// is release 23.0.0 of January 2023. Whether any company prefix was issued
// there before that has not been established. What supports the harbour is that
// GS1 annotates history where it exists, as it does for the neighbouring prefix
// 951, writes nothing of the kind here, and stands its own examples on it.
// Those are two agreeing indications and not a register of what was issued, and
// docs/05 says so in the same words.
type EAN struct{}

// Contract checks.
var (
	_ provider.Provider     = EAN{}
	_ provider.CheckDigited = EAN{}
	_ provider.RangeAware   = EAN{}
)

const (
	// eanLength is thirteen digits.
	eanLength = 13
	// eanBodyLength is what the check digit is computed over.
	eanBodyLength = 12
	// eanCheckPosition is the last position, counted from zero.
	eanCheckPosition = 12
	// eanModulus is the divisor of the standard check digit algorithm.
	eanModulus = 10
	// eanHeavyWeight and eanLightWeight are the two multipliers of Table 7-8.
	// The heavy one falls on the character next to the check digit.
	eanHeavyWeight = 3
	eanLightWeight = 1
)

// eanDigits is the alphabet of this scheme, which is numeric throughout.
//
// It is spelled out here rather than shared with the tail of the vehicle number
// because the two are the same ten characters for unrelated reasons. Binding
// them together would make a change to one scheme's alphabet silently move the
// other, which is the kind of coupling this project has already paid for once.
const eanDigits = "0123456789"

// eanDemonstrationPrefix is the GS1 prefix set aside for demonstrations and
// examples of the GS1 system, from Table 1-4 of release 26.0.
//
// This is the safe harbour and it is the only place this scheme generates.
const eanDemonstrationPrefix = "952"

// eanDrawnLength and eanDrawnSpace are the free digits between the prefix and
// the check digit.
//
// Nine digits are drawn in one draw rather than nine, because a draw needs a
// coordinate and nine of them for one value would cost more than the value is
// worth. The bound is a thousand million, comfortably inside an int64.
const (
	eanDrawnLength = eanBodyLength - len(eanDemonstrationPrefix)
	eanDrawnSpace  = 1_000_000_000
)

// ID names this provider.
//
// No country prefix, because the scheme is international. The catalog published
// this name before this file existed, in the applies_to of both checksum
// entries, so it is read from there rather than chosen here.
func (EAN) ID() string { return "ean" }

// Capabilities are what this provider brings.
//
// Two, and the second is the rarer one. The check digit is what the catalog
// asks of this scheme. The reserved range is the claim that everything this
// tool writes sits where nothing real can sit, and it is the machine readable
// form of the harbour: docs/05 has a table saying the same thing for a reader,
// but the declaration below is the only copy a script can count.
//
// There is no claim to write into an unassigned block. No catalog entry aims
// that corruption at this scheme, and naming blocks without being sure of them
// would be a promise made to a manifest that nothing here could keep.
// Masks are thirteen digits, unpunctuated. The bars printed above them are a
// rendering of the number rather than part of it.
func (EAN) Masks() []string { return []string{strings.Repeat(string(provider.MaskDigit), eanLength)} }

func (EAN) Capabilities() []capability.Name {
	return []capability.Name{
		capability.IdentifierCheckDigit,
		capability.IdentifierReservedRange,
	}
}

// Generate produces one article number.
//
// The agreement is not read. Nothing in this scheme is bound to another column:
// the digits after the prefix identify a company and an article within it, and
// neither is a value a recipe holds elsewhere.
func (e EAN) Generate(req provider.Request) (string, error) {
	value := make([]byte, 0, eanLength)
	value = append(value, eanDemonstrationPrefix...)
	value = appendFrom(value, eanDigits,
		req.Source.IntN(eanAt(req, "ean-body"), eanDrawnSpace), eanDrawnLength)
	// A place for the check digit, so the value is thirteen characters before
	// the arithmetic reads it.
	value = append(value, '0')

	digit, ok := eanCheckDigit(value)
	if !ok {
		// Unreachable while the draws come from digits, and loud rather than
		// silent if that ever stops being true.
		return "", fmt.Errorf("ean: the drawn characters do not form a value this scheme can complete")
	}
	value[eanCheckPosition] = digit
	return string(value), nil
}

// Validate answers whether a value is a correct thirteen digit article number.
//
// It is written from the rules of the standard rather than from what Generate
// happens to produce. Checking a generator with its own arithmetic would make
// one mistake twice and let the two halves agree with each other.
func (EAN) Validate(value string) provider.Verdict {
	if len(value) != eanLength {
		return provider.Invalid(fmt.Sprintf(
			"an article number is thirteen digits and %s is %d",
			provider.Show(value), len(value)))
	}
	if at, ok := eanStrangeCharacter(value); ok {
		return provider.Invalid(fmt.Sprintf(
			"position %d of %s is not a digit, and this scheme is numeric throughout",
			at+1, provider.Show(value)))
	}
	want, ok := eanCheckDigit([]byte(value))
	if !ok {
		return provider.Invalid(fmt.Sprintf(
			"no check digit completes %s", provider.Show(value)))
	}
	if got := value[eanCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf(
			"the check digit is %s and should be %s",
			provider.Show(string(got)), provider.Show(string(want))))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
func (EAN) CheckDigits(value string) ([]int, bool) {
	if len(value) != eanLength {
		return nil, false
	}
	return []int{eanCheckPosition}, true
}

// WithCheckDigits returns the value with its check digit recomputed.
//
// It never refuses for arithmetic reasons. Every twelve digit body has a
// completion, because ten residues fit in ten digits, so a mutator that moved a
// digit always gets a consistent value back.
func (EAN) WithCheckDigits(value string) (string, error) {
	if len(value) != eanLength {
		return "", fmt.Errorf("ean: %s is not thirteen digits", provider.Show(value))
	}
	rebuilt := []byte(value)
	digit, ok := eanCheckDigit(rebuilt)
	if !ok {
		return "", fmt.Errorf(
			"ean: %s carries a character that is not a digit, so no check digit completes it",
			provider.Show(value))
	}
	rebuilt[eanCheckPosition] = digit
	return string(rebuilt), nil
}

// InReservedRange reports whether a value sits in the demonstration prefix.
//
// This is the claim the tool makes about its own output, so it is strict about
// the shape as well as the prefix. A string that is not an article number at
// all is not sitting safely anywhere.
func (EAN) InReservedRange(value string) bool {
	if len(value) != eanLength {
		return false
	}
	if _, strange := eanStrangeCharacter(value); strange {
		return false
	}
	return value[:len(eanDemonstrationPrefix)] == eanDemonstrationPrefix
}

// ReservedRanges describes the harbour in words, for the report that has to
// tell a user why their safe data is refused by an application checking ranges.
func (EAN) ReservedRanges() []string {
	return []string{
		"numbers beginning 952, which GS1 sets aside for demonstrations and examples of its own system. " +
			"No company is issued a prefix there, so a number in this range cannot belong to a real article. " +
			"An application that only accepts article numbers it can find in a product database will reject these, " +
			"and that rejection is the range working rather than a fault in the data.",
	}
}

// --- the arithmetic, kept in one place ---------------------------------------

// eanCheckDigit computes the last digit over the twelve before it.
//
// The steps are those of Table 7-8: multiply each position by its weight, add,
// then subtract the sum from the nearest equal or higher multiple of ten. The
// weights are taken by walking the body from its right hand end, which is how
// the standard aligns them and is what keeps the rule true for the longer
// fields as well as for this one.
func eanCheckDigit(value []byte) (byte, bool) {
	if len(value) != eanLength {
		return 0, false
	}
	sum := 0
	for back := 0; back < eanBodyLength; back++ {
		symbol := value[eanBodyLength-1-back]
		if !isDigit(symbol) {
			return 0, false
		}
		weight := eanLightWeight
		if back%2 == 0 {
			weight = eanHeavyWeight
		}
		sum += int(symbol-'0') * weight
	}
	// The nearest equal or higher multiple of ten, said without a special case
	// for a sum that is already one. A computed ten is written as nought.
	return byte('0' + (eanModulus-sum%eanModulus)%eanModulus), true
}

// eanStrangeCharacter returns the first position holding something that is not
// a digit, and whether there is one.
func eanStrangeCharacter(value string) (int, bool) {
	for at := 0; at < len(value); at++ {
		if !isDigit(value[at]) {
			return at, true
		}
	}
	return 0, false
}

// eanAt is the address the draw is derived from.
//
// It carries its own purpose, so this value cannot come out of the same number
// as another field of the same row, and the value of a row does not depend on
// the order the rows were asked for.
func eanAt(req provider.Request, purpose string) seed.Coordinate {
	return seed.Coordinate{
		Entity:  req.At.Entity,
		Row:     req.At.Row,
		Field:   req.At.Field,
		Purpose: req.At.Purpose + ":" + purpose,
	}
}
