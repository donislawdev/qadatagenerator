package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// three is a question with three answers that counts which one was given.
func three() (*Question, *[3]int) {
	given := &[3]int{}
	question := NewQuestion("Keep what you wrote?",
		Reply{Words: "Cancel", Do: func() { given[0]++ }},
		Reply{Words: "Don't save", Do: func() { given[1]++ }},
		Reply{Words: "Save", Do: func() { given[2]++ }})
	return question, given
}

// 🔴 Escape gives the answer that changes nothing, from whichever button the
// keyboard is on, so a person backing out of a dialog they did not expect
// loses nothing.
func TestEscapeGivesTheWayOut(t *testing.T) {
	for at := range 3 {
		question, given := three()
		question.buttons[at].TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if *given != [3]int{1, 0, 0} {
			t.Errorf("Escape on button %d gave %v, want only the way out", at, *given)
		}
	}
}

// The main answer is the last, drawn filled, and the keyboard stands on it when
// the dialog opens, so Enter gives it.
func TestTheDialogOpensWithTheKeyboardOnTheMainAnswer(t *testing.T) {
	question, given := three()
	window := test.NewWindow(nil)
	defer window.Close()
	window.Resize(fyne.NewSize(900, 600))
	question.Show(window, "Save the recipe?")

	main := question.buttons[2]
	if !main.Filled || question.buttons[0].Filled || question.buttons[1].Filled {
		t.Error("the main answer is not the one filled button")
	}
	if window.Canvas().Focused() != main {
		t.Fatalf("the keyboard is on %v rather than on the main answer", window.Canvas().Focused())
	}
	main.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if *given != [3]int{0, 0, 1} {
		t.Errorf("Enter gave %v, want the main answer once", *given)
	}
	if overlay := window.Canvas().Overlays().Top(); overlay != nil {
		t.Error("the dialog is still over the window after it was answered")
	}
}

// An answer is carried out once, whatever arrives after it before the dialog
// has gone: a key held down, a second click.
func TestAnAnswerIsCarriedOutOnce(t *testing.T) {
	question, given := three()
	question.Answer(1)
	question.Answer(1)
	question.Answer(2)
	if *given != [3]int{0, 1, 0} {
		t.Errorf("three presses gave %v, want the first one only", *given)
	}
}
