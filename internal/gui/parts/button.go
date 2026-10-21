// Package parts holds the controls this window uses in place of the toolkit's
// own, where the toolkit's own falls short of what a person expects.
//
// It is a short list on purpose, and it is not a widget set. Anything here has
// to name what the toolkit does, what a person expects instead, and where that
// expectation was measured.
package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
)

// Button is a button this window draws itself.
//
// # 🔴 Why it stopped being an extension of the toolkit's, on 2026-09-09
//
// It used to be widget.Button with one key added and a rectangle laid over it,
// and that was the right shape while the only complaint was the Enter key. The
// owner put the accepted drawing beside a render of the window and the answer
// was plain: the mock built its buttons out of paint.ButtonFace and the window
// built them out of the toolkit's, so the two were two drawings of one thing
// and free to look nothing alike. They did.
//
// The comment at the top of paint/control.go had promised they could not drift,
// "because there is one of them". There was one of them and the window was not
// using it. A promise about a shared drawing is worth nothing until both sides
// draw from it, which is the same family as a colour added to a register that
// no line of code reads.
//
// # What this costs, said plainly
//
// A renderer of our own is a piece of the toolkit written again, and this
// project keeps a register of those with a ceiling on it, set so that adding
// one is a decision somebody made rather than a habit. The owner made that
// decision on 2026-09-09, having seen what the alternative looks like. The
// entry in HandmadeAllowed says what was considered and why it does not fit,
// which is what the register is for.
//
// # What is gained beyond the look, and none of it is cosmetic
//
//   - The gap between an icon and its label. The toolkit takes it from
//     SizeNamePadding, which this window answers with nought on purpose, so an
//     icon on a stock button TOUCHES its label. Drawing the row ourselves takes
//     that gap from the scale like every other gap here.
//   - The focus mark. The toolkit floods a focused button with the focus
//     colour, measured at 1.23 for the mark and 1.56 for the label on it, which
//     is the whole reason parts.Flooded existed until DEC-253 deleted it with
//     nothing left to flood. A ring we draw needs no such repair and the ink of
//     the label never has to change.
//   - The disabled state. One branch of one face, instead of an importance, a
//     themed colour and a rectangle somebody has to remember to hide.
type Button struct {
	widget.DisableableWidget

	// Text is the label and OnTapped what pressing it does. Both exported to
	// match the toolkit's own button, so a caller moving between them does not
	// have to learn a second spelling.
	Text     string
	OnTapped func()

	// Icon is the NAME of one of this window's icons, empty for none.
	//
	// A name and not a resource, because the ink of an icon depends on the
	// state the button is in, and a resource is a colour already decided.
	Icon string

	// Leading puts the label against the leading edge, which a column of places
	// to go needs and a row of actions does not. Plain drops the edge, for a
	// button that is a place rather than a slab.
	Leading bool
	Plain   bool

	// Current marks a plain button as the place the window is on, so the face
	// draws the tinted ground under it in every state, the pointer's included.
	// The sidebar sets it, and the reason it lives here rather than under the
	// button is beside paint.ButtonLook.
	Current bool

	// Filled marks the one action a screen is for, drawn as a slab in the
	// accent rather than as another outlined word among several. NewPrimaryButton
	// sets it, and a screen has exactly one.
	Filled bool

	// Confirmed is what the button says for a while after Confirm, with a tick
	// in place of its icon: the word for something done that a person cannot
	// see, such as a clipboard filled. Empty for a button whose effect shows on
	// the screen. The button is as wide as the longer of its two words, so
	// nothing beside it moves when it changes.
	Confirmed string

	// confirm is whether Confirmed is what it says now, and the count back.
	confirm confirmation

	// keyboardIsHere is kept here rather than read from the canvas, because the
	// canvas is wrong about it at exactly the moment it changes: a control
	// redraws from inside FocusLost, before the focus manager has moved its own
	// record.
	keyboardIsHere bool

	// pointerIsHere and pointerIsDown are the pointer's half of the same
	// question, and they arrived on 2026-09-09 with the states of the face.
	//
	// 🔴 They are not an addition. The toolkit's own button has carried a
	// hovered flag since always and blends ColorNameHover with it, so taking
	// its drawing away on 2026-09-09 took a state a person could see and put
	// nothing in its place. This widget does not even embed widget.Button any
	// more, so the driver had nothing to call.
	pointerIsHere bool
	pointerIsDown bool
}

