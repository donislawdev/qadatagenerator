package paint

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The faces of the panel a kind is picked from: the panel itself, a heading
// inside it and one option under a heading. Drawn here for the reason every
// other face is, and it was measured on this one before it was drawn: the mock
// drew these rows out of the primitives inside tools/makieta, so the window
// would have been the second drawing of them (DEC-244).

// OptionLook is everything the face of one option depends on.
type OptionLook struct {
	// Name is the word a person picks, and the word the chooser will show.
	Name string
	// Detail stands beside the name, and may be empty. On the quick screen it
	// is what a person calls that kind, beside the code a recipe uses. Until
	// 2026-10-21 it was the written form of a value, a row of noughts the owner
	// found told nobody anything (DEC-262).
	Detail string
	// Note stands at the far end of the row, in the smaller size, and may be
	// empty: on the quick screen, how long a value of that kind is.
	Note string
	// NameWidth is a floor under the column of names, so that every detail in
	// the panel starts on the same line. Measured from the widest name by
	// OptionNameWidth rather than written down.
	NameWidth float32
	// Chosen says this is the option the chooser holds. It is said by a mark
	// and by the ink of the name, so it survives the ground changing under
	// the pointer.
	Chosen bool
	// Current says the keyboard is on this option, and Hovered that the
	// pointer is. Both are said by the ground, the way every menu says which
	// row a press would pick. Until 2026-10-20 the keyboard drew an edge round
	// the row, which made a list read as a stack of text fields, and the owner
	// called it ugly (DEC-260).
	Current bool
	Hovered bool
	// Compact gives up the floor under the height of a row, so the row is as
	// tall as its words. The list under a code editor's caret is read that way,
	// a line of text to a row, and it opens inside a field where every row
	// taller than a line is a row fewer in sight (DEC-338).
	Compact bool
}

// OptionFace is the look of one option in the panel.
func OptionFace(look OptionLook) fyne.CanvasObject {
	var ground fyne.CanvasObject = Clear()
	nameInk := theme.TextPrimary
	// The room for the mark is reserved on every row, chosen or not, so the
	// detail column does not shift when the choice moves.
	var mark fyne.CanvasObject = Space(theme.IconSmall, 0)
	lit := look.Hovered || look.Current
	switch {
	case lit && look.Chosen:
		ground = Fill(theme.BgCurrentHover, theme.RadiusControl)
	case lit:
		ground = Fill(theme.BgHover, theme.RadiusControl)
	case look.Chosen:
		ground = Fill(theme.BgCurrent, theme.RadiusControl)
	}
	if look.Chosen {
		nameInk = theme.TextAccent
		mark = Middle(Icon(IconCheck, theme.TextAccent, theme.IconSmall))
	}

	columns := Row(theme.NameStepRelated,
		AtLeast(look.NameWidth, 0, Middle(Text(look.Name, theme.TypeBody, nameInk))),
		Middle(Text(look.Detail, theme.TypeBody, theme.TextSecondary)))
	end := mark
	if look.Note != "" {
		end = Row(theme.NameStepRelated,
			Middle(Text(look.Note, theme.TypeLabel, theme.TextSecondary)), mark)
	}
	inner := container.NewBorder(nil, nil, nil, end, columns)

	face := container.NewStack(ground,
		Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepRelated, inner))
	if look.Compact {
		return face
	}
	return AtLeast(0, theme.OptionHeight, face)
}

// OptionInset is how far from the edge of a row its name begins, which is
// what a list placed under a word moves left by so that the names in it stand
// under the word.
func OptionInset() float32 { return theme.StepOf(theme.NameStepRelated) }

// OptionWidth is how wide the face of an option is with everything on it in
// full, measured from the face for the reason OptionNameWidth gives.
func OptionWidth(look OptionLook) float32 { return OptionFace(look).MinSize().Width }

// ellipsis ends words cut to a width.
const ellipsis = "…"

// CutToWidth is words at a rung of the type ladder that take no more than a
// width: the words themselves when they fit, otherwise as many whole
// characters as fit with an ellipsis after them, and nothing when not even the
// ellipsis does. Never cut inside a character, for the reason TableCell gives.
//
// The cut is guessed from the share of the width that fits and then moved a
// character at a time, so a sentence costs a few measurements rather than one
// per character.
func CutToWidth(words, rung string, width float32) string {
	size := theme.TypeSize(rung)
	measure := func(text string) float32 { return fyne.MeasureText(text, size, fyne.TextStyle{}).Width }
	full := measure(words)
	if full <= width {
		return words
	}
	runes := []rune(words)
	fits := func(count int) bool { return measure(string(runes[:count])+ellipsis) <= width }
	count := min(max(int(float32(len(runes))*width/full), 0), len(runes))
	for count > 0 && !fits(count) {
		count--
	}
	for count < len(runes) && fits(count+1) {
		count++
	}
	if count == 0 && !fits(0) {
		return ""
	}
	return string(runes[:count]) + ellipsis
}

// OptionNameWidth is how wide the column of names has to be to hold any of
// these without the details beside them moving.
//
// Measured from the face rather than added up, for the reason ChooserWidth
// gives: arithmetic written beside a drawing is a second copy of the drawing.
func OptionNameWidth(names ...string) float32 {
	widest := float32(0)
	for _, name := range names {
		if measured := Text(name, theme.TypeBody, theme.TextPrimary).MinSize().Width; measured > widest {
			widest = measured
		}
	}
	return widest
}

// OptionHeading is the caption a group of options stands under. A group step
// above it, twice what it was until 2026-10-21, so a heading reads as the start
// of a group rather than as one more row (the owner found the panel cramped).
func OptionHeading(words string) fyne.CanvasObject {
	return Pad(theme.NameStepGroup, theme.NameStepHairline,
		theme.NameStepRelated, theme.NameStepHairline, Caption(words))
}

// PanelGround is the card the panel is, with the one shadow this window draws.
//
// # Why bg.surface and not the ground of a control
//
// The panel holds a field, and a field stands one step above whatever it is
// on, which is what tells it apart from the card around it. The mock had the
// two the other way round, with the field a step DOWN from the panel, and
// nothing in it could say so, because the mock's field was a picture and the
// window's is parts.Field with a ground of its own.
//
// The edge is the quiet line a card wears rather than the edge of a control.
// The shadow already lifts the panel off the screen, and the louder edge read
// as one more white outline (DEC-260).
func PanelGround() *canvas.Rectangle {
	ground := Bordered(theme.BgSurface, theme.LineQuiet, theme.RadiusCard)
	ground.Shadow = PanelShadow()
	return ground
}
