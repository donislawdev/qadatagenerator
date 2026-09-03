package recipeview_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
)

// The offer belongs to the empty screen. Anywhere else it would be a control
// asking to replace what somebody has written, which is a question this screen
// has no reason to ask.
func TestTheOfferOfAnExampleIsOnTheEmptyScreenAndNowhereElse(t *testing.T) {
	view := here(t, real(t))
	if !strings.Contains(rendered(t, view), i18n.English.T(i18n.RecipeExample)) {
		t.Error("the empty screen does not offer an example")
	}

	view.SetRecipe("version: \"0.1\"\n")
	if drawn := rendered(t, view); strings.Contains(drawn, i18n.English.T(i18n.RecipeExample)) {
		t.Errorf("the offer of an example is still on a screen with a recipe on it:\n%s", drawn)
	}
}

// 🔴 The list is held against the register in both directions, the way the same
// list is on the other surface: a generator missing from it is a value nobody
// can find, and a name in it that the register does not carry is a promise this
// build cannot keep.
func TestTheListNamesEveryGeneratorThisBuildHas(t *testing.T) {
	view := here(t, real(t))
	view.ShowGenerators()
	drawn := rendered(t, view)

	source := values.New("", "")
	for _, name := range source.Names() {
		if !strings.Contains(drawn, name+" - ") {
			t.Errorf("the list does not name %q", name)
		}
	}
	if listed := strings.Count(drawn, " - "); listed != len(source.Names()) {
		t.Errorf("the list has %d rows and the register carries %d generators",
			listed, len(source.Names()))
	}
}

// 🔴 A generator this build cannot fill in a locale says so with the reason,
// because a name on a list is a promise. Somebody who writes it and gets a
// refusal goes looking for a mistake in their own recipe, and there is none:
// the list this build carries is what is missing.
func TestTheListSaysWhereAGeneratorCannotBeFilled(t *testing.T) {
	view := here(t, real(t))
	view.ShowGenerators()
	drawn := rendered(t, view)

	source := values.New("", "")
	refused := 0
	for _, name := range source.Names() {
		for _, locale := range values.Locales() {
			fine, why := source.CanProduce(name, locale)
			if fine {
				continue
			}
			refused++
			said := spoken(drawn)
			for _, wanted := range []string{name, locale, why} {
				if !strings.Contains(said, wanted) {
					t.Errorf("%s cannot be filled in %s and the list does not say %q",
						name, locale, wanted)
				}
			}
		}
	}
	if refused == 0 {
		t.Fatal("every generator can be filled in every locale this build carries, so this " +
			"guard looked at nothing. It has to be rewritten rather than deleted")
	}
}

// 🔴 The list stays open while somebody types, and this is the whole reason it
// is built once rather than with the answer under it. The answer is redrawn on
// every key press, and a list rebuilt with it would shut itself the moment
// anybody used it while reading it.
func TestTheListStaysOpenWhileSomebodyTypes(t *testing.T) {
	view := here(t, real(t))
	view.ShowGenerators()
	first := rendered(t, view)

	view.SetRecipe("version: \"0.1\"\nentities:\n  a:\n    count: 1\n    fields:\n      x: {}\n")

	name := values.New("", "").Names()[0]
	if !strings.Contains(first, name+" - ") {
		t.Fatalf("the list was not open before the recipe was typed, so this proves nothing:\n%s",
			first)
	}
	if drawn := rendered(t, view); !strings.Contains(drawn, name+" - ") {
		t.Errorf("the list shut itself when the answer changed:\n%s", drawn)
	}
}

// spoken is everything the screen says, as one run of words.
//
// 🔴 A wrapped sentence does not appear in the drawing as a sentence. The
// toolkit lays it out line by line and each line is a text of its own, so a
// guard looking for the whole of one finds nothing however plainly the screen
// says it. Joining the pieces asks what the screen SAYS rather than how it
// broke the line, which is the same rule that put an end to a guard here
// looking for the letters of the word line and finding linearGradient.
func spoken(drawn string) string {
	var said []string
	for _, piece := range strings.Split(drawn, "<text") {
		opened := strings.Index(piece, ">")
		closed := strings.Index(piece, "</text>")
		if opened < 0 || closed < opened {
			continue
		}
		said = append(said, strings.TrimSpace(piece[opened+1:closed]))
	}
	return strings.Join(said, " ")
}

// Contract check: the probe that photographs this state reaches it the same way
// these guards do.
var _ = (*recipeview.View).ShowGenerators
