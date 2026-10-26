package recipeyaml_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
)

// The list under values and the key held whole, as the reader sees them
// (DEC-266, 2026-10-26). Each has a test of its own rather than a line in the
// table of problems, because each is a guard with a mutation of its own.

// refusedAt fails unless reading the recipe reports a problem at one place
// saying one thing.
func refusedAt(t *testing.T, source, where, what string) {
	t.Helper()
	result := recipeyaml.Decode([]byte(source))
	for _, found := range result.Problems {
		if found.Where == where && strings.Contains(found.What, what) {
			return
		}
	}
	for _, found := range result.Problems {
		t.Logf("got %q: %s", found.Where, found.What)
	}
	t.Fatalf("no problem at %q saying %q", where, what)
}

const withField = "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: "

// 🔴 The list and a generator are two places for the value to come from, and
// whichever won would be the tool choosing.
func TestAListUnderValuesIsRefusedBesideAnotherSource(t *testing.T) {
	refusedAt(t, withField+"{ generator: y, values: [\"p\", \"q\"] }\n",
		"entities.a.fields.x.values", "also names a generator or an expression")
	refusedAt(t, withField+"{ expression: \"${y}\", values: [\"p\"] }\n",
		"entities.a.fields.x.values", "also names a generator or an expression")
}

// A list with nothing in it and an entry of nothing are both refused, the
// second because a blank row is what the share under empty says.
func TestAnEmptyListOrAnEmptyEntryIsRefused(t *testing.T) {
	refusedAt(t, withField+"{ values: [] }\n", "entities.a.fields.x.values", "the list of values is empty")
	refusedAt(t, withField+"{ values: [\"p\", \"\"] }\n", "entities.a.fields.x.values[1]", "the entry is empty")
}

// An entry written twice has no meaning this build gives it, and is refused at
// the place of the second one.
func TestAnEntryWrittenTwiceInTheListIsRefused(t *testing.T) {
	refusedAt(t, withField+"{ values: [\"p\", \"q\", \"p\"] }\n", "entities.a.fields.x.values[2]", "names \"p\" twice")
}

// A key with a blank part identifies nothing, and the run holds the key whole.
func TestAPartOfTheKeyAskedToBeEmptyIsRefused(t *testing.T) {
	refusedAt(t, "version: \"0.1\"\nentities:\n  a:\n    primary_key: [x]\n    fields:\n      x: { generator: y, empty: 5% }\n",
		"entities.a.primary_key", "the key includes x, which is asked to be empty")
}
