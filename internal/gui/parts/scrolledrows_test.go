package parts_test

import (
	"fmt"
	"sort"
	"strconv"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// listUnderTest is a list in a window of a known size, with a bind that
// remembers which index each row was given, so that a test can ask what a row
// shows without reading its pixels.
type listUnderTest struct {
	list    *parts.ScrolledRows
	ground  *canvas.Rectangle
	window  fyne.Window
	shows   map[*parts.TapRow]int
	label   string
	builtAs int
}

func newListUnderTest(t *testing.T, count int, room fyne.Size) *listUnderTest {
	t.Helper()
	under := &listUnderTest{ground: canvas.NewRectangle(nil), shows: map[*parts.TapRow]int{}, label: "first"}
	under.list = parts.NewScrolledRows(under.ground, palette.NameStepTight, func() fyne.CanvasObject {
		under.builtAs++
		return parts.NewTapRow(nil, nil)
	})
	under.hold(count, "first")
	under.window = test.NewWindow(under.list.Framed())
	under.window.SetPadded(false)
	under.window.Resize(room)
	t.Cleanup(under.window.Close)
	return under
}

func (u *listUnderTest) hold(count int, label string) {
	u.label = label
	u.list.Hold(count, 0, func(row fyne.CanvasObject, index int) {
		tap := row.(*parts.TapRow)
		u.shows[tap] = index
		words := label + " " + strconv.Itoa(index)
		tap.Rebind(func(bool) fyne.CanvasObject { return canvas.NewText(words, nil) }, nil)
	})
}

// placed is a row as the window shows it: where it stands in what the list
// scrolls, and which index it was given.
type placed struct {
	top   float32
	index int
	row   *parts.TapRow
}

// inWindow is every row the window shows, top to bottom.
func (u *listUnderTest) inWindow() []placed {
	var found []placed
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if !object.Visible() {
			return
		}
		switch shape := object.(type) {
		case *parts.TapRow:
			found = append(found, placed{top: shape.Position().Y, index: u.shows[shape], row: shape})
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(shape).Objects() {
				walk(child)
			}
		}
	}
	walk(u.window.Canvas().Content())
	sort.Slice(found, func(i, j int) bool { return found[i].top < found[j].top })
	return found
}

// viewOf is what a person sees of a list: the stretch of what it scrolls that
// is in the window, and how many rows of what height stand in it.
type viewOf struct {
	offset, window, pad, height float32
	count                       int
}

// wrongWith says everything wrong about the rows shown for a view: a row out of
// view, a row on a place that is not its own, a gap a person would see as empty
// card, or a row too many.
func wrongWith(rows []placed, view viewOf) []string {
	var wrong []string
	if view.count == 0 {
		if len(rows) > 0 {
			wrong = append(wrong, fmt.Sprintf("a list of nothing shows %d rows", len(rows)))
		}
		return wrong
	}
	cardEnds := view.pad + view.height*float32(view.count)
	seenFrom := max(view.offset, view.pad)
	seenTo := min(view.offset+view.window, cardEnds)
	for _, row := range rows {
		if row.top != view.pad+view.height*float32(row.index) {
			wrong = append(wrong, fmt.Sprintf("the row of index %d stands at %v, which is the place of "+
				"another index", row.index, row.top))
		}
		if row.top+view.height <= view.offset || row.top >= view.offset+view.window {
			wrong = append(wrong, fmt.Sprintf("the row of index %d is shown at %v, outside the window "+
				"from %v to %v", row.index, row.top, view.offset, view.offset+view.window))
		}
	}
	if len(rows) == 0 {
		return append(wrong, fmt.Sprintf("no row is shown between %v and %v", seenFrom, seenTo))
	}
	if rows[0].top > seenFrom {
		wrong = append(wrong, fmt.Sprintf("the first row shown starts at %v, under empty card from %v",
			rows[0].top, seenFrom))
	}
	for at := 1; at < len(rows); at++ {
		if rows[at].top != rows[at-1].top+view.height {
			wrong = append(wrong, fmt.Sprintf("between the rows of index %d and %d there is a gap or an "+
				"overlap", rows[at-1].index, rows[at].index))
		}
	}
	if last := rows[len(rows)-1]; last.top+view.height < seenTo {
		wrong = append(wrong, fmt.Sprintf("the last row shown ends at %v, over empty card to %v",
			last.top+view.height, seenTo))
	}
	return wrong
}

func (u *listUnderTest) view(height float32, count int) viewOf {
	return viewOf{
		offset: u.list.Offset.Y, window: u.list.Size().Height,
		pad: palette.StepOf(palette.NameStepTight), height: height, count: count,
	}
}

// The canary of wrongWith: a view it has to call right, and three plantings it
// has to call wrong. Without it every test below could pass on a checker that
// looks at nothing.
func TestTheCheckOfARowsPlaceCatchesWhatItIsFor(t *testing.T) {
	view := viewOf{offset: 100, window: 100, pad: 4, height: 40, count: 10}
	right := []placed{{top: 84, index: 2}, {top: 124, index: 3}, {top: 164, index: 4}}
	if wrong := wrongWith(right, view); len(wrong) > 0 {
		t.Fatalf("a view drawn right was called wrong: %v", wrong)
	}
	plantings := map[string][]placed{
		"a row missing in the middle":   {right[0], right[2]},
		"a row showing another index":   {right[0], {top: 124, index: 7}, right[2]},
		"the row at the foot left out":  {right[0], right[1]},
		"a row out of view still shown": append([]placed{{top: 4, index: 0}}, right...),
	}
	for name, planted := range plantings {
		if wrong := wrongWith(planted, view); len(wrong) == 0 {
			t.Errorf("%s passed the check", name)
		}
	}
}

