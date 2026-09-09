package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
)

// Select is a chooser this window draws itself, over the behaviour of the
// toolkit's.
//
// # 🔴 Why it stopped being the toolkit's drawing, on 2026-09-09
//
// The same finding that moved the button one commit earlier, and the same
// measurement: the accepted drawing built its choosers out of paint.ChooserFace
// and the window built them out of widget.Select wearing two patches, so there
// were two drawings of one control and nothing holding them together. Counted
// on the day: three uses of that face in the mock, NONE in the product.
//
// # What was tried first, and why the theme could not reach it
//
// Dressing the toolkit's chooser is cheaper and leaves no renderer to go stale,
// which is why it was the road taken for a month. Its ceiling was read on the
// source of v2.8.1 rather than guessed, and two of the four differences are
// beyond it:
//
//   - The edge. The toolkit draws none for a chooser, so this package laid a
//     rectangle BESIDE the control and showed or hid it by hand.
//   - The focus mark. selectRenderer.bgColor at widget/select.go:432 floods the
//     whole control with the focus colour, and the value written across it
//     measures 1.56 where WCAG 1.4.3 asks for 4.5. No colour repairs that,
//     which is what parts.Flooded exists to say, so this package overrode the
//     theme of a scope around every chooser to change the ink of the value.
//
// Both patches are gone with this renderer, along with the height floor and the
// scope they needed to live in. The register of pieces rebuilt here gains two
// entries and the tree loses three wrappers per chooser.
//
// # What stays the toolkit's, and it is the larger half
//
// Everything that is not the drawing: the pop up menu, where it opens, the keys
// that walk the options, the selection itself and the binding. None of that is
// reachable from outside the package - showPopUp is private and so is the pop
// up it keeps - and none of it is what the owner was looking at.
//
// 🔴 A renderer of our own does not disturb it, which was read rather than
// assumed: Tapped needs no renderer beyond an animation it skips when nil, and
// showPopUp asks the widget for its size and its canvas, never for its
// drawing.
type Select struct {
	widget.Select

	// keyboardIsHere is kept here rather than read from the canvas, because the
	// canvas is wrong about it at exactly the moment it changes: a control
	// redraws from inside FocusLost, before the focus manager has moved its own
	// record.
	keyboardIsHere bool
}

// Interfaces are asserted here for the reason Button asserts them: a widget
// that quietly stops being focusable still compiles and still draws, and the
// only symptom is a person with a keyboard never reaching it.
var (
	_ fyne.Widget    = (*Select)(nil)
	_ fyne.Tappable  = (*Select)(nil)
	_ fyne.Focusable = (*Select)(nil)
)

// NewSelect builds a chooser that draws itself.
//
// # 🔴 Why the toolkit's constructor is read from rather than copied
//
// Until 2026-09-08 this line was `&Select{Select: *widget.NewSelect(...)}`, and
// the comment above it said ExtendBaseWidget would point the widget at this
// type rather than at the one thrown away. That sentence was false, and it cost
// every chooser in the window its value.
//
// widget.NewSelect ends in s.ExtendBaseWidget(s), so the struct it returns
// already carries an impl pointing at ITSELF. Copying the struct copies that
// pointer. ExtendBaseWidget is `if w.super() != nil { return }`
// (widget/widget.go), so the call below was a no-op, and impl went on naming
// the discarded widget. BaseWidget.Resize then ends in
// cache.Renderer(impl).Layout(size) - the size lands on this widget, and the
// LAYOUT lands on the renderer of the one nobody draws. Measured: every child
// of the drawn renderer stays 0x0 while the widget itself reports 611x27, so
// the value is truncated to an ellipsis and drawn at -4,-4. 54 choosers across
// 22 screen states, on both screens that have one.
//
// 🔴 It matters more now than it did then, and in a way that would be harder to
// see. widget.Select.MinSize begins with s.ExtendBaseWidget(s) on the EMBEDDED
// widget, so a chooser that reached that line with no impl would answer with
// the toolkit's own size and the toolkit's own drawing, for the rest of its
// life. It cannot, because the line below runs first, and that is the whole
// reason this is a constructor rather than a struct literal callers write.
//
// The placeholder is READ from a throwaway chooser rather than spelled again
// here: the toolkit does not export that string, and a second copy of it would
// drift the first time the toolkit changed its wording.
func NewSelect(options []string, changed func(string)) *Select {
	chooser := &Select{}
	chooser.Options = options
	chooser.OnChanged = changed
	chooser.PlaceHolder = widget.NewSelect(nil, nil).PlaceHolder
	chooser.ExtendBaseWidget(chooser)
	return chooser
}

