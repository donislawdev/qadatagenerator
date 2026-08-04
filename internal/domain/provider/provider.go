// Package provider is the contract a source of country bound values has to
// meet.
//
// It is the contract and not the implementations. The interface lives in the
// domain because the corruption engine calls it, and the implementations live
// under the adapters because they carry reference data and locale knowledge.
// That is the dependency rule applied to the one place the project promises
// to accept contributions.
//
// # Three duties, and a fourth that is easy to miss
//
// A provider produces a value, says whether a value is correct, and declares
// what it brings. The fourth duty is the reason the third exists: a mutator
// asks for capabilities, and a provider that claims one has to be able to
// answer the questions that capability implies. Claiming to encode a date and
// being unable to say where the date sits would make a mutator that cannot
// work but reports that it can.
//
// So a declared capability is checked against an implemented interface, in
// both directions, by a test. Declaring identifier.check-digit without
// implementing CheckDigited fails the build, and so does the reverse. This is
// the same closing of a loop that the catalog does between its entries and
// their implementations.
//
// # Ethics is a property of the provider, not of the caller
//
// Reserved and never assigned ranges are the default inside the provider,
// which is what makes skipping them a deliberate act rather than a forgotten
// flag. A value that could belong to a real person is not something a caller
// should have to remember to avoid.
package provider

