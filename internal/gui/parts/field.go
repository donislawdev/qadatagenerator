package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// FieldEntry is the toolkit's text field, wearing this window's look.
//
// # What was wrong, measured rather than felt
//
// The owner said on 2026-09-09 that the how many field "looks odd somehow", and
// then, of the focused one, that the line was too thick and that the chooser
// beside it lit up differently. Read off the stored tree of the quick screen,
// where a chooser and a field stand in ONE row of ONE form, both 34 tall:
//
//	chooser  the word ssn at 12, 8, focus shown as a ring inside its edge
//	field    the word 10  at  4, 4, focus shown as the edge itself
//
// Nobody chose any of that. Both positions come from the renderer of
// widget.Entry, which puts text at SizeNameInnerPadding from the left and at
// twice SizeNameInputBorder from the top, and goes on doing so when the field
// is stretched taller, so the spare height all falls below the words. At the
// field's own height those same numbers happen to centre the text, which is why
// none of it showed until DEC-232 gave every control one height.
//
// # 🔴 The rule this exists to keep
//
// The owner stated it in one sentence: "one control looks like x, the next one
// y, the third z - do not do that, the GUI has to be consistent, it is one
// program". So this does not merely repair the field. It makes the field say
// focus with the SAME MARK as everything else: its own edge, redrawn in
// mark.focus. Two controls in one row no longer disagree about what focus looks
// like, which is DEC-215 applied to a second dimension.
//
// # How, without taking the control over
//
// B-262 settled that widget.Entry cannot be rebuilt: it keeps its chrome in the
// renderer and its behaviour in the widget, the opposite way round from
// widget.Select, so taking its renderer means writing the caret, the selection,
// the scrolling and the shortcuts again.
//
// So the toolkit's chrome is switched OFF rather than replaced, through a theme
// handed to this subtree alone:
//
//   - the ground and the edge of an input are transparent, and this draws both,
//     out of the tokens every other control uses
//   - SizeNameInputBorder is nought, which makes the toolkit's edge disappear in
//     every state INCLUDING focus, since the focused edge is drawn at that same
//     thickness. That is what frees the focus mark
//
// With the edge gone, where the words sit is this file's business, and the space
// around the field is what places them: eight from the left plus the four the
// renderer adds is twelve, and eight from the top with nothing added is eight.
// Both are what the chooser measures.
//
// The caret and the selection stay the toolkit's, drawn in colours this window
// already answers with its own tokens.
type FieldEntry struct {
	widget.Entry
	// onFocus is told when the keyboard arrives and leaves, and onHover when
	// the pointer does, so that the ground drawn around this field can answer.
	onFocus func(bool)
	onHover func(bool)
	// dressed is this field in the ground and the marks of the window, made
	// once. See Field for why once.
	dressed fyne.CanvasObject
}

// Asserted so that dropping one of these is a compile error rather than a field
// that silently stops answering, which is how the chooser lost its hover.
var (
	_ fyne.Widget       = (*FieldEntry)(nil)
	_ fyne.Focusable    = (*FieldEntry)(nil)
	_ desktop.Hoverable = (*FieldEntry)(nil)
)

// NewFieldEntry builds a text field that looks like the rest of this window.
//
// 🔴 The struct is built empty and its fields set, rather than copying a
// widget.Entry built by the toolkit into it. Copying by value takes the
// widget's own pointer to itself along, and ExtendBaseWidget then returns
// without doing anything, which leaves the control drawn with the size of one
// object and the layout of another: children at 0x0 and labels collapsed to an
// ellipsis, with nothing failing anywhere. That cost 54 choosers across 22
// states and was found by looking at a picture.
func NewFieldEntry() *FieldEntry {
	field := &FieldEntry{}
	field.ExtendBaseWidget(field)
	return field
}

// FocusGained tells whoever draws around this field that the keyboard is here.
func (f *FieldEntry) FocusGained() {
	f.Entry.FocusGained()
	if f.onFocus != nil {
		f.onFocus(true)
	}
}

// FocusLost tells them it left.
func (f *FieldEntry) FocusLost() {
	f.Entry.FocusLost()
	if f.onFocus != nil {
		f.onFocus(false)
	}
}

// MouseIn tells whoever draws around this field that the pointer is on it.
//
// # 🔴 Why a field has to be told at all, checked at the source
//
// widget.Entry answers a pointer with the shape of the cursor and with nothing
// a picture can see: entry.go of v2.8.1 has Cursor and no MouseIn, where
// button.go and select.go both keep a hovered flag and draw from it. Measured
// by tools/najazd.py on the real window on 2026-09-14, five runs: the pointer
// on either field of the Quick screen changed nought pixels, against 3588 to
// 15130 for every button and chooser in the same row (B-282). The owner's rule
// asks every control for every state, so the field is given the pointer here,
// on the object the driver finds in the tree, and the ground beside it answers
// the way a quiet button's does.
func (f *FieldEntry) MouseIn(*desktop.MouseEvent) {
	if f.onHover != nil {
		f.onHover(true)
	}
}

// MouseMoved is a pointer moving over the field, which changes nothing the
// ground shows.
func (f *FieldEntry) MouseMoved(*desktop.MouseEvent) {}

// MouseOut tells them the pointer left.
func (f *FieldEntry) MouseOut() {
	if f.onHover != nil {
		f.onHover(false)
	}
}

