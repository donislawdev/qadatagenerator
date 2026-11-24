package takeview

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// watchedWindow is a window that says whether it was asked to close, and does
// not close itself.
//
// The toolkit's own test window carries out a close without asking, so a screen
// that hands the window back at the end of a run cannot be watched through one.
type watchedWindow struct {
	fyne.Window
	closed int
}

func (w *watchedWindow) Close() { w.closed++ }

// heldWatch is the watch for a signal, with the signal itself sent by the test.
//
// Nothing here touches a real signal, which is deliberate: what the screen owes
// is an answer to being asked, and whether the runtime ever asks is a separate
// question with a guard of its own beside the watch itself.
type heldWatch struct {
	asked    func()
	watching int
	released int
}

func (h *heldWatch) watch(asked func()) func() {
	h.asked = asked
	h.watching++
	return func() { h.released++ }
}

// screenThatCanBeAskedToStop is the screen with a run and a watch a test drives.
func screenThatCanBeAskedToStop(t *testing.T, window fyne.Window) (*View, *held, *heldWatch) {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	runs, watch := &held{}, &heldWatch{}
	view := New(i18n.English, window, described, take.Open, background.Work{
		Run: runs.background, Read: background.ReadHere,
		Hold: background.NewClosing(window, watch.watch).Hold,
	}, folderless, nil)
	view.Content()
	return view, runs, watch
}

// The system asking this program to stop ends the run first and gives the
// window up afterwards.
//
// 🔴 This is the sixth unbreakable rule closed in the direction it was open.
// Every file of a run is written under a temporary name and the run holds a
// marker naming what it is about to produce, so a process killed in the middle
// of one leaves both in somebody's output directory and the next run finds the
// marker and refuses for a reason that stopped being true. The command line has
// caught the two signals since the day it could write anything. The window
// caught its own close button and nothing else, so docker stop, a desktop
// logging out and a machine shutting down each cost a person a directory to
// clean by hand. One function, two behaviours, decided by which surface was
// asked.
//
// The order is the whole of it. Being asked to stop is not the same as being
// stopped: the run ends where it chooses to, which is between two records, and
// the window is given up only once it has.
func TestTheSystemAskingThisProgramToStopEndsTheRunFirst(t *testing.T) {
	dir := t.TempDir()
	window := &watchedWindow{Window: test.NewWindow(nil)}
	defer window.Window.Close()
	view, runs, watch := screenThatCanBeAskedToStop(t, window)
	view.Open(sourceFile(t, dir))
	view.delimiter.SetText(separator)
	view.ChooseLayers([]string{orderedLayers})
	view.out.SetText(dir)

	if watch.watching != 0 {
		t.Fatal("the screen is watching for a signal before any run has begun, so it would " +
			"take the two signals away from a person with nothing to clean up")
	}
	view.Start()
	if watch.watching != 1 {
		t.Fatalf("a run began and the screen asked for the watch %d times rather than once, "+
			"so a run stopped by the system would leave what it was writing behind",
			watch.watching)
	}

	watch.asked()
	if window.closed != 0 {
		t.Error("the window was given up while the run was still walking the file, which is " +
			"the process ending in the middle of a write")
	}
	if view.stage != stageStopping {
		t.Errorf("the run is in stage %d rather than being asked to end", view.stage)
	}

	runs.carryOut(t)
	if window.closed != 1 {
		t.Errorf("the run ended and the window was given up %d times rather than once, so "+
			"being asked to stop left the program running", window.closed)
	}
	if watch.released != 1 {
		t.Errorf("the watch was released %d times rather than once, so the two signals stay "+
			"caught outside a run, where there is nothing to take back", watch.released)
	}
}

// A run that ends on its own gives the two signals back.
//
// Outside a run this window has written nothing, so a person killing it is owed
// the instant end their desktop would have given them. A watch left in place
// would swallow that and leave them pressing the key again.
func TestARunThatEndsOnItsOwnGivesTheSignalsBack(t *testing.T) {
	dir := t.TempDir()
	window := test.NewWindow(nil)
	defer window.Close()
	view, runs, watch := screenThatCanBeAskedToStop(t, window)
	view.Open(sourceFile(t, dir))
	view.delimiter.SetText(separator)
	view.ChooseLayers([]string{orderedLayers})
	view.out.SetText(dir)

	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the run ended in stage %d rather than having written its files", view.stage)
	}
	if watch.released != 1 {
		t.Errorf("the run ended and the watch was released %d times rather than once",
			watch.released)
	}
}

// A signal that arrives after the run has already ended gives the window up
// rather than arming the next run to close it.
//
// 🔴 The signal is noticed on a goroutine and acted on where the screen draws,
// so a run ending in between is ordinary rather than exotic. Setting the flag
// that closes the window without looking would leave it set, and the NEXT run
// somebody started would end by taking the window away from them.
func TestASignalThatArrivesAfterTheRunGivesTheWindowUpAtOnce(t *testing.T) {
	dir := t.TempDir()
	window := &watchedWindow{Window: test.NewWindow(nil)}
	defer window.Window.Close()
	view, runs, watch := screenThatCanBeAskedToStop(t, window)
	view.Open(sourceFile(t, dir))
	view.delimiter.SetText(separator)
	view.ChooseLayers([]string{orderedLayers})
	view.out.SetText(dir)

	view.Start()
	asked := watch.asked
	runs.carryOut(t)
	if window.closed != 0 {
		t.Fatal("the window was given up by a run nobody asked to stop")
	}

	asked()
	if window.closed != 1 {
		t.Errorf("a signal arriving after the run had ended gave the window up %d times "+
			"rather than once", window.closed)
	}
	// A directory of its own, because the first run's files would otherwise
	// refuse the second run for a reason this guard is not about.
	view.out.SetText(t.TempDir())
	view.Start()
	if view.stage != stageRunning {
		t.Errorf("the next run is in stage %d rather than going on, so the late signal was "+
			"kept and ended a run somebody started afterwards", view.stage)
	}
}
