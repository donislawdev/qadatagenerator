package parts

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The panel a kind is picked from, driven the way a person drives it: keys go
// to whatever the canvas has focused, and taps go to what is under them.

func kinds() []Choice {
	return []Choice{
		{Value: "ssn", Detail: "000-00-0000", Group: "United States"},
		{Value: "aba", Detail: "000000000", Group: "United States"},
		{Value: "pesel", Detail: "00000000000", Group: "Poland"},
		{Value: "nip", Detail: "0000000000", Group: "Poland"},
		{Value: "iban", Detail: "AA00 0000", Group: "International"},
		{Value: "first_name", Group: "People"},
	}
}

func words() PanelWords {
	return PanelWords{Prompt: "Type to filter", Counted: "kinds", Matches: "%d of %d match",
		Enter: "Enter to pick", Nothing: "Nothing matches"}
}

// opened builds a chooser on a window and opens its panel, handing back both.
func opened(t *testing.T, chosen string) (*Select, fyne.Window) {
	t.Helper()
	chooser := NewSelect(nil, func(string) {})
	chooser.SetChoices(kinds(), words())
	chooser.SetSelected(chosen)
	window := test.NewWindow(chooser)
	window.Resize(fyne.NewSize(900, 700))
	t.Cleanup(window.Close)
	chooser.Open()
	if !chooser.IsOpen() || window.Canvas().Overlays().Top() == nil {
		t.Fatal("the panel did not open, so nothing below would be about it")
	}
	return chooser, window
}

// press sends a key the way the driver does: to whatever holds the keyboard.
func press(t *testing.T, window fyne.Window, key fyne.KeyName) {
	t.Helper()
	focused := window.Canvas().Focused()
	if focused == nil {
		t.Fatal("nothing holds the keyboard, so the key would go nowhere")
	}
	focused.TypedKey(&fyne.KeyEvent{Name: key})
}

// typed sends characters the way the driver does.
func typed(t *testing.T, window fyne.Window, text string) {
	t.Helper()
	for _, r := range text {
		focused := window.Canvas().Focused()
		if focused == nil {
			t.Fatal("nothing holds the keyboard, so the character would go nowhere")
		}
		focused.TypedRune(r)
	}
}

// TestOpeningAGroupedPanelPutsTheKeyboardInTheSearchField holds the first thing
// a person expects: to start typing.
func TestOpeningAGroupedPanelPutsTheKeyboardInTheSearchField(t *testing.T) {
	chooser, window := opened(t, "pesel")
	if _, inField := window.Canvas().Focused().(*searchEntry); !inField {
		t.Fatalf("the keyboard is on %T, not in the search field", window.Canvas().Focused())
	}
	if chooser.panel.current != chooser.panel.indexOf("pesel") {
		t.Errorf("the keyboard row is %d, where the chosen kind stands at %d",
			chooser.panel.current, chooser.panel.indexOf("pesel"))
	}
}

// TestTypingFiltersTheRowsAndTheFootCountsThem holds the search.
func TestTypingFiltersTheRowsAndTheFootCountsThem(t *testing.T) {
	chooser, window := opened(t, "ssn")
	panel := chooser.panel
	if panel.foot.Text != "6 kinds" {
		t.Errorf("before typing the foot says %q", panel.foot.Text)
	}

	typed(t, window, "Pol")
	shown := []string{}
	for _, row := range panel.rows {
		if row.shown {
			shown = append(shown, row.choice.Value)
		}
	}
	if strings.Join(shown, ",") != "pesel,nip" {
		t.Errorf("typing Pol left %v on screen, where the heading Poland holds pesel and nip", shown)
	}
	if panel.foot.Text != "2 of 6 match" {
		t.Errorf("after typing the foot says %q", panel.foot.Text)
	}
	if panel.current != panel.indexOf("pesel") {
		t.Errorf("the keyboard is on row %d rather than on the first match", panel.current)
	}
	for _, row := range panel.rows {
		if row.heading != nil && row.heading.Visible() != (row.choice.Group == "Poland") {
			t.Errorf("the heading %q is shown %v", row.choice.Group, row.heading.Visible())
		}
	}
}

