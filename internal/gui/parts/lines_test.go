package parts

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// longRecipe is thirty lines, each opening with its own number written as
// "lNN:", every third one long enough to fold under itself in a narrow field,
// and every seventh in Polish letters, which are two bytes each and one
// character. The number in the words is what lets a test know which line a row
// begins without counting anything the way the editor does.
func longRecipe() string {
	var lines []string
	for line := 1; line <= 30; line++ {
		words := fmt.Sprintf("l%02d: value", line)
		if line%3 == 0 {
			words += strings.Repeat(" folded words", 8)
		}
		if line%7 == 0 {
			words += " zażółć gęślą jaźń"
		}
		if line%5 == 0 {
			words = "  " + words
		}
		lines = append(lines, words)
	}
	return strings.Join(lines, "\n")
}

// framedEditor puts a framed editor alone on a canvas of a size, with a text.
func framedEditor(t testing.TB, text string, size fyne.Size) (*Editor, fyne.Window) {
	t.Helper()
	editor := NewEditor(nil)
	window := test.NewWindow(editor.Framed())
	t.Cleanup(window.Close)
	window.SetPadded(false)
	window.Resize(size)
	editor.SetText(text)
	return editor, window
}

// shownRow is a row with its place on the canvas.
type shownRow struct {
	line int
	y    float32
}

// lineOpening reads the number a row of longRecipe opens with, and false for a
// row that continues a line folded above it.
var lineOpening = regexp.MustCompile(`^\s*l(\d\d):`)

// drawnObjects is every object under one, as the renderers hold them now.
//
// 🔴 Not test.LaidOutObjects, which lays every object out again as it walks,
// and that moved the field's scrolled area back to its top: the first run of
// the test below read the rows at the top and found every number of a
// scrolled field wrong.
func drawnObjects(object fyne.CanvasObject) []fyne.CanvasObject {
	objects := []fyne.CanvasObject{object}
	var children []fyne.CanvasObject
	switch held := object.(type) {
	case fyne.Widget:
		children = test.WidgetRenderer(held).Objects()
	case *fyne.Container:
		children = held.Objects
	}
	for _, child := range children {
		objects = append(objects, drawnObjects(child)...)
	}
	return objects
}

// drawnLines is every row the toolkit drew that begins a line and is at least
// partly in sight, read from the toolkit's own texts and their places on the
// canvas, and the height of one row.
func drawnLines(t *testing.T, editor *Editor) ([]shownRow, float32) {
	t.Helper()
	var drawn *widget.RichText
	for _, object := range drawnObjects(editor) {
		if rich, ok := object.(*widget.RichText); ok && len(rich.Segments) == 1 {
			if words, ok := rich.Segments[0].(*widget.TextSegment); ok && words.Text == editor.Text {
				drawn = rich
			}
		}
	}
	if drawn == nil {
		t.Fatal("the toolkit's text of the editor was not found, so there is nothing to compare with")
	}
	driver := fyne.CurrentApp().Driver()
	top := driver.AbsolutePositionForObject(editor.area).Y
	bottom := top + editor.area.Size().Height
	var rows []shownRow
	var height float32
	for _, object := range drawnObjects(drawn) {
		words, ok := object.(*canvas.Text)
		if !ok {
			continue
		}
		height = words.Size().Height
		y := driver.AbsolutePositionForObject(words).Y
		opening := lineOpening.FindStringSubmatch(words.Text)
		if opening == nil || y+height <= top || y >= bottom {
			continue
		}
		line, _ := strconv.Atoi(opening[1])
		rows = append(rows, shownRow{line: line, y: y})
	}
	return rows, height
}

// numbersShown is every number the column draws, with its place on the canvas.
func numbersShown(editor *Editor) []shownRow {
	driver := fyne.CurrentApp().Driver()
	var numbers []shownRow
	for _, object := range editor.marks.holder.Objects {
		number, ok := object.(*canvas.Text)
		if !ok {
			continue
		}
		line, _ := strconv.Atoi(number.Text)
		numbers = append(numbers, shownRow{line: line, y: driver.AbsolutePositionForObject(number).Y})
	}
	return numbers
}

