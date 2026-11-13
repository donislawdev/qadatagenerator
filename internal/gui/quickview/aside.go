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

	// The name of the kind stands alone at the head of the panel. A bolt
	// stood beside it, the same one every kind wore, so it said nothing about
	// the kind and drew the eye before the facts did. The owner had it removed
	// on 2026-10-20.
	blocks := []fyne.CanvasObject{
		paint.Strong(about.Generator, palette.TypeSection, palette.TextPrimary),
		v.fact(i18n.QuickAboutProduces, produces(about)),
	}
	// Right under what the kind produces, because it narrows that line: the
	// kind is an account number, and the account numbers are Polish (DEC-281).
	if len(about.Countries) > 0 {
		blocks = append(blocks, v.countries(about))
	}

	// 🔴 The two claims below are shown only for a kind that HAS a numbering
	// scheme, and that is not a tidy up. A first name has no check digit in the
	// same way it has no colour, so a line saying "None" beside one would
	// answer a question nobody asked and would make the same word mean two
	// different things on one screen.
	if about.Scheme != "" {
		if len(about.Masks) > 0 {
			blocks = append(blocks, v.format(about))
		}
		blocks = append(blocks, v.checkDigit(about))
		// 🔴 A line before the last block, and it is the only one in this
		// panel. Everything above it is what the scheme IS, and the block
		// below is the one claim this window makes about whether a value could
		// belong to somebody. Rule nine is about that block more than about any
		// other on the screen, so it is worth separating from the facts rather
		// than being the fourth caption down a column.
		blocks = append(blocks, paint.Hairline(palette.LineQuiet), v.reservedRange(about))
	}
	if about.Unavailable != "" {
		blocks = append(blocks, v.unavailable(about))
	}

	// 🔴 The gap to the values beside it is drawn HERE rather than around the
	// stack this goes into, and the difference is a state nobody would look at:
	// the panel is nothing at all when no kind is chosen, and a margin around
	// an empty stack is an empty strip of window with no reason to exist.
	//
	// The step is the one the accepted drawing uses for it, read out of
	// tools/makieta rather than picked to taste. Before this the two stood at
	// ZERO, in all nine states of this screen, which is what the owner saw and
	// what nothing measured until tools/odstepy.py existed: the toolkit's own
	// padding is nought in this theme by decision, so a stock border layout
	// puts two regions edge to edge and says nothing.
	return paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepSection, palette.NameStepHairline,
		paint.AtLeast(palette.AsideWidth, 0,
			paint.Card(palette.NameStepGroup, paint.Column(palette.NameStepGroup, blocks...))))
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

// format is the written form of one value, and how many places in it vary.
//
// # 🔴 Why the mask itself is on screen and not only a sentence about it
//
// Because a person reading ten values beside it is comparing shapes, and a mask
// is the same shape. It says in one line what a sentence needs three for: how
// long, where the punctuation falls, whether letters appear. This was the
// largest of the things the owner found missing here on 2026-09-09.
//
// # Why the sentence under it is built rather than written
//
// The count comes from quick.Places, so the mode reads the mask and this file
// only chooses words for what it read. A sentence written per scheme would be
// eleven claims in a translation file, where nothing could check any of them
// against the values they describe.
//
// A scheme with two forms gets two lines of mask, one under the other and
// shortest first, so that the shapes can be compared as easily as the values.
func (v *View) format(about quick.Description) fyne.CanvasObject {
	lines := []fyne.CanvasObject{paint.Caption(v.text.T(i18n.QuickAboutFormat))}
	for _, mask := range about.Masks {
		lines = append(lines, paint.Mono(mask, palette.TypeValue, palette.TextPrimary))
	}
	lines = append(lines, v.paragraph(v.length(about.Masks)))
	return paint.Column(palette.NameStepHairline, lines...)
}

// countries names the countries a scheme of no one country produces values for,
// one to a line the way the forms of a scheme are, so that no separator has to
// be written in this file.
func (v *View) countries(about quick.Description) fyne.CanvasObject {
	lines := []fyne.CanvasObject{paint.Caption(v.text.T(i18n.QuickAboutCountries))}
	for _, country := range about.Countries {
		lines = append(lines, v.paragraph(v.text.T(countryKey(country))))
	}
	return paint.Column(palette.NameStepHairline, lines...)
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
//
// Built again only when the kind or the locale differs from the last time, the
// two things it reads. Every draw passes through here, and rebuilding it each
// time made nine widgets per draw that the toolkit kept for a minute: measured
// on 2026-11-13 as nine renderers more after every draw of a thousand values
// (B-370).
func (v *View) describe(generator string) {
	asked := describedFor{generator: generator, locale: v.locale}
	if asked == v.described {
		return
	}
	v.described = asked
	drawn := v.aside(generator)
	if drawn == nil {
		v.about.Objects = nil
		v.about.Refresh()
		return
	}
	v.about.Objects = []fyne.CanvasObject{drawn}
	v.about.Refresh()
}

// describedFor is what the panel describing a kind was last drawn for.
type describedFor struct{ generator, locale string }
