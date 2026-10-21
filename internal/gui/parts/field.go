package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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
//   - the edge the toolkit strokes round the field is put away (withoutEdge),
//     in every state INCLUDING focus, since the focused edge is that same
//     rectangle in another colour. That is what frees the focus mark. Until
//     2026-10-21 it was given no width instead, which took the caret with it
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
	// onFocus is told when the keyboard arrives and leaves, onHover when the
	// pointer does, and onDisable when the field goes out of reach and comes
	// back, so that the ground drawn around this field can answer.
	onFocus   func(bool)
	onHover   func(bool)
	onDisable func(bool)
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

// Disable puts the field out of reach and tells whoever draws around it.
//
// # Why a field has to be told, checked at the source
//
// widget.Entry takes Disable from DisableableWidget, which sets a flag and
// refreshes the toolkit's renderer - widget/widget.go:180 of v2.8.1 - and that
// renderer greys the words and nothing else. The ground and the edge are drawn
// beside it, by this file, so a field nobody told went on wearing the edge that
// says it can be used for the whole of a run (DEC-253).
func (f *FieldEntry) Disable() {
	f.Entry.Disable()
	if f.onDisable != nil {
		f.onDisable(true)
	}
}

// Enable gives the field back and tells them so.
func (f *FieldEntry) Enable() {
	f.Entry.Enable()
	if f.onDisable != nil {
		f.onDisable(false)
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

	grounds, look := textGrounds(entry.Disabled())
	entry.onFocus, entry.onHover, entry.onDisable = look.focus, look.hover, look.disable

	// The words are placed by this space rather than by the toolkit: eight from
	// the left, to which the renderer adds its own four, which is where a
	// chooser starts its word. Nothing at the foot, so the field keeps whatever
	// height the row gives it.
	step := palette.StepOf(palette.NameStepTight)
	var typed fyne.CanvasObject = container.NewThemeOverride(shown, bareField{oneLine: true})
	if leading != nil {
		typed = container.New(layout.NewBorderLayout(nil, nil, leading, nil), leading, typed)
	}
	inside := container.New(layout.NewCustomPaddedLayout(step, 0, step, step), typed)

	entry.dressed = container.NewStack(append(grounds, inside)...)
	return entry.dressed
}

// textLooks are the grounds a text control can stand on: resting, under the
// pointer, with the keyboard, both, and out of reach.
//
// Five rather than eight, because out of reach is one ground whoever is on the
// field. See paint.FieldGround for why.
var textLooks = []paint.FieldLook{
	{},
	{Hovered: true},
	{Focused: true},
	{Focused: true, Hovered: true},
	{Disabled: true},
}

// textGrounds builds the grounds a text control stands on, all but the one it
// starts on hidden, and returns them with the look that says which is showing.
//
// It is the one drawing every text field in this window wears, so that a field
// and the chooser or button beside it answer the pointer, the keyboard and
// being out of reach the same way (DEC-215, DEC-237, DEC-253). The control that
// wears it wires its own hooks to the look: the text field does, and so does
// the recipe editor, which is the second text control and the reason this is a
// helper rather than a block inside one of them.
func textGrounds(disabled bool) ([]fyne.CanvasObject, *groundLook) {
	shown := &groundLook{each: make(map[paint.FieldLook]fyne.CanvasObject, len(textLooks))}
	grounds := make([]fyne.CanvasObject, 0, len(textLooks))
	resting := paint.FieldLook{}
	for _, look := range textLooks {
		ground := paint.FieldGround(look)
		if look != resting {
			ground.Hide()
		}
		shown.each[look] = ground
		grounds = append(grounds, ground)
	}
	shown.current = shown.each[resting]
	shown.disable(disabled)
	return grounds, shown
}

// groundLook is which of the grounds of one text control is showing, and the
// states that decide it.
//
// The states are kept even while they change nothing: a field enabled with the
// pointer resting on it and the keyboard in it has to come back under both,
// not at rest.
type groundLook struct {
	look    paint.FieldLook
	current fyne.CanvasObject
	each    map[paint.FieldLook]fyne.CanvasObject
}

func (g *groundLook) focus(here bool)  { g.look.Focused = here; g.show() }
func (g *groundLook) hover(here bool)  { g.look.Hovered = here; g.show() }
func (g *groundLook) disable(off bool) { g.look.Disabled = off; g.show() }

// show puts up the ground the states ask for. Out of reach wins over the
// pointer and the keyboard, as it does on the chooser and the button.
func (g *groundLook) show() {
	wanted := g.look
	if wanted.Disabled {
		wanted = paint.FieldLook{Disabled: true}
	}
	ground := g.each[wanted]
	if ground == g.current {
		return
	}
	// Hidden first, and the order matters: hiding is what asks the window to
	// draw again. Showing alone sets a flag and nothing else, container.go:186
	// of v2.8.1. Both kinds of ground here ask on hiding: a container at
	// container.go:96, and the bare rectangle a field out of reach stands on at
	// canvas/rectangle.go:60.
	g.current.Hide()
	ground.Show()
	g.current = ground
}

// withoutEdge is the toolkit's drawing of a text field with the edge it strokes
// round itself put away, and nothing else changed.
//
// # Why the edge is put away rather than given no width
//
// Until 2026-10-21 the bare theme below set the width of that edge to nought,
// which took the edge away in every state and the caret with it: widget.Entry
// of v2.8.2 draws its caret as wide as the same number, widget/entry.go:1918.
// The width is now the caret's, so the toolkit would draw its edge again - at
// rest in a colour the bare theme makes transparent, but with the keyboard in
// the field in the colour of the caret, and out of reach in the grey of
// disabled words. Neither colour can be taken away without the caret or the
// words.
//
// The edge is the one rectangle among the toolkit's objects that has a stroke,
// widget/entry.go:183, and nothing in the toolkit shows it again once hidden:
// it sets its colour and its size and never its visibility. The words stay
// where they were, because the entry moves them in by the width of the edge and
// its text takes the same number away again, widget/richtext.go:608.
//
// # 🔴 Why hidden here rather than cut by a clip, tried first on the same day
//
// A field spilling over an area that clips is how the window would hide the
// edge, but not how a render of it would: the toolkit's software painter, which
// draws every picture this project stores, moves a rounded rectangle clipped at
// its top or left side INTO the clip rather than cutting it, at
// internal/painter/software/draw.go in drawOblongStroke. The renders showed the
// edge as a bent line inside every focused field, and the window would not have.
// A picture that differs from the window is the fault GUI rule six names.
func withoutEdge(drawing fyne.WidgetRenderer) fyne.WidgetRenderer {
	for _, object := range drawing.Objects() {
		if edge, ok := object.(*canvas.Rectangle); ok && edge.StrokeWidth > 0 {
			edge.Hide()
		}
	}
	return drawing
}

// CreateRenderer is the toolkit's drawing of the field without its edge. See
// withoutEdge.
func (f *FieldEntry) CreateRenderer() fyne.WidgetRenderer {
	return withoutEdge(f.Entry.CreateRenderer())
}

// bareField is this window's theme with the chrome of an input taken out of it.
//
// # Why it asks the running application rather than naming a theme
//
// So that it cannot drift. The window has swapped themes once already, and a
// theme named here would go on answering with the old palette on the day the
// next swap happens, with nothing failing: the field would simply be the one
// control still wearing the previous window.
//
// oneLine is set for a field of one line. Such a field scrolls sideways under
// a value longer than itself, and the toolkit draws a bar for that along its
// foot: the owner measured it on 2026-10-21 as half the field under the
// pointer, on the seed. A field of one line follows its caret on its own, so
// the bar is given no width there and the recipe editor keeps its own.
type bareField struct {
	oneLine bool
}

var _ fyne.Theme = bareField{}

// Color hides the things this control draws itself, and the shadow, and defers
// the rest.
//
// The shadow is what the toolkit lays along the side of a field where more of
// the text lies out of sight. On a field of one line that is a dark smear at
// its end, and on the recipe editor it was the shadow along its right side the
// owner did not want on 2026-10-21. The caret moving and the bar of the editor
// already say that there is more.
func (b bareField) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case toolkit.ColorNameInputBackground, toolkit.ColorNameInputBorder, toolkit.ColorNameShadow:
		return color.Transparent
	}
	return current().Color(name, variant)
}

// Size gives the toolkit's edge the width of the caret, and the bars of a
// field of one line no width.
//
// 🔴 The edge itself is put away by withoutEdge, so the focus mark stays the
// one every other control uses. It was nought until 2026-10-21, see withoutEdge
// for why that cost the caret.
func (b bareField) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case toolkit.SizeNameInputBorder:
		return palette.CaretThickness
	case toolkit.SizeNameScrollBar, toolkit.SizeNameScrollBarSmall:
		if b.oneLine {
			return 0
		}
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
