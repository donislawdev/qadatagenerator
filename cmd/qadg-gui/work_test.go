package main

import "testing"

// heldWindow is a window that keeps what was asked of it, so that the test can
// see the cross taken over without opening anything.
type heldWindow struct {
	intercept func()
	closed    int
}

func (w *heldWindow) SetCloseIntercept(intercept func()) { w.intercept = intercept }
func (w *heldWindow) Close()                             { w.closed++ }

// 🔴 Every screen is handed every place of work, the window's closing among
// them.
//
// Until 2026-10-27 each screen was given a place of work written out for it,
// and the one for the recipe screen had no watch for a signal, though the
// screen knew what to do with one. A container stopped while that screen wrote
// a dataset therefore ended the process where it stood and left the temporary
// files and the marker behind. Nothing failed, because a port that is absent
// is a port nobody calls.
func TestEveryScreenIsHandedEveryPlaceOfWork(t *testing.T) {
	window := &heldWindow{}
	work := placesOfWork(window)
	if work.Run == nil || work.Produce == nil || work.Read == nil || work.Hold == nil {
		t.Fatalf("a place of work is missing, so the screen handed it does that part of its "+
			"work wherever it can: run %t, produce %t, read %t, hold %t", work.Run != nil,
			work.Produce != nil, work.Read != nil, work.Hold != nil)
	}
	letGo := work.Hold(func() {})
	if window.intercept == nil {
		t.Error("holding the window for a run did not take the cross over, so closing it " +
			"during the run ends the process in the middle of a write")
	}
	letGo()
	if window.intercept != nil {
		t.Error("the run let go and the cross is still taken over")
	}
}
