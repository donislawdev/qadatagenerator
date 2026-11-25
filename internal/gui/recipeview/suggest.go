package recipeview

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/gui/damageparts"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// What the list under the caret of the editor offers, and what each offer
// means beside it (DEC-338).
//
// The offers come from the reader of recipes, which is the one place that
// knows what a recipe takes where (recipeyaml.Suggest). This file adds the
// words a person reads beside each offer. A key gets a line of its own, and a
// value gets the name another screen already gives it where there is one: a
// generator is named as on the Quick value screen, a layer and the defects per
// row as on the screens that order damage, a locale by its country. A value
// that means only itself, such as true, and a name the recipe gave, stand
// alone.

// buildVocabulary is what this build carries that a recipe names: its
// generators and the types it fills. The layers of its catalog follow once the
// screen is handed the catalog (OfferLayersOf).
func buildVocabulary() recipeyaml.Vocabulary {
	source := values.New("", "")
	return recipeyaml.Vocabulary{Generators: source.Names(), Types: source.FilledTypes()}
}

// OfferLayersOf hands the list the layers a catalog describes damage in, for
// the policy of a recipe. It is the catalog the checks of this screen are made
// against, handed in the same way, so the list cannot offer a layer the check
// then refuses. A screen never handed one offers no layers, which is what a
// screen built for a picture of something else needs.
func (v *View) OfferLayersOf(described catalog.Catalog) {
	v.vocabulary.Layers = mutators.Coverage{Described: described}.Layers()
}

// suggest is the question the editor asks while somebody types: the reader's
// offers at the caret, each with its meaning beside it.
func (v *View) suggest(text string, caret int) parts.Completion {
	asked := recipeyaml.Suggest(text, caret, v.vocabulary)
	offers := make([]parts.Offer, 0, len(asked.Offers))
	for _, offer := range asked.Offers {
		offers = append(offers, parts.Offer{
			Choice: parts.Choice{Value: offer.Word, Detail: v.describe(offer)},
			Writes: offer.Write,
			More:   offer.Then,
		})
	}
	return parts.Completion{From: asked.From, To: asked.To, Offers: offers}
}

// describe is what an offer means, in words, or nothing for one that means
// only itself.
func (v *View) describe(offer recipeyaml.Offer) string {
	switch {
	case offer.Kind == recipeyaml.OfferValue && offer.Key == "layers":
		return damageparts.LayerName(v.text, offer.Word)
	case offer.Kind == recipeyaml.OfferValue && offer.Key == "per_row":
		return damageparts.PerRowLabel(v.text, offer.Word)
	}
	key, described := describedBy(offer)
	if !described {
		return ""
	}
	return v.text.T(key)
}

// valuesSaid are the keys whose values have a line of their own in the
// catalogue. The values of the others are words such as true, false and utf-8,
// which say what they mean.
var valuesSaid = []string{"type", "counted_in", "format", "dialect", "consistency"}

// describedBy is the key of the catalogue saying what an offer means, and
// false for an offer that means only itself or whose name comes from another
// screen's list above.
func describedBy(offer recipeyaml.Offer) (i18n.Key, bool) {
	switch {
	case offer.Kind == recipeyaml.OfferName:
		return "", false
	case offer.Kind == recipeyaml.OfferKey:
		return i18n.Key("recipe.key." + offer.Word), true
	case offer.Key == "version":
		return i18n.RecipeValueVersion, true
	case offer.Key == "generator":
		return i18n.Key("kind." + offer.Word), true
	case offer.Key == "locale":
		_, country, _ := strings.Cut(offer.Word, "-")
		return i18n.Key("country." + country), true
	}
	for _, said := range valuesSaid {
		if offer.Key == said {
			return i18n.Key("recipe.value." + offer.Key + "." + offer.Word), true
		}
	}
	return "", false
}
