package us

import (
	"fmt"
	"sort"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// ABA is the routing number of a United States financial institution.
//
// # Where the rules came from, and the two halves have different answers
//
// The shape was read at the registrar. ABA Routing Number Policy & Procedures
// version 9/20, section IV, published by LexisNexis Risk Solutions as the
// official routing number registrar, gives nine digits in three parts: a four
// digit Federal Reserve Routing Symbol, a four digit ABA Institution
// Identifier and a check digit. The same section allocates the leading two
// digit series, and that table is quoted in the independent validator.
//
// The arithmetic was not read anywhere. The registrar sends the reader to a
// publication it sells, the Federal Reserve directory FAQ is silent on the
// check digit, and the rule is most likely inside the paid ANSI X9 standards.
// So the weights here rest on a measurement, and this file says so rather than
// letting a reader assume otherwise.
//
// Measured against 12 CFR part 229 appendix A, a federal regulation that prints
// complete routing numbers of the Federal Reserve Banks and the Federal Home
// Loan Banks: 67 of 67 close under the weights 3 7 1 repeated. Walking all one
// thousand weight vectors over that sample leaves exactly ten that close it,
// the zero vector and the nine multiples of this one modulo ten, which are the
// same congruence. There is no other solution in the space, so the measurement
// forces these weights rather than merely agreeing with them.
//
// What that does not establish is that the registrar writes the rule this way
// in the publication it sells. The numbers are real and were promulgated in
// federal law, so the rule they satisfy is the rule, but "the weights come from
// the source" would be untrue and is not written anywhere.
//
// # Ethics, and this scheme is the hard case
//
// There is no harbour. The registrar marks several series "Reserved for future
// use" and adds, in its own words, that their use "will be determined by the
// Board". Spare stock is not a safe range: generating there would be picking a
// block because it looks free, which is the mistake this project already made
// once when a Social Security range that looked obviously spare turned out to
// be the space of a real taxpayer identifier.
//
// So correct values are generated inside the series the policy assigns to
// banks, every one of them could belong to a real institution, and the defence
// is the same as for the Polish identifiers: the dataset carrying these has to
// be marked synthetic in its own metadata, which is the duty of whatever writes
// the output.
//
// This is the reverse of US.ssn and the contrast is worth holding. There the
// base value already sits in a never issued area, so a corruption aimed at
// unassigned ranges had to be asked a narrower question. Here the base value
// never sits in one, so the same corruption always applies.
type ABA struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider           = ABA{}
	_ provider.CheckDigited       = ABA{}
	_ provider.UnassignedWritable = ABA{}
)

const (
	// abaLength is the number of digits and abaCheckPosition is the index of
	// the check digit. abaBodyLength is what it is computed over.
	abaLength        = 9
	abaCheckPosition = 8
	abaBodyLength    = abaCheckPosition
	// abaGroupedLength is the length of the four four one form that federal
	// law prints these numbers in.
	abaGroupedLength = abaLength + 2
	// abaSeriesDigits is how many leading digits carry the series.
	abaSeriesDigits = 2
)

// abaBodyWeights are applied to the eight digits the check digit is computed
// from. See the type comment for where they come from.
//
// # Why eight and not nine
//
// The rule is a property of all nine digits: the weighted total of the whole
// number is divisible by ten, and the ninth weight is one. That is how the
// independent validator states it, and stating it there is what makes the two
// halves worth comparing.
//
// Here only eight are needed, because a weight of one on the last digit makes
// the completing digit the plain complement of the total over the rest. Holding
// nine here would leave the ninth unused, and an unused entry in a table of
// constants is one nobody can break: a mutation changing it would pass every
// test while looking like a change to the arithmetic. Measured on 2026-08-03,
// which is how this was found.
var abaBodyWeights = [abaBodyLength]int{3, 7, 1, 3, 7, 1, 3, 7}

// generatedSeries is the span pillar one generates inside.
//
// It is the series the policy assigns to the checks and electronic
// transactions of banks, and it is chosen because this scheme offers nothing
// safer. Every value here could belong to a real institution, which is stated
// in the type comment rather than hidden.
const (
	generatedSeriesLow  = 1
	generatedSeriesHigh = 12
)

