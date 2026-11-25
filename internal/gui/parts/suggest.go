package parts

import (
	"slices"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// The list under the caret of an editor saying what can be written there, the
// way a code editor suggests while somebody types (DEC-338).
//
// # Why the list stands inside the field rather than over the window
//
// A key goes to whatever the top overlay has focused and to nothing under it
// (picker, above), so a list in an overlay would take the keyboard from the
// field the moment it opened, and a person typing a word would type into the
// list. The list is drawn by the editor's own renderer, after the text and the
// pane of the wheel, so it lies on top of both and the keyboard never moves.
// A click on a row keeps the keyboard in the field as well: the driver takes
// the keyboard from a field only when the field is not under the pointer
// (processMouseClicked in internal/driver/glfw/window.go of v2.8.2), and the
// list lies inside it.
//
// # What it is made of
//
// The panel of a chooser with no search field (picker): the same rows, the same
// row the keyboard is on, the same scrolling and the same placing under or
// over what it opened from (place). What is different is who holds the
// keyboard. Here the field keeps it and hands the panel the keys that move,
// pick and close while the panel is open.
//
// # What the editor knows and what it is told
//
// The editor knows nothing about recipes. Whoever builds it hands it a
// question, asked again whenever the text or the caret moves while the list is
// open: what can be written at this caret, and which characters a chosen offer
// replaces. The answer is already filtered by what was typed.

// Completion is what can be written at the caret, and the characters a chosen
// offer replaces.
type Completion struct {
	// From and To are where the replaced word begins and ends, counted in
	// characters of the text.
	From, To int
	Offers   []Offer
}

// Offer is one row of the list and what choosing it writes.
type Offer struct {
	// Choice is what the row shows: the word, and what it means beside it.
	Choice
	// Writes replaces the word at the caret, or the Value when it is empty.
	Writes string
	// More asks again once the offer is written, for a key whose values are
	// known, so that the list of them follows at once.
	More bool
}

// writes is what choosing the offer writes.
func (o Offer) writes() string {
	if o.Writes == "" {
		return o.Value
	}
	return o.Writes
}

// suggestions is the state of the list of one editor.
type suggestions struct {
	ask func(text string, caret int) Completion
	// shown is the answer the open list holds, and panel the list, nil while
	// it is closed.
	shown Completion
	panel *picker
	// holder is what the editor's renderer draws on top of the text. It holds
	// the panel while the list is open and nothing otherwise.
	holder *fyne.Container
	// writing is an offer being written, during which the text and the caret
	// move without anybody typing.
	writing bool
}

// Suggest turns the list on, with the question it asks about the text.
func (e *Editor) Suggest(ask func(text string, caret int) Completion) {
	e.hints = &suggestions{ask: ask, holder: container.NewWithoutLayout()}
}

// SuggestHere opens the list at the caret with everything that can be written
// there, which is what Ctrl+Space does. Exported so that a screen built for a
// picture and a test with no keyboard can open it.
func (e *Editor) SuggestHere() {
	if e.hints == nil || e.Disabled() {
		return
	}
	e.show(e.hints.ask(e.Text, e.CursorTextOffset()), false)
}

// Suggested is the words the open list shows, from the top, and nothing while
// it is closed.
func (e *Editor) Suggested() []string {
	if e.hints == nil || e.hints.panel == nil {
		return nil
	}
	words := make([]string, 0, len(e.hints.shown.Offers))
	for _, offer := range e.hints.shown.Offers {
		words = append(words, offer.Value)
	}
	return words
}

// suggesting says whether the list is open.
func (e *Editor) suggesting() bool { return e.hints != nil && e.hints.panel != nil }

// show opens the list with an answer, or closes it when the answer offers
// nothing. A list opened by typing also closes when its one offer is the word
// already written, so that Enter after a finished word begins a new line
// rather than writing the word again.
func (e *Editor) show(asked Completion, typed bool) {
	hints := e.hints
	if len(asked.Offers) == 0 || (typed && e.writtenAlready(asked)) {
		e.closeSuggestions()
		return
	}
	choices := make([]Choice, 0, len(asked.Offers))
	for _, offer := range asked.Offers {
		choices = append(choices, offer.Choice)
	}
	var panel *picker
	panel = newSuggestionPanel(choices, e.take, func() {
		if hints.panel == panel {
			e.closeSuggestions()
		}
	}, e.pointerOnList)
	hints.shown, hints.panel = asked, panel
	hints.holder.Objects = []fyne.CanvasObject{panel}
	e.placeSuggestions()
	hints.holder.Refresh()
}

// writtenAlready says whether the one offer of an answer is what the text
// already says in its place.
func (e *Editor) writtenAlready(asked Completion) bool {
	if len(asked.Offers) != 1 {
		return false
	}
	text := []rune(e.Text)
	from, to := min(max(asked.From, 0), len(text)), min(max(asked.To, 0), len(text))
	return from <= to && string(text[from:to]) == asked.Offers[0].writes()
}

// again asks once more after the text or the caret moved under an open list.
func (e *Editor) again() {
	if e.suggesting() && !e.hints.writing {
		e.show(e.hints.ask(e.Text, e.CursorTextOffset()), true)
	}
}

// closeSuggestions takes the list down.
func (e *Editor) closeSuggestions() {
	if !e.suggesting() {
		return
	}
	e.hints.panel, e.hints.shown = nil, Completion{}
	e.hints.holder.RemoveAll()
}

// take writes the offer chosen from the open list in place of the word it
// replaces, and asks again where the offer says more follows.
//
// # Why the word is taken away key by key and the offer pasted
//
// SetText would write it in one go and empty the history Ctrl+Z walks. So the
// caret is put at the start of the word, the word is taken by the toolkit's
// own Delete, and the offer goes in through the toolkit's own paste, from a
// clipboard that holds only the offer and is never the system's. All three are
// recorded the way typing is, so Ctrl+Z takes the offer back, and what a
// person copied before is still there to paste.
func (e *Editor) take(value string) {
	hints := e.hints
	at := slices.IndexFunc(hints.shown.Offers, func(offer Offer) bool { return offer.Value == value })
	if at < 0 {
		return
	}
	offer, from, to := hints.shown.Offers[at], hints.shown.From, hints.shown.To
	e.closeSuggestions()
	e.replace(from, to, offer.writes())
	if offer.More {
		e.SuggestHere()
	}
}

// replace puts text in place of the characters from one place to another.
func (e *Editor) replace(from, to int, text string) {
	e.hints.writing = true
	defer func() { e.hints.writing = false }()
	if e.SelectedText() != "" {
		// Left with no Shift ends the selection, as in outdent.
		e.Entry.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	length := len([]rune(e.Text))
	from, to = min(max(from, 0), length), min(max(to, 0), length)
	e.caretTo(from)
	for range max(to-from, 0) {
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDelete})
	}
	if text != "" {
		e.Entry.TypedShortcut(&fyne.ShortcutPaste{Clipboard: offered(text)})
	}
}

