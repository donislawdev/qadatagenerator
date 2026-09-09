package parts

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// drawingOf builds the renderer so that a test can look at what will be
// painted.
//
// The renderer is built here rather than fetched from the widget, because a
// widget that has never been on a canvas has no renderer cached and Refresh on
// it reaches nothing. CreateRenderer settles every colour on its way out, so
// what comes back is the drawing for the state the button is in at that moment.
// Every test below therefore puts the button in its state FIRST and asks for
// the drawing second.
func drawingOf(button *PrimaryButton) *primaryRenderer {
	return button.CreateRenderer().(*primaryRenderer)
}

func sameColour(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

// The primary button is pressed by both Enter keys and by the space bar.
//
// The toolkit's own button answers to the space bar alone, which is the fault
// parts.Button exists to repair. Repeating it in a new control would be paying
// for the same lesson twice.
func TestThePrimaryButtonIsPressedByEnterAndBySpace(t *testing.T) {
	for _, key := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace} {
		presses := 0
		button := NewPrimaryButton("generate this dataset", func() { presses++ })
		button.TypedKey(&fyne.KeyEvent{Name: key})
		if presses != 1 {
			t.Errorf("%s pressed the primary button %d times rather than once", key, presses)
		}
	}
}

// The space bar arriving as a rune presses it too.
//
// A focusable widget is handed the space bar through TypedRune as well as
// through TypedKey depending on the driver, so a control answering only one of
// them works on one machine and is silent on another.
func TestThePrimaryButtonIsPressedByTheSpaceRune(t *testing.T) {
	presses := 0
	button := NewPrimaryButton("generate this dataset", func() { presses++ })
	button.TypedRune(' ')
	if presses != 1 {
		t.Errorf("the space rune pressed the button %d times rather than once", presses)
	}
	button.TypedRune('a')
	if presses != 1 {
		t.Error("a letter pressed the primary button, which the pointer cannot do")
	}
}

// A primary button nobody can press with the pointer is not pressable with the
// keyboard either.
func TestADisabledPrimaryButtonIgnoresEveryKey(t *testing.T) {
	presses := 0
	button := NewPrimaryButton("generate this dataset", func() { presses++ })
	button.Disable()
	for _, key := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace} {
		button.TypedKey(&fyne.KeyEvent{Name: key})
	}
	button.TypedRune(' ')
	button.Tapped(nil)
	if presses != 0 {
		t.Errorf("a disabled primary button was pressed %d times", presses)
	}
}

// A primary button with nothing behind it does not take the window down.
func TestAPrimaryButtonWithNoHandlerSurvivesTheKey(t *testing.T) {
	button := NewPrimaryButton("nothing behind this", nil)
	button.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	button.TypedKey(nil)
	button.TypedRune(' ')
	button.Tapped(nil)
}

// 🔴 The keyboard standing on this button changes what is drawn.
//
// This is the guard nothing else holds. Every other control in this window
// marks focus with the toolkit's flood, which parts.Flooded then keeps
// readable, and both of those are watched by guards over in the screens
// package. This control draws its own focus, so a ring that stops being drawn
// breaks nothing anywhere else and shows up only as a person with a keyboard
// having no idea where they are.
//
// The question asked is the DIFFERENCE between the two states rather than the
// presence of a colour, because a ring painted the same in both states would
// pass a test that only asked whether a colour was set.
func TestTheKeyboardOnThePrimaryButtonIsVisible(t *testing.T) {
	resting := drawingOf(NewPrimaryButton("generate this dataset", nil))

	focused := NewPrimaryButton("generate this dataset", nil)
	focused.FocusGained()
	withKeyboard := drawingOf(focused)

	if sameColour(resting.ring.StrokeColor, withKeyboard.ring.StrokeColor) {
		t.Fatal("the primary button looks the same with and without the keyboard on it")
	}

	// 🔴 The SAME token every other control marks focus with, changed on
	// 2026-09-09. It used to be the ink of the label, which made this the one
	// control in the window saying focus in its own way - and a white ring
	// inside a filled button, which the owner called awful when he saw the
	// sheet of states. mark.focus measures 3.83 against this fill and 3.15
	// against the fill under the pointer.
	mark := palette.Ink(palette.MarkFocus)
	if !sameColour(withKeyboard.ring.StrokeColor, mark) {
		t.Errorf("the focus mark is %v rather than mark.focus %v, so this button marks focus differently from every control beside it",
			withKeyboard.ring.StrokeColor, mark)
	}

	// The keyboard leaving has to put it back, or the mark stays on a control
	// the keyboard is no longer on and every button ends up looking focused.
	focused.FocusLost()
	if !sameColour(drawingOf(focused).ring.StrokeColor, resting.ring.StrokeColor) {
		t.Error("the focus ring stayed after the keyboard left")
	}
}

