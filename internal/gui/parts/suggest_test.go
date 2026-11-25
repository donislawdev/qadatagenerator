package parts

import (
	"slices"
	"strings"
	"testing"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

// toyAsk is the question a screen hands the editor, written as small as it
// goes: three keys at a word being typed, and two values after the one key
// that has them. It reads no format, so what these tests hold is the list and
// the keys, not what a recipe offers.
func toyAsk(text string, caret int) Completion {
	runes := []rune(text)
	word := func(r rune) bool { return unicode.IsLetter(r) || r == '_' }
	from := caret
	for from > 0 && word(runes[from-1]) {
		from--
	}
	to := caret
	for to < len(runes) && word(runes[to]) {
		to++
	}
	// A word begins a line or follows a space, or it is not a place for one.
	if head := string(runes[:from]); head != "" && !strings.HasSuffix(head, " ") && !strings.HasSuffix(head, "\n") {
		return Completion{From: from, To: to}
	}
	all := []Offer{
		{Choice: Choice{Value: "generator", Detail: "What fills the column"}, Writes: "generator: ", More: true},
		{Choice: Choice{Value: "gender"}},
		{Choice: Choice{Value: "count"}, Writes: "count: "},
	}
	if strings.HasSuffix(string(runes[:from]), "generator: ") {
		all = []Offer{{Choice: Choice{Value: "pesel"}}, {Choice: Choice{Value: "first_name"}}}
	}
	typed := string(runes[from:caret])
	var offers []Offer
	for _, offer := range all {
		if strings.HasPrefix(offer.Value, typed) {
			offers = append(offers, offer)
		}
	}
	return Completion{From: from, To: to, Offers: offers}
}

// suggestingEditor is a framed editor with the toy question, holding the
// keyboard, with the caret at the end of its text.
func suggestingEditor(t *testing.T, text string, size fyne.Size) (*Editor, fyne.Canvas) {
	t.Helper()
	editor, window := framedEditor(t, text, size)
	editor.Suggest(toyAsk)
	canvas := window.Canvas()
	canvas.Focus(editor)
	if canvas.Focused() != fyne.Focusable(editor) {
		t.Fatal("the editor did not take the keyboard, so this test is about nothing")
	}
	placeCaret(editor, len([]rune(text)))
	return editor, canvas
}

func typeInto(editor *Editor, text string) {
	for _, r := range text {
		editor.TypedRune(r)
	}
}

func pressIn(editor *Editor, name fyne.KeyName) { editor.TypedKey(&fyne.KeyEvent{Name: name}) }

// Typing a word opens the list with what begins with it, every further letter
// narrows it, and a letter that leaves nothing closes it.
func TestTypingAWordOpensTheListAndEveryLetterNarrowsIt(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	if editor.Suggested() != nil {
		t.Fatalf("the list was open before anything was typed: %q", editor.Suggested())
	}
	for _, step := range []struct {
		typed string
		want  []string
	}{
		{"g", []string{"generator", "gender"}},
		{"e", []string{"generator", "gender"}},
		{"ne", []string{"generator"}},
		{"x", nil},
	} {
		typeInto(editor, step.typed)
		if got := editor.Suggested(); !slices.Equal(got, step.want) {
			t.Errorf("after %q the list showed %q, want %q", editor.Text, got, step.want)
		}
	}
}

// A space opens the list only where a value begins after it, which is after a
// colon, a comma or a bracket. A space of indentation opens nothing, or every
// line begun would bring a list up.
func TestASpaceOpensTheListOnlyWhereAValueBegins(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	typeInto(editor, "    ")
	if got := editor.Suggested(); got != nil {
		t.Errorf("spaces of indentation opened the list with %q", got)
	}
	editor.SetText("generator:")
	placeCaret(editor, len([]rune(editor.Text)))
	typeInto(editor, " ")
	if got := editor.Suggested(); !slices.Equal(got, []string{"pesel", "first_name"}) {
		t.Errorf("a space after the colon opened %q, want the two values", got)
	}
}

// Enter writes the offer the keyboard is on in place of the word, and a key
// whose values are known opens the list of them at once. Ctrl+Z takes the
// offer back, and what was on the system's clipboard is still there.
func TestEnterWritesTheOfferAndTheValuesFollow(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	clipboard := fyne.CurrentApp().Clipboard()
	clipboard.SetContent("copied before")

	typeInto(editor, "gen")
	pressIn(editor, fyne.KeyReturn)
	if editor.Text != "generator: " {
		t.Fatalf("Enter wrote %q, want %q", editor.Text, "generator: ")
	}
	if got := editor.Suggested(); !slices.Equal(got, []string{"pesel", "first_name"}) {
		t.Fatalf("after the key the list showed %q, want its two values", got)
	}
	pressIn(editor, fyne.KeyReturn)
	if editor.Text != "generator: pesel" || editor.Suggested() != nil {
		t.Fatalf("the value left %q with the list showing %q", editor.Text, editor.Suggested())
	}
	if editor.CursorTextOffset() != len([]rune(editor.Text)) {
		t.Errorf("the caret stands at %d rather than after what was written", editor.CursorTextOffset())
	}
	if got := clipboard.Content(); got != "copied before" {
		t.Errorf("writing an offer left %q on the system's clipboard", got)
	}
	editor.TypedShortcut(&fyne.ShortcutUndo{})
	if editor.Text != "generator: " {
		t.Errorf("Ctrl+Z after the value left %q, want the key back on its own", editor.Text)
	}
}

// The arrows move the keyboard through the list, and Tab writes as Enter does.
func TestTheArrowsChooseAndTabWrites(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	typeInto(editor, "ge")
	pressIn(editor, fyne.KeyDown)
	pressIn(editor, fyne.KeyDown)
	pressIn(editor, fyne.KeyUp)
	pressIn(editor, fyne.KeyDown)
	pressIn(editor, fyne.KeyTab)
	if editor.Text != "gender" {
		t.Errorf("Down, Down, Up, Down and Tab wrote %q, want the second offer", editor.Text)
	}
}

// Esc closes the list and only the next Esc arms the way out of the field, so
// that the key closing a list never also sends the keyboard away (DEC-275).
func TestEscClosesTheListBeforeItArmsTheWayOut(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	typeInto(editor, "co")
	pressIn(editor, fyne.KeyEscape)
	if editor.Suggested() != nil || editor.leaving {
		t.Fatalf("the first Esc left the list %q and the way out armed %v", editor.Suggested(), editor.leaving)
	}
	pressIn(editor, fyne.KeyEscape)
	if !editor.leaving {
		t.Error("the second Esc did not arm the way out")
	}
}

// A word typed to its end does not keep a list offering that same word, so
// that Enter after it begins a new line.
func TestAFinishedWordDoesNotHoldEnterBack(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	typeInto(editor, "gender")
	if got := editor.Suggested(); got != nil {
		t.Fatalf("the finished word kept the list open with %q", got)
	}
	pressIn(editor, fyne.KeyReturn)
	if editor.Text != "gender\n" {
		t.Errorf("Enter after the finished word gave %q, want a new line", editor.Text)
	}
}

// Ctrl+Space opens the list with everything that can be written, even where
// nothing has been typed.
func TestCtrlSpaceOpensTheList(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	editor.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeySpace, Modifier: fyne.KeyModifierControl})
	if got := editor.Suggested(); !slices.Equal(got, []string{"generator", "gender", "count"}) {
		t.Errorf("Ctrl+Space opened %q, want every offer", got)
	}
	if editor.Text != "" {
		t.Errorf("Ctrl+Space typed %q", editor.Text)
	}
}

