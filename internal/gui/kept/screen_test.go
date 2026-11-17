package kept

import (
	"testing"

	"fyne.io/fyne/v2"
)

// owners is the screen the faults of B-421 were measured on: 3840 by 2160
// pixels at a scale of 1.5, with the taskbar along the bottom taking 72 of
// them.
var owners = Display{
	Monitor: fyne.NewSize(2560, 1440),
	Work:    fyne.NewSize(2560, 1392),
}

func TestASizeThatFitsTheScreenIsLeftAsItIs(t *testing.T) {
	for _, size := range []fyne.Size{
		fyne.NewSize(1120, 780),
		// Exactly the room a centred window has on that screen.
		fyne.NewSize(2560, 1344),
	} {
		if got, refit := Fit(size, owners); refit || got != size {
			t.Errorf("%v fits the work area and came back as %v (refit %v)", size, got, refit)
		}
	}
}

// The two sizes measured as cut off on 2026-11-17, the window closed maximised
// that opened partly below the screen, and two larger on one side only. Each
// comes back no larger than a window that, centred on the monitor, stays clear
// of the taskbar: 1440 less twice the 48 the taskbar takes. The maximised one
// is the canary of the table above: a rule that let everything up to the work
// area through would pass that table and leave this window hanging off the
// bottom.
func TestASizeLargerThanTheScreenIsCutToWhatStaysClearOfTheTaskbar(t *testing.T) {
	for _, c := range []struct{ size, want fyne.Size }{
		{fyne.NewSize(2700, 1500), fyne.NewSize(2560, 1344)},
		{fyne.NewSize(2560, 1369.33), fyne.NewSize(2560, 1344)},
		{fyne.NewSize(100000, 100000), fyne.NewSize(2560, 1344)},
		{fyne.NewSize(1200, 1500), fyne.NewSize(1200, 1344)},
		{fyne.NewSize(3000, 900), fyne.NewSize(2560, 900)},
	} {
		got, refit := Fit(c.size, owners)
		if !refit || got != c.want {
			t.Errorf("%v on that screen came back as %v (refit %v), want %v", c.size, got, refit, c.want)
		}
	}
}

// The middle of the monitor is not the middle of the work area when the
// taskbar is on one side, and a monitor to the right of the first one has its
// own origin. Both are read from where the system puts them.
func TestTheRoomFollowsWhereTheTaskbarAndTheMonitorAre(t *testing.T) {
	left := Display{
		Monitor: fyne.NewSize(1920, 1080),
		Work:    fyne.NewSize(1860, 1080), WorkAt: fyne.NewPos(60, 0),
	}
	if got, _ := Fit(fyne.NewSize(5000, 5000), left); got != fyne.NewSize(1800, 1080) {
		t.Errorf("a taskbar on the left gives %v, want 1800 by 1080", got)
	}
	second := Display{
		Monitor: fyne.NewSize(1280, 720), MonitorAt: fyne.NewPos(2560, 0),
		Work: fyne.NewSize(1280, 690), WorkAt: fyne.NewPos(2560, 0),
	}
	if got, _ := Fit(fyne.NewSize(5000, 5000), second); got != fyne.NewSize(1280, 660) {
		t.Errorf("a monitor to the right gives %v, want 1280 by 660", got)
	}
}

// A screen the system described with a nought in it, or a work area that
// leaves no room in the middle, leaves the window as it was. Shrinking it to
// nothing would be worse than the fault this exists for.
func TestAScreenWithNothingUsableChangesNothing(t *testing.T) {
	size := fyne.NewSize(2700, 1500)
	for _, screen := range []Display{
		{},
		{Monitor: fyne.NewSize(2560, 1440)},
		{Monitor: fyne.NewSize(2560, 1440), Work: fyne.NewSize(2560, 600), WorkAt: fyne.NewPos(0, 840)},
	} {
		if got, refit := Fit(size, screen); refit || got != size {
			t.Errorf("on %+v the size became %v (refit %v)", screen, got, refit)
		}
	}
}
