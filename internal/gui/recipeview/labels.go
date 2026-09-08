package recipeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// The three shapes of text this screen writes. They are here rather than
// inline so that the drawing above reads as what it says and not as how it is
// styled.

// sentence is a paragraph that follows the width of the window rather than
// running off the edge of it.
//
// 🔴 Wrapping set on a label is not wrapping done. A row of the toolkit gives
// each child the width it asks for, and a wrapping label asks for the width of
// its whole sentence, so one placed in a row does not wrap at all. Every use of
// this goes into a column, where the width is the column's, which is what
// wrapping needs.
func sentence(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	return label
}

// heading names the part of the answer under it, with the rule the accepted
// drawing puts under a section.
//
// The rule is what separates the verdict from the preview and the preview from
// what was not looked at. Before 2026-09-08 these were bold text and the parts
// of the answer ran together, which matters here more than elsewhere because
// this pane can hold four of them at once.
func heading(words string) fyne.CanvasObject {
	return parts.Section(words)
}

// detail is what hangs under a problem: what would have been accepted, what to
// do instead, or what was not looked at.
//
// It is dimmer than the line above it and still a paragraph, because the value
// inside it is a list of generator names or somebody's own key and neither has
// a length this screen decides.
func detail(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	return label
}
