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
//
// 🔴 The real theme rather than the default test one, and not only for looks:
// this screen is built out of paint, which asks the theme for its type sizes by
// name, and the default test theme answers a name it does not know with nought.
// A label of size nought divides by zero in the text renderer three layouts on,
// which is why nothing here uses test.NewTempApp.
func TestMain(m *testing.M) {
	application := test.NewApp()
	application.Settings().SetTheme(fynetheme.Current())
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
		Mutator:   "L3.checksum.off_by_one",
		Place:     "customer   row 1   pesel",
		Applied:   true,
		Status:    Line{Text: "applied", Color: palette.TextSecondary},
		Expect:    Line{Text: "should be rejected"},
		Certainty: Line{Text: "should be rejected, and the certainty is hard.", Color: palette.TextSecondary},
		Values:    Line{Text: `was "44051401359"   now "44051401358"`, Color: palette.TextAccent},
	}
}

// Every part of an entry reaches a column or the card under the table, and the
// columns are the ones the headings name.
//
// 🔴 The pairing is what matters. A table asks the row for a cell by number and
// asks this package for a heading by the same number, so a column added in one
// place and not the other puts the heading of one thing above the values of
// another and goes on compiling. And a part of an entry that reaches neither
// the table nor the card is a fact the manifest carries and nobody sees.
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

	for _, want := range []string{entry.Mutator, entry.Place, entry.Status.Text, entry.Expect.Text} {
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
	for _, want := range []string{entry.Certainty.Text, entry.Values.Text} {
		found := false
		for _, line := range entry.Detail() {
			if strings.Contains(line.Text, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("neither the table nor the card under it carries %q", want)
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
	set(cell, Line{Text: "no column of that kind", Color: palette.StateWarning}, fyne.TextStyle{})
	set(cell, Line{Text: "applied", Color: palette.TextSecondary}, fyne.TextStyle{})

	if shows(cell, "no column of that kind") {
		t.Error("the cell still shows what the entry before it put there")
	}
	if !shows(cell, "applied") {
		t.Error("the cell does not show the entry now in it")
	}
}

// The outcome column says what became of the damage, and the card under the
// table carries what the columns do not.
//
// 🔴 The outcome is one column with two halves and this is the guard on the
// seam: a damage that did not happen shows its kind of failure whatever any
// application said, one that happened shows the application's verdict once
// there is one, and "applied" until then. The card carries the reason of one
// and the values and finding of the other, so an entry with none of them shows
// an empty card rather than somebody else's.
func TestTheOutcomeColumnAndTheCardDivideAnEntryBetweenThem(t *testing.T) {
	kind := Line{Text: "not built yet", Color: palette.StateWarning}
	reason := Line{Text: "Not applied: not built yet. This build does not carry that damage.", Color: palette.StateWarning}
	answer := Line{Text: "accepted", Color: palette.StateDanger}
	finding := Line{Text: "the application accepted this row   accepted although the manifest expected a rejection",
		Color: palette.StateDanger}
	values := Line{Text: `was "1250"   now "n/a"`, Color: palette.TextAccent}

	unapplied := Row{Applied: false, Status: kind, Reason: reason, Answer: answer}
	if got := unapplied.Outcome(); got.Text != kind.Text {
		t.Errorf("an unapplied entry shows %q in the outcome column rather than its kind of failure", got.Text)
	}
	applied := Row{Applied: true, Status: Line{Text: "applied"}}
	if got := applied.Outcome(); got.Text != "applied" {
		t.Errorf("an applied entry with no response shows %q in the outcome column", got.Text)
	}
	judged := Row{Applied: true, Status: Line{Text: "applied"}, Answer: answer, Finding: finding, Values: values}
	if got := judged.Outcome(); got.Text != answer.Text {
		t.Errorf("a judged entry shows %q in the outcome column rather than the application's verdict", got.Text)
	}

	if got := unapplied.Detail(); len(got) != 1 || got[0].Text != reason.Text {
		t.Errorf("the card reads an unapplied entry out as %v rather than as its reason", got)
	}
	for _, want := range []string{values.Text, finding.Text} {
		found := false
		for _, line := range judged.Detail() {
			if line.Text == want {
				found = true
			}
		}
		if !found {
			t.Errorf("the card reads a judged entry out without %q", want)
		}
	}
	if got := (Row{}).Detail(); len(got) != 0 {
		t.Errorf("an entry with nothing beyond its cells reads out %v", got)
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
	if got := string(segment.Style.ColorName); got != palette.TextPrimary {
		t.Errorf("a phrase with no colour named asks the theme for %q rather than for the text token", got)
	}
}

// Every token this screen asks the theme for is one the theme answers with the
// measured value for.
//
// 🔴 The bridge that closes here is easy to miss. A colour is a name by the time
// it reaches the toolkit, and a name the toolkit does not recognise comes back
// fully transparent, or as the text colour by this theme's own repair, so a
// phrase painted with it is either a hole or the wrong colour and the cause is
// nowhere near the symptom. Measured on 2026-09-15 before this screen moved to
// the tokens: the names of the old palette were not tokens of the theme the
// window wears, and "~ soft" stood at a contrast of 1.68.
func TestTheThemeAnswersForEveryColourThisScreenAsksFor(t *testing.T) {
	theme := fynetheme.Current()
	asked := map[string]bool{palette.TextPrimary: true}
	for _, screen := range everyKindOfScreen(t) {
		for _, line := range lines(screen) {
			if line.Color != "" {
				asked[line.Color] = true
			}
		}
	}
	if len(asked) < len([]string{palette.TextSecondary, palette.TextAccent, palette.StateWarning,
		palette.StateDanger}) {
		t.Fatalf("the screens built here ask for %d colours, too few to have reached every kind of entry", len(asked))
	}
	for name := range asked {
		if _, ours := palette.TokenOf(name); !ours {
			t.Errorf("the screen paints a line in %q, which is not a token of the theme, so the theme answers "+
				"with whatever it answers for a name it does not know", name)
			continue
		}
		want := palette.Ink(name)
		got := theme.Color(fyne.ThemeColorName(name), toolkit.VariantDark)
		if got != color.Color(want) {
			t.Errorf("the theme answers %v for %q and the register measured %v", got, name, want)
		}
	}
}

// lines is every coloured phrase a screen carries, wherever it stands.
func lines(screen Screen) []Line {
	all := append(append(append([]Line{}, screen.Notes...), screen.Unsettled...), screen.Limits...)
	for _, row := range screen.Rows {
		all = append(all, row.Status, row.Expect, row.Certainty, row.Reason, row.Values, row.Answer, row.Finding)
	}
	return all
}
