package shell

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// item is one name in the column, with the two shapes that say whether the
// window is on it.
//
// # Why the mark moved off the icon, measured 2026-09-09
//
// It used to be the toolkit's forward arrow, put on the button of the current
// screen and taken off the others. That worked while nothing else wanted the
// icon slot, and it stopped working the moment every screen gained an icon of
// its own: Open ran after the icons were set and overwrote all five of them, so
// the column showed one arrow and four names with nothing beside them. The
// symptom was an absence, which is the kind nobody goes looking for - the
// icons were placed, the code compiled, the tests passed, and the render simply
// had no icons in it.
//
// 🔴 The fix is not a workaround, and the register says so. bg.current is
// described in the theme as "the sidebar item naming the screen that is open"
// and mark.current as "the bar beside the sidebar item the window is on". Both
// were measured and written down when the ladders were built, and neither was
// read by a single line of code. The design this arrives at was already the
// designed one - what was missing was anything using it.
//
// # Why two shapes and not one
//
// The bar alone is three pixels and easy to miss on a column of five. The
// ground alone is one step of the ladder away from the sidebar, which reads as
// a smudge rather than as a selection. Together they say the same thing twice,
// which is what a state a person navigates by should do, and neither of them is
// a colour carrying meaning alone: docs/17 refuses that, because a hue is what
// somebody who does not separate hues cannot use.
//
// # Why the ground is the button's and the bar is the item's, since 2026-09-24
//
// The ground used to be a rectangle held here, under the button. The button
// draws its own ground under a pointer, opaque, so a pointer on the open screen
// covered the tint with grey - on the live window right after every click. The
// button is the one that knows the pointer, so it draws the tinted ground too,
// in every state, and this only tells it which item is the open one.
type item struct {
	button *parts.Button

	// mark is the bar at the leading edge. Held rather than looked up, because
	// whether it shows changes every time a person moves between screens.
	mark *canvas.Rectangle

	onScreen fyne.CanvasObject
}

// newItem dresses one button as a place to go.
//
// 🔴 The bar goes in the LEADING edge of a border and the button in the middle,
// which is the one arrangement that works here. A bar is three pixels wide and
// never grows, so it is safe in an edge - the rule this project has paid for
// says only what cannot grow may go there. The button does grow, with the name
// and with the translation, so it takes the middle, which is what a border
// hands its remaining width to.
func newItem(button *parts.Button) *item {
	entry := &item{
		button: button,
		mark:   canvas.NewRectangle(color.Transparent),
	}

	bar := paint.AtLeast(palette.MarkWidth, 0, entry.mark)
	// The name is inset from the bar rather than touching it. Without the
	// gap the icon sits against the edge of the window and the column reads
	// as having been cut off rather than as having a margin.
	name := paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepTight, palette.NameStepTight, button.OnScreen())
	entry.onScreen = container.New(layout.NewBorderLayout(nil, nil, bar, nil), bar, name)
	return entry
}

// show says whether the window is on this screen.
//
// Transparent rather than the sidebar's own colour for the state it is not in,
// which is the rule this project wrote down after finding eight rectangles
// painting a ground onto a different ground: a shape meaning "draw nothing
// here" has to BE nothing, because the surface underneath it is not always the
// one whoever wrote the call had in mind.
func (i *item) show(current bool) {
	i.button.Current = current
	i.mark.FillColor = color.Transparent
	if current {
		i.mark.FillColor = palette.Ink(palette.MarkCurrent)
	}
	i.redraw()
}

// redraw asks both to repaint, for the reason every renderer in this project
// does it: refreshing the container redraws the object it is given, and what
// changed lives on two children of it.
func (i *item) redraw() {
	i.button.Refresh()
	i.mark.Refresh()
}
