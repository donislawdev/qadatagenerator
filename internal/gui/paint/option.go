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
	// Detail stands beside the name, and may be empty. On the quick screen
	// it is the written form of a value of that kind, which is a fact rather
	// than a sentence, and is read character by character.
	Detail string
	// NameWidth is a floor under the column of names, so that every detail in
	// the panel starts on the same line. Measured from the widest name by
	// OptionNameWidth rather than written down.
	NameWidth float32
	// Chosen says this is the option the chooser holds. It is said by a mark
	// and by the ink of the name, so it survives the ground changing under
	// the pointer.
	Chosen bool
	// Current says the keyboard is on this option. Said by the edge, the way
	// every control here says where the keyboard is (DEC-243).
	Current bool
	// Hovered says the pointer is on this option. Said by the ground, the way
	// every control here says where the pointer is.
	Hovered bool
}

// OptionFace is the look of one option in the panel.
func OptionFace(look OptionLook) fyne.CanvasObject {
	var ground fyne.CanvasObject = Clear()
	nameInk := theme.TextPrimary
	// The room for the mark is reserved on every row, chosen or not, so the
	// detail column does not shift when the choice moves.
	var mark fyne.CanvasObject = Space(theme.IconSmall, 0)
	switch {
	case look.Hovered:
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
		Middle(Mono(look.Detail, theme.TypeBody, theme.TextSecondary)))
	inner := container.NewBorder(nil, nil, nil, mark, columns)

	layers := []fyne.CanvasObject{ground}
	if look.Current {
		layers = append(layers, inside(Outline(theme.MarkFocus, theme.RadiusControl)))
	}
	layers = append(layers, Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepRelated, theme.NameStepRelated, inner))
	return AtLeast(0, theme.ControlHeight, container.NewStack(layers...))
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

// OptionHeading is the caption a group of options stands under.
func OptionHeading(words string) fyne.CanvasObject {
	return Pad(theme.NameStepTight, theme.NameStepHairline,
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
func PanelGround() *canvas.Rectangle {
	ground := Bordered(theme.BgSurface, theme.BorderControl, theme.RadiusCard)
	ground.Shadow = PanelShadow()
	return ground
}
