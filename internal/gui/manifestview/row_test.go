package manifestview

import (
	"image/color"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Measuring a piece of text asks the running application how wide it is, so a
// cell laid out without one is a nil dereference rather than a quiet no
// operation. The window has an application by the time any cell is drawn, and a
// test has to say so.
func TestMain(m *testing.M) {
	application := test.NewApp()
	// The theme of this product rather than the one the toolkit tests with.
	// Colours are asked for by name here, and the test theme answers "color
	// text not defined" for every name of this palette, so a rendering checked
	// under it would be a rendering of something the window never shows.
	application.Settings().SetTheme(fynetheme.Dark{})
	os.Exit(m.Run())
}

// drawn collects every phrase an object is actually showing.
//
// The widget tree is read rather than the model, which is the whole point of
// this file. The model beside it says what the screen ought to say, and nothing
// but this connects that to what a person sees.
func drawn(object fyne.CanvasObject) []string {
	switch typed := object.(type) {
	case *widget.RichText:
		var found []string
		for _, segment := range typed.Segments {
			if text, ok := segment.(*widget.TextSegment); ok {
				found = append(found, text.Text)
			}
		}
		return found
	case *fyne.Container:
		var found []string
		for _, child := range typed.Objects {
			found = append(found, drawn(child)...)
		}
		return found
	}
	return nil
}

func shows(object fyne.CanvasObject, want string) bool {
	for _, one := range drawn(object) {
		if one == want {
			return true
		}
	}
	return false
}

func sample() Row {
	return Row{
		Mutator:    "L3.checksum.off_by_one",
		Place:      "customer   row 1   pesel",
		Status:     Line{Text: "applied", Color: palette.NameTextSubdued},
		Expect:     Line{Text: "should be rejected", Color: palette.NameRejection},
		Confidence: Line{Text: "hard expectation", Color: palette.NameTextSubdued},
		Values:     Line{Text: `was "44051401359"   now "44051401358"`, Color: palette.NameCorrupted},
	}
}

// Every part of an entry reaches a column, and the columns are the ones the
// headings name.
//
// 🔴 The pairing is what matters. A table asks the row for a cell by number and
// asks this package for a heading by the same number, so a column added in one
// place and not the other puts the heading of one thing above the values of
// another and goes on compiling.
func TestEveryPartOfAnEntryReachesTheColumnItsHeadingNames(t *testing.T) {
	entry := sample()
	cells := entry.Cells()
	if len(cells) != palette.EntryColumns() {
		t.Fatalf("an entry hands out %d cells and the table has %d columns", len(cells), palette.EntryColumns())
	}
	if len(columns) != palette.EntryColumns() {
		t.Fatalf("%d columns are named and the table has %d", len(columns), palette.EntryColumns())
	}
	if len(palette.EntryColumnWidths) != palette.EntryColumns() {
		t.Fatalf("%d widths are set and the table has %d columns",
			len(palette.EntryColumnWidths), palette.EntryColumns())
	}

	for _, want := range []string{
		entry.Mutator, entry.Place, entry.Status.Text,
		entry.Expect.Text, entry.Confidence.Text, entry.Values.Text,
	} {
		found := false
		for _, cell := range cells {
			if strings.Contains(cell.Text, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("no column of the table carries %q", want)
		}
	}
	// Every heading resolves to a sentence rather than to its own key, and so
	// does the sentence saying what that column holds. Both, because the state
	// with no manifest shows the second one and nothing else would notice a
	// column added with a heading and no explanation of it.
	for column := range columns {
		view := &View{text: i18n.English}
		if name := view.columnName(column); name == "" || strings.Contains(name, ".") {
			t.Errorf("column %d is headed %q", column, name)
		}
		holds := i18n.English.T(columns[column].Holds)
		if holds == "" || strings.Contains(holds, ".wait.") {
			t.Errorf("column %d does not say what it holds, it says %q", column, holds)
		}
	}
}

// A cell taken from the table shows this entry and nothing left over from the
// one before it.
//
// 🔴 This is the failure a virtualised table has and a column of widgets does
// not. One widget is reused for many cells, so a member left alone still holds
// what the previous entry put there, and a stale reason would look exactly like
// a fact about the entry now on screen. Nothing about that is visible in a
// review, and a screenshot of a short manifest would never show it, because the
// reuse only starts once there are more entries than cells on screen.
func TestAReusedCellCarriesNothingFromTheEntryBeforeIt(t *testing.T) {
	cell := painted(Line{})
	set(cell, Line{Text: "the column follows no scheme this damage suits", Color: palette.NameNotApplied}, fyne.TextStyle{})
	set(cell, Line{Text: "applied", Color: palette.NameTextSubdued}, fyne.TextStyle{})

	if shows(cell, "the column follows no scheme this damage suits") {
		t.Error("the cell still shows what the entry before it put there")
	}
	if !shows(cell, "applied") {
		t.Error("the cell does not show the entry now in it")
	}
}

// The one column allowed to carry prose carries the reason where there is one
// and what changed where there is not.
//
// A reason belongs to an entry that did not happen and values belong to one
// that did, so the two never occur together. The answer of an application joins
// the values, because it is about the same thing.
func TestTheDetailColumnCarriesTheReasonOrWhatChanged(t *testing.T) {
	reason := Line{Text: "this build does not carry that damage yet", Color: palette.NameNotApplied}
	answer := Line{Text: "the application accepted this row", Color: palette.NameTextSubdued}

	if got := (Row{Reason: reason}).Detail(); got.Text != reason.Text {
		t.Errorf("an unapplied entry shows %q in the detail column", got.Text)
	}
	if got := (Row{Answer: answer}).Detail(); got.Text != answer.Text {
		t.Errorf("an applied entry shows %q in the detail column", got.Text)
	}
	if got := (Row{}).Detail(); got.Text != "" {
		t.Errorf("an entry with neither shows %q in the detail column", got.Text)
	}

	values := Line{Text: `was "1250"   now "n/a"`, Color: palette.NameCorrupted}
	both := (Row{Values: values, Answer: answer}).Detail()
	for _, want := range []string{values.Text, answer.Text} {
		if !strings.Contains(both.Text, want) {
			t.Errorf("the detail column reads %q and drops %q", both.Text, want)
		}
	}
}

// A column nobody defined answers with nothing rather than taking the view down.
func TestAColumnThatDoesNotExistIsEmptyRatherThanFatal(t *testing.T) {
	for _, absent := range []int{-1, palette.EntryColumns()} {
		if got := sample().Cell(absent); got.Text != "" {
			t.Errorf("column %d answered %q", absent, got.Text)
		}
	}
}

// A filled cell actually draws something.
//
// 🔴 This is the guard for the failure that got all the way to a running window
// with every test green. Sixty rows appeared with their dividers drawn and not a
// word in any of them, because the row was a type that merely held a container:
// it satisfied the toolkit's interface, so it compiled and a test walking the
// tree found all its text, but the toolkit recognises a container by its
// concrete type and never descended into the children. So this one does not
// read the tree. It renders and asks whether any pixel differs from the ground.
func TestAFilledCellDrawsSomethingRatherThanAnEmptyRectangle(t *testing.T) {
	cell := painted(Line{})
	set(cell, sample().Expect, fyne.TextStyle{})

	window := test.NewWindow(cell)
	defer window.Close()
	window.Resize(fyne.NewSize(900, 200))

	image := window.Canvas().Capture()
	bounds := image.Bounds()
	backgroundR, backgroundG, backgroundB, _ := image.At(bounds.Min.X, bounds.Min.Y).RGBA()
	painted := 0
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := image.At(x, y).RGBA()
			if r != backgroundR || g != backgroundG || b != backgroundB {
				painted++
			}
		}
	}
	if painted == 0 {
		t.Fatal("a cell carrying a phrase rendered as an empty rectangle, so what a test reads off " +
			"the widget and what a person sees on the screen have come apart")
	}
}

