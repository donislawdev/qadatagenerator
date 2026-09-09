// Package us carries the providers of United States values and identifiers.
//
// They live under the adapters rather than in the domain because they hold
// knowledge of one country, which is exactly the thing this project expects to
// receive as contributions. The contract they satisfy is in the domain, where
// the corruption engine can see it.
package us

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// SSN is the United States Social Security number.
//
// # Where the rules came from
//
// Read at the Social Security Administration rather than recalled. The two
// sentences the whole of this file rests on are quoted in the constants below,
// because a rule restated in one's own words is a rule that can drift, and
// this project has already been bitten once by an identifier whose algorithm
// was written from memory and turned out not to exist.
//
// The number is nine digits in three parts, written 123-45-6789: a three digit
// area number, a two digit group number and a four digit serial number.
//
// # There is no check digit, and that is a fact about the scheme
//
// Nothing in an SSN verifies anything else in it. A wrong digit produces
// another well formed SSN, so the mutator that breaks a check digit has
// nothing to do here and this provider does not claim identifier.check-digit.
// Nor does it encode a date or a gender: since randomization the area number
// carries no geography and no chronology either.
//
// What the scheme does have is blocks of numbers that are never issued, and
// that single capability is what this provider brings.
//
// # The two tiers, which the source keeps apart and so does this file
//
// The wording quoted below draws a line that matters more than it looks.
// Group 00 and serial 0000 are called invalid outright. The area numbers are
// only excluded from assignment, which is a weaker statement: a number with
// area 666 looks entirely ordinary and is refused only by a validator holding
// the assignment rules.
//
// That line is the whole reason the corruption works. Generated data sits in
// area 666, which is never issued and therefore safe, and yet passes any
// ordinary format check. The corruption moves the value into a block that
// looks broken to anybody. Two never issued blocks, two different lines of
// defence, and Validate answers for the first while InReservedRange answers
// for the second.
//
// # Ethics
//
// Every value produced here sits in area 666, which the Administration has
// never assigned and has excluded from assignment, so no generated number can
// be a real person's. That is a property of this file rather than a flag the
// caller has to remember. It has a price, written down rather than hidden: an
// application that checks assignment ranges will reject this tool's correct
// data as well as its corrupted data.
type SSN struct{}

// Contract checks, so that a mismatch between what this provider claims and
// what it implements fails to compile rather than at run time.
var (
	_ provider.Provider           = SSN{}
	_ provider.RangeAware         = SSN{}
	_ provider.UnassignedWritable = SSN{}
)

// The two sentences this file rests on, read on the Administration's own page
// about number randomization:
//
//	Randomization also introduced previously unassigned area numbers for
//	assignment excluding area numbers 000, 666 and 900-999.
//
//	SSN randomization will not assign group number 00 or serial number 0000.
//	SSNs containing group number 00 or serial number 0000 will continue to be
//	invalid.
//
// They are kept here as prose and taken apart as numbers below. The validator
// written to check this one holds them the other way round, as the strings
// themselves, so that a sentence read wrongly once is not read wrongly twice.
//
// The pages refuse an automated fetch with a 403 and open normally in a
// browser, which is worth knowing before somebody records them as unreachable.

// The layout. Every rule of the scheme is about one of the three parts, so
// they are kept apart rather than as one nine digit string.
const (
	areaDigits   = 3
	groupDigits  = 2
	serialDigits = 4
	// bareLength is the number written without separators, writtenLength with
	// them. Both are the same number and both are accepted.
	bareLength    = areaDigits + groupDigits + serialDigits
	writtenLength = bareLength + 2
)

// generatedArea is the block pillar one occupies.
//
// Of the three never issued areas it is the only usable one. Area 000 looks
// broken, so correct data built on it would fail an ordinary format check and
// the tool would be blamed for it. Areas 900 to 999 are the space of the
// individual taxpayer identification number, so a number generated there has
// the shape of a real identifier belonging to a real person, which would break
// the ethics rule rather than satisfy it.
const generatedArea = 666

// The blocks a corruption can move a value into.
//
// TargetDefault is the name the catalog entry carries as the default of its
// target parameter. It means the zero area: the Administration's own guidance
// on showing a number that must not be anybody's points at 000, which makes it
// the block most reliably refused by somebody else's validator.
const (
	TargetDefault    = "structurally_never_issued"
	TargetZeroArea   = "zero_area"
	TargetZeroGroup  = "zero_group"
	TargetZeroSerial = "zero_serial"
)

// ID names this provider. The country prefix is ISO 3166-1 alpha-2, because a
// provider belongs to a country rather than to a language.
func (SSN) ID() string { return "US.ssn" }

