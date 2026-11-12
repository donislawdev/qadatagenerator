package quickview

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// TestEveryRefusalOfTheModeHasWordsOnThisScreen holds the register of this
// screen against the refusals the mode can produce, in both directions.
//
// A kind with no key here is shown as the text of an empty key, so a person
// refused that way is told nothing. Words for a kind the mode no longer has are
// a sentence about something nobody can be refused for.
func TestEveryRefusalOfTheModeHasWordsOnThisScreen(t *testing.T) {
	for _, gap := range refusedSaysGaps(refusedSays) {
		t.Error(gap)
	}
}

// TestTheGuardOfTheRefusalsSeesAGapInEitherDirection is the canary of the guard
// above: a register with one kind taken out and one the mode does not have put
// in, each of which the guard has to name.
func TestTheGuardOfTheRefusalsSeesAGapInEitherDirection(t *testing.T) {
	planted := maps.Clone(refusedSays)
	delete(planted, quick.Unavailable)
	planted[quick.RefusalKind(9999)] = i18n.QuickRefusedDate
	if gaps := refusedSaysGaps(planted); len(gaps) != 2 {
		t.Errorf("the guard found %d gaps in a register with two planted: %q", len(gaps), gaps)
	}
}

// TestAValueWithNoDataInTheLocaleIsRefusedInTheWordsOfThisScreen holds that the
// refusal is this screen's sentence rather than the engine's. The engine words
// its reason in English for the command line, and this screen is translated.
func TestAValueWithNoDataInTheLocaleIsRefusedInTheWordsOfThisScreen(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	view.UseSeed("for-the-test")
	view.UseLocale("pl-PL")
	view.Choose("postal_code")

	if said, want := view.Told(), i18n.English.T(i18n.QuickRefusedUnavailable); said != want {
		t.Errorf("a postal code in pl-PL was refused with %q rather than %q", said, want)
	}
}

// refusedSaysGaps is every way the register fails to answer for the mode. A
// key with no English text is the catalogue's business and has its own guard,
// so an empty key is the one thing about the words checked here.
func refusedSaysGaps(register map[quick.RefusalKind]i18n.Key) []string {
	kinds := quick.Kinds()
	if len(kinds) == 0 {
		return []string{"the mode reports no refusals at all, so this guard would check nothing"}
	}
	var gaps []string
	for _, kind := range kinds {
		if register[kind] == "" {
			gaps = append(gaps, fmt.Sprintf("the mode refuses as kind %d and this screen "+
				"has no words for it, so somebody would be refused in silence", kind))
		}
	}
	for kind := range register {
		if !slices.Contains(kinds, kind) {
			gaps = append(gaps, fmt.Sprintf("this screen carries words for kind %d, which "+
				"the mode does not refuse", kind))
		}
	}
	return gaps
}
