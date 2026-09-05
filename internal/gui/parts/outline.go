package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Outlined draws an edge around a control, so that the control can be seen
// before anybody puts the keyboard or the pointer on it.
//
// # What the toolkit does, and why that is not enough here
//
// A button at rest is a rectangle of ColorNameButton and nothing else. The
// toolkit's renderer at widget/button.go:342 sets a fill and a corner radius
// and never a stroke, so the only thing separating a button from the panel it
// stands on is the difference between those two fills.
//
// Measured through the pixels of the render on 2026-09-05, across every state
// of every screen: 248 buttons, of which 240 measured 1.07. The fill arrives as
// #28292E and the panel under it as #242424, where WCAG 1.4.11 asks for 3.0 of
// an interface element. Eight buttons passed, and all eight had the keyboard on
// them. A person reading the window sees the words and not the control.
//
// # Why the edge rather than the fill
//
// Because no fill answers at once, and that is arithmetic rather than taste.
// Four constraints bear on the fill of a button and they do not meet:
//
//   - to stand 3.0 against the panel it needs a luminance of at least 0.1529
//   - for the label #E8E8E8 to hold 4.5 it may have at most 0.1404
//   - for the focus colour to stand 3.0 against the resting button, at most
//     0.1333, where it measures 7.60 today
//   - for the label of a DISABLED button, #A0A0A0, to hold 4.5, at most 0.0392,
//     where it measures 5.55 today
//
// The floor is above every one of the three ceilings, and above the tightest of
// them by a factor of four. This is the same wall as the one Flooded describes
// next door, only higher.
//
// The other candidate, widget.HighImportance for one action per screen, was
// measured and turned down for a reason no reading of the source would have
// shown. A focused high importance button is blend(primary, focus), and the
// blend collapses to the overlay, so the button goes from #9ED8F0 at rest to
// #4EC9F5 with the keyboard on it: a focus indicator of 1.23, down from 7.60.
// It would repair one button per screen by breaking the mark on that same one.
//
// # Why a rectangle beside the control rather than a widget of our own
//
// Nothing here is drawn by us that the toolkit already draws. A button keeps
// its own fill, its hover, its tap animation, its icon and its focus flood, and
// gains one rectangle laid over it. A widget of our own would mean a renderer
// to go stale the next time the toolkit moves.
func Outlined(dressed fyne.CanvasObject) (*fyne.Container, *canvas.Rectangle) {
	// Transparent rather than a colour, because the control underneath paints
	// its own fill and this rectangle is here for its stroke alone.
	edge := canvas.NewRectangle(color.Transparent)
	edge.StrokeColor = palette.Named(palette.NameControlOutline).NRGBA()
	edge.StrokeWidth = palette.BorderThickness
	// The radius is asked of the theme rather than chosen, so that the edge
	// follows the corner the toolkit rounds the fill to. A radius of our own
	// would show as a straight corner crossing a curved one.
	edge.CornerRadius = beneath().Size(toolkit.SizeNameButtonRadius)

	// # Why two containers of the toolkit rather than one layout of ours
	//
	// The first version here was a layout written by hand, and the guard that
	// refuses a rebuilt piece of the toolkit turned it down, correctly. What
	// the arrangement has to do is give the control the whole size and the
	// edge the size less its own stroke, and the toolkit has both halves: a
	// stack gives every child the full size, and a padded layout gives its
	// child the size less a margin on each side. Nesting the second inside the
	// first says the same thing with nothing of ours in it, so there is no
	// layout here to go stale when the toolkit moves.
	//
	// The edge comes second, so it is drawn over the fill of the control
	// rather than under it.
	inset := palette.OutlineInset
	around := container.New(layout.NewCustomPaddedLayout(inset, inset, inset, inset), edge)
	return container.NewStack(dressed, around), edge
}
