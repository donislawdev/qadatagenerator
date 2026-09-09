// Package paint draws the look of this window out of the toolkit's primitives.
//
// # Why this is a package of its own beside parts
//
// The two answer different questions and mixing them is what produced a window
// the owner called generic on 2026-09-08.
//
// parts holds CONTROLS: things a person presses, types into or moves the
// keyboard onto. Every one of them extends a widget of the toolkit, because
// behaviour is the expensive half and the toolkit already has it right.
//
// This package holds APPEARANCE: a card, a band of chrome, a badge, a heading,
// an icon in a colour. None of it is pressed, none of it takes focus, and none
// of it holds state. That is a rule rather than a description, and it is the
// line that keeps the two apart as both grow: anything here that grows a
// callback or a FocusGained belongs next door.
//
// # Why appearance is built rather than taken from the toolkit
//
// Settled as DEC-228 after three rounds of rework in one day. A screen made of
// stock widgets looks like the toolkit, because that is exactly what it is. The
// toolkit gives one ground, one text colour and one corner radius, so a card
// cannot be told from a band and a border cannot be told from a divider. The
// primitives underneath give all of it: canvas.Rectangle carries a corner
// radius per corner and a shadow, canvas.Text carries a size and a face of its
// own, and both were measured on the source of Fyne 2.8.1 rather than recalled.
//
// # What this package refuses to do
//
// Rebuild anything the toolkit already has. Column and Row live next door in
// parts and take their gap from the scale, and this package uses them rather
// than growing a second pair. Two layouts here have no counterpart in the
// toolkit and say so where they are declared.
package paint

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Fill is a rectangle in the colour of a token, rounded by the given radius.
//
// The radius is a parameter rather than read from the token, because the same
// colour is a card in one place and a badge in another, and the radius is what
// says which. Pass 0 for a shape with square corners, which is what every band
// of chrome touching the edge of the window uses.
func Fill(token string, radius float32) *canvas.Rectangle {
	shape := canvas.NewRectangle(theme.Ink(token))
	shape.CornerRadius = radius
	return shape
}

// Bordered is a filled rectangle with an edge of its own.
//
// # Why the edge is a whole step of the ladder away from the fill
//
// Measured on 2026-09-08: an edge one step from its fill measures around 1.1,
// which nobody can see, so the shape reads as having no edge at all and the
// card is indistinguishable from the window. The tokens carry the thresholds
// and the test beside the theme walks them, so the caller only has to name the
// right pair.
func Bordered(fillToken, edgeToken string, radius float32) *canvas.Rectangle {
	shape := Fill(fillToken, radius)
	shape.StrokeColor = theme.Ink(edgeToken)
	shape.StrokeWidth = theme.BorderThickness
	return shape
}

// Hairline is the thinnest line the window draws, across the full width it is
// given.
//
// # Why a rectangle rather than canvas.Line
//
// A line is drawn between two points and a layout has no way to stretch it, so
// every caller would have to know the width it will end up at. A rectangle with
// a minimum height and no minimum width takes whatever width the layout hands
// it, which is how a rule under a heading follows the heading rather than being
// told how long to be.
func Hairline(token string) fyne.CanvasObject {
	line := canvas.NewRectangle(theme.Ink(token))
	line.SetMinSize(fyne.NewSize(0, theme.HairlineThickness))
	return line
}

// Gap is empty space of a named step of the scale, for the one case a layout
// cannot express: a break between two groups inside a single column.
func Gap(step string) fyne.CanvasObject {
	return AtLeast(0, theme.StepOf(step), canvas.NewRectangle(color.Transparent))
}

// AtLeast gives a thing a floor under its size without drawing anything.
//
// # Why this is needed at all
//
// The toolkit sizes a container to what its children ask for, and several
// things here ask for almost nothing: a row of a list is as tall as its text, a
// button is as wide as its label. A drawing that says a row is 42 tall has no
// other way to say it, since padding around the text would change with the text.
//
// Pass 0 for an axis that should stay free.
func AtLeast(width, height float32, content fyne.CanvasObject) *fyne.Container {
	floor := canvas.NewRectangle(color.Transparent)
	floor.SetMinSize(fyne.NewSize(width, height))
	return container.NewStack(floor, content)
}

