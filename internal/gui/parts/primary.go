package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// PrimaryButton is the one action a screen is for, drawn as a filled slab
// rather than as another outlined word among several.
//
// # Why this exists at all
//
// The palette has carried action.primary and text.on-primary since the
// redesign of 2026-09-08. Both were measured, both were written down, and
// neither was used by a single line of code, so the window came out grey from
// end to end and no screen said which of its buttons was the one to press.
// A colour added to a register and used nowhere is a promise rather than a
// capability, which is the same family as a generator nobody calls.
//
// # Why the toolkit could not be asked for this
//
// widget.HighImportance is the toolkit's own answer and it was measured and
// turned down on 2026-09-05, for a reason no reading of the source would have
// shown. A focused high importance button is blend(primary, focus), and the
// blend at widget/button.go:453 collapses an opaque overlay to the overlay
// itself, so the button goes from its resting fill to the flat focus colour:
// a focus indicator of 1.23, where WCAG 1.4.11 asks for 3.0. It would have
// repaired one button per screen by breaking the mark on that same one.
//
// The refusal is about the toolkit's FOCUS drawing, not about filled buttons,
// and that distinction is what leaves this road open. A renderer of our own
// decides what focus looks like, so it does not have to be a flood.
//
// # Which ladder this reads from, and why it moved on 2026-09-09
//
// It asked the old palette for action.primary until that day, and the old
// palette answers #8A8A8A, which is a GREY. DEC-232 built a ladder with an
// accent on it and DEC-233 put that ladder on the window, so the one filled
// thing in the whole window went on being drawn in the colour of the design
// that preceded both. Nothing failed: a grey fill measures 5.55 on the new
// window ground, so every guard stayed quiet while the screen said nothing
// about which of its buttons was the one to press.
//
// That is the same family as the fault this type was written to fix. A colour
// added to a register and used nowhere is a promise rather than a capability,
// and a colour left behind when the register moves is a capability that has
// quietly become a promise again.
//
// # The arithmetic that makes this one work
//
// Measured 2026-09-09 on the values in the ladder, against the thresholds the
// token register carries:
//
//   - fill.primary, which is accent.9 #7059D6, stands 3.73 against the window
//     ground and 3.49 against a card, where an interface element needs 3.0. So
//     this button needs no added edge to be found, which is the whole reason
//     Outlined exists for the others: their fill stands 1.07 and cannot be
//     seen at all.
//   - text.on-fill, which is white, stands 5.14 on that fill, where a label
//     needs 4.5.
//   - 🔴 the white is not a preference. neutral.12, the ink every other piece
//     of text in this window uses, measures 4.35 on that same fill and misses
//     4.5. The ladder of neutrals cannot carry a label here, which is what the
//     token beside White in the theme says in words and this is the number.
//   - the focus ring is drawn in that same ink and INSIDE the fill, so both of
//     its sides sit on fill.primary and it stands 5.14 as well.
//
// 🔴 The ring stayed inside the fill, and the reason it was put there did NOT
// survive this change. It was written when the ink was #1B1B1B, the very
// colour of the old window ground: a ring on the outer edge had one side on
// the window measuring 1.00, so it was invisible on that side. White measures
// 19.15 on the new window ground, so today a ring drawn on the outer edge
// would be seen perfectly well. What keeps it inside is now a claim about
// SHAPE rather than about contrast - a ring on the outer edge reads as the
// button having grown by two pixels, and the buttons beside it have an edge of
// their own at that exact position. Left as it was for a reason that no longer
// holds, this would have been a rule nobody could check.
//
// # What a disabled one looks like, and why it stops being primary
//
// It loses the fill entirely and keeps only a quiet label. That is the rule
// already written for every other button in this window: a control nobody can
// press is not a control to find, and leaving the loudest thing on the screen
// on the one thing that does nothing is worse than leaving it off. It also
// means no new colour pair is introduced for a disabled state, since the
// toolkit's own disabled ink on the window ground is a pair already measured.
type PrimaryButton struct {
	widget.DisableableWidget

	// Text is the label, kept exported to match the toolkit's own button so
	// that a caller changing it does not have to learn a second spelling.
	Text string

	// OnTapped runs when the button is pressed, by pointer or by key.
	OnTapped func()

	// keyboardIsHere is kept here rather than read from the canvas, for the
	// reason parts.Button records next door: the canvas is wrong about focus
	// at exactly the moment it changes, because a control redraws from inside
	// FocusLost before the focus manager has moved its own record.
	keyboardIsHere bool
}

// Interfaces are asserted here rather than discovered at a call site, because
// a widget that quietly stops being focusable still compiles and still draws,
// and the only symptom is a person with a keyboard never reaching it.
var (
	_ fyne.Widget    = (*PrimaryButton)(nil)
	_ fyne.Tappable  = (*PrimaryButton)(nil)
	_ fyne.Focusable = (*PrimaryButton)(nil)
)

// NewPrimaryButton builds the filled action of a screen.
func NewPrimaryButton(label string, tapped func()) *PrimaryButton {
	button := &PrimaryButton{Text: label, OnTapped: tapped}
	button.ExtendBaseWidget(button)
	return button
}

// SetText changes the label and redraws.
//
// Named the way the toolkit's own button names it, because a caller swapping
// one for the other should not have to learn a second spelling. The recipe
// screen is that caller: its one action reads Generate while nothing is running
// and Stop while something is.
func (b *PrimaryButton) SetText(label string) {
	b.Text = label
	b.Refresh()
}

