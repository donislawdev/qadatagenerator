package paint

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Two layouts the toolkit does not have, and the guard against rebuilding the
// toolkit is what made each of them state its case here.
//
// Everything else this window arranges is done with a layout of Fyne: a column
// and a row with a named gap, a padded box, a border, a stack. These two are
// the pair that has no counterpart, and both exist for the same measured
// reason: container.NewVBox gives every child its MINIMUM height and not one
// pixel more, and layout.NewSpacer stretches the gaps BETWEEN children rather
// than the children. So a list that grows with the window cannot be built from
// them at all.

// evenRows divides the height it is given equally between its children.
type evenRows struct{ floor float32 }

func (l evenRows) MinSize(objects []fyne.CanvasObject) fyne.Size {
	widest := float32(0)
	for _, object := range objects {
		if width := object.MinSize().Width; width > widest {
			widest = width
		}
	}
	return fyne.NewSize(widest, l.floor*float32(len(objects)))
}

func (l evenRows) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	height := size.Height / float32(len(objects))
	top := float32(0)
	for _, object := range objects {
		object.Resize(fyne.NewSize(size.Width, height))
		object.Move(fyne.NewPos(0, top))
		top += height
	}
}

// EvenRows stacks rows that share the height they are given equally.
//
// The floor is the shortest a row may be, which is what the container asks for
// when the window has nothing to spare.
func EvenRows(rows ...fyne.CanvasObject) *fyne.Container {
	return container.New(evenRows{floor: theme.RowHeightMin}, rows...)
}

// fitHeight takes all the height it is offered and hands its child only as much
// as the rows inside will actually fill, capped per row.
type fitHeight struct {
	rows           int
	floor, ceiling float32
	chrome         float32
}

func (l fitHeight) MinSize(objects []fyne.CanvasObject) fyne.Size {
	widest := float32(0)
	for _, object := range objects {
		if width := object.MinSize().Width; width > widest {
			widest = width
		}
	}
	return fyne.NewSize(widest, l.floor*float32(l.rows)+l.chrome)
}

func (l fitHeight) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if l.rows == 0 {
		return
	}
	perRow := (size.Height - l.chrome) / float32(l.rows)
	if perRow < l.floor {
		perRow = l.floor
	}
	if perRow > l.ceiling {
		perRow = l.ceiling
	}
	height := perRow*float32(l.rows) + l.chrome
	for _, object := range objects {
		object.Resize(fyne.NewSize(size.Width, height))
		object.Move(fyne.NewPos(0, 0))
	}
}

// FitHeight wraps a card of rows so that the card grows with the window up to
// the point where its rows stop growing, and no further.
//
// # Why the surplus stays transparent rather than going to the card
//
// 🔴 Measured on 2026-09-08, and it is the difference between two arrangements
// that look identical in the code. A card stretched to the full height of the
// window had 450 pixels of empty interior at 1600 by 1180, and an empty framed
// box reads as a failure, while the same empty space on the bare window reads
// as room. So the card is given exactly the height its rows fill and the rest
// of the space is simply not drawn on.
//
// The chrome is the padding the card puts above its first row and below its
// last, named from the scale. It is added twice here rather than multiplied,
// because it is two margins and not one margin doubled, and the guard on
// literals would refuse the two in any case.
func FitHeight(rows int, chrome string, card fyne.CanvasObject) *fyne.Container {
	margin := theme.StepOf(chrome)
	return container.New(fitHeight{
		rows:    rows,
		floor:   theme.RowHeightMin,
		ceiling: theme.RowHeightMax,
		chrome:  margin + margin,
	}, card)
}