// A phrase in the envelope asks for the width its words need, and one in a cell
// of the table does not.
//
// 🔴 This is the guard for a failure that reached a screenshot twice, in two
// different disguises, and that no test of content can see: the words are all
// present in the widget either way, and only the width tells the two cases
// apart. Text that may truncate is cut to the width the layout gave it, and a
// form works out its label column from the smallest size those same objects
// report. Put one inside the other and the envelope renders as a row of
// ellipses, which is exactly what it did.
func TestAPhraseInTheEnvelopeAsksForTheWidthItsWordsNeed(t *testing.T) {
	short := plain(Line{Text: "Seed"}, false)
	long := plain(Line{Text: "Seed of a run with a considerably longer name than that"}, false)
	if long.MinSize().Width <= short.MinSize().Width {
		t.Errorf("a long phrase asks for %v and a short one for %v, so the envelope does not size "+
			"itself from its own words and will be cut down to nothing",
			long.MinSize().Width, short.MinSize().Width)
	}

	// A cell of the table is the opposite case and has to be willing to be
	// narrow, because its column width is set rather than asked for.
	cell := painted(Line{Text: "L4.coherence.national_id_vs_birthdate"})
	if cell.Truncation != fyne.TextTruncateEllipsis {
		t.Error("a cell of the table does not trim itself, so a long value will run over the column beside it")
	}
}

// A phrase with no colour of its own is painted the ordinary text colour rather
// than left transparent.
func TestAPhraseWithNoColourIsStillVisible(t *testing.T) {
	segment, ok := painted(Line{Text: "plain"}).Segments[0].(*widget.TextSegment)
	if !ok {
		t.Fatal("a phrase is not built out of a piece of text at all")
	}
	if got := string(segment.Style.ColorName); got != palette.NameText {
		t.Errorf("a phrase with no colour named asks the theme for %q rather than for the text colour", got)
	}
}

// Every colour this screen asks the theme for is one the theme answers with the
// measured value for.
//
// 🔴 The bridge that closes here is easy to miss. A colour is a name by the time
// it reaches the toolkit, and a name the toolkit does not recognise comes back
// fully transparent rather than as an error, so the control is an invisible hole
// and the cause is nowhere near the symptom.
func TestTheThemeAnswersForEveryColourThisScreenAsksFor(t *testing.T) {
	dark := fynetheme.Dark{}
	for _, name := range palette.ColorNames {
		want := palette.Named(name).NRGBA()
		got := dark.Color(fyne.ThemeColorName(name), toolkit.VariantDark)
		if got != color.Color(want) {
			t.Errorf("the theme answers %v for %q and the palette measured %v", got, name, want)
		}
	}
}
