package background

// Window is the part of a window that its closing needs.
//
// fyne.Window has both, and the coordinator is handed this rather than the
// whole of it so that a test can watch what is asked of a window without
// opening one. The toolkit's own test window carries out a close without
// asking whoever asked to be asked, so it could not show the one thing this
// file is about.
type Window interface {
	SetCloseIntercept(callback func())
	Close()
}

// Hold keeps the window from closing, and this program from ending on a signal,
// until the run that asked lets go.
//
// stop is how the run is asked to end when somebody asks for the window or the
// system asks this program to stop. letGo is called once the run has ended, and
// a second call does nothing. All three are called on the thread that draws.
//
// It is deliberately a function rather than the coordinator itself, for the
// reason the other ports are: a screen built with nobody looking at it is
// handed nothing here, and every stored state of every screen is built that
// way.
type Hold func(stop func()) (letGo func())

// Question is what the window asks a person before it goes, because going
// would lose something of theirs that is nowhere else, such as a recipe typed
// and never saved.
//
// It calls goOn once the person has answered in a way that lets the window go,
// and stay when they would rather keep it, exactly one of the two and as late as
// it needs: a question that saves first answers after the save. Called on the
// thread that draws.
type Question func(goOn, stay func())

// AskFirst is the port a screen is handed to put a question to whoever closes
// the window. Like Hold it is a function, and nothing for a screen built with
// nobody looking at it.
type AskFirst func(question Question)

// Closing is the one answer this window gives to being asked to close while
// something is being written, or while something would be lost.
//
// # Why one for the window rather than one per screen
//
// Until 2026-10-27 each screen that writes took the close button over at the
// start of its run and gave it back at the end, on the same window. A run on
// one screen ending therefore gave the button back while a run on the other was
// still writing, and the cross then ended the process in the middle of that
// write: its temporary files and the marker holding its name stayed in
// somebody's directory, where the next run finds the marker and refuses. The
// screen that writes a dataset was not watching for a signal at all, though it
// knew what to do with one, because each screen was handed a place of work of
// its own and that one was written without the watch. Both are one fault seen
// twice: a question about the whole window answered by each screen for itself.
//
// So the window is held for as long as anything holds it, and given up when the
// last thing lets go.
//
// # Why the watch lives exactly as long as the holding
//
// Outside a run this window has written nothing and holds no marker, so a
// person killing it is owed the instant end their desktop would have given
// them. That is the boundary the command line draws, and the reason
// WatchForClosing gives for not starting until there is a run.
//
// # Why a person is asked and a signal is not
//
// The cross, and Cmd+Q on a Mac, come from somebody sitting in front of the
// window, who can answer whether to keep what would be lost. Cmd+Q reaches the
// window as the same request as the cross: the Quit item of the menu GLFW 3.4
// builds calls terminate, its applicationShouldTerminate turns that into a
// close request of every window and cancels the ending (src/cocoa_init.m), and
// Fyne 2.8.2 with no menu of ours leaves that item where it is (resetDarwinMenu
// in menu_darwin.m). Read in both sources on 2026-11-25, not run on a Mac.
//
// A signal comes from a system that is stopping this
// program and will kill it a few seconds later whatever it is shown, so a
// question there would be a dialog nobody reads in front of a process about to
// end. A signal therefore ends the runs and the window as it always did, and
// only the cross asks. The questions come after the runs have ended rather than
// before, so that a save still going on when the cross is pressed has finished,
// and failed or not, by the time anybody is asked whether there is anything to
// keep.
//
// # Why it needs no lock
//
// Everything here is called on the thread that draws. A run starts and answers
// there, the cross is pressed there, and the watch hands a signal over to it
// before calling in. Nothing here is touched by a goroutine.
type Closing struct {
	window Window
	watch  Watch
	// held is the way to stop each run going on, in the order they began.
	held []holding
	// next numbers the holdings, so that letting go finds its own entry rather
	// than whichever entry happens to stand at the same place.
	next int
	// leaving says that somebody asked for the window while something was
	// held, so that it is given up once nothing is.
	leaving bool
	// personAsked says that the leaving was asked for by the cross rather than
	// by a signal, so the questions are asked before the window goes.
	personAsked bool
	// questions are asked one after another, in the order the screens put
	// them, and asking says that one of them is waiting for its answer, so
	// that a second press of the cross does not ask it twice.
	questions []Question
	asking    bool
	// gone says the window has been given up, so that an answer arriving
	// after a signal took the window away does not give it up a second time.
	gone bool
	// release stops the watch, and is nothing while nothing is held.
	release func()
}

type holding struct {
	number int
	stop   func()
}

