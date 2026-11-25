package parts

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Editor is the toolkit's multi line text field, in which Tab indents a recipe
// the way YAML is indented and Esc, then Tab, leaves the field (DEC-275).
//
// # What the toolkit does, and why that is not enough here
//
// widget.Entry answers fyne.Tabbable with its own MultiLine field
// (widget/entry.go:152 of v2.8.2), and the driver then delivers Tab to it
// rather than moving the focus on. Left as it is, the field types a TAB
// CHARACTER, and a recipe indented with tab characters is refused by our
// reader at the first one (measured 2026-09-02: a character that cannot start
// any token). So the key a person writing YAML presses to indent would produce
// a recipe this tool refuses, and the keyboard could not leave the field, which
// is the trap WCAG names in 2.1.2.
//
// # What this does instead
//
// Tab types two spaces where the caret is, which is how a recipe is indented.
// Shift+Tab takes up to two spaces off the start of the line the caret is on.
// No tab character is ever typed, so the one this field produced is gone.
//
// Esc arms the way out: the next Tab moves the focus on and the next Shift+Tab
// moves it back. Anything else a person does in the field disarms it - another
// key, a shortcut, a click - and so does the keyboard going elsewhere. WCAG
// 2.1.2 does not forbid a field that keeps Tab. It forbids one a person cannot
// leave, and allows a way out other than Tab alone when the person is told it,
// which is why the screen says so under the field rather than in a manual.
//
// # Why an extension rather than a control of our own
//
// Everything this field does stays the toolkit's: the drawing, the caret, the
// selection, the wrapping, the scrolling, and the history Ctrl+Z walks back.
// Indenting is typed through the toolkit's own handling of a key, so it can be
// undone like anything else typed, one press of Ctrl+Z per space as for spaces
// typed by hand: the toolkit keeps every space as a step of its own
// (widget/entry.go:2072 of v2.8.2). That is the toolkit's own extension
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

	// leaving is Esc having been pressed, so that the next Tab leaves the
	// field rather than indenting.
	leaving bool
	// shiftDown is whether Shift is held, as this field has seen it, for when
	// the driver cannot say. See shifted.
	shiftDown bool

	// The column of line numbers, the band across the line the caret is on, and
	// what they are drawn from (lines.go, DEC-337). marks exists once the
	// editor is framed. area is the toolkit's scrolling area inside this field,
	// found among the objects of its renderer, and nil while it has not been
	// found. lines is the map of the text, kept until the text changes.
	marks    *lineMarks
	area     *container.Scroll
	lines    lineMap
	problems map[int]bool
	// keyboard is whether this field holds the keyboard, which the toolkit
	// keeps in a field of its own nobody else can read.
	keyboard bool

	// hints is the list under the caret saying what can be written there
	// (suggest.go, DEC-338), and nil for an editor nobody turned it on for.
	hints *suggestions
	// pointerOnText and pointerOnRow are where the pointer is on the field:
	// on the text, or on a row of the list, which takes it from the text.
	pointerOnText, pointerOnRow bool
}

// Asserted so that dropping one of these is a compile error rather than a field
// that silently stops answering the pointer or the keys, which is how the
// chooser once lost its hover. Tabbable is what makes the driver hand Tab to
// this field at all, and Keyable is how it learns that Shift is held.
var (
	_ fyne.Widget       = (*Editor)(nil)
	_ fyne.Focusable    = (*Editor)(nil)
	_ fyne.Tabbable     = (*Editor)(nil)
	_ desktop.Hoverable = (*Editor)(nil)
	_ desktop.Keyable   = (*Editor)(nil)
	_ desktop.Mouseable = (*Editor)(nil)
)

// indent is what one press of Tab types, and Shift+Tab takes away up to as many
// spaces as it holds: the indent every recipe this project writes uses. Text
// rather than a count, because it is two characters of the recipe and not a
// distance on the screen.
const indent = "  "

// NewEditor builds a multi line field in which Tab indents.
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
	// Called by the toolkit after every move of the caret and every change of
	// the text, once it has scrolled the caret into sight, which is also the
	// only report of that scroll (lines.go).
	editor.OnCursorChanged = editor.rowsMoved
	editor.ExtendBaseWidget(editor)
	return editor
}