// The list closes whenever what it was about goes away: a click puts the caret
// elsewhere, the keyboard leaves, the text is replaced, another shortcut runs.
func TestTheListClosesWhenWhatItWasAboutGoesAway(t *testing.T) {
	editor, canvas := suggestingEditor(t, "", fyne.NewSize(400, 300))
	for _, one := range []struct {
		name string
		do   func()
	}{
		{"a click in the field", func() { editor.MouseDown(&desktop.MouseEvent{}) }},
		{"the keyboard leaving", func() { canvas.Unfocus() }},
		{"the text replaced", func() { editor.SetText("") }},
		{"another shortcut", func() { editor.TypedShortcut(&fyne.ShortcutSelectAll{}) }},
		{"the field going out of reach", func() { editor.Disable() }},
	} {
		editor.Enable()
		canvas.Focus(editor)
		editor.SetText("")
		typeInto(editor, "g")
		if editor.Suggested() == nil {
			t.Fatalf("%s: the list did not open, so closing it proves nothing", one.name)
		}
		one.do()
		if got := editor.Suggested(); got != nil {
			t.Errorf("%s left the list open with %q", one.name, got)
		}
	}
}

// A click on a row writes that row's offer.
//
// The row is tapped by hand rather than through test.Tap, which takes the
// keyboard from whatever holds it before every tap on a control that does not
// take it. The driver of a real window does not, when the control holding the
// keyboard lies under the pointer (processMouseClicked in
// internal/driver/glfw/window.go of v2.8.2), and the list lies inside the
// field. That half is seen in the live window rather than here.
func TestAClickOnARowWritesIt(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	typeInto(editor, "ge")
	editor.hints.panel.rows[1].tap.Tapped(&fyne.PointEvent{})
	if editor.Text != "gender" || editor.Suggested() != nil {
		t.Errorf("a click on the second row left %q and the list %q", editor.Text, editor.Suggested())
	}
}

