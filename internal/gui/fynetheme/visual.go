package fynetheme

import (
	"image/color"

	"fyne.io/fyne/v2"
	toolkit "fyne.io/fyne/v2/theme"

	"github.com/donislawdev/qadatagenerator/internal/gui/assets"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Visual dresses the toolkit in the twelve step ladders and the embedded faces.
//
// # How it came to be the only theme
//
// It was written beside Dark, the theme of the first window, rather than as a
// change to it, because changing Dark would have moved every colour of five
// screens at once on a day when none of them was being rebuilt. Each screen
// moved onto this theme as it was rebuilt, the last on 2026-09-23, and Dark
// was deleted with DEC-253.
//
// # What it answers that the toolkit's own theme does not
//
// Two things, and both were the reason the window looked like the toolkit.
//
// The faces. The toolkit ships Noto Sans and DejaVu Sans Mono, and Dark handed
// every font question to it. Half of the screen this was built for is generated
// values in the mono face, so that single answer carried most of the impression
// the owner called generic.
//
// The type sizes. The ladder in the theme has five rungs, and a widget can only
// take one of them if the theme answers to its name. That is what makes a
// wrapping label possible at a size of ours, which canvas text cannot do at all.
type Visual struct{}

// Ensure the toolkit still accepts this as a theme if its interface grows.
var _ fyne.Theme = Visual{}

// tokenFor maps a colour the toolkit asks about onto a token of ours.
//
// # Why so few entries
//
// The look of this window is drawn from primitives rather than from stock
// widgets, so most of what the toolkit paints is either invisible under our own
// shapes or belongs to a control we have not dressed yet. An entry here is a
// claim that the toolkit paints that thing where a person can see it, and a
// claim nobody checked is worse than a gap.
//
// A name absent here falls through to the toolkit's own dark theme, because the
// entries are what this project has an opinion about and the rest are not
// claimed to be measured.
var tokenFor = map[fyne.ThemeColorName]string{
	toolkit.ColorNameBackground:        palette.BgWindow,
	toolkit.ColorNameForeground:        palette.TextPrimary,
	toolkit.ColorNameForegroundOnError: palette.TextPrimary,
	toolkit.ColorNameDisabled:          palette.TextSecondary,
	toolkit.ColorNamePlaceHolder:       palette.TextSecondary,
	toolkit.ColorNameError:             palette.StateDanger,
	toolkit.ColorNameSuccess:           palette.StateSuccess,
	toolkit.ColorNameWarning:           palette.StateWarning,
	toolkit.ColorNameInputBackground:   palette.BgControl,
	toolkit.ColorNameInputBorder:       palette.BorderControl,
	toolkit.ColorNameSeparator:         palette.LineQuiet,
	toolkit.ColorNameHover:             palette.BgHover,
	// 🔴 Its own token since 2026-09-09, and it was bg.current until then.
	//
	// bg.current is the sidebar item of the open screen, and it landed here
	// because the two wanted the same thing in words - a ground a step lighter
	// than the one under it. In numbers they do not: selected text sits on the
	// inside of a field, and one step of the neutral ladder against the next
	// measures 1.09, so a person selecting a seed saw nothing happen. The owner
	// reported exactly that. bg.selection is measured against every ground the
	// toolkit draws this colour on, and the arithmetic is beside the token.
	toolkit.ColorNameSelection: palette.BgSelection,
	// The header row of the table the manifest browser draws, read in the
	// source of 2.8.1 rather than assumed: widget/table.go:1226 and 1761.
	toolkit.ColorNameHeaderBackground:  palette.BgHeader,
	toolkit.ColorNameMenuBackground:    palette.BgSurface,
	toolkit.ColorNameOverlayBackground: palette.BgSurface,
	toolkit.ColorNameButton:            palette.BgControl,
	// 🔴 Primary is what the toolkit paints on a stock control that is chosen
	// or holds the keyboard, and it answers with the MARK rather than with the
	// accent fill. Corrected on 2026-09-09 after the switch to this theme
	// turned a guard red on the take screen.
	//
	// The fill was the obvious answer and it is wrong for a measured reason: on
	// a focused field the toolkit replaces the resting border with this colour,
	// so the pair is this against border.control, and two saturated mid tones
	// measure 1.26 where WCAG 1.4.11 asks for 3.0. The register carries the
	// arithmetic across the whole accent ladder beside the token.
	//
	// Nothing of ours reaches the accent fill through here, which is what makes
	// the correction safe: paint.ButtonFace asks for fill.primary by name, and
	// the filled parts.Button is drawn by that face, so this entry is read only
	// by stock widgets.
	//
	// Focus is deliberately NOT the same value: measured on 2026-08-18, a
	// control painted in the colour focus arrives in has nothing left to show
	// that it HAS focus, and the pixels of the first window said so.
	toolkit.ColorNamePrimary: palette.MarkActive,
	toolkit.ColorNameFocus:   palette.MarkCurrent,
}

// unpainted are the colours the toolkit asks for and this window answers with
// nothing at all, each with what disappears.
//
// # Why a register of its own rather than a token
//
// Every token of the palette is a colour somebody can see, and a test holds
// all of them to that, because a transparent token is how a typo becomes an
// invisible control. These are the opposite claim, that the toolkit must not
// paint this anywhere, so they stand apart where the claim can be read.
//
// # 🔴 The shadow, since 2026-11-12 (B-370)
//
// The toolkit lays it along the edge of anything that scrolls and under its
// dialogs. The owner did not want it on the list a kind is picked from, on the
// recipe editor or at the end of a field holding more than it shows, and a
// theme handed to those controls alone is not available: the toolkit takes a
// new theme scope every time it draws such a control again after a minute out
// of sight, and parses every font for the scope anew, about 6 MB of live heap
// each time, kept until the program ends. Measured on 2026-11-12 with the
// in-process probe, four walks of the screens a minute and a half apart held
// 78.6, 111.3, 144.0 and 176.7 MB. So the shadow goes everywhere or nowhere,
// and the owner chose everywhere, having seen both drawn. What that takes with
// it: the shadow at the foot of a scrolling area saying there is more below,
// which the scroll bar beside it still says, and the shadow under the
// toolkit's dialogs, which open over a window they dim.
var unpainted = map[fyne.ThemeColorName]string{
	toolkit.ColorNameShadow: "the shadow of the toolkit, along what scrolls and under its dialogs",
}

// Color answers with a token where this project has one.
//
// The variant is ignored on purpose. There is one theme by decision, so being
// asked for the light one and getting this one is the decision working rather
// than a branch somebody forgot.
func (Visual) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	if _, none := unpainted[name]; none {
		return color.Transparent
	}
	if token, ours := tokenFor[name]; ours {
		return palette.Ink(token)
	}
	// A screen may also ask for a token by its own name, which is what lets a
	// control of the toolkit carry one of our colours without a second table
	// here naming every control.
	if _, ours := palette.TokenOf(string(name)); ours {
		return palette.Ink(string(name))
	}

	// # Why the toolkit's answer is inspected rather than returned
	//
	// Measured on 2026-08-12 and it has not changed: a name the toolkit does
	// not know comes back fully transparent rather than as an error, so a
	// control painted with it is an invisible hole and the cause is nowhere
	// near the symptom. Partial transparency is legitimate, since a shadow is
	// meant to be seen through, so only the fully transparent answer is
	// replaced. It is replaced with text, because a colour nobody claimed
	// should be loud enough to be found. The ban on silence, applied to a pixel.
	fromToolkit := toolkit.DefaultTheme().Color(name, toolkit.VariantDark)
	if _, _, _, alpha := fromToolkit.RGBA(); alpha == 0 {
		return palette.Ink(palette.TextPrimary)
	}
	return fromToolkit
}