// AcceptsTab takes the Tab key, so that the driver hands it to TypedKey below
// rather than moving the focus on.
//
// The driver reaches this by asserting fyne.Tabbable on whatever the canvas has
// focused (internal/driver/glfw/window.go:710 of v2.8.2), and Shift+Tab takes
// the same road (:777). What is focused is this type, because this type is what
// goes into the container, so the assertion finds this answer rather than the
// embedded one.
func (e *Editor) AcceptsTab() bool { return true }

// TypedKey indents on Tab, takes the indent back on Shift+Tab, arms the way out
// on Esc, indents the line Enter begins, and hands every other key to the
// toolkit. While the list of suggestions is open, the keys that move in it,
// pick from it and close it are its own (suggestKey).
//
// A key that only modifies another, Shift above all, leaves the way out armed:
// the driver hands this field a press of Shift on its own before Shift+Tab, and
// disarming on it would make Esc, then Shift+Tab, indent instead of leaving.
//
// Tab with Ctrl, Alt or Super held does nothing. The driver delivers it here
// rather than moving the focus, and until 2026-11-12 the toolkit typed a tab
// character with it, which is the one character a recipe may not be indented
// with.
func (e *Editor) TypedKey(key *fyne.KeyEvent) {
	if key == nil || e.Disabled() || e.suggestKey(key) {
		return
	}
	switch {
	case key.Name == fyne.KeyTab:
		e.tab()
	case key.Name == fyne.KeyEscape:
		e.leaving = true
	case modifierKeys[key.Name]:
		e.Entry.TypedKey(key)
	case key.Name == fyne.KeyReturn || key.Name == fyne.KeyEnter:
		// A new line begins indented like the one it ends, and one step
		// deeper after a line that opens a block (indentAfter, DEC-337).
		e.leaving = false
		e.Entry.TypedKey(key)
		e.indentNewLine()
	default:
		e.leaving = false
		e.Entry.TypedKey(key)
		// A key that moved the caret or took a character away changed the
		// word an open list is about.
		e.again()
	}
}

// modifierKeys are the keys that only change what another key means.
var modifierKeys = map[fyne.KeyName]bool{
	desktop.KeyShiftLeft: true, desktop.KeyShiftRight: true,
	desktop.KeyControlLeft: true, desktop.KeyControlRight: true,
	desktop.KeyAltLeft: true, desktop.KeyAltRight: true,
	desktop.KeySuperLeft: true, desktop.KeySuperRight: true,
}

// tab is one press of Tab, with or without Shift.
func (e *Editor) tab() {
	held := e.held()
	if held&^fyne.KeyModifierShift != 0 {
		return
	}
	shifted := held&fyne.KeyModifierShift != 0
	if e.leaving {
		e.leaving = false
		e.leave(shifted)
		return
	}
	if shifted {
		e.outdent()
		return
	}
	for _, space := range indent {
		e.Entry.TypedRune(space)
	}
}

// leave moves the focus to the next control, or to the one before.
func (e *Editor) leave(back bool) {
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	canvas := app.Driver().CanvasForObject(e)
	if canvas == nil {
		return
	}
	if back {
		canvas.FocusPrevious()
		return
	}
	canvas.FocusNext()
}

// held is the modifier keys held now.
//
// Asked of the driver, which is what the toolkit's own field does for Shift+Tab
// (widget/entry.go:858), because a key pressed before the keyboard arrived in
// this field was never seen by it. A driver that cannot say, which is the one
// a test runs on, leaves the answer to what this field saw of Shift.
func (e *Editor) held() fyne.KeyModifier {
	if app := fyne.CurrentApp(); app != nil {
		if keys, ok := app.Driver().(desktop.Driver); ok {
			return keys.CurrentKeyModifiers()
		}
	}
	if e.shiftDown {
		return fyne.KeyModifierShift
	}
	return 0
}

