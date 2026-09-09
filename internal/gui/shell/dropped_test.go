package shell

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// dropTarget records what a screen was handed, so the routing can be checked
// without a window and without anything reading a disk.
type dropTarget struct{ got [][]fyne.URI }

func (d *dropTarget) take(dropped []fyne.URI) { d.got = append(d.got, dropped) }

func screensWithTargets(t *testing.T) (*Shell, *dropTarget, *dropTarget) {
	t.Helper()
	first, second := &dropTarget{}, &dropTarget{}
	return New([]Screen{
		{Name: "first", Content: widget.NewLabel("first"), OnDropped: first.take},
		{Name: "second", Content: widget.NewLabel("second"), OnDropped: second.take},
		{Name: "third", Content: widget.NewLabel("third")},
	}, Brand{}), first, second
}

func dropped() []fyne.URI {
	return []fyne.URI{storage.NewFileURI("/data/contacts.csv")}
}

// A dropped file reaches the screen a person is looking at.
//
// # Why this is the whole subject of the guard
//
// The toolkit takes one drop callback for the entire window, so every screen
// that wanted files would be overwriting the last one to register. Routing it
// here is what makes several screens possible, and routing it wrongly is
// invisible: dropping a file while reading About and having another screen
// quietly act on it looks, from inside that screen, exactly like working.
func TestADroppedFileReachesTheScreenOnDisplay(t *testing.T) {
	frame, first, second := screensWithTargets(t)

	frame.Dropped(fyne.NewPos(10, 10), dropped())
	if len(first.got) != 1 {
		t.Fatalf("the screen on display was handed %d drops rather than one", len(first.got))
	}
	if len(second.got) != 0 {
		t.Errorf("a screen nobody was looking at was handed %d drops, so a file dropped "+
			"while reading one screen would be acted on by another", len(second.got))
	}

	frame.Open(1)
	frame.Dropped(fyne.NewPos(10, 10), dropped())
	if len(second.got) != 1 {
		t.Errorf("after moving to the second screen it was handed %d drops rather than "+
			"one, so the frame is routing by something other than what is open",
			len(second.got))
	}
	if len(first.got) != 1 {
		t.Errorf("the screen no longer on display was handed %d drops, so it goes on "+
			"acting on files after somebody has moved away from it", len(first.got))
	}
}

// A screen that takes no files ignores a drop rather than failing.
//
// Dropping a file onto the About screen is an ordinary mistake, and the answer
// to it is nothing happening. A screen with no handler must not be a crash.
func TestAScreenThatTakesNoFilesIgnoresADrop(t *testing.T) {
	frame, _, _ := screensWithTargets(t)
	frame.Open(2)
	frame.Dropped(fyne.NewPos(0, 0), dropped())
}

// An empty drop reaches nobody.
//
// The toolkit reports a drop with no files in some situations, and a screen
// handed an empty list would be asked to open the first of nothing.
func TestADropOfNoFilesReachesNobody(t *testing.T) {
	frame, first, _ := screensWithTargets(t)
	frame.Dropped(fyne.NewPos(0, 0), nil)
	if len(first.got) != 0 {
		t.Errorf("a drop carrying no file was handed to a screen anyway, which asks it " +
			"to act on the first of nothing")
	}
}
