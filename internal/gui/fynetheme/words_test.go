package fynetheme

import (
	"sort"
	"strings"
	"testing"

	"fyne.io/fyne/v2/lang"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// Every word handed to the toolkit is one the catalogue actually carries.
//
// A key the catalogue lacks answers with its own name, so the file dialog
// would read "toolkit.cancel" on its button and nothing would fail.
func TestEveryWordHandedToTheToolkitIsInTheCatalogue(t *testing.T) {
	for id, key := range toolkitWords {
		if said := i18n.English.T(key); said == string(key) || said == "" {
			t.Errorf("the toolkit asks for %q and the catalogue has no text for %s", id, key)
		}
	}
}

// 🔴 After Speak, the toolkit says what the catalogue says, on whatever desktop
// this runs.
//
// The catalogue here carries a marker for every word rather than English,
// because English is also what the toolkit falls back to when it finds nothing,
// so a test asking for English would pass with Speak doing nothing at all. The
// markers can only come from the catalogue. On a Polish desktop this is the
// test that fails without Speak, measured by leaving the call out.
func TestTheToolkitSaysTheWordsOfTheCatalogue(t *testing.T) {
	ids := make([]string, 0, len(toolkitWords))
	var source strings.Builder
	for id, key := range toolkitWords {
		ids = append(ids, id)
		source.WriteString(string(key) + ": marker " + string(key) + "\n")
	}
	sort.Strings(ids)
	marked, err := i18n.Parse("xx", []byte(source.String()))
	if err != nil {
		t.Fatalf("the test catalogue did not parse: %v", err)
	}
	t.Cleanup(func() {
		if err := Speak(i18n.English); err != nil {
			t.Errorf("putting English back failed: %v", err)
		}
	})

	if err := Speak(marked); err != nil {
		t.Fatalf("the toolkit refused the words: %v", err)
	}
	for _, id := range ids {
		want := "marker " + string(toolkitWords[id])
		if said := lang.X(id, "fallback of the toolkit"); said != want {
			t.Errorf("the toolkit says %q for %q, want %q from the catalogue", said, id, want)
		}
	}
}
