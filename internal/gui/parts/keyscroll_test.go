package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// reading is a column taller than any window these tests open.
func reading() fyne.CanvasObject {
	rows := make([]fyne.CanvasObject, 0, 60)
	for i := 0; i < 60; i++ {
		rows = append(rows, widget.NewLabel("a line of the thing being read"))
	}
	return container.NewVBox(rows...)
}

// onScreen puts a scrolling area in a window with something else beside it, so
// that focus has somewhere to leave to.
func onScreen(t *testing.T, content fyne.CanvasObject) (*KeyScroll, fyne.Canvas) {
	t.Helper()
	scroll := NewKeyScroll(content)
	elsewhere := NewButton("somewhere else", func() {})
	window := test.NewWindow(container.New(
		newTopFirst(), elsewhere, scroll.Framed()))
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(400, 300))
	return scroll, window.Canvas()
}

// newTopFirst lays a bar above a body, naming them in reading order.
//
// The toolkit's own border appends its edges after its middle, which this
// project has paid for twice, so these tests do not use it.
func newTopFirst() fyne.Layout { return &topFirst{} }

type topFirst struct{}

func (topFirst) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(objects[1].MinSize().Width, objects[0].MinSize().Height+objects[1].MinSize().Height)
}

func (topFirst) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	bar := objects[0].MinSize().Height
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(fyne.NewSize(size.Width, bar))
	objects[1].Move(fyne.NewPos(0, bar))
	objects[1].Resize(fyne.NewSize(size.Width, size.Height-bar))
}

// The keyboard reaches a scrolling area, which the toolkit's own never allows.
//
// 🔴 Measured on the About screen on 2026-08-24 before this existed: the whole
// window offered three places to stand, all of them the names of the screens,
// and 74% of that screen could not be reached without a mouse.
func TestTheKeyboardReachesAScrollingArea(t *testing.T) {
	scroll, canvas := onScreen(t, reading())

	for i := 1; i <= 4; i++ {
		canvas.FocusNext()
		if canvas.Focused() == fyne.Focusable(scroll) {
			return
		}
	}
	t.Fatalf("the keyboard never reached the scrolling area, it stands on %T", canvas.Focused())
}

// Every key a person expects to drive a scrolling area with drives it.
func TestTheSixKeysDriveAScrollingArea(t *testing.T) {
	scroll, canvas := onScreen(t, reading())
	canvas.Focus(scroll)
	if scroll.Content.Size().Height <= scroll.Size().Height {
		t.Fatal("the content fits the window, so this test would prove nothing about scrolling")
	}

	press := func(key fyne.KeyName) float32 {
		canvas.Focused().TypedKey(&fyne.KeyEvent{Name: key})
		return scroll.Offset.Y
	}

	if down := press(fyne.KeyDown); down <= 0 {
		t.Errorf("Down left the offset at %v, so an arrow does nothing", down)
	}
	afterDown := scroll.Offset.Y
	if up := press(fyne.KeyUp); up >= afterDown {
		t.Errorf("Up left the offset at %v after Down had it at %v", up, afterDown)
	}
	if page := press(fyne.KeyPageDown); page <= lineStep() {
		t.Errorf("PageDown moved to %v, which is no further than an arrow does", page)
	}
	afterPage := scroll.Offset.Y
	if back := press(fyne.KeyPageUp); back >= afterPage {
		t.Errorf("PageUp left the offset at %v after PageDown had it at %v", back, afterPage)
	}
	end := press(fyne.KeyEnd)
	if want := scroll.Content.Size().Height - scroll.Size().Height; end != want {
		t.Errorf("End reached %v rather than the last screenful at %v", end, want)
	}
	if home := press(fyne.KeyHome); home != 0 {
		t.Errorf("Home reached %v rather than the start", home)
	}
}

// A page keeps the line somebody was reading, rather than jumping past it.
//
// A page that moves by exactly its own height leaves nothing joining the two
// screenfuls, so a person reading prose loses their place at every jump.
func TestAPageKeepsTheLineSomebodyWasReading(t *testing.T) {
	scroll, canvas := onScreen(t, reading())
	canvas.Focus(scroll)
	canvas.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})

	if scroll.Offset.Y >= scroll.Size().Height {
		t.Errorf("a page moved %v with a window of %v, so nothing joins the two screenfuls",
			scroll.Offset.Y, scroll.Size().Height)
	}
}

// Content that fits the window has nowhere to go and stays there.
//
// The clamping is the toolkit's rather than ours, and this says so out loud: it
// is the state where a key does nothing, and a key that did something here
// would scroll a person off the only screenful there is.
func TestAScrollingAreaWithNothingToScrollStaysPut(t *testing.T) {
	scroll, canvas := onScreen(t, widget.NewLabel("one line"))
	canvas.Focus(scroll)

	for _, key := range []fyne.KeyName{fyne.KeyDown, fyne.KeyPageDown, fyne.KeyEnd} {
		canvas.Focused().TypedKey(&fyne.KeyEvent{Name: key})
		if scroll.Offset.Y != 0 {
			t.Fatalf("%v moved a scrolling area with nothing to scroll to %v", key, scroll.Offset.Y)
		}
	}
}

// The frame says when the scrolling area holds the keyboard, and stops saying
// it when the keyboard leaves.
//
// 🔴 It is asked of the frame rather than of the widget tree, for the reason
// this project measured on 2026-08-18: the tree names a colour rather than
// carrying its value, so a guard reading names is green with one colour on the
// screen.
func TestTheFrameSaysWhereTheKeyboardIs(t *testing.T) {
	scroll, canvas := onScreen(t, reading())
	quiet := scroll.ring.frame.StrokeColor

	canvas.Focus(scroll)
	marked := scroll.ring.frame.StrokeColor
	if marked == quiet {
		t.Fatal("the keyboard arrived and the frame is drawn exactly as it was, so nothing " +
			"on the screen says where a person is standing")
	}
	if marked != palette.Ink(palette.MarkFocus) {
		t.Errorf("the frame is drawn %v rather than in the one colour every control marks focus with", marked)
	}

	canvas.FocusNext()
	if canvas.Focused() == fyne.Focusable(scroll) {
		t.Fatal("focus did not leave, so this proves nothing about leaving")
	}
	if scroll.ring.frame.StrokeColor != quiet {
		t.Error("the keyboard left and the frame still says it is here")
	}
}

// The frame is thick enough to arrive on the screen.
//
// 🔴 Measured on 2026-08-24 on the outline of a text field: the toolkit draws a
// line offset by half its own width, so at a width of one it falls between two
// rows of pixels and every pixel of it is a half blend. The same value is used
// here rather than a fresh number, so the two cannot drift apart.
func TestTheFrameIsThickEnoughToBeSeen(t *testing.T) {
	scroll := NewKeyScroll(reading())
	if scroll.ring.frame.StrokeWidth != palette.BorderThickness {
		t.Errorf("the frame is %v thick rather than the measured %v",
			scroll.ring.frame.StrokeWidth, palette.BorderThickness)
	}
}