// The blocks a corruption can move a value into.
//
// Every one of them is a series the registrar marks "Reserved for future use",
// so nothing is issued there today and a value moved into one cannot belong to
// anybody. That is all a corruption needs, and it is a weaker claim than the
// one US.ssn can make about its own blocks.
//
// TargetABADefault carries the name the catalog entry passes as the default of
// its target parameter. The name says never issued and this scheme has no such
// block, so for this provider it means the lowest spare series instead.
//
// 🔴 That mismatch is deliberate and recorded rather than papered over. The
// parameter is one name shared by every provider the mutator applies to, and
// making it per provider is a change to the catalog, which is a public contract
// frozen at the first release. It waits in the backlog with the other
// identifier names that need a pass before the repository becomes public.
const (
	TargetABADefault  = "structurally_never_issued"
	TargetSpareLow    = "spare_series_low"
	TargetSpareMiddle = "spare_series_middle"
	TargetSpareHigh   = "spare_series_high"
)

// abaSpareSeries maps a target name onto the first series of a spare span.
//
// The three spans are far apart on purpose, so that a corruption cannot be
// mistaken for a neighbouring assignment. The low one is the first series the
// policy marks spare and carries no history at all, which is why it is the
// default: series 90 and 91 once held non par items, and a block with a past is
// a worse place to put a value than a block without one.
var abaSpareSeries = map[string]int{
	TargetABADefault:  13,
	TargetSpareLow:    13,
	TargetSpareMiddle: 33,
	TargetSpareHigh:   73,
}

// ID names this provider. The country prefix is ISO 3166-1 alpha-2, because a
// provider belongs to a country rather than to a language.
func (ABA) ID() string { return "US.aba" }

// Capabilities are what this provider brings.
//
// Two, and the absence of the third is the whole ethics of this scheme.
// identifier.reserved-range is deliberately not declared: it means the scheme
// has a harbour, a block nothing is ever issued from, that pillar one generates
// inside by default. This scheme has none, and the absence of the declaration
// is the machine readable way of saying so. A report reading the registry
// learns from it that a dataset carrying these numbers must be marked
// synthetic.
//
// identifier.unassigned-target is declared, because moving a value into a
// spare series is a different question with a different answer, and one this
// scheme can answer.
func (ABA) Capabilities() []capability.Name {
	return []capability.Name{
		capability.IdentifierCheckDigit,
		capability.IdentifierUnassignedTarget,
	}
}

// Generate produces one routing number for the place described by the request.
//
// The agreement is not read, and that is a property of the scheme rather than
// an omission: a routing number identifies an institution and encodes nothing
// about a person, so there is nothing in it that could be made to agree with a
// date of birth or a gender.
//
// The check digit always exists, unlike the Polish tax number, because the rule
// is modulo ten. Every body completes, so there is no digit to choose and no
// refusal to make.
func (a ABA) Generate(req provider.Request) (string, error) {
	draw := func(purpose string, bound int64) int {
		return int(req.Source.IntN(seed.Coordinate{
			Entity:  req.At.Entity,
			Row:     req.At.Row,
			Field:   req.At.Field,
			Purpose: req.At.Purpose + purpose,
		}, bound))
	}

	series := generatedSeriesLow + draw(":aba-series", generatedSeriesHigh-generatedSeriesLow+1)
	rest := draw(":aba-body", 1000000)

	digits := appendFixedDigits(make([]int, 0, abaLength), series, abaSeriesDigits)
	digits = appendFixedDigits(digits, rest, abaBodyLength-abaSeriesDigits)
	return digitsToText(append(digits, abaCheckDigit(digits))), nil
}

// Validate answers whether a value is a correct routing number.
//
// It is written from the rules of the scheme rather than from what Generate
// happens to produce. It answers about shape and check digit only, and
// deliberately not about whether the series is one the registrar assigns: a
// number in a spare series is well formed, and refusing it here would mean this
// tool's own corrupted output failed for the wrong reason.
func (ABA) Validate(value string) provider.Verdict {
	parsed, ok := parseABA(value)
	if !ok {
		return provider.Invalid("an ABA routing number is nine digits, written 123456789 or 1234-5678-9")
	}
	if want, got := abaCheckDigit(parsed.digits[:abaCheckPosition]), parsed.digits[abaCheckPosition]; got != want {
		return provider.Invalid(fmt.Sprintf("the check digit is %d and should be %d", got, want))
	}
	return provider.Valid()
}

// CheckDigits says where the check digit sits.
//
// It is the last character in both accepted forms, so the offset follows the
// shape the value was written in rather than a fixed position that would be
// wrong for one of them.
func (ABA) CheckDigits(value string) ([]int, bool) {
	if _, ok := parseABA(value); !ok {
		return nil, false
	}
	return []int{len(value) - 1}, true
}

