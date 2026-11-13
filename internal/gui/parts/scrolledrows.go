package parts

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// ScrolledRows is a scrolling column of rows of one height, of which only the
// rows in view exist.
//
// # Why it exists
//
// The quick screen built a row for every value it produced, and every row is a
// widget with a renderer. Measured in the real window on 2026-11-12 with
// tools/pamiec_w_oknie.py (B-370), two runs: each draw of a thousand values
// added 1018 renderers and about 3.5 MB of live heap that stayed until the
// toolkit cleaned up a minute later, and every frame walked all thousand rows
// whether they could be seen or not. That walk is what made the window lag
// under the pointer at ten thousand values (DEC-262), and it is why the screen
// could not show more than a thousand.
//
// Here the number of rows is what the area can show, about twenty five at the
// size the window is kept, whatever the number of values. A new set of values
// rebinds the rows that are there rather than building new ones.
//
// # Why not widget.List
//
// It builds only the rows in view too, and it was turned down for the same
// screen on 2026-09-09 (DEC-242) for a reason that still holds: it gives every
// row a fixed height, while this card spends spare height on its rows, so that
// four values in a tall window are not four lines over a field of black. It
// also marks the keyboard with a lit row, where every control of this window
// marks it with its edge (DEC-243). This keeps the area of the window, the ring
// and the keys of KeyScroll, and the rows of the screen.
//
// # How a row finds its value
//
// Row i stands at padding plus i times the height of a row, and the rows in view
// are the ones whose place falls in the window of the area. Row i is held by
// slot i modulo the number of rows built, so scrolling by one row rebinds one
// row rather than all of them, and a row keeps its value for as long as it is in
// view, which is what keeps the pointer and the word of a click on the value
// they belong to.
//
// # 🔴 What bounds the number of values
//
// Positions are float32 in the toolkit, which holds every whole number up to
// 16 777 216 exactly. A row of RowHeightMin at the scale the owner's machine
// draws at, 1.5, is 63 device pixels, so past about 266 000 rows the place of a
// row is no longer a whole pixel. Producing a hundred thousand values takes
// 20 to 87 ms by kind (measured 2026-11-12, four runs each, the first
// dropped), which is the other bound, and it is the mode's, not this type's.
type ScrolledRows struct {
	KeyScroll

	// body is what the area scrolls: the ground and the rows built so far, laid
	// out by rowsInView.
	body   *fyne.Container
	ground fyne.CanvasObject
	// pad is the room between the edge of the ground and the rows, on all four
	// sides.
	pad   float32
	build func() fyne.CanvasObject

	count int
	// width is the room the widest row of the set needs, said by whoever knows
	// what the rows show.
	width float32
	bind  func(row fyne.CanvasObject, index int)

	// built are the rows made so far, and shows says which index each holds,
	// or noIndex.
	built []fyne.CanvasObject
	shows []int
}

// noIndex is what a row holds before it was bound to any value.
const noIndex = -1

// NewScrolledRows makes an empty list. The ground is drawn behind the rows, as
// tall as they are and no taller, padding is the named step between its edge
// and the rows, and build makes one row whenever the area needs another.
//
// Nothing is shown until Hold says how many rows there are.
func NewScrolledRows(ground fyne.CanvasObject, padding string, build func() fyne.CanvasObject) *ScrolledRows {
	rows := &ScrolledRows{ground: ground, pad: palette.StepOf(padding), build: build}
	rows.body = container.New(rowsInView{rows: rows}, ground)
	rows.setUp(rows.body)
	rows.OnScrolled = func(fyne.Position) { rows.place() }
	rows.ExtendBaseWidget(rows)
	return rows
}

// Hold shows a new set of rows from the top: count of them, the widest needing
// width, and bind putting the value of an index into a row.
//
// 🔴 bind is taken anew with every set, and the screen hands one that closes
// over the values of that set. A row built for the last set may still be asked
// to draw itself, by the end of the word saying a click worked, and a bind
// reading the screen's current values would then index a set that is shorter or
// gone.
func (r *ScrolledRows) Hold(count int, width float32, bind func(row fyne.CanvasObject, index int)) {
	r.count, r.width, r.bind = count, width, bind
	for slot := range r.shows {
		r.shows[slot] = noIndex
	}
	if count == 0 {
		r.ground.Hide()
	} else {
		r.ground.Show()
	}
	r.to(fyne.Position{})
	r.Refresh()
}

