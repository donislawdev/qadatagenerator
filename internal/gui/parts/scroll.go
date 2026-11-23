package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Scroll is the toolkit's scrolling container, with one notch of a mouse wheel
// moving it as far as the system says a notch moves.
//
// Every scrolling area of this window moves that way: KeyScroll and
// ScrolledRows by embedding this type, the list of a chooser by being one, and
// the table and the recipe editor, which keep their scrolling inside the
// toolkit, through wheelLayer.
//
// # Why the toolkit's own step is not kept
//
// The toolkit moves every area by a fixed step whatever the system is set to,
// palette.ToolkitWheelStep. On Windows and Linux that is 25 units, less
// than half of the three lines of text one notch moves in every other program
// out of the box. The owner reported the window as scrolling slowly on
// 2026-11-17 (B-420). Measured on 2026-11-23 on the trees tools/podglad draws at
// the size the window opens in: the About page holds 1659 units below its fold,
// which is 67 notches from the top to the bottom at the toolkit's step. The
// neighbouring project took the same report about the same toolkit the same
// week and measured a notch there at 4 to 7 ms on the real window, under a frame
// of 60 Hz, so what was slow was the length of a step rather than the drawing.
// That cost was not measured here, and B-420 says why.
//
// # Why an extension rather than a copy
//
// Everything a scroll does stays the toolkit's: the bars, the offset, the
// shadows, a drag on the bar, the clamping at both ends. Only the distance a
// wheel asks for is answered here, and one type answers it for every area, so
// how the wheel feels is changed in one place.
type Scroll struct {
	container.Scroll
}

// NewScroll makes content scrollable up and down.
//
// Vertical only, for the reason NewKeyScroll gives: every scrolling area in
// this window is a column.
func NewScroll(content fyne.CanvasObject) *Scroll {
	scroll := &Scroll{}
	scroll.setUp(content)
	// Before the toolkit's own constructor would, because the first call wins:
	// the area has to be drawn and refreshed as this type, or the wheel would
	// reach the toolkit's scroll underneath and never this one.
	scroll.ExtendBaseWidget(scroll)
	return scroll
}

// setUp is the area before the toolkit is told what object it is, so that a
// type built on this one is the same area and extends the base widget itself.
func (s *Scroll) setUp(content fyne.CanvasObject) {
	s.Direction = container.ScrollVerticalOnly
	s.Content = content
}

// Scrolled moves the area by the system's distance for the notches turned.
func (s *Scroll) Scrolled(event *fyne.ScrollEvent) {
	s.Scroll.Scrolled(wheelTurned(event, s.Size().Height))
}

// WheelLines is how many lines one notch of the wheel moves, as the system is
// set: a negative number is a whole view at a time and nought is not at all.
//
// The window answers it from the system where the system has the setting,
// which is Windows (cmd/qadg-gui/wheel_windows.go), and the default stands
// everywhere else and in every test.
var WheelLines = func() int { return palette.WheelLinesDefault }

// wheelTurned is the event a wheel turned by event asks of an area showing view
// units: the same event, with the distance up and down answered here. A turn to
// the side, which a touchpad or Shift with the wheel makes, keeps the
// toolkit's distance, because no scrolling area of this window moves sideways
// and the system's setting is about lines.
func wheelTurned(event *fyne.ScrollEvent, view float32) *fyne.ScrollEvent {
	moved := *event
	moved.Scrolled.DY = wheelDistance(event.Scrolled.DY, view)
	return &moved
}

// wheelLine is one line of the window's text, measured rather than written, so
// a notch keeps its meaning when the text grows or the screen is scaled.
func wheelLine() float32 {
	return fyne.MeasureText("Ag", palette.TypeSize(palette.TypeBody), fyne.TextStyle{}).Height
}

// notchDistance is how far one notch moves an area showing view units.
func notchDistance(view float32) float32 {
	lines := WheelLines()
	if lines < 0 {
		return view
	}
	return float32(lines) * wheelLine()
}
