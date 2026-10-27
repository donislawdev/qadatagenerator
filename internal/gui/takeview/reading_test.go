package takeview

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// heldReads is a place for a read to happen that does not happen until a test
// says so.
//
// It keeps what the screen handed over instead of carrying it out, which is
// what lets a test stand in the middle of a read and then answer the reads in
// whatever order it chooses. Everything it runs is the real thing: the same
// closure the window hands to the same reader.
type heldReads struct {
	waiting []func() func()
}

func (h *heldReads) read(read func() func(), _ func(error) func()) {
	h.waiting = append(h.waiting, read)
}

// answer carries out the read asked for nth, counting from zero, and hands the
// screen what it found.
func (h *heldReads) answer(t *testing.T, nth int) {
	t.Helper()
	if nth >= len(h.waiting) {
		t.Fatalf("the screen asked for %d reads and this wants the one numbered %d",
			len(h.waiting), nth)
	}
	h.waiting[nth]()()
}

// screenWaitingOnAFile is the screen with a read that goes nowhere until a test
// says so.
func screenWaitingOnAFile(t *testing.T, window fyne.Window) (*View, *heldReads, *held) {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	reads, runs := &heldReads{}, &held{}
	view := New(i18n.English, window, described, take.Open, background.Work{
		Run: runs.background, Read: reads.read,
	}, nil)
	view.Content()
	return view, reads, runs
}

// Reading a file happens somewhere other than the thread that draws, and the
// screen says so while it waits.
//
// 🔴 It did not until 2026-09-01. take.Open walks the whole of somebody else's
// file, and it was called where the file chooser answered, which is the thread
// the toolkit draws on. Measured next door on 2026-08-26 at 11.7 seconds for a
// hundred thousand rows of four columns, and a file this mode exists for is
// bigger than that. For all of it the window repainted nothing, answered no
// key, and offered nothing to press. The file beside this one had spelled out
// exactly why a RUN may not happen there, and nobody had applied the same
// sentence to the read that comes before it.
//
// The second half is point eight of the GUI checklist. A window that has not
// answered yet and a window with nothing to say look alike, so the one that is
// working has to say it.
func TestReadingAFileHappensSomewhereOtherThanTheThreadThatDraws(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()
	source := sourceFile(t, t.TempDir())

	view, reads, _ := screenWaitingOnAFile(t, window)
	view.Open(source)

	if len(reads.waiting) != 1 {
		t.Fatalf("choosing a file asked for %d reads at the place of work rather than one, "+
			"so the file is being read where the choice was made", len(reads.waiting))
	}
	if view.opened != nil {
		t.Error("the screen already holds the file, so the read happened where it was asked " +
			"for and the place of work was handed nothing worth doing")
	}
	if said := texts(view.Content()); !saying(said, i18n.English.T(i18n.TakeReading)) {
		t.Error("the screen does not say that it is reading, so a person waiting on a large " +
			"file cannot tell it from a screen that has nothing to say")
	}

	reads.answer(t, 0)
	if view.opened == nil {
		t.Fatal("the read answered and the screen does not hold the file")
	}
	if said := texts(view.Content()); saying(said, i18n.English.T(i18n.TakeReading)) {
		t.Error("the screen still says it is reading after the read has answered")
	}
}

// Nothing damages a file while the description of it is being replaced.
//
// 🔴 The separator is read again on every keystroke, so a read in flight is the
// ordinary state of this screen while somebody corrects a comma to a semicolon.
// A run begun in that moment would damage the file it was told about a moment
// ago and write a manifest saying it had been told something else. Neither the
// file nor the manifest would look wrong afterwards.
func TestNoRunBeginsWhileTheFileIsBeingReadAgain(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()
	dir := t.TempDir()

	view, reads, runs := screenWaitingOnAFile(t, window)
	view.Open(sourceFile(t, dir))
	reads.answer(t, 0)
	view.ChooseLayers([]string{orderedLayers})
	view.out.SetText(dir)
	view.delimiter.SetText(separator)
	reads.answer(t, 1)
	if view.damage.Disabled() {
		t.Fatal("the button that damages the file is disabled before the read that this " +
			"guard is about, so it proves nothing")
	}

	// Another separator, one character and perfectly legal, which reads the
	// file again.
	//
	// 🔴 The first shape of this used two characters, and two characters are
	// refused by the mode on their own one line further down in Start. So the
	// run was stopped by something other than the read and the mutation aimed
	// at this guard went uncaught. A guard has to be red for its own reason.
	view.delimiter.SetText(",") // punctuation: test data
	if !view.damage.Disabled() {
		t.Error("the button that damages the file can be pressed while the file is being " +
			"read again, so a run would use a description the screen is replacing")
	}
	view.Start()
	if runs.work != nil {
		t.Error("a run began while the file was being read again")
	}
}

