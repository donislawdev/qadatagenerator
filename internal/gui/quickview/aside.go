package quickview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// The panel beside the values, which answers the question a person meeting this
// program cannot answer for themselves: what am I actually holding.
//
// # Why it exists, and it is a measurement rather than a decoration
//
// At the size this window is kept, a list of ten nine digit values left about
// four hundred pixels of empty window to the right of it, because a value is
// short and a window is wide. Emptiness is filled with CONTENT or it is not
// filled at all, and the most useful content here is the one thing the values
// themselves cannot say: whether a wrong digit would be caught, and whether the
// number could belong to somebody.
//
// # Why every line of it is derived
//
// The facts come from quick.Describe, which reads what each provider declares.
// Nothing here is a sentence somebody wrote about a scheme, because a sentence
// about a scheme is prose and prose is the one thing in this project that
// nothing checks. What this file decides is only how a fact is worded.

// aside is the whole panel for one kind, or nothing when there is no kind.
func (v *View) aside(generator string) fyne.CanvasObject {
	if generator == "" {
		return nil
	}
	about := quick.Describe(generator, v.locale)

	blocks := []fyne.CanvasObject{
		paint.Row(palette.NameStepTight,
			paint.Middle(paint.Icon(paint.IconBolt, palette.TextAccent, palette.IconMedium)),
			paint.Middle(paint.Strong(about.Generator, palette.TypeSection, palette.TextPrimary))),
		v.fact(i18n.QuickAboutProduces, produces(about)),
	}

	// 🔴 The two claims below are shown only for a kind that HAS a numbering
	// scheme, and that is not a tidy up. A first name has no check digit in the
	// same way it has no colour, so a line saying "None" beside one would
	// answer a question nobody asked and would make the same word mean two
	// different things on one screen.
	if about.Scheme != "" {
		blocks = append(blocks, v.checkDigit(about), v.reservedRange(about))
	}
	if about.Unavailable != "" {
		blocks = append(blocks, v.unavailable(about))
	}

	return paint.AtLeast(palette.AsideWidth, 0,
		paint.Card(palette.NameStepGroup, paint.Column(palette.NameStepGroup, blocks...)))
}

// produces is what this kind makes, and the scheme behind it where there is one.
//
// The words are the vocabulary of the recipe format rather than of this window,
// which is the rule the screen next door already follows for the same reason:
// what a person pastes into a recipe has to read the same in every language.
func produces(about quick.Description) string {
	if about.Scheme == "" {
		return about.Produces
	}
	return about.Produces + " (" + about.Scheme + ")"
}

// checkDigit says whether the number can catch a wrong character on its own.
func (v *View) checkDigit(about quick.Description) fyne.CanvasObject {
	said := i18n.QuickAboutCheckDigitNone
	if about.CheckDigit {
		said = i18n.QuickAboutCheckDigitCarried
	}
	return paint.Column(palette.NameStepHairline,
		paint.Caption(v.text.T(i18n.QuickAboutCheckDigit)),
		v.paragraph(v.text.T(said)))
}

// reservedRange says whether a value produced here could belong to somebody.
//
// 🔴 Both answers are drawn, and the one that is not reassuring is drawn in the
// colour that means look at this. A scheme with a block set aside gets a badge,
// because a badge is what somebody scanning sees, and a scheme without one gets
// a sentence saying exactly that rather than an absence. An absence reads as
// reassurance, and rule nine of the handbook is about this sentence more than
// about any other on the screen.
func (v *View) reservedRange(about quick.Description) fyne.CanvasObject {
	said, mark := i18n.QuickAboutRangeNone, paint.Pill(
		v.text.T(i18n.QuickAboutRange), palette.StateWarning, paint.IconWarn)
	if about.ReservedRange {
		said, mark = i18n.QuickAboutRangeReserved, paint.Pill(
			v.text.T(i18n.QuickAboutHarbour), palette.StateSuccess, paint.IconShield)
	}
	return paint.Column(palette.NameStepHairline,
		paint.Row(palette.NameStepHairline, mark, paint.Spacer()),
		v.paragraph(v.text.T(said)))
}

// unavailable is why this build cannot produce the chosen kind in the chosen
// locale.
//
// The reason comes from the mode rather than from here, so a kind that gains
// data stops saying it has none without anybody editing a sentence.
func (v *View) unavailable(about quick.Description) fyne.CanvasObject {
	return paint.Column(palette.NameStepHairline,
		paint.Caption(v.text.T(i18n.QuickAboutUnavailable)),
		v.paragraph(about.Unavailable))
}

// fact is a caption over one short value, for the lines that are a word rather
// than a sentence.
func (v *View) fact(name i18n.Key, value string) fyne.CanvasObject {
	return paint.Column(palette.NameStepHairline,
		paint.Caption(v.text.T(name)),
		paint.Mono(value, palette.TypeBody, palette.TextPrimary))
}

// paragraph is a sentence that follows the width of the panel rather than
// running off the edge of it.
//
// 🔴 A label rather than the canvas text everything else here is drawn with,
// and the reason is the one thing canvas text cannot do: it does not wrap, and
// it asks for the width of its whole string, so a sentence built that way would
// make this panel as wide as its longest sentence. A label wraps, and it can
// take a size from our own ladder because the theme answers to the names of the
// rungs, which is the whole reason that bridge exists.
//
// The quieter of the two text colours arrives through the toolkit's own name
// for it rather than by being painted here, because a label resolves its colour
// through the theme of the scope it stands in and cannot be told one directly.
func (v *View) paragraph(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
	return container.NewStack(label)
}

// describe puts the panel for the chosen kind on screen.
//
// 🔴 It is called from the place that settles the FIELDS rather than from the
// place that settles the values, and the two are different moments. The panel
// describes the kind, so it must be right the moment a kind is chosen, whether
// or not that choice produced anything. Hanging it off the values instead would
// have left the wrong scheme described beside every refusal, which is exactly
// the state where somebody most needs to read what the kind is.
func (v *View) describe(generator string) {
	drawn := v.aside(generator)
	if drawn == nil {
		v.about.Objects = nil
		v.about.Refresh()
		return
	}
	v.about.Objects = []fyne.CanvasObject{drawn}
	v.about.Refresh()
}
