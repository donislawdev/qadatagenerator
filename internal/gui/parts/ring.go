package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// ring is the one mark of focus every control this window draws itself wears:
// an outline in mark.focus that appears while the keyboard is on the control
// and is transparent otherwise.
//
// # Why one of these rather than three
//
// It is the third place the same shape was needed: the scrolling area drew it
// first, then the text field, then the table of a preview. The rule this
// project keeps says the same set of visual properties used a third time is a
// component rather than a copy, and the reason is the one the owner stated for
// the whole window: focus has to look like ONE thing across every control, and
// three spellings of it drift the day one of them is touched (DEC-215, DEC-246).
//
// # Why an outline stacked over, not painted
//
// A stroke on a shape with no fill covers nothing but its own edge, and a
// rectangle takes no pointer or key event, so the control under it goes on
// answering both. It is mounted SECOND in the stack so the edge is drawn over
// what the control draws rather than under it: on a control with a ground of
// its own, an outline drawn first has its inner half painted over, which
// tools/ognisko.py measured on the real window as a mark at half strength.
type ring struct{ frame *canvas.Rectangle }

// newRing makes a ring rounded to the given radius, dark until it is turned on.
//
// The stroke is BorderThickness for the reason a text field's edge is: at a
// width of one the toolkit draws the line between two rows of pixels and every
// pixel of it is a half blend, so the mark comes out faint.
func newRing(radius float32) ring {
	frame := canvas.NewRectangle(color.Transparent)
	frame.StrokeColor = color.Transparent
	frame.StrokeWidth = palette.BorderThickness
	frame.CornerRadius = radius
	return ring{frame: frame}
}

// framed is the control with the ring over it, which is what a screen puts on
// itself. Bare, the control is reached by the keyboard and shows nothing that
// says so.
func (r ring) framed(control fyne.CanvasObject) fyne.CanvasObject {
	return container.NewStack(control, r.frame)
}

// set turns the ring on when the keyboard arrives and off when it leaves.
func (r ring) set(on bool) {
	if on {
		r.frame.StrokeColor = palette.Ink(palette.MarkFocus)
	} else {
		r.frame.StrokeColor = color.Transparent
	}
	r.frame.Refresh()
}