// An answer about a file nobody is waiting for any more is not drawn.
//
// 🔴 Reads are superseded rather than refused, and this is the half that makes
// that safe. A separator typed while the file is being read makes that read an
// answer to a question nobody is asking any more, and drawing it would put the
// file as it was read under the previous separator on a screen showing the
// current one, which is the screen describing a file nobody has.
func TestAnAnswerAboutAFileNobodyIsWaitingForIsNotDrawn(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()
	dir := t.TempDir()
	source := sourceFile(t, dir)

	view, reads, _ := screenWaitingOnAFile(t, window)
	view.Open(source)
	reads.answer(t, 0)
	view.delimiter.SetText(separator)
	reads.answer(t, 1)
	columns := len(view.opened.Table().Columns)
	// The separator that reads the file as one column, and while that read
	// goes on, the one that reads it as three again.
	view.delimiter.SetText(",") // punctuation: test data
	view.delimiter.SetText(separator)
	if len(reads.waiting) != 3 {
		t.Fatalf("the screen asked for %d reads rather than the three this guard walks",
			len(reads.waiting))
	}

	reads.answer(t, 2)
	if now := len(view.opened.Table().Columns); now != columns {
		t.Errorf("an answer about the file as it was read a keystroke ago was drawn, "+
			"leaving the screen describing %d columns under a separator that gives %d",
			now, columns)
	}
	if !view.readingNow() {
		t.Error("the screen stopped saying it is reading while the read about the separator " +
			"on screen has not even begun")
	}
	if len(reads.waiting) != 4 {
		t.Fatalf("the read about the separator on screen was never asked for, so the screen "+
			"would wait for ever: %d reads asked", len(reads.waiting))
	}
	reads.answer(t, 3)
	if view.readingNow() {
		t.Error("the screen still says it is reading after the read it was waiting for answered")
	}
}

// A separator typed while the file is being read waits for that read rather
// than reading the file again beside it.
//
// 🔴 Every character of the separator reads the whole file, measured at 11.7
// seconds for a hundred thousand rows, and until 2026-10-27 every one of them
// started at once and ran to its end, side by side. Correcting one separator
// to another by selecting it and typing three characters walked the file three
// times at the same moment, and two of the answers were thrown away.
func TestASeparatorTypedWhileTheFileIsReadWaitsForTheRead(t *testing.T) {
	window := test.NewWindow(nil)
	defer window.Close()
	source := sourceFile(t, t.TempDir())

	view, reads, _ := screenWaitingOnAFile(t, window)
	view.Open(source)
	reads.answer(t, 0)
	for _, typed := range []string{"|", ",", ":", separator} { // punctuation: test data
		view.delimiter.SetText(typed)
	}
	if len(reads.waiting) != 2 {
		t.Fatalf("four separators typed during one read asked for %d reads at the place of "+
			"work rather than the one in flight, so each of them walks the file beside it",
			len(reads.waiting)-1)
	}
	reads.answer(t, 1)
	if len(reads.waiting) != 3 {
		t.Fatalf("the read that ended left %d more behind it rather than the one about the "+
			"separator on screen", len(reads.waiting)-2)
	}
	reads.answer(t, 2)
	if columns := len(view.opened.Table().Columns); columns != 3 {
		t.Errorf("the screen describes %d columns, so the read it kept was not the one about "+
			"the separator on screen", columns)
	}
}

// Both panes of this screen can be read without a mouse.
//
// 🔴 The toolkit's own scrolling area answers to the wheel and to nothing else,
// so a pane wrapped in one is content the keyboard cannot reach at all. Measured
// on the About screen on 2026-08-24, where the same absence left 73.6 percent of
// the screen behind a mouse. This screen has two of them and is the taller: the
// description of a file grows with the number of columns in it.
//
// The order is part of the promise. What the file holds is read before what to
// do about it, and the walk has to agree with the eye.
func TestBothPanesOfThisScreenCanBeReadWithoutAMouse(t *testing.T) {
	view, _, canvas := onACanvas(t)

	// One lap of the walk and no more, because the second lap would meet the
	// same panes again and say the screen has four of them.
	canvas.FocusNext()
	first := canvas.Focused()
	var reached []fyne.Focusable
	for pressed := 1; pressed <= walkLength; pressed++ {
		canvas.FocusNext()
		if canvas.Focused() == first {
			break
		}
		if _, isScroll := canvas.Focused().(*parts.KeyScroll); isScroll {
			reached = append(reached, canvas.Focused())
		}
	}
	// One pane since 2026-09-15 (DEC-245): the settings stand in a column that
	// fits the window and the description of the file is the one thing that
	// grows with what it describes, so it is the one thing that scrolls.
	if len(reached) != 1 {
		t.Fatalf("the keyboard reached %d scrolling panes of this screen in a whole walk of "+
			"it, and the panel saying what the file holds is the one there is", len(reached))
	}
	if reached[0] != fyne.Focusable(view.fileScroll) {
		t.Error("the scrolling pane the keyboard reaches is not the one saying what the file holds")
	}
}
