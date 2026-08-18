package shell

import (
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

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

// The screen being read is marked by weight rather than by colour alone.
//
// 🔴 Measured rather than assumed: the meaning colours of this palette differ
// from one another by a contrast of 1.18 to 1.41, so somebody who does not
// separate hues sees one shade of grey. A mark carried by colour alone is a
// mark they do not have.
func TestTheScreenBeingReadIsMarkedWithoutRelyingOnColour(t *testing.T) {
	frame := New(screens())
	frame.Open(1)

	if frame.buttons[1].Importance != widget.HighImportance {
		t.Error("the name of the screen being read carries no weight of its own")
	}
	if frame.buttons[0].Importance == widget.HighImportance {
		t.Error("the name of a screen nobody is reading is marked as if it were")
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
