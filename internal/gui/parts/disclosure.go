package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
)

// Disclosure is a heading a person presses to show what is under it, and press
// again to put it away.
//
// # Why this is not widget.Accordion, which does the same thing
//
// The toolkit's accordion builds its own heading, and that heading is a
// widget.Button it creates inside its renderer at widget/accordion.go:246.
// Nothing outside the toolkit can reach it.
//
// That mattered on 2026-09-05. A focused button is flooded with the focus
// colour and its label measured 1.56 against the 4.5 of WCAG 1.4.3, and the
// repair was a theme around the control, which parts.Flooded carried until
// DEC-253 deleted it with nothing left to flood. A theme can
// only be given to a subtree, and the smallest subtree containing that heading
// is the whole accordion, list included. Measured rather than assumed: the
// rows of that list are widget.NewLabel, so they read the same colour name the
// heading does, and darkening the heading darkens every row under it.
//
// So the choice was between reaching a control we cannot reach and building
// the one thing the accordion gave us that we still want. This is the second.
//
// # The chevron is kept, and the reason it could be is measured
//
// The toolkit draws an arrow on its heading that turns as the section opens,
// and losing it would leave the state of the section to be read off the
// section. The first drawing here did lose it, and the render showed a second
// thing with it: the heading is as wide as the pane, so a centred label sat
// 371 pixels in, which reads as a title of the window rather than a heading of
// a list.
//
// Both are answered by the toolkit's own button. The alignment is a field of
// it, and the arrow is a themed resource, which the stored tree of the
// accordion showed as themed="foreground". A themed resource is painted with
// the colour its own widget resolves that name to, so the arrow turns dark on
// a flooded heading exactly as the words beside it do, and it needs no
// separate answer for WCAG 1.4.11.
//
// # Why this is a plain type and not a widget
//
// It draws nothing of its own. It is a button and a body in a column, so
// building a renderer here would be rebuilding a column the toolkit already
// has, which is the mistake this package has already paid for once.
type Disclosure struct {
	header *Button
	body   fyne.CanvasObject
	column *fyne.Container
}

// NewDisclosure builds a heading with something hidden under it.
func NewDisclosure(title string, body fyne.CanvasObject) *Disclosure {
	disclosure := &Disclosure{body: body}
	disclosure.header = NewButton(title, disclosure.toggle)
	// A heading is read from where the text begins, and this one is as wide as
	// the pane it heads. The toolkit centres a label by default, which put the
	// words of this one 371 pixels from the left when it was first drawn.
	disclosure.header.Leading = true
	// No edge, because a heading is not a thing to press so much as a thing to
	// open, and this one is as wide as the pane it heads, so a box around it
	// would read as a box around the section. The owner settled that on
	// 2026-09-05 with the number beside it, and it was the toolkit's low
	// importance until the button began drawing itself on 2026-09-09.
	disclosure.header.Plain = true
	body.Hide()
	disclosure.markState()
	disclosure.column = container.NewVBox(disclosure.header.OnScreen(), body)
	return disclosure
}

// markState puts the arrow of the heading the way the section is.
//
// The two icons are this window's own chevrons rather than the toolkit's pair.
// They were the toolkit's until the button began drawing itself, at which point
// asking the theme for them stopped being the only route to a chevron of ours
// and started being the long way round to the same picture.
func (d *Disclosure) markState() {
	if d.body.Visible() {
		d.header.ShowIcon(paint.IconChevronUp)
		return
	}
	d.header.ShowIcon(paint.IconChevron)
}

// OnScreen is what a layout is given.
//
// The heading inside is handed over already dressed, so a caller that puts this
// on screen cannot forget the part that keeps the heading readable.
func (d *Disclosure) OnScreen() fyne.CanvasObject { return d.column }

// Open shows the body, for a caller with no keyboard.
//
// It is exported for the reason the screen's own openers are: the guard that
// holds what every screen draws and the probe that renders each state to a
// picture both have to reach the open state, and a section nobody can
// photograph is a section nobody reviews.
func (d *Disclosure) Open() {
	if d.body.Visible() {
		return
	}
	d.body.Show()
	d.markState()
	d.column.Refresh()
}

// Close puts the body away again.
func (d *Disclosure) Close() {
	if !d.body.Visible() {
		return
	}
	d.body.Hide()
	d.markState()
	d.column.Refresh()
}

// IsOpen says whether the body is showing.
func (d *Disclosure) IsOpen() bool { return d.body.Visible() }

func (d *Disclosure) toggle() {
	if d.body.Visible() {
		d.Close()
		return
	}
	d.Open()
}
