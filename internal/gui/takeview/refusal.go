package takeview

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// refusalWords is what this screen says about one refusal of the mode, by key.
//
// Keys rather than sentences, because a sentence in a view is invisible to
// whoever adds a language, and this screen is the translated surface.
type refusalWords struct {
	// what is the opening line, which names the setting that is wrong.
	what i18n.Key
	// why is the reason.
	why i18n.Key
	// allowed is what would have been taken, for the refusals whose answer is a
	// sentence. It is empty for the ones whose answer is a list, and the list
	// then comes from the refusal itself, because only the catalog knows it.
	allowed i18n.Key
	// instead is what to do about it, in terms of this screen rather than of a
	// command. That is the part that cannot be shared with the other surface:
	// there the answer is a command to type, here it is a control to click.
	instead i18n.Key
}

// refusalSays is every refusal the mode can produce, in the words this screen
// uses for it.
//
// # Why all seven, when this screen can only produce three
//
// The kinds of damage are tick boxes, the manifest forms are a list and the
// setting for defects per record is a list, so nothing a person clicks here can
// name a value that does not exist. Four of these refusals are therefore
// unreachable from this screen today.
//
// They are written anyway, and the reason is the day somebody swaps a list for
// a text field. On that day the refusal becomes reachable, and a screen that
// had skipped it would answer with silence rather than with words. The guard
// holding this register against the list of kinds cannot tell those two
// situations apart, and it should not have to.
var refusalSays = map[take.RefusalKind]refusalWords{
	take.NoLayers: {
		what: i18n.TakeRefusalNoLayers, why: i18n.TakeRefusalNoLayersWhy,
		instead: i18n.TakeRefusalNoLayersDo,
	},
	take.UnknownLayer: {
		what: i18n.TakeRefusalUnknownLayer, why: i18n.TakeRefusalUnknownLayerWhy,
		instead: i18n.TakeRefusalUnknownLayerDo,
	},
	take.UnreadableDelimiter: {
		what: i18n.TakeRefusalDelimiter, why: i18n.TakeRefusalDelimiterWhy,
		allowed: i18n.TakeRefusalDelimiterAllowed, instead: i18n.TakeRefusalDelimiterDo,
	},
	take.UnknownManifestForm: {
		what: i18n.TakeRefusalManifestForm, why: i18n.TakeRefusalManifestFormWhy,
		instead: i18n.TakeRefusalManifestFormDo,
	},
	take.UnreadableShare: {
		what: i18n.TakeRefusalShare, why: i18n.TakeRefusalShareWhy,
		allowed: i18n.TakeRefusalShareAllowed, instead: i18n.TakeRefusalShareDo,
	},
	take.UnknownPerRow: {
		what: i18n.TakeRefusalPerRow, why: i18n.TakeRefusalPerRowWhy,
		instead: i18n.TakeRefusalPerRowDo,
	},
	take.WouldOverwriteSource: {
		what: i18n.TakeRefusalOverwriteSource, why: i18n.TakeRefusalOverwriteSourceWhy,
		allowed: i18n.TakeRefusalOverwriteSourceAllowed,
		instead: i18n.TakeRefusalOverwriteSourceDo,
	},
}

// refusal draws a refusal as the four parts a person needs to act on it.
//
// # Why a shape as well as the words
//
// The palette of this project was measured, and its meaningful colours differ
// in hue while barely differing in lightness: somebody who does not separate
// hues sees one shade of grey. So a refusal is marked by an icon beside it and
// by naming the setting in its first line, and the colour is left to do nothing
// on its own.
func (v *View) refusal(refused take.Refusal) fyne.CanvasObject {
	words, known := refusalSays[refused.Kind]
	if !known {
		// Unreachable while the guard beside this holds. It says so rather than
		// describing this as some other refusal, because a message that names
		// what it does not know is recoverable and one that confidently names
		// the wrong cause sends somebody to fix what was never broken.
		words = refusalWords{
			what: i18n.TakeRefusalUnnamed, why: i18n.TakeRefusalUnnamedWhy,
			instead: i18n.TakeRefusalUnnamedDo,
		}
	}

	headline := widget.NewLabel(v.text.T(words.what))
	headline.TextStyle = fyne.TextStyle{Bold: true}
	headline.Wrapping = fyne.TextWrapWord
	lines := []fyne.CanvasObject{
		beside(theme.WarningIcon(), headline),
		wrapped(v.text.T(words.why)),
	}
	if allowed := v.allowedIn(refused, words); allowed != "" {
		lines = append(lines, wrapped(v.text.T(i18n.TakeRefusalAllowed)+": "+allowed))
	}
	lines = append(lines, wrapped(v.text.T(i18n.TakeRefusalInstead)+": "+
		v.text.T(words.instead)))
	return container.NewVBox(lines...)
}

// allowedIn is what would have been taken, from whichever of the two places
// knows it.
//
// A refusal whose answer is a set carries that set, because only the catalog
// can say which kinds of damage this build describes. A refusal whose answer is
// a sentence has it here. Neither is a fallback for the other: they answer for
// different refusals, and a refusal answering with neither is a refusal that
// sends its reader to the source code.
func (v *View) allowedIn(refused take.Refusal, words refusalWords) string {
	if len(refused.Allowed) > 0 {
		return strings.Join(refused.Allowed, ", ")
	}
	if words.allowed == "" {
		return ""
	}
	return v.text.T(words.allowed)
}
