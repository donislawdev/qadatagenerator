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
	"fyne.io/fyne/v2/widget"
)

// KeyTable is the toolkit's table, plus a word about focus.
type KeyTable struct {
	widget.Table

	// onFocus is told before the table acts on the change, so that whatever it
	// sets is already true by the time the refresh inside the toolkit runs.
	onFocus func(has bool)
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
	table := &KeyTable{onFocus: onFocus}
	table.Length = length
	table.CreateCell = create
	table.UpdateCell = update
	table.ShowHeaderRow = true
	table.ExtendBaseWidget(table)
	return table
}

// FocusGained records the focus before the toolkit redraws the cell.
func (t *KeyTable) FocusGained() {
	if t.onFocus != nil {
		t.onFocus(true)
	}
	t.Table.FocusGained()
}

// FocusLost records the loss before the toolkit redraws every cell, which is the
// whole reason this type exists.
func (t *KeyTable) FocusLost() {
	if t.onFocus != nil {
		t.onFocus(false)
	}
	t.Table.FocusLost()
}
