//go:build darwin

package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
)

// On macOS a notch moves an area by what the toolkit hands over, because the
// system's distances arrive already scaled and accelerated there
// (wheel_darwin.go). The setting of lines must change nothing, or the window
// would scroll a Mac differently from every other program on it.
func TestOnMacOSTheWheelIsTheSystems(t *testing.T) {
	was := WheelLines
	t.Cleanup(func() { WheelLines = was })
	WheelLines = func() int { return 7 }

	area := NewScroll(container.NewGridWrap(fyne.NewSize(10, 5000)))
	window := test.NewWindow(area)
	defer window.Close()
	window.Resize(fyne.NewSize(200, 200))
	area.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -13)})
	if area.Offset.Y < 12.5 || area.Offset.Y > 13.5 {
		t.Errorf("a turn the system measured as 13 moved the area %.1f", area.Offset.Y)
	}
}
