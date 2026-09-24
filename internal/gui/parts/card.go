package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Card is a titled box: a heading, a rule the whole width of the box under it,
// and the thing the heading names.
//
// # What it is for, and how it differs from the two neighbours
//
// Section is a heading with a line under it and nothing around it, which says
// where one part of a screen begins. Panel is a box with no heading, which says
// that the text in it is there to be taken rather than read. Neither says "this
// group of controls belongs together and is called this", and that is what a
// screen holding three groups of settings needs most.
//
// Measured on the window at the size the owner keeps it, 1670 by 1210 on
// 2026-09-08: the right hand pane of the recipe screen held a heading, a row of
// settings, another heading and a command, all the same distance apart and all
// on the same ground. Nothing said where one group stopped and the next began.
//
// # Why the rule reaches the edges
//
// This is the whole reason a card is built here rather than by putting a
// Section inside a Panel. A padded box gives its child the size less a margin
// on every side, so a line drawn by the child stops short of the box at both
// ends and reads as a line somebody drew in the wrong place. Here the heading
// and the body are padded separately and the rule between them is not, which is
// what makes it a rule across a card rather than a line inside one.
//
// # What aside is for
//
// The right hand end of the heading, for the one fact belonging to the whole
// group rather than to any control in it: how many generators there are,
// whether anything has been saved yet. It may be nil, and most cards pass nil.
func Card(heading string, aside fyne.CanvasObject, body fyne.CanvasObject) fyne.CanvasObject {
	room := palette.StepOf(palette.NameStepRelated)

	var top fyne.CanvasObject = headingLabel(heading)
	if aside != nil {
		top = container.NewBorder(nil, nil, nil, aside, top)
	}

	// The rule is a rectangle rather than widget.NewSeparator, because a
	// separator takes its thickness from the theme and its colour from the
	// theme's separator entry, and a card has to draw the same line as the box
	// around it. Two lines a shade apart on one card read as a mistake.
	rule := canvas.NewRectangle(palette.Named(palette.NameRowDivider).NRGBA())
	rule.SetMinSize(fyne.NewSize(0, palette.HairlineThickness))

	head := container.NewVBox(
		container.New(layout.NewCustomPaddedLayout(room, room, room, room), top),
		rule,
	)
	inside := container.New(layout.NewCustomPaddedLayout(room, room, room, room), body)

	return container.NewStack(boxGround(), container.NewBorder(head, nil, nil, nil, inside))
}