// stepFor maps the toolkit's spacing questions onto the closed scale.
var stepFor = map[fyne.ThemeSizeName]string{
	toolkit.SizeNameInnerPadding: palette.NameStepHairline,
	toolkit.SizeNameLineSpacing:  palette.NameStepHairline,
}

// rungFor lets a widget take a size from our ladder by name.
//
// # Why the ladder is reachable at all
//
// widget.Label carries a SizeName since Fyne 2.6 and resolves it through the
// theme, and fyne.ThemeSizeName is a plain string, so a theme may answer to any
// name it likes. That pair is what makes a wrapping paragraph possible at a
// size of ours: canvas text takes a size directly but never wraps, and a label
// wraps but only at a size the theme knows.
//
// The names are the rungs themselves, so nothing here can drift from the ladder.
var rungFor = func() map[fyne.ThemeSizeName]string {
	rungs := make(map[fyne.ThemeSizeName]string, len(palette.TypeNames))
	for _, rung := range palette.TypeNames {
		rungs[fyne.ThemeSizeName(rung)] = rung
	}
	return rungs
}()

func (Visual) Size(name fyne.ThemeSizeName) float32 {
	if rung, ours := rungFor[name]; ours {
		return palette.TypeSize(rung)
	}
	if step, ours := stepFor[name]; ours {
		return palette.StepOf(step)
	}
	switch name {
	// Padding is nought because every gap in this window comes from the
	// scale, through a column or a row that was told which step to use. The
	// toolkit's own padding is a seventh distance nobody chose, and it is
	// applied between everything, which is the fault measured on 2026-09-08:
	// one gap where the drawing has six.
	case toolkit.SizeNamePadding:
		return 0
	// The outline of a text field is a line width rather than a distance
	// between things, and the scale measures the latter. The reason for the
	// number is beside it in the theme, and it is the difference between a
	// mark that exists and one anybody can see.
	case toolkit.SizeNameInputBorder:
		return palette.BorderThickness
	// A separator is a drawn line, so it is as thick as every other line in
	// the window. It was the hairline STEP of the spacing scale until
	// 2026-10-21, which is four units: the rule beside the column of screens
	// and the dividers of every table read as grey bars rather than as lines.
	case toolkit.SizeNameSeparatorThickness:
		return palette.HairlineThickness
	case toolkit.SizeNameText:
		return palette.TypeSize(palette.TypeBody)
	case toolkit.SizeNameInputRadius, toolkit.SizeNameSelectionRadius:
		return palette.RadiusControl
	case toolkit.SizeNameScrollBar:
		return palette.ScrollBarThickness
	case toolkit.SizeNameScrollBarSmall:
		return palette.ScrollBarResting
	case toolkit.SizeNameScrollBarRadius:
		return palette.ScrollBarRadius
	}
	return toolkit.DefaultTheme().Size(name)
}

