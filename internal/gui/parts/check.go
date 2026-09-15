package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
)

// Check is a check box that draws itself, out of the same face the mock draws.
//
// # Why not widget.Check, read on the source of v2.8.1
//
// The toolkit's check box marks focus with a filled circle behind the box
// (widget/check.go, focusIndicator in ColorNameFocus) and draws the box out of
// two icons of its own. Neither is the ring every other control of this window
// draws, and neither could be reached from outside: the circle is the
// renderer's, the icons are the theme's. Eight of these stand in one column of
// the take screen beside fields and choosers that draw themselves, and a
// column where one kind of control marks focus with a circle and the others
// with an edge is two programs (DEC-215, DEC-245).
//
// # What it does not take over
//
// Nothing about a check box lives in the toolkit's widget that this needs:
// a tap toggles a flag and tells a callback, and the keys are Space and Enter.
// So unlike Select this does not embed the toolkit's control at all.
type Check struct {
	widget.DisableableWidget

	// Text is the label beside the box, and OnChanged is told when the box
	// changes, by a tap, a key or SetChecked.
	Text      string
	Checked   bool
	OnChanged func(bool)

	keyboardIsHere bool
	pointerIsHere  bool
	pointerIsDown  bool
}

// Asserted so that dropping a method is a compile error rather than a box that
// silently stops answering, which is how the chooser lost its hover.
var (
	_ fyne.Widget       = (*Check)(nil)
	_ fyne.Tappable     = (*Check)(nil)
	_ fyne.Focusable    = (*Check)(nil)
	_ desktop.Hoverable = (*Check)(nil)
	_ desktop.Mouseable = (*Check)(nil)
)

// NewCheck builds a check box with the words beside it and what to tell when
// it changes.
func NewCheck(label string, changed func(bool)) *Check {
	check := &Check{Text: label, OnChanged: changed}
	check.ExtendBaseWidget(check)
	return check
}

// OnScreen is what a layout is handed, and it is the control itself: the face
// is drawn by its renderer, so there is nothing to dress it in.
func (c *Check) OnScreen() fyne.CanvasObject { return c }

// SetChecked puts the box in a state, the way a tap would, and tells the
// callback when that is a change.
//
// The callback runs for a change made from code as well as from a tap, which
// is what the toolkit's check does and what the take screen relies on: the
// guard reaching a state where a run is possible ticks the boxes and expects
// the same settings a person's clicks would produce.
func (c *Check) SetChecked(on bool) {
	if c.Checked == on {
		return
	}
	c.Checked = on
	c.Refresh()
	if c.OnChanged != nil {
		c.OnChanged(on)
	}
}

// Tapped toggles the box.
func (c *Check) Tapped(*fyne.PointEvent) {
	if c.Disabled() {
		return
	}
	c.SetChecked(!c.Checked)
}

// MouseIn is the pointer arriving, which the face shows on the box.
func (c *Check) MouseIn(*desktop.MouseEvent) {
	c.pointerIsHere = true
	c.Refresh()
}

// MouseMoved is the pointer moving over the box, which changes nothing.
func (c *Check) MouseMoved(*desktop.MouseEvent) {}

// MouseOut is the pointer leaving, and it forgets a press as well, for the
// reason Button gives: the release lands on whatever is under the pointer by
// then, and never on the control that was pressed.
func (c *Check) MouseOut() {
	c.pointerIsHere, c.pointerIsDown = false, false
	c.Refresh()
}

// MouseDown is the button going down on the box.
func (c *Check) MouseDown(*desktop.MouseEvent) {
	c.pointerIsDown = true
	c.Refresh()
}

// MouseUp is the button coming up again.
func (c *Check) MouseUp(*desktop.MouseEvent) {
	c.pointerIsDown = false
	c.Refresh()
}

// FocusGained draws the ring.
func (c *Check) FocusGained() {
	c.keyboardIsHere = true
	c.Refresh()
}

// FocusLost takes it away.
func (c *Check) FocusLost() {
	c.keyboardIsHere = false
	c.Refresh()
}

// TypedRune toggles on the space bar, which is the key a check box answers to
// in every toolkit.
func (c *Check) TypedRune(r rune) {
	if r == ' ' {
		c.Tapped(nil)
	}
}

// TypedKey toggles on Space and Enter.
func (c *Check) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	if event.Name == fyne.KeyReturn || event.Name == fyne.KeyEnter || event.Name == fyne.KeySpace {
		c.Tapped(nil)
	}
}

// CreateRenderer draws the face.
func (c *Check) CreateRenderer() fyne.WidgetRenderer {
	drawing := &checkRenderer{check: c}
	drawing.Refresh()
	return drawing
}

// checkRenderer holds the face of the check box and rebuilds it on refresh,
// the way buttonRenderer does: the face is a picture of a state, and a state
// change is a new picture.
type checkRenderer struct {
	check  *Check
	holder *fyne.Container
}

func (r *checkRenderer) Layout(size fyne.Size) {
	r.holder.Resize(size)
	r.holder.Move(fyne.NewPos(0, 0))
}

func (r *checkRenderer) MinSize() fyne.Size { return r.holder.MinSize() }

func (r *checkRenderer) Refresh() {
	face := paint.CheckFace(paint.CheckLook{
		Label:    r.check.Text,
		Checked:  r.check.Checked,
		Focused:  r.check.keyboardIsHere,
		Disabled: r.check.Disabled(),
		Hovered:  r.check.pointerIsHere,
		Pressed:  r.check.pointerIsDown,
	})
	if r.holder == nil {
		r.holder = container.NewStack(face)
		return
	}
	r.holder.Objects = []fyne.CanvasObject{face}
	r.holder.Refresh()
}

func (r *checkRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.holder} }

func (r *checkRenderer) Destroy() {}