import (
	"sort"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Gender is what an identifier encodes, where it encodes one.
//
// It is deliberately a small closed set rather than free text. The values an
// identifier can carry are decided by the numbering scheme, not by the vocabulary
// of the recipe, and a provider has to be able to answer for all of them.
type Gender string

const (
	// Female and Male are the two values numbering schemes actually encode.
	Female Gender = "female"
	Male   Gender = "male"
	// Unknown is what a value decodes to when the scheme carries no gender
	// or the value is malformed. It is never a thing to generate.
	Unknown Gender = "unknown"
)

// Place is a settlement a postal code belongs to.
//
// The region is here rather than left out because a place name alone does not
// identify a place. The United States registry carries more than two hundred
// codes reading New York and a hundred reading Springfield across different
// states, so a pair built from the name alone would be true of some code and
// false of the one in hand. Naming the region is what lets the reverse direction
// mean anything.
//
// Region is the word the corpus already uses for the tier below a country, and
// it is deliberately not called state: it is a voivodeship in Poland and a state
// in the United States, and a provider contract that took sides would need a
// second name the day a third country arrived.
type Place struct {
	Name   string
	Region string
}

// Date is a calendar date without a clock, a zone or a format.
//
// The domain is not allowed to read the clock, and a date type carrying one
// is an invitation to do it. Formatting is the business of whoever writes the
// value out, because a date rendered by the settings of whichever machine ran
// the tool would break byte stability across platforms.
type Date struct {
	Year  int
	Month int
	Day   int
}

// Request is everything a provider needs to produce one value.
type Request struct {
	// Source and At are the address of the value. Randomness is derived from
	// them rather than drawn from a stream, so a provider must not hold a
	// counter and must not care in what order it is called.
	Source seed.Source
	At     seed.Coordinate
	// Agrees carries the values this one has to be consistent with, such as
	// the birth date and gender a national identifier encodes. It is empty
	// when the recipe declared no rule binding this field to another.
	Agrees Agreement
}

// The roles a bound value can play.
//
// An agreement is keyed by role and not by the name the user gave a column. A
// provider has to know that it was handed a date of birth. It has no business
// knowing whether the recipe called it birth_date, dob or data_urodzenia, and
// resolving one to the other is the work of the use case that reads the
// consistency rules.
const (
	RoleBirthDate = "birth_date"
	RoleGender    = "gender"
)

// Agreement is the values a generated value has to be consistent with, keyed
// by the role each one plays.
//
// It is a type rather than a bare map so that reading it cannot depend on map
// iteration order by accident, and so that a missing binding is answered
// rather than returned as an empty string that looks like a value.
type Agreement struct {
	values map[string]string
}

// NewAgreement records what a value has to agree with.
func NewAgreement(values map[string]string) Agreement {
	copied := make(map[string]string, len(values))
	for field, value := range values {
		copied[field] = value
	}
	return Agreement{values: copied}
}

// Value returns what another field holds, and whether it was bound at all.
func (a Agreement) Value(field string) (string, bool) {
	value, ok := a.values[field]
	return value, ok
}

// Fields returns the bound field names, sorted.
func (a Agreement) Fields() []string {
	fields := make([]string, 0, len(a.values))
	for field := range a.values {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

// Verdict is the answer to whether a value is correct for this provider.
//
// A bare boolean would be cheaper and worse. The reason is what lets a report
// say why a value was rejected, and a validator that only says no is one
// nobody can act on.
//
// # Three answers, because two of them were being spelled the same way
//
// There is a third thing a validator can mean, and until 2026-08-03 it had
// nowhere to live. A scheme can have a form this build recognises and does not
// check: the statistical number has a fourteen digit form for a local unit, and
// only the nine digit form is implemented. Answering that with Valid false put
// a correct value and a broken value in the same bucket, and the difference was
// carried only by prose in Reason, which nothing reads programmatically.
//
// That matters most in the mode where the column arrives from somebody else's
// file. Reporting a defect in a value that has none is the failure this project
// avoided carefully when it taught the tax number three ways of being written,
// and then reintroduced one file away.
//
// Valid stays false for the third answer, so nothing that already branches on
// it changes behaviour. What it gains is the ability to ask the other question.
type Verdict struct {
	// Valid says the value satisfies every rule this provider checks.
	Valid bool
	// Unsupported says the value is a form of this scheme that this build
	// recognises and does not check, so it is neither approved nor faulted.
	//
	// A caller deciding whether to report a defect has to ask this before
	// trusting Valid. A caller deciding whether it may corrupt the value has
	// to treat it as a refusal, because corrupting what we cannot check would
	// produce a manifest claiming something nobody verified.
	Unsupported bool
	// Reason is why the value is not valid, empty when it is. It names the
	// rule that failed, such as a wrong check digit or a length the scheme
	// does not allow.
	Reason string
}

// Valid is the verdict for a correct value.
func Valid() Verdict { return Verdict{Valid: true} }

// Invalid is the verdict for a value that breaks a rule of the scheme.
func Invalid(reason string) Verdict { return Verdict{Reason: reason} }

// Unsupported is the verdict for a value this build recognises as a form of
// the scheme and does not check. The reason has to name the form, so that a
// report can tell a user their data was passed over rather than approved.
func Unsupported(reason string) Verdict { return Verdict{Unsupported: true, Reason: reason} }

// Provider produces and checks the values of one identifier or one kind of
// locale bound value.
type Provider interface {
	// ID is the identifier of this provider, such as PL.pesel. Country bound
	// providers carry an ISO 3166-1 alpha-2 prefix and international
	// standards carry none, because a provider belongs to a country and not
	// to a language.
	ID() string
	// Capabilities are what this provider brings into a run. Every name
	// returned here must be supplied by a provider according to the
	// registry, and every capability that implies an interface must be
	// backed by one.
	Capabilities() []capability.Name
	// Generate produces one value for the place described by the request.
	// The same request must always produce the same value.
	Generate(req Request) (string, error)
	// Validate says whether a value is correct for this scheme. It is
	// written from the rules of the scheme and never from what Generate
	// happens to produce, because otherwise one mistake would be made twice
	// and agree with itself.
	Validate(value string) Verdict
}

// CheckDigited is required of a provider declaring identifier.check-digit.
//
// Being able to compute the correct check digit is what makes it possible to
// break one deliberately, which is the whole of layer L3 for identifiers.
type CheckDigited interface {
	// CheckDigits returns the positions carrying the check digits, as byte
	// offsets into the value, and whether the value is long enough to have
	// them.
	CheckDigits(value string) ([]int, bool)
	// WithCheckDigits returns the value with its check digits recomputed, so
	// that a mutator can change a digit and ask for a value that is
	// consistent again, or leave it inconsistent on purpose.
	WithCheckDigits(value string) (string, error)
}

// DateEncoder is required of a provider declaring identifier.encodes-date.
type DateEncoder interface {
	// EncodedDate reads the date out of the value.
	EncodedDate(value string) (Date, bool)
	// WithDate returns the value carrying a different date, check digits
	// included. This is what lets a mutator produce an identifier that is
	// well formed and disagrees with the rest of the record.
	WithDate(value string, date Date) (string, error)
}

// GenderEncoder is required of a provider declaring identifier.encodes-gender.
type GenderEncoder interface {
	EncodedGender(value string) (Gender, bool)
	WithGender(value string, gender Gender) (string, error)
}

// RangeAware is required of a provider declaring identifier.reserved-range.
//
// # Two questions, and running them together makes the mutator useless
//
// A numbering scheme with unassigned blocks usually has more than one, and
// they are not interchangeable. Where the scheme has a block that is never
// issued, generating inside it is the ethical default, so every value this
// tool produces for that scheme already sits in an unassigned block. A mutator
// aimed at unassigned ranges therefore cannot ask whether a value is in a
// reserved range and stop when the answer is yes: for such a scheme the answer
// is always yes and the mutator would never apply to anything.
//
// That is why the write side is here rather than in the engine. The applicable
// question is not whether the value is reserved but whether it already sits in
// the particular block the corruption is aimed at, and only the provider knows
// how many blocks the scheme has or how to write one. Putting that in the
// engine would make the second country with reserved ranges an edit to the
// engine instead of a new file.
//
// # Why this is two interfaces and not one
//
// The read and the write side answer questions that different schemes answer
// differently, and bundling them made one of them a lie.
//
// A harbour is a block a registrar says nothing will ever be issued from.
// Where a scheme has one, generating inside it is the ethical default, and the
// tool can say its own output could not belong to anybody. Some schemes have
// no harbour at all: their registrar names blocks it has merely not issued yet
// and says out loud it may issue them later. Generating there would be the
// mistake of picking a range because it looks free.
//
// Moving a value into an unassigned block is a different question with a
// different answer. It is safe for both kinds, because a value there cannot
// belong to anybody today, which is all a corruption needs.
//
// So a scheme without a harbour still supports the corruption, and a provider
// has to be able to say that. One interface would have forced it to claim a
// harbour it does not have, and a manifest reader would have believed it.
type RangeAware interface {
	// InReservedRange reports whether the value sits in a range that is not
	// assigned to a real subject. It is the claim this tool makes about its
	// own output, not a test of whether a corruption applies.
	InReservedRange(value string) bool
	// ReservedRanges describes the ranges in words, for the report that has
	// to tell a user why their safe data is rejected by an application that
	// checks ranges.
	ReservedRanges() []string
}

// UnassignedWritable is required of a provider declaring
// identifier.unassigned-target.
//
// # Why the write side is here rather than in the engine
//
// The applicable question is not whether a value is reserved but whether it
// already sits in the particular block the corruption is aimed at, and only
// the provider knows how many blocks the scheme has or how to write one.
// Putting that in the engine would make the second country with unassigned
// ranges an edit to the engine instead of a new file.
//
// Asking the other question instead would break the mutator outright. Where a
// scheme has a harbour, every value this tool generates already sits in an
// unassigned block, so a mutator that stopped when the value was reserved
// would never apply to anything.
type UnassignedWritable interface {
	// UnassignedTargets names the blocks a value can be moved into, sorted.
	// These are the names a catalog entry passes as its target parameter.
	UnassignedTargets() []string
	// WithUnassigned returns the value moved into the named block.
	//
	// It refuses when the value already sits there, and that refusal is the
	// applicability answer: it is the one case the manifest has to record as
	// applied false with a reason, rather than as a corruption that changed
	// nothing and said so to nobody.
	WithUnassigned(value, target string) (string, error)
}

// PostalLinked is required of a provider declaring reference.postal-to-place.
//
// # Both directions, because the capability promises both
//
// The registry answers which place a code belongs to and which codes a place
// holds. Pillar one needs the first to put a coherent pair in a record. A report
// explaining why a value was rejected needs the second, and so does anybody
// generating a record from the place inwards.
//
// # Why there is a write side
//
// The corruption aimed at this capability puts a real code next to a place it
// does not belong to, so it has to build a value rather than merely ask about
// one. Leaving that to the engine would put the registry of one country inside
// the engine, which is the mistake the unassigned range interface already made
// once and had corrected.
//
// The applicable question is therefore not whether the code has a place. It is
// whether a different real code exists whose place differs, which is the block
// the corruption aims at. Asking the first would be true of every value this
// tool generates and would never refuse anything.
type PostalLinked interface {
	// PlaceFor returns the place a code belongs to, and whether the registry
	// carries the code at all. A code it does not carry is answered rather
	// than guessed, because a guessed pair is the defect this tool sells.
	PlaceFor(code string) (Place, bool)
	// CodesFor returns the codes belonging to a place, sorted. It is empty for
	// a place the registry does not carry.
	CodesFor(place Place) []string
	// WithDifferentPlace returns a real code belonging to some other place.
	//
	// It refuses when the code is not in the registry, and that refusal is the
	// applicability answer the manifest has to record as applied false with a
	// reason. The same input always gives the same output: a corruption that
	// varied between two runs of one seed would break the promise the whole
	// project rests on.
	WithDifferentPlace(code string) (string, error)
}

// implied maps a capability to the interface a provider claiming it must
// implement.
//
// Only capabilities that imply extra questions appear here. Declaring one
// that is not listed is a claim about what the provider knows rather than
// about what it can be asked, and needs no second interface.
var implied = map[capability.Name]func(Provider) bool{
	capability.IdentifierCheckDigit: func(p Provider) bool {
		_, ok := p.(CheckDigited)
		return ok
	},
	capability.IdentifierEncodesDate: func(p Provider) bool {
		_, ok := p.(DateEncoder)
		return ok
	},
	capability.IdentifierEncodesGender: func(p Provider) bool {
		_, ok := p.(GenderEncoder)
		return ok
	},
	capability.IdentifierUnassignedTarget: func(p Provider) bool {
		_, ok := p.(UnassignedWritable)
		return ok
	},
	capability.IdentifierReservedRange: func(p Provider) bool {
		_, ok := p.(RangeAware)
		return ok
	},
	capability.ReferencePostalToPlace: func(p Provider) bool {
		_, ok := p.(PostalLinked)
		return ok
	},
}

// ImpliedInterfaces returns the capabilities that require an interface,
// sorted.
func ImpliedInterfaces() []capability.Name {
	names := make([]capability.Name, 0, len(implied))
	for name := range implied {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

// Check reports every way a provider fails the contract, sorted.
//
// It returns findings rather than a boolean so that a contributor adding a
// provider for their country sees all of them at once instead of fixing them
// one build at a time. That is the same rule the recipe validator follows.
func Check(p Provider) []string {
	var findings []string

	if p.ID() == "" {
		findings = append(findings, "provider has no identifier")
	}

	seen := make(map[capability.Name]bool)
	for _, name := range p.Capabilities() {
		if seen[name] {
			findings = append(findings, string(name)+": declared twice")
			continue
		}
		seen[name] = true

		supplier, known := capability.SupplierOf(name)
		if !known {
			findings = append(findings, string(name)+": not a capability in the registry")
			continue
		}
		if supplier != capability.Provider {
			findings = append(findings, string(name)+": supplied by "+string(supplier)+", not by a provider")
			continue
		}
		if implements, ok := implied[name]; ok && !implements(p) {
			findings = append(findings, string(name)+": declared but the interface it implies is not implemented")
		}
	}

	// The other direction. An interface implemented without the matching
	// declaration is the quieter defect of the two, because everything works
	// except that no mutator ever asks.
	for name, implements := range implied {
		if implements(p) && !seen[name] {
			findings = append(findings, string(name)+": implemented but not declared")
		}
	}

	sort.Strings(findings)
	return findings
}
