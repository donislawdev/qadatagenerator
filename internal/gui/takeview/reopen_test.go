package takeview

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

// A separator that is briefly unusable does not take the file away.
//
// # What this is really about
//
// 🔴 The separator is typed one character at a time, and the moment between
// deleting one and typing the next is an empty value the reader refuses. The
// screen answered that refusal the way it answers a file that cannot be opened
// at all, by letting go of the path. The next keystroke then had nothing to
// reopen, so the screen stayed on its empty state for good and a person lost
// their file to a backspace with no way back but choosing it again.
//
// The recovery is the half that matters: keeping the path is only useful if the
// next usable separator brings the file back.
func TestASeparatorThatIsBrieflyUnusableDoesNotTakeTheFileAway(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()
	source := sourceFile(t, t.TempDir())

	view, _ := screenOn(t, window)
	if err := view.Open(source); err != nil {
		t.Fatalf("the screen could not read the file: %v", err)
	}
	view.delimiter.SetText(separator)
	if view.opened == nil {
		t.Fatal("the file is not open before the separator is touched")
	}

	view.delimiter.SetText("")
	if view.path != source {
		t.Fatalf("an unusable separator took the path away, leaving %q", view.path)
	}

	view.delimiter.SetText(separator)
	if view.opened == nil {
		t.Fatal("a usable separator did not bring the file back")
	}
	if view.path != source {
		t.Fatalf("the screen holds %q rather than the file it was given", view.path)
	}
}
