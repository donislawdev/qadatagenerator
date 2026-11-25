package parts

import (
	"fmt"
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The wheel over a panel of rows stops with a whole row at the top, in both
// panels a picker makes, at every distance the system can set a notch to, and
// whether it is turned a notch at a time or a touchpad's fraction at a time
// (B-429). The wheel goes through test.Scroll, which finds the area under the
// pointer the way the driver of a real window does. On macOS the system's
// setting is not read (wheel_darwin.go), so there every turn of the loop below
// is the same notch, which is what the system hands over.
//
// The canary is the distance a notch moves on its own: had it been a whole
// number of rows, the panel would stop on a row without rowScroll and the test
// could not tell the fault from its cure.
func TestTheWheelStopsAPanelOnAWholeRow(t *testing.T) {
	was := WheelLines
	t.Cleanup(func() { WheelLines = was })

	panels := []struct {
		name  string
		build func() *picker
	}{
		{"suggestions", func() *picker {
			return newSuggestionPanel(manyChoices(false), func(string) {}, func() {}, func(bool) {})
		}},
		{"chooser", func() *picker {
			return newPicker(manyChoices(true), "", words(), func(string) {}, func() {})
		}},
	}
	for _, kind := range panels {
		t.Run(kind.name, func(t *testing.T) {
			panel, window := panelOnWindow(t, kind.build())
			at := panel.Position().Add(fyne.NewPos(panel.Size().Width/2, panel.Size().Height/2))
			area := panel.scroll
			turn := func(notches float32) {
				test.Scroll(window.Canvas(), at, 0, -notches*palette.ToolkitWheelStep)
			}

			WheelLines = func() int { return palette.WheelLinesDefault }
			notch := wheelDistance(palette.ToolkitWheelStep, area.Size().Height)
			if isStop(area, notch) {
				t.Fatalf("canary: a notch alone moves %.1f, which is already a stop, so this test proves nothing", notch)
			}
			// Where the system's setting is read, a notch out of the box is
			// longer than any row, so the check that every notch moves is tried.
			if runtime.GOOS != "darwin" && notch <= longestStep(area) {
				t.Fatalf("a notch of %.1f is no longer than the longest step between rows, %.1f, so nothing checks that a notch moves",
					notch, longestStep(area))
			}

			for _, lines := range []int{1, 3, 5, -1} {
				WheelLines = func() int { return lines }
				walkDownAndUp(t, area, lines, turn)
			}

			// A touchpad: ten tenths of a notch end where one notch does.
			WheelLines = func() int { return palette.WheelLinesDefault }
			area.ScrollToOffset(fyne.Position{})
			turn(1)
			whole := area.Offset.Y
			area.ScrollToOffset(fyne.Position{})
			for range 10 {
				turn(0.1)
			}
			if distance(area.Offset.Y, whole) > 0.5 {
				t.Errorf("ten tenths of a notch stood at %.1f and one notch at %.1f", area.Offset.Y, whole)
			}

			// The arrows move the view the way ScrollToOffset does, and the next
			// notch starts from where they left it rather than from the wheel.
			turn(3)
			area.ScrollToOffset(fyne.Position{})
			turn(1)
			if distance(area.Offset.Y, whole) > 0.5 {
				t.Errorf("a notch after the view was put back at the top stood at %.1f, and from the top a notch stands at %.1f",
					area.Offset.Y, whole)
			}
		})
	}
}

// The stop nearest the aim, with the end standing in for a row that begins past
// it, and the higher of two stops as near.
func TestTheStopIsTheNearestRowOrTheEnd(t *testing.T) {
	tops := []float32{0, 20, 40, 60, 80}
	cases := []struct {
		aim, end, want float32
	}{
		{0, 70, 0},
		{9, 70, 0},
		{11, 70, 20},
		{10, 70, 0},
		{58, 70, 60},
		{66, 70, 70},
		{70, 70, 70},
		{5, 0, 0},
	}
	for _, each := range cases {
		if got := stopNear(each.aim, each.end, tops); got != each.want {
			t.Errorf("aim %.0f with the end at %.0f stopped at %.0f, and %.0f was wanted", each.aim, each.end, got, each.want)
		}
	}
	if got := stopNear(33, 50, nil); got != 50 {
		t.Errorf("with no rows the stop is %.0f rather than the end", got)
	}
}

// walkDownAndUp turns the wheel a notch at a time from the top to the end and
// back, and fails on the first view that stands in the middle of a row.
//
// A notch longer than the longest step between two stops has to move the view
// every time, since the aim then leaves the reach of the stop it stood on. A
// shorter one may not: the turns add up and the view moves every few notches,
// which is the case of a single line a notch and of macOS, where the test's
// notch is the toolkit's unscaled distance.
func walkDownAndUp(t *testing.T, area *rowScroll, lines int, turn func(float32)) {
	t.Helper()
	area.ScrollToOffset(fyne.Position{})
	end := area.Content.MinSize().Height - area.Size().Height
	everyNotchMoves := wheelDistance(palette.ToolkitWheelStep, area.Size().Height) > longestStep(area)
	for notch := 0; area.Offset.Y < end; notch++ {
		if notch > 200 {
			t.Fatalf("with %d line(s) a notch, 200 notches did not reach the end at %.1f", lines, end)
		}
		before := area.Offset.Y
		turn(1)
		if !isStop(area, area.Offset.Y) {
			t.Fatalf("with %d line(s) a notch, the view stood at %.1f, in the middle of a row", lines, area.Offset.Y)
		}
		if everyNotchMoves && area.Offset.Y <= before {
			t.Fatalf("with %d line(s) a notch, longer than every step between rows, a notch down from %.1f did not move the view",
				lines, before)
		}
	}
	for notch := 0; area.Offset.Y > 0; notch++ {
		if notch > 200 {
			t.Fatalf("with %d line(s) a notch, 200 notches up did not reach the top", lines)
		}
		turn(-1)
		if !isStop(area, area.Offset.Y) {
			t.Fatalf("with %d line(s) a notch, the view stood at %.1f on the way up", lines, area.Offset.Y)
		}
	}
}

// manyChoices is a list long enough to scroll, under headings when grouped.
func manyChoices(grouped bool) []Choice {
	choices := make([]Choice, 0, 40)
	for index := range 40 {
		choice := Choice{Value: fmt.Sprintf("value_%02d", index), Detail: "what it means"}
		if grouped {
			choice.Group = fmt.Sprintf("Group %d", index/7)
		}
		choices = append(choices, choice)
	}
	return choices
}

// panelOnWindow puts a panel on a window at the height a panel is given, cut to
// end on a whole row as every open panel is.
func panelOnWindow(t *testing.T, panel *picker) (*picker, fyne.Window) {
	t.Helper()
	window := test.NewWindow(container.NewWithoutLayout(panel))
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(900, 700))
	panel.Resize(fyne.NewSize(panel.Width(320), panel.Fitted(240)))
	panel.Move(fyne.NewPos(10, 10))
	if room := panel.scroll.Content.MinSize().Height - panel.scroll.Size().Height; room <= 0 {
		t.Fatalf("the panel shows every row, so there is nothing to scroll (room %.1f)", room)
	}
	return panel, window
}

// longestStep is the furthest apart two neighbouring stops stand.
func longestStep(area *rowScroll) float32 {
	end := area.Content.MinSize().Height - area.Size().Height
	longest, last := float32(0), float32(0)
	for _, top := range append(area.tops(), end) {
		top = min(top, end)
		longest, last = max(longest, top-last), top
	}
	return longest
}

// isStop says the view stands at the top of a row or a heading, or at the end.
func isStop(area *rowScroll, offset float32) bool {
	end := area.Content.MinSize().Height - area.Size().Height
	if distance(offset, end) < 0.5 {
		return true
	}
	for _, top := range area.tops() {
		if distance(offset, top) < 0.5 {
			return true
		}
	}
	return false
}
