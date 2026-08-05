package catalog_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

// usable is an entry with nothing wrong with it, for a test to break in one
// place at a time. Built here rather than read from the shipped catalog,
// because a fixture taken from the thing under test agrees with it however
// wrong both are.
func usable() catalog.Entry {
	return catalog.Entry{
		ID:      "L2.number.text_in_numeric",
		Layer:   catalog.FieldShape,
		Summary: "Puts letters where a number belongs.",
		AppliesTo: catalog.AppliesTo{
			FieldTypes: []string{"integer"},
			Formats:    []string{"csv"},
			Providers:  []string{catalog.Any},
		},
		Expectation: catalog.Rejection,
		Confidence:  catalog.Hard,
		Reversible:  true,
		Requires:    []capability.Name{capability.FormatCSV},
		Example:     catalog.Example{Before: "42", After: "forty two"},
	}
}

func TestAWholeEntryIsAccepted(t *testing.T) {
	built, problems := catalog.New("0.1", "draft", []catalog.Entry{usable()})
	if len(problems) > 0 {
		t.Fatalf("an entry with nothing wrong with it was refused: %v", problems)
	}
	if built.Len() != 1 {
		t.Errorf("the catalog holds %d entries rather than one", built.Len())
	}
	if built.Version() != "0.1" || built.Status() != "draft" {
		t.Errorf("version %q status %q did not survive", built.Version(), built.Status())
	}
	if _, ok := built.Lookup("L2.number.text_in_numeric"); !ok {
		t.Error("the entry cannot be looked up by its own identifier")
	}
}

// Each of these is a defect that would otherwise reach a manifest, and each is
// broken in exactly one place so that the message names the right thing.
func TestEveryWayAnEntryCanBeUnusableIsRefused(t *testing.T) {
	cases := []struct {
		what   string
		break_ func(*catalog.Entry)
		says   string
	}{
		{"no identifier", func(e *catalog.Entry) { e.ID = "" }, "carries no identifier"},
		{"unknown layer", func(e *catalog.Entry) { e.Layer = "L9" }, "not one this build knows"},
		{"layer disagrees with the identifier",
			func(e *catalog.Entry) { e.Layer = catalog.Domain }, "does not begin with its own layer"},
		{"no summary", func(e *catalog.Entry) { e.Summary = " " }, "no summary"},
		{"unknown expectation",
			func(e *catalog.Entry) { e.Expectation = "shrug" }, "not one of rejection"},
		{"unknown confidence",
			func(e *catalog.Entry) { e.Confidence = "probably" }, "neither hard nor soft"},
		{"soft with nothing explaining it", func(e *catalog.Entry) {
			e.Confidence = catalog.Soft
			e.Example.Note = ""
			e.ApplicabilityCondition = ""
		}, "nothing saying what the reaction depends on"},
		{"a dimension left empty",
			func(e *catalog.Entry) { e.AppliesTo.Formats = nil }, "applies_to.formats is empty"},
		{"a capability nobody supplies",
			func(e *catalog.Entry) { e.Requires = []capability.Name{"format.telepathy"} },
			"which no supplier in this build brings"},
		{"a parameter with no name", func(e *catalog.Entry) {
			e.Parameters = []catalog.Parameter{{Description: "How much."}}
		}, "carries no name"},
		{"a parameter with no description", func(e *catalog.Entry) {
			e.Parameters = []catalog.Parameter{{Name: "ratio"}}
		}, "carries no description"},
		{"nothing to damage", func(e *catalog.Entry) { e.Example.Before = "" }, "no before value"},
	}

	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			entry := usable()
			one.break_(&entry)
			built, problems := catalog.New("0.1", "draft", []catalog.Entry{entry})
			if len(problems) == 0 {
				t.Fatalf("%s was accepted", one.what)
			}
			if !strings.Contains(strings.Join(problems, "\n"), one.says) {
				t.Errorf("the refusal does not say %q: %v", one.says, problems)
			}
			if built.Len() != 0 {
				t.Errorf("a catalog of %d entries came back with the problems", built.Len())
			}
		})
	}
}