// faceFor picks one of the embedded faces for a style the toolkit asks about.
//
// # Why the family is embedded rather than asked of the system
//
// A window that asks the operating system for a face gets a different one on
// each platform, and a layout measured against one of them is wrong on the
// other two. The argument for shipping the faces is the argument for pinning
// the toolchain, and the licence for doing so was read on the day the files
// arrived: the note is in the assets package beside them.
func faceFor(style fyne.TextStyle) string {
	switch {
	case style.Monospace && style.Bold:
		return assets.MonoSemiBold
	case style.Monospace:
		return assets.MonoRegular
	case style.Bold && style.Italic:
		return assets.SansSemiItalic
	case style.Bold:
		return assets.SansSemiBold
	case style.Italic:
		return assets.SansItalic
	}
	return assets.SansRegular
}

// faces are built once rather than per call.
//
// The toolkit asks for a face on every text object it lays out, and building a
// resource each time would copy a quarter of a megabyte per label. The map is
// filled at start and only read afterwards, so there is nothing to synchronise:
// the concurrency rule of docs/14 is answered by there being no write.
var faces = func() map[string]fyne.Resource {
	names := []string{
		assets.SansRegular, assets.SansSemiBold, assets.SansItalic,
		assets.SansSemiItalic, assets.MonoRegular, assets.MonoSemiBold,
	}
	built := make(map[string]fyne.Resource, len(names))
	for _, name := range names {
		built[name] = fyne.NewStaticResource(name, assets.Font(name))
	}
	return built
}()

func (Visual) Font(style fyne.TextStyle) fyne.Resource {
	// Symbol is the toolkit's own icon face and is not part of this family.
	// Answering with a text face would draw a letter where an arrow belongs.
	if style.Symbol {
		return toolkit.DefaultTheme().Font(style)
	}
	return faces[faceFor(style)]
}

// Icon hands back this window's own icon where the toolkit draws one inside a
// control, and the toolkit's own everywhere else.
//
// # Why the arrow of a chooser is answered here and not drawn
//
// A chooser draws its own arrow, from whatever the theme answers for
// IconNameArrowDropDown at widget/select.go:441. This window carries nineteen
// icons of one family, and a chooser standing in a row of them with the
// toolkit's arrow beside them is the kind of difference nobody can name and
// everybody sees: another weight, another set, another idea of what an arrow
// looks like. Answering here dresses it without a renderer, which is the whole
// difference between a line of code and a control of our own.
//
// 🔴 This sentence used to say that replacing these would be a claim about
// controls this window does not dress. That was true while nothing here was
// dressed. Dressing the chooser is exactly what changed on 2026-09-09, so the
// sentence went with the fact rather than staying behind it.
//
// 🔴 # Why BOTH arrows and not just the one a chooser reads
//
// Measured rather than reasoned, and the measurement is the reason this comment
// exists. Answering only the downward arrow moved 14 of the 15 states of the
// recipe screen, which has no chooser on it at all. The cause is
// parts.Disclosure: it asks for the downward arrow while it is shut and the
// upward one while it is open, both through this method. Half an answer would
// have given ONE control two families of arrow depending on whether a person
// had opened it, which is worse than the toolkit's own pair and would have
// shown up as nothing anybody could name.
//
// The general form is worth keeping: an icon answered here is answered for
// every control that asks, and the set of controls that ask is not the set the
// change was aimed at. The stored trees are what say which those are.
//
// # What stays the toolkit's, and why that is not laziness
//
// Everything else. An icon inside a control this window has no opinion about
// is better drawn by the library that draws the control, and swapping one for
// the pleasure of it would put our family in places nothing here has looked
// at. A new entry belongs in this switch when a screen shows the control that
// carries it.
func (Visual) Icon(name fyne.ThemeIconName) fyne.Resource {
	switch name {
	case toolkit.IconNameArrowDropDown:
		return paint.IconResource(paint.IconChevron, palette.TextSecondary, palette.IconSmall)
	case toolkit.IconNameArrowDropUp:
		return paint.IconResource(paint.IconChevronUp, palette.TextSecondary, palette.IconSmall)
	}
	return toolkit.DefaultTheme().Icon(name)
}
