package parts

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// 🔴 The words of a field start where the words of a chooser start.
//
// That was the owner's first report about this control and it is the reason the
// space around the field exists. The renderer adds its own inner padding to
// whatever room it is given, so the guard measures the SUM rather than either
// half: what a person sees is where the text begins, not which of the two
// numbers put it there.
func TestAFieldStartsItsWordsWhereAChooserDoes(t *testing.T) {
	window := fyne.CurrentApp().Settings().Theme()
	around := palette.StepOf(palette.NameStepTight)
	fromRenderer := (bareField{}).Size(toolkit.SizeNameInnerPadding)
	if got, want := around+fromRenderer, palette.StepOf(palette.NameStepRelated); got != want {
		t.Errorf("the words of a field start %v in, where a chooser starts its own at %v", got, want)
	}
	// And the inner padding is still the window's, so this is space put around
	// the field rather than a second palette living inside it.
	if fromRenderer != window.Size(toolkit.SizeNameInnerPadding) {
		t.Error("a field answers the inner padding itself, which would be a value the window does not know about")
	}
}

// 🔴 A field dressed twice is one dressing, and the marks stay on the object a
// screen put on itself.
//
// The Quick screen dressed its count field on every draw and mounted only the
// first dressing, so the keyboard arrived at grounds that were never on the
// screen: nought pixels on the real window (B-279). The screen builds nothing
// before it decides now, and this is the half that holds anywhere else.
func TestAFieldDressedTwiceKeepsTheMarksOnTheFirstDressing(t *testing.T) {
	field := NewFieldEntry()
	first := Field(field)
	if second := Field(field); second != first {
		t.Fatal("dressing a field twice made two objects, and only one of them can be on a screen")
	}
	field.FocusGained()
	if marks := focusEdgesIn(first); marks != 1 {
		t.Errorf("the keyboard is on the field and the dressing a screen holds draws %d marks, want 1",
			marks)
	}
}

// 🔴 A field answers the pointer, and it answers it the way a quiet button
// does, because the two stand in one row.
//
// The toolkit draws nothing for a pointer on a field, measured on the real
// window as nought pixels against thousands for each control beside it
// (B-282), so the answer here is a decision and the decision is the button's:
// the same ground under the pointer, and the ring of focus drawn on it when
// both arrive.
func TestAFieldAnswersThePointerTheWayAQuietButtonDoes(t *testing.T) {
	field := NewFieldEntry()
	dressed := Field(field)
	resting := shapesShownIn(dressed)
	underPointer := palette.Ink(palette.BgHover)

	field.MouseIn(&desktop.MouseEvent{})
	if hovered := shapesShownIn(dressed); hovered == resting {
		t.Errorf("the pointer is on the field and it draws exactly as at rest:\n  %s", resting)
	}
	if !showsFill(dressed, underPointer) {
		t.Error("the pointer is on the field and the ground under it is not the one a quiet button stands on")
	}

	// Both at once: the ring of the keyboard on the ground of the pointer,
	// which is the state a person is in for the whole of a click.
	field.FocusGained()
	if marks := focusEdgesIn(dressed); marks != 1 {
		t.Errorf("keyboard and pointer together draw %d rings of focus, want 1", marks)
	}
	if !showsFill(dressed, underPointer) {
		t.Error("the keyboard arrived and the ground of the pointer went, though the pointer is still there")
	}

	field.MouseOut()
	if marks := focusEdgesIn(dressed); marks != 1 {
		t.Errorf("the pointer left and the ring of the keyboard went with it, %d rings", marks)
	}
	if showsFill(dressed, underPointer) {
		t.Error("the pointer left and its ground stayed")
	}

	field.FocusLost()
	if back := shapesShownIn(dressed); back != resting {
		t.Errorf("both left and the field is not as it was:\n  before %s\n  after  %s", resting, back)
	}
}

// shapesShownIn is every shape a person would see in a dressing, in drawing
// order, with the colours it is drawn in. Hidden grounds are skipped, because a
// hidden ground is one nobody sees, and that is the whole of what the field's
// states are made of.
func shapesShownIn(object fyne.CanvasObject) string {
	var seen []string
	var walk func(fyne.CanvasObject)
	walk = func(each fyne.CanvasObject) {
		if each == nil || !each.Visible() {
			return
		}
		switch shape := each.(type) {
		case *canvas.Rectangle:
			seen = append(seen, fmt.Sprintf("rect fill=%v stroke=%v", shape.FillColor, shape.StrokeColor))
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		}
	}
	walk(object)
	return fmt.Sprint(seen)
}

// showsFill reports whether a visible shape in the dressing is filled with
// that colour.
func showsFill(object fyne.CanvasObject, ink color.Color) bool {
	return strings.Contains(shapesShownIn(object), fmt.Sprintf("fill=%v", ink))
}

func sameInk(first, second color.Color) bool {
	fr, fg, fb, fa := first.RGBA()
	sr, sg, sb, sa := second.RGBA()
	return fr == sr && fg == sg && fb == sb && fa == sa
}
