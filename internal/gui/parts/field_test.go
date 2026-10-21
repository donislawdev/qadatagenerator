package parts

import (
	"fmt"
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
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

// textControl is a text control of this package, reached through the hooks a
// driver uses.
type textControl interface {
	fyne.Disableable
	fyne.Focusable
	desktop.Hoverable
}

// textControls are both text controls, each with the way it is dressed. The
// dressing is handed back rather than done, so that a test can disable a
// control before a screen has put it on itself.
func textControls() []struct {
	name  string
	build func() (textControl, func() fyne.CanvasObject)
} {
	return []struct {
		name  string
		build func() (textControl, func() fyne.CanvasObject)
	}{
		{"the field", func() (textControl, func() fyne.CanvasObject) {
			field := NewFieldEntry()
			return field, func() fyne.CanvasObject { return Field(field) }
		}},
		{"the recipe editor", func() (textControl, func() fyne.CanvasObject) {
			editor := NewEditor(func(string) {})
			return editor, editor.Framed
		}},
	}
}

// 🔴 A text control out of reach looks like a chooser out of reach, whoever is
// on it, and comes back the way the pointer and the keyboard left it.
//
// DEC-253. A field kept the edge that says it can be used through the whole of
// a run on the take screen, because the toolkit greys the words and the ground
// is drawn beside it by this package, which nothing had told. It is compared
// with the chooser rather than with a colour, because what was decided is one
// family: a colour right on its own would stay green while the two drifted.
func TestATextControlOutOfReachLooksLikeAChooserOutOfReach(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
	chooser.Disable()
	want := groundsIn(chooserFace(chooser))
	if want == "[]" {
		t.Fatal("a disabled chooser shows no ground, so the rest of this compares with nothing")
	}

	for _, each := range textControls() {
		t.Run(each.name, func(t *testing.T) {
			control, dress := each.build()
			dressed := dress()
			resting := shapesShownIn(dressed)

			control.Disable()
			off := shapesShownIn(dressed)
			if got := groundsIn(dressed); got != want {
				t.Errorf("%s out of reach shows %s, where the chooser beside it shows %s",
					each.name, got, want)
			}

			control.MouseIn(&desktop.MouseEvent{})
			control.FocusGained()
			if touched := shapesShownIn(dressed); touched != off {
				t.Errorf("%s answered the pointer and the keyboard while out of reach:\n"+
					"  disabled %s\n  touched  %s", each.name, off, touched)
			}

			// Given back with both still on it, which is where a person's hand
			// is when a run ends under it.
			control.Enable()
			if marks := focusEdgesIn(dressed); marks != 1 {
				t.Errorf("%s came back with the keyboard in it and draws %d rings, want 1",
					each.name, marks)
			}
			if !showsFill(dressed, palette.Ink(palette.BgHover)) {
				t.Errorf("%s came back with the pointer on it and not on the ground of the pointer",
					each.name)
			}

			control.MouseOut()
			control.FocusLost()
			if back := shapesShownIn(dressed); back != resting {
				t.Errorf("%s is not as it was before any of this:\n  before %s\n  after  %s",
					each.name, resting, back)
			}
		})
	}
}

// 🔴 A text control with the keyboard in it shows where the next letter goes,
// and draws no edge of the toolkit's own in any state.
//
// The owner could not see where they were typing on 2026-10-21: the caret of
// widget.Entry is as wide as the edge it strokes round itself, and every field
// here gave that edge no width. The edge is now put away instead, so the same
// walk asks both halves: the caret has width, and nothing the toolkit strokes
// is shown with the keyboard in the field or with the field out of reach.
func TestATextControlShowsItsCaretAndNoEdgeOfTheToolkit(t *testing.T) {
	for _, each := range textControls() {
		t.Run(each.name, func(t *testing.T) {
			control, dress := each.build()
			window := test.NewWindow(dress())
			t.Cleanup(window.Close)
			window.Canvas().Focus(control)
			drawn := control.(fyne.Widget)

			if got := widestCaretIn(drawn); got < palette.CaretThickness {
				t.Errorf("%s with the keyboard in it draws its caret %v wide, want %v",
					each.name, got, palette.CaretThickness)
			}
			if edges := toolkitEdgesIn(drawn); edges != 0 {
				t.Errorf("%s with the keyboard in it shows %d edges of the toolkit's own "+
					"inside the window's ring", each.name, edges)
			}
			control.Disable()
			if edges := toolkitEdgesIn(drawn); edges != 0 {
				t.Errorf("%s out of reach shows %d edges of the toolkit's own", each.name, edges)
			}
		})
	}
}

// widestCaretIn is the widest visible bar, taller than it is wide and filled
// with something, inside a control's own drawing. That is what a caret is: the
// ground the toolkit lays under the words is there too, in any shape a small
// window gives it, but transparent, since the window draws its own.
func widestCaretIn(control fyne.CanvasObject) float32 {
	widest := float32(0)
	walkDrawing(control, func(shape *canvas.Rectangle) {
		size := shape.Size()
		if shape.FillColor == nil || shape.StrokeWidth > 0 || size.Height <= size.Width {
			return
		}
		if _, _, _, alpha := shape.FillColor.RGBA(); alpha > 0 && size.Width > widest {
			widest = size.Width
		}
	})
	return widest
}

// toolkitEdgesIn counts the visible shapes with a stroke inside a control's own
// drawing, which is where the toolkit keeps its edge and the window keeps none.
func toolkitEdgesIn(control fyne.CanvasObject) int {
	found := 0
	walkDrawing(control, func(shape *canvas.Rectangle) {
		if shape.StrokeWidth > 0 {
			found++
		}
	})
	return found
}

// walkDrawing visits every visible rectangle a control draws, through the
// renderers of the widgets inside it.
func walkDrawing(object fyne.CanvasObject, visit func(*canvas.Rectangle)) {
	if object == nil || !object.Visible() {
		return
	}
	switch shape := object.(type) {
	case *canvas.Rectangle:
		visit(shape)
	case *fyne.Container:
		for _, child := range shape.Objects {
			walkDrawing(child, visit)
		}
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(shape).Objects() {
			walkDrawing(child, visit)
		}
	}
}

// A control disabled before a screen dressed it is dressed out of reach. The
// dressing is made once, when a screen first draws, and a screen may disable a
// control before that.
func TestATextControlDisabledBeforeItIsDressedIsDressedOutOfReach(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
	chooser.Disable()
	want := groundsIn(chooserFace(chooser))

	for _, each := range textControls() {
		t.Run(each.name, func(t *testing.T) {
			control, dress := each.build()
			control.Disable()
			if got := groundsIn(dress()); got != want {
				t.Errorf("%s disabled before it was dressed shows %s, where a chooser out of reach shows %s",
					each.name, got, want)
			}
		})
	}
}

// groundsIn is every visible shape in a drawing that fills something, with what
// decides how it looks: the fill, the edge and the corners. A shape filling
// nothing is left out, since the face of a chooser holds one that keeps its
// size and nobody sees it.
func groundsIn(object fyne.CanvasObject) string {
	var seen []string
	var walk func(fyne.CanvasObject)
	walk = func(each fyne.CanvasObject) {
		if each == nil || !each.Visible() {
			return
		}
		switch shape := each.(type) {
		case *canvas.Rectangle:
			if shape.FillColor == nil {
				return
			}
			if _, _, _, alpha := shape.FillColor.RGBA(); alpha == 0 {
				return
			}
			seen = append(seen, fmt.Sprintf("fill=%v edge=%v width=%v corners=%v",
				shape.FillColor, shape.StrokeColor, shape.StrokeWidth, shape.CornerRadius))
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		}
	}
	walk(object)
	return fmt.Sprint(seen)
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