// TestAFilterThatMatchesNothingSaysSoAndEnterDoesNothing holds the empty state:
// a list that quietly goes blank reads as a list that broke.
func TestAFilterThatMatchesNothingSaysSoAndEnterDoesNothing(t *testing.T) {
	chooser, window := opened(t, "ssn")
	panel := chooser.panel
	typed(t, window, "zzz")
	if panel.scroll.Visible() || !panel.nothing.Visible() {
		t.Error("nothing matches and the rows are still shown, or the sentence saying so is not")
	}
	if panel.foot.Text != "0 of 6 match" {
		t.Errorf("the foot says %q", panel.foot.Text)
	}
	press(t, window, fyne.KeyReturn)
	if !chooser.IsOpen() || chooser.Selected != "ssn" {
		t.Errorf("Enter on nothing changed the choice to %q or closed the panel", chooser.Selected)
	}
}

// TestTheArrowsWalkTheRowsAndEnterPicksOne holds the keyboard road end to end.
func TestTheArrowsWalkTheRowsAndEnterPicksOne(t *testing.T) {
	picked := ""
	chooser := NewSelect(nil, func(value string) { picked = value })
	chooser.SetChoices(kinds(), words())
	chooser.SetSelected("ssn")
	window := test.NewWindow(chooser)
	window.Resize(fyne.NewSize(900, 700))
	defer window.Close()
	chooser.Open()

	press(t, window, fyne.KeyDown)
	press(t, window, fyne.KeyDown)
	if chooser.panel.current != chooser.panel.indexOf("pesel") {
		t.Fatalf("two presses of Down put the keyboard on row %d", chooser.panel.current)
	}
	press(t, window, fyne.KeyUp)
	press(t, window, fyne.KeyReturn)
	if chooser.IsOpen() {
		t.Error("Enter picked and the panel stayed open")
	}
	if chooser.Selected != "aba" || picked != "aba" {
		t.Errorf("the chooser holds %q and the screen was told %q, where aba was picked", chooser.Selected, picked)
	}
	if window.Canvas().Overlays().Top() != nil {
		t.Error("the overlay is still on the canvas after the pick")
	}
}

// TestTheArrowsStopAtTheEndsRatherThanWrapping holds what a person holding a
// key down arrives at.
func TestTheArrowsStopAtTheEndsRatherThanWrapping(t *testing.T) {
	chooser, window := opened(t, "first_name")
	press(t, window, fyne.KeyDown)
	if chooser.panel.current != len(kinds())-1 {
		t.Errorf("Down on the last row moved the keyboard to %d", chooser.panel.current)
	}
	for i := 0; i < 10; i++ {
		press(t, window, fyne.KeyUp)
	}
	if chooser.panel.current != 0 {
		t.Errorf("ten presses of Up landed on row %d rather than the first", chooser.panel.current)
	}
}

// TestEscapeAndATapOutsideCloseWithoutChoosing holds the two ways out.
func TestEscapeAndATapOutsideCloseWithoutChoosing(t *testing.T) {
	chooser, window := opened(t, "ssn")
	press(t, window, fyne.KeyDown)
	press(t, window, fyne.KeyEscape)
	if chooser.IsOpen() || chooser.Selected != "ssn" {
		t.Errorf("Escape left the panel open or changed the choice to %q", chooser.Selected)
	}

	chooser.Open()
	if !chooser.IsOpen() {
		t.Fatal("the panel did not open a second time")
	}
	test.TapCanvas(window.Canvas(), fyne.NewPos(880, 680))
	if chooser.IsOpen() || chooser.Selected != "ssn" {
		t.Errorf("a tap outside left the panel open or changed the choice to %q", chooser.Selected)
	}
}

