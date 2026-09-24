package parts

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

func sameColour(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// drawnBy collects every rectangle and every piece of text a control draws,
// through its renderer, since a control that draws itself keeps its shapes
// behind one.
func drawnBy(control fyne.Widget) (fills []color.Color, inks []color.Color) {
	var walk func(fyne.CanvasObject)
	walk = func(each fyne.CanvasObject) {
		switch shape := each.(type) {
		case *canvas.Rectangle:
			fills = append(fills, shape.FillColor)
		case *canvas.Text:
			inks = append(inks, shape.Color)
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(shape).Objects() {
				walk(child)
			}
		}
	}
	walk(control)
	return fills, inks
}

func holds(colours []color.Color, wanted color.Color) bool {
	for _, each := range colours {
		if sameColour(each, wanted) {
			return true
		}
	}
	return false
}

// 🔴 The filled button is painted from the two tokens that were measured as a
// pair, and not from two colours that happen to look right.
//
// text.on-fill is read ON fill.primary and measures 4.75 there. neutral.12,
// the ink every other label in this window uses, measures 4.12 on this fill and
// misses the 4.5 a label needs, so the one thing that could silently go wrong
// here is somebody making this label consistent with its neighbours.
//
// Moved here from the test of the separate control this used to be, on
// 2026-09-24, when the filled button became a flag on Button.
func TestAFilledButtonPaintsItselfFromTheTokens(t *testing.T) {
	fills, inks := drawnBy(NewPrimaryButton("Generate this dataset", nil))
	if !holds(fills, palette.Ink(palette.FillPrimary)) {
		t.Errorf("no shape of the filled button is fill.primary, it drew %v", fills)
	}
	if !holds(inks, palette.Ink(palette.TextOnFill)) {
		t.Errorf("the label is not text.on-fill, it drew %v", inks)
	}
	if holds(inks, palette.Ink(palette.TextPrimary)) {
		t.Error("the label took the ordinary text ink, which measures 4.12 on this fill and misses 4.5")
	}

	quiet, _ := drawnBy(NewButton("Copy", nil))
	if holds(quiet, palette.Ink(palette.FillPrimary)) {
		t.Error("an ordinary button is filled too, so a screen no longer says which of its buttons it is for")
	}
}

// 🔴 A disabled filled button drops the fill.
//
// A control nobody can press is not a control to find. Left filled, the loudest
// thing on the screen would be the one thing that does nothing. The recipe
// screen opens in exactly this state, so it is the first filled button anybody
// sees there.
func TestADisabledFilledButtonStopsBeingFilled(t *testing.T) {
	button := NewPrimaryButton("Generate this dataset", nil)
	button.Disable()
	fills, inks := drawnBy(button)
	if holds(fills, palette.Ink(palette.FillPrimary)) {
		t.Error("a button nobody can press is still painted as the action of the screen")
	}
	if holds(inks, palette.Ink(palette.TextOnFill)) {
		t.Error("a disabled button keeps the ink meant to be read on a fill it no longer has")
	}
}

// TestMain starts an application WEARING THE WINDOW'S OWN THEME.
//
// 🔴 It wore the toolkit's default until 2026-09-09, and that is a measurement
// error rather than a detail: these are the controls of this window, and the
// numbers they are built from - the step a word is inset by, the height of a
// control, the thickness of an edge - all come from the theme. A guard asking
// where the words of a field start got 16 in the test and 12 in the window,
// which is the answer to a question nobody was asking.
func TestMain(m *testing.M) {
	application := test.NewApp()
	application.Settings().SetTheme(fynetheme.Visual{})
	m.Run()
}

// A button is pressed by Enter as well as by the space bar.
//
// 🔴 This is the guard for silence, seen on a real window on 2026-08-18: the
// keyboard stood on the button that opens a manifest, Enter did nothing at all,
// and only the space bar worked. A person on this desktop presses Enter first,
// and a control that ignores it is indistinguishable from a program that stopped
// answering.
func TestAButtonIsPressedByEnterAndBySpace(t *testing.T) {
	for _, key := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace} {
		presses := 0
		button := NewButton("open manifest", func() { presses++ })
		button.TypedKey(&fyne.KeyEvent{Name: key})
		if presses != 1 {
			t.Errorf("%s pressed the button %d times rather than once", key, presses)
		}
	}
}

// The space bar arriving as a rune presses it too, and a letter does not.
//
// A focusable widget is handed the space bar through TypedRune as well as
// through TypedKey depending on the driver, so a control answering only one of
// them works on one machine and is silent on another.
func TestAButtonIsPressedByTheSpaceRune(t *testing.T) {
	presses := 0
	button := NewPrimaryButton("generate this dataset", func() { presses++ })
	button.TypedRune(' ')
	if presses != 1 {
		t.Errorf("the space rune pressed the button %d times rather than once", presses)
	}
	button.TypedRune('a')
	if presses != 1 {
		t.Error("a letter pressed the button, which the pointer cannot do")
	}
}

// A key that means nothing here reaches the toolkit rather than pressing.
func TestAnotherKeyDoesNotPressTheButton(t *testing.T) {
	presses := 0
	button := NewButton("open manifest", func() { presses++ })
	for _, key := range []fyne.KeyName{fyne.KeyEscape, fyne.KeyTab, fyne.KeyDown, fyne.KeyA} {
		button.TypedKey(&fyne.KeyEvent{Name: key})
	}
	if presses != 0 {
		t.Errorf("keys that mean nothing on a button pressed it %d times", presses)
	}
}

// A button nobody can press with the pointer is not pressable with the keyboard
// either.
//
// 🔴 The screen has one such button: the one that opens a response record is
// unavailable until there is a manifest to judge it against. A key that reached
// past that would open a comparison with nothing on the other side of it.
func TestADisabledButtonIgnoresEnter(t *testing.T) {
	presses := 0
	button := NewButton("open application response", func() { presses++ })
	button.Disable()
	button.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if presses != 0 {
		t.Error("a disabled button was pressed by the keyboard, which the pointer cannot do")
	}
}

// A button with nothing behind it does not take the window down.
func TestAButtonWithNoHandlerSurvivesTheKey(t *testing.T) {
	button := NewButton("nothing behind this", nil)
	button.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	button.TypedKey(nil)
}