// disagreements says every way the numbers differ from the rows that begin a
// line: a row with no number, a number with no row, a number beside another
// row.
func disagreements(rows, numbers []shownRow) []string {
	at := make(map[int]float32, len(numbers))
	for _, number := range numbers {
		at[number.line] = number.y
	}
	var wrong []string
	for _, row := range rows {
		y, there := at[row.line]
		switch {
		case !there:
			wrong = append(wrong, fmt.Sprintf("line %d is in sight at %.1f and has no number", row.line, row.y))
		case math.Abs(float64(y-row.y)) > 0.5:
			wrong = append(wrong, fmt.Sprintf("line %d stands at %.1f and its number at %.1f", row.line, row.y, y))
		}
		delete(at, row.line)
	}
	for line, y := range at {
		wrong = append(wrong, fmt.Sprintf("number %d at %.1f stands beside no line that begins there", line, y))
	}
	return wrong
}

// agreeing fails the test with every disagreement, after making sure there
// were rows to agree about.
func agreeing(t *testing.T, editor *Editor, when string) {
	t.Helper()
	rows, _ := drawnLines(t, editor)
	if len(rows) < 3 {
		t.Fatalf("%s: only %d lines are in sight, so the numbers are checked against almost nothing", when, len(rows))
	}
	for _, wrong := range disagreements(rows, numbersShown(editor)) {
		t.Errorf("%s: %s", when, wrong)
	}
}

// The number of every line stands beside the first row the toolkit drew for
// it, and nowhere else: not beside the rows of a line folded under itself, and
// not beside a line scrolled out of sight. Asked while the field is scrolled by
// the wheel over the text and over the column, after the caret pulled it to the
// end, and after the field was made narrower, which folds the lines again
// (DEC-337).
//
// The rows are read from the texts the toolkit drew and the numbers each line
// opens with, so nothing here counts or folds the way the editor does.
func TestEveryNumberStandsBesideTheFirstRowOfItsLine(t *testing.T) {
	editor, window := framedEditor(t, longRecipe(), fyne.NewSize(300, 260))
	agreeing(t, editor, "at the top")

	middle := fyne.NewPos(window.Canvas().Size().Width/2, window.Canvas().Size().Height/2)
	test.Scroll(window.Canvas(), middle, 0, -200)
	if editor.area.Offset.Y == 0 {
		t.Fatal("the wheel over the text did not scroll the field, so this says nothing about scrolling")
	}
	agreeing(t, editor, "scrolled by the wheel over the text")

	room := editor.marks.room
	column := fyne.CurrentApp().Driver().AbsolutePositionForObject(room)
	overColumn := column.Add(fyne.NewPos(room.Size().Width/2, room.Size().Height/2))
	before := editor.area.Offset.Y
	test.Scroll(window.Canvas(), overColumn, 0, -150)
	if editor.area.Offset.Y == before {
		t.Error("the wheel over the column of numbers did not scroll the field")
	}
	agreeing(t, editor, "scrolled by the wheel over the column")

	placeCaret(editor, len([]rune(editor.Text)))
	agreeing(t, editor, "with the caret pulled to the end")

	window.Resize(fyne.NewSize(220, 260))
	agreeing(t, editor, "made narrower")

	// With the caret at the very start, folding the lines again leaves the
	// caret where it was, and the toolkit then refreshes nothing (its layout
	// refreshes only when the caret's row moved, widget/entry.go of v2.8.2), so
	// the numbers move only because the editor's own layout puts them back.
	placeCaret(editor, 0)
	window.Resize(fyne.NewSize(300, 260))
	agreeing(t, editor, "made wider with the caret at the start")

	// The area moves back to the top when the text becomes short enough to
	// fit. It says so through OnScrolled before the toolkit calls
	// OnCursorChanged, measured here on 2026-11-24: a copy of the area's own
	// clamp in the column was tried and changed nothing this step can see, so
	// it went, and this step stays to hold the order.
	test.Scroll(window.Canvas(), middle, 0, -300)
	if editor.area.Offset.Y == 0 {
		t.Fatal("the field is not scrolled, so shrinking the text below says nothing")
	}
	editor.SetText(strings.Join(strings.Split(longRecipe(), "\n")[:4], "\n"))
	agreeing(t, editor, "shrunk to fit while scrolled")
}

