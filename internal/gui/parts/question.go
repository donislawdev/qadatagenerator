package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Reply is one answer a Question offers: the words on its button and what
// pressing it does.
type Reply struct {
	Words string
	Do    func()
}

// Question is something the window asks a person in a dialog over itself, with
// one button per answer.
//
// # Why not the toolkit's confirm dialog
//
// dialog.NewConfirm offers two answers, and keeping a recipe somebody has not
// saved asks for three: save it, let it go, or stay. Its buttons are the
// toolkit's own button as well, which this window stopped using on 2026-09-09 to
// draw the pointer and the keyboard in its own palette, so a dialog of them
// would be the one place where a button looks unlike every other.
//
// # The order of the answers, and the two the keys give
//
// The first answer is the one that changes nothing, and Escape gives it, so a
// person backing out of a dialog they did not expect loses nothing. The last is
// the one the question is for. It is filled, and the keyboard stands on it when
// the dialog opens, so Enter gives it. They are laid out left to right in that
// order, the main answer on the right where the toolkit's own dialogs put it.
//
// # Why a plain type and not a widget
//
// For the reason Disclosure gives: it draws nothing of its own. It is words and
// a row of buttons in a column, and the dialog around them is the toolkit's.
type Question struct {
	sentence string
	choices  []Reply
	buttons  []*Button
	face     fyne.CanvasObject
	shown    *dialog.CustomDialog
	// answered says an answer was given, so that a second press arriving
	// before the dialog has gone, a key held down or a double click, does not
	// carry out a second answer.
	answered bool
}

// NewQuestion builds a question: a sentence and the answers to it, the way out
// first and the main answer last.
func NewQuestion(sentence string, choices ...Reply) *Question {
	question := &Question{sentence: sentence, choices: choices}
	row := make([]fyne.CanvasObject, 0, len(choices))
	for at, choice := range choices {
		button := NewButton(choice.Words, func() { question.Answer(at) })
		button.OnEscape = func() { question.Answer(0) }
		question.buttons = append(question.buttons, button)
		row = append(row, button)
	}
	if len(question.buttons) > 0 {
		question.buttons[len(question.buttons)-1].Filled = true
	}
	question.face = paint.AtLeast(palette.QuestionWidth, 0,
		paint.Column(palette.NameStepGroup, paint.Paragraph(sentence),
			paint.Trail(paint.Spacer(), paint.Row(palette.NameStepRelated, row...))))
	return question
}

// Says is the sentence the question asks, for a caller with nobody looking.
func (q *Question) Says() string { return q.sentence }

// OnScreen is the words and the buttons, without the dialog around them, which
// is what the catalogue of controls shows.
func (q *Question) OnScreen() fyne.CanvasObject { return q.face }

// Show asks the question in a dialog over the window, under a title, with the
// keyboard on the main answer.
func (q *Question) Show(window fyne.Window, title string) {
	q.shown = dialog.NewCustomWithoutButtons(title, q.face, window)
	q.shown.Show()
	if len(q.buttons) > 0 {
		window.Canvas().Focus(q.buttons[len(q.buttons)-1])
	}
}

// Answer gives the answer at this place, as pressing its button would: the
// dialog goes, and then the answer is carried out, so that a second dialog the
// answer opens stands on the window rather than on this one.
func (q *Question) Answer(at int) {
	if at < 0 || at >= len(q.choices) || q.answered {
		return
	}
	q.answered = true
	if q.shown != nil {
		q.shown.Hide()
		q.shown = nil
	}
	if do := q.choices[at].Do; do != nil {
		do()
	}
}