// OnScreen is what a layout is given.
//
// It hands back the chooser itself, and it stays for the reason the same method
// on Button stays: until this control drew itself a layout had to be given the
// scope the theme override lived in, and removing the method now would be a
// rename of every call site saying nothing new.
func (s *Select) OnScreen() fyne.CanvasObject { return s }

// FocusGained notes that the keyboard arrived, then lets the toolkit redraw.
//
// The flag is set before the toolkit's own method rather than after, because
// that method ends in the Refresh that builds the face.
func (s *Select) FocusGained() {
	s.keyboardIsHere = true
	s.Select.FocusGained()
}

// FocusLost notes that the keyboard left, then lets the toolkit redraw.
func (s *Select) FocusLost() {
	s.keyboardIsHere = false
	s.Select.FocusLost()
}

// TypedKey opens the list on Enter as well as on the keys the toolkit answers.
//
// # Why this is here at all
//
// widget.Select opens on Space, Up and Down, and does nothing at all on Enter.
// That is the fault the button was given a key for on 2026-08-18, arriving at
// the control next door: a person presses the key that means yes, gets silence,
// and cannot tell a control that ignores the key from a program that has
// stopped answering. Silence in place of an answer is the one thing this
// product refuses everywhere else.
//
// It is spelled as the key the toolkit already understands rather than as a
// call, because opening the list is showPopUp and that method is private. Both
// names of the key are accepted because a keyboard has two of them and nobody
// thinks of them as different.
func (s *Select) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}
	if event.Name == fyne.KeyReturn || event.Name == fyne.KeyEnter {
		s.Select.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		return
	}
	s.Select.TypedKey(event)
}

// CreateRenderer draws the face.
func (s *Select) CreateRenderer() fyne.WidgetRenderer {
	// The toolkit's own CreateRenderer begins by extending the embedded widget,
	// and this one does not, deliberately. Reaching here at all means the cache
	// asked THIS type for a renderer, which it only does when impl already
	// names it. Calling it would be a no-op on every path that exists and a way
	// to hide the missing constructor on any path that ever did not.
	drawing := &chooserRenderer{chooser: s}
	drawing.Refresh()
	return drawing
}

// hasKeyboard is what the face is told, and what a guard can ask.
func (s *Select) hasKeyboard() bool { return s.keyboardIsHere }

// chooserRenderer draws whatever face the state of the chooser asks for.
//
// 🔴 The face is swapped INSIDE a holder rather than replacing a field, for the
// reason written at length beside buttonRenderer: the toolkit takes Objects()
// when it builds the renderer and lays out what it took, so a field reassigned
// on every Refresh produces a tree nobody was given. Nothing fails and no test
// sees it, because a guard reads the object the renderer HOLDS while the window
// draws the one it was GIVEN. What Objects() hands back has to be one identity
// for the life of the renderer.
type chooserRenderer struct {
	chooser *Select
	holder  *fyne.Container
}

func (r *chooserRenderer) Layout(size fyne.Size) {
	r.holder.Resize(size)
	r.holder.Move(fyne.NewPos(0, 0))
}

func (r *chooserRenderer) MinSize() fyne.Size { return r.holder.MinSize() }

func (r *chooserRenderer) Refresh() {
	value, unchosen := r.chooser.Selected, false
	if value == "" {
		value, unchosen = r.chooser.PlaceHolder, true
	}
	face := paint.ChooserFace(paint.ChooserLook{
		Value: value,
		// 🔴 Wide enough for anything this chooser could be asked to show, not
		// for what it shows now. ChooserWidth says why, and the short of it is
		// that a row of settings that shuffles sideways when somebody chooses
		// something is worse than one that is wider than it needs to be. The
		// placeholder is in the reckoning because it is a value this control
		// can display.
		Width:    paint.ChooserWidth(append([]string{r.chooser.PlaceHolder}, r.chooser.Options...)...),
		Unchosen: unchosen,
		Focused:  r.chooser.hasKeyboard(),
		Disabled: r.chooser.Disabled(),
	})
	if r.holder == nil {
		r.holder = container.NewStack(face)
		return
	}
	r.holder.Objects = []fyne.CanvasObject{face}
	r.holder.Refresh()
}

func (r *chooserRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.holder} }

func (r *chooserRenderer) Destroy() {}
