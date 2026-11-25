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
			Note:   v.length(about.Masks),
			Group:  v.groupName(about.Group),
		})
	}
	return choices
}

// groupRank is the order of the headings: the country of the locale first,
// then the other countries, then what belongs to no country, then people,
// places and accounts.
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
	case quick.GroupAccounts:
		return rankAccounts
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
	rankAccounts
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
	case quick.GroupAccounts:
		return v.text.T(i18n.QuickGroupAccounts)
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

// length is how long a value written in these masks is, in words: eleven
// digits, twenty eight letters or digits, or 15 or 16 digits. Nothing for a
// kind with no mask, such as a name, whose length is not a fact about the kind.
//
// The count comes from quick.Places, so the mode reads the mask and this file
// only chooses words for what it read. The panel and the card beside the
// values say it with this one function, so the two cannot disagree.
//
// # Why forms of one kind share one noun
//
// Because "15 digits or 16 digits" says digits twice for no reason, and the
// payment card is the scheme that has two forms. Forms that differ in what
// their places hold are said one after another instead. No scheme here needs
// that, and it is written so that the first one to need it is not said wrong.
func (v *View) length(masks []string) string {
	if len(masks) == 0 {
		return ""
	}
	or := " " + v.text.T(i18n.QuickAboutOr) + " "
	var counts, phrases []string
	shared := true
	_, firstLetters := quick.Places(masks[0])
	for _, mask := range masks {
		places, letters := quick.Places(mask)
		shared = shared && letters == firstLetters
		count := strconv.Itoa(places)
		phrases = append(phrases, count+" "+v.text.T(heldIn(letters)))
		// Two forms of one length, such as one with separators and one
		// without, are one count rather than "11 or 11".
		if len(counts) == 0 || counts[len(counts)-1] != count {
			counts = append(counts, count)
		}
	}
	if shared {
		return strings.Join(counts, or) + " " + v.text.T(heldIn(firstLetters))
	}
	return strings.Join(phrases, or)
}

// heldIn is the noun for what the places of a mask may hold.
func heldIn(letters bool) i18n.Key {
	if letters {
		return i18n.QuickAboutLettersOrDigits
	}
	return i18n.QuickAboutDigits
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
