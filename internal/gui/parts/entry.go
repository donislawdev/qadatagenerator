package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
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
// question and the mark it shows for focus. That is the toolkit's own extension
// mechanism, so the rule against rebuilding pieces of it is untouched and there
// is no renderer here to go stale when the toolkit moves.
//
// # 🔴 Why it wears the same ground and mark as the small fields
//
// On a screen built out of paint, a text field answers the pointer with a
// ground and the keyboard with a ring in mark.focus, and a chooser or button
// beside it does the same (DEC-215, DEC-237). A multi line field is a text
// field too, so it wears the SAME grounds the single line one does, through the
// same helper: the toolkit draws nothing a picture can see for a pointer on an
// entry (B-282) and marks focus with an edge of its own, which would be a
// second language of focus on a screen that has one. So the toolkit's chrome is
// switched off with the same bare theme and this draws the ground itself.
//
// # onFocus, onHover and onDisable
//
// Told when the keyboard and the pointer arrive and leave and when the field
// goes out of reach, so the grounds drawn around the field can answer. They are
// the same three hooks the single line field carries, for the same reason. No
// screen disables the editor today, and it answers anyway: a control is
// expected to have all its states, and the catalogue of DEC-253 shows them.
type Editor struct {
	widget.Entry
	onFocus   func(bool)
	onHover   func(bool)
	onDisable func(bool)
	dressed   fyne.CanvasObject
}

// Asserted so that dropping one of these is a compile error rather than a field
// that silently stops answering the pointer, which is how the chooser once lost
// its hover.
var (
	_ fyne.Widget       = (*Editor)(nil)
	_ fyne.Focusable    = (*Editor)(nil)
	_ desktop.Hoverable = (*Editor)(nil)
)

// NewEditor builds a multi line field the keyboard can leave.
func NewEditor(changed func(string)) *Editor {
	editor := &Editor{}
	editor.MultiLine = true
	// 🔴 Folded at a word, changed on 2026-10-21. Until then a long line was
	// clipped and scrolled to sideways, the toolkit's own default, on the
	// argument that a recipe is written in lines that mean something. The owner
	// typed into it and found a line running off the field under a shadow, with
	// nothing to say where it went: a line nobody can see the end of is worse
	// than one folded under itself. Folding also keeps what clipping was chosen
	// for, a field that does not ask for the width of its longest line, since a
	// folding entry scrolls only up and down, widget/entry.go:1940.
	editor.Wrapping = fyne.TextWrapWord
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

// Framed is the editor dressed in the grounds and the mark every text control
// in this window wears, which is what a screen puts on itself. Built once, for
// the reason the single line field is: the grounds hold the focus, and dressing
// it twice would hand the keyboard to a ground nobody mounted (B-279).
//
// The words are placed by this padding rather than by the toolkit: a step of
// the scale on every side, which is where a field's text sits.
func (e *Editor) Framed() fyne.CanvasObject {
	if e.dressed != nil {
		return e.dressed
	}
	// Never refused: what is wrong with a recipe is said line by line under the
	// editor, and an edge in red would point at the whole document.
	grounds, look := textGrounds(e.Disabled(), false)
	e.onFocus, e.onHover, e.onDisable = look.focus, look.hover, look.disable
	step := palette.StepOf(palette.NameStepTight)
	typed := container.NewThemeOverride(e, bareField{})
	inside := container.New(layout.NewCustomPaddedLayout(step, step, step, step), typed)
	e.dressed = container.NewStack(append(grounds, inside)...)
	return e.dressed
}

// CreateRenderer is the toolkit's drawing of the editor without its edge, the
// same as the field of one line. See withoutEdge.
func (e *Editor) CreateRenderer() fyne.WidgetRenderer {
	return withoutEdge(e.Entry.CreateRenderer())
}

// FocusGained tells the grounds the keyboard is here, then lets the toolkit
// place the caret.
func (e *Editor) FocusGained() {
	if e.onFocus != nil {
		e.onFocus(true)
	}
	e.Entry.FocusGained()
}

// FocusLost tells them it left, then lets the toolkit take the caret.
func (e *Editor) FocusLost() {
	if e.onFocus != nil {
		e.onFocus(false)
	}
	e.Entry.FocusLost()
}

// MouseIn tells the grounds the pointer is on the field, which the toolkit's
// entry does not (B-282): it answers a pointer with the shape of the cursor and
// nothing a picture can see.
func (e *Editor) MouseIn(*desktop.MouseEvent) {
	if e.onHover != nil {
		e.onHover(true)
	}
}

// MouseMoved is a pointer moving over the field, which changes nothing the
// ground shows.
func (e *Editor) MouseMoved(*desktop.MouseEvent) {}

// MouseOut tells them the pointer left.
func (e *Editor) MouseOut() {
	if e.onHover != nil {
		e.onHover(false)
	}
}

// Disable puts the editor out of reach and tells the grounds, which the
// toolkit cannot: see FieldEntry.Disable.
func (e *Editor) Disable() {
	e.Entry.Disable()
	if e.onDisable != nil {
		e.onDisable(true)
	}
}

// Enable gives it back.
func (e *Editor) Enable() {
	e.Entry.Enable()
	if e.onDisable != nil {
		e.onDisable(false)
	}
}