// The check above can fail: numbers put beside every row as drawn, folded rows
// included, and numbers one row too low both disagree with what was drawn.
func TestTheCheckOfTheNumbersCanFail(t *testing.T) {
	editor, _ := framedEditor(t, longRecipe(), fyne.NewSize(300, 260))
	rows, height := drawnLines(t, editor)
	if len(rows) < 3 {
		t.Fatal("too few lines in sight for a canary")
	}
	low := make([]shownRow, 0, len(rows))
	for _, row := range rows {
		low = append(low, shownRow{line: row.line, y: row.y + height})
	}
	if len(disagreements(rows, low)) == 0 {
		t.Error("numbers one row too low passed")
	}
	extra := append(append([]shownRow{}, rows...), shownRow{line: 99, y: rows[0].y + height})
	if len(disagreements(rows, extra)) == 0 {
		t.Error("a number beside a folded row passed")
	}
	if len(disagreements(rows, rows[1:])) == 0 {
		t.Error("a line with no number passed")
	}
}

// An empty field has one line, numbered, and a text ending in a newline has an
// empty last line, numbered too, which is where the caret goes after Enter.
func TestTheEmptyLastLineHasANumber(t *testing.T) {
	editor, _ := framedEditor(t, "", fyne.NewSize(300, 200))
	if numbers := numbersShown(editor); len(numbers) != 1 || numbers[0].line != 1 {
		t.Errorf("an empty field shows %v, want the number 1 alone", numbers)
	}
	editor.SetText("version: 1\n")
	numbers := numbersShown(editor)
	if len(numbers) != 2 || numbers[1].line != 2 {
		t.Errorf("a text ending in a newline shows %v, want 1 and 2", numbers)
	}
}

// The column is wide enough for three digits from the start, so that the text
// does not move at the tenth line or the hundredth, and wider once the text
// has a thousand lines.
func TestTheColumnGrowsOnlyPastItsLeastDigits(t *testing.T) {
	editor, _ := framedEditor(t, "a", fyne.NewSize(300, 200))
	one := editor.marks.room.MinSize().Width
	editor.SetText(strings.Repeat("a\n", 150))
	if got := editor.marks.room.MinSize().Width; got != one {
		t.Errorf("the column is %.1f wide at 151 lines and %.1f at one", got, one)
	}
	editor.SetText(strings.Repeat("a\n", 1200))
	if got := editor.marks.room.MinSize().Width; got <= one {
		t.Errorf("the column is %.1f wide at 1201 lines, no wider than %.1f at one", got, one)
	}
	if palette.GutterDigitsAtLeast < 3 {
		t.Errorf("the column holds %d digits at least, which moves the text at the tenth or hundredth line",
			palette.GutterDigitsAtLeast)
	}
}

// numberOf is the number the column draws for a line, and false when it draws
// none.
func numberOf(editor *Editor, line int) (*canvas.Text, bool) {
	for _, object := range editor.marks.holder.Objects {
		if number, ok := object.(*canvas.Text); ok && number.Text == strconv.Itoa(line) {
			return number, true
		}
	}
	return nil, false
}

