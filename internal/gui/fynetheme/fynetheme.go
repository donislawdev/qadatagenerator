// Package fynetheme dresses the toolkit in the palette this project measured.
//
// # Why this is a separate package from the palette
//
// The palette compiles without the graphical toolkit, which is what let the
// bridge to docs/17 close before the toolkit was ever added. Putting the
// adapter beside the values would have undone that: every check of a colour
// would then need the C toolchain to build. So the values stay where anything
// can read them, and this package is the only place that knows a toolkit
// exists at all.
//
// # What it delegates and why that is not laziness
//
// Type sizes, fonts and icons go to the toolkit's own theme. docs/17 says
// plainly that concrete type sizes arrive with the first window and are taken
// from measured behaviour rather than from another platform's ladder, and
// nothing has been measured yet. Inventing a ladder here would be prose with
// no measurement behind it, which is the one thing this project refuses
// everywhere else.
package fynetheme

import (
	"image/color"

	"fyne.io/fyne/v2"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Dark is the one theme this product has.
//
// One theme rather than following the desktop is a decision with a price that
// was written down when it was made: on a light desktop the window stands
// apart from everything around it, and some testers work in light. The gain is
// one palette, one set of contrast measurements and one appearance in every
// screenshot, and it is what the owner chose.
type Dark struct{}

// Ensure the toolkit still accepts this as a theme if its interface grows.
var _ fyne.Theme = Dark{}

// colourFor maps a name the toolkit asks for onto a name we measured.
//
// A name absent here falls through to the toolkit's own dark theme rather than
// to a guess of ours. That keeps the surface honest: the entries below are the
// colours this project has an opinion about, and the rest are not claimed to
// be measured.
var colourFor = map[fyne.ThemeColorName]string{
	toolkit.ColorNameBackground:        palette.NameSurface,
	toolkit.ColorNameForeground:        palette.NameText,
	toolkit.ColorNameForegroundOnError: palette.NameText,
	toolkit.ColorNameDisabled:          palette.NameTextSubdued,
	toolkit.ColorNamePlaceHolder:       palette.NameTextSubdued,
	toolkit.ColorNameError:             palette.NameRejection,
	toolkit.ColorNameSuccess:           palette.NameAcceptance,
	toolkit.ColorNameWarning:           palette.NameWarning,
	toolkit.ColorNamePrimary:           palette.NameFocus,
	toolkit.ColorNameFocus:             palette.NameFocus,
	toolkit.ColorNameSelection:         palette.NameRowSelected,
	toolkit.ColorNameHover:             palette.NameRowHover,
	toolkit.ColorNameSeparator:         palette.NameRowDivider,
	toolkit.ColorNameInputBorder:       palette.NameBorderActive,
}

// Color answers with a measured colour where this project has one.
//
// The variant is ignored on purpose. There is one theme, so asking for the
// light one and getting the dark one is the decision working rather than a
// branch somebody forgot.
func (Dark) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if ours, found := colourFor[name]; found {
		return palette.Named(ours).NRGBA()
	}
	// A screen may also ask for an entry of the palette by its own name, which
	// is what lets a control of the toolkit carry a measured colour. The
	// alternative was a piece of canvas text per coloured phrase, and that does
	// not trim itself to the space it was given: measured on 2026-08-18 in a
	// window 1342 pixels wide, columns of such text ran over one another and
	// two facts about different things were drawn in the same place.
	if entry, ours := palette.Lookup(string(name)); ours {
		return entry.NRGBA()
	}
	if name == toolkit.ColorNameMenuBackground || name == toolkit.ColorNameOverlayBackground {
		return palette.Background.NRGBA()
	}

	// # Why the answer of the toolkit is inspected rather than returned
	//
	// Measured on 2026-08-12: a name the toolkit does not know comes back
	// fully transparent rather than as an error, so a control painted with it
	// is an invisible hole and the cause is nowhere near the symptom. Partial
	// transparency is legitimate, since a shadow is meant to be see through,
	// so only the fully transparent answer is replaced.
	//
	// It is replaced with the text colour rather than with the background,
	// because a colour nobody claimed should be visible enough to be found
	// and fixed. This is the ban on silence applied to a pixel.
	// The variant is forced rather than passed through, and the toolkit's own
	// documentation is what asks for this: the theme that ignores the user's
	// preference is deprecated in favour of a custom theme doing exactly what
	// this one does. There is one theme here by decision, so the dark variant
	// is the answer whatever was asked for.
	fromToolkit := toolkit.DefaultTheme().Color(name, toolkit.VariantDark)
	if _, _, _, alpha := fromToolkit.RGBA(); alpha == 0 {
		return palette.Named(palette.NameText).NRGBA()
	}
	return fromToolkit
}

// sizeFor maps the toolkit's spacing questions onto the closed scale.
//
// Only spacing is answered here. A type size is not a distance between things
// and the scale says nothing about it, so claiming one would be inventing a
// number the document explicitly leaves until it can be measured.
var sizeFor = map[fyne.ThemeSizeName]string{
	toolkit.SizeNamePadding:            palette.NameStepTight,
	toolkit.SizeNameInnerPadding:       palette.NameStepHairline,
	toolkit.SizeNameLineSpacing:        palette.NameStepHairline,
	toolkit.SizeNameSeparatorThickness: palette.NameStepHairline,
}

func (Dark) Size(name fyne.ThemeSizeName) float32 {
	if ours, found := sizeFor[name]; found {
		return palette.StepOf(ours)
	}
	return toolkit.DefaultTheme().Size(name)
}

func (Dark) Font(style fyne.TextStyle) fyne.Resource {
	return toolkit.DefaultTheme().Font(style)
}

func (Dark) Icon(name fyne.ThemeIconName) fyne.Resource {
	return toolkit.DefaultTheme().Icon(name)
}
