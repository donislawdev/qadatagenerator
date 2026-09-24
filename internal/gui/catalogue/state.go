package catalogue

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
)

// state is one condition a control is shown in, a column of a page.
//
// The five are the states the rules for the window ask of every control, and a
// control is shown in the ones its drawing answers to. Which those are is not
// decided here: the guard beside this file draws each control before and after
// each state, and a state that changes nothing is refused as firmly as a state
// that changes something and is missing.
type state int

const (
	rest state = iota
	pointer
	pressed
	focus
	disabled
	stateCount
)

// stateNames are the words over each column.
var stateNames = [stateCount]i18n.Key{
	rest:     i18n.CatalogueStateRest,
	pointer:  i18n.CatalogueStatePointer,
	pressed:  i18n.CatalogueStatePressed,
	focus:    i18n.CatalogueStateFocus,
	disabled: i18n.CatalogueStateDisabled,
}

// apply puts a control into this state the way the driver would, and reports
// whether the control takes it at all.
//
// The interfaces are the ones the driver asks for by type, so a control that
// answers here is a control a person can put into the same state. A press is
// a pointer held down, so it arrives on top of the pointer, as it does in a
// window.
func (s state) apply(control any) bool {
	switch s {
	case rest:
		return true
	case pointer:
		hoverable, takes := control.(desktop.Hoverable)
		if takes {
			hoverable.MouseIn(&desktop.MouseEvent{})
		}
		return takes
	case pressed:
		mouseable, takes := control.(desktop.Mouseable)
		if !takes || !pointer.apply(control) {
			return false
		}
		mouseable.MouseDown(&desktop.MouseEvent{})
		return true
	case focus:
		focusable, takes := control.(fyne.Focusable)
		if takes {
			focusable.FocusGained()
		}
		return takes
	case disabled:
		disableable, takes := control.(fyne.Disableable)
		if takes {
			disableable.Disable()
		}
		return takes
	}
	return false
}

// before is the state this one is judged against: what the drawing looked like
// a moment before a person put the control into it. A press is judged against
// the pointer it arrived on top of, everything else against rest.
func (s state) before() state {
	if s == pressed {
		return pointer
	}
	return rest
}