// The pointer moving from the text onto a row of the list stays on the field,
// so the field keeps the ground it wears under the pointer. The driver tells
// the field the pointer left and then tells the row it arrived, in that order,
// and the field went dark in the live window on 2026-11-24 until the row said
// so to the field (DEC-338).
func TestThePointerOnTheListStaysOnTheField(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	lit := false
	editor.onHover = func(here bool) { lit = here }
	typeInto(editor, "ge")
	row := editor.hints.panel.rows[1].tap
	editor.MouseIn(&desktop.MouseEvent{})
	editor.MouseOut()
	row.MouseIn(&desktop.MouseEvent{})
	if !lit {
		t.Error("the pointer went from the text onto the list and the field went dark")
	}
	row.MouseOut()
	if lit {
		t.Error("the pointer left the list and the field stayed lit")
	}
}

// The list stands under the word it replaces, inside the field, and over the
// word when the room below would cut it short. The names in it begin where the
// word begins rather than where the caret is, the way a code editor lines them
// up.
func TestTheListStandsUnderTheWordInsideTheField(t *testing.T) {
	size := fyne.NewSize(400, 300)
	editor, _ := suggestingEditor(t, "count: 1\n      ", size)
	typeInto(editor, "gen")
	anchor, row, inSight := editor.wordAnchor(editor.hints.shown.From)
	if !inSight {
		t.Fatal("the caret's row is not in sight in a field this size")
	}
	panel := editor.hints.panel
	if top := panel.Position().Y; top < anchor.Y+row {
		t.Errorf("the list begins at %v, over the row of the word, which ends at %v", top, anchor.Y+row)
	}
	if names := panel.Position().X + panel.nameInset(); names != anchor.X {
		t.Errorf("the names in the list begin at %v rather than where the word does, %v", names, anchor.X)
	}
	caretX := editor.area.Position().X + editor.CursorPosition().X
	if anchor.X >= caretX {
		t.Errorf("the word begins at %v, not left of the caret at %v", anchor.X, caretX)
	}
	inside(t, editor, panel)

	low, _ := suggestingEditor(t, strings.Repeat("line\n", 9), fyne.NewSize(400, 220))
	typeInto(low, "ge")
	anchor, _, inSight = low.wordAnchor(low.hints.shown.From)
	if !inSight {
		t.Fatal("the last row is not in sight, so where the list stands proves nothing")
	}
	if bottom := low.hints.panel.Position().Y + low.hints.panel.Size().Height; bottom > anchor.Y {
		t.Errorf("with no room below, the list ends at %v, under the top of the word's row at %v", bottom, anchor.Y)
	}
	inside(t, low, low.hints.panel)
}

