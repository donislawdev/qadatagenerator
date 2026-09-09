package paint

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The faces of the controls, drawn here and given behaviour next door.
//
// # Why a face is separate from the control at all
//
// A widget of the toolkit draws itself, and the drawing is the half this
// project needs to replace while the behaviour is the half it must not. Keeping
// the face as a function returning shapes means the same drawing serves a real
// control in parts and a still picture in a mock, and the two cannot drift,
// because there is one of them.
//
// This is also what makes the rule of DEC-228 checkable rather than aspirational:
// nothing here takes a callback, so nothing here can quietly become a control.

// ButtonLook is everything the face of a button depends on.
//
// # Why a struct and not five arguments
//
// 🔴 Because the face has to carry the STATES, and that is what makes it usable
// by the real control rather than only by a still picture.
//
// Until 2026-09-09 this took a label, an icon, a flag and a width, which is
// everything a drawing needs and not everything a control needs. So the mock
// drew buttons out of these shapes and the window drew them out of the
// toolkit's, and the two were free to look nothing alike - which is exactly
// what the owner saw when he put the two pictures side by side. The comment at
// the top of this file promised they could not drift because there was one
// drawing. There was one drawing and one of them was not using it.
//
// A caller that omits every state gets the resting look, so the mock says what
// it means without spelling out three falses.
type ButtonLook struct {
	// Label is the words on the button, and Icon the name of one of this
	// window's icons, empty for a button that shows none.
	Label string
	Icon  string

	// Filled marks the one action a screen is for.
	Filled bool

	// Width is a floor rather than a size. Zero means whatever the label needs.
	Width float32

	// Leading puts the label against the leading edge instead of centring it,
	// which is what a column of places to go needs and what a row of actions
	// does not.
	Leading bool

	// Plain drops the edge. A place to go in a column is not a slab to press,
	// and an edge around each of five names reads as a box around each rather
	// than as a list. The owner settled the same thing for the heading of the
	// generator list on 2026-09-05.
	Plain bool

	// Focused and Disabled are the two states a person can put a button into.
	Focused  bool
	Disabled bool
}

// ButtonFace is the look of a button, in whatever state it is in.
//
// # Why filled and quiet rather than a list of importances
//
// There is exactly one action a screen is for, and everything else is either a
// way out or a way sideways. That was settled on the pixels: four screens of
// five drew every button identically, and a screen where nothing stands out is
// a screen where a person has to read every control to find the one they want.
//
// The filled one carries the accent. The quiet one carries an edge, because
// DEC-211 settled that a shape a person can press has to say so, and measured
// what happens when it does not: 240 buttons at a contrast of 1.07.
//
// # What each state looks like, and why
//
//   - DISABLED loses both the fill and the edge and keeps a quiet label. An
//     edge says a control can be used, so leaving it on the one thing nobody
//     can press would make it the most findable thing on the screen.
//   - FOCUSED gains a ring INSIDE the shape, in the same ink as the label. It
//     is inside rather than around for the reason PrimaryButton records: a ring
//     on the outer edge sits where the resting edge already is, and reads as
//     the button having grown rather than as a mark.
//
// The ring is drawn by this face rather than by the toolkit, which is the whole
// point of the control wearing it. The toolkit's own answer floods the control
// with the focus colour, and that flood was measured at 1.23 for the mark and
// 1.56 for the label under it.
func ButtonFace(look ButtonLook) fyne.CanvasObject {
	ink := theme.TextPrimary
	var ground fyne.CanvasObject
	switch {
	case look.Disabled:
		ink = theme.TextSecondary
		ground = Clear()
	case look.Filled:
		ink = theme.TextOnFill
		ground = Fill(theme.FillPrimary, theme.RadiusControl)
	case look.Plain:
		ground = Clear()
	default:
		// An edge around nothing rather than a fill of its own. A quiet
		// button stands on three grounds here, and any fill would be a
		// patch of the wrong colour on two of them. Seen on the render.
		ground = Outline(theme.BorderControl, theme.RadiusControl)
	}

	row := []fyne.CanvasObject{}
	if look.Icon != "" {
		row = append(row, Middle(Icon(look.Icon, ink, theme.IconSmall)))
	}
	row = append(row, Middle(Text(look.Label, theme.TypeBody, ink)))

	// The gap between the icon and the words comes from the scale here, which
	// is the difference between this face and the toolkit's button: the toolkit
	// takes that gap from its own padding, and this window answers nought to
	// that question on purpose, so an icon on a stock button touches its label.
	words := Row(theme.NameStepTight, row...)

	// 🔴 Room either side of the words, and it is not decoration. Centring alone
	// asks for exactly the width of the label, so the edge is drawn ON the first
	// and last letter and a button reads as a word somebody boxed too tightly.
	// Seen on the render the day the button began drawing itself: every button
	// with no width of its own came out clipped.
	var inner fyne.CanvasObject = Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepGroup, theme.NameStepGroup, container.NewCenter(words))
	if look.Leading {
		inner = Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepRelated,
			container.New(layout.NewBorderLayout(nil, nil, words, nil), words))
	}

	dressed := []fyne.CanvasObject{ground, inner}
	if look.Focused && !look.Disabled {
		dressed = []fyne.CanvasObject{ground, ring(ink), inner}
	}
	return AtLeast(look.Width, theme.ControlHeight, container.NewStack(dressed...))
}

