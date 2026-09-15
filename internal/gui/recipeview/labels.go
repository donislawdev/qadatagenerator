package recipeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The shapes of text this screen writes. They are here rather than inline so
// that the drawing reads as what it says and not as how it is styled.

// sentence is a paragraph in the ink of the recipe's own words: a problem, a
// crash. It follows the width of the card rather than running off the edge of
// it.
//
// 🔴 Wrapping set on a label is not wrapping done. A row of the toolkit gives
// each child the width it asks for, and a wrapping label asks for the width of
// its whole sentence, so one placed in a row does not wrap at all. Every use of
// this goes into a column, where the width is the column's, which is what
// wrapping needs.
func sentence(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
	return container.NewStack(label)
}

// paragraph is a sentence that wraps to the width it is given, in the type of
// the body and the ink of an explanation: what hangs under a problem, what a
// run would write, what was not looked at.
//
// A label of the toolkit's rather than paint.Text, because paint.Text is a
// piece of canvas and never wraps, and the sentences here are among the
// longest on any screen of this window. The value inside is a list of
// generator names or somebody's own key, and neither has a length this screen
// decides.
func (v *View) paragraph(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
	return container.NewStack(label)
}

// told is a headline with a shape beside it: the verdict on the recipe, or on
// the run.
//
// The shape rather than a colour on its own, for the reason measured for this
// palette: the colours that carry meaning differ in hue while barely differing
// in lightness, so somebody who does not separate hues sees one shade of grey.
func (v *View) told(icon, ink, words string) fyne.CanvasObject {
	return paint.Marked(icon, ink, paint.Strong(words, palette.TypeBody, palette.TextPrimary))
}

// fact is a caption over a value, the way the card on the take screen states
// one.
func (v *View) fact(name i18n.Key, value string) fyne.CanvasObject {
	return paint.Column(palette.NameStepHairline,
		paint.Caption(v.text.T(name)),
		v.paragraph(value))
}