// A meaning too long for the field is cut at the panel's edge with an
// ellipsis, and the name beside it, which is what gets written, stays whole.
// A field wide enough gives the meaning back in full.
func TestAMeaningTooLongForTheFieldIsCutAndTheNameIsNot(t *testing.T) {
	long := "The parent table and how many rows each parent gets, and then some more words"
	ask := func(text string, caret int) Completion {
		return Completion{From: caret, To: caret, Offers: []Offer{
			{Choice: Choice{Value: "belongs_to", Detail: long}},
			{Choice: Choice{Value: "count", Detail: "How many rows"}},
		}}
	}
	for _, one := range []struct {
		width float32
		cut   bool
	}{{420, true}, {1400, false}} {
		editor, window := framedEditor(t, "", fyne.NewSize(one.width, 300))
		editor.Suggest(ask)
		window.Canvas().Focus(editor)
		editor.SuggestHere()
		panel := editor.hints.panel
		shown := panel.rows[0].choice
		if shown.Value != "belongs_to" || panel.rows[1].choice.Detail != "How many rows" {
			t.Errorf("at %v the name or a short meaning was cut: %+v", one.width, panel.rows)
		}
		if cut := shown.Detail != long; cut != one.cut {
			t.Errorf("at %v the meaning reads %q, cut %v, want cut %v", one.width, shown.Detail, cut, one.cut)
		}
		if one.cut && !strings.HasSuffix(shown.Detail, "…") {
			t.Errorf("the cut meaning %q does not end on an ellipsis", shown.Detail)
		}
		inside(t, editor, panel)
		if row := panel.rows[0].tap; row.MinSize().Width > panel.scroll.Size().Width {
			t.Errorf("at %v the row asks for %v and the list is %v wide, so its end runs under the edge",
				one.width, row.MinSize().Width, panel.scroll.Size().Width)
		}
	}
}

// inside holds that the panel lies within the editor, which is what keeps a
// click on it from taking the keyboard away.
func inside(t *testing.T, editor *Editor, panel *picker) {
	t.Helper()
	at, size, room := panel.Position(), panel.Size(), editor.Size()
	if at.X < 0 || at.Y < 0 || at.X+size.Width > room.Width || at.Y+size.Height > room.Height {
		t.Errorf("the list at %v of %v runs outside the field of %v", at, size, room)
	}
}

// A list whose word scrolls out of sight closes rather than hanging where the
// word was.
func TestTheListClosesWhenItsWordScrollsAway(t *testing.T) {
	editor, _ := suggestingEditor(t, strings.Repeat("line\n", 40), fyne.NewSize(400, 200))
	placeCaret(editor, 0)
	typeInto(editor, "g")
	if editor.Suggested() == nil {
		t.Fatal("the list did not open at the top, so scrolling proves nothing")
	}
	editor.area.ScrollToOffset(fyne.NewPos(0, editor.area.Content.Size().Height))
	editor.area.OnScrolled(editor.area.Offset)
	if got := editor.Suggested(); got != nil {
		t.Errorf("the word scrolled away and the list still shows %q", got)
	}
}

// The list is what the editor's renderer draws last, so it lies over the text
// and is found under the pointer before the pane of the wheel.
func TestTheListIsDrawnOverTheText(t *testing.T) {
	editor, _ := suggestingEditor(t, "", fyne.NewSize(400, 300))
	typeInto(editor, "g")
	objects := test.WidgetRenderer(editor).Objects()
	if last := objects[len(objects)-1]; last != fyne.CanvasObject(editor.hints.holder) {
		t.Errorf("the renderer draws %T last rather than the list", last)
	}
	if !slices.Contains(editor.hints.holder.Objects, fyne.CanvasObject(editor.hints.panel)) {
		t.Error("the holder drawn last does not hold the open list")
	}
}