// TestATapOnARowPicksIt holds the pointer road.
func TestATapOnARowPicksIt(t *testing.T) {
	chooser, _ := opened(t, "ssn")
	chooser.panel.rows[chooser.panel.indexOf("iban")].tap.Tapped(nil)
	if chooser.IsOpen() || chooser.Selected != "iban" {
		t.Errorf("tapping the iban row left the panel open or chose %q", chooser.Selected)
	}
}

// TestAFlatListOpensWithoutAFieldAndStillAnswersTheKeyboard holds the other
// shape of the panel, which the locale and gender choosers open.
func TestAFlatListOpensWithoutAFieldAndStillAnswersTheKeyboard(t *testing.T) {
	chooser := NewSelect([]string{"en-US", "pl-PL"}, func(string) {})
	chooser.SetSelected("en-US")
	// Placed by hand at a size of its own, so that the panel has a chooser
	// narrower than the window to take its width from.
	chooser.Resize(fyne.NewSize(160, palette.ControlHeight))
	chooser.Move(fyne.NewPos(20, 20))
	window := test.NewWindow(container.NewWithoutLayout(chooser))
	window.Resize(fyne.NewSize(600, 400))
	defer window.Close()
	chooser.Open()
	if chooser.panel.search != nil || chooser.panel.foot != nil {
		t.Error("a flat list opened with a search field or a foot")
	}
	if width := chooser.panel.Size().Width; width != chooser.Size().Width {
		t.Errorf("a flat list opened %v wide over a chooser %v wide", width, chooser.Size().Width)
	}
	if window.Canvas().Focused() != fyne.Focusable(chooser.panel) {
		t.Fatalf("the keyboard is on %T rather than on the panel", window.Canvas().Focused())
	}
	typed(t, window, "p")
	press(t, window, fyne.KeyReturn)
	if chooser.Selected != "pl-PL" {
		t.Errorf("typing p and Enter chose %q", chooser.Selected)
	}
}

// TestOpeningTwiceOpensOnce holds that a second request does not stack a
// second panel over the first.
func TestOpeningTwiceOpensOnce(t *testing.T) {
	chooser, window := opened(t, "ssn")
	first := chooser.panel
	chooser.Open()
	if chooser.panel != first || len(window.Canvas().Overlays().List()) != 1 {
		t.Errorf("a second Open made a second panel, %d overlays on the canvas", len(window.Canvas().Overlays().List()))
	}
}

// TestThePanelStaysInsideTheCanvas holds the geometry, case by case, on the
// pure function that decides it.
//
// The rows would like 441 and the least the panel can be is 128, which are
// the numbers of the six kinds above read off a probe, and they are written
// here rather than measured so that each case says what it is about.
func TestThePanelStaysInsideTheCanvas(t *testing.T) {
	const wanted, least = 441, 128
	chooser := fyne.NewSize(120, 34)
	gap := palette.StepOf(palette.NameStepHairline)
	cases := []struct {
		name string
		room fyne.Size
		at   fyne.Position
		want string
	}{
		{"room below: under the chooser, as tall as its rows", fyne.NewSize(1120, 780), fyne.NewPos(256, 120), "below"},
		{"near the right edge: slid left", fyne.NewSize(1120, 780), fyne.NewPos(1000, 120), "below"},
		{"near the foot: opens above", fyne.NewSize(1120, 780), fyne.NewPos(256, 700), "above"},
		{"a canvas narrower than the panel: as wide as the canvas", fyne.NewSize(300, 780), fyne.NewPos(10, 120), "below"},
		{"no room either side: least height, kept inside", fyne.NewSize(400, 150), fyne.NewPos(10, 60), "inside"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			position, size := place(c.room, c.at, chooser, fyne.NewSize(palette.PanelWidth, wanted), least)
			right, bottom := position.X+size.Width, position.Y+size.Height
			if position.X < gap || position.Y < gap || right > c.room.Width-gap || bottom > c.room.Height-gap {
				t.Errorf("the panel stands at %v with size %v in a canvas of %v", position, size, c.room)
			}
			if size.Height < least {
				t.Errorf("the panel is %v tall, under the least it can be", size.Height)
			}
			switch c.want {
			case "below":
				if position.Y != c.at.Y+chooser.Height+gap {
					t.Errorf("the panel opened at %v rather than under the chooser", position.Y)
				}
			case "above":
				if bottom > c.at.Y {
					t.Errorf("the panel reaches %v, over the chooser at %v", bottom, c.at.Y)
				}
			}
		})
	}
}

