package catalogue

import (
	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// page is one page of the catalogue: a section per control, top to bottom,
// inside the margins the screens of the window keep.
func page(text i18n.Catalogue, samples Samples, controls []control) fyne.CanvasObject {
	sections := make([]fyne.CanvasObject, 0, len(controls))
	for _, shown := range controls {
		sections = append(sections, section(text, samples, shown))
	}
	return paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepMajor, palette.NameStepMajor,
		paint.Column(palette.NameStepSection, sections...))
}

// section is one control: its name over a line, then its variants.
//
// # How a section reads
//
// As a table of states. The names of the states stand once over the columns,
// the name of each variant stands at the start of its row, and every cell of a
// block is one size, so the eye reads down a column and compares one state
// across the variants. The size is the largest cell of the block, worked out
// from the cells rather than chosen, so a longer word or another font moves it
// with them.
//
// A variant standing alone is turned on its side: its states go down the page,
// each named where a variant would be. A very long label is as wide as a row,
// and five of them side by side would run past the edge of the window, where
// Fyne draws them over whatever stands there.
func section(text i18n.Catalogue, samples Samples, shown control) fyne.CanvasObject {
	names := namesWidth(text, shown)
	blocks := []fyne.CanvasObject{paint.SectionHeading(text.T(shown.name), nil)}
	for _, block := range blocksOf(shown.variants) {
		if block[0].alone {
			blocks = append(blocks, sideways(text, samples, shown.states, block[0], names))
			continue
		}
		blocks = append(blocks, asTable(text, samples, shown.states, block, names))
	}
	return paint.Column(palette.NameStepGroup, blocks...)
}

// blocksOf puts the variants that share a cell size together, and every
// variant standing alone in a block of its own, after them.
func blocksOf(variants []variant) [][]variant {
	var shared []variant
	var apart [][]variant
	for _, each := range variants {
		if each.alone {
			apart = append(apart, []variant{each})
			continue
		}
		shared = append(shared, each)
	}
	if len(shared) == 0 {
		return apart
	}
	return append([][]variant{shared}, apart...)
}

// asTable is a block of variants: the names of the states over the columns,
// then a row per variant.
func asTable(text i18n.Catalogue, samples Samples, states []state, block []variant, names float32) fyne.CanvasObject {
	heading := make([]fyne.CanvasObject, 0, len(states))
	size, captions := fyne.Size{}, fyne.Size{}
	for _, each := range states {
		caption := cell(stateName(text, each))
		captions = captions.Max(caption.MinSize())
		heading = append(heading, caption)
	}
	cells := make([][]fyne.CanvasObject, len(block))
	for at, shown := range block {
		for _, each := range states {
			drawn := cell(controlIn(text, samples, shown, each))
			size = size.Max(drawn.MinSize())
			cells[at] = append(cells[at], drawn)
		}
	}
	// The captions take the width of the cells under them and a height of
	// their own, so the heading sits on the first row rather than floating a
	// control's height above it.
	size = size.Max(fyne.NewSize(captions.Width, 0))
	captions.Width = size.Width
	rows := []fyne.CanvasObject{named(nil, names, paint.Grid(captions, heading...))}
	for at, shown := range block {
		rows = append(rows, named(variantName(text, shown), names, paint.Grid(size, cells[at]...)))
	}
	return paint.Column(palette.NameStepTight, rows...)
}

// sideways is one variant with its states going down the page.
func sideways(text i18n.Catalogue, samples Samples, states []state, shown variant, names float32) fyne.CanvasObject {
	rows := []fyne.CanvasObject{variantName(text, shown)}
	for _, each := range states {
		drawn := controlIn(text, samples, shown, each)
		if !shown.fills {
			drawn = cell(drawn)
		}
		rows = append(rows, named(stateName(text, each), names, drawn))
	}
	return paint.Column(palette.NameStepTight, rows...)
}

// named puts a name at the start of a row, in a column as wide as the widest
// name of the section, so that what follows the names starts at one edge.
func named(name fyne.CanvasObject, width float32, rest fyne.CanvasObject) fyne.CanvasObject {
	var standing fyne.CanvasObject = paint.Space(width, 0)
	if name != nil {
		standing = paint.AtLeast(width, 0, paint.Middle(name))
	}
	return paint.Lead(standing, rest)
}

// namesWidth is the widest name that stands at the start of a row of this
// section, variant and state alike, with the gap after it.
func namesWidth(text i18n.Catalogue, shown control) float32 {
	widest := float32(0)
	for _, each := range shown.variants {
		widest = max(widest, variantName(text, each).MinSize().Width)
	}
	for _, each := range shown.states {
		widest = max(widest, stateName(text, each).MinSize().Width)
	}
	return widest + palette.StepOf(palette.NameStepGroup)
}

// variantName and stateName are the two kinds of name on a page, told apart
// the way a table of this window tells a row from a column: the name of a
// column is a caption.
func variantName(text i18n.Catalogue, shown variant) fyne.CanvasObject {
	return paint.Text(text.T(shown.name), palette.TypeBody, palette.TextPrimary)
}

func stateName(text i18n.Catalogue, in state) fyne.CanvasObject {
	return paint.Caption(text.T(stateNames[in]))
}

// cell gives what stands in it the size it asks for rather than the size of
// the cell, so that a button in a wide cell is still as wide as a button, and
// the room left over is the gap between it and the next one.
//
// The gap is the one between rows of a list, split over the two sides of a
// cell. Five choosers and the names beside them fill the width of a window
// opened at StartWidth with a gap one step wider, measured on the render of
// 2026-09-25.
func cell(content fyne.CanvasObject) fyne.CanvasObject {
	return paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepHairline, palette.NameStepTight,
		paint.Column(palette.NameStepHairline, paint.Row(palette.NameStepHairline, content)))
}

// controlIn is one control of one variant, built fresh and put into a state.
func controlIn(text i18n.Catalogue, samples Samples, shown variant, in state) fyne.CanvasObject {
	made, drawn := shown.build(text, samples)
	if !in.apply(made) {
		// A register that lists a state its control cannot take is a fault
		// of this package, and the guard beside it refuses one. Drawing the
		// control at rest under the name of that state would be a picture
		// that lies about what it shows.
		panic("the catalogue of controls lists a state its control cannot take: " +
			text.T(shown.name) + ", " + text.T(stateNames[in]))
	}
	return drawn
}
