package parts

import (
	"fyne.io/fyne/v2"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The two pieces of text this package still hands out as toolkit labels.
//
// # Where the rest of the text went
//
// This file once held four steps: a title, a section heading, a sentence and a
// quieter detail line. The screens moved onto the ladder of type sizes in the
// theme (TypeScale, drawn by paint.Text) when they were rebuilt, and the
// section heading and the detail line stopped being called by anything. They
// were deleted with DEC-253, because an export nothing uses is a component the
// catalogue of components would have to show and no screen would ever look
// like.
//
// What is left is here for a reason each. The title is a label the frame keeps
// and rewrites as screens are opened, which a drawn piece of text cannot be.
// The sentence is a paragraph that wraps to the width it is given, which is a
// behaviour of the toolkit's label rather than a size.

// Title names the screen somebody is looking at.
//
// # Why the window draws this and not each screen
//
// The frame is what knows which screen is open, and it already holds the name
// of every one of them to write the column down the side. A title drawn by each
// screen would be five copies of one line, free to drift apart in size, in
// spacing and in whether it is there at all, and one of them would eventually
// not be there. Drawn by the frame it cannot be missing from a screen, because
// no screen is what puts it on.
//
// The label itself is handed back rather than something to draw, because the
// frame keeps one of these and rewrites it as screens are opened.
func Title(words string) *widget.Label {
	label := widget.NewLabel(words)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.SizeName = toolkit.SizeNameSubHeadingText
	return label
}

// Sentence is a paragraph that follows the width it is given rather than
// running off the edge of the window.
//
// 🔴 Wrapping set on a label is not wrapping done. A row of the toolkit gives
// each child the width it asks for, and a wrapping label asks for the width of
// its whole sentence, so one placed in a row does not wrap at all. Every use of
// this belongs in a column, where the width is the column's, which is what
// wrapping needs.
func Sentence(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	return label
}
