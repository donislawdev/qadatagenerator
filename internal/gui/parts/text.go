package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The steps of text this window writes, and the reason there are four of them.
//
// # What was there before
//
// Two, in practice: bold and not bold. Every screen named its own parts with
// widget.NewLabelWithStyle and a bold style, and four packages had grown a
// private copy of the same three line helper. Seen on a real window on
// 2026-09-08, which was the first time anybody looked at one: with a single
// size everywhere, no screen said what it was, no section said where it began,
// and the owner's word for the result was that it is ugly. That is a fair
// reading of a wall of one size.
//
// # Why these sizes and not a scale of our own
//
// The toolkit already answers four, read at theme/size.go:263 to 269 on
// 2026-09-08: 14 for text, 18 for a sub heading, 24 for a heading and 11 for a
// caption. Asking for them by name costs nothing and keeps the one place that
// answers for a size the theme, which is the same rule the palette follows for
// colour.
//
// # What is deliberately NOT done here
//
// The base size is not moved. It was turned down for the reason it is still
// turned down: every layout in the window and every stored pattern is measured
// against it, so changing it moves all of them at once and none of that would
// be visible in the source. The ladder is built by asking for the sizes ABOVE
// and BELOW the base instead, which leaves the base where every stored state
// already has it.
//
// The accepted drawing asks for 20 over a body of 13, a ratio of 1.54. A sub
// heading over our body is 18 over 14, a ratio of 1.29. That is quieter than
// the drawing and louder than nothing, and the alternative on offer is 24 over
// 14, a ratio of 1.71, which is louder than the drawing asks for. The nearer of
// the two is taken.

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
// frame keeps one of these and rewrites it as screens are opened. The three
// below hand back what a layout takes, since nothing rewrites those.
func Title(words string) *widget.Label {
	label := widget.NewLabel(words)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.SizeName = toolkit.SizeNameSubHeadingText
	return label
}

// Section names one part of a screen, with the line the drawing puts under it.
//
// # Why the line rather than a larger size
//
// The accepted drawing asks for 15 where the body is 13, which is a difference
// of two points and not a step anybody reads as one. What separates a section
// there is the rule beneath the words, and that is the part doing the work. So
// this is the body size in bold with a divider under it, which is what the
// drawing amounts to, rather than a size close enough to the body to look like
// a mistake.
//
// This is also the only thing in the window that draws a region. Panels and
// boxes were turned down in the redesign, so a heading with a line under it is
// what says where one part of a screen stops and the next begins.
func Section(words string) fyne.CanvasObject {
	return container.NewVBox(headingLabel(words), widget.NewSeparator())
}

// headingLabel is what the words of a heading look like, in one place.
//
// Section and Card both name a group and have to name it the same way, and
// they cannot share the whole heading because they draw the line under it
// differently: Section draws a separator beneath itself, and Card draws a rule
// that reaches the edges of the box. Sharing the words alone is what keeps one
// spelling of a heading rather than two that drift apart the first time either
// changes.
func headingLabel(words string) *widget.Label {
	label := widget.NewLabel(words)
	label.TextStyle = fyne.TextStyle{Bold: true}
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

// Detail is the quieter line that hangs under something else: what would have
// been accepted, what to do instead, what was not looked at.
//
// Dimmer AND smaller, where the window until now had only dimmer. The caption
// size is what the toolkit answers for helper text, and low importance is what
// answers for its colour, so neither is a value invented here.
func Detail(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	label.SizeName = toolkit.SizeNameCaptionText
	return label
}
