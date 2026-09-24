// Package damageparts is what a screen says about damage it orders: what each
// kind of damage is called, which extra form of the manifest a run exports, how
// much of an order this build carries out, and what a finished run did with it.
//
// # Why a package of its own rather than a part of either screen
//
// Two screens order damage since 2026-09-24: the one that damages somebody
// else's file and the one that generates from a recipe with a policy (DEC-252).
// Everything here was written once, on the first of them, and the second needed
// every piece of it. A copy would be the drift the rule about changing a look in
// one place exists to prevent: the day the counts gained a line, one screen
// would say it and the other would not, and both would look finished.
//
// # What is deliberately not here
//
// Anything a screen owns: its settings, its card, its sentences about itself.
// This package says the same facts the same way on every screen that shows them
// and nothing else, the way internal/damage works them out the same way for
// every mode.
package damageparts

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// nameOfLayer is what each kind of damage is called on screen.
//
// # Why this exists at all
//
// Until 2026-09-08 the tick boxes were labelled L0, L1, L2 and so on, which is
// how the catalogue files them and tells a person opening this window nothing.
// The handbook says outright that an internal term never reaches a label, and
// docs/08 describes the take screen as a person choosing "field format" and
// "value meaning". The code and the document disagreed, and the code was the
// one that was wrong.
//
// # Why the code stays beside the name
//
// It is what --layers takes in a terminal, what a policy in a recipe names and
// what the manifest reports back, so dropping it would cut the window off from
// all three. A person who reads the manifest afterwards has to be able to match
// what they ordered to what it says.
//
// 🔴 This map is CLOSED against the catalogue by a test. A kind of damage with
// no name here would otherwise fall back to its bare code and look, on screen,
// exactly like the thing this map was written to remove.
var nameOfLayer = map[string]i18n.Key{
	"L0":  i18n.TakeLayerBytes,
	"L1":  i18n.TakeLayerStructure,
	"L2":  i18n.TakeLayerFormat,
	"L3":  i18n.TakeLayerMeaning,
	"L4":  i18n.TakeLayerBetween,
	"L5":  i18n.TakeLayerAcross,
	"UNI": i18n.TakeLayerUnicode,
	"REV": i18n.TakeLayerUnusual,
}

// LayerLabel is one kind of damage named, with its code beside it.
//
// A code with no name falls back to the bare code rather than to an empty
// label. It is the wrong label and it is still a label, which is the better of
// the two failures a person can be handed.
func LayerLabel(text i18n.Catalogue, code string) string {
	key, named := nameOfLayer[code]
	if !named {
		return code
	}
	return fmt.Sprintf(text.T(i18n.TakeLayerCode), text.T(key), code)
}

// FormChoices is every extra form of the manifest, with the absence of one named
// rather than left blank.
//
// A list whose first entry is empty asks a person to work out what empty means,
// and the answer here is a real option with a name of its own.
func FormChoices(text i18n.Catalogue) []string {
	return append([]string{text.T(i18n.TakeSettingNone)}, manifestexport.Forms()...)
}

// FormNamed turns what was picked back into what a run takes, where the name
// for no export is a word on screen and an empty setting underneath.
func FormNamed(text i18n.Catalogue, picked string) string {
	if picked == text.T(i18n.TakeSettingNone) {
		return ""
	}
	return picked
}

// NameWidth is the width of a column holding these kinds of damage by name, so
// that the figures beside them line up and can be compared down a card.
//
// A screen whose list of kinds changes under a person passes every kind it
// offers rather than the ones ordered, so that ticking one does not move the
// figures of the others.
//
// 🔴 Measured on the name as PlanRows draws it, in the heavier weight. It was
// measured by paint.OptionNameWidth, which measures the regular weight a picker
// draws its options in, so the column came out narrower than any heavy name in
// it. Nothing showed on the take screen, where the widest of all eight kinds
// hid the difference, and the figures of the recipe screen, which measures only
// the two kinds a policy names, stood three pixels apart on the first render.
func NameWidth(text i18n.Catalogue, codes []string) float32 {
	widest := float32(0)
	for _, code := range codes {
		widest = max(widest, planName(text, code).MinSize().Width)
	}
	return widest
}