// Middle centres something on the vertical axis inside whatever it is given.
//
// A label in a row is otherwise pinned to the top of that row, and a row taller
// than its text is the normal case here rather than the exception, since rows
// grow with the window.
func Middle(content fyne.CanvasObject) *fyne.Container {
	return container.NewVBox(layout.NewSpacer(), content, layout.NewSpacer())
}

// Pad puts space on the four sides of something, each side named from the
// scale.
//
// # Why four names rather than one
//
// A band of chrome wants more room left and right than above and below, because
// the window is wider than it is tall and the eye reads the margin as a
// proportion. One value for all four sides is the case Inset next door already
// covers.
func Pad(top, bottom, left, right string, content fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedLayout(
		theme.StepOf(top), theme.StepOf(bottom),
		theme.StepOf(left), theme.StepOf(right)), content)
}

// PadAll puts the same named step on all four sides.
func PadAll(step string, content fyne.CanvasObject) *fyne.Container {
	return Pad(step, step, step, step, content)
}

// PadLeft puts a named step on the LEFT and nothing anywhere else.
//
// # Why this exists when Pad takes four names
//
// Because the scale has no step meaning none, and it must not gain one: a
// distance of nought is not a distance, and a seventh entry called "zero" is
// how a closed scale stops being closed. Pad therefore cannot express "a gap on
// this side only", and the case is common enough to have already cost a defect.
//
// The one that produced it: the seed field on the quick screen stands as the
// centre of a border layout so that it can never overflow the row, and a centre
// touches its edge with nothing between, because the padding of this theme is
// nought by decision. Padding all four sides instead would have moved the field
// down by the same step and taken it out of line with the four beside it.
func PadLeft(step string, content fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewCustomPaddedLayout(0, 0, theme.StepOf(step), 0), content)
}

// On stacks content over a ground, which is how every card, band and row in
// this window is built: a rectangle underneath and the content over it.
//
// A stack rather than a background property, because the toolkit has no such
// property on a container. Every child of a stack is given the full size, which
// is exactly what a ground wants.
func On(ground fyne.CanvasObject, content fyne.CanvasObject) *fyne.Container {
	return container.NewStack(ground, content)
}

// PanelShadow is the shadow under the one thing in this window that floats.
//
// See the theme for why there is exactly one of it, and why its colour is
// stated there rather than built here.
func PanelShadow() canvas.Shadow {
	return canvas.Shadow{
		Color:      theme.PanelShadowColour(),
		BlurRadius: theme.PanelShadowBlur,
		Spread:     theme.PanelShadowSpread,
		Offset:     fyne.NewPos(0, theme.PanelShadowDrop),
		Variant:    canvas.BoxShadow,
	}
}

// Clear is a shape that occupies its place and draws nothing.
//
// # Why this exists rather than each caller reaching for a colour
//
// 🔴 Found on the render on 2026-09-08, and it is a family rather than a slip.
// Every place that wanted "nothing here" was written as a rectangle in the
// colour of the ground it expected to sit on, and every one of them was wrong
// the moment it sat on a different ground: the empty half of a sidebar item
// drew the window colour onto the sidebar, and the reserved room at the end of
// a list row drew the card colour onto a striped row. Both are one step of the
// ladder away from their surroundings, which is precisely the amount that reads
// as a smear rather than as a fault.
//
// Nothing about this can be seen by reading the code, because at each call site
// the colour named IS the colour of the ground the author had in mind.
func Clear() *canvas.Rectangle {
	return canvas.NewRectangle(color.Transparent)
}

// Space reserves room and draws nothing, which is the pair of Clear and AtLeast
// most callers actually want.
func Space(width, height float32) *fyne.Container {
	return AtLeast(width, height, Clear())
}

// Outline is an edge around nothing: a shape a person can press, with whatever
// is behind it showing through.
//
// # Why a quiet button has no fill of its own
//
// It stands on three grounds in this window - the window, a band of chrome, and
// the row under the pointer - and a rectangle has one fill. Filling it with any
// of the three paints a patch of the wrong colour on the other two. The edge is
// what says it is a control, which is what DEC-211 settled and measured.
func Outline(edgeToken string, radius float32) *canvas.Rectangle {
	shape := Clear()
	shape.CornerRadius = radius
	shape.StrokeColor = theme.Ink(edgeToken)
	shape.StrokeWidth = theme.BorderThickness
	return shape
}
