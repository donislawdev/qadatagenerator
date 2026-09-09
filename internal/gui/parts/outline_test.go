package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestTheEdgeIsDrawnInsideTheControlItOutlines holds the one piece of
// arithmetic this file has, which is where the stroke lands.
//
// # Why this is a test rather than a sentence
//
// The project has paid for the difference once already. A border promised 3.29
// and arrived as 2.12, because the toolkit centres a stroke on the edge of its
// rectangle and at a width of one every pixel of it is a half blend with what
// is behind. The answer both times is to put the whole stroke on whole pixels,
// and this is what checks that it still is.
//
// It lays the real arrangement out rather than calling a layout on its own, so
// that nesting the two containers differently is caught here as well.
func TestTheEdgeIsDrawnInsideTheControlItOutlines(t *testing.T) {
	control := canvas.NewRectangle(nil)
	group, edge := Outlined(control)
	size := fyne.NewSize(220, 28)

	group.Resize(size)

	if control.Size() != size {
		t.Errorf("the control was laid out at %v rather than at the %v it was given",
			control.Size(), size)
	}
	wantAt := fyne.NewPos(palette.OutlineInset, palette.OutlineInset)
	if edge.Position() != wantAt {
		t.Errorf("the edge sits at %v, where half of its own stroke is %v. "+
			"Anywhere else and half the line falls outside the control",
			edge.Position(), wantAt)
	}
	wantSize := fyne.NewSize(size.Width-palette.BorderThickness, size.Height-palette.BorderThickness)
	if edge.Size() != wantSize {
		t.Errorf("the edge measures %v rather than %v, so the far side of the stroke "+
			"lands somewhere other than the near side does", edge.Size(), wantSize)
	}
	if edge.StrokeWidth != palette.BorderThickness {
		t.Errorf("the edge is %v thick where the palette says %v",
			edge.StrokeWidth, palette.BorderThickness)
	}
}

// TestTheEdgeAsksForNothingOfItsOwn holds that outlining a control does not
// move it.
//
// An edge that added to the minimum size would push every button two pixels
// wider than the toolkit meant it to be, and would do it to forty stored
// screens at once, which reads in the diff as a change somebody made on purpose.
func TestTheEdgeAsksForNothingOfItsOwn(t *testing.T) {
	button := widget.NewButton("Damage this file", nil)
	group, _ := Outlined(button)

	if group.MinSize() != button.MinSize() {
		t.Errorf("outlining asks for %v where the control alone asks for %v",
			group.MinSize(), button.MinSize())
	}
}

// TestADisabledButtonShowsNoEdge and its pair hold the rule the owner chose:
// an edge says a control can be pressed, so the one control nobody can press
// must not carry one, and neither must a heading as wide as its own pane.
//
// 🔴 Rewritten 2026-09-09 when the button began drawing itself. It used to read
// a rectangle the control kept beside it and ask whether that was visible. That
// rectangle is gone: the face decides the edge from the state, so the question
// moved from "is the edge hidden" to "does the drawing contain an edge at all".
//
// The rule did not move an inch, and neither did the moment it is asked at -
// both states are set after the button exists, which is why each is checked by
// changing the button rather than by building a second one.
func edgesIn(object fyne.CanvasObject) int {
	switch shape := object.(type) {
	case *canvas.Rectangle:
		if shape.StrokeWidth > 0 {
			return 1
		}
		return 0
	case *fyne.Container:
		found := 0
		for _, child := range shape.Objects {
			found += edgesIn(child)
		}
		return found
	}
	return 0
}

// drawnFace is what the renderer of this button would put on the screen.
func drawnFace(button *Button) fyne.CanvasObject {
	drawing := button.CreateRenderer().(*buttonRenderer)
	return drawing.holder
}

func TestADisabledButtonShowsNoEdge(t *testing.T) {
	button := NewButton("Damage this file", func() {})
	if edgesIn(drawnFace(button)) == 0 {
		t.Fatal("a fresh button draws no edge, so the rest of this proves nothing")
	}

	button.Disable()
	if edgesIn(drawnFace(button)) != 0 {
		t.Error("a disabled button still draws the edge that says it can be pressed")
	}

	button.Enable()
	if edgesIn(drawnFace(button)) == 0 {
		t.Error("a button that became pressable again did not get its edge back")
	}
}

func TestAHeadingShowsNoEdge(t *testing.T) {
	button := NewButton("What this build can generate", func() {})
	button.Plain = true

	if edgesIn(drawnFace(button)) != 0 {
		t.Error("a heading draws the edge of a button, which is a box round a section")
	}
}
