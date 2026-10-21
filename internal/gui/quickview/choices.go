package quickview

import (
	"sort"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// What the list of kinds offers, apart from the screen that shows it.
//
// The kinds are grouped under headings and each carries the written form of a
// value beside it, so that a person choosing among dozens has something to
// recognise besides a name (DEC-244). Every fact here is read off the mode:
// the group from quick.Describe, the written form from the mask a scheme
// declares. This file turns the facts into words and an order, which is the
// half that belongs to a surface.

// choices are the kinds of this locale, grouped and ordered for the panel.
func (v *View) choices() []parts.Choice {
	described := make([]quick.Description, 0, len(v.generators))
	for _, generator := range v.generators {
		described = append(described, quick.Describe(generator, v.locale))
	}
	home := countryOf(v.locale)
	sort.SliceStable(described, func(i, j int) bool {
		left, right := groupRank(described[i].Group, home), groupRank(described[j].Group, home)
		if left != right {
			return left < right
		}
		if described[i].Group.Country != described[j].Group.Country {
			return v.groupName(described[i].Group) < v.groupName(described[j].Group)
		}
		return described[i].Generator < described[j].Generator
	})
	choices := make([]parts.Choice, 0, len(described))
	for _, about := range described {
		choices = append(choices, parts.Choice{
			Value:  about.Generator,
			Detail: v.text.T(kindKey(about.Generator)),
			Note:   v.length(about.Mask),
			Group:  v.groupName(about.Group),
		})
	}
	return choices
}

// groupRank is the order of the headings: the country of the locale first,
// then the other countries, then what belongs to no country, then people and
// places.
//
// # Why the locale's country comes first
//
// Because the locale is the one thing on this screen that was chosen before
// the kind, and a person who set it to Polish is more likely after a PESEL than
// after an SSN. It is an order, not a filter: the other countries stay a
// scroll away.
func groupRank(group quick.Group, home string) int {
	switch group.Kind {
	case quick.GroupCountry:
		if group.Country == home {
			return rankHome
		}
		return rankOtherCountry
	case quick.GroupInternational:
		return rankInternational
	case quick.GroupPeople:
		return rankPeople
	case quick.GroupPlaces:
		return rankPlaces
	}
	return rankUnknown
}

// The order of the headings, first to last.
const (
	rankHome = iota
	rankOtherCountry
	rankInternational
	rankPeople
	rankPlaces
	rankUnknown
)

// countryOf is the country a locale names, which is the part after its dash.
func countryOf(locale string) string {
	if dash := strings.LastIndex(locale, "-"); dash >= 0 {
		return locale[dash+1:]
	}
	return ""
}

// groupName is the heading a group is read under.
//
// A country is looked up by its code, and the guard beside this file holds
// that every country a scheme carries has a name in the catalogue, so the code
// itself never reaches the screen.
func (v *View) groupName(group quick.Group) string {
	switch group.Kind {
	case quick.GroupCountry:
		return v.text.T(countryKey(group.Country))
	case quick.GroupInternational:
		return v.text.T(i18n.QuickGroupInternational)
	case quick.GroupPeople:
		return v.text.T(i18n.QuickGroupPeople)
	case quick.GroupPlaces:
		return v.text.T(i18n.QuickGroupPlaces)
	}
	return ""
}

// countryKey is the catalogue key naming a country, from its alpha-2 code.
func countryKey(code string) i18n.Key {
	return i18n.Key("country." + code)
}

// kindKey is the key of the catalogue naming a kind the way a person says it,
// built the way countryKey is, with the same guard beside it.
func kindKey(generator string) i18n.Key {
	return i18n.Key("kind." + generator)
}

// length is how long a value written in this mask is, in words: eleven digits,
// or twenty eight letters or digits. Nothing for a kind with no mask, such as
// a name, whose length is not a fact about the kind.
//
// The count comes from quick.Places, so the mode reads the mask and this file
// only chooses words for what it read. The panel and the card beside the
// values say it with this one function, so the two cannot disagree.
func (v *View) length(mask string) string {
	if mask == "" {
		return ""
	}
	places, letters := quick.Places(mask)
	said := i18n.QuickAboutDigits
	if letters {
		said = i18n.QuickAboutLettersOrDigits
	}
	return strconv.Itoa(places) + " " + v.text.T(said)
}

// panelWords are what the panel says, in the language of this screen.
func (v *View) panelWords() parts.PanelWords {
	return parts.PanelWords{
		Prompt:  v.text.T(i18n.QuickPickPrompt),
		Counted: v.text.T(i18n.QuickPickKinds),
		Matches: v.text.T(i18n.QuickPickMatches),
		Enter:   v.text.T(i18n.QuickPickEnter),
		Nothing: v.text.T(i18n.QuickPickNothing),
	}
}
