package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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
	onScreen       *container.ThemeOverride
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
	chooser.onScreen = Flooded(chooser, chooser.hasKeyboard)
	return chooser
}

// OnScreen is what a layout is given, rather than the chooser itself. Button
// says why.
func (s *Select) OnScreen() fyne.CanvasObject { return s.onScreen }

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
