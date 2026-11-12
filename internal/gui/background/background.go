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
// Measured on 2026-10-27: three million rows of four columns, 109 MB, take
// about 0.6 seconds to read, and this mode exists for files larger than that.
// A window doing that where the button was pressed is a window that freezes
// for as long as the file is large. Until that day this paragraph cited 11.7
// seconds for a hundred thousand rows, which was a measurement of GENERATING
// rows with word lists read again for every cell, carried over from
// internal/guard/concurrency.go as though it were the reading.
//
// # Why a panic stops here rather than anywhere
//
// A panic on a goroutine cannot be recovered by whoever started it, so every
// launcher below catches one and hands it back as a failure. Until 2026-09-02
// none of them did, and any defect in the engine ended the window without a
// word, taking along whatever was typed into it and not saved. The command
// line had answered the same event since the day it could run anything.
//
// This is not about tidying up after a crash, and the difference was measured
// rather than assumed: a deferred call on the goroutine that panics runs even
// with nothing recovering it, so a run still removes the files and the marker
// it was holding. What was missing was the account and the window to show it
// in.
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
	"runtime/debug"
	"runtime/metrics"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/produce"
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
	// Produce is where a run that makes a dataset from a recipe happens.
	//
	// A second port of the same shape rather than one made general, for the
	// reason the shape exists at all: the answer is a value the port carries,
	// so a stand in can hand a screen an outcome without writing any files, and
	// the outcome of the two modes is a different type. One port over both
	// would carry an empty interface, and every screen would open it again.
	Produce Produce
	// Read is where somebody else's file is read.
	Read Read
	// Hold keeps the window, and this program, from ending where a run stands,
	// or is nothing for a caller with no window behind the screen. It is the
	// port of the one Closing of the window, and the reasons stand there.
	Hold Hold
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

// Produce is where a dataset is made, and how the answer comes back.
//
// The same shape as Run above and for the same reason, which is written out
// there: the four outcomes of a run have to be reachable without a disk.
type Produce func(work func() (produce.Written, error), done func(produce.Written, error))

// Read is where a file is read, and how what was read reaches the screen.
//
// The function handed in does the reading and returns what to do about it. That
// continuation is built on the goroutine that read, out of values it made
// itself, and it is run on the thread that draws. So nothing is shared between
// the two threads and nothing is written into anything a screen might be
// reading, which is the same promise Run keeps by handing its answer over.
//
// # Why there is a second function, and why it is not optional
//
// It says what the screen does when the reading itself comes apart, and it is
// a parameter rather than a courtesy because the two ports beside this one
// already have somewhere for a failure to go and this one had nowhere. A panic
// during a read was therefore the end of the window, silently, and the reading
// this window does most is judging a recipe while somebody types it.
//
// Written as something a caller may add, it would be the guard somebody does
// not add. As a parameter the compiler asks every screen the question, and a
// screen that has nothing to say can still only answer deliberately.
type Read func(read func() func(), crashed func(error) func())

// Watch starts noticing that the system has asked this program to stop, and
// answers with the way to stop noticing.
//
// asked is called on the thread that draws.
type Watch func(asked func()) (release func())

// caught runs the work and turns a panic into a failure the screen can show.
//
// # Why a panic has to become a value here of all places
//
// A panic on a goroutine cannot be recovered by whoever started it. It ends
// the process, so without this the window goes out mid sentence and takes
// along whatever somebody had typed and not saved. The command line has
// answered this since the day it could run anything, in internal/cli, and the
// window had no answer at all until 2026-09-02.
//
// It is not about tidying up after the crash, and that distinction is measured
// rather than assumed. A deferred call on the goroutine that panics runs even
// when nothing recovers it, so a run removes its own temporary files and its
// own marker while the stack unwinds. What was being lost is the account of
// what happened and the window it would have been shown in.
//
// # Why it is generic when this package has one other generic function
//
// The three ports below carry three different answers, and the panic has to
// arrive as the same failure in all three. Written out per port it would be
// the same six lines three times, which is how two of them end up drifting the
// day somebody improves one.
func caught[T any](work func() (T, error)) (answer T, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fault.FromPanic(recovered)
		}
	}()
	return work()
}

// InBackground runs the work off the drawing thread and answers on it.
//
// This is the one the window uses. It is short because everything that makes
// concurrency dangerous is kept out of it: nothing here is shared, the work
// owns everything it touches, and the answer is handed over rather than
// written into anything the screen might be reading.
func InBackground(work func() (take.Result, error), done func(take.Result, error)) {
	startAndAnswer(work, done, fyne.Do)
}

// ProduceInBackground makes a dataset off the drawing thread and answers on it.
func ProduceInBackground(work func() (produce.Written, error),
	done func(produce.Written, error)) {
	startAndAnswer(work, done, fyne.Do)
}

// startAndAnswer is both of the ports above, with the crossing back to the
// drawing thread handed in.
//
// Split for the same one reason watchForClosing below is split, and it is the
// reason worth splitting for: fyne.Do needs a toolkit that is running, so
// without this seam the only way to ask whether a panic still ends the window
// would be to open one. What the two ports have to promise is that the work is
// started elsewhere and that a panic in it comes back as an answer, and both
// of those are visible from here.
func startAndAnswer[T any](work func() (T, error), done func(T, error),
	onTheDrawingThread func(func())) {
	go func() {
		answer, err := caught(work)
		onTheDrawingThread(func() { done(answer, err) })
	}()
}

