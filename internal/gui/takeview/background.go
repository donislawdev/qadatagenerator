package takeview

import (
	"errors"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// This file is the only place in the window that starts concurrent work, and it
// is named in guard.ConcurrencyAllowed for exactly that reason. Everything
// concurrent about damaging a file from a screen lives here, so that the list
// stays one file long per surface and the screens themselves stay single
// threaded.
//
// # Why a run cannot simply happen where the button is pressed
//
// The toolkit draws on one thread and a button is pressed on it. A run that
// walked somebody else's file there would hold that thread for as long as the
// file takes, and the window would stop repainting, stop answering the keyboard
// and stop offering the way to stop the run. The three things this screen owes
// a person during a run are exactly the three that thread has to be free for.
//
// # Why the answer travels back through the toolkit rather than straight
//
// fyne.Do hands a function to the thread that draws, which is required of
// anything graphical touched from a goroutine we started. Calling the screen
// back directly would draw from the wrong thread, and the failure that produces
// is not a crash at the call. It is a window that repaints wrongly some of the
// time, which is the hardest kind of fault to find and the easiest to write.

// Background is where the work of a run happens and how its answer comes back.
//
// # Why this is handed in rather than written into the screen
//
// The same reason the reader of files is handed in. Every state this screen can
// be in then has a way in that needs neither a disk nor a goroutine, including
// the states worth looking at: a run in flight, a run that finished, a run
// somebody stopped and a run that failed. A screen that started its own
// goroutine could only be photographed in whichever state the scheduler happened
// to leave it in.
//
// It is deliberately not an interface with one method. What varies between the
// real caller and a test is where the work runs, which is a function, and a type
// with one method here would be a name for the same thing.
type Background func(work func() (take.Result, error), done func(take.Result, error))

// InBackground runs the work off the drawing thread and answers on it.
//
// This is the one the window uses. It is three lines because everything that
// makes concurrency dangerous is kept out of it: nothing here is shared, the
// work owns everything it touches, and the answer is handed over rather than
// written into anything the screen might be reading.
func InBackground(work func() (take.Result, error), done func(take.Result, error)) {
	go func() {
		result, err := work()
		fyne.Do(func() { done(result, err) })
	}()
}

// ErrStopped is what a run ends with when somebody stopped it from the window.
//
// It is a value rather than a message, because the screen has to tell a run
// somebody ended from one that failed, and it says each of them in words of its
// own. Telling them apart by the text of an error would be a screen reading a
// sentence written for a person, which this project has already paid for once in
// the register of bank routing numbers.
var ErrStopped = errors.New("stopped from the window")

// stopper is how a person ends a run that is already going.
//
// It answers a question rather than acting, which is the same arrangement the
// command line makes for a signal. The run itself decides where it may stop,
// so the files are removed on the goroutine that made them and nothing races
// with a write in progress.
type stopper struct {
	// stopping is closed once somebody has asked. Closing rather than sending
	// is what makes the answer repeatable: a send can be taken once, and the
	// run would then be told on its next record that it may carry on.
	stopping chan struct{}
}

func newStopper() *stopper { return &stopper{stopping: make(chan struct{})} }

// stop asks the run to end.
//
// Asking twice is ordinary rather than exotic, because the button that calls
// this is on screen until the run answers, so it takes the two presses a person
// makes when nothing seems to have happened. Closing a closed channel is a
// panic, so the state is read before it is changed. That is safe without a lock
// for one reason worth stating: this is only ever called from the thread that
// draws, and the goroutine doing the work only ever reads.
func (s *stopper) stop() {
	select {
	case <-s.stopping:
	default:
		close(s.stopping)
	}
}

// asked reports whether somebody has asked for the run to end.
func (s *stopper) asked() bool {
	select {
	case <-s.stopping:
		return true
	default:
		return false
	}
}

// Stopped returns the failure that ends the run, or nil to carry on.
//
// The kind is the one the command line uses when a person interrupts a run,
// because that is what this is. Nothing in the window reads the kind, and it is
// carried anyway so that the two surfaces describe one event the same way.
func (s *stopper) Stopped() error {
	if !s.asked() {
		return nil
	}
	return fault.Wrap(fault.Interrupted, ErrStopped, "the run ended early")
}
