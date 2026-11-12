package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
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
// So the toolkit's chrome is put away rather than replaced, and this draws the
// ground and the edge out of the tokens every other control uses:
//
//   - the ground and the edge the toolkit draws for an input are hidden
//     (withoutEdge), in every state INCLUDING focus, since the focused edge is
//     that same rectangle in another colour. That is what frees the focus mark
//   - the shadow the toolkit lays at the end of a field holding more than it
//     shows is gone from the whole window (fynetheme, unpainted)
//
// 🔴 No theme is handed to the field alone, and that is measured rather than
// tidy: the toolkit gives such a theme a new scope every time it draws the
// field again after a minute out of sight, and parses every font for the scope
// anew, about 6 MB that stay until the program ends (B-370, the reason is
// beside fynetheme's unpainted). What comes back with that: a field holding
// more than it shows draws the toolkit's scroll bar along its foot, as thin as
// a resting bar and as thick as a bar under the pointer, because the theme of
// the window has one width for the bars of a field and of a list. The owner
// took that on 2026-11-12, having seen it drawn, as the price of the memory.
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
	onInvalid func(bool)
	// invalid is whether the field holds what its screen would refuse. It is
	// kept here as well as told, because a screen may judge its field before
	// the field is dressed, and the dressing has to start in that state.
	invalid bool
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

// SetInvalid marks the field as holding what its screen would refuse, or takes
// the mark away, and tells whoever draws around it.
//
// # Why the screen judges and the field only shows
//
// What is wrong with a value belongs to the screen and to the mode behind it,
// and a field that knew the rule would be a second copy of it. So the field
// carries a yes or no and the drawing answers it, the way it answers Disable.
func (f *FieldEntry) SetInvalid(wrong bool) {
	if f.invalid == wrong {
		return
	}
	f.invalid = wrong
	if f.onInvalid != nil {
		f.onInvalid(wrong)
	}
}

// Invalid reports whether the field is marked as holding what its screen would
// refuse.
func (f *FieldEntry) Invalid() bool { return f.invalid }

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

	grounds, look := textGrounds(entry.Disabled(), entry.invalid)
	entry.onFocus, entry.onHover, entry.onDisable = look.focus, look.hover, look.disable
	entry.onInvalid = look.refuse

	// The words are placed by this space rather than by the toolkit: eight from
	// the left, to which the renderer adds its own four, which is where a
	// chooser starts its word. Nothing at the foot, so the field keeps whatever
	// height the row gives it.
	step := palette.StepOf(palette.NameStepTight)
	typed := shown
	if leading != nil {
		typed = container.New(layout.NewBorderLayout(nil, nil, leading, nil), leading, typed)
	}
	inside := container.New(layout.NewCustomPaddedLayout(step, 0, step, step), typed)

	entry.dressed = container.NewStack(append(grounds, inside)...)
	return entry.dressed
}

// textLooks are the grounds a text control can stand on: resting, under the
// pointer, with the keyboard, both, and out of reach - and the first four again
// for a field holding what its screen would refuse (B-363).
//
// Nine rather than sixteen, because out of reach is one ground whoever is on the
// field and whatever it holds. See paint.FieldGround for why.
var textLooks = []paint.FieldLook{
	{},
	{Hovered: true},
	{Focused: true},
	{Focused: true, Hovered: true},
	{Disabled: true},
	{Invalid: true},
	{Invalid: true, Hovered: true},
	{Invalid: true, Focused: true},
	{Invalid: true, Focused: true, Hovered: true},
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
func textGrounds(disabled, invalid bool) ([]fyne.CanvasObject, *groundLook) {
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
	shown.look.Invalid = invalid
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
func (g *groundLook) refuse(bad bool)  { g.look.Invalid = bad; g.show() }

// show puts up the ground the states ask for. Out of reach wins over the
// pointer, the keyboard and a refused value, as it does on the chooser and the
// button: nobody can change the value, so nothing says it should be changed.
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

// withoutEdge is the toolkit's drawing of a text field with the ground it lays
// under the words and the edge it strokes round them put away, and nothing
// else changed.
//
// # Why put away rather than given no colour or no width
//
// Both are the toolkit's rectangles, and neither can be taken away through its
// theme without taking something else: widget.Entry of v2.8.2 draws its caret
// as wide as the edge, widget/entry.go:1918, so an edge of no width took the
// caret with it, which the owner reported on 2026-10-21 as not seeing where
// they typed. Its colour is the caret's under the keyboard and the grey of
// disabled words out of reach. And a colour handed to the field alone needs a
// theme of the field's own, which costs the window a set of fonts every time
// the field is drawn again after a minute out of sight (FieldEntry says why).
//
// They are the only two rectangles among the objects the renderer of the entry
// hands over, widget/entry.go:181 to 186, the caret living inside the content
// rather than here. Nothing in the toolkit shows either again once hidden: it
// sets their colours and sizes and never their visibility. The words stay
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
		if chrome, ok := object.(*canvas.Rectangle); ok {
			chrome.Hide()
		}
	}
	return drawing
}

// CreateRenderer is the toolkit's drawing of the field without its ground and
// its edge. See withoutEdge.
func (f *FieldEntry) CreateRenderer() fyne.WidgetRenderer {
	return withoutEdge(f.Entry.CreateRenderer())
}
