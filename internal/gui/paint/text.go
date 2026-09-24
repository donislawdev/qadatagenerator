package paint

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Text is one run of words at a named rung of the type ladder, in the colour of
// a token.
//
// # Why canvas.Text rather than the toolkit's label
//
// A label resolves its own size and colour through the theme of the scope it is
// in, which is right for a control and wrong for a heading: two labels of
// different sizes need two scopes, and a scope is a container in the tree. This
// takes both directly, so a screen states what it is drawing in one line and
// the stored tree of that screen names the size rather than a scope.
//
// # What this cannot do, so that nobody reaches for it there
//
// It does not wrap and it does not truncate. A canvas text asks for the width
// of its whole string and takes it, so a sentence built this way inside a
// column makes that column as wide as the sentence. Anything that has to wrap
// is a label with a size name, and the bridge in fynetheme is what makes those
// names resolvable.
func Text(words, rung, token string) *canvas.Text {
	return styled(words, rung, token, fyne.TextStyle{})
}

// Strong is the same at the weight the drawing uses for a heading and for the
// one word of a row that has to be found first.
//
// # Why bold rather than a heavier face of its own
//
// The embedded family carries Regular and SemiBold, and the toolkit picks
// between them from this flag. A third weight would be another face in the
// binary for a difference nobody asked for.
func Strong(words, rung, token string) *canvas.Text {
	return styled(words, rung, token, fyne.TextStyle{Bold: true})
}

// Mono is a value or a command, in the face those are read in.
//
// # Why a second face at all
//
// A generated identifier is compared character by character against what an
// application stored, and a proportional face makes that comparison harder than
// it has to be: 1 and l and I are three widths and one shape. A command is the
// same argument, since it is copied and typed rather than read.
//
// This is also the single biggest reason the window looked like a default
// application before: the toolkit ships DejaVu Sans Mono, and half of this
// screen is values.
func Mono(words, rung, token string) *canvas.Text {
	return styled(words, rung, token, fyne.TextStyle{Monospace: true})
}

// Caption is a group heading: the smallest rung, in capitals, in the ink of a
// caption. That ink was the quieter of the two text tokens until 2026-09-24,
// when the owner chose the accent for the structure of a screen.
//
// # Why capitals rather than letter spacing
//
// The usual treatment for a heading of this kind spaces the letters apart, and
// Fyne 2.8.1 cannot: fyne.TextStyle carries Bold, Italic, Monospace, Symbol,
// Underline, Strikethrough and TabWidth, and nothing about tracking. Read on
// the source rather than recalled. So capitals carry it alone.
//
// # Why the caller does not pass the case
//
// A caption written in capitals at one call site and in sentence case at
// another is two kinds of heading, and the difference would be invisible in
// review. The rule lives here instead, where there is one of it.
//
// 🔴 What would make the capitals here wrong: the first translation of this
// window. Turkish capitalises a dotless i differently, and strings.ToUpper does
// not know that any more than a hand written loop would. When the translation
// file arrives, the capitals move INTO the translated string, where a
// translator can write them, and this call goes away.
func Caption(words string) *canvas.Text {
	return Strong(strings.ToUpper(words), theme.TypeLabel, theme.TextCaption)
}

// Paragraph is a sentence that wraps to the width it is given, at the body rung
// in the quieter ink the toolkit gives low importance.
//
// # Why a label rather than Text above
//
// It wraps, and canvas text cannot. Wrapping set on a label is still not
// wrapping done: a row gives each child the width it asks for and a wrapping
// label asks for its whole sentence, so every use of this belongs in a column.
//
// # Why it is here rather than on each screen
//
// The take screen and the recipe screen each built it, identically to the
// character, and the part both of them show about damage would have been the
// third copy (DEC-252). Three copies of one look is the point at which the look
// stops having one place to change it.
func Paragraph(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	label.SizeName = fyne.ThemeSizeName(theme.TypeBody)
	return container.NewStack(label)
}

func styled(words, rung, token string, style fyne.TextStyle) *canvas.Text {
	text := canvas.NewText(words, theme.Ink(token))
	text.TextSize = theme.TypeSize(rung)
	text.TextStyle = style
	return text
}

// TableCell is one piece of text in a table, on one line and cut with an
// ellipsis. SetTableCell writes into one, which is what a table asks for every
// time it shows a cell again.
//
// 🔴 Cut rather than wrapped, and never mid character. Values in a table here
// are deliberately multi byte: this tool generates for pl-PL and a whole module
// of damage exists to produce characters outside the basic plane. Cutting by
// byte puts invalid UTF-8 on screen, which this project has already repaired
// once in a report.
//
// It was a method of the recipe screen until the catalogue of controls needed
// the same cell on 2026-09-25 (DEC-253). A table in the catalogue drawn with a
// cell of its own would be a table no screen shows.
func TableCell() *widget.RichText {
	text := widget.NewRichTextWithText("")
	text.Truncation = fyne.TextTruncateEllipsis
	text.Wrapping = fyne.TextWrapOff
	return text
}

// SetTableCell writes words into a cell TableCell made, and does nothing to
// anything else, since the cells come from a pool the toolkit hands back and
// forth.
func SetTableCell(cell fyne.CanvasObject, words string) {
	text, isCell := cell.(*widget.RichText)
	if !isCell {
		return
	}
	text.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: words}}
	text.Refresh()
}
