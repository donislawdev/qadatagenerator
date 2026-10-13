package shell

import (
	"image/color"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
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
	frame := New(screens(), Brand{})
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
	frame := New(all, Brand{})
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
	frame := New(screens(), Brand{})
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
	frame := New(nil, Brand{})
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
	}, Brand{})

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
	frame := New(all, Brand{})

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
			// 🔴 The check on importance is gone because importance is gone:
			// this button draws itself since 2026-09-09 and has no such field,
			// so there is no longer a way for a name in this column to be
			// painted by the toolkit at all. What replaced it is stricter and
			// sits one line up - the mark has to DIFFER between the two states,
			// which is the thing the importance check was protecting.
			_ = button
		}
	}
}

// The product's name and its promise are on screen, and neither is there when
// the frame was not given one.
//
// 🔴 Both halves, and the second is the one worth having. A frame that always
// drew a mark would pass the first half while making every test window carry a
// brand nobody asked it for, and the two call sites that build this frame are
// not the only things that build it: every test in this package does.
//
// The words come from the caller rather than from here for the reason the type
// says: the version lives in a package this frame has no business importing and
// the wording belongs to a translator. So this asks that what it was handed
// reaches the screen, which is the only part of it this package decides.
func TestTheProductSaysWhatItIsAndWhatItPromises(t *testing.T) {
	frame := New(screens(), Brand{
		Name:    "QA Data Generator",
		Mark:    paint.IconBolt,
		Version: "v9.9.9",
		Promise: "fully offline",
	})
	drawn := spokenIn(frame.Content())

	for _, want := range []string{"QA Data Generator", "v9.9.9", "fully offline"} {
		if !strings.Contains(drawn, want) {
			t.Errorf("the frame does not say %q anywhere, so the brand it was handed went nowhere", want)
		}
	}
	// The version and the promise are one line rather than two, which is what
	// makes them read as one note about the build instead of two facts stacked.
	if !strings.Contains(drawn, "v9.9.9"+brandSeparator+"fully offline") {
		t.Error("the version and the promise are not joined into one line")
	}

	bare := spokenIn(New(screens(), Brand{}).Content())
	if strings.Contains(bare, brandSeparator) {
		t.Error("a frame given no brand still draws the line that separates its parts")
	}
}

// spokenIn is every word drawn anywhere in this object.
//
// A walk of the tree rather than a look at the fields, because what this asks
// is whether the words REACH the screen, and a field holding them proves only
// that somebody stored them.
func spokenIn(object fyne.CanvasObject) string {
	var said strings.Builder
	var walk func(fyne.CanvasObject)
	walk = func(node fyne.CanvasObject) {
		switch typed := node.(type) {
		case *canvas.Text:
			said.WriteString(typed.Text + "\n")
		case *widget.Label:
			said.WriteString(typed.Text + "\n")
		case *fyne.Container:
			for _, child := range typed.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(typed).Objects() {
				walk(child)
			}
		}
	}
	walk(object)
	return said.String()
}
