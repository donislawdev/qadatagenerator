package paint

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Column and Row are the two stacks every screen is built out of.
//
// # Why they live here rather than beside the controls
//
// They were written in parts, which is where the CONTROLS of this window live,
// and they are not controls: they carry no behaviour, take no callback and hold
// no state. That misplacement had a cost that was not visible until a control
// needed a primitive of its own. paint drew the faces of the controls, so paint
// imported parts, so parts could not import paint without a cycle - and the
// thing parts needed from paint was a rectangle that gives a control a floor
// under its height.
//
// Moving two functions that were never controls is the whole fix, and the guard
// against rebuilding the toolkit has nothing new to object to: both of them are
// still the toolkit's own layout, asked for the distance rather than assuming
// it.

// Column stacks things down the screen with a gap named from the scale.
//
// # Why this exists, measured 2026-09-08
//
// This is the single biggest reason the window did not look like the drawing it
// was built from, and it went unnoticed for as long as the window existed.
//
// The scale of spacings has six steps and has had since before the first
// screen: hairline, tight, related, group, section, major. The accepted drawing
// uses them the way any drawing does - small gaps inside a group of things read
// together, large gaps between groups read apart. That difference is what makes
// a screen look composed rather than poured.
//
// container.NewVBox puts the SAME distance between everything, and that
// distance is the theme's padding, which is the smallest step there is. So a
// screen built out of plain columns has one gap where the drawing has six, and
// every part of it sits the same distance from every other part. Nothing groups
// with anything, and the eye is given nothing to hold on to. The window read as
// flat for that reason and not because of any colour or any font.
//
// The toolkit has the answer already and this project had not reached for it:
// layout.NewCustomPaddedVBoxLayout takes the distance rather than assuming it.
// So there is no layout of our own here, and the guard against rebuilding
// pieces of the toolkit has nothing to object to.
//
// # Why the gap is named rather than given as a number
//
// Same reason every colour in this window is named: a number written at the one
// place it is used is a number nobody can hold to a scale, and six screens each
// choosing their own would drift apart within a session. The name is what makes
// two screens agree without either of them knowing about the other.
func Column(step string, children ...fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedVBoxLayout(theme.StepOf(step)), children...)
}

// Row lays things across the screen with a gap named from the scale.
//
// The companion of Column and for the same reason. A row of buttons wants them
// close enough to read as a set, and the toolkit's own row gives them the same
// distance as everything else on the screen gets.
func Row(step string, children ...fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedHBoxLayout(theme.StepOf(step)), children...)
}
