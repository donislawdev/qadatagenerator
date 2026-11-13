package background

import "time"

// When the window gives back what it no longer uses.
//
// # Why there is anything to give back
//
// Measured inside the real window on 2026-11-12 (tools/pamiec_w_oknie.py),
// two things hold memory nothing uses any more, and neither is ours to change:
//
//   - The toolkit keeps the renderer of an object that has left the window for
//     a minute, and destroys it only in the clean that follows a painted frame
//     (Fyne 2.8.2, internal/cache/base.go Clean, and drawSingleFrame in
//     internal/driver/glfw/loop.go). A window nobody touches paints nothing, so
//     after five quick screens of a thousand values each the window held 5502
//     renderers for as long as it stood alone, and 1042 to 1064 after one
//     frame.
//   - Go keeps the pages a collection has freed, up to about twice what is
//     alive, and returns them slowly. debug.FreeOSMemory gave 25 to 81 MB back.
//
// # How the window knows it is left alone
//
// By what it allocates. Measured on 2026-11-12, in two runs of two: left
// alone, this window allocated at most 0.3 MB in any seventy seconds, while
// opening a screen allocated about 10 MB and drawing a thousand values about
// 11. So
// no screen has to say that something happened, and a screen added in ten
// years is covered the day it exists. A run allocates for as long as it
// writes, so nothing is given back under a run without asking about runs.
//
// A field holding the keyboard counts as use. Its caret fades in and out, and
// the toolkit repaints the whole window on every frame of a fade
// (widget/entry_cursor_anim.go), which allocated 1.1 to 1.8 MB in twenty
// seconds on every screen with a field, measured on 2026-11-13. That is use
// until the window loses the keyboard of the desktop: going to another
// program takes it away, the caret stops (internal/driver/glfw/window.go,
// processFocused), and the quiet starts. Only the losing stops it, so in a
// window that never had the desktop's keyboard the caret goes on fading.
//
// # What happens, in order
//
// Once a whole tidyQuiet has gone by with under tidyBudget allocated, the
// window asks for one frame, by marking for redrawing an object that is not in
// it, so nothing in the window is touched. The way the neighbouring project asks, redrawing the root of the
// window, walks every object in it, hidden ones included, frees every texture
// and makes renderers for widgets that had none (internal/driver/common
// canvas.go, freeObject). Measured against each other, interleaved, two runs
// of each: the same renderers left (1042 and 1062 of 5502) and the same memory
// back (163.7 to 164.8 MB either way), so the one that does less is used.
// tidySettle later the memory goes back, off the thread that draws: in
// the neighbouring project one call of FreeOSMemory in ten took 2.5 seconds on
// that thread, for a reason nobody found, and it would land on somebody coming
// back to the window. Then the window waits to be used again, so one left open
// for a day is tidied once.
//
// # What it does not do
//
// A minimised window paints no frames either, so there the renderers stay and
// only what Go holds goes back. A window left with a field holding the
// keyboard, and nobody switching away from it, is never quiet: its own frames
// let the renderers go, and what Go holds stays. Memory the toolkit holds for
// good, the fonts parsed for every scope of a theme override among them
// (B-370), is not released by any of this.

const (
	// tidyLook is how often the allocations are read: a read of one counter of
	// the runtime, without stopping anything.
	tidyLook = 5 * time.Second
	// tidyQuiet is how long nothing has to happen. Past the minute the toolkit
	// keeps a renderer (cache.ValidDuration), so the renderers that left the
	// window with the last change have expired by the frame.
	tidyQuiet = 70 * time.Second
	// tidySettle is how long after the frame the memory goes back. The toolkit
	// cleans at most once in ten seconds, and a clean refused for coming too
	// soon is carried to the next one, at most ten seconds later.
	tidySettle = 12 * time.Second
	// tidyBudget is what the window may allocate across a whole quiet and
	// still be quiet. Left alone it allocated at most 0.3 MB in any seventy
	// seconds seen, the toolkit's own sparse jumps included (89 and 189 KB,
	// once in five and a half minutes), and a fading caret 3.8 MB or more on
	// the lightest screen, measured on 2026-11-13. A caret that ever came
	// under the budget would be tidied after like a window left alone, which
	// costs one frame and one release and nothing worse.
	//
	// Counted over the quiet rather than per look, and that is measured rather
	// than tidy. The first version asked each look for under 256 KB, and the
	// real window then tidied every ninety seconds for as long as it stood
	// alone: the release itself allocates 0.5 to 0.8 MB, which the next look
	// took for somebody using the window. Over the quiet, a trickle can never
	// add up to use however long the window stands: counted from the last
	// event, the slow one would reach any budget within the hour.
	tidyBudget = 2 << 20
)

// step is what one look asks of the window.
type step int

const (
	stepNothing step = iota
	// stepFrame asks the toolkit for one frame.
	stepFrame
	// stepRelease gives the memory back.
	stepRelease
)

// tidy is the one quiet being waited out, for the whole window. Every method
// is called from one goroutine, the one in TidyWhenLeftAlone, and nothing else
// reaches it, so it needs no lock and holds no channel.
type tidy struct {
	// quiet is the looks since the quiet began, oldest first, trimmed to the
	// last tidyQuiet and the one look before it, so that the first and the
	// last span the whole of it.
	quiet []reading
	// used says something has happened since the memory last went back.
	used bool
	// framed is when the frame was asked for, and zero while none is.
	framed time.Time
	// settling says the last look gave the memory back, so what the next one
	// sees allocated is the release's own.
	settling bool
}

// reading is one look: when, and the running total of allocations then.
type reading struct {
	at        time.Time
	allocated uint64
}

// newTidy begins with the window just built, which is the most it ever
// allocates at once, so it begins used.
func newTidy(allocated uint64, now time.Time) *tidy {
	return &tidy{quiet: []reading{{at: now, allocated: allocated}}, used: true}
}

// look reads the running total of allocations at now and answers with what
// the window should do.
func (t *tidy) look(allocated uint64, now time.Time) step {
	here := reading{at: now, allocated: allocated}
	if !t.framed.IsZero() {
		// What is allocated between the frame and the release is the frame
		// itself, which is not somebody using the window. Somebody coming back
		// in those seconds gets the memory back from under them, which costs
		// the pages faulted in again and nothing else.
		if now.Sub(t.framed) < tidySettle {
			return stepNothing
		}
		t.framed = time.Time{}
		t.used = false
		t.settling = true
		return stepRelease
	}
	if t.settling {
		// The release collects, and what it lets go of allocates on its way
		// out. The quiet starts after that, from here.
		t.settling = false
		t.quiet = append(t.quiet[:0], here)
		return stepNothing
	}
	t.quiet = append(t.quiet, here)
	for len(t.quiet) > 2 && now.Sub(t.quiet[1].at) >= tidyQuiet {
		t.quiet = t.quiet[1:]
	}
	began := t.quiet[0]
	if allocated-began.allocated >= tidyBudget {
		t.used = true
		t.quiet = append(t.quiet[:0], here)
		return stepNothing
	}
	if t.used && now.Sub(began.at) >= tidyQuiet {
		t.framed = now
		return stepFrame
	}
	return stepNothing
}
