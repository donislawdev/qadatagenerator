package parts

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// KeyScroll is the toolkit's scrolling container, reachable by Tab and driven
// by the keys a person expects to drive one with.
//
// # What the toolkit does, and why that is not enough here
//
// container.Scroll answers to the mouse wheel and to nothing else. It has no
// FocusGained, no TypedKey and no TypedRune, so it never appears in the walk of
// the keyboard, and nothing in the toolkit scrolls it on any key.
//
// Measured on 2026-08-24 on the About screen, which is one long scrolling
// column: the whole window offered three places for the keyboard to stand and
// all three were the names of the screens down the left. The content stood
// 3169 pixels tall in a window showing 836 of them, and Down, PageDown, End and
// the space bar each left the offset at zero. **Seventy four percent of that
// screen could not be reached without a mouse**, which is WCAG 2.1.1 and rule
// 4.9 of docs/17 broken on three quarters of a screen.
//
// # Why an extension rather than a widget of our own
//
// Everything a scrolling container does stays the toolkit's: the layout, the
// bars, the wheel, the clamping at both ends. What is added is the keyboard.
// That is the toolkit's own extension mechanism, the same one the button next
// door uses, so the register of pieces this project rebuilt is not touched and
// there is no renderer here to go stale when the toolkit moves.
//
// # Why the frame is composed rather than painted
//
// A scrolling area has nothing that could show it holds the keyboard: it has no
// border of its own and no background. Painting one would mean a renderer, and
// a renderer would make this a piece of the toolkit rebuilt rather than
// extended, against a register that may only get smaller. So the frame is a
// rectangle beside it in a stack, and Framed is what a screen adds to itself.
type KeyScroll struct {
	container.Scroll
	frame *canvas.Rectangle
}

// NewKeyScroll makes content scrollable and reachable without a mouse.
//
// Vertical only, because every scrolling area in this window is a column and a
// direction nobody uses is a direction nobody checks.
func NewKeyScroll(content fyne.CanvasObject) *KeyScroll {
	// Nothing is painted inside the frame, only its outline. A filled
	// rectangle behind a scrolling area is a second background, and the day
	// somebody changes the window colour it becomes a visible rectangle
	// nobody put there on purpose.
	scroll := &KeyScroll{frame: canvas.NewRectangle(color.Transparent)}
	scroll.Direction = container.ScrollVerticalOnly
	scroll.Content = content
	// The same width the outline of a text field needed, and for the same
	// measured reason: at a width of one the toolkit draws a line between two
	// rows of pixels and every pixel of it is a half blend.
	scroll.frame.StrokeWidth = palette.BorderThickness
	scroll.frame.StrokeColor = color.Transparent
	scroll.ExtendBaseWidget(scroll)
	return scroll
}

// Framed is this scrolling area together with the frame that says when it holds
// the keyboard. It is what a screen puts on itself.
func (s *KeyScroll) Framed() fyne.CanvasObject {
	return container.NewStack(s.frame, s)
}

// FocusGained draws the frame in the colour that means the keyboard is here.
func (s *KeyScroll) FocusGained() {
	s.frame.StrokeColor = palette.Named(palette.NameFocus).NRGBA()
	s.frame.Refresh()
}

// FocusLost takes it away again.
func (s *KeyScroll) FocusLost() {
	s.frame.StrokeColor = color.Transparent
	s.frame.Refresh()
}

// TypedRune ignores text. A scrolling area is read rather than written into.
//
// The space bar is deliberately not a page down here. It is a page down in a
// browser and a press in every control this window carries, and one key meaning
// two things on one screen is worse than one key meaning nothing.
func (s *KeyScroll) TypedRune(rune) {}

// TypedKey drives the scrolling area with the six keys a person expects.
//
// Clamping at both ends is the toolkit's, measured rather than assumed: asking
// for an offset past the end of the content lands exactly on the last screenful,
// and asking for one above the start lands on zero. Content shorter than the
// window stays at zero whatever is pressed, so the state with nothing to scroll
// needs no arithmetic here.
func (s *KeyScroll) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	switch event.Name {
	case fyne.KeyDown:
		s.by(lineStep())
	case fyne.KeyUp:
		s.by(-lineStep())
	case fyne.KeyPageDown:
		s.by(s.page())
	case fyne.KeyPageUp:
		s.by(-s.page())
	case fyne.KeyHome:
		s.ScrollToTop()
	case fyne.KeyEnd:
		s.ScrollToBottom()
	}
}

// lineStep is how far one press of an arrow travels.
//
// The largest step of the scale rather than a number of its own, because that
// step is what separates the parts of a screen read separately, and it lands
// close to the height of one row of text.
func lineStep() float32 { return palette.StepOf(palette.NameStepMajor) }

// page is a screenful less one line, so the line somebody was reading is still
// there after the jump. A page that moves by exactly its own height leaves
// nothing to join the two screenfuls by.
func (s *KeyScroll) page() float32 {
	if height := s.Size().Height - lineStep(); height > 0 {
		return height
	}
	return s.Size().Height
}

func (s *KeyScroll) by(delta float32) {
	s.ScrollToOffset(fyne.NewPos(s.Offset.X, s.Offset.Y+delta))
}
