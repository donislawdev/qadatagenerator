package paint_test

import (
	"testing"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestAControlWithoutALabelStandsLevelWithOneThatHasIt holds the band of every
// screen that has one: a field under its label and a button beside it with no
// label start at the same height.
//
// # 🔴 What went wrong, and why no test saw it
//
// The room left where a label is missing was read off the type ladder, the
// label's type size plus the hairline step. A line of text is taller than its
// type size, so on the live window of 2026-09-24 the buttons of the recipe band
// stood 4 units above the field beside them (B-310). Every stored tree recorded
// that faithfully, because a stored tree records what happened rather than what
// a row promises. This asks the promise: where the control lands once the row
// is laid out, which is the effect and not the presence of a helper.
//
// The words are two lengths on purpose. A label is one line of canvas text and
// its height must not depend on what it says, and a guard written with one
// label could not tell that apart from a room that happened to fit it.
func TestAControlWithoutALabelStandsLevelWithOneThatHasIt(t *testing.T) {
	application := test.NewTempApp(t)
	application.Settings().SetTheme(fynetheme.Visual{})

	for _, words := range []string{"Write to", "Extra manifest form"} {
		named := canvas.NewRectangle(nil)
		bare := canvas.NewRectangle(nil)
		row := paint.Row(theme.NameStepRelated,
			paint.FieldLabel(words, named), paint.Unlabelled(bare))
		row.Resize(row.MinSize())

		if named.Position().Y != bare.Position().Y {
			t.Errorf("under the label %q the control starts at %v, and the control with no "+
				"label beside it at %v, so the band is not level",
				words, named.Position().Y, bare.Position().Y)
		}
		// A room of nothing would be level only with a label of nothing, and
		// the label above is real words, so this cannot be passed by dropping
		// the room altogether.
		if bare.Position().Y == 0 {
			t.Error("the control with no label starts at the top of its column, so no room " +
				"was left where the label would be")
		}
	}
}

// TestTheRoomForAMissingLabelIsTheHeightOfALine holds why the helper exists: the
// size of the type is NOT the room a label takes, and the day it is, this test
// says so and the reason for the helper has to be written again.
func TestTheRoomForAMissingLabelIsTheHeightOfALine(t *testing.T) {
	application := test.NewTempApp(t)
	application.Settings().SetTheme(fynetheme.Visual{})

	line := paint.Text("Write to", theme.TypeLabel, theme.TextSecondary).MinSize().Height
	if size := theme.TypeSize(theme.TypeLabel); line <= size {
		t.Errorf("a label line is %v tall and its type %v, so reading the room off the "+
			"ladder would be right after all", line, size)
	}
	if got := paint.Unlabelled(canvas.NewRectangle(nil)).MinSize().Height; got < line {
		t.Errorf("the room left for a missing label is %v, shorter than the line %v it stands for",
			got, line)
	}
}
