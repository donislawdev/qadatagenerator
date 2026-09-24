package manifestview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The card under the table reads out one entry in full.
//
// # Why a card rather than a column of the table
//
// The sentences an entry carries - why it could not be applied, the value
// before and after, what the application did and how that compares - are as
// long as somebody's data, and a column for them is a judgement about how much
// to show. The column that held them had 720 units in a table the window hands
// 828, so it was never on the screen (DEC-247). Here they have the width of the
// window to wrap in, one entry at a time, and the table keeps to what fits in a
// cell.
//
// # What it says when there is no entry to read out
//
// What to do. Point eight of the GUI checklist: an empty card reads as a fault
// in the program, and this one is the first thing under the table a person
// meets on opening the screen.
//
// # One thing at a time, and a heading saying which (DEC-249)
//
// The card reads out an entry OR the envelope of the run, never both, and the
// heading over it names the one it holds. It is the card of the take screen
// again, which holds the plan of a run or what the last one did (DEC-245):
// both together were taller than the room there was, and there as here the
// heading is what keeps a person from reading one as the other. Choosing an
// entry, by a click or by the keyboard, puts the entry back.

// drawDetail fills the card with the one thing it says, and its heading with
// what that is.
func (v *View) drawDetail() {
	v.drawCardHeading()
	var blocks []fyne.CanvasObject
	switch {
	case v.reading != "":
		blocks = []fyne.CanvasObject{v.paragraph(v.text.T(i18n.ManifestReading))}
	case v.unread != nil:
		// The sentence names the failure and the failure names itself, so the
		// two together say whose fault it is without this screen deciding.
		blocks = []fyne.CanvasObject{v.told(Line{
			Text: v.text.T(i18n.ManifestUnreadable) + " " + fault.Says(v.unread), Color: palette.StateDanger})}
	case v.showing == cardRun:
		blocks = v.runBlocks()
	case v.showing == cardLimits:
		blocks = v.limitBlocks()
	case !v.hasPromise:
		blocks = []fyne.CanvasObject{v.paragraph(v.text.T(i18n.ManifestDetailWait))}
	case !v.hasSelected || v.selected >= len(v.screen.Rows):
		blocks = []fyne.CanvasObject{v.paragraph(v.text.T(i18n.ManifestDetailSelect))}
	default:
		blocks = v.entry(v.selected, v.screen.Rows[v.selected])
	}
	v.detailBox.Objects = []fyne.CanvasObject{paint.Column(palette.NameStepHairline, blocks...)}
	v.detailBox.Refresh()
}

// drawCardHeading names what the card holds, and labels each button beside the
// name with where a press takes it.
//
// The heading is built again only when its title changes or the button for
// the limits comes or goes, because this runs on every arrow key and both
// change only when the card switches what it holds or a comparison is opened.
// It stands a group's step under the table, which is the room the card had
// above it before it had a heading, and that step is above it only: padded on
// the sides as well, its words stood four pixels in from the heading of the
// table on the first render.
//
// # Why the limits have a button of their own and only beside a comparison
//
// What a comparison could not see is a third thing the card can hold (DEC-251).
// Its button exists only while there is such a comparison, since a button that
// leads to nothing is a stop for the keyboard and a promise the screen breaks.
// Its label carries the count, so how much is there is known without pressing.
func (v *View) drawCardHeading() {
	title := i18n.ManifestHeadingEntry
	aboutSays, limitsSays := i18n.ManifestHeadingRun, i18n.ManifestHeadingLimits
	switch v.showing {
	case cardRun:
		title, aboutSays = i18n.ManifestHeadingRun, i18n.ManifestShowEntry
	case cardLimits:
		title, limitsSays = i18n.ManifestHeadingLimits, i18n.ManifestShowEntry
	}
	if label := v.text.T(aboutSays); v.about.Text != label {
		v.about.SetText(label)
	}
	limitLabel := v.text.T(limitsSays)
	if limitsSays == i18n.ManifestHeadingLimits {
		limitLabel += " (" + strconv.Itoa(len(v.screen.Limits)) + ")"
	}
	if v.seeLimits.Text != limitLabel {
		v.seeLimits.SetText(limitLabel)
	}
	withLimits := v.hasResponse && len(v.screen.Limits) > 0
	if title == v.cardTitle && withLimits == v.headedLimit && len(v.cardHeading.Objects) > 0 {
		return
	}
	v.cardTitle, v.headedLimit = title, withLimits
	buttons := fyne.CanvasObject(v.about)
	if withLimits {
		buttons = paint.Row(palette.NameStepTight, v.seeLimits, v.about)
	}
	above := paint.Gap(palette.NameStepGroup)
	v.cardHeading.Objects = []fyne.CanvasObject{container.New(layout.NewBorderLayout(above, nil, nil, nil),
		above, headed(v.text.T(title), buttons))}
	v.cardHeading.Refresh()
}

// entry is one entry read out: the damage and where it landed with its number,
// then every sentence the entry carries beyond its four cells.
func (v *View) entry(at int, row Row) []fyne.CanvasObject {
	place := between + row.Place + between +
		v.text.T(i18n.ManifestDetailEntry) + " " + strconv.Itoa(at+1) + " " +
		v.text.T(i18n.ManifestFactsOf) + " " + strconv.Itoa(len(v.screen.Rows))
	blocks := []fyne.CanvasObject{paint.Row(palette.NameStepTight,
		paint.Strong(row.Mutator, palette.TypeBody, palette.TextPrimary),
		paint.Text(place, palette.TypeBody, palette.TextSecondary))}
	for _, line := range row.Detail() {
		blocks = append(blocks, v.told(line))
	}
	return blocks
}

// cut is a sentence on one line, cut with an ellipsis where the slot ends,
// for the band: a piece of canvas text would ask for its whole width and take
// it from what stands beside it.
func (v *View) cut(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapOff
	label.Truncation = fyne.TextTruncateEllipsis
	label.Importance = widget.LowImportance
	label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
	return label
}

// paragraph is a sentence that wraps to the width it is given, in the type of
// the body and the ink of an explanation.
//
// A label of the toolkit's rather than paint.Text, because paint.Text is a
// piece of canvas and never wraps, and the sentences here are among the
// longest on any screen of this window.
func (v *View) paragraph(words string) fyne.CanvasObject {
	return v.paragraphInk(words, palette.TextSecondary)
}

// paragraphInk is a wrapping sentence in the ink the model asked for.
//
// 🔴 The ink is a colour name the theme answers for, given to the label by
// way of a rich text segment rather than a label's importance, because a label
// knows five importances and this screen needs the warning and danger tokens
// on a wrapping sentence. A rich text wraps like a label does.
func (v *View) paragraphInk(words, ink string) fyne.CanvasObject {
	if ink == "" || ink == palette.TextPrimary {
		label := widget.NewLabel(words)
		label.Wrapping = fyne.TextWrapWord
		label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
		return container.NewStack(label)
	}
	if ink == palette.TextSecondary {
		label := widget.NewLabel(words)
		label.Wrapping = fyne.TextWrapWord
		label.Importance = widget.LowImportance
		label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
		return container.NewStack(label)
	}
	text := widget.NewRichText(segmentOf(Line{Text: words, Color: ink}))
	text.Wrapping = fyne.TextWrapWord
	return container.NewStack(text)
}
