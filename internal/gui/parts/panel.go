package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Panel is a box with a quieter ground than the window and a line around it.
//
// # What it is for
//
// Two things on the accepted drawing are drawn this way and nothing else is:
// the value the quick screen produces, and the command that would produce the
// same thing in a terminal. Both are text somebody has come to TAKE rather than
// to read, by copying it or by typing it out, and the box is what says so. On
// the window as it stood on 2026-09-08 neither had one, so the value sat
// between a field and a button reading as a third setting, and the command sat
// under a sentence reading as more of the sentence.
//
// # Why a fill AND a line
//
// The fill alone is the surface colour, which stands 1.13 against the window
// behind it. That is a difference somebody notices only when told it is there,
// and the same arithmetic that put an edge on every button applies here. The
// line is the divider colour, which is what every other line in this window is
// drawn in, so a box does not introduce a value of its own.
//
// # Why this is built the way Outlined is built
//
// Same reason, in the same words: a stack gives every child the full size and a
// padded layout gives its child the size less a margin, so the two together say
// what is wanted with nothing of ours in it. There is no layout here to go
// stale when the toolkit moves, and the guard that refuses a rebuilt piece of
// the toolkit has nothing to object to.
func Panel(inner fyne.CanvasObject) fyne.CanvasObject {
	// The text is held off the edge of its own box by the step the scale keeps
	// for things read together. Without it the words start where the line is,
	// which reads as text that has overflowed rather than text in a box.
	room := palette.StepOf(palette.NameStepRelated)
	padded := container.New(layout.NewCustomPaddedLayout(room, room, room, room), inner)
	return container.NewStack(boxGround(), padded)
}

// boxGround is the quieter ground with a line around it that both boxed things
// in this window stand on.
//
// Panel and Card are two different arrangements on ONE surface. Kept apart they
// would be two rectangles a shade or a radius from each other, and a screen
// showing both would have the difference read as meaning something. The
// arrangement is what differs between them and the ground is what does not.
func boxGround() *canvas.Rectangle {
	ground := canvas.NewRectangle(palette.Named(palette.NameSurface).NRGBA())
	ground.StrokeColor = palette.Named(palette.NameRowDivider).NRGBA()
	ground.StrokeWidth = palette.BorderThickness
	// The corner is asked of the theme rather than chosen, so a box follows the
	// same curve as the fields and the buttons beside it. A radius of our own
	// would show as one corner disagreeing with every other on the screen.
	ground.CornerRadius = beneath().Size(toolkit.SizeNameInputRadius)
	return ground
}