// offered is a clipboard holding one offer, so that the toolkit's paste writes
// it without the system's clipboard being touched.
type offered string

func (o offered) Content() string   { return string(o) }
func (o offered) SetContent(string) {}

// suggestKey is what a key does while the list is open, and false for a key
// the field answers as it always does.
//
// Down and Up move, Enter and Tab write the offer the keyboard is on, and Esc
// closes the list without arming the way out, so that the next Esc does that
// (DEC-275). Shift+Tab closes the list and goes on to take an indent back.
func (e *Editor) suggestKey(key *fyne.KeyEvent) bool {
	if !e.suggesting() {
		return false
	}
	panel := e.hints.panel
	switch key.Name {
	case fyne.KeyDown:
		panel.move(1)
	case fyne.KeyUp:
		panel.move(-1)
	case fyne.KeyReturn, fyne.KeyEnter:
		panel.pick(panel.current)
	case fyne.KeyEscape:
		e.closeSuggestions()
	case fyne.KeyTab:
		if e.held()&fyne.KeyModifierShift != 0 {
			e.closeSuggestions()
			return false
		}
		panel.pick(panel.current)
	default:
		return false
	}
	return true
}

// opensByTyping says whether typing that character opens the list by itself:
// a letter, a digit or an underscore that begins or goes on with a word, and
// a space, a bracket or a comma after which a value or a list item begins.
func opensByTyping(typed, before rune) bool {
	switch {
	case unicode.IsLetter(typed) || unicode.IsDigit(typed) || typed == '_':
		return true
	case typed == '[' || typed == '{' || typed == ',':
		return true
	case typed == ' ':
		return before == ':' || before == ',' || before == '[' || before == '{' || before == '-'
	}
	return false
}