// Capabilities are what this provider brings.
//
// One, and the absences are as deliberate as the presence. The scheme carries
// no check digit, no date and no gender, so claiming any of those would be a
// promise it cannot keep and would hand a mutator a capability that turns out
// to be empty while it is corrupting.
// Both range capabilities are declared, and they are two claims rather than
// one. The Administration names areas nothing is ever issued from, so this
// scheme has a harbour and pillar one generates inside it. It also knows the
// blocks a corruption can move a value into. A scheme with no harbour would
// declare only the second.
// Mask is nine digits in three groups, punctuated, which is how the
// administration writes it and how this provider generates it.
//
// The hyphens are in the mask because they are in every value: Generate sets
// hyphenated, so a mask without them would be describing the bare form, which
// Validate accepts and nothing here produces.
func (SSN) Mask() string {
	group := func(n int) string { return strings.Repeat(string(provider.MaskDigit), n) }
	return group(areaDigits) + "-" + group(groupDigits) + "-" + group(serialDigits)
}

func (SSN) Capabilities() []capability.Name {
	return []capability.Name{
		capability.IdentifierReservedRange,
		capability.IdentifierUnassignedTarget,
	}
}

// Generate produces one number for the place described by the request.
//
// The agreement is not read, and that is a property of the scheme rather than
// an omission: an SSN encodes nothing about its holder, so there is nothing in
// it that could be made to agree with a date of birth or a gender. A recipe
// binding one to this field is asking for something the scheme cannot express,
// and answering that belongs to the use case that reads the consistency rules,
// which can see both sides. This file can only decline to pretend.
func (SSN) Generate(req provider.Request) (string, error) {
	draw := func(purpose string, bound int64) int {
		return int(req.Source.IntN(seed.Coordinate{
			Entity:  req.At.Entity,
			Row:     req.At.Row,
			Field:   req.At.Field,
			Purpose: req.At.Purpose + purpose,
		}, bound))
	}
	// Both parts start at one rather than nought, since a zero group and a
	// zero serial are the two the source calls invalid. That leaves ninety
	// nine groups and nine thousand nine hundred and ninety nine serials,
	// which is 989 901 distinct numbers in this area.
	return number{
		area:       generatedArea,
		group:      1 + draw(":ssn-group", 99),
		serial:     1 + draw(":ssn-serial", 9999),
		hyphenated: true,
	}.String(), nil
}

// Validate answers whether a value is a well formed SSN.
//
// It is the first of the two lines of defence, and it is deliberately the one
// an ordinary application also has: the shape, plus the parts that make a
// number structurally void. It does not refuse area 666 or the 900 block,
// because those look like perfectly ordinary numbers and refusing them here
// would mean this tool's own correct output failed its own validator.
//
// Whether the Administration would ever have issued the number is the other
// question, and InReservedRange answers it.
func (SSN) Validate(value string) provider.Verdict {
	parsed, ok := parse(value)
	if !ok {
		return provider.Invalid("an SSN is nine digits, written 123456789 or 123-45-6789")
	}
	switch {
	case parsed.area == 0:
		return provider.Invalid("the area number is 000, which is never issued")
	case parsed.group == 0:
		return provider.Invalid("the group number is 00, which the scheme calls invalid")
	case parsed.serial == 0:
		return provider.Invalid("the serial number is 0000, which the scheme calls invalid")
	}
	return provider.Valid()
}

// InReservedRange reports whether the value sits in a block the Administration
// never assigns.
//
// It is true of everything this provider generates, on purpose, and that is
// why it is not the question a mutator asks about applicability. See the
// contract in the domain for why those are two different questions.
func (SSN) InReservedRange(value string) bool {
	parsed, ok := parse(value)
	if !ok {
		return false
	}
	// Areas from the first quoted sentence, parts from the second.
	return parsed.area == 0 || parsed.area == generatedArea || parsed.area >= 900 ||
		parsed.group == 0 || parsed.serial == 0
}

// ReservedRanges describes the blocks in words.
//
// The report needs these because of a consequence users would otherwise meet
// as a defect: an application validating assignment ranges rejects this tool's
// correct data too. A limitation the interface does not say out loud is one
// the user discovers as a bug in the tool.
func (SSN) ReservedRanges() []string {
	return []string{
		"area number 000, never assigned and looks broken to any validator",
		"area number 666, never assigned and looks entirely ordinary",
		"area numbers 900 to 999, never assigned and the shape of a taxpayer identification number",
		"group number 00, which the scheme calls invalid",
		"serial number 0000, which the scheme calls invalid",
	}
}

