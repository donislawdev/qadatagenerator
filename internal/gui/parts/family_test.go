package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// 🔴 THE GUARD FOR ONE PROGRAM LOOKING LIKE ONE PROGRAM.
//
// The owner wrote it on 2026-09-09, after three controls disagreed in an hour:
// "one control looks like x, the next one y, the third z - do not do that, the
// GUI has to be consistent, it is one program".
//
// Every control a person can reach with the keyboard says so with the same
// mark, and this walks all of them rather than trusting each one's own test. A
// test per control is exactly what let the field drift: it passed on its own
// terms while standing in a row next to a chooser that did something else.
//
// # What it asks, and why that and not the colour
//
// The DRAWING. The two wrong answers before this guard existed were both a
// reasonable colour - an edge four pixels thick, then a thin edge in the accent
// - and neither was the ring the rest of the window draws. A guard asking
// "does this control mark focus somehow" would have been green through both.
//
// # What it cannot see
//
// A control this list does not name. Nothing in the code says how many
// focusable controls this package has, so the list below is the claim, and a
// control added without a line here is the one failure this cannot catch.
func TestEveryControlMarksFocusWithTheSameRing(t *testing.T) {
	kinds := []struct {
		name  string
		build func() (focus func(), drawn fyne.CanvasObject)
	}{
		{"button", func() (func(), fyne.CanvasObject) {
			control := NewButton("Copy", func() {})
			return control.FocusGained, control
		}},
		{"filled button", func() (func(), fyne.CanvasObject) {
			control := NewPrimaryButton("Generate", func() {})
			return control.FocusGained, control
		}},
		{"chooser", func() (func(), fyne.CanvasObject) {
			control := NewSelect([]string{"one", "two"}, func(string) {})
			return control.FocusGained, control
		}},
		{"field", func() (func(), fyne.CanvasObject) {
			control := NewFieldEntry()
			dressed := Field(control)
			return control.FocusGained, dressed
		}},
	}

	for _, kind := range kinds {
		t.Run(kind.name, func(t *testing.T) {
			focus, drawn := kind.build()

			if marks := focusEdgesIn(drawn); marks != 0 {
				t.Fatalf("a %s nobody is on already draws %d focus edges", kind.name, marks)
			}
			focus()
			if marks := focusEdgesIn(drawn); marks != 1 {
				t.Errorf("a focused %s draws %d edges in mark.focus, want exactly 1",
					kind.name, marks)
			}
		})
	}
}

// focusEdgesIn counts the focus marks a control is actually drawing.
//
// A mark is a stroked shape in mark.focus, and it has to be VISIBLE: a control
// holding a hidden one is a control drawing nothing, which is the state a
// person sees.
func focusEdgesIn(object fyne.CanvasObject) int {
	found := 0
	var walk func(fyne.CanvasObject)
	walk = func(each fyne.CanvasObject) {
		if each == nil || !each.Visible() {
			return
		}
		switch shape := each.(type) {
		case *canvas.Rectangle:
			if shape.StrokeWidth > 0 && sameInk(shape.StrokeColor, palette.Ink(palette.MarkFocus)) {
				found++
			}
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		case fyne.Widget:
			// Through the renderer, because a control that draws itself keeps
			// its shapes behind one and a walk over containers alone would
			// report that nothing is drawn.
			for _, child := range test.WidgetRenderer(shape).Objects() {
				walk(child)
			}
		}
	}
	walk(object)
	return found
}
