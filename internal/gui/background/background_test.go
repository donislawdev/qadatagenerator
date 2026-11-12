package background

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Releasing a watch twice does not take the program down.
//
// 🔴 This is not a hypothetical. The watch is released when a run ends and
// again when the window is given up, and closing a closed channel is a panic.
// A panic in a window is not a message on a terminal, it is the program
// disappearing while somebody was working, and it would happen at exactly the
// moment the program was trying to leave cleanly.
func TestReleasingAWatchTwiceIsSafe(t *testing.T) {
	release := WatchForClosing(func() {})
	release()
	release()
}

// A stopper answers the same thing however many times it is asked.
//
// 🔴 The button that asks is on screen until the run answers, so it takes the
// two presses a person makes when nothing seems to have happened. A second ask
// that closed the channel again would be a panic, and one that sent instead of
// closing would leave the run told, on its next record, that it may carry on.
func TestAStopperAnswersTheSameThingHoweverOftenItIsAsked(t *testing.T) {
	halt := NewStopper()
	if err := halt.Stopped(); err != nil {
		t.Fatalf("a run nobody has asked to stop was told to end with %v", err)
	}

	halt.Stop()
	halt.Stop()
	for asked := 1; asked <= 3; asked++ {
		err := halt.Stopped()
		if err == nil {
			t.Fatalf("asked %d times after being stopped, the run was told it may carry on",
				asked)
		}
		if !errors.Is(err, ErrStopped) {
			t.Errorf("the run was ended by %v rather than by somebody stopping it", err)
		}
		if kind := fault.KindOf(err); kind != fault.Interrupted {
			t.Errorf("the run ended as %v rather than as an interruption, so the two "+
				"surfaces describe one event differently", kind)
		}
	}
}

// A read whose place of work is here happens where it was asked for, and what
// it found is acted on.
//
// This is the place of work for a caller with no window: the guard holding what
// every screen draws and the probe rendering them. Both need the screen
// finished by the time the call returns.
func TestAReadHereHappensWhereItWasAskedFor(t *testing.T) {
	read, acted := false, false
	ReadHere(func() func() {
		read = true
		return func() { acted = true }
	}, nil)
	if !read {
		t.Error("the read never happened")
	}
	if !acted {
		t.Error("the read happened and what it found was never acted on, so a caller with " +
			"no drawing thread would be left holding an unfinished screen")
	}
}

// A read with nothing to do about it is allowed.
//
// A screen drops the answer to a question nobody is asking any more, and it
// says so by handing back nothing. Calling that nothing would take the window
// down for a screen that decided, correctly, that its answer no longer matters.
func TestAReadWithNothingToDoAboutItIsAllowed(t *testing.T) {
	ReadHere(func() func() { return nil }, nil)
}

// # Why a panic here is a different event from a panic on a terminal
//
// A panic on a goroutine cannot be recovered by whoever started it, so before
// 2026-09-02 anything this window handed over could end the process. On the
// command line that costs a stack trace instead of four lines. In a window it
// costs the window, and with it whatever somebody had typed and not saved.
//
// The engine behind these ports is the same engine, and it had a live way to
// panic on the day this was written: a recipe declaring a range of children
// wider than the arithmetic behind it, typed into the editor, reached this
// through the reading port on the next keystroke.
func TestWorkThatComesApartIsAnsweredRatherThanEndingTheWindow(t *testing.T) {
	answered := make(chan error, 1)

	startAndAnswer(func() (int, error) { panic("the engine came apart") },
		func(_ int, err error) { answered <- err },
		func(act func()) { act() })

	err := <-answered
	if err == nil {
		t.Fatal("a panic in the work was answered as a success, so the screen would draw one")
	}
	if fault.KindOf(err) != fault.Runtime {
		t.Errorf("the failure carries kind %v rather than a defect of ours, and the kind is "+
			"what decides how it is reported", fault.KindOf(err))
	}
	if !strings.Contains(err.Error(), fault.Crashed) {
		t.Errorf("the failure says %q, which does not name itself as a defect of this build, "+
			"so a screen showing it would blame whoever was using it", err)
	}
	if !strings.Contains(err.Error(), "the engine came apart") {
		t.Errorf("the failure says %q and drops what actually happened, which is the one "+
			"thing a bug report needs", err)
	}
}

