// Keyboard handling for a table of rows.
//
// # Why a type here at all, when the rule is to use what the toolkit has
//
// The toolkit's table already answers the keyboard: it takes focus, the arrow
// keys move a highlight from cell to cell and the space bar selects one. None of
// that is rebuilt here and none of it should be. What the toolkit does not offer
// is a way to hear that focus has left, and without hearing it a mark drawn on
// the cell somebody is standing on stays behind after they tab away, pointing at
// a place the keyboard is no longer at.
//
// Asking the canvas instead was tried first and measured on 2026-08-18. It does
// not work, and the reason is worth keeping: when focus leaves, the table
// refreshes all of its cells from inside FocusLost, and the canvas still names
// the table as the focused object at that moment, because the focus manager only
// moves its record after FocusLost returns. Every cell therefore renders as
// though it still had focus. The probe printed six cells all saying focused=true
// while the button beside them had already taken focus.
//
// So this is the toolkit's own extension mechanism rather than a table of our
// own: an embedded table, two methods that say what happened, and nothing else.
// ExtendBaseWidget leaves the implementation alone once it is set, so the table's
// own call to it inside CreateRenderer does not undo the extension.
//
// # 🔴 OnHighlighted hears the keyboard here, and only the keyboard
//
// The toolkit calls OnHighlighted for two different things. The arrow keys move
// its highlight and it says where to, and the pointer passing over a cell says
// the same thing about that cell without moving anything: widget/table.go:777
// in fyne 2.8.1, from hoverAt, which leaves the keyboard's place alone. A
// screen cannot tell the two apart from the call, and the manifest browser
// could not: its card read out whatever entry the pointer last crossed and
// stayed there after the pointer left the table, beside the violet of the entry
// somebody had clicked (B-287, measured on the real window on 2026-09-23).
//
// A click was worse in a way nobody saw. Tapped hands the table focus first,
// FocusGained reports the keyboard's OLD place through the same call, and only
// then does the click select its own cell. A screen that follows the keyboard
// by selecting would scroll to where the keyboard last was and back again, and
// the row under the pointer would jump.
//
// So the pointer's events reach the toolkit with OnHighlighted out of reach,
// and the hover rectangle, the selection and everything else the toolkit does
// for a pointer stay exactly as they were. What a screen hears through
// OnHighlighted is what the name has always promised: where the keyboard is.
//
// # Why it lives here rather than beside the screen that first needed it
//
// It was written inside the manifest browser, which was the only screen with a
// table. The recipe editor's preview is the second, and a copy would be two
// spellings of one extension: the day one of them learned something about focus
// the other would go on trailing a key press behind. Moved on 2026-09-02, when
// the second user arrived rather than in anticipation of one.
package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The driver finds the pointer's half by type, so a method below that stopped
// matching the interface would go on compiling while the toolkit's own method
// took the event, OnHighlighted and all.
var (
	_ fyne.Tappable     = (*KeyTable)(nil)
	_ desktop.Hoverable = (*KeyTable)(nil)
)

// KeyTable is the toolkit's table, plus a word about focus.
type KeyTable struct {
	widget.Table

	// onFocus is told before the table acts on the change, so that whatever it
	// sets is already true by the time the refresh inside the toolkit runs.
	onFocus func(has bool)
	// ring is the mark of focus a screen built out of paint shows on every
	// control. It is mounted only by Framed, so a screen using the table bare
	// never shows it.
	ring ring
	// widths are the columns as Widths set them, and stretch is the one that
	// takes the remainder on every resize, or nothing when Widths was not
	// called and the toolkit's fixed widths are all there is.
	widths  []float32
	stretch int
}

// NewKeyTable builds a table that says when the keyboard arrives and leaves.
//
// The fields are set here rather than through the toolkit's constructor because
// that constructor returns its own type, which cannot be embedded after the
// fact. What it does is two assignments and a call to ExtendBaseWidget, and all
// three are done below.
func NewKeyTable(
	length func() (rows int, cols int),
	create func() fyne.CanvasObject,
	update func(widget.TableCellID, fyne.CanvasObject),
	onFocus func(has bool),
) *KeyTable {
	table := &KeyTable{onFocus: onFocus, ring: newRing(palette.RadiusControl)}
	table.Length = length
	table.CreateCell = create
	table.UpdateCell = update
	table.ShowHeaderRow = true
	table.ExtendBaseWidget(table)
	return table
}

