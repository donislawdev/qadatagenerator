// Package parts holds the controls this window uses in place of the toolkit's
// own, where the toolkit's own falls short of what a person expects.
//
// It is a short list on purpose, and it is not a widget set. Anything here has
// to name what the toolkit does, what a person expects instead, and where that
// expectation was measured.
package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Button is the toolkit's button, pressed by Enter as well as by the space bar.
//
// # What the toolkit does, and why that is not enough here
//
// widget.Button answers to fyne.KeySpace and to nothing else. Seen on a real
// window on 2026-08-18: with the keyboard on the button that opens a manifest,
// Enter did nothing at all, and the space bar opened the chooser. A person who
// works on this desktop presses Enter, gets silence, and has no way to tell a
// control that ignores the key from a program that stopped responding.
//
// Silence in place of an answer is the one thing this product refuses
// everywhere else, so it is refused here too.
//
// # Why an extension rather than a control of our own
//
// Everything a button does stays the toolkit's: the drawing, the focus ring, the
// hover, the importance, the icon. What is added is one key. That is the
// toolkit's own extension mechanism, so the rule against rebuilding pieces of it
// is not touched, and there is no renderer here to go stale when the toolkit
// moves.
type Button struct {
	widget.Button
}

// NewButton builds a button that answers to both keys.
func NewButton(label string, tapped func()) *Button {
	button := &Button{}
	button.Text = label
	button.OnTapped = tapped
	button.ExtendBaseWidget(button)
	return button
}

// TypedKey presses on Enter, and hands every other key to the toolkit.
//
// Both names are accepted because a keyboard has two of these keys and a person
// does not think of them as different: fyne.KeyReturn is the one above the right
// shift, fyne.KeyEnter is the one on the number pad.
func (b *Button) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	if event.Name == fyne.KeyReturn || event.Name == fyne.KeyEnter {
		// Disabled is asked here rather than left to the toolkit, because this
		// branch never reaches it. A control that cannot be pressed by the
		// pointer must not become pressable by the keyboard.
		if b.OnTapped != nil && !b.Disabled() {
			b.OnTapped()
		}
		return
	}
	b.Button.TypedKey(event)
}
