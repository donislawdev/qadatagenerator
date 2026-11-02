package quickview

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
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

// Every kind the panel offers carries the name a person calls it, never a key.
//
// The same reason as the headings above: the key is built from the generator
// when the panel is drawn, so the catalogue's own test cannot see a generator
// that arrived without a name. This walks every kind in every locale.
func TestEveryKindIsNamedInTheCatalogue(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	for _, locale := range quick.Locales() {
		view.UseLocale(locale)
		for _, choice := range view.choices() {
			if choice.Detail == "" || strings.HasPrefix(choice.Detail, "kind.") {
				t.Errorf("%s in %s would stand beside %q, which is a key rather than its name",
					choice.Value, locale, choice.Detail)
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

// Every kind with a mask says at its end how long a value of it is, in the
// words the card beside the values uses, and a kind without one says nothing
// rather than a word standing in for it. Until 2026-10-21 the mask itself stood
// beside the name, a row of noughts the owner found told nobody anything.
func TestAKindWithAMaskSaysHowLongItIs(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	for _, choice := range view.choices() {
		about := quick.Describe(choice.Value, view.locale)
		if want := view.length(about.Masks); choice.Note != want {
			t.Errorf("%s says %q at its end, where its masks %q are %q", choice.Value, choice.Note, about.Masks, want)
		}
		if len(about.Masks) > 0 && choice.Note == "" {
			t.Errorf("%s has the masks %q and says nothing about its length", choice.Value, about.Masks)
		}
	}
}

// A scheme with more than one form says every length once and the noun once,
// and forms holding different things say each with its own noun.
//
// The masks are built from the constants rather than typed, for the reason the
// guard over the mask reader gives: a case written in an old mask character and
// still passing would be a guard over a language nothing else speaks.
func TestAKindWithSeveralFormsSaysEachLengthOnce(t *testing.T) {
	view := New(i18n.English, &heldClipboard{})
	digits := func(n int) string { return strings.Repeat(string(provider.MaskDigit), n) }
	either := func(n int) string { return strings.Repeat(string(provider.MaskAlnum), n) }
	for _, each := range []struct {
		why   string
		masks []string
		said  string
	}{
		{"one form", []string{digits(11)}, "11 digits"},
		{"two lengths of digits", []string{digits(15), digits(16)}, "15 or 16 digits"},
		{"one length written two ways", []string{digits(4) + "-" + digits(4), digits(8)}, "8 digits"},
		{"forms holding different things", []string{digits(9), either(17)},
			"9 digits or 17 letters or digits"},
		{"no form at all", nil, ""},
	} {
		if got := view.length(each.masks); got != each.said {
			t.Errorf("%s: %q reads %q, where it has to read %q", each.why, each.masks, got, each.said)
		}
	}
}