// Interfaces are asserted here rather than discovered at a call site, because a
// widget that quietly stops being focusable still compiles and still draws, and
// the only symptom is a person with a keyboard never reaching it.
//
// 🔴 The last two are what makes a pointer reach this control at all. The
// driver asks for them by type - internal/driver/glfw/window.go:478 and :494 -
// so a control that stops implementing one goes on compiling, goes on drawing,
// and simply stops answering the mouse.
var (
	_ fyne.Widget       = (*Button)(nil)
	_ fyne.Tappable     = (*Button)(nil)
	_ fyne.Focusable    = (*Button)(nil)
	_ desktop.Hoverable = (*Button)(nil)
	_ desktop.Mouseable = (*Button)(nil)
)

// NewButton builds a button that answers to both keys.
func NewButton(label string, tapped func()) *Button {
	button := &Button{Text: label, OnTapped: tapped}
	button.ExtendBaseWidget(button)
	return button
}

// NewPrimaryButton builds the one action a screen is for: the same button,
// filled.
//
// # Why the toolkit could not be asked for this
//
// widget.HighImportance is the toolkit's own filled button and it was measured
// and turned down on 2026-09-05. A focused one is blend(primary, focus), and the
// blend at widget/button.go:453 collapses an opaque overlay to the overlay
// itself, so the button goes from its resting fill to the flat focus colour: a
// focus mark of 1.23 where WCAG 1.4.11 asks for 3.0.
//
// # 🔴 Why this is a flag on Button and no longer a control of its own
//
// It was one until 2026-09-24, with a renderer keeping three shapes and a label
// in bold, while the accepted drawing of every screen built the same action out
// of paint.ButtonFace with Filled set. So the mock and the window drew the one
// button a screen is for twice, differently: a bold label and widths of its own
// in the window, the face's label and widths in the drawing the owner accepted.
// That is the drift Button stopped drawing through the toolkit to end, back in
// one control. The copy also carried every key and pointer handler of Button a
// second time. Now the look is one branch of one face, and a change to how
// buttons look is a change in one place.
//
// What the fill is, the ink on it and why that ink is white rather than the
// ink every other label uses are measured beside paint.FilledGround.
func NewPrimaryButton(label string, tapped func()) *Button {
	button := NewButton(label, tapped)
	button.Filled = true
	return button
}

// OnScreen is what a layout is given.
//
// 🔴 It hands back the button itself, and it stays because twenty two call
// sites say it. Until this control drew itself a layout had to be given a
// container instead: the toolkit resolved the colour of a flooded label through
// the theme of the scope that label was in, so there had to BE a scope. Nothing
// resolves through a scope any more, so the wrapper went and the method did
// not, since removing it would be a rename of twenty two lines saying nothing
// new.
func (b *Button) OnScreen() fyne.CanvasObject { return b }

// SetText changes the label and redraws.
func (b *Button) SetText(label string) {
	b.Text = label
	b.Refresh()
}

// ShowIcon puts one of this window's icons on the button.
func (b *Button) ShowIcon(name string) {
	b.Icon = name
	b.Refresh()
}

// Tapped presses the button, unless it cannot be pressed.
// Confirm makes the button say Confirmed, with a tick, for
// two palette.ConfirmationLeg, and then its own word again. See confirmation.
func (b *Button) Confirm() {
	if b.Confirmed != "" {
		b.confirm.start(b.Refresh)
	}
}

// Settle makes the button say its own word again at once.
func (b *Button) Settle() { b.confirm.end(b.Refresh) }

// Confirming reports whether the button says Confirmed now, for a test that
// asks the button rather than its picture.
func (b *Button) Confirming() bool { return b.confirm.on }

func (b *Button) Tapped(*fyne.PointEvent) {
	if b.Disabled() || b.OnTapped == nil {
		return
	}
	b.OnTapped()
}

// MouseIn notes that the pointer arrived and redraws.
func (b *Button) MouseIn(*desktop.MouseEvent) {
	b.pointerIsHere = true
	b.Refresh()
}

// MouseMoved is required by the interface and has nothing to do: this control
// looks the same wherever inside itself the pointer is.
func (b *Button) MouseMoved(*desktop.MouseEvent) {}

// MouseOut notes that the pointer left, and forgets any press with it.
//
// # 🔴 Why the press is forgotten here, which is not obvious and is measured
//
// Because MouseUp does not necessarily come back. The driver finds the object
// under the pointer AT THE MOMENT THE BUTTON IS RELEASED and sends the release
// to that one - internal/driver/glfw/window.go:475 and :583 - so pressing this
// button, sliding off it and letting go delivers MouseUp to whatever is under
// the pointer by then, or to nothing at all.
//
// Without this line that leaves the button drawn as pressed for the rest of its
// life, with no way back short of pressing it again. Nothing would fail and no
// test reading the tree would see it, because the tree would be a legal drawing
// of a state the widget genuinely believes it is in.
func (b *Button) MouseOut() {
	b.pointerIsHere, b.pointerIsDown = false, false
	b.Refresh()
}