// NewClosing makes the coordinator of one window.
//
// watch is how a signal is noticed, and may be nothing for a caller with no
// process to be asked about.
func NewClosing(window Window, watch Watch) *Closing {
	return &Closing{window: window, watch: watch}
}

// Hold is the port a screen is handed, as Hold describes it.
func (c *Closing) Hold(stop func()) (letGo func()) {
	c.next++
	number := c.next
	c.held = append(c.held, holding{number: number, stop: stop})
	if len(c.held) == 1 {
		c.cover()
		if c.watch != nil {
			c.release = c.watch(c.systemLeaves)
		}
	}
	if c.leaving {
		// 🔴 Somebody has already asked to leave and the window is waiting for
		// the runs going on to end. A run begun after that ask is stopped at
		// once rather than waited for. The ask may be a container being
		// stopped, which kills the process a few seconds later whatever it is
		// doing, and a run that outlives the ask is the run that gets killed
		// in the middle of a write.
		stop()
	}
	return func() { c.letGo(number) }
}

// letGo ends one holding, and gives the window up when it was the last one and
// somebody had asked for it.
func (c *Closing) letGo(number int) {
	at := -1
	for index, one := range c.held {
		if one.number == number {
			at = index
		}
	}
	if at < 0 {
		// Let go of already. A screen that lets go on the way out of a run
		// and again on the way out of the window is ordinary, and taking a
		// second entry off the list would end somebody else's holding.
		return
	}
	c.held = append(c.held[:at], c.held[at+1:]...)
	if len(c.held) > 0 {
		return
	}
	c.cover()
	if c.release != nil {
		c.release()
		c.release = nil
	}
	if c.leaving {
		c.leaving = false
		if c.personAsked {
			c.personAsked = false
			c.ask(0)
			return
		}
		c.goNow()
	}
}

// AskFirst is the port a screen is handed, as AskFirst describes it.
func (c *Closing) AskFirst(question Question) {
	c.questions = append(c.questions, question)
	c.cover()
}

// cover takes the cross over while anything is held or anything is to be
// asked, and gives it back to the toolkit otherwise, so a window with nothing
// at stake closes the instant a person asks, as their desktop would close it.
func (c *Closing) cover() {
	if len(c.held) > 0 || len(c.questions) > 0 {
		c.window.SetCloseIntercept(c.personLeaves)
		return
	}
	c.window.SetCloseIntercept(nil)
}

// personLeaves is the cross: stop everything that is held, then ask the
// questions, then give the window up.
func (c *Closing) personLeaves() {
	if c.asking || c.gone {
		// A question is open already. Asking it again over itself would
		// leave two dialogs for one press, and the first answer would
		// close a window the second was still asking about.
		return
	}
	if len(c.held) == 0 {
		c.ask(0)
		return
	}
	c.leaving, c.personAsked = true, true
	c.stopAll()
}

// systemLeaves is a signal: stop everything that is held, and give the window
// up once it has stopped, asking nothing.
//
// 🔴 A signal is noticed on a goroutine and acted on here, so it can arrive
// after the last run has already let go. That is ordinary rather than exotic,
// and the window is then given up at once: nothing is half written any more.
// Setting the flag anyway would leave it set, and the NEXT run somebody
// started would end by taking the window away from them.
func (c *Closing) systemLeaves() {
	if c.gone {
		return
	}
	if len(c.held) == 0 {
		c.goNow()
		return
	}
	// A person may have asked first. The system asking after them wins,
	// for the reason the type gives: it is not going to wait for an answer.
	c.leaving, c.personAsked = true, false
	c.stopAll()
}

// stopAll asks every run going on to stop.
func (c *Closing) stopAll() {
	// A copy, because a run asked to stop is allowed to answer at once and let
	// go of the list being walked.
	for _, one := range append([]holding(nil), c.held...) {
		one.stop()
	}
}

// ask puts the question at this place in the list, and the next one once it
// lets the window go, and gives the window up after the last.
//
// An answer is taken once. A question that calls both, or one of them twice,
// would otherwise walk the list twice or close a window it had just kept.
func (c *Closing) ask(at int) {
	if c.gone {
		return
	}
	if at == len(c.questions) {
		c.asking = false
		if len(c.held) > 0 {
			// Something a question started, a save, is still going on. The
			// window goes once it ends, and there is nothing left to ask.
			c.leaving = true
			c.stopAll()
			return
		}
		c.goNow()
		return
	}
	c.asking = true
	answered := false
	c.questions[at](func() {
		if !answered {
			answered = true
			c.ask(at + 1)
		}
	}, func() {
		if !answered {
			answered = true
			c.asking = false
		}
	})
}

// goNow gives the window up, once.
func (c *Closing) goNow() {
	c.gone = true
	c.window.Close()
}
