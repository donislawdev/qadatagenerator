package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
)

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
