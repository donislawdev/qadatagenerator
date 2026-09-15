package manifestview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The section over the table: the envelope of the run, closed until somebody
// opens it, and under it the sentences a comparison says about the run as a
// whole.
//
// # Why the envelope is closed by default
//
// The seed, the locale and the number of entries stand in the band beside the
// file, and how much of what was ordered happened stands at the heading of the
// table. What is left - when the run was, which catalog and which format,
// which recipe or policy or source, how many rows - is what somebody consults
// to tell one manifest from the one beside it a week later, and not what they
// opened the screen for. Measured on the first render of the mock: a card of
// those pairs over the table left the table three rows at the size the owner
// keeps the window (DEC-247).

// drawRun fills the envelope with what is known about the run, and the strip
// under it with what a comparison says about the whole file.
func (v *View) drawRun() {
	blocks := []fyne.CanvasObject{}
	fields := v.screen.Run
	counts := v.screen.Counts
	if !v.hasPromise {
		waiting := Waiting(v.text)
		fields, counts = waiting.Run, waiting.Counts
		blocks = append(blocks, v.paragraph(v.text.T(i18n.ManifestRunWait)))
	}
	blocks = append(blocks, v.envelope(append(append([]Field{}, fields...), counts...)))
	v.runBox.Objects = []fyne.CanvasObject{paint.Column(palette.NameStepRelated, blocks...)}
	v.runBox.Refresh()

	v.notesBox.Objects = nil
	if v.hasResponse {
		v.notesBox.Objects = []fyne.CanvasObject{paint.Pad(palette.NameStepRelated,
			palette.NameStepHairline, palette.NameStepHairline, palette.NameStepHairline, v.notes())}
	}
	v.notesBox.Refresh()
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

// notes are the sentences of a comparison about the run as a whole, each with
// the shape its colour asks for, and under them the section holding what the
// comparison could not see.
//
// The shape rather than the colour on its own, for the reason measured for
// this palette: the colours that carry meaning differ in hue while barely
// differing in lightness, so somebody who does not separate hues sees one
// shade of grey.
func (v *View) notes() fyne.CanvasObject {
	lines := make([]fyne.CanvasObject, 0, len(v.screen.Notes)+1)
	for _, note := range v.screen.Notes {
		lines = append(lines, v.told(note))
	}
	if len(v.screen.Limits) > 0 {
		lines = append(lines, v.limits())
	}
	return paint.Column(palette.NameStepTight, lines...)
}

// told is a sentence with the shape its ink asks for beside it: a warning
// sign for what needs attention, a check for what agreed, and nothing for a
// plain fact.
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

// limits is the section holding what the comparison could not see, closed
// until somebody opens it.
//
// Closed, because these are sentences about the FORMAT of the response record
// rather than about this run, three of them at two lines each, and every line
// above the table is a row of the table gone. The count in the heading says
// how much is there without opening it.
func (v *View) limits() fyne.CanvasObject {
	lines := make([]fyne.CanvasObject, 0, len(v.screen.Limits))
	for _, limit := range v.screen.Limits {
		lines = append(lines, v.paragraphInk(limit.Text, limit.Color))
	}
	title := v.text.T(i18n.ManifestHeadingLimits) + " (" + strconv.Itoa(len(v.screen.Limits)) + ")"
	return parts.NewDisclosure(title, paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepHairline, palette.NameStepHairline,
		paint.Card(palette.NameStepGroup, paint.Column(palette.NameStepHairline, lines...)))).OnScreen()
}
