package capability_test

import (
	"sort"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
)

// The defect this package exists to prevent, asked about directly.
//
// The architecture once said a provider declares the capabilities and that a
// mutator is available when the provider brings them. Under that reading the
// capabilities below would have to come from a provider, no provider could
// ever bring them, and every mutator of layer L0 and L1 would report that it
// could not be applied. The names are listed one by one rather than counted,
// because a count passes for the wrong reason as soon as somebody adds one.
func TestTheCapabilitiesNoProviderCanBringAreNotProviderCapabilities(t *testing.T) {
	notFromAProvider := map[capability.Name]capability.Supplier{
		capability.OutputCharsetControl:    capability.Writer,
		capability.OutputLineEndingControl: capability.Writer,
		capability.OutputByteLevelWrite:    capability.Writer,
		capability.OutputBOMControl:        capability.Writer,
		capability.FormatCSV:               capability.Writer,
		capability.FormatJSON:              capability.Writer,
		capability.SchemaFieldType:         capability.Schema,
		capability.SchemaMaxLength:         capability.Schema,
		capability.SchemaRequiredFields:    capability.Schema,
		capability.KeyPrimary:              capability.Schema,
		capability.CorpusUnicode:           capability.Corpus,
		capability.UnicodeNormalization:    capability.Engine,
	}

	for name, want := range notFromAProvider {
		got, known := capability.SupplierOf(name)
		if !known {
			t.Errorf("%s is not in the registry at all", name)
			continue
		}
		if got != want {
			t.Errorf("%s is supplied by %q, expected %q", name, got, want)
		}
	}

	for _, name := range capability.SuppliedBy(capability.Provider) {
		if _, listed := notFromAProvider[name]; listed {
			t.Errorf("%s is listed as coming from a provider", name)
		}
	}
}

// A capability that is asked for but does not exist is a defect in whoever
// asked. Answering with a zero supplier would turn it into a mutator that is
// silently never applicable, which is the failure nobody sees.
func TestAnUnknownCapabilityIsAnsweredAsUnknown(t *testing.T) {
	if capability.Known("identifier.check-digits") {
		t.Error("a near miss of a real name is reported as known")
	}
	if _, known := capability.SupplierOf("nonsense"); known {
		t.Error("an invented name has a supplier")
	}
}

// Every name in the registry has to have a supplier, and the answers have to
// come back in a fixed order. Go randomises map iteration on purpose, and
// these lists end up in a manifest.
func TestTheRegistryIsCompleteAndSorted(t *testing.T) {
	all := capability.All()
	if len(all) == 0 {
		t.Fatal("the registry is empty")
	}
	if !sort.SliceIsSorted(all, func(i, j int) bool { return all[i] < all[j] }) {
		t.Errorf("All is not sorted: %v", all)
	}

	counted := 0
	for _, supplier := range []capability.Supplier{
		capability.Writer, capability.Schema, capability.Provider,
		capability.Corpus, capability.Engine,
	} {
		names := capability.SuppliedBy(supplier)
		if !sort.SliceIsSorted(names, func(i, j int) bool { return names[i] < names[j] }) {
			t.Errorf("SuppliedBy(%s) is not sorted: %v", supplier, names)
		}
		counted += len(names)
	}
	if counted != len(all) {
		t.Errorf("the five suppliers account for %d capabilities out of %d, so one has a supplier outside the set", counted, len(all))
	}
}

func TestMissingReportsWhatIsAbsent(t *testing.T) {
	run := capability.NewSet(capability.FormatCSV, capability.SchemaFieldType)

	// Sorted, because this list is quoted as the reason a corruption was not
	// applied, and two runs of one seed have to say it the same way.
	missing := run.Missing([]capability.Name{
		capability.UnicodeNormalization,
		capability.IdentifierCheckDigit,
		capability.FormatCSV,
	})
	want := []capability.Name{capability.IdentifierCheckDigit, capability.UnicodeNormalization}
	if len(missing) != len(want) {
		t.Fatalf("expected %v, got %v", want, missing)
	}
	for i := range want {
		if missing[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, missing)
		}
	}

	if !run.Covers([]capability.Name{capability.FormatCSV}) {
		t.Error("a capability that is present is reported as missing")
	}
}

// The degenerate inputs this will actually receive: a run with no provider and
// no corpus, and a mutator that needs nothing.
func TestTheEmptyCasesAreAnsweredRatherThanFailing(t *testing.T) {
	var nothing capability.Set

	if nothing.Has(capability.FormatCSV) {
		t.Error("an empty set claims a capability")
	}
	if got := nothing.Missing(nil); got != nil {
		t.Errorf("a mutator requiring nothing is missing nothing, got %v", got)
	}
	if !nothing.Covers(nil) {
		t.Error("a mutator requiring nothing is not covered by an empty run")
	}
	if got := nothing.Missing([]capability.Name{capability.FormatCSV}); len(got) != 1 {
		t.Errorf("expected the one missing name, got %v", got)
	}
	if got := nothing.Names(); len(got) != 0 {
		t.Errorf("an empty set names something: %v", got)
	}
}

// Assembling a run adds capabilities from several suppliers, and doing so must
// not reach back into a set somebody else is holding.
func TestWithDoesNotChangeTheSetItCameFrom(t *testing.T) {
	writer := capability.NewSet(capability.FormatCSV)
	whole := writer.With(capability.IdentifierCheckDigit)

	if writer.Has(capability.IdentifierCheckDigit) {
		t.Error("With changed the set it was called on")
	}
	if !whole.Has(capability.FormatCSV) || !whole.Has(capability.IdentifierCheckDigit) {
		t.Error("With lost something")
	}

	// The zero value has no map, so this is where a naive implementation
	// panics rather than answering.
	var nothing capability.Set
	if got := nothing.With(capability.FormatJSON); !got.Has(capability.FormatJSON) {
		t.Error("With on an empty set produced nothing")
	}
}
