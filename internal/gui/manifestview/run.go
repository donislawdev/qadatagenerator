package manifestview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The run as a whole: its envelope, which the card under the table says in
// place of an entry when somebody asks for it, and what a comparison could not
// see, which the card says the same way. What a comparison says about the
// whole file stands in the band beside the file (DEC-251).
//
// # Why the envelope waits for a press
//
// The seed, the locale and the number of entries stand in the band beside the
// file, and how much of what was ordered happened stands at the heading of the
// table. What is left - when the run was, which catalog and which format,
// which recipe or policy or source, how many rows - is what somebody consults
// to tell one manifest from the one beside it a week later, and not what they
// opened the screen for. Measured on the first render of the mock: a card of
// those pairs over the table left the table three rows at the size the owner
// keeps the window (DEC-247).
//
// # Why in the card and not over the table
//
// Over the table it made the window taller by what it holds, and on the real
// window that was 172 pixels for a manifest and 269 beside a comparison, past
// the 1392 the owner's screen has to give (B-286, docs/17 section 31.4). The
// card is a fixed height that scrolls inside, so what the run holds is no
// longer a height anybody's window has to find (DEC-249).

// runBlocks is the envelope as the card reads it out: what is known about the
// run, or before any manifest the names of what will be known.
func (v *View) runBlocks() []fyne.CanvasObject {
	var blocks []fyne.CanvasObject
	fields := v.screen.Run
	counts := v.screen.Counts
	if !v.hasPromise {
		waiting := Waiting(v.text)
		fields, counts = waiting.Run, waiting.Counts
		blocks = append(blocks, v.paragraph(v.text.T(i18n.ManifestRunWait)))
	}
	return append(blocks, v.envelope(append(append([]Field{}, fields...), counts...)))
}

// envelope lays the pairs out in as many columns as the theme says, label and
// value, so that the labels line up under each other in each column.
//
// # Why a form rather than columns of a fixed width
//
// A form column is as wide as the widest label in it and no wider, worked out
// from the labels themselves. That is what the envelope needs and what an
// earlier attempt got wrong twice: first a single row of eleven pairs, which
// reached the frame of a window two thousand pixels wide and would have been
// cut off in anything narrower, then cells of a fixed size, which trimmed every
// value in the envelope down to an ellipsis.
//
// 🔴 The labels are built out of text that does NOT trim itself, and the
// values out of text that does, and the difference is measured twice. Read in
// the toolkit's source on 2026-08-18: text that may truncate is cut to the
// width the layout gave it, and a form works out the width of its label column
// from the smallest size those same objects report. Labels that trim make the
// two a circle, and the screen it produced showed an envelope of nothing but
// ellipses. Values that do NOT trim overflow the column of the grid they stand
// in: rendered on 2026-09-15, a timestamp ran into the label of the column
// beside it, because the grid divides its width equally and a value is as long
// as somebody's data. So the label column is sized from the labels and the
// value takes what is left of the grid's column, cut with an ellipsis where a
// value does not fit, and the whole value is in the file the screen is of.
func (v *View) envelope(all []Field) fyne.CanvasObject {
	share := (len(all) + palette.EnvelopeColumns - 1) / palette.EnvelopeColumns
	forms := make([]fyne.CanvasObject, 0, palette.EnvelopeColumns)
	for from := 0; from < len(all); from += share {
		to := from + share
		if to > len(all) {
			to = len(all)
		}
		forms = append(forms, form(all[from:to]))
	}
	return container.NewGridWithColumns(palette.EnvelopeColumns, forms...)
}

// form is one column of label and value pairs.
func form(fields []Field) fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, 0, len(fields)+len(fields))
	for _, field := range fields {
		cells = append(cells, plain(Line{Text: field.Label, Color: palette.TextSecondary}, false))
		cells = append(cells, bold(Line{Text: field.Value}))
	}
	return container.New(layout.NewFormLayout(), cells...)
}

// told is a sentence with the shape its ink asks for beside it: a warning
// sign for what needs attention, a check for what agreed, and nothing for a
// plain fact.
//
// The shape rather than the colour on its own, for the reason measured for
// this palette: the colours that carry meaning differ in hue while barely
// differing in lightness, so somebody who does not separate hues sees one
// shade of grey.
func (v *View) told(line Line) fyne.CanvasObject {
	words := v.paragraphInk(line.Text, line.Color)
	switch line.Color {
	case palette.StateDanger:
		return paint.Marked(paint.IconAlert, palette.StateDanger, words)
	case palette.StateWarning:
		return paint.Marked(paint.IconWarn, palette.StateWarning, words)
	case palette.StateSuccess:
		return paint.Marked(paint.IconCheck, palette.StateSuccess, words)
	}
	return words
}

// limitBlocks is what the comparison could not see, as the card reads it out.
//
// # Why in the card and not in a section over the table (DEC-251)
//
// These are sentences about the FORMAT of the response record rather than
// about this run, three of them at two lines each. Over the table they were a
// section that opened in place, and opening it made the window taller by what
// it held, on top of the strip it stood in (B-299). The card is a fixed height
// that scrolls inside, the same answer the envelope of the run got in DEC-249.
func (v *View) limitBlocks() []fyne.CanvasObject {
	blocks := make([]fyne.CanvasObject, 0, len(v.screen.Limits))
	for _, limit := range v.screen.Limits {
		blocks = append(blocks, v.paragraphInk(limit.Text, limit.Color))
	}
	return blocks
}