// planName is one kind of damage as a row of the plan names it.
func planName(text i18n.Catalogue, code string) *canvas.Text {
	return paint.Strong(LayerLabel(text, code), palette.TypeBody, palette.TextPrimary)
}

// PlanRows says, for each kind of damage ordered, how much of it this build
// carries out: the name, and "3 of 9" in figures, because a figure is what
// somebody compares across rows.
//
// It is the same answer a dry run gives in a terminal, worked out by the same
// walk over the catalog, so the screens say it in the words of the window and
// with the same numbers.
func PlanRows(text i18n.Catalogue, planned damage.Planned, names float32) fyne.CanvasObject {
	rows := []fyne.CanvasObject{paint.Caption(text.T(i18n.TakePlanCarried))}
	for _, layer := range planned.ByLayer {
		rows = append(rows, paint.Row(palette.NameStepRelated,
			paint.AtLeast(names, 0, planName(text, layer.Layer)),
			paint.Mono(strconv.Itoa(layer.Implemented)+" "+text.T(i18n.TakePlanOf)+" "+
				strconv.Itoa(layer.Ordered), palette.TypeBody, palette.TextPrimary)))
	}
	return paint.Column(palette.NameStepTight, rows...)
}

// Counts says what a finished run did with the damage it was ordered.
//
// # Why every count and not the applied one
//
// A run where most of what was ordered went nowhere looks, from the applied
// count alone, like a run that barely did anything. That is the state docs/17
// calls a partial run, and it is the one this product exists to make visible,
// so it is marked rather than left to whoever compares two numbers.
//
// 🔴 A manifest this build will not read back is said FIRST, because it changes
// what the counts are worth: a manifest nobody can open on the next screen is a
// manifest that has to be read some other way.
//
// The counts are bare numbers beside their labels rather than phrases of their
// own, because the label already names what is being counted and the number is
// data rather than a sentence.
func Counts(text i18n.Catalogue, counts damage.Counts, oversized []damage.Oversize) []fyne.CanvasObject {
	var lines []fyne.CanvasObject
	if len(oversized) > 0 {
		names := make([]string, 0, len(oversized))
		for _, one := range oversized {
			names = append(names, one.Name)
		}
		lines = append(lines,
			pair(text, i18n.TakeResultOversized, listOf(names)),
			notice(text, i18n.TakeResultOversizedWhy))
	}
	lines = append(lines,
		pair(text, i18n.TakeResultOrdered, strconv.Itoa(counts.Requested)),
		pair(text, i18n.TakeResultApplied, strconv.Itoa(counts.Applied)))
	if missing := counts.Missing(); missing > 0 {
		lines = append(lines,
			pair(text, i18n.TakeResultMissing, strconv.Itoa(missing)),
			notice(text, i18n.TakeResultMissingWhy))
	}
	if counts.Unpartnered > 0 {
		lines = append(lines,
			pair(text, i18n.TakeResultUnpartnered, strconv.Itoa(counts.Unpartnered)),
			paint.Paragraph(text.T(i18n.TakeResultUnpartneredWhy)))
	}
	return lines
}

// listOf writes names the way a sentence does.
//
// A copy of the one each screen carries, and a known family rather than an
// oversight: it is not about damage, so it is not exported from here, and the
// place all of them belong in is a question of its own.
func listOf(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		joined := ""
		for at, name := range names[:len(names)-1] {
			if at > 0 {
				joined += ", "
			}
			joined += name
		}
		return joined + " and " + names[len(names)-1]
	}
}

// pair is a label and the value beside it. Values are data rather than
// interface text, so they are not keys.
func pair(text i18n.Catalogue, label i18n.Key, value string) fyne.CanvasObject {
	return paint.Row(palette.NameStepTight,
		paint.Strong(text.T(label), palette.TypeBody, palette.TextPrimary),
		paint.Text(value, palette.TypeBody, palette.TextSecondary))
}

// notice is a sentence with the mark of a warning before it.
func notice(text i18n.Catalogue, key i18n.Key) fyne.CanvasObject {
	return paint.Marked(paint.IconWarn, palette.StateWarning, paint.Paragraph(text.T(key)))
}
