package parts

import (
	"fyne.io/fyne/v2"
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
//
// # 🔴 The frame is the ring every control draws, and it is only there if
// the screen took it
//
// Framed is the whole of the mark. A screen that puts the scrolling area on
// itself bare gets an area the keyboard reaches and nothing that shows it did:
// FocusGained recolours a rectangle nobody mounted, every test of this type
// passes, and the real window shows nought pixels of change. That was the
// Quick screen until 2026-09-14 (B-279), and the guard that walks a screen's
// stops counting rings on the SCREEN is what holds it now.
//
// The colour is mark.focus, the one shade every other control draws its ring
// in, since 2026-09-14. The ring itself is the shared one every control that
// draws its own focus wears, rounded like a card here because a scrolling area
// is the size of one and, on the screen where what scrolls IS a card, a
// control sized ring would land inside the card's own edge.
type KeyScroll struct {
	container.Scroll
	ring ring
}

// NewKeyScroll makes content scrollable and reachable without a mouse.
//
// Vertical only, because every scrolling area in this window is a column and a
// direction nobody uses is a direction nobody checks.
func NewKeyScroll(content fyne.CanvasObject) *KeyScroll {
	scroll := &KeyScroll{}
	scroll.setUp(content)
	scroll.ExtendBaseWidget(scroll)
	return scroll
}

// setUp is everything a scrolling area of this window is before the toolkit is
// told what object it is, so that a type built on this one, ScrolledRows, is
// the same area and then extends the base widget with itself.
func (s *KeyScroll) setUp(content fyne.CanvasObject) {
	s.ring = newRing(palette.RadiusCard)
	s.Direction = container.ScrollVerticalOnly
	s.Content = content
}

// Framed is this scrolling area together with the frame that says when it holds
// the keyboard. It is what a screen puts on itself, and the only thing that
// puts the mark on the screen at all: bare, the area is reached by the keyboard
// and shows nought pixels of change, which was the Quick screen until
// 2026-09-14 (B-279).
func (s *KeyScroll) Framed() fyne.CanvasObject { return s.ring.framed(s) }

// FocusGained draws the frame in the colour that means the keyboard is here.
func (s *KeyScroll) FocusGained() { s.ring.set(true) }

// FocusLost takes it away again.
func (s *KeyScroll) FocusLost() { s.ring.set(false) }

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
		s.to(fyne.Position{})
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
	s.to(fyne.NewPos(s.Offset.X, s.Offset.Y+delta))
}

// to moves the area and tells OnScrolled, which the toolkit's ScrollToOffset
// does not do.
//
// 🔴 Read on the source of v2.8.2 (internal/widget/scroller.go): OnScrolled is
// called by the wheel, by the bar and by an offset the toolkit had to clamp, and
// ScrollToOffset sets the offset it was given before asking whether it moved, so
// a move the keys make is never reported. Content that builds only what is in
// view, ScrolledRows, would then stay where the keys left it, showing empty
// card. End needs nothing here, because ScrollToBottom goes through the path the
// wheel takes.
//
// A move the toolkit clamped is reported twice, once by it and once here, and
// whoever listens is asked to treat a second report of the same place as
// nothing.
func (s *KeyScroll) to(offset fyne.Position) {
	before := s.Offset
	s.ScrollToOffset(offset)
	if s.Offset != before && s.OnScrolled != nil {
		s.OnScrolled(s.Offset)
	}
}