// Tapped presses the button, unless it cannot be pressed.
func (b *PrimaryButton) Tapped(*fyne.PointEvent) {
	if b.Disabled() || b.OnTapped == nil {
		return
	}
	b.OnTapped()
}

// FocusGained notes that the keyboard arrived and redraws.
func (b *PrimaryButton) FocusGained() {
	b.keyboardIsHere = true
	b.Refresh()
}

// FocusLost notes that the keyboard left and redraws.
func (b *PrimaryButton) FocusLost() {
	b.keyboardIsHere = false
	b.Refresh()
}

// TypedRune presses on the space bar, which is what the toolkit's own button
// answers to and therefore what a person arriving from any other control on
// this screen will try first.
func (b *PrimaryButton) TypedRune(r rune) {
	if r == ' ' {
		b.Tapped(nil)
	}
}

// TypedKey presses on Enter.
//
// Both names are accepted because a keyboard has two of these keys and nobody
// thinks of them as different: fyne.KeyReturn is above the right shift and
// fyne.KeyEnter is on the number pad. The toolkit's own button answers to
// neither, which is the whole reason parts.Button exists, and repeating that
// omission in a new control would be repeating a fault already paid for.
func (b *PrimaryButton) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	if event.Name == fyne.KeyReturn || event.Name == fyne.KeyEnter || event.Name == fyne.KeySpace {
		b.Tapped(nil)
	}
}

// CreateRenderer draws the slab, the ring and the label.
func (b *PrimaryButton) CreateRenderer() fyne.WidgetRenderer {
	fill := canvas.NewRectangle(color.Transparent)

	ring := canvas.NewRectangle(color.Transparent)
	ring.StrokeWidth = palette.BorderThickness

	label := canvas.NewText(b.Text, color.Transparent)
	label.Alignment = fyne.TextAlignCenter
	label.TextStyle = fyne.TextStyle{Bold: true}

	drawing := &primaryRenderer{button: b, fill: fill, ring: ring, label: label}
	drawing.Refresh()
	return drawing
}

type primaryRenderer struct {
	button *PrimaryButton
	fill   *canvas.Rectangle
	ring   *canvas.Rectangle
	label  *canvas.Text
}

// Layout puts the ring inside the fill rather than on its edge.
//
// The inset is one whole stroke, so that a stroke drawn centred on the path
// lands entirely within the fill. Half of it, which is what an edge drawn
// around a control uses, would leave the outer half of the ring on the window
// ground and give that half a contrast of 1.00.
func (r *primaryRenderer) Layout(size fyne.Size) {
	r.fill.Resize(size)
	r.fill.Move(fyne.NewPos(0, 0))

	inset := palette.BorderThickness
	r.ring.Resize(fyne.NewSize(size.Width-2*inset, size.Height-2*inset))
	r.ring.Move(fyne.NewPos(inset, inset))

	height := r.label.MinSize().Height
	r.label.Resize(fyne.NewSize(size.Width, height))
	r.label.Move(fyne.NewPos(0, (size.Height-height)/2))
}

// MinSize is the label plus the room the toolkit's own button gives its label.
//
// The same arithmetic as buttonRenderer.MinSize at widget/button.go:303, which
// is the label size plus its padding, and the padding is the inner padding of
// the theme on every side. Room chosen here instead would put this button at a
// different height from the outlined ones it stands beside, and a row of
// buttons of two heights reads as a mistake before it reads as a hierarchy.
func (r *primaryRenderer) MinSize() fyne.Size {
	room := 2 * beneath().Size(toolkit.SizeNameInnerPadding)
	size := r.label.MinSize()
	return fyne.NewSize(size.Width+2*room, size.Height+room)
}

// Refresh settles every colour from the palette and the state of the button.
func (r *primaryRenderer) Refresh() {
	radius := beneath().Size(toolkit.SizeNameButtonRadius)
	r.fill.CornerRadius = radius
	r.ring.CornerRadius = radius - palette.BorderThickness

	r.label.Text = r.button.Text
	r.label.TextSize = beneath().Size(toolkit.SizeNameText)

	if r.button.Disabled() {
		r.fill.FillColor = color.Transparent
		r.label.Color = beneath().Color(toolkit.ColorNameDisabled, toolkit.VariantDark)
		r.ring.StrokeColor = color.Transparent
		r.redraw()
		return
	}

	ink := palette.Ink(palette.TextOnFill)
	r.fill.FillColor = palette.Ink(palette.FillPrimary)
	r.label.Color = ink
	if r.button.keyboardIsHere {
		r.ring.StrokeColor = ink
	} else {
		r.ring.StrokeColor = color.Transparent
	}
	r.redraw()
}

// redraw asks each piece to repaint itself.
//
// canvas.Refresh on the widget is not enough on its own here: it redraws the
// object it is given, and the colours that changed live on three children of
// it. The toolkit's own renderers do the same thing for the same reason.
func (r *primaryRenderer) redraw() {
	r.fill.Refresh()
	r.ring.Refresh()
	r.label.Refresh()
}

func (r *primaryRenderer) Objects() []fyne.CanvasObject {
	// The label is last so that it is drawn over the fill, and the ring is
	// between them so that a focused button does not have its ring hidden by
	// its own slab.
	return []fyne.CanvasObject{r.fill, r.ring, r.label}
}

func (r *primaryRenderer) Destroy() {}