// A soft entry explains what the reaction depends on. Either place will do,
// because both reach a reader.
func TestSoftConfidenceIsExplainedByEitherPlace(t *testing.T) {
	withNote := usable()
	withNote.Confidence = catalog.Soft
	withNote.Example.Note = "Depends on whether the system holds the reference data."
	if _, problems := catalog.New("0.1", "draft", []catalog.Entry{withNote}); len(problems) > 0 {
		t.Errorf("a soft entry explained in its note was refused: %v", problems)
	}

	withCondition := usable()
	withCondition.Confidence = catalog.Soft
	withCondition.ApplicabilityCondition = "Only where the field is nullable."
	if _, problems := catalog.New("0.1", "draft", []catalog.Entry{withCondition}); len(problems) > 0 {
		t.Errorf("a soft entry explained by its condition was refused: %v", problems)
	}
}

// One entry missing from a catalog is a mutator a policy can order and the
// manifest will describe neither way, so a broken entry takes the rest with it.
func TestOneBrokenEntryRefusesTheWholeCatalog(t *testing.T) {
	good := usable()
	bad := usable()
	bad.ID = "L2.number.second_one"
	bad.Expectation = "shrug"

	built, problems := catalog.New("0.1", "draft", []catalog.Entry{good, bad})
	if len(problems) == 0 {
		t.Fatal("a catalog with one broken entry was accepted")
	}
	if built.Len() != 0 {
		t.Errorf("%d entries came back, and a partial catalog is the thing being prevented", built.Len())
	}
}

func TestTwoEntriesUnderOneIdentifierAreRefused(t *testing.T) {
	_, problems := catalog.New("0.1", "draft", []catalog.Entry{usable(), usable()})
	if !strings.Contains(strings.Join(problems, "\n"), "two entries carry this identifier") {
		t.Errorf("a duplicated identifier was not named: %v", problems)
	}
}

func TestAnEmptyCatalogAndAMissingVersionAreRefused(t *testing.T) {
	if _, problems := catalog.New("0.1", "draft", nil); len(problems) == 0 {
		t.Error("a catalog with no entries was accepted, and this build could then do no damage at all")
	}
	if _, problems := catalog.New("", "draft", []catalog.Entry{usable()}); len(problems) == 0 {
		t.Error("a catalog with no version was accepted, and the manifest header quotes it")
	}
}

// Go randomises map iteration on purpose, so an order that came from a map
// would differ between two runs of one seed. These identifiers reach the
// manifest, which is the one place that cannot happen.
func TestIdentifiersComeBackSortedAndDetached(t *testing.T) {
	third := usable()
	third.ID = "L2.number.zzz"
	second := usable()
	second.ID = "L2.number.mmm"
	built, problems := catalog.New("0.1", "draft", []catalog.Entry{third, usable(), second})
	if len(problems) > 0 {
		t.Fatalf("refused: %v", problems)
	}

	ids := built.IDs()
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			t.Fatalf("identifiers came back out of order: %v", ids)
		}
	}

	ids[0] = "changed by the caller"
	if built.IDs()[0] == "changed by the caller" {
		t.Error("a caller can reach into the catalog through the slice it was handed")
	}
}

func TestLookupSaysWhenItDoesNotKnow(t *testing.T) {
	built, _ := catalog.New("0.1", "draft", []catalog.Entry{usable()})
	if _, ok := built.Lookup("L3.checksum.invented"); ok {
		t.Error("an identifier nobody described was answered as known")
	}
}

func TestEntriesCanBeReadByLayer(t *testing.T) {
	other := usable()
	other.ID = "L3.checksum.off_by_one"
	other.Layer = catalog.Domain
	built, problems := catalog.New("0.1", "draft", []catalog.Entry{usable(), other})
	if len(problems) > 0 {
		t.Fatalf("refused: %v", problems)
	}
	if got := built.InLayer(catalog.Domain); len(got) != 1 || got[0].ID != "L3.checksum.off_by_one" {
		t.Errorf("reading layer L3 gave %v", got)
	}
	if got := built.InLayer(catalog.Security); len(got) != 0 {
		t.Errorf("a layer with no entries gave %v rather than nothing", got)
	}
}

// The zero value is what a caller holds before anything is loaded. It answers
// rather than panicking, because a nil map read is the sort of failure that
// surfaces far from its cause.
func TestTheEmptyCatalogAnswersRatherThanFailing(t *testing.T) {
	var empty catalog.Catalog
	if empty.Len() != 0 || len(empty.IDs()) != 0 || len(empty.Entries()) != 0 {
		t.Error("the zero catalog claims to hold something")
	}
	if _, ok := empty.Lookup("L2.number.text_in_numeric"); ok {
		t.Error("the zero catalog answered that it knows an entry")
	}
}