// MouseDown notes that the pointer is holding this button down.
//
// The press is recorded even on a disabled button, and the face throws it away:
// the state of the pointer is a fact about the pointer, while whether it means
// anything is a question about the control. Keeping the two apart is what stops
// a button that is enabled mid press from being wrong about itself.
func (b *Button) MouseDown(*desktop.MouseEvent) {
	b.pointerIsDown = true
	b.Refresh()
}

// MouseUp notes that the pointer let go.
func (b *Button) MouseUp(*desktop.MouseEvent) {
	b.pointerIsDown = false
	b.Refresh()
}

// FocusGained notes that the keyboard arrived and redraws.
func (b *Button) FocusGained() {
	b.keyboardIsHere = true
	b.Refresh()
}

// FocusLost notes that the keyboard left and redraws.
func (b *Button) FocusLost() {
	b.keyboardIsHere = false
	b.Refresh()
}

// TypedRune presses on the space bar, which is what the toolkit's own button
// answers to and therefore what a person arriving from any other control will
// try first.
func (b *Button) TypedRune(r rune) {
	if r == ' ' {
		b.Tapped(nil)
	}
}

// TypedKey presses on Enter as well as on the space bar.
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
// Both names of the key are accepted because a keyboard has two of them and
// nobody thinks of them as different: fyne.KeyReturn is above the right shift
// and fyne.KeyEnter is on the number pad.
func (b *Button) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	if event.Name == fyne.KeyReturn || event.Name == fyne.KeyEnter || event.Name == fyne.KeySpace {
		b.Tapped(nil)
	}
}

// CreateRenderer draws the face.
func (b *Button) CreateRenderer() fyne.WidgetRenderer {
	drawing := &buttonRenderer{button: b}
	drawing.Refresh()
	return drawing
}

// buttonRenderer draws whatever face the state of the button asks for.
//
// # Why the face is rebuilt rather than repainted
//
// A renderer usually keeps its shapes and changes their colours, which is what
// the filled button did with three of them until 2026-09-24. This one holds a face that
// is a small tree - a ground, maybe a ring, a row of an icon and some words -
// and the tree itself differs between states: a disabled button has no ring and
// no edge, and a button with no icon has no image in it.
//
// Keeping one tree and hiding parts of it would mean this file knowing the
// shape of the face, which is knowledge belonging on the other side of the
// boundary. Rebuilding costs a handful of objects on a state change a person
// caused, and it is what keeps the drawing in one place.
//
// 🔴 The face is swapped INSIDE a holder rather than replacing the field, and
// that distinction cost a render. The first version kept the face in a field
// and built a new one on every Refresh, which reads correctly and draws
// nothing: the toolkit takes Objects() when it builds the renderer and lays out
// what it took, so every later face was a tree nobody had been given. On screen
// the buttons lost their edge and their size and became bare words sitting
// wherever the last layout had put them. Nothing failed, and no test saw it -
// the tree of a button is behind its renderer, so the guards were reading the
// object the renderer HELD while the window drew the one it had been GIVEN.
type buttonRenderer struct {
	button *Button
	holder *fyne.Container
}

func (r *buttonRenderer) Layout(size fyne.Size) {
	r.holder.Resize(size)
	r.holder.Move(fyne.NewPos(0, 0))
}

func (r *buttonRenderer) MinSize() fyne.Size { return r.holder.MinSize() }

func (r *buttonRenderer) Refresh() {
	look := paint.ButtonLook{
		Label:    r.button.Text,
		Icon:     r.button.Icon,
		Leading:  r.button.Leading,
		Plain:    r.button.Plain,
		Current:  r.button.Current,
		Filled:   r.button.Filled,
		Focused:  r.button.keyboardIsHere,
		Disabled: r.button.Disabled(),
		Hovered:  r.button.pointerIsHere,
		Pressed:  r.button.pointerIsDown,
	}
	if r.button.Confirmed != "" {
		// As wide as the wider of the two words, measured on the face that
		// draws each, so the row beside it stays where it is when it changes.
		said := look
		said.Label, said.Icon = r.button.Confirmed, paint.IconCheck
		look.Width = max(paint.ButtonFace(look).MinSize().Width, paint.ButtonFace(said).MinSize().Width)
		if r.button.confirm.on {
			look.Label, look.Icon = said.Label, said.Icon
		}
	}
	face := paint.ButtonFace(look)
	if r.holder == nil {
		r.holder = container.NewStack(face)
		return
	}
	r.holder.Objects = []fyne.CanvasObject{face}
	r.holder.Refresh()
}

func (r *buttonRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.holder} }

func (r *buttonRenderer) Destroy() {}
