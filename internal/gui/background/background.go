// Package background is the one place in the window that starts concurrent
// work, and it is named in guard.ConcurrencyAllowed for exactly that reason.
//
// # Why one file for the whole window rather than one per screen
//
// The register of files allowed to start concurrent work asks for a paragraph
// per entry, and says that a third entry of the same shape deserves the same
// paragraph. This file is that paragraph written once for a surface rather than
// once for each screen of it. Everything the window must not do on the thread
// that draws lives here, and every screen stays single threaded and knows only
// that it handed something over.
//
// It began as internal/gui/takeview/background.go, which held the run alone.
// Two more things then turned out to need the same treatment, and both are on
// other screens: reading somebody's file, and noticing that the system has
// asked this program to stop. Filing each of them beside the screen that wants
// it would have grown the register by one entry per screen, and the register is
// meant to be read rather than counted.
//
// # Why none of this may happen where it is asked for
//
// The toolkit draws on one thread and a control is used on it. Anything held
// there for as long as a file takes stops the window repainting, stops it
// answering the keyboard, and stops it offering the way out of whatever is
// going on. Those three are exactly what a person is owed while they wait, so
// this is not a choice about speed.
//
// Measured on 2026-08-26 in the neighbouring package: a hundred thousand rows
// of four columns took 11.7 seconds to walk. A window doing that where the
// button was pressed is a window the desktop offers to kill.
//
// # Why every answer travels back through the toolkit
//
// fyne.Do hands a function to the thread that draws, which is required of
// anything graphical touched from a goroutine we started. Calling a screen back
// directly would draw from the wrong thread, and the failure that produces is
// not a crash at the call. It is a window that repaints wrongly some of the
// time, which is the hardest kind of fault to find and the easiest to write.
//
// # Why the exception for syscall lives here as well
//
// A termination sent by a system has no name outside package syscall, and that
// package is refused everywhere else because it can do whatever the operating
// system can. guard.SyscallAllowed therefore names this file and the one
// identifier it may take, the same bargain internal/cli/signals.go already
// makes, and for the same runs: a container stopped with docker stop and a
// desktop session logging out both send that signal, and both are runs nobody
// is watching.
package background

