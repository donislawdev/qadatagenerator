package catalogyaml

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// The tests here are inside the package rather than beside it, because the
// refusals are the point and proving them needs a broken catalog fed to the
// reader. The shipped one cannot be broken on purpose, so parse takes the bytes
// as an argument and this is the only place that can hand it different ones.

func TestTheShippedCatalogLoads(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("the catalog this build ships does not load: %v", err)
	}
	if loaded.Len() == 0 {
		t.Fatal("the catalog loaded and carries no entries")
	}
	if loaded.Version() == "" {
		t.Error("the catalog carries no version, and the manifest header quotes it")
	}
}

// Every entry has to arrive complete, and the fields most easily lost in a
// mapping are the ones a reader would not miss until much later.
func TestEveryEntryArrivesWithWhatTheModelNeeds(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, entry := range loaded.Entries() {
		if entry.Summary == "" {
			t.Errorf("%s: summary did not survive the read", entry.ID)
		}
		if len(entry.AppliesTo.FieldTypes) == 0 || len(entry.AppliesTo.Formats) == 0 ||
			len(entry.AppliesTo.Providers) == 0 {
			t.Errorf("%s: applies_to did not survive the read", entry.ID)
		}
		if entry.Example.Before == "" {
			t.Errorf("%s: the before value did not survive the read", entry.ID)
		}
		for _, required := range entry.Requires {
			if !capability.Known(required) {
				t.Errorf("%s: requires %q, which no supplier brings", entry.ID, required)
			}
		}
	}
}

// A default is carried with the value the catalog wrote, and a default written
// as null is one of those values rather than a gap. Reading null as an absent
// key would turn the one mutator that uses it into a mutator with nothing to
// write, and dropping the values altogether would leave every mutator with
// nothing to tune.
func TestParameterDefaultsSurviveIncludingTheOneWrittenAsNull(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	written := defaultOf(t, loaded, "L1.json.duplicate_key", "second_value")
	if written != nil {
		t.Errorf("second_value default is %v, and the catalog writes null", written)
	}

	// The other half. Without it, a reader that dropped every default would
	// still pass the assertion above, since it too would answer nil.
	charset := defaultOf(t, loaded, "L0.encoding.wrong_charset", "target_charset")
	if charset == nil {
		t.Error("target_charset came back with no default, and the catalog names one")
	}
}

func defaultOf(t *testing.T, loaded catalog.Catalog, id catalog.ID, parameter string) any {
	t.Helper()
	entry, ok := loaded.Lookup(id)
	if !ok {
		t.Fatalf("%s is not in this catalog any more", id)
	}
	for _, carried := range entry.Parameters {
		if carried.Name == parameter {
			return carried.Default
		}
	}
	t.Fatalf("%s has no %s parameter any more", id, parameter)
	return nil
}

// An empty after value is a result rather than a gap, and one entry exists to
// produce it. Reading it as a missing key would refuse the entry for doing
// exactly what it says it does, which is how the first run of this reader
// failed.
func TestAnEmptyAfterValueIsAResultAndNotAGap(t *testing.T) {
	loaded, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entry, ok := loaded.Lookup("L2.string.empty")
	// 🔴 A failure and not a skip (B-319). An identifier of a mutator is never
	// removed, so its absence is a fault of its own, and a skip would have
	// turned it green.
	if !ok {
		t.Fatal("L2.string.empty is not in this catalog any more, and an identifier of a " +
			"mutator is deprecated rather than removed")
	}
	if entry.Example.After != "" {
		t.Errorf("after is %q, and the mutator replaces a text value with an empty string",
			entry.Example.After)
	}
	if entry.Example.Before == "" {
		t.Error("before is empty, so the entry starts from nothing and shows nothing")
	}
}

// A value left unquoted in the file is read as whatever it resembles, and the
// catalog carries dates and digit strings where that would change the value.
func TestAnExampleThatIsNotTextIsRefused(t *testing.T) {
	_, err := parse([]byte(strings.Replace(
		withOneMutator(""), `before: "42"`, "before: 010", 1)))
	requireRefusal(t, err, "rather than text, so quote it")
}

// A parameter with no default at all is the one defect the model cannot see,
// because by the time it has a Parameter there is nothing left to tell an
// absent key from a null one.
func TestAParameterWithNoDefaultIsRefused(t *testing.T) {
	_, err := parse([]byte(withOneMutator(`
    parameters:
      - name: ratio
        description: How much of it to do.`)))
	requireRefusal(t, err, "has no default")
}

// A parameter with no name cannot be set by a policy and cannot be named in a
// manifest, so it is a knob nobody can reach.
//
// The check that catches it lives in the model, and the reader has to hand the
// model what the file actually said. It used to substitute a stand in name for
// the message and pass that on as well, which left the model looking at a
// parameter that seemed named and the check silent. Found by reading rather
// than by any test, which is why there is now one.
func TestAParameterWithNoNameIsRefused(t *testing.T) {
	_, err := parse([]byte(withOneMutator(`
    parameters:
      - default: 0.5
        description: How much of it to do.`)))
	requireRefusal(t, err, "carries no name")
}