// While the field holds the keyboard, the number of the caret's line is drawn
// in the ink read first and a band lies across every row of that line, folded
// rows included. Without the keyboard, or with words selected, there is no
// band, and every number is in the quieter ink (DEC-337).
func TestTheLineOfTheCaretIsLitOnlyWhileTheFieldHasTheKeyboard(t *testing.T) {
	editor, window := framedEditor(t, longRecipe(), fyne.NewSize(300, 260))
	third := strings.Index(editor.Text, "l03:")
	placeCaret(editor, third+2)
	if _, _, shown := editor.marks.Shown(); shown {
		t.Error("the band is drawn while the field does not hold the keyboard")
	}

	window.Canvas().Focus(editor)
	placeCaret(editor, third+2)
	number, there := numberOf(editor, 3)
	if !there {
		t.Fatal("line 3 has no number")
	}
	if number.Color != palette.Ink(palette.TextPrimary) {
		t.Error("the number of the caret's line is not in the ink read first")
	}
	if other, _ := numberOf(editor, 2); other.Color != palette.Ink(palette.TextSecondary) {
		t.Error("the number of another line is not in the quieter ink")
	}
	top, bottom, shown := editor.marks.Shown()
	if !shown {
		t.Fatal("no band across the caret's line while the field holds the keyboard")
	}
	rows, height := drawnLines(t, editor)
	var three, four float32
	for _, row := range rows {
		switch row.line {
		case 3:
			three = row.y
		case 4:
			four = row.y
		}
	}
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(editor.marks).Y
	if math.Abs(float64(origin+top-three)) > 0.5 || math.Abs(float64(origin+bottom-four)) > 0.5 {
		t.Errorf("the band runs from %.1f to %.1f, want from line 3 at %.1f to line 4 at %.1f, across its %.0f folded rows",
			origin+top, origin+bottom, three, four, (four-three)/height)
	}

	editor.TypedShortcut(&fyne.ShortcutSelectAll{})
	if _, _, shown := editor.marks.Shown(); shown {
		t.Error("the band is drawn under a selection")
	}

	// Selecting everything moved the caret to the end, so the lit number is
	// whichever line that is, and after the keyboard leaves none may be lit.
	window.Canvas().Unfocus()
	if _, _, shown := editor.marks.Shown(); shown {
		t.Error("the band stayed after the keyboard left")
	}
	for _, object := range editor.marks.holder.Objects {
		if number, ok := object.(*canvas.Text); ok && number.Color != palette.Ink(palette.TextSecondary) {
			t.Errorf("number %s stayed lit after the keyboard left", number.Text)
		}
	}
}

// markAt is whether the column draws a mark level with a number.
func markAt(editor *Editor, number *canvas.Text) bool {
	for _, object := range editor.marks.holder.Objects {
		mark, ok := object.(*fyne.Container)
		if !ok {
			continue
		}
		if mark.Position().Y < number.Position().Y+number.Size().Height &&
			mark.Position().Y+mark.Size().Height > number.Position().Y {
			return true
		}
	}
	return false
}

// A line the reader found a problem on carries a mark beside its number, a
// line it did not carries none, and a line the text does not have marks
// nothing.
func TestALineWithAProblemIsMarkedBesideItsNumber(t *testing.T) {
	editor, _ := framedEditor(t, longRecipe(), fyne.NewSize(300, 260))
	editor.MarkLines([]int{2, 99})
	two, _ := numberOf(editor, 2)
	one, _ := numberOf(editor, 1)
	if !markAt(editor, two) {
		t.Error("line 2 has a problem and no mark")
	}
	if markAt(editor, one) {
		t.Error("line 1 has no problem and a mark")
	}
	if got := editor.Marked(); len(got) != 2 {
		t.Errorf("the editor keeps %v as marked, want 2 and 99", got)
	}
	editor.MarkLines(nil)
	if markAt(editor, two) {
		t.Error("the mark stayed after the problems were taken away")
	}
}

// Going to a line puts the caret where its words begin, past the indentation,
// gives the field the keyboard and scrolls the line into sight. A line past
// the end goes to the last line.
func TestGoingToALineBringsTheCaretAndTheLineIntoSight(t *testing.T) {
	editor, window := framedEditor(t, longRecipe(), fyne.NewSize(300, 260))
	editor.GoToLine(25)
	if window.Canvas().Focused() != fyne.Focusable(editor) {
		t.Error("the field did not take the keyboard")
	}
	if got, want := editor.CursorTextOffset(), len([]rune(editor.Text[:strings.Index(editor.Text, "l25:")])); got != want {
		t.Errorf("the caret is at %d, want %d, where the words of line 25 begin", got, want)
	}
	if editor.LineOfCaret() != 25 {
		t.Errorf("the caret is on line %d, want 25", editor.LineOfCaret())
	}
	rows, _ := drawnLines(t, editor)
	inSight := false
	for _, row := range rows {
		inSight = inSight || row.line == 25
	}
	if !inSight {
		t.Error("line 25 is not in sight after going to it")
	}
	editor.GoToLine(500)
	if editor.LineOfCaret() != 30 {
		t.Errorf("going past the end put the caret on line %d, want the last, 30", editor.LineOfCaret())
	}
}

