package takeview

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/gui/damageparts"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// labelOfPerRow is how many defects one record may carry, said in words, by
// the names both screens ordering damage share (damageparts.PerRowLabel).
func (v *View) labelOfPerRow(value string) string { return damageparts.PerRowLabel(v.text, value) }

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

// labelOfLayer is one kind of damage named, with its code beside it. The names
// are shared with every screen that orders damage, so they live where all of
// them look (DEC-252).
func (v *View) labelOfLayer(code string) string {
	return damageparts.LayerLabel(v.text, code)
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
