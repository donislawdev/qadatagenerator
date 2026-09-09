package shell

import (
	"image/color"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// A control of this toolkit builds its renderer against the running
// application, so refreshing one without an application is a nil dereference
// rather than a quiet no operation. The window has an application by the time
// any of this runs, and a test has to say so.
func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

func screens() []Screen {
	return []Screen{
		{Name: "Manifest browser", Content: widget.NewLabel("the browser")},
		{Name: "About", Content: widget.NewLabel("the about screen")},
	}
}

// The frame opens on the first screen rather than on nothing.
//
// A window that starts empty looks like one that failed to load, and a person
// meeting it has no way to tell those apart.
func TestTheFrameOpensOnTheFirstScreen(t *testing.T) {
	frame := New(screens())
	showing, any := frame.Showing()
	if !any {
		t.Fatal("the frame shows no screen at all")
	}
	if showing != 0 {
		t.Errorf("the frame opened on screen %d rather than on the first one", showing)
	}
}

// Every screen has a name that can be pressed, and pressing it shows that
// screen.
func TestEveryScreenCanBeOpenedByItsName(t *testing.T) {
	all := screens()
	frame := New(all)
	if len(frame.buttons) != len(all) {
		t.Fatalf("%d screens have %d names to press", len(all), len(frame.buttons))
	}
	for index, name := range frame.buttons {
		if name.Text != all[index].Name {
			t.Errorf("the name at position %d reads %q and the screen there is %q",
				index, name.Text, all[index].Name)
		}
		name.OnTapped()
		showing, _ := frame.Showing()
		if showing != index {
			t.Errorf("pressing %q showed screen %d", name.Text, showing)
		}
	}
}

// Asking for a screen that is not there leaves the frame as it was.
//
// A frame that emptied itself on a bad index would turn a mistake in one place
// into a window showing nothing, which reads to a person as the program having
// crashed.
func TestAskingForAScreenThatIsNotThereChangesNothing(t *testing.T) {
	frame := New(screens())
	frame.Open(1)
	for _, absent := range []int{-1, len(screens())} {
		frame.Open(absent)
		showing, any := frame.Showing()
		if !any || showing != 1 {
			t.Errorf("asking for screen %d left the frame showing %d, present %v", absent, showing, any)
		}
	}
}

// The column of names keeps one width whatever is in it.
//
// Otherwise it would change width when a screen is added, when a name is
// translated and when the desktop scales, so everything beside it would shift
// sideways for no reason the person caused.
func TestTheColumnOfNamesKeepsOneWidth(t *testing.T) {
	layout := fixedWidth{width: palette.SidebarWidth}
	narrow := widget.NewLabel("A")
	wide := widget.NewLabel("a name considerably longer than the one above it")

	for _, object := range []fyne.CanvasObject{narrow, wide} {
		if got := layout.MinSize([]fyne.CanvasObject{object}).Width; got != palette.SidebarWidth {
			t.Errorf("the column asks for %v rather than for the width in the theme, %v",
				got, palette.SidebarWidth)
		}
	}

	layout.Layout([]fyne.CanvasObject{wide}, fyne.NewSize(palette.SidebarWidth*2, 400))
	if wide.Size().Width != palette.SidebarWidth {
		t.Errorf("the column laid its content out at %v rather than at %v",
			wide.Size().Width, palette.SidebarWidth)
	}
}

// A frame built with no screens at all shows nothing and says so, rather than
// pretending there is a screen to open.
func TestAFrameWithNoScreensSaysThereIsNothingShowing(t *testing.T) {
	frame := New(nil)
	if _, any := frame.Showing(); any {
		t.Error("a frame built with no screens claims to be showing one")
	}
	if frame.Content() == nil {
		t.Error("a frame with no screens builds no content at all, so a window would have nothing to hold")
	}
}

// The keyboard meets the names of the screens before the screen itself.
//
// 🔴 Measured on 2026-08-18 before this was true: Tab reached the control inside
// the screen first, and the column of names, which stands down the left and is
// what a person reads first, came last. The drawing said nothing about it. The
// cause is that the convenient way of building a border hands the middle over
// before the edges, and the keyboard walks the order it was handed.
func TestTheKeyboardMeetsTheNamesBeforeTheScreen(t *testing.T) {
	// Built with the same button the screens use, so the walk meets the type a
	// person actually reaches rather than a stand in of another kind.
	inside := parts.NewButton("something inside the screen", func() {})
	frame := New([]Screen{
		{Name: "Manifest browser", Content: inside},
		{Name: "About", Content: widget.NewLabel("the about screen")},
	})

	window := test.NewWindow(frame.Content())
	defer window.Close()
	window.Resize(fyne.NewSize(900, 600))

	canvas := window.Canvas()
	wanted := []string{"Manifest browser", "About", "something inside the screen"}
	for at, name := range wanted {
		canvas.FocusNext()
		button, ok := canvas.Focused().(*parts.Button)
		if !ok {
			t.Fatalf("stop %d of the walk is %T rather than a button", at+1, canvas.Focused())
		}
		if button.Text != name {
			t.Errorf("stop %d of the walk is %q, and reading the window it should be %q",
				at+1, button.Text, name)
		}
	}
}

// importancePaints is every importance that gives a button a colour of its
// own, and the colour it gives.
//
// 🔴 Read out of the toolkit rather than remembered, at widget/button.go:383
// in fyne v2.8.0, where buttonColorNames turns an importance into a pair of
// theme colours. Medium and Low are absent on purpose: Low leaves the label on
// the ordinary foreground and asks for a background only while the focus or
// the pointer is on it, and that background is the button colour rather than
// any colour carrying meaning here.
var importancePaints = map[widget.Importance]string{
	widget.HighImportance:    "in the primary colour",
	widget.DangerImportance:  "in the error colour",
	widget.WarningImportance: "in the warning colour",
	widget.SuccessImportance: "in the success colour",
}

// The screen a person is on is marked by a shape, not by a colour alone.
//
// 🔴 Colour alone was what it used to be, and it collided with the one other
// thing this window says in colour: where the keyboard is. Both were the same
// measured value, so on a real window the mark of the open screen and the mark
// of focus were the same picture. docs/17 asks for a shape beside any colour
// carrying meaning, and this is that shape.
//
// 🔴 The check on importance asked for exactly MediumImportance until
// 2026-09-08, which was BROADER THAN ITS OWN REASON and wrong about the fact
// it stated: it reported that importance 2 paints the name in the colour
// belonging to focus, and importance 2 is Low, which paints nothing. High is
// the one that does. So the check refused a value for a reason that was not
// true of it, and let Danger, Warning and Success through, which do exactly
// what it was written to stop. It now names the importances that paint, and
// the register above says where that list was read.
func TestTheOpenScreenIsMarkedByAShapeRatherThanByColourAlone(t *testing.T) {
	all := screens()
	frame := New(all)

	for index := range all {
		frame.Open(index)
		for at, button := range frame.buttons {
			// 🔴 The bar rather than the icon, rewritten 2026-09-09 when the
			// mark moved off the icon slot.
			//
			// It asked whether the button carried an icon, which was the right
			// question while the icon WAS the mark and became the wrong one
			// the day every screen gained an icon of its own to say what it is.
			// Left as it was, this would have passed for the worst possible
			// reason: five icons present, so five names "marked", so nothing
			// distinguishing them - the exact fault it exists to catch.
			//
			// The rule it holds has not moved an inch. What says which screen
			// is open must be something a person who does not separate hues can
			// see, and a bar that is there or is not is such a thing, in the
			// same way an icon that is there or is not was.
			//
			// Asked as a DIFFERENCE rather than as a presence, for the reason
			// this project keeps relearning: a mark painted the same in both
			// states passes any test that only asks whether a colour was set.
			marked := frame.items[at].mark.FillColor != color.Transparent
			if at == index && !marked {
				t.Errorf("screen %d is open and its name carries no mark, so only colour says which screen it is",
					index)
			}
			if at != index && marked {
				t.Errorf("screen %d is not open and its name carries the mark, so two names claim to be current",
					at)
			}
			if colour, painted := importancePaints[button.Importance]; painted {
				t.Errorf("the name of screen %d asks for importance %v, which paints the button %s, and a "+
					"name marked by a colour is a mark somebody who does not separate hues cannot see",
					at, button.Importance, colour)
			}
		}
	}
}
