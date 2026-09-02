package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// The keyboard can leave this field, which is the whole reason it exists.
//
// This is the behaviour a person meets rather than the answer to a method: the
// field is focused on a real canvas, Tab is pressed the way the driver presses
// it, and the question is where the focus ended up. Asking AcceptsTab directly
// would pass even if nothing ever consulted it.
func TestTheKeyboardCanLeaveTheEditor(t *testing.T) {
	editor := NewEditor(nil)
	elsewhere := NewButton("somewhere else", func() {})
	window := test.NewWindow(container.New(newTopFirst(), editor, elsewhere))
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(400, 300))

	canvas := window.Canvas()
	canvas.Focus(editor)
	if canvas.Focused() != fyne.Focusable(editor) {
		t.Fatalf("the editor did not take focus, so this test is about nothing")
	}

	pressTab(t, canvas)

	if canvas.Focused() == fyne.Focusable(editor) {
		t.Errorf("the keyboard is still in the editor after Tab, which is the trap " +
			"WCAG 2.1.2 names. Nothing on this screen can be reached past it")
	}
}

// The toolkit's own multi line field keeps the key, which is what makes the
// test above about our override rather than about a default.
//
// 🔴 It is also an early warning. If a later toolkit gives the Tab key back on
// its own, this fails, and the extension beside it can go rather than sitting
// there forever explaining a problem somebody else fixed.
func TestTheToolkitsOwnMultiLineFieldStillKeepsTheTabKey(t *testing.T) {
	theirs := widget.NewMultiLineEntry()
	tabbable, ok := any(theirs).(fyne.Tabbable)
	if !ok {
		t.Fatalf("the toolkit's multi line field no longer answers the question at all, " +
			"so the extension beside this test is aimed at something that moved")
	}
	if !tabbable.AcceptsTab() {
		t.Errorf("the toolkit's own multi line field now releases Tab. The extension " +
			"beside this test exists only because it did not, so read it again and " +
			"consider deleting it")
	}
}

// The driver asks the FOCUSED object, through an interface, and that is the
// road this answer has to survive.
//
// It is separate from the test above because the two fail for different
// reasons: that one is about where the focus ends up, this one about whether
// the question reaches us at all. An embedded widget that stopped answering
// fyne.Tabbable would leave the toolkit asking the inner type, and a type
// checker has nothing to say about that.
//
// 🔴 What neither of them covers, measured rather than assumed on 2026-09-02:
// deleting the ExtendBaseWidget call reddens NOTHING here. That call registers
// the outer type for drawing, and these tests are about the answer to a
// question, so they are the wrong instrument for it and say so instead of
// implying otherwise.
func TestTheEditorRefusesTabThroughTheInterfaceTheDriverUses(t *testing.T) {
	editor := NewEditor(nil)
	tabbable, ok := any(editor).(fyne.Tabbable)
	if !ok {
		t.Fatalf("the editor does not answer fyne.Tabbable, so the driver never asks it")
	}
	if tabbable.AcceptsTab() {
		t.Errorf("the editor accepts Tab, so the driver will type with it rather than " +
			"moving the focus on")
	}
}

// What the field is for still works, because a field the keyboard can leave and
// nothing else would be a poor trade.
func TestTheEditorStillTakesTextAndReportsIt(t *testing.T) {
	var seen string
	editor := NewEditor(func(text string) { seen = text })
	editor.SetText("version: 1\nentities:\n")

	if editor.Text != "version: 1\nentities:\n" {
		t.Errorf("the editor holds %q rather than what was put in it", editor.Text)
	}
	if seen != "version: 1\nentities:\n" {
		t.Errorf("the editor reported %q rather than what it holds, so nothing "+
			"watching it would validate the right recipe", seen)
	}
	if !editor.MultiLine {
		t.Errorf("the editor is not multi line, and a recipe is written in lines")
	}
}

// pressTab presses Tab the way the driver does when the focused object refuses
// it, which is by moving the focus rather than by delivering the key.
func pressTab(t *testing.T, canvas fyne.Canvas) {
	t.Helper()
	if focused, ok := canvas.Focused().(fyne.Tabbable); ok && focused.AcceptsTab() {
		canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
		return
	}
	canvas.FocusNext()
}