// Framed is the list together with the ring saying it holds the keyboard.
//
// Stated here rather than inherited, because the one KeyScroll has would put
// the area inside this type into the window rather than this type, and the
// toolkit would then lay out and resize an object that does not know its rows.
func (r *ScrolledRows) Framed() fyne.CanvasObject { return r.ring.framed(r) }

// Resize places the rows again, because a taller area shows more of them while
// what it scrolls, already taller than the area, keeps its size and is not laid
// out again.
func (r *ScrolledRows) Resize(size fyne.Size) {
	r.KeyScroll.Resize(size)
	r.place()
}

// heightOfRow is the height every row is given: the room shared out between
// them, held between the floor and the ceiling of a row of values.
func (r *ScrolledRows) heightOfRow(room float32) float32 {
	share := (room - r.pad - r.pad) / float32(r.count)
	return min(max(share, palette.RowHeightMin), palette.RowHeightMax)
}

// place puts the ground under the rows and a row on every place in view.
//
// Asked again for a place it already serves it changes nothing, which is what
// lets the keys report a clamped move twice (KeyScroll.to).
func (r *ScrolledRows) place() {
	room := r.body.Size()
	if r.count == 0 || room.Height <= 0 {
		r.hideAll()
		return
	}
	height := r.heightOfRow(room.Height)
	r.ground.Move(fyne.Position{})
	r.ground.Resize(fyne.NewSize(room.Width, height*float32(r.count)+r.pad+r.pad))

	first, last := r.inView(height)
	if last < first {
		r.hideAll()
		return
	}
	r.buildUpTo(last - first + 1)
	slots := len(r.built)
	for index := first; index <= last; index++ {
		slot := index % slots
		row := r.built[slot]
		if r.shows[slot] != index {
			r.bind(row, index)
			r.shows[slot] = index
		}
		row.Resize(fyne.NewSize(room.Width-r.pad-r.pad, height))
		row.Move(fyne.NewPos(r.pad, r.pad+height*float32(index)))
		row.Show()
	}
	for slot, index := range r.shows {
		if index < first || index > last {
			r.built[slot].Hide()
		}
	}
}

// inView is the first and the last index whose row falls in the window of the
// area, or a last below the first when none does.
func (r *ScrolledRows) inView(height float32) (first, last int) {
	window := r.Size().Height
	if window <= 0 {
		return 0, noIndex
	}
	top := r.Offset.Y - r.pad
	first = max(int(top/height), 0)
	last = min(int(math.Ceil(float64((top+window)/height)))-1, r.count-1)
	return first, last
}

// buildUpTo makes rows until there are at least wanted of them. Rows are never
// thrown away: an area that grew once is likely to grow again, and a row out of
// view is hidden, so the toolkit neither walks nor paints it.
func (r *ScrolledRows) buildUpTo(wanted int) {
	for len(r.built) < wanted {
		row := r.build()
		r.built = append(r.built, row)
		r.shows = append(r.shows, noIndex)
		r.body.Objects = append(r.body.Objects, row)
	}
}

// hideAll hides every row and leaves each the value it holds, so that a row
// coming back into view with that value is not bound again.
func (r *ScrolledRows) hideAll() {
	for _, row := range r.built {
		row.Hide()
	}
}

// rowsInView lays out what the list scrolls: as tall as all of its rows at
// their floor, as wide as its widest row, and the rows placed by the list.
type rowsInView struct{ rows *ScrolledRows }

func (l rowsInView) MinSize([]fyne.CanvasObject) fyne.Size {
	r := l.rows
	return fyne.NewSize(r.width+r.pad+r.pad, palette.RowHeightMin*float32(r.count)+r.pad+r.pad)
}

func (l rowsInView) Layout([]fyne.CanvasObject, fyne.Size) { l.rows.place() }