// WithCheckDigits returns the value with its check digit recomputed, in the
// shape it arrived in.
//
// It never refuses for arithmetic reasons. The rule is modulo ten, so every
// body has a completion, which is the difference from the Polish tax number
// and the reason there is no second result here.
func (ABA) WithCheckDigits(value string) (string, error) {
	parsed, ok := parseABA(value)
	if !ok {
		return "", fmt.Errorf("aba: %q is not nine digits", value)
	}
	parsed.digits[abaCheckPosition] = abaCheckDigit(parsed.digits[:abaCheckPosition])
	return parsed.String(), nil
}

// UnassignedTargets names the spare series a corruption can move a value into,
// sorted.
func (ABA) UnassignedTargets() []string {
	targets := make([]string, 0, len(abaSpareSeries))
	for name := range abaSpareSeries {
		targets = append(targets, name)
	}
	sort.Strings(targets)
	return targets
}

// WithUnassigned returns the value moved into the named spare series, with the
// check digit recomputed so that the result is well formed and still refused by
// anybody holding the allocation table.
//
// It refuses when the value already sits there, and that refusal is the
// applicability answer rather than a guard. Unlike US.ssn this can only happen
// to a value somebody else supplied, because pillar one never generates inside
// a spare series.
func (ABA) WithUnassigned(value, target string) (string, error) {
	parsed, ok := parseABA(value)
	if !ok {
		return "", fmt.Errorf("aba: %q is not nine digits", value)
	}
	series, known := abaSpareSeries[target]
	if !known {
		return "", fmt.Errorf("aba: %q is not a block this scheme has, the choices are %s",
			target, strings.Join(ABA{}.UnassignedTargets(), ", "))
	}

	moved := parsed
	moved.digits[0] = series / 10
	moved.digits[1] = series % 10
	moved.digits[abaCheckPosition] = abaCheckDigit(moved.digits[:abaCheckPosition])
	if moved == parsed {
		return "", fmt.Errorf("aba: %s already sits in %s, so moving it there would change nothing",
			value, target)
	}
	return moved.String(), nil
}

// --- the arithmetic, kept in one place ---------------------------------------

// abaCheckDigit computes the ninth digit over the first eight.
//
// The weight of the check digit itself is one, so the digit that makes the
// weighted total of the whole number divisible by ten is the complement of the
// total over the first eight. The independent validator states the same rule
// the other way round, as a property of all nine digits, which is what makes
// the two worth comparing.
func abaCheckDigit(body []int) int {
	sum := 0
	for position, digit := range body {
		sum += digit * abaBodyWeights[position]
	}
	return (10 - sum%10) % 10
}

// --- the number, kept together with the shape it arrived in --------------------

// abaValue is a parsed routing number.
//
// It keeps whether the value was grouped so that a corruption hands back the
// shape it was given. Reformatting on the way through would be a second change
// nobody asked for.
type abaValue struct {
	digits  [abaLength]int
	grouped bool
}

// String writes the number out in the shape it arrived in.
func (v abaValue) String() string {
	if !v.grouped {
		return digitsToText(v.digits[:])
	}
	var built strings.Builder
	built.Grow(abaGroupedLength)
	for index, digit := range v.digits {
		if index == 4 || index == 8 {
			built.WriteByte('-')
		}
		built.WriteByte(byte('0' + digit))
	}
	return built.String()
}

// parseABA reads either accepted way of writing the number and refuses
// everything else.
//
// The grouped form is the one 12 CFR part 229 appendix A prints, which is where
// the measurement behind the weights came from, so a value copied out of the
// regulation has to be readable.
func parseABA(value string) (abaValue, bool) {
	bare := value
	grouped := false
	if len(value) == abaGroupedLength && value[4] == '-' && value[9] == '-' {
		bare = value[0:4] + value[5:9] + value[10:11]
		grouped = true
	}
	digits, ok := digitsOf(bare)
	if !ok || len(digits) != abaLength {
		return abaValue{}, false
	}
	parsed := abaValue{grouped: grouped}
	copy(parsed.digits[:], digits)
	return parsed, true
}

// appendFixedDigits writes a value out as a fixed number of digits, most
// significant first.
func appendFixedDigits(digits []int, value, width int) []int {
	divisor := 1
	for remaining := 1; remaining < width; remaining++ {
		divisor *= 10
	}
	for ; divisor > 0; divisor /= 10 {
		digits = append(digits, value/divisor%10)
	}
	return digits
}

func digitsToText(digits []int) string {
	var built strings.Builder
	built.Grow(len(digits))
	for _, digit := range digits {
		built.WriteByte(byte('0' + digit))
	}
	return built.String()
}
