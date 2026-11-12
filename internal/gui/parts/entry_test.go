package parts

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// editorBetween puts an editor between two buttons on a real canvas, focused,
// so that a test can ask where the keyboard goes.
func editorBetween(t *testing.T, text string, size fyne.Size) (*Editor, *Button, *Button, fyne.Canvas) {
	t.Helper()
	before := NewButton("before", func() {})
	editor := NewEditor(nil)
	after := NewButton("after", func() {})
	window := test.NewWindow(container.NewVBox(before, editor, after))
	t.Cleanup(window.Close)
	window.Resize(size)
	editor.SetText(text)

	canvas := window.Canvas()
	canvas.Focus(editor)
	if canvas.Focused() != fyne.Focusable(editor) {
		t.Fatalf("the editor did not take focus, so this test is about nothing")
	}
	return editor, before, after, canvas
}

// pressTab presses Tab the way the driver does: it hands the key to the
// focused control when that control takes Tab, and moves the focus itself
// otherwise. Shift arrives first as a key of its own, as it does from the
// driver, and goes up afterwards to whatever holds the keyboard by then.
func pressTab(t *testing.T, canvas fyne.Canvas, shift bool) {
	t.Helper()
	shiftKey := &fyne.KeyEvent{Name: desktop.KeyShiftLeft}
	if shift {
		if keys, ok := canvas.Focused().(desktop.Keyable); ok {
			keys.KeyDown(shiftKey)
		}
		canvas.Focused().TypedKey(shiftKey)
	}
	switch focused := canvas.Focused().(type) {
	case fyne.Tabbable:
		if focused.AcceptsTab() {
			canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
			break
		}
		moveFocus(canvas, shift)
	default:
		moveFocus(canvas, shift)
	}
	if shift {
		if keys, ok := canvas.Focused().(desktop.Keyable); ok {
			keys.KeyUp(shiftKey)
		}
	}
}

func moveFocus(canvas fyne.Canvas, back bool) {
	if back {
		canvas.FocusPrevious()
		return
	}
	canvas.FocusNext()
}

// placeCaret puts the caret at a place in the text through the editor itself.
func placeCaret(editor *Editor, place int) { editor.caretTo(place) }

// The keyboard can leave this field, which a field that keeps Tab has to answer
// for: Esc, then Tab, goes on to the next control, and Esc, then Shift+Tab, back
// to the one before (DEC-275).
//
// This is the behaviour a person meets rather than the answer to a method: the
// field is focused on a real canvas, the keys are pressed the way the driver
// presses them, and the question is where the focus ended up. Without a way
// out the field would be the trap WCAG 2.1.2 names.
func TestTheKeyboardCanLeaveTheEditor(t *testing.T) {
	editor, before, after, canvas := editorBetween(t, "version: 1\n", fyne.NewSize(400, 300))

	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	pressTab(t, canvas, false)
	if canvas.Focused() != fyne.Focusable(after) {
		t.Errorf("Esc, then Tab, left the keyboard on %T rather than on the control after the field",
			canvas.Focused())
	}

	canvas.Focus(editor)
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	pressTab(t, canvas, true)
	if canvas.Focused() != fyne.Focusable(before) {
		t.Errorf("Esc, then Shift+Tab, left the keyboard on %T rather than on the control before the field",
			canvas.Focused())
	}
	if editor.Text != "version: 1\n" {
		t.Errorf("leaving the field changed what is in it: %q", editor.Text)
	}
}

// Without Esc first, Tab stays in the field and indents, and Shift+Tab takes the
// indent back, so that the key a person writing YAML presses does what they
// press it for.
func TestTabIndentsByTwoSpacesWhereTheCaretIs(t *testing.T) {
	editor, _, _, canvas := editorBetween(t, "entities:\nusers:\n", fyne.NewSize(400, 300))
	placeCaret(editor, len("entities:\n"))

	pressTab(t, canvas, false)
	if canvas.Focused() != fyne.Focusable(editor) {
		t.Fatalf("Tab took the keyboard out of the field to %T", canvas.Focused())
	}
	if want := "entities:\n  users:\n"; editor.Text != want {
		t.Errorf("Tab made %q, want %q", editor.Text, want)
	}
	if strings.ContainsRune(editor.Text, '\t') {
		t.Error("Tab typed a tab character, which no recipe may be indented with")
	}
	if got, want := editor.CursorTextOffset(), len("entities:\n  "); got != want {
		t.Errorf("the caret is at %d after the indent, want %d, right after the two spaces", got, want)
	}
}