// outdent takes up to two spaces off the start of the line the caret is on,
// and keeps the caret beside the same character.
//
// # Why the caret is moved by hand and the spaces taken by the toolkit's Delete
//
// The toolkit has no call that puts the caret at a place in the text, and
// SetText, which could replace the line, empties the history Ctrl+Z walks.
// So the caret is placed through the two fields the toolkit exports (caretTo),
// and the spaces are taken by pressing Delete through the toolkit's own
// handling of the key, which records them and tells whoever watches the text.
//
// The toolkit is told Shift went up for the length of this, because it still
// believes Shift is down and would select rather than move. It is told Shift
// went down again afterwards, which is what a person pressing it again would
// tell it.
func (e *Editor) outdent() {
	e.Entry.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	defer func() {
		if e.held()&fyne.KeyModifierShift != 0 {
			e.Entry.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
		}
	}()
	if e.SelectedText() != "" {
		// Left with no Shift puts the caret at the start of what was selected
		// and ends the selection, so the line is the one the selection begins
		// on and no Delete below can take the selection with it.
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	text := []rune(e.Text)
	at := min(max(e.CursorTextOffset(), 0), len(text))
	start := at
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	spaces := 0
	for spaces < len(indent) && start+spaces < len(text) && text[start+spaces] == ' ' {
		spaces++
	}
	if spaces == 0 {
		return
	}
	e.caretTo(start)
	for range spaces {
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	}
	e.caretTo(max(start, at-spaces))
}

// caretTo puts the caret at a place in the text, counted in characters.
//
// The toolkit counts the caret in rows as drawn, so a long line folded under
// itself is several rows, and a place in the text is found by walking the rows
// from the top to the last one beginning at or before it.
//
// Setting the two fields is the whole of it. The caret is drawn from them
// (widget/entry.go:234 of v2.8.2), and the second copy the toolkit keeps for
// drawing a selection is brought level by its own handling of every key and
// click before anything reads it.
func (e *Editor) caretTo(place int) {
	row := 0
	for {
		next, there := e.rowStart(row + 1)
		if !there || next > place {
			break
		}
		row++
	}
	begin, _ := e.rowStart(row)
	e.CursorRow, e.CursorColumn = row, place-begin
	e.Refresh()
}

// rowStart is where a row as drawn begins in the text, and false when the field
// has no such row.
//
// Every row after the first begins past the first character, and the toolkit
// answers a row it does not have with the column it was given, which is nought
// here. So nought for a row after the first is the answer for a row that is not
// there (widget/entry.go, textPosFromRowCol, of v2.8.2).
func (e *Editor) rowStart(row int) (int, bool) {
	e.CursorRow, e.CursorColumn = row, 0
	begin := e.CursorTextOffset()
	return begin, row == 0 || begin > 0
}

// TypedRune types a character, and never a tab character: Tab is a key this
// field answers in TypedKey, and a tab cannot indent a recipe. Typing disarms
// the way out, because a person typing is still writing, and it opens the list
// of suggestions or asks it again (typedRune).
//
// The space of Ctrl+Space is not typed. The shortcut opens the list, and on
// Windows the system sends the space after it as well, as a character of its
// own. Ctrl with Alt is AltGr on a Windows keyboard, which types characters, so
// a space with both held is typed.
func (e *Editor) TypedRune(r rune) {
	e.leaving = false
	if r == '\t' {
		return
	}
	if held := e.held(); r == ' ' && held&fyne.KeyModifierControl != 0 && held&fyne.KeyModifierAlt == 0 {
		return
	}
	e.Entry.TypedRune(r)
	e.typedRune(r)
}

// TypedShortcut disarms the way out, for the same reason typing does, opens
// the list of suggestions on Ctrl+Space, and lets the toolkit copy, paste and
// undo. Any other shortcut closes the list, because what it was about may no
// longer be in the text.
func (e *Editor) TypedShortcut(shortcut fyne.Shortcut) {
	e.leaving = false
	if custom, ok := shortcut.(*desktop.CustomShortcut); ok &&
		custom.KeyName == fyne.KeySpace && custom.Modifier == fyne.KeyModifierControl {
		e.SuggestHere()
		return
	}
	e.closeSuggestions()
	e.Entry.TypedShortcut(shortcut)
}

// SetText replaces the text, and closes the list of suggestions, which was
// about the text replaced.
func (e *Editor) SetText(text string) {
	e.closeSuggestions()
	e.Entry.SetText(text)
}

// KeyDown notes Shift for when the driver cannot say, and lets the toolkit
// note it for selecting.
func (e *Editor) KeyDown(key *fyne.KeyEvent) {
	if key != nil && (key.Name == desktop.KeyShiftLeft || key.Name == desktop.KeyShiftRight) {
		e.shiftDown = true
	}
	e.Entry.KeyDown(key)
}

// KeyUp notes that Shift went up.
func (e *Editor) KeyUp(key *fyne.KeyEvent) {
	if key != nil && (key.Name == desktop.KeyShiftLeft || key.Name == desktop.KeyShiftRight) {
		e.shiftDown = false
	}
	e.Entry.KeyUp(key)
}

// MouseDown disarms the way out: a click in the field is a person going on
// writing, not leaving. It closes the list of suggestions, because the caret
// goes wherever the click was.
func (e *Editor) MouseDown(event *desktop.MouseEvent) {
	e.leaving = false
	e.closeSuggestions()
	e.Entry.MouseDown(event)
}

// Framed is the editor dressed in the grounds and the mark every text control
// in this window wears, which is what a screen puts on itself. Built once, for
// the reason the single line field is: the grounds hold the focus, and dressing
// it twice would hand the keyboard to a ground nobody mounted (B-279).
//
// The words are placed by this padding rather than by the toolkit: a step of
// the scale on every side, which is where a field's text sits. No theme of the
// editor's own stands around it, for the reason FieldEntry gives.
//
// Inside the padding the layout leaves a gap at the leading edge for the
// column of line numbers, and what the column and the band across the line of
// the caret draw lies under the gap and the text (lineMarks), so both are
// inside the field's frame and on its ground (DEC-337). Nothing of it takes
// the keyboard, so the keyboard's walk is unchanged.
func (e *Editor) Framed() fyne.CanvasObject {
	if e.dressed != nil {
		return e.dressed
	}
	// Never refused: what is wrong with a recipe is said line by line under the
	// editor, and an edge in red would point at the whole document.
	grounds, look := textGrounds(e.Disabled(), false)
	e.onFocus, e.onHover, e.onDisable = look.focus, look.hover, look.disable
	e.marks = newLineMarks(e)
	lined := container.New(layout.NewBorderLayout(nil, nil, e.marks.room, nil), e.marks.room, e)
	step := palette.StepOf(palette.NameStepTight)
	inside := container.New(layout.NewCustomPaddedLayout(step, step, step, step),
		container.NewStack(e.marks, lined))
	e.dressed = container.NewStack(append(grounds, inside)...)
	return e.dressed
}

// CreateRenderer is the toolkit's drawing of the editor without its edge, the
// same as the field of one line (see withoutEdge), with a pane laid over it so
// that a notch of the wheel moves the lines as far as it moves every other area
// of this window. The toolkit keeps the editor's scrolling area inside this
// renderer, and wheelLayer says how the pane reaches it. The same area is what
// the column of numbers follows (editorRenderer).
func (e *Editor) CreateRenderer() fyne.WidgetRenderer {
	wheel := withWheel(withoutEdge(e.Entry.CreateRenderer()))
	drawing := &editorRenderer{wheeled: wheel, editor: e}
	e.follow(wheel.layer.area)
	return drawing
}

// editorRenderer is the wheeled drawing of the editor that also tells the
// column of numbers when the rows can have moved, and draws the list of
// suggestions on top of everything else (suggest.go).
//
// A change of width folds the text again inside the toolkit's layout of the
// field, and the toolkit says nothing about it (the layout of its content at
// widget/entry.go:1817 of v2.8.2 resizes the text and calls no one). So the
// numbers and the list are put back after every layout, once the toolkit has
// finished it.
type editorRenderer struct {
	*wheeled
	editor *Editor
	// objects is the wheeled list with the holder of the list of suggestions
	// after it, kept so that a frame asking several times allocates nothing.
	objects []fyne.CanvasObject
}

// Layout lets the toolkit and the pane lay out, then puts the numbers and the
// list back.
func (r *editorRenderer) Layout(size fyne.Size) {
	r.wheeled.Layout(size)
	if hints := r.editor.hints; hints != nil {
		hints.holder.Resize(size)
	}
	r.editor.follow(r.layer.area)
	r.editor.rowsMoved()
}

// Objects is the wheeled list with the holder of the list of suggestions last,
// so that the list is drawn over the text and found under the pointer before
// the pane of the wheel. It follows the area when the toolkit's renderer handed
// over another list and the pane found another area in it.
func (r *editorRenderer) Objects() []fyne.CanvasObject {
	objects := r.wheeled.Objects()
	r.editor.follow(r.layer.area)
	hints := r.editor.hints
	if hints == nil {
		return objects
	}
	kept := len(r.objects) == len(objects)+1 && slices.Equal(r.objects[:len(objects)], objects) &&
		r.objects[len(objects)] == fyne.CanvasObject(hints.holder)
	if !kept {
		r.objects = append(append(make([]fyne.CanvasObject, 0, len(objects)+1), objects...), hints.holder)
	}
	return r.objects
}

// follow keeps the area the column follows, and asks it to say when the wheel
// or its bar moved it. The toolkit's entry sets nothing on that field of its
// area (no OnScrolled in widget/entry.go of v2.8.2), so taking it takes nothing
// from the toolkit.
func (e *Editor) follow(area *container.Scroll) {
	if area == e.area {
		return
	}
	e.area = area
	if area != nil {
		area.OnScrolled = func(fyne.Position) { e.rowsMoved() }
	}
}

// rowsMoved puts back everything drawn where the rows of the text are: the
// numbers, the marks and the band (lines.go), and the list of suggestions.
func (e *Editor) rowsMoved() {
	e.redrawLines()
	e.placeSuggestions()
}

// FocusGained tells the grounds the keyboard is here, then lets the toolkit
// place the caret. The way out starts disarmed, whatever happened on the last
// visit. The number of the caret's line and the band across it follow.
func (e *Editor) FocusGained() {
	e.leaving = false
	e.keyboard = true
	if e.onFocus != nil {
		e.onFocus(true)
	}
	e.Entry.FocusGained()
	e.redrawLines()
}

// FocusLost tells them it left, then lets the toolkit take the caret. The way
// out is disarmed, and Shift is forgotten, because its release will go to
// whatever holds the keyboard next. The band and the list of suggestions go
// with the caret.
func (e *Editor) FocusLost() {
	e.leaving, e.shiftDown = false, false
	e.keyboard = false
	e.closeSuggestions()
	if e.onFocus != nil {
		e.onFocus(false)
	}
	e.Entry.FocusLost()
	e.redrawLines()
}

// MouseIn tells the grounds the pointer is on the field, which the toolkit's
// entry does not (B-282): it answers a pointer with the shape of the cursor and
// nothing a picture can see.
func (e *Editor) MouseIn(*desktop.MouseEvent) {
	e.pointerOnText = true
	e.showPointer()
}

// pointerOnList is the pointer arriving on a row of the list of suggestions
// and leaving it. The row takes the pointer from the field it stands in, and
// the field would go dark under a pointer that never left it (DEC-338), the
// way it would under the column of numbers without lineMarks passing it on.
func (e *Editor) pointerOnList(here bool) {
	e.pointerOnRow = here
	e.showPointer()
}

// showPointer tells the grounds whether the pointer is anywhere on the field:
// on its text or on a row of its list.
func (e *Editor) showPointer() {
	if e.onHover != nil {
		e.onHover(e.pointerOnText || e.pointerOnRow)
	}
}

// MouseMoved is a pointer moving over the field, which changes nothing the
// ground shows.
func (e *Editor) MouseMoved(*desktop.MouseEvent) {}

// MouseOut tells them the pointer left, unless it went onto the list.
func (e *Editor) MouseOut() {
	e.pointerOnText = false
	e.showPointer()
}

// Disable puts the editor out of reach and tells the grounds, which the
// toolkit cannot: see FieldEntry.Disable.
func (e *Editor) Disable() {
	e.closeSuggestions()
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