// typedRune opens the list or asks it again after a character was typed.
func (e *Editor) typedRune(r rune) {
	if e.hints == nil {
		return
	}
	if e.suggesting() {
		e.again()
		return
	}
	text := []rune(e.Text)
	caret := min(max(e.CursorTextOffset(), 0), len(text))
	// The character before the one just typed, which stands before the caret.
	before, typedAt := ' ', caret-1
	if typedAt >= 1 {
		before = text[typedAt-1]
	}
	if opensByTyping(r, before) {
		e.show(e.hints.ask(e.Text, caret), true)
	}
}

// placeSuggestions puts the open list under the word it replaces, or over it
// where there is more room above, inside the field, with the names in it
// standing under the word, and closes it when the caret has scrolled out of
// sight.
func (e *Editor) placeSuggestions() {
	if !e.suggesting() {
		return
	}
	word, row, inSight := e.wordAnchor(e.hints.shown.From)
	if !inSight {
		e.closeSuggestions()
		return
	}
	panel := e.hints.panel
	anchor := fyne.NewPos(max(word.X-panel.nameInset(), 0), word.Y)
	wanted := fyne.NewSize(panel.Width(0), panel.ContentHeight())
	position, size := place(e.Size(), anchor, fyne.NewSize(0, row), wanted, panel.MinSize().Height)
	if fitted := panel.Fitted(size.Height); fitted < size.Height {
		// A list over the word keeps its foot against the word.
		if position.Y < anchor.Y {
			position.Y += size.Height - fitted
		}
		size.Height = fitted
	}
	panel.fitDetails(size.Width)
	panel.Resize(size)
	panel.Move(position)
	panel.reveal()
}

// wordAnchor is where the word beginning at a place stands in the field: its
// left edge and the top of the caret's row, the height of a row, and whether
// that row is in sight.
//
// The caret is asked of the toolkit once, which costs a walk through the text
// (rows in lines.go), and the word's start is found from it by measuring what
// was typed of the word, since the two stand on the same row.
func (e *Editor) wordAnchor(from int) (fyne.Position, float32, bool) {
	area := e.area
	if area == nil || area.Content == nil {
		return fyne.Position{}, 0, false
	}
	caret, height := e.CursorPosition(), e.rowHeight()
	top := area.Position().Y + caret.Y - area.Offset.Y
	inSight := area.Position().Y
	if top+height <= inSight || top >= inSight+area.Size().Height {
		return fyne.Position{}, 0, false
	}
	text := []rune(e.Text)
	at := min(max(e.CursorTextOffset(), 0), len(text))
	from = min(max(from, 0), at)
	typed := fyne.MeasureText(string(text[from:at]), e.Theme().Size(theme.SizeNameText), e.TextStyle)
	left := area.Position().X + caret.X - area.Offset.X - typed.Width
	return fyne.NewPos(max(left, 0), top), height, true
}