// Shift+Tab takes up to two spaces off the start of the line the caret is on,
// wherever on the line the caret is, and keeps the caret beside the same
// character.
func TestShiftTabTakesTheIndentBackFromTheStartOfTheLine(t *testing.T) {
	cases := []struct {
		name, text string
		caret      int
		want       string
		caretAfter int
	}{
		{"four spaces lose two", "a:\n    b: 1\n", len("a:\n    b: "), "a:\n  b: 1\n", len("a:\n  b: ")},
		{"one space loses one", "a:\n b: 1\n", len("a:\n b"), "a:\nb: 1\n", len("a:\nb")},
		{"no indent is left alone", "a:\nb: 1\n", len("a:\nb"), "a:\nb: 1\n", len("a:\nb")},
		{"a caret inside the indent goes to the start", "a:\n  b\n", len("a:\n "), "a:\nb\n", len("a:\n")},
		{"the first line", "  a: 1", len("  a"), "a: 1", len("a")},
		{"the last line with nothing after it", "a:\n  ", len("a:\n  "), "a:\n", len("a:\n")},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			editor, _, _, canvas := editorBetween(t, each.text, fyne.NewSize(400, 300))
			placeCaret(editor, each.caret)
			pressTab(t, canvas, true)
			if canvas.Focused() != fyne.Focusable(editor) {
				t.Fatalf("Shift+Tab took the keyboard out of the field to %T", canvas.Focused())
			}
			if editor.Text != each.want {
				t.Errorf("Shift+Tab made %q, want %q", editor.Text, each.want)
			}
			if got := editor.CursorTextOffset(); got != each.caretAfter {
				t.Errorf("the caret is at %d, want %d", got, each.caretAfter)
			}
		})
	}
}

// A line folded under itself is taken back from where it starts, not from the
// row the caret happens to be on.
//
// 🔴 The field counts the caret in rows as drawn and the indent belongs to the
// line as written, so a long line in a narrow field is where the two part. The
// width here makes the line fold, and the test says so before it asks anything.
func TestShiftTabOnALineFoldedUnderItselfTakesTheIndentFromItsStart(t *testing.T) {
	long := "    description: " + strings.Repeat("several words that fold ", 8)
	editor, _, _, canvas := editorBetween(t, "a:\n"+long, fyne.NewSize(160, 400))
	caret := len("a:\n") + len(long) - 3
	placeCaret(editor, caret)
	if editor.CursorRow < 3 {
		t.Fatalf("the caret is on row %d, so the line did not fold and this case is about nothing",
			editor.CursorRow)
	}

	pressTab(t, canvas, true)
	if want := "a:\n  " + long[4:]; editor.Text != want {
		t.Errorf("Shift+Tab on a folded line made %q, want %q", editor.Text, want)
	}
	if got := editor.CursorTextOffset(); got != caret-2 {
		t.Errorf("the caret is at %d, want %d, beside the same character", got, caret-2)
	}

	// The caret is placed by its two fields, and the toolkit keeps a second
	// copy for drawing a selection. Selecting from here has to start here, or
	// the first Shift+Right after an outdent would select from where the caret
	// used to be.
	shift := &fyne.KeyEvent{Name: desktop.KeyShiftLeft}
	editor.KeyDown(shift)
	editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	editor.KeyUp(shift)
	if got, want := editor.SelectedText(), string([]rune(editor.Text)[caret-2]); got != want {
		t.Errorf("Shift+Right after the outdent selected %q, want %q, the character after the caret", got, want)
	}
}

// Shift+Tab with a selection takes the indent of the line the selection begins
// on, and leaves the selected words where they are.
//
// 🔴 The toolkit's Delete erases a selection, so taking spaces with a selection
// still standing would take the selection with them.
func TestShiftTabWithASelectionLeavesTheSelectedWords(t *testing.T) {
	editor, _, _, canvas := editorBetween(t, "  a: 1\n  b: 2\n", fyne.NewSize(400, 300))
	editor.TypedShortcut(&fyne.ShortcutSelectAll{})
	if editor.SelectedText() == "" {
		t.Fatal("nothing is selected, so this case is about nothing")
	}
	pressTab(t, canvas, true)
	if want := "a: 1\n  b: 2\n"; editor.Text != want {
		t.Errorf("Shift+Tab with everything selected made %q, want %q", editor.Text, want)
	}
}

// An indent and taking it back are undone by Ctrl+Z, the way the same spaces
// typed or deleted by hand are.
//
// Asked because the obvious way to rewrite a line, SetText, empties the history
// Ctrl+Z walks, and a person who indents by mistake expects to undo it. One
// press of Ctrl+Z per space, because the toolkit's history keeps every space as
// a step of its own (widget/entry.go:2072 of v2.8.2 will not merge text holding
// a word separator), which is also what two spaces typed by hand cost.
func TestAnIndentAndItsTakingBackAreUndoneLikeTyping(t *testing.T) {
	editor, _, _, canvas := editorBetween(t, "a:\n  b: 1\n", fyne.NewSize(400, 300))
	placeCaret(editor, len("a:\n  b"))
	undo := func() {
		for range indent {
			editor.TypedShortcut(&fyne.ShortcutUndo{})
		}
	}

	pressTab(t, canvas, true)
	if editor.Text != "a:\nb: 1\n" {
		t.Fatalf("Shift+Tab made %q", editor.Text)
	}
	undo()
	if editor.Text != "a:\n  b: 1\n" {
		t.Errorf("Ctrl+Z after Shift+Tab gave %q rather than the indent back", editor.Text)
	}

	placeCaret(editor, len("a:\n"))
	pressTab(t, canvas, false)
	if editor.Text != "a:\n    b: 1\n" {
		t.Fatalf("Tab made %q", editor.Text)
	}
	undo()
	if editor.Text != "a:\n  b: 1\n" {
		t.Errorf("Ctrl+Z after Tab gave %q rather than the text before it", editor.Text)
	}
}

