package takeview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// drawFile says what the file holds and how that was arrived at.
//
// # Why the basis of every conclusion is on screen rather than in the manifest
//
// The manifest is where it lives for good, and this is the same information for
// the person deciding whether to go ahead. They are the only reader able to say
// that a conclusion about their own file is wrong, and they can only say it if
// they can see what it rests on.
//
// Nothing here is shortened. A column nothing was recognised in is exactly the
// one worth knowing about, because that is where ordered damage will have had
// nothing to work with.
func (v *View) drawFile() {
	table := v.opened.Table()
	drawn := []fyne.CanvasObject{
		heading(v.text.T(i18n.TakeHeadingFile)),
		widget.NewLabelWithStyle(table.Entity, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		pair(v.text.T(i18n.TakeFileRows), counted(table.Rows, "record")),
		pair(v.text.T(i18n.TakeFileColumns), counted(len(table.Columns), "column")),
		wrapped(v.text.T(i18n.TakeFileNothingDeclared)),
	}
	drawn = append(drawn, v.warnings(len(table.Columns))...)
	drawn = append(drawn, v.conclusions()...)
	// 🔴 Wrapped rather than plain, and the difference is not cosmetic. This is
	// the longest sentence on the screen by a wide margin, and an unwrapped
	// label takes whatever width it wants. Seen on the first render: it ran off
	// the right edge AND pushed every control below it out of the window with
	// it, because a column is as wide as its widest child. One missing wrap
	// made four settings unreachable.
	drawn = append(drawn, wrapped(v.text.T(i18n.TakeFileNotInferred)),
		wrapped(listOf(v.opened.NotInferred())))

	v.fileBox.Objects = drawn
	v.fileBox.Refresh()
}

// warnings are the things worth saying before somebody damages this file.
//
// # Why a single column is called out rather than reported as a number
//
// A file written by a European spreadsheet separates its fields with a
// semicolon, which is the whole reason the separator is a setting at all. Read
// with the wrong one, such a file is one column holding entire lines, and every
// count on this screen is then true and useless. Saying "1 column" and stopping
// there sends somebody off to damage a file this tool never read.
func (v *View) warnings(columns int) []fyne.CanvasObject {
	var said []fyne.CanvasObject
	if columns == 1 {
		said = append(said, v.notice(i18n.TakeFileOneColumn))
	}
	if v.opened.HadByteOrderMark() {
		said = append(said, v.notice(i18n.TakeFileByteOrderMark))
	}
	return said
}

// notice is something the screen wants read rather than scanned past.
//
// It carries an icon as well as its words, because the meaningful colours of
// this palette were measured as differing in hue while barely differing in
// lightness. A notice told apart by colour alone is a notice somebody who does
// not separate hues never sees.
func (v *View) notice(key i18n.Key) fyne.CanvasObject {
	return beside(theme.WarningIcon(), wrapped(v.text.T(key)))
}

// conclusions are what each column was taken to hold, with what that rests on,
// for as many columns as this surface accounts for.
//
// # Why it stops, and what it costs when it does not
//
// One column here is six objects, four of them carrying text, and the panel
// they sit in lays out every one of them whether it is on screen or not.
// Measured on 2026-09-03: 1088 bytes per column, so a file of 5000 columns is
// 5.4 MB of widgets before anything is drawn and a file of 260 000 would be
// about 283 MB. The log describing the same file stops at fifty for the reason
// recorded as DEC-200, and this stops at the same number because it is the same
// conclusions read by the same person. A different number would mean the two
// surfaces disagreed about how much of a file they account for.
//
// 🔴 What is not described is counted rather than left out, which is the whole
// of the difference between a bound and a cut. The scheme count still walks
// every column, because a file where nothing was recognised is a file where
// ordered damage had nothing to work with, and that is true of the columns
// nobody is shown as much as of the ones they are.
func (v *View) conclusions() []fyne.CanvasObject {
	shape := v.opened.Shape()
	findings := shape.Findings()
	var drawn []fyne.CanvasObject
	schemes := 0
	for _, finding := range findings {
		if finding.Scheme != "" {
			schemes++
		}
	}
	described := findings
	if len(described) > take.ColumnsListed {
		described = described[:take.ColumnsListed]
	}
	for _, finding := range described {
		drawn = append(drawn, v.conclusion(finding))
	}
	if left := len(findings) - len(described); left > 0 {
		drawn = append(drawn, wrapped(v.text.T(i18n.TakeFileMoreColumns)+": "+
			strconv.Itoa(left)))
	}
	if schemes == 0 {
		drawn = append(drawn, v.notice(i18n.TakeFileNothingRecognised))
	}
	for _, relation := range shape.Relations() {
		drawn = append(drawn, wrapped(relation.Field+" "+
			v.text.T(i18n.TakeFileAgreesWith)+" "+relation.With+" "+string(relation.Kind)+
			", "+v.text.T(i18n.TakeFileBecause)+" "+relation.Basis))
	}
	return drawn
}

// conclusion is one column, what it was taken to hold and why.
//
// The name of the column and the name of what it holds are data rather than
// interface text, so neither is a key. They are what somebody pastes into a
// report and they read the same in every language.
func (v *View) conclusion(finding infer.Finding) fyne.CanvasObject {
	held := finding.Type
	if finding.Scheme != "" {
		held = finding.Scheme
	}
	name := widget.NewLabelWithStyle(finding.Field, fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true})
	return container.NewVBox(
		container.NewHBox(name, widget.NewLabel(v.text.T(i18n.TakeFileTakenFor)+" "+held)),
		wrapped(v.text.T(i18n.TakeFileBecause)+" "+finding.Basis),
	)
}
