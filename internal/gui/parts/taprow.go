package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// TapRow is one row of a list that answers the pointer and does something when
// it is clicked.
//
// # Why it exists
//
// The owner asked for it in one sentence on 2026-09-09: "I cannot copy a single
// value from values". The values were a drawing - rectangles and text out of
// paint - and a drawing takes no notice of anybody.
//
// # Why not a button
//
// parts.Button already draws itself and already answers the pointer, and it was
// tried first. It takes a label, an icon and a width, and a row here is two
// columns that have to line up down the whole list: a right aligned index in a
// monospace face, then the value. Passing that as a label would mean the button
// laying out the columns, which is the one thing it cannot do.
//
// # Why the row builds its own contents through a function
//
// So that the drawing stays in paint, where nothing takes a callback and so
// nothing can quietly become a control. This holds behaviour and asks the
// screen for the picture, which is the same split parts.Button keeps between
// itself and paint.ButtonFace.
//
// # 🔴 Why this does NOT take the keyboard, which is a decision and not an
// oversight
//
// Every other control in this window is reachable by keyboard and this one is
// not. The reason is the ceiling of the screen it lives on: quick draws up to
// MostShownAtOnce values, which is twenty thousand, and a focusable row would
// put twenty thousand stops in the Tab order. A person tabbing out of the seed
// field would be pressing Tab for the rest of the afternoon, which is worse for
// a keyboard than having no stop at all.
//
// What the keyboard still has: the Copy button in the heading, which takes the
// whole set. What it does not have is one value out of many, and that is
// written down in B-268 with the shape of the answer - a list widget of the
// toolkit, which holds ONE stop and moves a highlight inside itself with the
// arrow keys.
type TapRow struct {
	widget.BaseWidget
	// Draw is the picture of this row, asked for again whenever the state
	// changes. The flag says whether the pointer is on this row.
	Draw func(hovered bool) fyne.CanvasObject
	// OnTapped is what a click on this row does.
	OnTapped func()

	pointerIsHere bool
}

// The interfaces the driver asks about by type, asserted here so that dropping
// one is a compile error rather than a row that silently stops answering.
var (
	_ fyne.Widget       = (*TapRow)(nil)
	_ fyne.Tappable     = (*TapRow)(nil)
	_ desktop.Hoverable = (*TapRow)(nil)
)

// NewTapRow builds a row that redraws under the pointer and acts when clicked.
func NewTapRow(draw func(hovered bool) fyne.CanvasObject, tapped func()) *TapRow {
	row := &TapRow{Draw: draw, OnTapped: tapped}
	row.ExtendBaseWidget(row)
	return row
}

// Tapped does whatever this row is for.
func (r *TapRow) Tapped(*fyne.PointEvent) {
	if r.OnTapped != nil {
		r.OnTapped()
	}
}

// MouseIn notes that the pointer arrived and redraws.
func (r *TapRow) MouseIn(*desktop.MouseEvent) {
	r.pointerIsHere = true
	r.Refresh()
}

// MouseMoved has nothing to do, since a row looks the same wherever inside
// itself the pointer is.
func (r *TapRow) MouseMoved(*desktop.MouseEvent) {}

// MouseOut notes that the pointer left and redraws.
func (r *TapRow) MouseOut() {
	r.pointerIsHere = false
	r.Refresh()
}

// Hovered says whether the pointer is on this row, for a test that would
// otherwise have to read pixels.
func (r *TapRow) Hovered() bool { return r.pointerIsHere }

// CreateRenderer draws the row.
func (r *TapRow) CreateRenderer() fyne.WidgetRenderer {
	drawing := &tapRowRenderer{row: r, holder: container.NewStack()}
	drawing.Refresh()
	return drawing
}

// tapRowRenderer swaps the picture inside a holder rather than replacing it.
//
// 🔴 The holder is the part that is not obvious, and parts.Button paid for the
// lesson: the toolkit takes Objects() once, when it builds the renderer, and
// lays out what it took. A renderer that builds a fresh picture into a FIELD on
// every Refresh draws the one the toolkit was given and never the new one, so
// the control looks frozen while every test reading it passes.
type tapRowRenderer struct {
	row    *TapRow
	holder *fyne.Container
}

func (d *tapRowRenderer) Layout(size fyne.Size) {
	d.holder.Resize(size)
	d.holder.Move(fyne.NewPos(0, 0))
}

func (d *tapRowRenderer) MinSize() fyne.Size { return d.holder.MinSize() }

func (d *tapRowRenderer) Refresh() {
	picture := fyne.CanvasObject(container.NewStack())
	if d.row.Draw != nil {
		picture = d.row.Draw(d.row.pointerIsHere)
	}
	d.holder.Objects = []fyne.CanvasObject{picture}
	d.holder.Refresh()
}

func (d *tapRowRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{d.holder} }

func (d *tapRowRenderer) Destroy() {}
