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

// TestADisabledButtonShowsNoEdge and its pair hold the rule the owner chose,
// at the one moment it changes: a button becomes unpressable while the window
// is running, and the mark saying it can be pressed has to go with it.
func TestADisabledButtonShowsNoEdge(t *testing.T) {
	button := NewButton("Damage this file", func() {})
	if !button.edge.Visible() {
		t.Fatal("a fresh button shows no edge, so the rest of this proves nothing")
	}

	button.Disable()
	if button.edge.Visible() {
		t.Error("a disabled button still shows the edge that says it can be pressed")
	}

	button.Enable()
	if !button.edge.Visible() {
		t.Error("a button that became pressable again did not get its edge back")
	}
}

// TestAHeadingShowsNoEdge covers the second case, and covers it at the moment
// it is decided rather than at build time: importance is set after the button
// exists, and the toolkit offers no hook for it, so the edge is settled when
// the control is handed to a layout.
func TestAHeadingShowsNoEdge(t *testing.T) {
	button := NewButton("What this build can generate", func() {})
	button.Importance = widget.LowImportance

	button.OnScreen()

	if button.edge.Visible() {
		t.Error("a heading shows the edge of a button, which is a box drawn round a section")
	}
}
