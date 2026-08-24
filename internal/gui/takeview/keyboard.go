package takeview

import (
	"fyne.io/fyne/v2"
)

// # Why this screen moves the keyboard at all
//
// A run changes which controls there are. The button that damages the file
// leaves the screen while the run is going on and the way to stop it takes its
// place, and every setting goes out of reach for the same length of time. The
// toolkit does not follow either move: a control that is disabled keeps the
// keyboard it had, and so does one taken out of the tree.
//
// Measured on 2026-08-24 on a real canvas, before any of this existed. Starting
// a run from the keyboard left the keyboard on a button that was no longer
// drawn, so nothing on the screen said where a person was standing. The way out
// was not where anybody would look for it either: the toolkit skips a disabled
// control before it compares it to the one holding focus, so it never finds its
// place in the walk and the next press of Tab answers with the FIRST control on
// the screen. A person who stopped a run and pressed Tab was thrown to the top
// of the form.
//
// Neither of those is an answer somebody can act on, and silence in place of an
// answer is what this product refuses everywhere else.
//
// # Why the keyboard is only ever moved off a control that has just gone
//
// Moving it at any other time would be taking it from where a person put it.
// The window has a sidebar and two other screens, and none of them are this
// screen's to move.
//
// Gone means one of two things, and both are asked rather than assumed. A
// control this screen took out of the tree is named, because there are exactly
// two of them and they are the two buttons of a run. A control that is merely
// disabled is recognised by asking it, which is what reaches the tick boxes for
// the kinds of damage: they are the children of a group, so this package never
// holds them and could not name them if it wanted to.

// handKeyboardOver moves the keyboard onto arrived when the control it is
// standing on has just gone, and reports whether it did.
//
// # Why the answer is worth keeping
//
// The end of a run owes a place to somebody who was working by keyboard and
// owes nothing to somebody working by mouse. It cannot tell them apart by
// looking, because the toolkit clears focus on every tap of a button: a run
// started with the mouse leaves the keyboard nowhere, and so does a Tab pressed
// in the moment between asking a run to stop and the run answering, when every
// control here is out of reach and there is nothing for Tab to find. Both look
// like nothing at all afterwards.
//
// So the start of the run answers the question while it can still be answered,
// and the end of it acts on that answer.
func (v *View) handKeyboardOver(arrived fyne.Focusable, gone ...fyne.Focusable) bool {
	canvas := v.canvas()
	if canvas == nil {
		// The screen is not on a canvas yet, so there is no keyboard to move.
		// That is the ordinary state of a screen assembled before its window,
		// which is how the stored states of it are built.
		return false
	}
	standing := canvas.Focused()
	if standing == nil || !departed(standing, gone) {
		return false
	}
	v.place(canvas, arrived)
	return true
}

// giveTheKeyboardBack puts the keyboard on arrived at the end of a run, for
// somebody who had it when the run began.
//
// Unlike the hand over above it acts on a keyboard standing nowhere, because
// during a run nowhere is a place only this screen can have put it. It still
// leaves alone a keyboard somebody has since moved somewhere usable, which
// during a run means the sidebar: those names stay in reach for the whole of
// one, so a person can walk away from a run they started and this must not walk
// them back.
func (v *View) giveTheKeyboardBack(arrived fyne.Focusable, gone ...fyne.Focusable) {
	if !v.hadKeyboard {
		return
	}
	v.hadKeyboard = false
	canvas := v.canvas()
	if canvas == nil {
		return
	}
	if standing := canvas.Focused(); standing != nil && !departed(standing, gone) {
		return
	}
	v.place(canvas, arrived)
}

// place puts the keyboard on arrived, or on the one control this screen always
// has when arrived cannot take it.
//
// The second half is here because the toolkit answers a focus it cannot carry
// out with silence rather than with a failure: focusing something disabled,
// hidden or absent from the tree returns having done nothing at all. Without
// asking afterwards, the keyboard would simply stay on the control that has
// gone, which is the failure this whole file exists to end.
func (v *View) place(canvas fyne.Canvas, arrived fyne.Focusable) {
	canvas.Focus(arrived)
	if canvas.Focused() != arrived {
		canvas.Focus(v.chooseFile)
	}
}

// departed reports whether the control the keyboard is standing on has gone,
// either because this screen took it off the screen or because it disabled it.
func departed(standing fyne.Focusable, gone []fyne.Focusable) bool {
	for _, control := range gone {
		if standing == control {
			return true
		}
	}
	// Asked rather than looked up in a list, which is what makes this reach the
	// tick boxes for the kinds of damage. Disabling the group disables each box
	// inside it, and those boxes belong to the toolkit.
	control, answers := standing.(fyne.Disableable)
	return answers && control.Disabled()
}

// canvas is where this screen is drawn, or nothing when it is not drawn yet.
//
// It is asked for through a control that is always in the tree. The two buttons
// of a run each leave it in turn, so a canvas looked up through one of those
// would be found or not found depending on what the screen was doing.
func (v *View) canvas() fyne.Canvas {
	running := fyne.CurrentApp()
	if running == nil || running.Driver() == nil {
		return nil
	}
	return running.Driver().CanvasForObject(v.chooseFile)
}