// TestTheChosenRowIsMarkedAndTheKeyboardRowIsLit holds that the two states are
// drawn, by reading the shapes of the faces rather than pixels: the chosen row
// carries the icon of the check on the ground of the chosen, and the keyboard
// row the ground a pointer gives a row. No row is outlined, because an outline
// made the list read as a stack of text fields (DEC-260).
func TestTheChosenRowIsMarkedAndTheKeyboardRowIsLit(t *testing.T) {
	chooser, window := opened(t, "pesel")
	press(t, window, fyne.KeyDown)
	panel := chooser.panel
	if panel.current != panel.indexOf("nip") {
		t.Fatalf("the keyboard is on row %d", panel.current)
	}
	for _, row := range panel.rows {
		row.tap.Refresh()
	}
	chosen := test.WidgetRenderer(panel.rows[panel.indexOf("pesel")].tap).Objects()[0]
	current := test.WidgetRenderer(panel.rows[panel.indexOf("nip")].tap).Objects()[0]
	resting := test.WidgetRenderer(panel.rows[panel.indexOf("iban")].tap).Objects()[0]
	if !hasIcon(chosen) || hasIcon(current) || hasIcon(resting) {
		t.Errorf("the check stands on chosen %v, current %v, resting %v", hasIcon(chosen), hasIcon(current), hasIcon(resting))
	}
	if !hasGround(chosen, palette.BgCurrent) || !hasGround(current, palette.BgHover) ||
		hasGround(resting, palette.BgCurrent) || hasGround(resting, palette.BgHover) {
		t.Errorf("chosen on its ground %v, current lit %v, resting bare %v",
			hasGround(chosen, palette.BgCurrent), hasGround(current, palette.BgHover),
			!hasGround(resting, palette.BgCurrent) && !hasGround(resting, palette.BgHover))
	}
	if hasOutline(chosen) || hasOutline(current) || hasOutline(resting) {
		t.Errorf("a focus outline stands on chosen %v, current %v, resting %v",
			hasOutline(chosen), hasOutline(current), hasOutline(resting))
	}
}

// hasGround says whether a picture holds a shape filled in the given token.
func hasGround(object fyne.CanvasObject, token string) bool {
	found := false
	walkShapes(object, func(shape fyne.CanvasObject) {
		if rectangle, ok := shape.(*canvas.Rectangle); ok && rectangle.FillColor == palette.Ink(token) {
			found = true
		}
	})
	return found
}

// hasIcon says whether a picture holds an image, which only the check is here.
func hasIcon(object fyne.CanvasObject) bool {
	found := false
	walkShapes(object, func(shape fyne.CanvasObject) {
		if _, image := shape.(*canvas.Image); image {
			found = true
		}
	})
	return found
}

// hasOutline says whether a picture holds a shape stroked in the focus mark.
func hasOutline(object fyne.CanvasObject) bool {
	found := false
	walkShapes(object, func(shape fyne.CanvasObject) {
		if rectangle, ok := shape.(*canvas.Rectangle); ok && rectangle.StrokeColor == palette.Ink(palette.MarkFocus) {
			found = true
		}
	})
	return found
}

func walkShapes(object fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(object)
	if holder, ok := object.(*fyne.Container); ok {
		for _, child := range holder.Objects {
			walkShapes(child, visit)
		}
	}
}