// UnassignedTargets names the blocks a corruption can move a value into.
//
// Four names and three blocks. TargetDefault is the name the catalog entry
// carries and it means the same block as TargetZeroArea, which is said here
// rather than left to be discovered by whoever compares the two lists. The
// alias is kept on the list because the list is also what the refusal prints
// when an unknown target arrives, and a name the catalog uses has to appear
// among the choices offered.
func (SSN) UnassignedTargets() []string {
	return []string{TargetDefault, TargetZeroArea, TargetZeroGroup, TargetZeroSerial}
}

// WithUnassigned returns the value moved into the named block.
//
// The refusal when nothing would change is the point of the method rather than
// a guard on it. Pillar one already generates inside a never issued block, so
// a corruption aimed at the same block would produce the value it was given
// and report success, which is the quiet kind of nothing this tool exists to
// argue against.
func (SSN) WithUnassigned(value, target string) (string, error) {
	parsed, ok := parse(value)
	if !ok {
		return "", fmt.Errorf("ssn: %s is not nine digits", provider.Show(value))
	}

	moved := parsed
	switch target {
	case TargetDefault, TargetZeroArea:
		moved.area = 0
	case TargetZeroGroup:
		moved.group = 0
	case TargetZeroSerial:
		moved.serial = 0
	default:
		return "", fmt.Errorf("ssn: %s is not a block this scheme has, the choices are %s",
			provider.Show(target), strings.Join(SSN{}.UnassignedTargets(), ", "))
	}

	if moved == parsed {
		return "", fmt.Errorf("ssn: %s already sits in %s, so moving it there would change nothing",
			value, target)
	}
	return moved.String(), nil
}

// --- the number, kept as its three parts --------------------------------------

// number is a parsed SSN.
//
// It keeps whether the value was written with separators so that a corruption
// hands back the shape it was given. Reformatting a value on the way through
// would be a second change nobody asked for, and it would show up in a diff as
// though the corruption had done more than it did.
type number struct {
	area, group, serial int
	hyphenated          bool
}

// parse reads either way of writing the number, and refuses everything else.
//
// Both spellings are the same number, and a validator refusing the one without
// separators would report a defect where there is none. Anything that is
// neither shape is refused rather than repaired, because guessing what a value
// was meant to be is how a tool quietly changes somebody's data.
func parse(value string) (number, bool) {
	var bare string
	hyphenated := false

	switch {
	case len(value) == bareLength:
		bare = value
	case len(value) == writtenLength &&
		value[areaDigits] == '-' && value[areaDigits+1+groupDigits] == '-':
		bare = value[:areaDigits] +
			value[areaDigits+1:areaDigits+1+groupDigits] +
			value[areaDigits+2+groupDigits:]
		hyphenated = true
	default:
		return number{}, false
	}

	digits, ok := digitsOf(bare)
	if !ok {
		return number{}, false
	}
	return number{
		area:       digits[0]*100 + digits[1]*10 + digits[2],
		group:      digits[3]*10 + digits[4],
		serial:     digits[5]*1000 + digits[6]*100 + digits[7]*10 + digits[8],
		hyphenated: hyphenated,
	}, true
}

// String writes the number back in the shape it arrived in.
func (n number) String() string {
	var built strings.Builder
	built.Grow(writtenLength)

	writeDigits(&built, n.area, areaDigits)
	if n.hyphenated {
		built.WriteByte('-')
	}
	writeDigits(&built, n.group, groupDigits)
	if n.hyphenated {
		built.WriteByte('-')
	}
	writeDigits(&built, n.serial, serialDigits)
	return built.String()
}

// writeDigits writes a value zero padded to a fixed width.
//
// It refuses a value too large for the width instead of writing its last few
// digits. Silent truncation would turn a group of one hundred into the string
// 00, which is one of the two the scheme calls invalid, so the tool would
// produce a broken number while every line of it read correctly. That can only
// happen through a defect in this package and never through user input, which
// is why it fails loudly here rather than returning an error nobody could act
// on.
func writeDigits(built *strings.Builder, value, width int) {
	if value < 0 || value >= pow10(width) {
		panic(fmt.Sprintf("ssn: %d does not fit in %d digits", value, width))
	}
	for place := pow10(width - 1); place > 0; place /= 10 {
		built.WriteByte(byte('0' + value/place%10))
	}
}

func pow10(exponent int) int {
	result := 1
	for i := 0; i < exponent; i++ {
		result *= 10
	}
	return result
}

func digitsOf(value string) ([]int, bool) {
	digits := make([]int, 0, len(value))
	for _, symbol := range value {
		if symbol < '0' || symbol > '9' {
			return nil, false
		}
		digits = append(digits, int(symbol-'0'))
	}
	return digits, true
}
