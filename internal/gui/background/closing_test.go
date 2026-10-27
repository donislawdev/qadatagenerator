package background

import "testing"

// askedWindow is a window that says what was asked of it and carries none of
// it out, so that a test can press the cross the way the toolkit would.
type askedWindow struct {
	intercept func()
	closed    int
}

func (w *askedWindow) SetCloseIntercept(intercept func()) { w.intercept = intercept }
func (w *askedWindow) Close()                             { w.closed++ }

// askedWatch is the watch for a signal, with the signal sent by the test.
type askedWatch struct {
	asked    func()
	watching int
	released int
}

func (w *askedWatch) watch(asked func()) func() {
	w.asked = asked
	w.watching++
	return func() { w.released++ }
}

// run is one run holding the window, which counts how often it was asked to
// stop.
type run struct {
	stopped int
	letGo   func()
}

func heldBy(closing *Closing) *run {
	held := &run{}
	held.letGo = closing.Hold(func() { held.stopped++ })
	return held
}

// 🔴 A run on one screen ending does not give the window back while a run on
// another is still writing.
//
// Until 2026-10-27 each screen took the cross over at the start of its run and
// gave it back at the end, on the same window, so the run that ended first left
// the other one to be cut off by the next press of the cross, in the middle of
// a write, with its temporary files and its marker left behind.
func TestTheWindowIsHeldUntilTheLastRunLetsGo(t *testing.T) {
	window, watch := &askedWindow{}, &askedWatch{}
	closing := NewClosing(window, watch.watch)
	if window.intercept != nil || watch.watching != 0 {
		t.Fatal("the window is held before anything has asked for it")
	}

	first, second := heldBy(closing), heldBy(closing)
	if watch.watching != 1 {
		t.Errorf("two runs set the watch up %d times rather than once", watch.watching)
	}
	first.letGo()
	if window.intercept == nil {
		t.Fatal("the first run to end gave the cross back while the second is still writing")
	}
	if watch.released != 0 {
		t.Error("the first run to end stopped the watch while the second is still writing")
	}

	second.letGo()
	if window.intercept != nil {
		t.Error("the last run ended and the cross is still taken over")
	}
	if watch.released != 1 {
		t.Errorf("the last run ended and the watch was released %d times rather than once",
			watch.released)
	}
	if window.closed != 0 {
		t.Error("the window was given up although nobody asked for it")
	}
}

// Asking for the window stops every run going on, and the window goes once the
// last of them has stopped.
func TestAskingForTheWindowStopsEveryRunAndGoesAfterTheLast(t *testing.T) {
	window := &askedWindow{}
	closing := NewClosing(window, nil)
	first, second := heldBy(closing), heldBy(closing)

	window.intercept()
	if first.stopped != 1 || second.stopped != 1 {
		t.Fatalf("the cross asked the runs to stop %d and %d times rather than once each",
			first.stopped, second.stopped)
	}
	first.letGo()
	if window.closed != 0 {
		t.Fatal("the window was given up while the second run was still writing")
	}
	second.letGo()
	if window.closed != 1 {
		t.Errorf("both runs ended and the window was given up %d times rather than once",
			window.closed)
	}
}

// A run begun after somebody asked to leave is stopped at once.
//
// 🔴 The ask may be a container being stopped, which kills the process a few
// seconds later whatever it is doing. A run that outlives the ask is the run
// that gets killed in the middle of a write.
func TestARunBegunAfterTheAskToLeaveIsStoppedAtOnce(t *testing.T) {
	window := &askedWindow{}
	closing := NewClosing(window, nil)
	first := heldBy(closing)
	window.intercept()

	late := heldBy(closing)
	if late.stopped != 1 {
		t.Fatalf("a run begun after the ask to leave was asked to stop %d times rather than "+
			"once, so the window waits for it", late.stopped)
	}
	first.letGo()
	late.letGo()
	if window.closed != 1 {
		t.Errorf("both runs ended and the window was given up %d times rather than once",
			window.closed)
	}
}

// A signal that arrives once nothing is held gives the window up at once, and
// leaves nothing set for the next run.
//
// 🔴 A signal is noticed on a goroutine and acted on where the window draws, so
// a run ending in between is ordinary rather than exotic. Remembering the ask
// anyway would stop the NEXT run somebody started.
func TestASignalAfterTheLastRunGivesTheWindowUpAndArmsNothing(t *testing.T) {
	window, watch := &askedWindow{}, &askedWatch{}
	closing := NewClosing(window, watch.watch)
	first := heldBy(closing)
	asked := watch.asked
	first.letGo()

	asked()
	if window.closed != 1 {
		t.Fatalf("a signal arriving after the run gave the window up %d times rather than once",
			window.closed)
	}
	next := heldBy(closing)
	if next.stopped != 0 {
		t.Error("the late signal was kept and stopped a run somebody started afterwards")
	}
}

// Letting go twice ends no other run's holding.
//
// A screen that lets go on the way out of a run and again on the way out of
// the window is ordinary, and a second letting go that took somebody else's
// entry would give the window back under a run still writing.
func TestLettingGoTwiceEndsNoOtherHolding(t *testing.T) {
	window := &askedWindow{}
	closing := NewClosing(window, nil)
	first, second := heldBy(closing), heldBy(closing)

	first.letGo()
	first.letGo()
	if window.intercept == nil {
		t.Fatal("the first run let go twice and the second run lost the window it was holding")
	}
	second.letGo()
	if window.intercept != nil {
		t.Error("the last run let go and the cross is still taken over")
	}
}