// 🔴 The point of the type: a list of a hundred thousand values builds what the
// window shows and not one row more.
func TestAListOfAHundredThousandBuildsOnlyTheRowsInView(t *testing.T) {
	const count = 100000
	under := newListUnderTest(t, count, fyne.NewSize(300, 400))
	rows := under.inWindow()
	for _, wrong := range wrongWith(rows, under.view(palette.RowHeightMin, count)) {
		t.Error(wrong)
	}
	most := int(under.list.Size().Height/palette.RowHeightMin) + 2
	if under.builtAs > most {
		t.Errorf("the list built %d rows for a window that shows at most %d", under.builtAs, most)
	}
}

// The rows follow every way the area moves: the keys, which the toolkit does
// not report on its own (KeyScroll.to), and the wheel, which it does.
func TestTheRowsFollowTheKeysAndTheWheel(t *testing.T) {
	const count = 1000
	under := newListUnderTest(t, count, fyne.NewSize(300, 400))
	moves := []struct {
		name string
		move func()
	}{
		{"Down", func() { under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown}) }},
		{"PageDown", func() { under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyPageDown}) }},
		{"the wheel", func() {
			under.list.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -3*palette.RowHeightMin)})
		}},
		{"End", func() { under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd}) }},
		{"Up", func() { under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp}) }},
		{"Home", func() { under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyHome}) }},
	}
	for _, each := range moves {
		before := under.list.Offset.Y
		each.move()
		if under.list.Offset.Y == before {
			t.Fatalf("%s did not move the area, so it says nothing about the rows", each.name)
		}
		for _, wrong := range wrongWith(under.inWindow(), under.view(palette.RowHeightMin, count)) {
			t.Errorf("after %s: %s", each.name, wrong)
		}
		if each.name == "End" {
			rows := under.inWindow()
			if last := rows[len(rows)-1].index; last != count-1 {
				t.Errorf("at the end of the list the last row shown is %d, not the last value", last)
			}
		}
	}
}

// 🔴 What widget.List could not do and this card does: a few values in a tall
// window share its height, up to the ceiling of a row, and the ground ends under
// the last of them rather than going on as an empty frame.
func TestFewRowsShareTheHeightAndTheGroundEndsUnderTheLast(t *testing.T) {
	pad := palette.StepOf(palette.NameStepTight)
	cases := []struct {
		count int
		room  float32
		row   float32
	}{
		// Room for more than the ceiling: rows stop at it, the rest is window.
		{count: 4, room: 400, row: palette.RowHeightMax},
		// Room between floor and ceiling: rows take all of it.
		{count: 7, room: 7*(palette.RowHeightMin+palette.RowHeightMax)/2 + 2*pad, row: (palette.RowHeightMin + palette.RowHeightMax) / 2},
		// No room to spare: rows sit on the floor and the area scrolls.
		{count: 40, room: 400, row: palette.RowHeightMin},
	}
	for _, each := range cases {
		under := newListUnderTest(t, each.count, fyne.NewSize(300, each.room))
		rows := under.inWindow()
		for _, wrong := range wrongWith(rows, under.view(each.row, each.count)) {
			t.Errorf("%d rows in %v: %s", each.count, each.room, wrong)
		}
		if got := rows[0].row.Size().Height; got != each.row {
			t.Errorf("%d rows in %v are %v tall each, where they should be %v", each.count, each.room, got, each.row)
		}
		if got, want := under.ground.Size().Height, each.row*float32(each.count)+2*pad; got != want {
			t.Errorf("%d rows in %v stand on a ground %v tall, where they fill %v", each.count, each.room, got, want)
		}
	}
}

// A row that leaves the view at one end and comes back at the other, holding a
// new value, does not bring the pointer with it, and lights up again on the
// next move of a pointer that really is on it.
func TestARowMovedToAnotherValueForgetsThePointer(t *testing.T) {
	under := newListUnderTest(t, 1000, fyne.NewSize(300, 400))
	first := under.inWindow()[0].row
	first.MouseIn(&desktop.MouseEvent{})
	if !first.Hovered() {
		t.Fatal("the row did not take the pointer, so this says nothing")
	}

	for under.shows[first] == 0 {
		before := under.list.Offset.Y
		under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
		if under.list.Offset.Y == before {
			t.Fatal("the row of the first value was never moved to another, so this says nothing")
		}
	}
	if first.Hovered() {
		t.Errorf("the row now showing value %d still says the pointer is on it", under.shows[first])
	}
	first.MouseMoved(&desktop.MouseEvent{})
	if !first.Hovered() {
		t.Error("a pointer moving on a row that forgot it did not light it up again")
	}
}

// A new set starts at the top, every row shown holds a value of the new set,
// and a set of nothing shows nothing, ground included.
func TestANewSetStartsAtTheTopWithEveryRowRebound(t *testing.T) {
	under := newListUnderTest(t, 1000, fyne.NewSize(300, 400))
	under.list.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})

	under.hold(500, "second")
	if under.list.Offset.Y != 0 {
		t.Errorf("a new set opens %v down rather than at its first value", under.list.Offset.Y)
	}
	rows := under.inWindow()
	for _, wrong := range wrongWith(rows, under.view(palette.RowHeightMin, 500)) {
		t.Error(wrong)
	}
	for _, row := range rows {
		if words := textIn(row.row); words != "second "+strconv.Itoa(row.index) {
			t.Errorf("the row of index %d draws %q, which is from the set before", row.index, words)
		}
	}

	under.hold(0, "none")
	if rows := under.inWindow(); len(rows) > 0 {
		t.Errorf("a set of nothing shows %d rows", len(rows))
	}
	if under.ground.Visible() {
		t.Error("a set of nothing still draws the ground of a card")
	}
}
