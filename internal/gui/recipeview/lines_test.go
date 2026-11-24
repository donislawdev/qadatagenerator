package recipeview_test

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/gui/screens"
)

// rowsToTap is every row on the screen a click acts on, from the top.
func rowsToTap(object fyne.CanvasObject) []*parts.TapRow {
	var rows []*parts.TapRow
	if row, ok := object.(*parts.TapRow); ok {
		rows = append(rows, row)
	}
	var children []fyne.CanvasObject
	switch held := object.(type) {
	case fyne.Widget:
		children = test.WidgetRenderer(held).Objects()
	case *fyne.Container:
		children = held.Objects
	}
	for _, child := range children {
		rows = append(rows, rowsToTap(child)...)
	}
	return rows
}

// The lines a problem names are marked beside their numbers while the answer
// stands, a click on the problem puts the caret on its line with the keyboard
// in the editor, and a clean answer takes the marks away (DEC-337).
//
// The canary is the screen before a problem: no marks and no row to click,
// so the counts after it are about the problem and not about the screen.
func TestAClickOnAProblemTakesTheCaretToItsLine(t *testing.T) {
	view := here(t, real(t))
	window := test.NewWindow(view.Content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(1120, 780))

	view.SetRecipe(screens.CleanRecipe)
	if marked := view.Marked(); len(marked) != 0 {
		t.Fatalf("a clean recipe has lines marked: %v", marked)
	}
	if rows := rowsToTap(view.Content()); len(rows) != 0 {
		t.Fatalf("a clean recipe has %d rows to click before any problem, so the count below says nothing", len(rows))
	}

	view.SetRecipe(screens.BrokenRecipe)
	report, _ := view.Report()
	var lines []int
	for _, found := range report.Problems {
		if found.Line > 0 {
			lines = append(lines, found.Line)
		}
	}
	if len(lines) == 0 {
		t.Fatal("the broken sample names no line, so there is nothing to go to")
	}
	if marked := view.Marked(); !slices.Equal(marked, lines) {
		t.Errorf("the lines marked are %v, want those the problems name, %v", marked, lines)
	}
	rows := rowsToTap(view.Content())
	if len(rows) != len(lines) {
		t.Fatalf("%d rows can be clicked, want one per problem naming a line, %d", len(rows), len(lines))
	}
	if view.LineOfCaret() == lines[0] {
		t.Fatalf("the caret is already on line %d, so a click could not be told from no click", lines[0])
	}
	test.Tap(rows[0])
	if view.LineOfCaret() != lines[0] {
		t.Errorf("the click put the caret on line %d, want %d", view.LineOfCaret(), lines[0])
	}
	if _, ok := window.Canvas().Focused().(*parts.Editor); !ok {
		t.Errorf("the keyboard is on %T after the click, want the editor", window.Canvas().Focused())
	}

	view.SetRecipe(screens.CleanRecipe)
	if marked := view.Marked(); len(marked) != 0 {
		t.Errorf("the marks stayed after a clean answer: %v", marked)
	}
}
