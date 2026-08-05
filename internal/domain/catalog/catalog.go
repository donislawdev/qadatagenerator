// Package catalog is the model of the corruption catalog: what damage this
// tool knows how to do, and what a test should expect once it is done.
//
// # Two halves that meet at an identifier
//
// The metadata of a mutator is data. It is published in
// spec/corruption-catalog.yaml, which is the canonical form, and it travels
// into the binary from there. The behaviour is code, registered under the same
// identifier. Nothing joins the two except that identifier, which is what lets
// the catalog grow without the engine being touched. The catalog grows cheaply
// and the engine does not, so that is where the seam belongs.
//
// This package holds the metadata half and the closure between the halves. It
// knows nothing about YAML, because the adapter that reads the file is the only
// place allowed to know that, and nothing about how damage is applied.
//
// # Refusing the whole catalog rather than one entry
//
// A malformed entry stops the catalog from loading at all. Dropping the entry
// and carrying on would leave a mutator that quietly does not exist: a policy
// could order it and the manifest would say neither that it was applied nor
// that it was skipped, which is the exact silence this product exists to find
// in other people's tools. The catalog is also our own data inside our own
// binary, so the only way a bad entry gets here is our mistake, and a loud
// failure is the cheap answer to that.
package catalog

import (
	"sort"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
)

// ID is the stable identifier of one mutator, written LAYER.area.variant, for
// example L3.checksum.off_by_one.
//
// Identifiers reach manifests that live for years in other people's test
// repositories, so they are permanent once published. Retiring a mutator means
// marking it deprecated, never deleting it and never moving it to another
// layer.
type ID string

// Layer says which line of defence a mutator attacks.
type Layer string

// The layers and the cross cutting modules. The layers are ordered from the
// bytes of the file up to the integrity of the whole set, and the three modules
// cut across all of them.
const (
	Bytes      Layer = "L0"
	Container  Layer = "L1"
	FieldShape Layer = "L2"
	Domain     Layer = "L3"
	Business   Layer = "L4"
	SetLevel   Layer = "L5"

	Security Layer = "SEC"
	Unicode  Layer = "UNI"
	Reverse  Layer = "REV"
)

// Expectation is what the application under test should do with the damage.
type Expectation string

const (
	// Rejection means the application should refuse this input.
	Rejection Expectation = "rejection"
	// Acceptance means it should accept the value and keep it. An
	// application that stores something other than what it was given has
	// not accepted the value, it has accepted a different one.
	Acceptance Expectation = "acceptance"
	// Warning means it should accept the value and flag it.
	Warning Expectation = "warning"
)

// Confidence says how much the expectation is worth.
type Confidence string

const (
	// Hard means a specification, an algorithm or a type is violated, so no
	// reaction is a defect in the application.
	Hard Confidence = "hard"
	// Soft means the right reaction depends on the business rules of the
	// system under test, so no reaction is not necessarily a defect.
	Soft Confidence = "soft"
)

// Reserved values of the three applies_to dimensions.
const (
	// Any means the mutator is meaningful for every value of a dimension.
	Any = "any"
	// None means the dimension does not apply at all, which is the normal
	// answer for a mutator that damages the file rather than a field.
	None = "none"
)

// AppliesTo says where a mutator is meaningful. All three dimensions are
// always present, each carrying either real values or one of the two reserved
// words.
type AppliesTo struct {
	FieldTypes []string
	Formats    []string
	Providers  []string
}

// Parameter is one knob of a mutator, with the value used when a policy says
// nothing about it.
type Parameter struct {
	Name        string
	Description string

	// Default is deliberately untyped, because the catalog carries several
	// kinds of scalar. Measured across the starter set on 2026-08-05:
	// twenty three strings, eight whole numbers, five fractions, one
	// boolean and one null.
	//
	// The null is a value rather than a gap. L1.json.duplicate_key writes it
	// as the second value under a repeated key, so a model that folded it
	// into "no default given" would change what that mutator does without
	// saying so. An entry whose parameter has no default at all is refused
	// when the catalog is read, which is what makes nil here mean the file
	// said null and nothing else.
	//
	// Rendering one of these into a manifest is not settled and belongs to
	// the engine. A fraction printed by two different routes is exactly the
	// sort of thing that breaks byte stability between platforms, so
	// whoever writes it down owns that decision rather than inheriting it
	// from here.
	Default any
}

// Example is the before and after pair carried by every entry.
//
// The pair is a test case rather than an illustration. A published example that
// disagrees with what the engine does is a specification drifting away from the
// behaviour, and turning the pair into a test is what stops that happening
// quietly.
type Example struct {
	Before string
	After  string
	// Note is optional prose on why the entry is worth running, or on what
	// makes the default parameter the valuable variant.
	Note string
}

// Entry is one mutator as the catalog describes it.
type Entry struct {
	ID          ID
	Layer       Layer
	Summary     string
	AppliesTo   AppliesTo
	Parameters  []Parameter
	Expectation Expectation
	Confidence  Confidence
	// Reversible says whether the original value can be reconstructed from
	// the manifest.
	Reversible bool
	// Requires names what the valid data generator has to be able to produce
	// before this mutator can run at all.
	//
	// The names are the capability registry's own type rather than plain
	// strings, so a name that exists in no registry cannot travel further
	// than the reader that built this entry.
	Requires []capability.Name
	Example  Example

	// ApplicabilityCondition is present when the mutator only makes sense
	// under a stated condition. A policy asking for it outside that
	// condition produces a manifest entry with applied set to false and this
	// condition as the reason.
	ApplicabilityCondition string
	// Limitation is a consequence the user has to be told about, including
	// one that reaches beyond the corrupted rows.
	Limitation string
}

// Catalog is the whole set of entries this build carries.
//
// The zero value is empty and usable for reading, which keeps a caller that
// never loaded one from having to handle a nil.
type Catalog struct {
	version string
	status  string
	entries map[ID]Entry
	ordered []ID
}

// Version is the catalog version, which is one of the four versions the tool
// carries and the one that reaches the manifest header.
func (c Catalog) Version() string { return c.version }

// Status is what the catalog says about its own maturity, for example draft.
func (c Catalog) Status() string { return c.status }

// Len is how many mutators this build describes.
func (c Catalog) Len() int { return len(c.entries) }

// Lookup returns one entry and whether the catalog has it.
//
// The second result matters. An unknown identifier is a defect in whatever
// asked, and answering with a zero entry would turn it into a mutator with no
// layer, no expectation and no requirements, which is a shape that would travel
// a long way before anybody noticed.
func (c Catalog) Lookup(id ID) (Entry, bool) {
	entry, ok := c.entries[id]
	return entry, ok
}

// IDs returns every identifier the catalog carries, sorted.
//
// Sorted because Go randomises map iteration on purpose and these identifiers
// reach the manifest and the console. An order that differed between two runs
// of one seed would break the promise this whole project is built on.
func (c Catalog) IDs() []ID {
	ids := make([]ID, len(c.ordered))
	copy(ids, c.ordered)
	return ids
}

// Entries returns every entry, in identifier order.
func (c Catalog) Entries() []Entry {
	entries := make([]Entry, 0, len(c.ordered))
	for _, id := range c.ordered {
		entries = append(entries, c.entries[id])
	}
	return entries
}

// InLayer returns the entries of one layer, in identifier order.
func (c Catalog) InLayer(layer Layer) []Entry {
	var entries []Entry
	for _, id := range c.ordered {
		if entry := c.entries[id]; entry.Layer == layer {
			entries = append(entries, entry)
		}
	}
	return entries
}

func sortIDs(ids []ID) {
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
}
