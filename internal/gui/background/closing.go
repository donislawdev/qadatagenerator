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

// Closing is the one answer this window gives to being asked to close while
// something is being written.
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
		c.window.SetCloseIntercept(c.leave)
		if c.watch != nil {
			c.release = c.watch(c.leave)
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
	c.window.SetCloseIntercept(nil)
	if c.release != nil {
		c.release()
		c.release = nil
	}
	if c.leaving {
		c.leaving = false
		c.window.Close()
	}
}

// leave is what the cross and a signal both mean: stop everything that is
// held, and give the window up once it has stopped.
//
// 🔴 A signal is noticed on a goroutine and acted on here, so it can arrive
// after the last run has already let go. That is ordinary rather than exotic,
// and the window is then given up at once: nothing is half written any more.
// Setting the flag anyway would leave it set, and the NEXT run somebody
// started would end by taking the window away from them.
func (c *Closing) leave() {
	if len(c.held) == 0 {
		c.window.Close()
		return
	}
	c.leaving = true
	// A copy, because a run asked to stop is allowed to answer at once and let
	// go of the list being walked.
	for _, one := range append([]holding(nil), c.held...) {
		one.stop()
	}
}
