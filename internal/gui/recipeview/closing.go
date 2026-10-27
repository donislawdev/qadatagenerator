package recipeview

// The window and the system, while a run is going on.
//
// 🔴 The same arrangement the take and corrupt screen uses, and since
// 2026-10-27 the same code as well. Both screens can spend minutes writing
// files, and both cost the same thing if the process ends where it stands: the
// temporary files of the run and the marker holding a name are left in
// somebody's output directory, where the next run finds the marker and refuses
// for a reason that stopped being true.
//
// Until then each screen took the close button over for itself, on one window.
// A run here ending gave the button back while a run on the other screen was
// still writing, and this screen was never handed the watch for a signal,
// though it knew what to do with one. The question is about the window rather
// than about either screen, so it is answered by background.Closing, once.

// hold keeps the window from ending this run where it stands, or does nothing
// for a caller with no window behind the screen.
func (v *View) hold() func() {
	if v.work.Hold == nil {
		// A caller assembling this screen with nobody looking at it has no
		// window to hold. Said out loud rather than left to a nil call,
		// because every stored state of this screen is built that way.
		return nil
	}
	return v.work.Hold(v.Stop)
}

// release lets go of the window held for the run that has just ended.
func (v *View) release() {
	letGo := v.letGo
	v.letGo = nil
	if letGo != nil {
		letGo()
	}
}
