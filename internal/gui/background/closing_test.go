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

// handAnswered is a question a test answers by hand: it counts how often it was
// asked and keeps the two answers of the last asking.
type handAnswered struct {
	asked      int
	goOn, stay func()
}

func (q *handAnswered) ask(goOn, stay func()) {
	q.asked++
	q.goOn, q.stay = goOn, stay
}

// 🔴 With something to lose, the cross asks before the window goes, and the
// window goes only on the answer that lets it.
//
// Until 2026-11-25 the cross ended the window at once whenever nothing was
// being written, and a recipe typed and never saved went with it.
func TestTheCrossAsksBeforeTheWindowGoes(t *testing.T) {
	window, asking := &askedWindow{}, &handAnswered{}
	closing := NewClosing(window, nil)
	closing.AskFirst(asking.ask)
	if window.intercept == nil {
		t.Fatal("a question stands and the cross is still the toolkit's, so it would close without asking")
	}

	window.intercept()
	if asking.asked != 1 || window.closed != 0 {
		t.Fatalf("the cross asked %d times and closed the window %d times, want one question and no close",
			asking.asked, window.closed)
	}
	asking.goOn()
	if window.closed != 1 {
		t.Errorf("the answer letting the window go closed it %d times rather than once", window.closed)
	}
}

// Keeping the window keeps it, and the next press of the cross asks again.
func TestStayingKeepsTheWindowAndTheNextCrossAsksAgain(t *testing.T) {
	window, asking := &askedWindow{}, &handAnswered{}
	closing := NewClosing(window, nil)
	closing.AskFirst(asking.ask)

	window.intercept()
	asking.stay()
	if window.closed != 0 {
		t.Fatal("the answer keeping the window closed it")
	}
	window.intercept()
	if asking.asked != 2 {
		t.Errorf("the cross pressed after staying asked %d times in all rather than twice", asking.asked)
	}
}

// A second press of the cross while the question is open asks nothing.
func TestASecondCrossWhileAskingAsksNothing(t *testing.T) {
	window, asking := &askedWindow{}, &handAnswered{}
	closing := NewClosing(window, nil)
	closing.AskFirst(asking.ask)

	window.intercept()
	window.intercept()
	if asking.asked != 1 {
		t.Fatalf("two presses opened the question %d times", asking.asked)
	}
	asking.goOn()
	asking.goOn()
	if window.closed != 1 {
		t.Errorf("one answer given twice closed the window %d times", window.closed)
	}
}

// The question comes after the runs have ended, so that a save still going on
// has finished, well or not, by the time anybody is asked what there is to keep.
func TestTheQuestionComesAfterTheRunsHaveEnded(t *testing.T) {
	window, asking := &askedWindow{}, &handAnswered{}
	closing := NewClosing(window, nil)
	closing.AskFirst(asking.ask)
	held := heldBy(closing)

	window.intercept()
	if held.stopped != 1 || asking.asked != 0 {
		t.Fatalf("the cross stopped the run %d times and asked %d times, want the run first",
			held.stopped, asking.asked)
	}
	held.letGo()
	if asking.asked != 1 || window.closed != 0 {
		t.Fatalf("the run ended, the question was asked %d times and the window closed %d times",
			asking.asked, window.closed)
	}
	asking.goOn()
	if window.closed != 1 {
		t.Errorf("the window closed %d times rather than once", window.closed)
	}
	if window.intercept == nil {
		t.Error("the run let go and gave the cross back, though a question still stands")
	}
}

// 🔴 A signal asks nothing. It comes from a system that will kill the process a
// few seconds later whatever it is shown.
func TestASignalAsksNothing(t *testing.T) {
	window, watch, asking := &askedWindow{}, &askedWatch{}, &handAnswered{}
	closing := NewClosing(window, watch.watch)
	closing.AskFirst(asking.ask)
	held := heldBy(closing)

	// A person pressed the cross first and the system asked after them.
	window.intercept()
	watch.asked()
	held.letGo()
	if asking.asked != 0 {
		t.Errorf("a signal ended in a question asked %d times", asking.asked)
	}
	if window.closed != 1 {
		t.Errorf("the signal closed the window %d times rather than once", window.closed)
	}
}

// A question that saves before letting the window go keeps the window until the
// save it started has ended.
func TestAQuestionThatSavesKeepsTheWindowUntilTheSaveEnds(t *testing.T) {
	window := &askedWindow{}
	closing := NewClosing(window, nil)
	var saving func()
	closing.AskFirst(func(goOn, _ func()) {
		saving = closing.Hold(func() {})
		goOn()
	})

	window.intercept()
	if window.closed != 0 {
		t.Fatal("the window went while the save the question started was still going on")
	}
	saving()
	if window.closed != 1 {
		t.Errorf("the save ended and the window closed %d times rather than once", window.closed)
	}
}
