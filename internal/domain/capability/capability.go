// Package capability names what a run has to be able to do before a mutator
// can run at all.
//
// Every catalog entry declares a requires list, and this package is the other
// half of that declaration: it says which capabilities exist and who brings
// each one into a run.
//
// # Five suppliers, not one
//
// The architecture once said that a provider declares the capabilities and
// that the availability of a mutator follows from them. Both halves were
// wrong, and the count is what showed it. Of the capabilities in the registry
// a provider supplies eight. The rest come from the format writer, the schema
// view, the corpus and the engine itself. Reading the field the old way would
// have left ten mutators permanently unapplied, all of layer L0 and L1, since
// no provider brings output.charset-control or format.csv.
//
// That mistake had already appeared twice, once for a mutator whose
// applicability condition could never hold and once for one that had nothing
// to do in either supported locale. The third time it is the mechanism that
// gets fixed rather than the entry.
//
// # Where the truth lives
//
// The names and their suppliers are also in the published catalog, beside the
// entries that use them. Neither copy is free to drift: a check compares this
// registry with that file in both directions, because the catalog is data the
// tool ships and this is the code that reads it.
package capability

import "sort"

// Name is the identifier of one capability, as written in a requires list.
type Name string

// Supplier is who brings a capability into a run.
//
// It is part of the model rather than a comment because a comment cannot be
// compared against the catalog, and a classification nothing checks is a
// classification that drifts.
type Supplier string

const (
	// Writer is the format writer, which owns the bytes and the file shape.
	Writer Supplier = "writer"
	// Schema is the schema view, which owns what the data declares about
	// itself.
	Schema Supplier = "schema"
	// Provider is a country or locale bound source of values and
	// identifiers.
	Provider Supplier = "provider"
	// Corpus is a reference word list compiled into the binary.
	Corpus Supplier = "corpus"
	// Engine is the corruption engine itself, needing nothing external.
	Engine Supplier = "engine"
)

// The capability names. They are constants rather than bare strings so that a
// typo fails to compile instead of quietly making a mutator inapplicable at
// run time, which is a failure nobody would see.
const (
	OutputCharsetControl    Name = "output.charset-control"
	OutputLineEndingControl Name = "output.line-ending-control"
	OutputByteLevelWrite    Name = "output.byte-level-write"
	OutputBOMControl        Name = "output.bom-control"
	FormatCSV               Name = "format.csv"
	FormatJSON              Name = "format.json"

	SchemaFieldType      Name = "schema.field-type"
	SchemaMaxLength      Name = "schema.max-length"
	SchemaEnumValues     Name = "schema.enum-values"
	SchemaRequiredFields Name = "schema.required-fields"
	SchemaFieldRelations Name = "schema.field-relations"
	RelationForeignKey   Name = "relation.foreign-key"
	KeyPrimary           Name = "key.primary"
	KeyUnique            Name = "key.unique"

	IdentifierCheckDigit       Name = "identifier.check-digit"
	IdentifierEncodesDate      Name = "identifier.encodes-date"
	IdentifierEncodesGender    Name = "identifier.encodes-gender"
	IdentifierReservedRange    Name = "identifier.reserved-range"
	IdentifierUnassignedTarget Name = "identifier.unassigned-target"
	ReferencePostalToPlace     Name = "reference.postal-to-place"
	ProviderEmail              Name = "provider.email"
	CurrencyMinorUnits         Name = "currency.minor-units"
	DateArithmetic             Name = "date.arithmetic"

	CorpusUnicode            Name = "corpus.unicode"
	CorpusCoercibleValues    Name = "corpus.coercible-values"
	CorpusSentinelValues     Name = "corpus.sentinel-values"
	CorpusNamesInternational Name = "corpus.names-international"

	UnicodeNormalization Name = "unicode.normalization"
)

