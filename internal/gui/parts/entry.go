package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Editor is the toolkit's multi line text field that gives the Tab key back to
// the window instead of typing with it.
//
// # What the toolkit does, and why that is not enough here
//
// widget.Entry answers fyne.Tabbable, and its answer is its own MultiLine
// field. Read in the toolkit source at widget/entry.go:152, unchanged between
// v2.8.0 and v2.8.1:
//
//	func (e *Entry) AcceptsTab() bool { return e.MultiLine }
//
// The driver asks the FOCUSED object that question and, when the answer is yes,
// delivers Tab to the control rather than moving the focus on. So a person who
// reaches a multi line field with the keyboard cannot leave it with the
// keyboard: not forward to the buttons, not back to the column of screen names.
// That is a keyboard trap, which WCAG names in 2.1.2 and which this project
// refuses everywhere else.
//
// # Why refusing the key costs almost nothing here
//
// This field holds a recipe, and a recipe is YAML. Measured on 2026-09-02
// against our own reader rather than against the specification, because a rule
// nobody ran is prose: a recipe indented with tab characters is refused at the
// first one, with the reader answering that it found a character that cannot
// start any token. The control file indented with spaces was read and reported
// real problems.
//
// So the key that traps the keyboard is also a key that cannot produce a recipe
// this tool will accept. Refusing it frees the keyboard and removes an illegal
// character in the same move, which is why this is a refusal rather than a
// second shortcut a person has to learn.
//
// The cost is stated rather than discovered. A tab is legal INSIDE a quoted
// scalar, and one can no longer be typed here. YAML writes that one as the
// escape \t inside a double quoted value, so the character is still reachable
// and only the keystroke is gone.
//
// # Why an extension rather than a control of our own
//
// Everything this field does stays the toolkit's: the drawing, the caret, the
// selection, the wrapping, the scrolling. What changes is the answer to one
// question. That is the toolkit's own extension mechanism, so the rule against
// rebuilding pieces of it is untouched and there is no renderer here to go
// stale when the toolkit moves.
type Editor struct {
	widget.Entry
}

// NewEditor builds a multi line field the keyboard can leave.
func NewEditor(changed func(string)) *Editor {
	editor := &Editor{}
	editor.MultiLine = true
	// The same wrapping the toolkit's own multi line constructor uses. A recipe
	// is written in lines that mean something, so a long one is clipped and
	// scrolled to rather than folded. Clipping also keeps the field from asking
	// for the width of its longest line, which is how one control has already
	// pushed four settings off this window.
	editor.Wrapping = fyne.TextWrap(fyne.TextTruncateClip)
	// Monospace, because what goes in here is indented text and indentation is
	// what says which key belongs to which. In a proportional font two lines
	// indented by the same number of spaces do not line up, so the one thing a
	// person has to see in a recipe is the one thing the font hides.
	editor.TextStyle = fyne.TextStyle{Monospace: true}
	editor.OnChanged = changed
	editor.ExtendBaseWidget(editor)
	return editor
}

// AcceptsTab refuses the Tab key, so that the focus moves on instead.
//
// The driver reaches this by asserting fyne.Tabbable on whatever the canvas has
// focused, at internal/driver/glfw/window.go:679. What is focused is this type,
// because this type is what goes into the container, so the assertion finds the
// answer below rather than the embedded one.
func (e *Editor) AcceptsTab() bool { return false }