// A read that comes apart reaches the screen through the way out the screen
// chose, because a read had nowhere for a failure to go until this existed.
//
// 🔴 The reading port is the one that matters here rather than the one above.
// This window reads while somebody types, so the engine runs on the reading
// port far more often than on the running one, and that is the port that had
// no answer.
func TestAReadThatComesApartReachesTheWayOutTheScreenChose(t *testing.T) {
	drawn := make(chan error, 1)

	readInBackground(func() func() { panic("reading came apart") },
		func(err error) func() { return func() { drawn <- err } },
		func(act func()) { act() })

	err := <-drawn
	if fault.KindOf(err) != fault.Runtime {
		t.Errorf("the failure carries kind %v rather than a defect of ours", fault.KindOf(err))
	}
	if !strings.Contains(err.Error(), "reading came apart") {
		t.Errorf("the failure says %q and drops what actually happened", err)
	}
}

// A read that finishes is not sent down the way out for a crash, which is the
// half of the promise that is easy to lose while making the other half work.
func TestAReadThatFinishesDoesNotGoThroughTheWayOutForACrash(t *testing.T) {
	// Channels rather than flags, because the answer is made on one goroutine
	// and read on another. Written with flags this test had a data race in it,
	// and the race detector said so before anybody else did.
	acted, crashed := make(chan struct{}, 1), make(chan struct{}, 1)

	readInBackground(func() func() { return func() { acted <- struct{}{} } },
		func(error) func() { return func() { crashed <- struct{}{} } },
		func(act func()) { act() })

	select {
	case <-acted:
	case <-crashed:
		t.Fatal("a read that finished was reported as a crash, so a screen would draw a " +
			"defect instead of the answer it had")
	case <-time.After(2 * time.Second):
		t.Fatal("a read that finished was never acted on")
	}
}

// A panic in a ReadHere reaches the caller, which is the opposite promise from
// the one above and is deliberate.
//
// This is what the guard holding every screen and the probe drawing them use.
// Both own the thread they called on, so a panic arrives where it can be seen.
// Turning it into a drawn sentence would give them a screen that renders
// perfectly out of a defect, and a stored tree of that screen going green.
func TestAPanicInAReadHereReachesTheCallerThatOwnsTheThread(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Error("the panic was swallowed, so a probe would photograph a screen built " +
				"out of a defect and a stored tree of it would be green")
		}
	}()

	ReadHere(func() func() { panic("a probe wants to see this") }, nil)
}

// The tidy asks for its frame and then gives the memory back, both off the
// goroutine that started it, at the looks tidy.go says. When each comes is
// tidy_test.go's question. This one is whether the looks arriving on a clock
// reach the window at all.
func TestALeftAloneWindowIsAskedForAFrameAndThenGivesBack(t *testing.T) {
	looks := make(chan time.Time)
	framed, released := make(chan struct{}, 1), make(chan struct{}, 1)
	began := time.Date(2026, 11, 12, 18, 0, 0, 0, time.UTC)
	stop := tidyWhenLeftAlone(func() uint64 { return 0 }, began, looks, func() {},
		func() { framed <- struct{}{} }, func() { released <- struct{}{} })
	defer stop()

	looks <- began.Add(tidyQuiet)
	select {
	case <-framed:
	case <-released:
		t.Fatal("the memory went back before any frame was asked for, so the renderers the frame " +
			"lets go of would still be holding it")
	case <-time.After(2 * time.Second):
		t.Fatal("a window left alone for the whole quiet was never asked for a frame")
	}
	looks <- began.Add(tidyQuiet + tidySettle)
	select {
	case <-released:
	case <-time.After(2 * time.Second):
		t.Fatal("the frame came and the memory never went back")
	}
}

// sink keeps an allocation alive past the compiler, so the test below
// measures one that happened.
var sink []byte

// The counter the tidy reads exists on this system and counts. A runtime that
// renamed it would answer zero for ever, and the window would be tidied once
// after it opened and never again, with every test of the decision green.
func TestTheCounterTheTidyReadsCounts(t *testing.T) {
	before := allocatedSoFar()
	sink = make([]byte, 4<<20)
	after := allocatedSoFar()
	if after-before < 4<<20 {
		t.Fatalf("4 MB allocated moved the counter from %d to %d, so the tidy would never see the "+
			"window used", before, after)
	}
}

// Stopping the tidy twice is ordinary: the window closing stops it, and so
// does the end of main after it. Closing a closed channel is a panic, and it
// would come at the moment the program is leaving.
func TestStoppingTheTidyTwiceIsSafe(t *testing.T) {
	stoppedLooking := 0
	stop := tidyWhenLeftAlone(func() uint64 { return 0 }, time.Now(), make(chan time.Time),
		func() { stoppedLooking++ }, func() {}, func() {})
	stop()
	stop()
	if stoppedLooking != 1 {
		t.Errorf("the clock was stopped %d times, want once", stoppedLooking)
	}
}