// ReadInBackground reads off the drawing thread and acts on it there.
func ReadInBackground(read func() func(), crashed func(error) func()) {
	readInBackground(read, crashed, fyne.Do)
}

// readInBackground is the port above with the crossing handed in, for the
// reason startAndAnswer gives.
func readInBackground(read func() func(), crashed func(error) func(),
	onTheDrawingThread func(func())) {
	go func() {
		act, err := caught(func() (func(), error) { return read(), nil })
		if err != nil {
			act = crashed(err)
		}
		if act == nil {
			// A read with nothing to do about it is allowed rather than a
			// fault. Calling it anyway would take the window down for a
			// screen that decided, correctly, that its answer no longer
			// matters.
			return
		}
		onTheDrawingThread(act)
	}()
}

// ReadHere reads on the thread that asked, which is what a caller with no
// window of its own wants.
//
// It exists for the two callers that assemble screens without anybody looking:
// the guard holding what every screen draws, and the probe rendering them to
// pictures. Both want the screen finished by the time the call returns, and
// neither has a drawing thread to hand anything back to.
//
// # Why this one does not catch a panic while the one above does
//
// The one above catches because nobody else can: its goroutine is the end of
// the line and a panic there is the end of the window. Here the caller owns
// the thread, so a panic arrives where it was asked for, which is a test or a
// probe. Turning it into a drawn sentence would mean a screen that renders
// perfectly out of a defect, and a stored tree of that screen going green.
func ReadHere(read func() func(), _ func(error) func()) {
	if act := read(); act != nil {
		act()
	}
}

// TidyWhenLeftAlone gives back what the window no longer uses once nobody has
// used it for a while, and answers with the way to stop. When, and why there
// is anything to give back, is in tidy.go.
//
// # Why it lives here
//
// The quiet has to be waited out somewhere, and the thread that draws does
// nothing at all while nobody uses the window, which is the whole point. The
// memory then goes back off that thread, for the reason tidy.go gives. The
// frame is the one thing done on it, handed over through fyne.Do like every
// answer in this file.
//
// # Why a panic here stops the tidying and nothing else
//
// Every other goroutine in this file turns a panic into a failure a screen
// shows. This one has nobody to show it to, and the window ending over memory
// it could not give back would be the worse trade. The one panic it can meet
// is the toolkit's own, at the very end: fyne.Do sends on a queue the toolkit
// closes as it shuts down (Fyne 2.8.2, internal/driver/glfw/loop.go), checking
// a flag first, and a send between the flag and the close would land on a
// closed channel. Stopping the tidying when the window closes makes that a
// window of microseconds, and this makes it harmless.
func TidyWhenLeftAlone(window fyne.Canvas) (stop func()) {
	// Not in the window and never put there. Redrawing it is a frame that
	// touches nothing, which is what tidy.go asks for.
	nothing := canvas.NewRectangle(nil)
	ticker := time.NewTicker(tidyLook)
	return tidyWhenLeftAlone(allocatedSoFar, time.Now(), ticker.C, ticker.Stop,
		func() { fyne.Do(func() { window.Refresh(nothing) }) }, debug.FreeOSMemory)
}

// tidyWhenLeftAlone is the whole of the tidying, with the clock, the counter
// and what a frame and a release are handed in, for the reason
// watchForClosing gives: fyne.Do needs a running toolkit, and the clock would
// make a test wait out a real quiet.
func tidyWhenLeftAlone(allocated func() uint64, began time.Time, looks <-chan time.Time,
	stopLooking func(), frame, release func()) (stop func()) {
	t := newTidy(allocated(), began)
	done := make(chan struct{})
	go func() {
		defer func() { _ = recover() }()
		for {
			select {
			case <-done:
				return
			case now := <-looks:
				switch t.look(allocated(), now) {
				case stepFrame:
					frame()
				case stepRelease:
					release()
				}
			}
		}
	}()
	// Read and written on the thread that stops, and only there: the window
	// closing and the end of main are both on the thread that draws.
	stopped := false
	return func() {
		if stopped {
			return
		}
		stopped = true
		stopLooking()
		close(done)
	}
}

// allocatedSoFar is every byte the heap has handed out since the process
// began, read without stopping anything, unlike runtime.ReadMemStats. Zero if
// a runtime ever stops counting it, and the tidy then sees a window that is
// never used again after it opened: tidied once, then never, which is the
// cheap way to be wrong.
func allocatedSoFar() uint64 {
	sample := []metrics.Sample{{Name: "/gc/heap/allocs:bytes"}}
	metrics.Read(sample)
	if sample[0].Value.Kind() != metrics.KindUint64 {
		return 0
	}
	return sample[0].Value.Uint64()
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
//
// Latest uses the same thing for a reading that a newer question has made
// stale, because it is the same event seen from the other end: somebody on
// the screen no longer wants what is being worked out.
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
// It asks the Closing of the window to stop whatever is held, and each run
// decides what that means. It never removes a file itself, for the reason the
// stopper above states: the run owns what it is writing and ends it on the
// goroutine that made it.
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
