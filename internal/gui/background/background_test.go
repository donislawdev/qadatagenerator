package background

import (
	"errors"
	"testing"

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
	})
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
	ReadHere(func() func() { return nil })
}
