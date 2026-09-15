package manifestview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

// drawDetail fills the card with the one thing it says.
func (v *View) drawDetail() {
	var blocks []fyne.CanvasObject
	switch {
	case v.reading != "":
		blocks = []fyne.CanvasObject{v.paragraph(v.text.T(i18n.ManifestReading))}
	case v.unread != nil:
		// The sentence names the failure and the failure names itself, so the
		// two together say whose fault it is without this screen deciding.
		blocks = []fyne.CanvasObject{v.told(Line{
			Text: v.text.T(i18n.ManifestUnreadable) + " " + fault.Says(v.unread), Color: palette.StateDanger})}
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