func TestAnEntryWithAnUnknownLayerIsRefused(t *testing.T) {
	_, err := parse([]byte(strings.Replace(withOneMutator(""), "layer: L2", "layer: L9", 1)))
	requireRefusal(t, err, "not one this build knows")
}

func TestAnEntryWhoseIdentifierDisagreesWithItsLayerIsRefused(t *testing.T) {
	_, err := parse([]byte(strings.Replace(withOneMutator(""), "layer: L2", "layer: L3", 1)))
	requireRefusal(t, err, "does not begin with its own layer")
}

func TestAnEntryRequiringACapabilityNobodySuppliesIsRefused(t *testing.T) {
	_, err := parse([]byte(strings.Replace(
		withOneMutator(""), "- format.csv", "- format.telepathy", 1)))
	requireRefusal(t, err, "which no supplier in this build brings")
}

func TestAnEntryWithoutAnExpectationIsRefused(t *testing.T) {
	_, err := parse([]byte(strings.Replace(withOneMutator(""), "expectation: rejection", "", 1)))
	requireRefusal(t, err, "expectation")
}

func TestTwoEntriesWithOneIdentifierAreRefused(t *testing.T) {
	doubled := withOneMutator("") + strings.SplitN(withOneMutator(""), "mutators:", 2)[1]
	_, err := parse([]byte(doubled))
	requireRefusal(t, err, "two entries carry this identifier")
}

func TestACatalogWithNoEntriesIsRefused(t *testing.T) {
	_, err := parse([]byte("catalog_version: \"0.1\"\nstatus: draft\nmutators: []\n"))
	requireRefusal(t, err, "knows no damage it can do")
}

func TestSomethingThatIsNotYAMLIsRefused(t *testing.T) {
	_, err := parse([]byte("catalog_version: \"0.1\"\n  mutators: [\n"))
	requireRefusal(t, err, "not readable as YAML")
}

// One broken catalog gives back every problem it has rather than the first,
// which is the same habit the recipe reader has and for the same reason.
func TestEveryProblemIsReportedAtOnce(t *testing.T) {
	broken := strings.NewReplacer(
		"layer: L2", "layer: L9",
		"confidence: hard", "confidence: probably",
	).Replace(withOneMutator(""))
	_, err := parse([]byte(broken))
	if err == nil {
		t.Fatal("a catalog with two problems was accepted")
	}
	for _, want := range []string{"not one this build knows", "neither hard nor soft"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not mention %q:\n%v", want, err)
		}
	}
}

// A catalog that will not read is a defect in this binary rather than in the
// user's recipe. Giving it the configuration kind would send somebody to look
// at their own file for our mistake.
func TestARefusalSaysItIsOurDefectAndNotTheirRecipe(t *testing.T) {
	_, err := parse([]byte("catalog_version: \"0.1\"\nstatus: draft\nmutators: []\n"))
	if err == nil {
		t.Fatal("an empty catalog was accepted")
	}
	if got := fault.KindOf(err); got != fault.Runtime {
		t.Errorf("kind is %v, and a broken built in catalog is a tool failure", got)
	}
	message := err.Error()
	for _, part := range []string{
		"could not be read",        // what happened
		"knows no damage",          // why
		"A usable catalog carries", // what a good one looks like
		"qadg --version",           // what to do about it
	} {
		if !strings.Contains(message, part) {
			t.Errorf("the refusal has no %q in it:\n%s", part, message)
		}
	}
}

// The model refuses a catalog with any problem in it rather than dropping the
// entry, because an entry quietly missing is a mutator a policy can order and
// the manifest will describe neither way.
func TestABrokenEntryTakesTheWholeCatalogWithIt(t *testing.T) {
	good := withOneMutator("")
	broken := good + strings.Replace(
		strings.SplitN(good, "mutators:", 2)[1],
		"id: L2.number.text_in_numeric", "id: L2.number.second_one\n    layer: L9", 1)
	loaded, err := parse([]byte(broken))
	if err == nil {
		t.Fatal("a catalog with one broken entry was accepted")
	}
	if loaded.Len() != 0 {
		t.Errorf("a partial catalog of %d entries came back with the error", loaded.Len())
	}
}

func requireRefusal(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("this was accepted, and it should have been refused for %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal does not say %q:\n%v", want, err)
	}
}

// withOneMutator is a smallest usable catalog, with room to replace one block.
// It is written out here rather than cut from the shipped file, because a
// fixture taken from the thing under test agrees with it however wrong both
// are.
func withOneMutator(parameters string) string {
	if parameters == "" {
		parameters = "\n    parameters: []"
	}
	return `catalog_version: "0.1"
status: draft

mutators:

  - id: L2.number.text_in_numeric
    layer: L2
    summary: Puts letters where a number belongs.
    applies_to:
      field_types: [integer]
      formats: [csv]
      providers: [any]` + parameters + `
    expectation: rejection
    confidence: hard
    reversible: true
    requires:
      - format.csv
    example:
      before: "42"
      after: "forty two"
`
}
