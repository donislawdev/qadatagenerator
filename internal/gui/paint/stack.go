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

// Lead puts something at the left at the width it asks for, and hands the rest
// of the width to what stands beside it.
//
// # Why a border and not a row
//
// A row gives every child the width it asks for and a field asks for very
// little, so the last field of a row of settings came out as wide as a word.
// The toolkit's border layout is the one that hands a side what is left, and
// it is named here so that a screen says which thing grows rather than
// spelling the layout out. The objects are given in reading order, which is
// also the order the keyboard walks them - the convenient NewBorder appends
// its centre first, which this project has paid for twice.
func Lead(leading, rest fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewBorderLayout(nil, nil, leading, nil), leading, rest)
}

// Trail is Lead the other way round: what stands on the right keeps the width
// it asks for, and what is left goes to the rest.
func Trail(rest, trailing fyne.CanvasObject) *fyne.Container {
	return container.New(layout.NewBorderLayout(nil, nil, nil, trailing), rest, trailing)
}

// Marked puts an icon before something, level with its first line: a verdict
// with its shape, a warning with its triangle.
//
// The shape rather than a colour on its own, for the reason measured for this
// palette: the colours that carry meaning differ in hue while barely differing
// in lightness, so somebody who does not separate hues sees one shade of grey.
// It was a method of the take screen until the recipe screen needed the same
// row on 2026-09-15.
//
// # 🔴 Why the icon stands in a column of its own
//
// The leading side of a border is given the whole height of the row, and an
// image centres itself in whatever it is given. So beside a sentence of three
// lines the icon stood level with the second one, and nothing said so: every
// sentence it had marked until 2026-09-23 fitted on one line. The mock of the
// About screen was the first with a notice long enough to wrap. A column gives
// the icon its own height at the top, which is what "level with its first line"
// promised.
func Marked(icon, ink string, words fyne.CanvasObject) *fyne.Container {
	return Lead(
		container.NewVBox(Pad(theme.NameStepHairline, theme.NameStepHairline,
			theme.NameStepHairline, theme.NameStepTight,
			Icon(icon, ink, theme.IconSmall))),
		words)
}

// Grid lays cells of one size out in as many columns as the width allows, and
// wraps the rest onto the next row.
//
// It is the toolkit's grid wrap, named here for the same reason Row and Column
// are: a screen says what it wants and not which layout does it. The cell is
// measured by the caller from the widest thing that will stand in it, because
// the toolkit wants one size for every cell and a cell narrower than its words
// lets them run into the next column - Fyne does not clip.
//
// 🔴 The cell is never nought. The toolkit counts columns by dividing the
// width by the cell plus its padding, and this window answers nought to the
// padding on purpose, so a cell of nought is nought over nought: NaN, which
// then arrives as the height of the grid, the height of the window and the
// width of a label three layouts away, where the text renderer slices by it
// and panics. Measured on 2026-09-15 on a screen built with an empty
// catalogue, where the grid had no kinds to measure a cell from.
func Grid(cell fyne.Size, cells ...fyne.CanvasObject) *fyne.Container {
	if cell.Width < theme.HairlineThickness {
		cell.Width = theme.HairlineThickness
	}
	if cell.Height < theme.HairlineThickness {
		cell.Height = theme.HairlineThickness
	}
	return container.NewGridWrap(cell, cells...)
}