// Enter begins the new line indented like the one it ended, and one indent
// deeper after a line that opens a block.
func TestIndentAfterALine(t *testing.T) {
	for _, each := range []struct {
		line string
		want int
	}{
		{"entities:", 2},
		{"  fields:", 4},
		{"  fields:   # the columns", 4},
		{"    first_name: { generator: first_name }", 4},
		{"version: \"0.1\"", 0},
		{"seed: \"a: b:\"", 0},
		{"name: 'it''s:'", 0},
		{"  - pesel matches: [birth_date, gender]", 2},
		{"  - only_when:", 6},
		{"  -   spaced:", 8},
		{"  # entities:", 2},
		{"", 0},
		{"    ", 4},
		{"key: value # not: a block:", 0},
		{"key: \"quoted \\\" hash # inside\":", 2},
	} {
		if got := indentAfter(each.line); got != each.want {
			t.Errorf("after %q the next line is indented %d, want %d", each.line, got, each.want)
		}
	}
}

// Pressed in the field, Enter types the newline and the indent, wherever the
// caret was on the line, and Ctrl+Z takes back the indent before the newline,
// like the indent of Tab.
func TestEnterIndentsTheNewLineAndCanBeUndone(t *testing.T) {
	editor, _, _, _ := editorBetween(t, "entities:\n  customer:", fyne.NewSize(400, 300))
	placeCaret(editor, len([]rune(editor.Text)))
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if want := "entities:\n  customer:\n    "; editor.Text != want {
		t.Fatalf("Enter after a line opening a block made %q, want %q", editor.Text, want)
	}
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if want := "entities:\n  customer:\n    \n    "; editor.Text != want {
		t.Errorf("Enter on the keypad made %q, want %q", editor.Text, want)
	}

	editor.SetText("seed: x\nlocale: pl-PL")
	placeCaret(editor, len("seed: x"))
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if want := "seed: x\n\nlocale: pl-PL"; editor.Text != want {
		t.Errorf("Enter after a plain line at the left edge made %q, want %q", editor.Text, want)
	}

	editor.SetText("  count: 10")
	placeCaret(editor, len("  count: 10"))
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	for range len("  ") {
		editor.TypedShortcut(&fyne.ShortcutUndo{})
	}
	if want := "  count: 10\n"; editor.Text != want {
		t.Errorf("undoing the indent left %q, want %q", editor.Text, want)
	}
}

// largestRecipe is a recipe at the size the reader accepts, 256 KB, in short
// lines the way a recipe is written, so that the column has the most lines it
// can ever be asked about.
func largestRecipe() string {
	line := "      first_name: { generator: first_name }\n"
	return strings.Repeat(line, 256*1024/len(line))
}

// The cost of putting the column back on a recipe of the largest size, with
// the text changed since the last time (a key press) and unchanged (a scroll),
// beside the cost of the toolkit's own field taking one typed character on the
// same text, so the share of a key press the column takes can be read off
// (DEC-337). Measured with -bench 'Lines|TheToolkitTaking' -count 6.
func BenchmarkLinesAfterAKeyPress(b *testing.B) {
	editor, _ := framedEditor(b, largestRecipe(), fyne.NewSize(500, 700))
	b.ResetTimer()
	for range b.N {
		editor.lines = lineMap{}
		editor.redrawLines()
	}
}

func BenchmarkLinesAfterAScroll(b *testing.B) {
	editor, _ := framedEditor(b, largestRecipe(), fyne.NewSize(500, 700))
	b.ResetTimer()
	for range b.N {
		editor.redrawLines()
	}
}

func BenchmarkTheToolkitTakingOneCharacter(b *testing.B) {
	editor, _ := framedEditor(b, largestRecipe(), fyne.NewSize(500, 700))
	editor.OnCursorChanged = nil
	b.ResetTimer()
	for range b.N {
		editor.Entry.TypedRune('x')
	}
}

// The mouse on the column lights the field as the mouse on its text does, so
// that the ground does not go out while the pointer crosses into the column.
func TestThePointerOnTheColumnLightsTheField(t *testing.T) {
	editor := NewEditor(nil)
	dressed := editor.Framed()
	window := test.NewWindow(dressed)
	t.Cleanup(window.Close)
	editor.marks.MouseIn(&desktop.MouseEvent{})
	if !showsFill(dressed, palette.Ink(palette.BgHover)) {
		t.Error("the pointer on the column did not light the field")
	}
	editor.marks.MouseOut()
	if showsFill(dressed, palette.Ink(palette.BgHover)) {
		t.Error("the field stayed lit after the pointer left the column")
	}
}
