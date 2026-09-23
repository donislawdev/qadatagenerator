package screens

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryStopOfTheKeyboardDrawsExactlyOneFocusRing walks the keyboard round
// a screen the way a person does and, at every stop, counts the rings in
// mark.focus that are on the screen. The answer has to be one.
//
// # 🔴 What it was written after, measured on the real window
//
// tools/ognisko.py on 2026-09-11: fourteen stops on the Quick screen, twelve of
// them with a mark of 3.03 to 14.26, and two - the count field and the card of
// values - with ZERO pixels different from the window with no focus on it at
// all. Both controls had code that draws a mark, and every test of both was
// green, because every test asked the CONTROL whether it had drawn one and none
// asked the SCREEN whether the drawing was on it. The mark of the count field
// lived on a ground nobody had mounted, since the field was dressed again on
// every draw and the dressing before the last one was the one on screen. The
// frame of the card was a rectangle the screen had never been handed (B-279).
//
// # Why exactly one, and why the whole window rather than the control
//
// Nought is the fault above. Two would be a mark left behind on the previous
// stop, which tools/ognisko.py catches by differencing against the neighbour
// and which nothing else in this package would notice. And the count is taken
// over the whole window rather than over the focused control, because the fault
// is precisely that a control and its mark can be two objects with only one of
// them on the screen.
//
// # Which screens
//
// The ones built on paint, named in the list below, which since 2026-09-23 is
// all five. That is the event CLAUDE.md names as the condition for the gate and
// the mutations to come back.
func TestEveryStopOfTheKeyboardDrawsExactlyOneFocusRing(t *testing.T) {
	// The take screen joined on 2026-09-15 (DEC-245), in the state it opens
	// in and in the one with a file read, since the two walk different
	// controls: the card and the panel hold different things. The recipe screen
	// joined the same day (DEC-246), in the empty state and in one with a
	// preview and somewhere to write, which between them walk every control it
	// has: the empty one reaches the example button, the ready one the filled
	// button that writes and a table. The manifest browser joined the same
	// evening (DEC-247), in its three states: the empty one walks the section
	// over the shape of the table, the compared one adds the section holding
	// what the comparison could not see and the table and the card under it.
	// The About screen joined on 2026-09-23 (DEC-250) in the one state it has,
	// which walks both buttons of the band and the page: before that day the
	// picture of it had the buttons disabled and walked two stops fewer than the
	// window (B-283).
	built := []string{"quick-opens", "take-empty", "take-plan", "recipe-empty", "recipe-ready",
		"browser-empty", "browser-promised", "browser-compared", "about"}
	data := sampleWithSource(t)

	for _, name := range built {
		state, found := stateNamed(name)
		if !found {
			t.Fatalf("no state is called %q, so this walk has no screen to start from", name)
		}
		t.Run(name, func(t *testing.T) {
			content := Assemble(i18n.English, data, state)
			window := test.NewWindow(content)
			defer window.Close()
			// Twice and at two sizes, for the reason render() gives: one pass
			// leaves every wrapping sentence a line taller than it is.
			window.Resize(fyne.NewSize(size.Width-1, size.Height))
			window.Resize(size)

			canvas := window.Canvas()
			canvas.Focus(nil)
			if rings := focusRingsOn(content); rings != 0 {
				t.Fatalf("%d rings in mark.focus before the keyboard has been anywhere", rings)
			}

			canvas.FocusNext()
			first := canvas.Focused()
			if first == nil {
				t.Fatal("the first Tab landed nowhere, so this screen offers the keyboard nothing")
			}
			stops := 0
			for {
				stops++
				// A walk that never comes back round is a walk with no end,
				// and the number is well past any screen this window has.
				if stops > 100 {
					t.Fatal("a hundred Tabs and the walk has not come back to where it started")
				}
				if rings := focusRingsOn(content); rings != 1 {
					t.Errorf("stop %d, on %T: %d rings in mark.focus on the screen, want exactly 1. "+
						"The control may well believe it drew one, so ask what is mounted rather "+
						"than what was built", stops, canvas.Focused(), rings)
				}
				canvas.FocusNext()
				if canvas.Focused() == first {
					break
				}
			}
			// The canary. The column of screens alone is five stops, so a walk
			// shorter than that reached nothing of the screen and proved
			// nothing about it.
			if stops < 6 {
				t.Errorf("the walk closed after %d stops, which is the column of screens and "+
					"nothing of the screen itself", stops)
			}
		})
	}
}

// stateNamed finds one state by the name a picture of it carries.
func stateNamed(name string) (State, bool) {
	for _, state := range All() {
		if state.Name == name {
			return state, true
		}
	}
	return State{}, false
}

// focusRingsOn counts the marks of focus a person would see: stroked shapes in
// mark.focus that are visible, with nothing hidden above them.
//
// It walks what is DRAWN rather than what was built, which is the whole
// difference between this and the test each control has of itself: a ground
// somebody hid, or one nobody put in the tree, draws nothing, however well the
// control that owns it believes it is marked.
func focusRingsOn(object fyne.CanvasObject) int {
	found := 0
	mark := rgba(palette.Ink(palette.MarkFocus))
	var walk func(fyne.CanvasObject)
	walk = func(each fyne.CanvasObject) {
		if each == nil || !each.Visible() {
			return
		}
		switch shape := each.(type) {
		case *canvas.Rectangle:
			if shape.StrokeWidth > 0 && rgba(shape.StrokeColor) == mark {
				found++
			}
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		case fyne.Widget:
			// Through the renderer, because a control that draws itself keeps
			// its shapes behind one, and a walk over containers alone would
			// report that a focused button draws nothing.
			for _, child := range test.WidgetRenderer(shape).Objects() {
				walk(child)
			}
		}
	}
	walk(object)
	return found
}