import (
	"errors"
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// Work is everywhere a screen sends what must not happen on the thread that
// draws.
//
// # Why it is handed to a screen rather than reached for inside one
//
// Every state a screen can be in then has a way in that needs no goroutine,
// including the states worth looking at: a run in flight, a file still being
// read, a run somebody stopped. A screen that started its own goroutine could
// only be photographed in whichever state the scheduler happened to leave it
// in, and the stored trees of this window would go red at random.
type Work struct {
	// Run is where a run of the take and corrupt mode happens.
	Run Run
	// Read is where somebody else's file is read.
	Read Read
	// Watch is how the system asking this program to stop reaches a screen, or
	// nothing for a caller that has no process to be asked about.
	Watch Watch
}

// Run is where the work of a run happens and how its answer comes back.
//
// # Why this carries the answer as a value while Read does not
//
// The two ports are deliberately different shapes, and the difference is what
// each one has to let a stand-in do. The states of this window include four
// outcomes of a run, and each of them has to be reachable without writing
// files, so a stand-in has to be able to hand a screen an outcome WITHOUT
// carrying the work out. That means the outcome is a value the port carries.
//
// A read has no such need, because what is read is already handed to the screen
// as a function of its own. A stand-in there decides where and when the read
// happens and never what it answers.
//
// It is deliberately not an interface with one method. What varies between the
// real caller and a test is where the work runs, which is a function, and a
// type with one method would be a name for the same thing.
type Run func(work func() (take.Result, error), done func(take.Result, error))

// Read is where a file is read, and how what was read reaches the screen.
//
// The function handed in does the reading and returns what to do about it. That
// continuation is built on the goroutine that read, out of values it made
// itself, and it is run on the thread that draws. So nothing is shared between
// the two threads and nothing is written into anything a screen might be
// reading, which is the same promise Run keeps by handing its answer over.
type Read func(read func() func())

// Watch starts noticing that the system has asked this program to stop, and
// answers with the way to stop noticing.
//
// asked is called on the thread that draws.
type Watch func(asked func()) (release func())

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

// ReadInBackground reads off the drawing thread and acts on it there.
func ReadInBackground(read func() func()) {
	go func() {
		act := read()
		if act == nil {
			// A read with nothing to do about it is allowed rather than a
			// fault. Calling it anyway would take the window down for a
			// screen that decided, correctly, that its answer no longer
			// matters.
			return
		}
		fyne.Do(act)
	}()
}

// ReadHere reads on the thread that asked, which is what a caller with no
// window of its own wants.
//
// It exists for the two callers that assemble screens without anybody looking:
// the guard holding what every screen draws, and the probe rendering them to
// pictures. Both want the screen finished by the time the call returns, and
// neither has a drawing thread to hand anything back to.
func ReadHere(read func() func()) {
	if act := read(); act != nil {
		act()
	}
}

// ErrStopped is what a run ends with when somebody stopped it from the window.
//
// It is a value rather than a message, because the screen has to tell a run
// somebody ended from one that failed, and it says each of them in words of its
// own. Telling them apart by the text of an error would be a screen reading a
// sentence written for a person, which this project has already paid for once in
// the register of bank routing numbers.
var ErrStopped = errors.New("stopped from the window")

// Stopper is how a person ends a run that is already going.
//
// It answers a question rather than acting, which is the same arrangement the
// command line makes for a signal. The run itself decides where it may stop,
// so the files are removed on the goroutine that made them and nothing races
// with a write in progress.
type Stopper struct {
	// stopping is closed once somebody has asked. Closing rather than sending
	// is what makes the answer repeatable: a send can be taken once, and the
	// run would then be told on its next record that it may carry on.
	stopping chan struct{}
}

// NewStopper makes the way to end one run.
func NewStopper() *Stopper { return &Stopper{stopping: make(chan struct{})} }

// Stop asks the run to end.
//
// Asking twice is ordinary rather than exotic, because the button that calls
// this is on screen until the run answers, so it takes the two presses a person
// makes when nothing seems to have happened. Closing a closed channel is a
// panic, so the state is read before it is changed. That is safe without a lock
// for one reason worth stating: this is only ever called from the thread that
// draws, and the goroutine doing the work only ever reads.
func (s *Stopper) Stop() {
	select {
	case <-s.stopping:
	default:
		close(s.stopping)
	}
}

// Asked reports whether somebody has asked for the run to end.
func (s *Stopper) Asked() bool {
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
func (s *Stopper) Stopped() error {
	if !s.Asked() {
		return nil
	}
	return fault.Wrap(fault.Interrupted, ErrStopped, "the run ended early")
}

// WatchForClosing notices the system asking this program to stop, and turns
// that ask into the one the close button already makes.
//
// # Why the window answers a signal at all
//
// A run writes every file under a temporary name and holds a marker naming what
// it is about to produce, so that a second run started beside it refuses rather
// than writing over the first. A process killed in the middle of a run leaves
// both behind, and the next run finds the marker and refuses for a reason that
// is no longer true. The command line has removed them since the day it could
// write anything, by catching the two signals and letting the run end where it
// chooses to. Until this existed the window did not, so the same interruption
// cost a person a directory they then had to clean by hand. That was the sixth
// unbreakable rule broken in one direction: one function, two behaviours,
// depending on which surface asked for it.
//
// # Why the watch does not start until there is a run
//
// Outside a run this window has written nothing and holds no marker, so there
// is nothing to take back and a person who kills it gets the instant end their
// desktop would have given them anyway. The watch covers the part with
// something to lose, which is the same boundary the command line draws.
//
// # What this asks for and what it does not
//
// It asks a screen to stop, and the screen decides what that means. It never
// removes a file itself, for the reason the stopper above states: the run owns
// what it is writing and ends it on the goroutine that made it.
func WatchForClosing(asked func()) (release func()) {
	return watchForClosing(asked, fyne.Do)
}

// watchForClosing is the whole of the watch, with the way an answer reaches the
// thread that draws handed in.
//
// Split for one test and it is the test worth splitting for: the only thing
// nothing else here can show is whether the runtime hands this program the
// signals it asked for. Sending a real one needs no window, and fyne.Do needs a
// running toolkit, so the two are separated rather than one of them faked.
func watchForClosing(asked func(), onTheDrawingThread func(func())) (release func()) {
	// Buffered, because a signal sent while nobody is receiving is dropped and
	// the first one is the one that matters.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		if _, arrived := <-signals; !arrived {
			// The run ended on its own and released the watch.
			return
		}
		// Hand the signal back to the system before anything else. From here on
		// a second one ends the process the way it would have without this
		// file, which matters because the run only stops where it checks:
		// somebody whose run is stuck in a write on a full disk has to be able
		// to get out.
		signal.Stop(signals)
		onTheDrawingThread(asked)
	}()
	// Released is read and written on the thread that draws and nowhere else,
	// which is what lets it be a plain flag. Closing a closed channel is a
	// panic, and a caller that releases a watch on the way out of a run and
	// again on the way out of a window would take the program down at exactly
	// the moment it was trying to leave cleanly.
	released := false
	return func() {
		if released {
			return
		}
		released = true
		// Safe in this order and in no other: signal.Stop waits for a delivery
		// already on its way, so after it returns nothing can send on a channel
		// about to be closed. Calling it twice is allowed, which is what makes
		// this correct when the goroutine above has already stopped the watch.
		signal.Stop(signals)
		close(signals)
	}
}