// Esc arms the way out and anything else the person does in the field disarms
// it, so a Tab long after an Esc indents as Tab always does. A key that only
// modifies another is not something else: Shift comes on its own before
// Shift+Tab.
func TestEscArmsTheWayOutAndAnythingElseDisarmsIt(t *testing.T) {
	disarming := []struct {
		name string
		act  func(*Editor, fyne.Canvas)
	}{
		{"a character", func(e *Editor, _ fyne.Canvas) { e.TypedRune('x') }},
		{"a key", func(e *Editor, _ fyne.Canvas) { e.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft}) }},
		{"a shortcut", func(e *Editor, _ fyne.Canvas) { e.TypedShortcut(&fyne.ShortcutSelectAll{}) }},
		{"a click", func(e *Editor, _ fyne.Canvas) { e.MouseDown(&desktop.MouseEvent{}) }},
		{"the keyboard going away and back", func(e *Editor, c fyne.Canvas) {
			c.Focus(nil)
			c.Focus(e)
		}},
	}
	for _, each := range disarming {
		t.Run(each.name, func(t *testing.T) {
			editor, _, _, canvas := editorBetween(t, "a: 1\n", fyne.NewSize(400, 300))
			editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
			each.act(editor, canvas)
			pressTab(t, canvas, false)
			if canvas.Focused() != fyne.Focusable(editor) {
				t.Errorf("after Esc and %s, Tab left the field rather than indenting", each.name)
			}
		})
	}

	t.Run("Shift on its own", func(t *testing.T) {
		editor, before, _, canvas := editorBetween(t, "a: 1\n", fyne.NewSize(400, 300))
		editor.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		editor.TypedKey(&fyne.KeyEvent{Name: desktop.KeyControlLeft})
		pressTab(t, canvas, true)
		if canvas.Focused() != fyne.Focusable(before) {
			t.Errorf("Esc, a modifier key and Shift+Tab left the keyboard on %T rather than before the field",
				canvas.Focused())
		}
	})
}

// No tab character reaches the recipe by being typed, however the key arrives.
func TestNoTabCharacterIsTypedIntoTheRecipe(t *testing.T) {
	editor, _, _, _ := editorBetween(t, "a: 1", fyne.NewSize(400, 300))
	editor.TypedRune('\t')
	if strings.ContainsRune(editor.Text, '\t') {
		t.Errorf("a tab character arriving as a character was typed: %q", editor.Text)
	}
}

// The driver asks the FOCUSED object, through an interface, whether it takes
// Tab, and that is the road this answer has to survive.
//
// It is separate from the tests above because the two fail for different
// reasons: those are about what Tab does, this one about whether the question
// reaches this type at all. An embedded widget that stopped answering
// fyne.Tabbable would leave the toolkit asking the inner type.
//
// 🔴 What none of them covers, measured rather than assumed on 2026-09-02:
// deleting the ExtendBaseWidget call reddens NOTHING here. That call registers
// the outer type for drawing, and these tests are about keys, so they are the
// wrong instrument for it and say so instead of implying otherwise.
func TestTheEditorTakesTabThroughTheInterfaceTheDriverUses(t *testing.T) {
	editor := NewEditor(nil)
	tabbable, ok := any(editor).(fyne.Tabbable)
	if !ok {
		t.Fatalf("the editor does not answer fyne.Tabbable, so the driver never asks it")
	}
	if !tabbable.AcceptsTab() {
		t.Errorf("the editor refuses Tab, so the driver moves the focus on instead of indenting")
	}
}

// The toolkit's own multi line field types a tab character on Tab, which is
// what makes this field's own answer to the key necessary.
//
// 🔴 It is also an early warning. If a later toolkit stops typing the character,
// this fails, and the reason written beside Editor has to be read again.
func TestTheToolkitsOwnMultiLineFieldTypesATabCharacter(t *testing.T) {
	theirs := widget.NewMultiLineEntry()
	window := test.NewWindow(theirs)
	t.Cleanup(window.Close)
	window.Canvas().Focus(theirs)
	theirs.TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
	if !strings.ContainsRune(theirs.Text, '\t') {
		t.Errorf("the toolkit's own multi line field no longer types a tab character on Tab "+
			"(it holds %q). Editor answers the key itself for that reason, so read it again", theirs.Text)
	}
}

// What the field is for still works, because a field that indents and nothing
// else would be a poor trade.
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
