package recipeview

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// caretMark stands where the caret is in the places below.
const caretMark = "‸"

// placesOfARecipe are the places a recipe offers something, one for every kind
// of offer: every section's keys, every key's values, a rule and the names a
// recipe gives. Whatever the list can show, one of these shows it.
var placesOfARecipe = []string{
	caretMark,
	"version: " + caretMark,
	"locale: " + caretMark,
	"entities:\n  c:\n    " + caretMark,
	"entities:\n  c:\n    fields:\n      a: { " + caretMark + " }",
	"entities:\n  c:\n    fields:\n      a: { type: " + caretMark + " }",
	"entities:\n  c:\n    fields:\n      a: { generator: " + caretMark + " }",
	"entities:\n  c:\n    fields:\n      a: { unique: " + caretMark + " }",
	"entities:\n  c:\n    fields:\n      a: { counted_in: " + caretMark + " }",
	"entities:\n  c:\n    primary_key: [" + caretMark + "]\n    fields:\n      a: {}",
	"entities:\n  c:\n    fields:\n      a: {}\n      b: {}\n    consistency:\n      - a " + caretMark,
	"entities:\n  c:\n    fields:\n      a: {}\n  d:\n    belongs_to:\n      " + caretMark,
	"policy:\n  " + caretMark,
	"policy:\n  layers: [" + caretMark + "]",
	"policy:\n  per_row: " + caretMark,
	"output:\n  " + caretMark,
	"output:\n  format: " + caretMark,
	"output:\n  dialect: " + caretMark,
	"output:\n  file_per_entity: " + caretMark,
	"output:\n  encoding: " + caretMark,
}

// offeredView is the screen with the real catalog's layers, the way the window
// builds it.
func offeredView(t *testing.T) *View {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("this build cannot read its own catalog of damage: %v", err)
	}
	nowhere := outward.NewFolders(i18n.English, outward.Nowhere{}, background.ReadHere, nil)
	view := New(i18n.English, Checker(described), nil, background.Work{Read: background.ReadHere}, nowhere, nil)
	view.OfferLayersOf(described)
	return view
}

// Every offer the list can show carries what it means, and every line of the
// catalogue written for an offer is one the list shows: closed from both sides,
// the way the names of the kinds and the layers are.
//
// A line missing here would put its own key beside the offer, such as
// recipe.key.count, which is the window reading a settings file at somebody. A
// line nothing offers is a sentence translated into every language for a list
// that never shows it.
func TestEveryOfferSaysWhatItMeansAndEveryMeaningIsOffered(t *testing.T) {
	view := offeredView(t)
	said := map[i18n.Key]bool{}
	shown := 0
	for _, marked := range placesOfARecipe {
		at := strings.Index(marked, caretMark)
		text := strings.Replace(marked, caretMark, "", 1)
		asked := recipeyaml.Suggest(text, utf8.RuneCountInString(marked[:at]), view.vocabulary)
		if len(asked.Offers) == 0 {
			t.Errorf("nothing is offered at %q, so it proves nothing about meanings", marked)
		}
		for _, offer := range asked.Offers {
			shown++
			if key, described := describedBy(offer); described {
				said[key] = true
			}
			meaning := view.describe(offer)
			if meansOnlyItself(offer) {
				continue
			}
			if meaning == "" || meaning == offer.Word || looksLikeAKey(meaning) {
				t.Errorf("%s at %q is shown beside %q rather than what it means", offer.Word, marked, meaning)
			}
		}
	}
	if shown == 0 {
		t.Fatal("the list offered nothing anywhere, so every check here passed by checking nothing")
	}
	for _, name := range i18n.English.Names() {
		written := strings.HasPrefix(string(name), "recipe.key.") || strings.HasPrefix(string(name), "recipe.value.")
		if written && !said[name] {
			t.Errorf("the catalogue says what %q means and the list never offers it", name)
		}
	}
}

// meansOnlyItself is an offer that stands alone in the list: a name the recipe
// gave, and a value such as true or utf-8.
func meansOnlyItself(offer recipeyaml.Offer) bool {
	if offer.Kind == recipeyaml.OfferName {
		return true
	}
	return offer.Kind == recipeyaml.OfferValue &&
		slices.Contains([]string{"unique", "required", "file_per_entity", "encoding"}, offer.Key)
}

// looksLikeAKey is a meaning that is the key of the catalogue rather than its
// line, which is what the catalogue answers for a key it does not carry.
func looksLikeAKey(meaning string) bool {
	for _, prefix := range []string{"recipe.", "kind.", "country.", "take."} {
		if strings.HasPrefix(meaning, prefix) {
			return true
		}
	}
	return false
}

// The screen's editor asks the reader, so typing in it offers what a recipe
// takes there, with the build's generators and the catalog's layers.
func TestTheRecipeEditorOffersWhatTheRecipeTakes(t *testing.T) {
	view := offeredView(t)
	for _, one := range []struct {
		text  string
		wants string
	}{
		{"entities:\n  c:\n    fields:\n      a: { generator: pes", "pesel"},
		{"policy:\n  layers: [L", "L3"},
		{"ver", "version"},
	} {
		got := view.suggest(one.text, utf8.RuneCountInString(one.text))
		var words []string
		for _, offer := range got.Offers {
			words = append(words, offer.Value)
		}
		if !slices.Contains(words, one.wants) {
			t.Errorf("after %q the list offered %q, without %s", one.text, words, one.wants)
		}
	}
}