// suppliers is the whole registry: every capability that exists, and who
// brings it.
var suppliers = map[Name]Supplier{
	OutputCharsetControl:    Writer,
	OutputLineEndingControl: Writer,
	OutputByteLevelWrite:    Writer,
	OutputBOMControl:        Writer,
	FormatCSV:               Writer,
	FormatJSON:              Writer,

	SchemaFieldType:      Schema,
	SchemaMaxLength:      Schema,
	SchemaEnumValues:     Schema,
	SchemaRequiredFields: Schema,
	SchemaFieldRelations: Schema,
	RelationForeignKey:   Schema,
	KeyPrimary:           Schema,
	KeyUnique:            Schema,

	IdentifierCheckDigit:       Provider,
	IdentifierEncodesDate:      Provider,
	IdentifierEncodesGender:    Provider,
	IdentifierReservedRange:    Provider,
	IdentifierUnassignedTarget: Provider,
	ReferencePostalToPlace:     Provider,
	ProviderEmail:              Provider,
	CurrencyMinorUnits:         Provider,
	DateArithmetic:             Provider,

	CorpusUnicode:            Corpus,
	CorpusCoercibleValues:    Corpus,
	CorpusSentinelValues:     Corpus,
	CorpusNamesInternational: Corpus,

	UnicodeNormalization: Engine,
}

// SupplierOf says who brings a capability, and whether it is known at all.
//
// The second result matters more than it looks. An unknown name is a defect
// in a catalog entry or in a provider, and answering with a zero supplier
// would turn it into a mutator that is quietly never applicable.
func SupplierOf(name Name) (Supplier, bool) {
	supplier, ok := suppliers[name]
	return supplier, ok
}

// Known reports whether a capability exists in the registry.
func Known(name Name) bool {
	_, ok := suppliers[name]
	return ok
}

// All returns every known capability, sorted.
func All() []Name {
	names := make([]Name, 0, len(suppliers))
	for name := range suppliers {
		names = append(names, name)
	}
	sortNames(names)
	return names
}

// SuppliedBy returns the capabilities one kind of supplier brings, sorted.
func SuppliedBy(supplier Supplier) []Name {
	var names []Name
	for name, who := range suppliers {
		if who == supplier {
			names = append(names, name)
		}
	}
	sortNames(names)
	return names
}

// Set is the capabilities present in one run, gathered from every supplier
// taking part in it.
//
// The zero value is usable and empty, which is the honest answer for a run
// that has no provider and no corpus rather than an error to handle.
type Set struct {
	present map[Name]bool
}

// NewSet gathers capabilities into a set.
func NewSet(names ...Name) Set {
	set := Set{present: make(map[Name]bool, len(names))}
	for _, name := range names {
		set.present[name] = true
	}
	return set
}

// With returns a set holding everything in this one plus the given names.
//
// It returns a new set rather than changing this one, so that assembling a
// run from several suppliers cannot accidentally hand two callers the same
// map.
func (s Set) With(names ...Name) Set {
	combined := Set{present: make(map[Name]bool, len(s.present)+len(names))}
	for name := range s.present {
		combined.present[name] = true
	}
	for _, name := range names {
		combined.present[name] = true
	}
	return combined
}

// Has reports whether the run can do this.
func (s Set) Has(name Name) bool {
	return s.present[name]
}

// Missing returns the required capabilities this run does not have, sorted.
//
// Sorted because the answer is quoted in the manifest as the reason a
// corruption was not applied, and Go randomises map iteration on purpose. An
// unsorted reason would differ between two runs of the same seed, which is
// the one thing this project is not allowed to do.
//
// A mutator that requires nothing gets an empty answer rather than an error.
func (s Set) Missing(required []Name) []Name {
	var missing []Name
	for _, name := range required {
		if !s.present[name] {
			missing = append(missing, name)
		}
	}
	sortNames(missing)
	return missing
}

// Covers reports whether the run can satisfy everything on the list.
func (s Set) Covers(required []Name) bool {
	return len(s.Missing(required)) == 0
}

// Names returns everything in the set, sorted.
func (s Set) Names() []Name {
	names := make([]Name, 0, len(s.present))
	for name := range s.present {
		names = append(names, name)
	}
	sortNames(names)
	return names
}

func sortNames(names []Name) {
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
}
