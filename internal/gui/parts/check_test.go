package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

// A check box answers a tap, the space bar and Enter with a toggle, tells its
// callback each time, and draws the mark only while it is ticked.
//
// The mark is counted rather than the flag read, for the reason every guard in
// this package gives: a flag set and never drawn is the shape of the fault the
// chooser had, and a person sees the picture.
func TestACheckBoxTogglesByTapAndByKeyAndDrawsItsMark(t *testing.T) {
	told := []bool{}
	check := NewCheck("Field format (L2)", func(on bool) { told = append(told, on) })
	if marksIn(check) != 0 {
		t.Fatal("a box nobody ticked draws a mark")
	}

	check.Tapped(nil)
	if !check.Checked || marksIn(check) != 1 {
		t.Errorf("tapped once: checked %v, marks drawn %d", check.Checked, marksIn(check))
	}
	check.TypedRune(' ')
	if check.Checked || marksIn(check) != 0 {
		t.Errorf("space on a ticked box: checked %v, marks drawn %d", check.Checked, marksIn(check))
	}
	check.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if !check.Checked {
		t.Error("Enter did not tick the box")
	}
	if want := []bool{true, false, true}; len(told) != len(want) || told[0] != want[0] || told[1] != want[1] || told[2] != want[2] {
		t.Errorf("the callback was told %v, want %v", told, want)
	}
}

// SetChecked from code tells the callback, because the take screen reaches
// the state where a run is possible by ticking boxes from outside a window and
// relies on the same settings a person's clicks produce.
func TestSettingACheckBoxFromCodeTellsTheCallbackOnceAndOnlyOnAChange(t *testing.T) {
	told := 0
	check := NewCheck("Value meaning (L3)", func(bool) { told++ })
	check.SetChecked(true)
	check.SetChecked(true)
	if told != 1 {
		t.Errorf("setting the same state twice told the callback %d times, want 1", told)
	}
	if marksIn(check) != 1 {
		t.Error("set from code and the mark is not drawn")
	}
}

// A disabled box keeps its mark, so that the kinds a run in flight was ordered
// with can still be read, and ignores a tap.
func TestADisabledCheckBoxKeepsItsMarkAndIgnoresATap(t *testing.T) {
	check := NewCheck("File bytes (L0)", nil)
	check.SetChecked(true)
	check.Disable()
	check.Tapped(nil)
	if !check.Checked {
		t.Error("a tap on a disabled box changed it")
	}
	if marksIn(check) != 1 {
		t.Error("a disabled box lost its mark, so nobody can read what a run was ordered with")
	}
}

// marksIn counts the marks drawn in a box: the one image its face carries when
// it is ticked, and nothing otherwise.
func marksIn(check *Check) int {
	found := 0
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		switch shape := object.(type) {
		case *canvas.Image:
			found++
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		}
	}
	for _, object := range test.WidgetRenderer(check).Objects() {
		walk(object)
	}
	return found
}
