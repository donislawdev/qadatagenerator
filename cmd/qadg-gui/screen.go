package main

import (
	"fyne.io/fyne/v2"
	"github.com/go-gl/glfw/v3.4/glfw"

	"github.com/donislawdev/qadatagenerator/internal/gui/kept"
)

// fitToScreen makes a window larger than the room on its screen small enough
// to fit, and puts every window in the middle of the monitor (B-421, the room
// and the measurements are in kept.Fit). Where the system says nothing usable
// about the screen, the window stays as it opened and where it opened.
//
// # Why every window goes to the middle
//
// The toolkit says neither where its window is nor how to move it, except to
// the middle. Where the system puts a new window, a large one hangs off the
// bottom of the screen, so the middle is the one place this program can put a
// window and know all of it is on the screen.
//
// # Why after the window is shown, and why through GLFW
//
// The toolkit offers no way to ask how big the screen is. GLFW, which the
// toolkit opens its windows with and which every build of this window already
// links, does: the primary monitor, where it stands, and its work area, in
// screen coordinates. Turning those into the window's units takes the scale
// the window is drawn at, and that is known for certain only once the window
// exists, from its canvas. Working the scale out beforehand would mean copying
// how the toolkit computes it on each system, which is the toolkit's business.
// By then GLFW has started, and this runs on the main thread, where main runs
// and where GLFW has to be asked.
//
// # Why two resizes
//
// The system makes a window too large for the screen smaller before the
// toolkit listens for its size, so the toolkit still believes the size it
// asked for. It learns the real size only when the system reports a change.
// Going to half the size first and then to the size that fits are two changes
// whatever the system did with the first size, so the toolkit hears the size
// the window really has even when the system has to make it smaller again.
//
// # What it does not cover
//
// The primary monitor is asked rather than the one the window is on, because
// the toolkit does not say where its window is, so a window that opens on
// another monitor is fitted to the primary one. A screen with a taskbar that
// hides itself has
// a work area as large as the monitor, so a window cut down to it still has
// its title bar partly above the top once centred.
func fitToScreen(window fyne.Window, size fyne.Size) {
	screen, ok := primaryScreen(window.Canvas().Scale())
	if !ok {
		return
	}
	if fitted, refit := kept.Fit(size, screen); refit {
		window.Resize(fyne.NewSize(fitted.Width/2, fitted.Height/2))
		window.Resize(fitted)
	}
	window.CenterOnScreen()
}

// primaryScreen is the primary monitor and its work area in the window's
// units, or false when the system says nothing usable.
//
// go-gl/glfw panics on any error GLFW has reported since it was last asked,
// not only on one from this call (error.go in v0.1.0-pre.2, the same reason
// failure.go catches a panic out of Show). A screen that cannot be read leaves
// the window as it opened, which is how every window opened before this
// existed, so a panic here is answered by not fitting rather than by stopping
// the program.
func primaryScreen(scale float32) (screen kept.Display, ok bool) {
	defer func() {
		if recover() != nil {
			screen, ok = kept.Display{}, false
		}
	}()
	if scale <= 0 {
		return kept.Display{}, false
	}
	monitor := glfw.GetPrimaryMonitor()
	if monitor == nil {
		return kept.Display{}, false
	}
	mode := monitor.GetVideoMode()
	if mode == nil {
		return kept.Display{}, false
	}
	units := func(coordinate int) float32 { return float32(coordinate) / scale }
	monitorX, monitorY := monitor.GetPos()
	workX, workY, workWidth, workHeight := monitor.GetWorkarea()
	return kept.Display{
		Monitor:   fyne.NewSize(units(mode.Width), units(mode.Height)),
		MonitorAt: fyne.NewPos(units(monitorX), units(monitorY)),
		Work:      fyne.NewSize(units(workWidth), units(workHeight)),
		WorkAt:    fyne.NewPos(units(workX), units(workY)),
	}, true
}
