package parts

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
)

// The guard over what a pointer does to a control, and the fault it was written
// after.
//
// # 🔴 What happened, measured on the real binary on 2026-09-09
//
// A window with the pointer over a sidebar item was identical to one with the
// pointer in empty space, to the last of 1,965,600 pixels. Every control in
// this package had stopped answering a mouse, and the owner found it by using
// the program.
//
// It was not a feature nobody had written. The toolkit draws a hover of its own
// - widget/button.go:382 and widget/select.go:435 both reach for
// ColorNameHover - so this was a capability these controls HAD and lost on the
// day they took their drawing off the toolkit. Nothing failed, because a state
// nobody draws leaves no mark to miss, and no test read a pointer.
//
// # Why the question is asked of the DRAWING and not of the flags
//
// Because a flag set and never read is exactly the shape of the fault. The
// chooser was still receiving MouseIn under its own renderer the whole time -
// the method is inherited and the toolkit still set its own hovered flag - and
// still redrew, and still drew the same picture. A guard asking whether the
// events arrive would have been green throughout.
//
// So this asks what a person asks: does it LOOK different.

// touchable is one control of this package that a pointer can reach, together
// with the two things a pointer does to it.
//
// The interfaces are taken from the widget rather than asserted at compile
// time, because a compile time assertion cannot fail at run time and a mutation
// that removes a method does not compile - which reports BROKEN rather than
// caught, and proves nothing.
type touchable struct {
	name   string
	widget fyne.Widget
}

func touchables() []touchable {
	plain := NewButton("About", func() {})
	plain.Plain = true
	return []touchable{
		{"a quiet button", NewButton("Copy", func() {})},
		{"a plain button, which draws no edge in any state", plain},
		{"the filled action of a screen", NewPrimaryButton("Generate", func() {})},
		{"a chooser", NewSelect([]string{"pesel", "nip"}, func(string) {})},
		{"a check box", NewCheck("Field format (L2)", func(bool) {})},
	}
}

// lookOf is every colour the drawing of a control carries, in the order it is
// drawn, which is as close as this package gets to what somebody sees.
//
// Images are skipped on purpose and the reason is worth keeping: an icon of
// this window is a bitmap with its ink already baked in, so a change of ink
// produces a different image rather than a different colour, and reading one
// here would mean rasterising it. No state below is carried by an icon alone.
func lookOf(widget fyne.Widget) string {
	var colours []string
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		switch shape := object.(type) {
		case *canvas.Rectangle:
			colours = append(colours, fmt.Sprintf("rect fill=%v stroke=%v", shape.FillColor, shape.StrokeColor))
		case *canvas.Text:
			colours = append(colours, fmt.Sprintf("text %q colour=%v", shape.Text, shape.Color))
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		}
	}
	for _, object := range widget.CreateRenderer().Objects() {
		walk(object)
	}
	return fmt.Sprint(colours)
}

func pointerOn(widget fyne.Widget) {
	widget.(desktop.Hoverable).MouseIn(&desktop.MouseEvent{})
}

func pointerOff(widget fyne.Widget) {
	widget.(desktop.Hoverable).MouseOut()
}

func pressOn(widget fyne.Widget) {
	widget.(desktop.Mouseable).MouseDown(&desktop.MouseEvent{})
}

func pressOff(widget fyne.Widget) {
	widget.(desktop.Mouseable).MouseUp(&desktop.MouseEvent{})
}

// TestEveryControlAPointerCanTouchDrawsEachOfItsStatesDifferently is the
// mechanical form of the sentence the owner used: everything looks the same,
// with no telling one from another.
//
// Three states and three comparisons, not two. Resting against hovered is the
// obvious one. Hovered against pressed is the one that would have been missed:
// a control could answer the pointer arriving and ignore the button going down,
// and every picture of it hovering would still be correct.
func TestEveryControlAPointerCanTouchDrawsEachOfItsStatesDifferently(t *testing.T) {
	for _, each := range touchables() {
		t.Run(each.name, func(t *testing.T) {
			resting := lookOf(each.widget)

			pointerOn(each.widget)
			hovered := lookOf(each.widget)
			if hovered == resting {
				t.Errorf("%s draws exactly the same under the pointer as at rest, so nothing "+
					"on screen says the pointer is on it:\n  %s", each.name, resting)
			}

			pressOn(each.widget)
			pressed := lookOf(each.widget)
			if pressed == hovered {
				t.Errorf("%s draws exactly the same held down as merely hovered, so nothing "+
					"answers the press until the window does:\n  %s", each.name, hovered)
			}
			if pressed == resting {
				t.Errorf("%s draws exactly the same held down as at rest:\n  %s", each.name, resting)
			}

			pressOff(each.widget)
			pointerOff(each.widget)
			if back := lookOf(each.widget); back != resting {
				t.Errorf("%s did not go back to how it looked before the pointer arrived:\n"+
					"  before %s\n  after  %s", each.name, resting, back)
			}
		})
	}
}

// TestAControlForgetsAPressThePointerCarriedAway covers the one way a pressed
// control could stay pressed for the rest of the run.
//
// # 🔴 Why this is not a hypothetical
//
// The driver finds the object under the pointer AT THE MOMENT THE BUTTON IS
// RELEASED and delivers MouseUp to that one - internal/driver/glfw/window.go:475
// and :583. Press a button, slide off it, let go: the release lands on whatever
// is under the pointer by then, or on nothing at all, and never on the control
// that was pressed.
//
// Nothing would fail. The control would go on drawing a state it genuinely
// believes it is in, and every stored tree of it would be a legal picture.
func TestAControlForgetsAPressThePointerCarriedAway(t *testing.T) {
	for _, each := range touchables() {
		t.Run(each.name, func(t *testing.T) {
			resting := lookOf(each.widget)

			pointerOn(each.widget)
			pressOn(each.widget)
			// No MouseUp. The pointer simply leaves, which is what the driver
			// reports when a press is released somewhere else.
			pointerOff(each.widget)

			if left := lookOf(each.widget); left != resting {
				t.Errorf("%s stayed drawn as touched after the pointer left while held down, "+
					"and nothing will ever tell it otherwise:\n  resting %s\n  left    %s",
					each.name, resting, left)
			}
		})
	}
}

// TestADisabledControlDrawsNothingForAPointer says that a control nobody can
// use does not answer a pointer, however much of one it receives.
//
// The events are still delivered and still recorded: the driver does not ask
// whether a widget is disabled before offering it a pointer, and a control that
// is enabled while the pointer sits on it must not then be wrong about where
// the pointer is. So the flags are kept and the drawing throws them away, and
// this is the guard over the second half of that.
func TestADisabledControlDrawsNothingForAPointer(t *testing.T) {
	for _, each := range touchables() {
		t.Run(each.name, func(t *testing.T) {
			disable, canBeDisabled := each.widget.(fyne.Disableable)
			if !canBeDisabled {
				t.Fatalf("%s cannot be disabled, so this guard is asking about a state it does "+
					"not have", each.name)
			}
			disable.Disable()
			off := lookOf(each.widget)

			pointerOn(each.widget)
			pressOn(each.widget)
			if touched := lookOf(each.widget); touched != off {
				t.Errorf("%s answered a pointer while disabled, so the screen offers something "+
					"nobody can use:\n  disabled %s\n  touched  %s", each.name, off, touched)
			}
		})
	}
}
