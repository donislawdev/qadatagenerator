// Package parts holds the controls this window uses in place of the toolkit's
// own, where the toolkit's own falls short of what a person expects.
//
// It is a short list on purpose, and it is not a widget set. Anything here has
// to name what the toolkit does, what a person expects instead, and where that
// expectation was measured.
package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
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

	// keyboardIsHere is kept here rather than read from the canvas, because
	// the canvas is wrong about it at exactly the moment it changes: a control
	// redraws from inside FocusLost, before the focus manager has moved its
	// own record.
	keyboardIsHere bool
	onScreen       fyne.CanvasObject

	// edge is the rectangle that says this is a control rather than a word.
	// It is held here rather than looked up in the tree, because whether it
	// shows depends on two things only the button knows, and both of them
	// change while the window is running.
	edge *canvas.Rectangle
}

// NewButton builds a button that answers to both keys.
func NewButton(label string, tapped func()) *Button {
	button := &Button{}
	button.Text = label
	button.OnTapped = tapped
	button.ExtendBaseWidget(button)
	dressed, edge := Outlined(Flooded(button, button.hasKeyboard))
	button.onScreen, button.edge = standing(dressed), edge
	button.markEdge()
	return button
}

// OnScreen is what a layout is given, rather than the button itself.
//
// The toolkit floods a focused control with the focus colour, so the label on
// it has to change colour to stay readable, and a colour is resolved through
// the theme of the scope a widget is in. Flooded above says why that scope can
// only be a container. This is where a layout is handed it.
//
// A layout given the button instead still compiles, because a button is a
// canvas object like any other. What refuses it is the guard that walks each
// screen and the contrast measured on the render, which is the right place for
// it: the mistake is invisible in the source and plain in the pixels.
//
// The edge Outlined draws is settled here as well as at build time, because a
// caller sets Importance between the two and the toolkit gives no hook for it.
func (b *Button) OnScreen() fyne.CanvasObject {
	b.markEdge()
	return b.onScreen
}

// Refresh settles the edge, then lets the toolkit redraw.
//
// # Why this is the place, and what it does not reach
//
// DisableableWidget.Disable and Enable at widget/widget.go:180 and 165 end in
// impl.Refresh, where impl is this type, so becoming pressable and stopping
// being pressable both arrive here. SetIcon and SetText do NOT: they call
// BaseWidget.Refresh, which goes straight to the renderer and never to this
// method. That is why the edge is also settled in NewButton and in OnScreen,
// and why the guard walking every screen is what actually holds the rule.
func (b *Button) Refresh() {
	b.markEdge()
	b.Button.Refresh()
}

// markEdge shows the edge on a button somebody can press, and hides it
// otherwise.
//
// # The two cases that get no edge, and why each
//
// A DISABLED button is not a control to find, and the edge is what says a
// control is there. Leaving it on would make the one thing a person cannot use
// the most visible thing on the screen. It also gives the disabled state a mark
// of its own, which until now it did not have: the label went quiet and nothing
// else changed.
//
// A LOW IMPORTANCE button is a heading rather than a slab, which is what the
// toolkit itself says by leaving the fill off one. There is one in this window,
// the heading of the list of generators, and it is as wide as its pane, so an
// edge around it would read as a box drawn round a section. What makes it
// findable is the arrow beside its words, which is a themed resource and takes
// the same colour they do. The owner chose this on 2026-09-05, with the number
// beside it: that heading measures 1.00 against the panel, which is the worst
// in the window, and the arrow is the answer to it.
func (b *Button) markEdge() {
	if b.edge == nil {
		return
	}
	if b.Disabled() || b.Importance == widget.LowImportance {
		b.edge.Hide()
		return
	}
	b.edge.Show()
}

// hasKeyboard says whether the keyboard is on this button.
func (b *Button) hasKeyboard() bool { return b.keyboardIsHere }

// FocusGained notes that the keyboard arrived, then lets the toolkit redraw.
//
// The order matters and is the whole point: the toolkit's own FocusGained ends
// in a Refresh, and that Refresh is what asks the theme for the colour of the
// label. Setting the flag afterwards would paint one frame with the old answer.
func (b *Button) FocusGained() {
	b.keyboardIsHere = true
	b.Button.FocusGained()
}

// FocusLost notes that the keyboard left, then lets the toolkit redraw.
func (b *Button) FocusLost() {
	b.keyboardIsHere = false
	b.Button.FocusLost()
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
