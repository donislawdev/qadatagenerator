package quickview

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// Every heading the panel shows is a sentence of the catalogue, never a code.
//
// # Why this closes the register from the side of the schemes
//
// A country reaches the screen as the two letters before the dot of a scheme,
// and the catalogue names it under country.XX. Nothing joins the two but this
// test: a scheme for a country with no entry would show its code as a heading,
// and the catalogue test cannot see it, because the key is built at run time
// rather than declared. So this walks every kind in every locale and asks the
// view for the heading it would draw.
func TestEveryGroupHeadingIsWordedInTheCatalogue(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	for _, locale := range quick.Locales() {
		for _, generator := range quick.Generators(locale) {
			about := quick.Describe(generator, locale)
			heading := view.groupName(about.Group)
			if heading == "" || strings.HasPrefix(heading, "country.") || strings.HasPrefix(heading, "quick.") {
				t.Errorf("%s in %s would stand under the heading %q, which is a key rather than words",
					generator, locale, heading)
			}
		}
	}
}

// The headings come in the order the comment on groupRank promises, and the
// country of the locale comes first.
func TestTheHeadingsStartWithTheCountryOfTheLocale(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	for locale, first := range map[string]string{"en-US": "United States", "pl-PL": "Poland"} {
		view.UseLocale(locale)
		choices := view.choices()
		if len(choices) == 0 {
			t.Fatalf("no kinds in %s", locale)
		}
		if choices[0].Group != first {
			t.Errorf("in %s the first heading is %q and the country of the locale is %q", locale, choices[0].Group, first)
		}
		lastRank := -1
		for _, choice := range choices {
			rank := rankOfHeading(view, choice.Group)
			if rank < lastRank {
				t.Errorf("in %s the heading %q comes after one that ranks later", locale, choice.Group)
			}
			lastRank = rank
		}
	}
}

// rankOfHeading reads a heading back into its rank, so that the order of the
// choices can be checked against the order groupRank promises.
func rankOfHeading(view *View, heading string) int {
	for _, locale := range quick.Locales() {
		for _, generator := range quick.Generators(locale) {
			about := quick.Describe(generator, locale)
			if view.groupName(about.Group) == heading {
				return groupRank(about.Group, countryOf(view.locale))
			}
		}
	}
	return -1
}

// Every kind shows its written form beside it, and a kind without one shows
// nothing rather than a word standing in for it.
func TestAKindWithAMaskShowsItBesideItsName(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	for _, choice := range view.choices() {
		about := quick.Describe(choice.Value, view.locale)
		if choice.Detail != about.Mask {
			t.Errorf("%s shows %q beside its name and its mask is %q", choice.Value, choice.Detail, about.Mask)
		}
	}
}
