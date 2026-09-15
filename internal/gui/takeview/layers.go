package takeview

import (
	"fmt"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// nameOfLayer is what each kind of damage is called on screen.
//
// # Why this exists at all
//
// Until 2026-09-08 the tick boxes were labelled L0, L1, L2 and so on, which is
// how the catalogue files them and tells a person opening this window nothing.
// The handbook says outright that an internal term never reaches a label, and
// docs/08 describes this very screen as a person choosing "field format" and
// "value meaning". The code and the document disagreed, and the code was the
// one that was wrong.
//
// # Why the code stays beside the name
//
// It is what --layers takes in a terminal and what the manifest reports back,
// so dropping it would cut the window off from both. A person who reads the
// manifest afterwards has to be able to match what they ticked to what it
// says.
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

// nameOfPerRow is how many defects one record may carry, said in words.
//
// Same reason as the map above and the same closing test: the engine takes
// exactly_one and any, and a list offering those two spellings asks somebody
// to read a settings file in order to use a window.
var nameOfPerRow = map[string]i18n.Key{
	"exactly_one": i18n.TakePerRowExactlyOne,
	"any":         i18n.TakePerRowAny,
}

// labelOfPerRow is one such setting named, or the bare value if nothing names
// it, for the reason given at labelOfLayer.
func (v *View) labelOfPerRow(value string) string {
	key, named := nameOfPerRow[value]
	if !named {
		return value
	}
	return v.text.T(key)
}

// perRowLabels is every such setting said the way a person reads it.
func (v *View) perRowLabels() []string {
	values := take.PerRowChoices()
	labels := make([]string, 0, len(values))
	for _, value := range values {
		labels = append(labels, v.labelOfPerRow(value))
	}
	return labels
}

// valueOfPerRowLabel turns what is picked back into what the engine is asked
// for.
func (v *View) valueOfPerRowLabel(label string) string {
	for _, value := range take.PerRowChoices() {
		if v.labelOfPerRow(value) == label {
			return value
		}
	}
	return label
}

// layerCodes is every kind of damage this build can apply to a file, in the
// order the catalogue gives them.
func (v *View) layerCodes() []string { return take.LayersIn(v.described) }

// layerLabels is the same list said the way a person reads it.
func (v *View) layerLabels() []string {
	codes := v.layerCodes()
	labels := make([]string, 0, len(codes))
	for _, code := range codes {
		labels = append(labels, v.labelOfLayer(code))
	}
	return labels
}

// labelOfLayer is one kind of damage named, with its code beside it.
//
// A code with no name falls back to the bare code rather than to an empty tick
// box. It is the wrong label and it is still a label, which is the better of
// the two failures a person can be handed.
func (v *View) labelOfLayer(code string) string {
	key, named := nameOfLayer[code]
	if !named {
		return code
	}
	return fmt.Sprintf(v.text.T(i18n.TakeLayerCode), v.text.T(key), code)
}

// kindBox is one kind of damage a person can tick, with the code the engine
// is asked for beside the control that asks.
type kindBox struct {
	code string
	box  *parts.Check
}

// kindBoxes is every kind this build offers, in the order the catalogue gives
// them. It is a list of data rendered in a loop rather than eight controls
// written out, which is what rule 8 of the GUI rules asks of anything that
// repeats, and it answers as one control to the code that freezes a run's
// settings.
//
// 🔴 It replaced widget.CheckGroup on 2026-09-15 (DEC-245). The group drew the
// toolkit's boxes, marked focus with a filled circle and stood in a column
// beside fields and choosers that mark focus with an edge, so the column spoke
// two languages of states.
type kindBoxes []kindBox

// Asserted, because settingControls hands this to the code that freezes and
// thaws a run's settings, and a list that quietly stopped answering would leave
// eight boxes usable while a run walks the file.
var _ fyne.Disableable = kindBoxes(nil)

func (k kindBoxes) Enable() {
	for _, kind := range k {
		kind.box.Enable()
	}
}

func (k kindBoxes) Disable() {
	for _, kind := range k {
		kind.box.Disable()
	}
}

// Disabled reports whether the boxes are out of reach, which they all are or
// none is.
func (k kindBoxes) Disabled() bool {
	for _, kind := range k {
		if !kind.box.Disabled() {
			return false
		}
	}
	return len(k) > 0
}

// chosen is every ticked kind, as the engine is asked for it, in the order the
// catalogue gives the kinds.
func (k kindBoxes) chosen() []string {
	codes := []string{}
	for _, kind := range k {
		if kind.box.Checked {
			codes = append(codes, kind.code)
		}
	}
	return codes
}

// tick puts exactly these kinds in, and takes every other out, the way a
// person clicking would, so that the callback runs for each change.
func (k kindBoxes) tick(codes []string) {
	wanted := make(map[string]bool, len(codes))
	for _, code := range codes {
		wanted[code] = true
	}
	for _, kind := range k {
		kind.box.SetChecked(wanted[kind.code])
	}
}

// controls is the boxes as things a layout is handed, in the same order.
func (k kindBoxes) controls() []fyne.CanvasObject {
	drawn := make([]fyne.CanvasObject, 0, len(k))
	for _, kind := range k {
		drawn = append(drawn, kind.box.OnScreen())
	}
	return drawn
}
