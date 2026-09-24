package paint

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// CheckLook is everything the face of a check box depends on.
//
// A struct for the reason ButtonLook and ChooserLook are: the face is drawn by
// the mock and by the control from ONE description, so the two cannot drift.
type CheckLook struct {
	// Label is the words beside the box.
	Label string
	// Checked is whether the box carries the mark.
	Checked bool

	// Focused, Disabled, Hovered and Pressed mean here exactly what they mean
	// on a button, and are drawn from the same tokens, because a check box and
	// a button on one screen have to look like one family (DEC-215, DEC-237).
	Focused  bool
	Disabled bool
	Hovered  bool
	Pressed  bool
}

// CheckFace is the look of a check box, in whatever state it is in.
//
// # Why the box is a small field rather than a small button
//
// The box holds a value somebody set, which is what a field does, so it wears
// the ground and the edge of a field: bg.control under border.control, the
// pointer changing the ground the way it does on every quiet control, focus as
// the edge in mark.focus. Rounded by RadiusCheck rather than RadiusControl,
// because at eighteen across a control's radius made it a circle - measured on
// the render of 2026-09-15 before the first line of this file.
//
// # Why the mark is drawn in the accent and the box is NOT filled with it
//
// A filled accent is reserved for the one action a screen is for. Eight boxes
// filled with it would stand beside the filled button and compete with it, so
// a chosen box keeps its ground and carries the mark in mark.current, which
// the register measures on every ground the box stands on. mark.current
// rather than text.accent, though both are accent.11: a tick is a mark and
// answers to 3.0, and the same shade on the pressed ground measures 4.33 on
// the graphite of 2026-09-24 (4.49 before it), which is short of what a label
// needs and well past what a mark needs. The token names the threshold, so the
// token is the difference.
//
// # Why disabled keeps the mark
//
// A disabled button loses its edge because nobody can press it. A disabled
// check box still SAYS something - which kinds a run in flight was ordered
// with - so it keeps its box and its mark and goes quiet: the edge in
// line.quiet, the words and the mark in text.secondary.
func CheckFace(look CheckLook) fyne.CanvasObject {
	ink, markInk := theme.TextPrimary, theme.MarkCurrent
	var box fyne.CanvasObject
	switch {
	case look.Disabled:
		ink, markInk = theme.TextSecondary, theme.TextSecondary
		box = inside(Outline(theme.LineQuiet, theme.RadiusCheck))
	default:
		under, edge := pointerGround(look.Hovered, look.Pressed)
		if under == "" {
			under = theme.BgControl
		}
		if look.Focused {
			edge = theme.MarkFocus
		}
		box = inside(Bordered(under, edge, theme.RadiusCheck))
	}

	var mark fyne.CanvasObject = Clear()
	if look.Checked {
		mark = container.NewCenter(Icon(IconCheck, markInk, theme.IconSmall))
	}
	square := AtLeast(theme.CheckBox, theme.CheckBox, container.NewStack(box, mark))

	return Row(theme.NameStepTight,
		Middle(square),
		Middle(Text(look.Label, theme.TypeBody, ink)))
}

// CheckWidth is how wide a cell has to be to hold any of these check boxes
// whole, box and words.
//
// Measured from the face rather than added up, for the reason ChooserWidth
// gives: arithmetic written beside a drawing is a second copy of the drawing.
// Rounded up, for the reason FieldWidth gives: a fractional width makes the
// layout round a neighbour's position into a gap nobody chose.
func CheckWidth(labels ...string) float32 {
	widest := float32(0)
	for _, label := range labels {
		if measured := CheckFace(CheckLook{Label: label}).MinSize().Width; measured > widest {
			widest = measured
		}
	}
	return float32(math.Ceil(float64(widest)))
}