// Field dresses a text field in the ground, the edge and the marks this window
// gives every control, and it does so ONCE: asked again for the same field it
// hands back the same object, the way OnScreen does on a button or a chooser.
//
// # 🔴 Why once, measured on the real window
//
// Dressing a field hands its focus to the grounds of THAT dressing. The Quick
// screen dressed its count field on every draw and mounted the dressing only
// when the row had really changed, so after the second draw the grounds on the
// screen belonged to one dressing and the keyboard arrived at another: nought
// pixels changed on the real window with the focus on that field, while the
// seed field beside it, dressed only when mounted, showed its ring (B-279,
// tools/ognisko.py). The screen no longer builds before it decides, and this
// makes the same mistake harmless anywhere else.
func Field(entry *FieldEntry) fyne.CanvasObject { return field(entry, entry) }

// field is Field with the object put in the tree named apart from the field
// whose focus is watched.
//
// # Why the two are ever different
//
// A type embedding FieldEntry to answer a key differently is a different
// object to the driver: the focus manager focuses whatever it found in the
// tree and calls THAT object's TypedKey. Putting the embedded field in the
// tree would put the embedded TypedKey in charge, and the override would be
// dead code that compiles. The search field of the panel a kind is picked
// from is that type, and is the reason this seam exists.
func field(entry *FieldEntry, shown fyne.CanvasObject) fyne.CanvasObject {
	return fieldWith(entry, shown, nil)
}

// fieldWith is field with something standing before the text, inside the
// edge: the magnifier of a search field, which is what tells a person the
// field filters rather than takes a value.
func fieldWith(entry *FieldEntry, shown fyne.CanvasObject, leading fyne.CanvasObject) fyne.CanvasObject {
	if entry.dressed != nil {
		return entry.dressed
	}

	// One ground per state the field can be in, all built and all but one
	// hidden, so that answering the keyboard or the pointer is showing a
	// different one rather than painting.
	resting := paint.FieldGround(false, false)
	underPointer := paint.FieldGround(false, true)
	withKeyboard := paint.FieldGround(true, false)
	both := paint.FieldGround(true, true)
	for _, ground := range []fyne.CanvasObject{underPointer, withKeyboard, both} {
		ground.Hide()
	}
	current := resting
	focused, hovered := false, false
	redraw := func() {
		wanted := resting
		switch {
		case focused && hovered:
			wanted = both
		case focused:
			wanted = withKeyboard
		case hovered:
			wanted = underPointer
		}
		if wanted == current {
			return
		}
		// Hidden first, and the order matters: hiding is what asks the window
		// to draw again. Showing alone sets a flag and nothing else,
		// container.go:186 of v2.8.1.
		current.Hide()
		wanted.Show()
		current = wanted
	}
	entry.onFocus = func(here bool) {
		focused = here
		redraw()
	}
	entry.onHover = func(here bool) {
		hovered = here
		redraw()
	}

	// The words are placed by this space rather than by the toolkit: eight from
	// the left, to which the renderer adds its own four, which is where a
	// chooser starts its word. Nothing at the foot, so the field keeps whatever
	// height the row gives it.
	step := palette.StepOf(palette.NameStepTight)
	var typed fyne.CanvasObject = container.NewThemeOverride(shown, bareField{})
	if leading != nil {
		typed = container.New(layout.NewBorderLayout(nil, nil, leading, nil), leading, typed)
	}
	inside := container.New(layout.NewCustomPaddedLayout(step, 0, step, step), typed)

	entry.dressed = container.NewStack(resting, underPointer, withKeyboard, both, inside)
	return entry.dressed
}

// bareField is this window's theme with the chrome of an input taken out of it.
//
// # Why it asks the running application rather than naming a theme
//
// So that it cannot drift. The window has swapped themes once already, and a
// theme named here would go on answering with the old palette on the day the
// next swap happens, with nothing failing: the field would simply be the one
// control still wearing the previous window.
type bareField struct{}

var _ fyne.Theme = bareField{}

// Color hides the two things this control draws itself and defers the rest.
func (b bareField) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case toolkit.ColorNameInputBackground, toolkit.ColorNameInputBorder:
		return color.Transparent
	}
	return current().Color(name, variant)
}

// Size takes the toolkit's edge away, which is what lets the focus mark be the
// same one every other control uses.
//
// 🔴 Nought rather than a smaller number, and it does two things at once. The
// edge disappears in every state, focus included, since the toolkit draws that
// edge at this thickness whatever colour it picks for it. And the words move to
// the top of the field, because the renderer places them at twice this - which
// is why the space above them is put back outside, where it can be chosen.
func (b bareField) Size(name fyne.ThemeSizeName) float32 {
	if name == toolkit.SizeNameInputBorder {
		return 0
	}
	return current().Size(name)
}

func (b bareField) Font(style fyne.TextStyle) fyne.Resource { return current().Font(style) }

func (b bareField) Icon(name fyne.ThemeIconName) fyne.Resource { return current().Icon(name) }

// current is the theme the window is wearing right now.
func current() fyne.Theme {
	if app := fyne.CurrentApp(); app != nil {
		if settings := app.Settings(); settings != nil {
			if wearing := settings.Theme(); wearing != nil {
				return wearing
			}
		}
	}
	// A control built before an application exists is a test, and the toolkit's
	// own theme is what it would otherwise have got.
	return toolkit.DefaultTheme()
}