// ring is the mark saying the keyboard is on this shape.
//
// Inset by a whole stroke rather than half of one, so that a stroke drawn
// centred on its path lands entirely inside the shape. Half, which is what an
// edge around a control uses, would leave the outer half of the ring on
// whatever the button is standing on, and that is a different measurement on
// every screen.
func ring(inkToken string) fyne.CanvasObject {
	mark := Outline(inkToken, theme.RadiusControl-theme.BorderThickness)
	return Pad(theme.NameStepHairline, theme.NameStepHairline,
		theme.NameStepHairline, theme.NameStepHairline, mark)
}

// FieldFace is the look of something typed into: a ground a step off the
// window, an edge, and the text inside it.
//
// The face carries no caret and no selection. Those belong to the control that
// wears it, and the toolkit draws both.
func FieldFace(value, rung string, width float32) fyne.CanvasObject {
	ground := Bordered(theme.BgControl, theme.BorderControl, theme.RadiusControl)
	text := Middle(Mono(value, rung, theme.TextPrimary))
	return AtLeast(width, theme.ControlHeight,
		On(ground, Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepRelated, text)))
}

// ChooserFace is the look of a control that opens a list: the field face with a
// mark at its trailing edge saying it opens.
//
// # Why the mark is drawn rather than left to the toolkit
//
// The toolkit draws its own arrow inside its chooser, in its own icon set and
// its own colour, and a screen carrying one stock arrow among nineteen icons of
// another family is the kind of detail nobody can name and everybody sees.
func ChooserFace(value string, width float32) fyne.CanvasObject {
	ground := Bordered(theme.BgControl, theme.BorderControl, theme.RadiusControl)
	inner := container.NewBorder(nil, nil, nil,
		Middle(Icon(IconChevron, theme.TextSecondary, theme.IconSmall)),
		Middle(Text(value, theme.TypeBody, theme.TextPrimary)))
	return AtLeast(width, theme.ControlHeight,
		On(ground, Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepTight, inner)))
}

// SearchFace is the look of the field a list is filtered with: the field face
// with a magnifier before the text and the accent on its edge, because it is
// the one control on an open panel that the keyboard starts in.
func SearchFace(prompt string) fyne.CanvasObject {
	ground := Bordered(theme.BgSurface, theme.MarkCurrent, theme.RadiusControl)
	inner := container.NewBorder(nil, nil,
		Pad(theme.NameStepHairline, theme.NameStepHairline, theme.NameStepHairline, theme.NameStepTight,
			Middle(Icon(IconSearch, theme.TextSecondary, theme.IconSmall))), nil,
		Middle(Text(prompt, theme.TypeBody, theme.TextSecondary)))
	return AtLeast(0, theme.ControlHeight,
		On(ground, Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepRelated, theme.NameStepRelated, inner)))
}
