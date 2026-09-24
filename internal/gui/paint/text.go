package paint

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

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

func styled(words, rung, token string, style fyne.TextStyle) *canvas.Text {
	text := canvas.NewText(words, theme.Ink(token))
	text.TextSize = theme.TypeSize(rung)
	text.TextStyle = style
	return text
}
