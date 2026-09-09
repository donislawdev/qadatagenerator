package paint

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

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

// ButtonFace is the look of a button, filled or quiet.
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
func ButtonFace(label, icon string, filled bool, width float32) fyne.CanvasObject {
	ink := theme.TextPrimary
	var ground fyne.CanvasObject
	if filled {
		ink = theme.TextOnFill
		ground = Fill(theme.FillPrimary, theme.RadiusControl)
	} else {
		// An edge around nothing rather than a fill of its own. A quiet
		// button stands on three grounds here, and any fill would be a
		// patch of the wrong colour on two of them. Seen on the render.
		ground = Outline(theme.BorderControl, theme.RadiusControl)
	}

	row := []fyne.CanvasObject{}
	if icon != "" {
		row = append(row, Middle(Icon(icon, ink, theme.IconSmall)))
	}
	row = append(row, Middle(Text(label, theme.TypeBody, ink)))

	inner := container.NewCenter(Row(theme.NameStepTight, row...))
	return AtLeast(width, theme.ControlHeight, On(ground, inner))
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
