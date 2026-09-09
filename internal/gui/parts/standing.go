package parts

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// standing gives a control the height every control in this window has.
//
// # Why the toolkit's own height was not it
//
// Measured on the tree of the Quick value screen on 2026-09-09, at the size the
// owner keeps the window: a button arrived 28 tall, a chooser 28, and a text
// field 24, where the theme has said ControlHeight is 34 since DEC-232. None of
// those three numbers is chosen by anybody. A button is as tall as its label
// plus the theme's inner padding, a chooser the same, and a field is whatever
// the toolkit decided, so the height of a form is a consequence of the font
// rather than a decision about the form.
//
// 🔴 The 24 beside the 28 is the part that shows. A chooser and a text field
// stand in ONE row of ONE form on this screen, and four pixels apart in height
// is not read as four pixels: it is read as one of them being wrong. That is
// the same fault DEC-215 settled for the edge of a chooser, arriving through
// the other dimension, and it comes from the same place - a difference the
// toolkit happens to produce rather than one anybody decided.
//
// # Why a floor rather than a fixed size
//
// A control that needs more room than the scale allows for is a control whose
// label is too long, and clipping it would hide the problem rather than show
// it. paint.AtLeast sets a minimum and lets anything bigger through, so a long
// label still fits and still reads as wrong, which is what a person needs in
// order to fix it.
//
// # Why this is a wrapper and not a renderer
//
// It is a transparent rectangle standing behind the control, so the toolkit
// goes on drawing every control exactly as it did. Nothing here is a widget or
// a layout of our own, so the register of pieces this project rebuilds gains no
// entry, and there is no renderer to go stale when the toolkit moves.
func standing(control fyne.CanvasObject) fyne.CanvasObject {
	return paint.AtLeast(0, palette.ControlHeight, control)
}

// Standing is the same floor for a control this package does not wrap.
//
// The text field of the toolkit is used directly on two screens, because
// everything a person expects of one - the caret, the selection, the clipboard,
// the characters outside ASCII - is behaviour this project must not rewrite.
// It has no OnScreen of its own to put the floor in, so the screen that places
// one asks for it here rather than reaching for a distance of its own.
func Standing(control fyne.CanvasObject) fyne.CanvasObject { return standing(control) }
