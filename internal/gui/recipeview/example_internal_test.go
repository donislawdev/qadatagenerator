package recipeview

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
)

// This one is inside the package because it presses the button rather than
// calling what the button calls. What a person does is tap it, and a guard that
// reaches past the control would go on passing the day the control stopped
// being wired to anything.

// 🔴 The first thing somebody meeting this screen can press has to leave a
// recipe that runs. An example drawing a list of problems in text nobody wrote
// is worse than no example at all, and nothing else would notice: it is a
// string, and a string compiles whatever is in it.
//
// The check is the one the window is really given, so this asks the same
// question the screen asks while a person types.
func TestTheExampleThisScreenOffersIsARecipeThatRuns(t *testing.T) {
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("this build cannot read its own catalog of damage: %v", err)
	}
	view := New(i18n.English, Checker(described), nil,
		background.Work{Read: background.ReadHere}, nil)
	window := test.NewWindow(view.Content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(900, 700))

	test.Tap(view.example)

	if !view.checked {
		t.Fatal("pressing the button judged nothing, so the example never reached the field")
	}
	if problems := view.answer.Report.Problems; len(problems) > 0 {
		t.Fatalf("the example this screen offers has %d problems in it: %v",
			len(problems), problems)
	}
	if view.answer.Refused != nil {
		t.Fatalf("the example reads correctly and cannot be run: %v", view.answer.Refused)
	}
	if len(view.answer.Sample.Tables) == 0 {
		t.Fatal("the example produces no rows, so the screen under it is an empty preview")
	}
	// 🔴 The keyboard goes into the field, because the button it was on has
	// left the screen. Without this a person who got here by keyboard is
	// standing on nothing, and the next Tab starts the walk again from the top.
	if focused := window.Canvas().Focused(); focused != view.editor {
		t.Errorf("after the example arrived the keyboard is on %T rather than in the field",
			focused)
	}
}
