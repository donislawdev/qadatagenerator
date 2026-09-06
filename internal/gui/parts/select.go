package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// Select is the toolkit's chooser, dressed so its value stays readable while
// the keyboard is on it.
//
// # Why this exists beside Button
//
// For one reason, and it is the same reason: widget/select.go:432 answers a
// focused chooser with ColorNameFocus for its whole background, exactly as
// widget/button.go:379 does for a button. The value written across that
// background is drawn in the ordinary foreground colour, which measures 1.56
// on it where WCAG 1.4.3 asks for 4.5. Flooded says what that number is and
// why no colour of fill repairs it.
//
// This one was found by reading the toolkit rather than by looking, because
// the catalogue of screen states had no state with the keyboard on a chooser,
// so nothing ever photographed it. The state was added with this type.
//
// # Why an extension rather than a control of our own
//
// Everything the chooser does stays the toolkit's: the list, the pop up, the
// keys, the drawing. What is added is a flag saying where the keyboard is, and
// the theme that reads it. There is no renderer here to go stale.
type Select struct {
	widget.Select

	// keyboardIsHere is kept here for the reason it is kept on Button: the
	// canvas is wrong about focus at the moment it changes, because a control
	// redraws from inside FocusLost before the focus manager moves its record.
	keyboardIsHere bool
	onScreen       fyne.CanvasObject

	// edge is the rectangle that says this is a control rather than a word.
	//
	// 🔴 # Why a chooser needs one when its arrow already says what it is
	//
	// Measured on 2026-09-06: the fill of a chooser at rest arrives as #202023
	// against a panel of #242424, which is 1.05 where WCAG 1.4.11 asks for 3.0.
	// The arrow beside the value is plain enough at 13.26, and it says WHAT the
	// control is. It does not say WHERE it is, which is the other half of what
	// a person needs in order to click one.
	//
	// The argument that settled it is not that number, though. A chooser and a
	// text field sit in ONE form on two screens of this window, rows apart, and
	// only one of them has a box: the toolkit draws an outline for an entry at
	// widget/entry.go and none for a chooser. That difference is an accident of
	// which widget the toolkit happens to border, not a decision anybody made,
	// and it reads on screen as two kinds of control where there is one.
	//
	// The heading of a disclosure was deliberately left without an edge on
	// 2026-09-05 and does not settle this one: the reason there was that the
	// heading is as wide as its pane, so a box around it reads as a box around
	// a section. A chooser is compact and stands in a form beside bordered
	// fields.
	edge *canvas.Rectangle
}

// NewSelect builds a chooser whose value can be read while the keyboard is on
// it.
//
// The toolkit's own constructor is called and its result copied, rather than
// the fields being set here one by one. It sets a placeholder from a string
// the toolkit does not export, and a copy of that string here would be a
// second source that drifts the first time the toolkit changes its wording.
// The copy is of a widget that has never been drawn, so there is nothing in it
// yet but the fields that constructor set, and ExtendBaseWidget below points
// the widget at this type rather than at the one thrown away.
func NewSelect(options []string, changed func(string)) *Select {
	chooser := &Select{Select: *widget.NewSelect(options, changed)}
	chooser.ExtendBaseWidget(chooser)
	chooser.onScreen, chooser.edge = Outlined(Flooded(chooser, chooser.hasKeyboard))
	chooser.markEdge()
	return chooser
}

// OnScreen is what a layout is given, rather than the chooser itself. Button
// says why.
func (s *Select) OnScreen() fyne.CanvasObject {
	s.markEdge()
	return s.onScreen
}

// Refresh settles the edge, then lets the toolkit redraw. Button says why this
// is the place and what it does not reach.
func (s *Select) Refresh() {
	s.markEdge()
	s.Select.Refresh()
}

// markEdge shows the edge on a chooser somebody can use, and hides it
// otherwise.
//
// Only the disabled case, because a chooser has no notion of importance. An
// edge says a control can be used, so leaving it on a disabled one would make
// the thing nobody can touch the most visible thing on the screen, and would
// take away the mark the disabled state gained when the button got one.
func (s *Select) markEdge() {
	if s.edge == nil {
		return
	}
	if s.Disabled() {
		s.edge.Hide()
		return
	}
	s.edge.Show()
}

func (s *Select) hasKeyboard() bool { return s.keyboardIsHere }

// FocusGained notes that the keyboard arrived, then lets the toolkit redraw.
//
// The flag is set first because the toolkit's own FocusGained ends in the
// Refresh that asks the theme for the colour of the value.
func (s *Select) FocusGained() {
	s.keyboardIsHere = true
	s.Select.FocusGained()
}

// FocusLost notes that the keyboard left, then lets the toolkit redraw.
func (s *Select) FocusLost() {
	s.keyboardIsHere = false
	s.Select.FocusLost()
}