// Framed is the table with the ring that says the keyboard is on it, which a
// screen built out of paint puts on itself. A screen that marks focus the
// toolkit's way uses the table bare and the ring is never mounted.
func (t *KeyTable) Framed() fyne.CanvasObject { return t.ring.framed(t) }

// Widths sets every column, and hands one of them whatever the table has left.
//
// # Why one column takes the remainder rather than every column having a width
//
// The toolkit's table knows only fixed widths, so a table of fixed columns is
// either narrower than its pane, with a strip of nothing on the right, or wider,
// with a scrollbar along the bottom and its last columns out of sight. The
// manifest browser had the second for its whole life (DEC-247). The column
// chosen here is the one that gains from a wide window, and it is never
// narrower than the width it was given, so that a narrow window still shows
// something of every column and scrolls sideways rather than crushing one.
//
// The remainder is worked out on every resize, which is the only moment the
// table's width is known. The toolkit lays a padding after every column, so
// that is taken off as well, or the last column would end one padding past
// the edge and grow a scrollbar for it.
func (t *KeyTable) Widths(widths []float32, stretch int) {
	t.widths = append([]float32(nil), widths...)
	t.stretch = stretch
	for column, width := range widths {
		t.SetColumnWidth(column, width)
	}
}

// Resize lays the table out at the size it is given, with the stretched
// column, if there is one, taking what the others leave.
//
// The toolkit's own resize goes first, so that the widths are set against a
// table that already knows how wide it is. SetColumnWidth refreshes only when
// the width really changes, so a resize to the same size costs nothing.
func (t *KeyTable) Resize(size fyne.Size) {
	t.Table.Resize(size)
	if t.widths == nil || t.stretch < 0 || t.stretch >= len(t.widths) {
		return
	}
	padding := t.Theme().Size(toolkit.SizeNamePadding)
	left := size.Width - padding*float32(len(t.widths))
	for column, width := range t.widths {
		if column != t.stretch {
			left -= width
		}
	}
	if left < t.widths[t.stretch] {
		left = t.widths[t.stretch]
	}
	t.SetColumnWidth(t.stretch, left)
}

// FocusGained records the focus before the toolkit redraws the cell, and draws
// the ring for a screen that mounted it.
func (t *KeyTable) FocusGained() {
	t.ring.set(true)
	if t.onFocus != nil {
		t.onFocus(true)
	}
	t.Table.FocusGained()
}

// FocusLost records the loss before the toolkit redraws every cell, which is the
// whole reason this type exists.
func (t *KeyTable) FocusLost() {
	t.ring.set(false)
	if t.onFocus != nil {
		t.onFocus(false)
	}
	t.Table.FocusLost()
}

// MouseIn, MouseMoved and Tapped are the toolkit's, with OnHighlighted kept out
// of reach while they run. The heading of this file says why.
func (t *KeyTable) MouseIn(event *desktop.MouseEvent) {
	t.pointing(func() { t.Table.MouseIn(event) })
}

// MouseMoved is the toolkit's hover, which moves its rectangle and nothing else.
func (t *KeyTable) MouseMoved(event *desktop.MouseEvent) {
	t.pointing(func() { t.Table.MouseMoved(event) })
}

// Tapped is the toolkit's click: focus, then the selection of the cell under
// the pointer, which a screen hears through OnSelected.
func (t *KeyTable) Tapped(event *fyne.PointEvent) {
	t.pointing(func() { t.Table.Tapped(event) })
}

// pointing runs one event of the pointer with OnHighlighted set aside, and puts
// it back however the event ends.
func (t *KeyTable) pointing(event func()) {
	heard := t.OnHighlighted
	t.OnHighlighted = nil
	defer func() { t.OnHighlighted = heard }()
	event()
}