// 🔴 The fill and the ink are the two tokens that were measured as a pair, and
// not two colours that happen to look right.
//
// text.on-fill is read ON fill.primary rather than on the window ground, and
// measures 5.14 there. A fill taken from anywhere else would leave that
// measurement describing a pair this control does not use.
//
// 🔴 The ink is asked for by NAME rather than compared against white, and that
// is the point of the test rather than a detail of it. neutral.12, the ink
// every other piece of text in this window uses, measures 4.35 on this fill and
// misses the 4.5 a label needs, so the one thing that could silently go wrong
// here is somebody making this label consistent with its neighbours. The token
// register carries that arithmetic and this asks the register.
func TestThePrimaryButtonPaintsItselfFromTheTokens(t *testing.T) {
	drawing := drawingOf(NewPrimaryButton("generate this dataset", nil))

	fill := palette.Ink(palette.FillPrimary)
	if !sameColour(drawing.fill.FillColor, fill) {
		t.Errorf("the slab is %v rather than fill.primary %v", drawing.fill.FillColor, fill)
	}

	ink := palette.Ink(palette.TextOnFill)
	if !sameColour(drawing.label.Color, ink) {
		t.Errorf("the label is %v rather than text.on-fill %v", drawing.label.Color, ink)
	}
	if sameColour(drawing.label.Color, palette.Ink(palette.TextPrimary)) {
		t.Error("the label took the ordinary text ink, which measures 4.35 on this fill and misses 4.5")
	}
}

// 🔴 A disabled primary button drops the fill.
//
// The rule is already written for every other button here: a control nobody can
// press is not a control to find. Left filled, the loudest thing on the screen
// would be the one thing that does nothing, which is worse than no hierarchy at
// all. The recipe screen opens in exactly this state, so this is the first
// primary button anybody sees.
func TestADisabledPrimaryButtonStopsBeingPrimary(t *testing.T) {
	button := NewPrimaryButton("generate this dataset", nil)
	button.Disable()
	drawing := drawingOf(button)

	fill := palette.Named(palette.NameActionPrimary).NRGBA()
	if sameColour(drawing.fill.FillColor, fill) {
		t.Error("a button nobody can press is still painted as the action of the screen")
	}

	ink := palette.Named(palette.NameTextOnPrimary).NRGBA()
	if sameColour(drawing.label.Color, ink) {
		t.Error("a disabled button keeps the ink meant to be read on a fill it no longer has")
	}
}

// The ring is drawn inside the fill rather than on its edge.
//
// Both sides of it have to sit on action.primary for the 4.99 to hold. Half a
// stroke of inset, which is what an edge drawn AROUND a control uses, would put
// the outer half on the window ground, and the window ground is the same
// #1B1B1B as the ink, which measures 1.00.
func TestTheFocusRingSitsInsideTheFill(t *testing.T) {
	button := NewPrimaryButton("generate this dataset", nil)
	button.FocusGained()
	drawing := drawingOf(button)

	size := fyne.NewSize(200, 40)
	drawing.Layout(size)

	inset := drawing.ring.Position().X
	if inset < palette.BorderThickness {
		t.Errorf("the ring is inset by %v, which is less than the whole stroke of %v it needs to stay off the window ground",
			inset, palette.BorderThickness)
	}
	if right := inset + drawing.ring.Size().Width; right > size.Width-palette.BorderThickness+0.01 {
		t.Errorf("the ring reaches %v of a button %v wide, so its far side is on the window ground", right, size.Width)
	}
}
