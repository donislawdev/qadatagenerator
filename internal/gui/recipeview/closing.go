package recipeview

// The window and the system, while a run is going on.
//
// 🔴 The same arrangement the take and corrupt screen uses, and deliberately
// the same rather than a second answer to one question. Two screens can now
// spend minutes writing files, and both of them cost the same thing if the
// process ends where it stands: the temporary files of the run and the marker
// holding a name are left in somebody's output directory, where the next run
// finds the marker and refuses for a reason that stopped being true.
//
// The one thing that is NOT shared is the code, and that is worth naming rather
// than fixing in passing. Sharing it would mean a type holding a window, a
// place of work and a way to stop, which is most of a screen, and the two
// screens stop for different reasons and draw different things while they do.
// The day a third screen writes, this is the paragraph that says the sharing is
// due.

// interceptClose makes the closing of the window stop the run first.
//
// Closing a window ends the process, and a person pressing the close button is
// asking to leave rather than to kill anything. So for the length of a run that
// ask stops the run, and the window closes once it has stopped. Outside a run
// the window closes the way every other window does, which is why this is given
// back rather than left in place.
func (v *View) interceptClose() {
	if v.window == nil {
		return
	}
	v.window.SetCloseIntercept(func() {
		v.closing = true
		v.Stop()
	})
}

func (v *View) releaseClose() {
	if v.window != nil {
		v.window.SetCloseIntercept(nil)
	}
}

// watchForClosing makes the system asking this program to stop mean what the
// close button means.
//
// They are the same ask from different people, and they cost the same thing, so
// the second is routed into the answer the window already had rather than given
// one of its own that could drift from it. The watch lives exactly as long as
// the run, because outside a run nothing is half written and an instant end is
// what a desktop would have given anyway.
func (v *View) watchForClosing() {
	if v.work.Watch == nil {
		// A caller assembling this screen with nobody looking at it has no
		// process to be asked about. Said out loud rather than left to a nil
		// call, because every stored state of this screen is built that way.
		return
	}
	v.releaseWatch = v.work.Watch(v.stoppedFromOutside)
}

func (v *View) releaseWatchForClosing() {
	if v.releaseWatch == nil {
		return
	}
	v.releaseWatch()
	v.releaseWatch = nil
}

// stoppedFromOutside is what a signal means once it reaches the thread that
// draws.
//
// 🔴 The run may already have ended by the time this arrives, and that is
// ordinary rather than exotic: the signal is noticed on a goroutine and acted
// on here. Setting the flag anyway would arm the NEXT run to close the window
// behind whoever started it.
func (v *View) stoppedFromOutside() {
	if !v.running() {
		v.leave()
		return
	}
	v.closing = true
	v.Stop()
}

// leave gives up the window, which is the answer to being asked to stop once
// there is nothing left to clean up.
func (v *View) leave() {
	if v.window != nil {
		v.window.Close()
	}
}
