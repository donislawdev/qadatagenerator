package parts

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
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
//     which is what parts.Flooded said until DEC-253 deleted it, so this
//     package overrode the theme of a scope around every chooser to change
//     the ink of the value.
//
// Both patches are gone with this renderer, along with the height floor and the
// scope they needed to live in. The register of pieces rebuilt here gains two
// entries and the tree loses three wrappers per chooser.
//
// # What stays the toolkit's, and what stopped being its on 2026-09-14
//
// The selection itself and the binding stay: Selected, SetSelected, Options,
// OnChanged and the placeholder are the toolkit's and are read as before.
//
// 🔴 The pop up does not. Until DEC-244 it was the toolkit's menu, opened by
// the toolkit's Tapped and its TypedKey, and the sentence above this one said
// it was the larger half and not what the owner was looking at. He looked at it
// on 2026-09-08 and asked for the panel he had been shown: grouped, with a
// search field, because there will be hundreds of kinds. The menu cannot hold
// either, and it is private, so it could not be extended. Tapped and TypedKey
// below open parts.picker in its place and never reach the toolkit's menu.
type Select struct {
	widget.Select

	// choices are the options with what the panel shows beside each and the
	// heading it stands under. Empty for a chooser built from bare strings,
	// which opens a flat panel of its Options.
	choices []Choice
	// words are what the panel says, in the language of the screen that
	// built this chooser.
	words PanelWords
	// panel and overlay are the open panel and the layer it stands in, nil
	// while the panel is closed.
	panel   *picker
	overlay fyne.CanvasObject

	// keyboardIsHere is kept here rather than read from the canvas, because the
	// canvas is wrong about it at exactly the moment it changes: a control
	// redraws from inside FocusLost, before the focus manager has moved its own
	// record.
	keyboardIsHere bool

	// pointerIsHere and pointerIsDown are the pointer's half of the same
	// question.
	//
	// 🔴 The first of them is a SECOND copy of something the embedded widget
	// already keeps, and it has to be, which is the whole finding here. The
	// toolkit's chooser has carried a hovered flag since always and colours
	// itself from it at widget/select.go:435, and it goes on setting that flag
	// under this renderer - MouseIn is inherited, so the events never stopped
	// arriving. The flag is unexported, so the drawing next door cannot read
	// it, and the redraw it triggers redraws the same picture.
	//
	// That is what taking the drawing away on 2026-09-09 actually did to this
	// control: not remove the hover, but leave it running with nothing able to
	// see it. Measured on the real binary, a window with the pointer on a
	// control was identical to one with the pointer in empty space.
	pointerIsHere bool
	pointerIsDown bool
}

// Interfaces are asserted here for the reason Button asserts them: a widget
// that quietly stops being focusable still compiles and still draws, and the
// only symptom is a person with a keyboard never reaching it.
var (
	_ fyne.Widget       = (*Select)(nil)
	_ fyne.Tappable     = (*Select)(nil)
	_ fyne.Focusable    = (*Select)(nil)
	_ desktop.Hoverable = (*Select)(nil)
	_ desktop.Mouseable = (*Select)(nil)
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

// MouseIn notes that the pointer arrived, then lets the toolkit note it too.
//
// The toolkit's own method is still called, and not because this drawing needs
// what it sets. A widget half told about its own state is a widget that answers
// one question correctly and the next one wrongly, and the flag it keeps is
// read by MinSize and by the pop up, neither of which this file owns.
func (s *Select) MouseIn(event *desktop.MouseEvent) {
	s.pointerIsHere = true
	s.Select.MouseIn(event)
}

// MouseMoved is required by the interface, and passes the event on for the
// reason MouseIn does.
func (s *Select) MouseMoved(event *desktop.MouseEvent) {
	s.Select.MouseMoved(event)
}

// MouseOut notes that the pointer left, and forgets any press with it.
//
// The press is dropped here for the reason written out at length on
// parts.Button: the driver delivers MouseUp to whatever is under the pointer
// when the button is released, so a press that slides off this control is never
// released on it and would otherwise be drawn for the rest of the run.
func (s *Select) MouseOut() {
	s.pointerIsHere, s.pointerIsDown = false, false
	s.Select.MouseOut()
}

// MouseDown notes that the pointer is holding this chooser down.
//
// 🔴 The toolkit's chooser is not Mouseable at all, so nothing is passed on
// here and nothing is being overridden. It answers a press through Tapped,
// which opens the list, and this state is what a person sees in the moment
// between the two.
func (s *Select) MouseDown(*desktop.MouseEvent) {
	s.pointerIsDown = true
	s.Refresh()
}

// MouseUp notes that the pointer let go.
func (s *Select) MouseUp(*desktop.MouseEvent) {
	s.pointerIsDown = false
	s.Refresh()
}

// TypedKey opens the panel on Enter as well as on the keys the toolkit answered.
//
// # Why Enter is here at all
//
// widget.Select opened on Space, Up and Down, and did nothing at all on Enter.
// That is the fault the button was given a key for on 2026-08-18, arriving at
// the control next door: a person presses the key that means yes, gets silence,
// and cannot tell a control that ignores the key from a program that has
// stopped answering. Silence in place of an answer is the one thing this
// product refuses everywhere else. Both names of the key are accepted because
// a keyboard has two of them and nobody thinks of them as different.
//
// 🔴 Only Left and Right reach the toolkit's TypedKey, and that is deliberate:
// they walk the options without opening anything, which is behaviour this
// project did not write and must not quietly take away. Its Space, Up and
// Down would open the toolkit's menu beside our panel, and the two would
// answer the same keystroke twice.
func (s *Select) TypedKey(event *fyne.KeyEvent) {
	if event == nil || s.Disabled() {
		return
	}
	switch event.Name {
	case fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace, fyne.KeyUp, fyne.KeyDown:
		s.open()
	case fyne.KeyLeft, fyne.KeyRight:
		s.Select.TypedKey(event)
	}
}

// Tapped opens the panel, in place of the toolkit's menu.
func (s *Select) Tapped(*fyne.PointEvent) {
	if s.Disabled() {
		return
	}
	s.open()
}

// SetChoices gives the chooser its options with what the panel shows beside
// each, and the words the panel says.
//
// The values become Options, so everything the toolkit does with Options -
// the selection, the placeholder, the binding - goes on working, and a chooser
// built from bare strings and one built from choices are one control.
func (s *Select) SetChoices(choices []Choice, words PanelWords) {
	s.choices = choices
	s.words = words
	options := make([]string, 0, len(choices))
	for _, choice := range choices {
		options = append(options, choice.Value)
	}
	s.Options = options
	s.Refresh()
}

// choicesToOffer is what the panel is built from: the choices when there are
// any, otherwise the bare options as a flat list.
func (s *Select) choicesToOffer() []Choice {
	if len(s.choices) > 0 {
		return s.choices
	}
	flat := make([]Choice, 0, len(s.Options))
	for _, option := range s.Options {
		flat = append(flat, Choice{Value: option})
	}
	return flat
}

// Open shows the panel over the canvas this chooser is on, and is the one
// thing a screen may ask of the panel directly, so that a picture of it open
// can be taken.
func (s *Select) Open() { s.open() }

// IsOpen says whether the panel is on screen, for a test that would otherwise
// have to read the overlays of the canvas.
func (s *Select) IsOpen() bool { return s.panel != nil }

// open puts the panel in an overlay under this chooser and gives it the
// keyboard.
//
// # 🔴 Why an overlay of our own rather than widget.PopUp
//
// PopUp draws a ground of its own under its content, in the overlay colour of
// the theme, with the toolkit's shadow and the toolkit's radius. Under a card
// that draws its own edge, its own radius and the one shadow this window
// allows, that would be a second ground showing at every corner and a second
// shadow under the first. The overlay stack takes any object, which is all
// PopUp does with it, read on the source of v2.8.1.
//
// The shield under the panel is a TapRow that draws nothing: a tap anywhere
// but on the panel reaches it and closes the panel, which is what PopUp does
// with its own dismiss.
func (s *Select) open() {
	if s.panel != nil {
		return
	}
	canvas := fyne.CurrentApp().Driver().CanvasForObject(s)
	if canvas == nil {
		return
	}
	s.panel = newPicker(s.choicesToOffer(), s.Selected, s.words, s.SetSelected, s.closePanel)
	s.panel.onShape = s.fit
	shield := NewTapRow(func(bool) fyne.CanvasObject { return paint.Clear() }, s.closePanel)
	layer := container.NewWithoutLayout(s.panel)
	s.overlay = container.NewStack(shield, layer)

	s.fit()
	canvas.Overlays().Add(s.overlay)
	s.overlay.Resize(canvas.Size())
	s.panel.reveal()
	canvas.Focus(s.panel.keyReceiver())
}

// fit sizes and places the open panel for the rows it holds now, under or
// over this chooser. Called when it opens and again whenever the filter
// changes what it holds, so the panel is always as tall as its rows and never
// taller than the ceiling.
func (s *Select) fit() {
	canvas := fyne.CurrentApp().Driver().CanvasForObject(s)
	if s.panel == nil || canvas == nil {
		return
	}
	at := fyne.CurrentApp().Driver().AbsolutePositionForObject(s)
	wanted := fyne.NewSize(s.panel.Width(s.Size().Width), s.panel.ContentHeight())
	position, size := place(canvas.Size(), at, s.Size(), wanted, s.panel.MinSize().Height)
	if fitted := s.panel.Fitted(size.Height); fitted < size.Height {
		// A panel opened above the chooser keeps its foot against it.
		if position.Y < at.Y {
			position.Y += size.Height - fitted
		}
		size.Height = fitted
	}
	s.panel.Resize(size)
	s.panel.Move(position)
}

// place is where the panel goes and how large it is, given the canvas, the
// chooser it opens under, how wide it wants to be, how tall its rows would
// like it, and the least height it can be.
//
// Under the chooser, as tall as its rows up to the ceiling - and kept inside
// the canvas rather than let run off it. A chooser near the right edge gets a
// panel slid left, a chooser near the foot gets one that opens ABOVE it when
// there is more room there, and a canvas smaller than the panel gets one the
// size of the canvas, which then scrolls. A pure function so that every one
// of those cases is a line in a test rather than a window somebody has to
// shrink by hand.
func place(room fyne.Size, at fyne.Position, chooser fyne.Size, wanted fyne.Size, least float32) (fyne.Position, fyne.Size) {
	gap := palette.StepOf(palette.NameStepHairline)
	// A gap on each side of the panel, so that it never touches an edge of
	// the canvas.
	margins := gap + gap
	width := min(wanted.Width, room.Width-margins)
	x := at.X
	if x+width+gap > room.Width {
		x = max(gap, room.Width-width-gap)
	}

	height := min(palette.PanelHeight, wanted.Height)
	below := room.Height - (at.Y + chooser.Height + gap) - gap
	above := at.Y - margins
	y := at.Y + chooser.Height + gap
	if below >= height || below >= above {
		height = max(least, min(height, below))
	} else {
		height = max(least, min(height, above))
		y = at.Y - gap - height
	}
	// A canvas with no room either side, which a test can make and a person
	// can too by shrinking the window: the panel is then its least height and
	// is slid up until it fits, rather than drawn off the bottom.
	y = max(gap, min(y, room.Height-height-gap))
	return fyne.NewPos(x, y), fyne.NewSize(width, height)
}

// closePanel takes the panel down. Focus goes back to whatever the canvas had
// under the overlay, which is this chooser, since an overlay has a focus of
// its own and never moved the one beneath it.
func (s *Select) closePanel() {
	if s.panel == nil {
		return
	}
	if canvas := fyne.CurrentApp().Driver().CanvasForObject(s); canvas != nil {
		canvas.Overlays().Remove(s.overlay)
	}
	s.panel, s.overlay = nil, nil
	s.Refresh()
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
		//
		// 🔴 Keeping the placeholder in was settled by looking, on 2026-09-09,
		// after the owner said the question was hard to judge from a
		// description. Three settings rows were rendered side by side: the
		// accepted drawing, this, and the same thing with the placeholder left
		// out. The last is the worst by a distance - every box hugs its own
		// value, so four settings come out at four widths with no room in any
		// of them, and the row reads as scraps rather than as a set.
		//
		// What that measured is worth writing down, because it is NOT what
		// this function claims to be for: the row reads well because every
		// chooser has SLACK beyond its value, and the accepted drawing has
		// slack in all three of its choosers at three different widths. Here
		// the slack arrives because "(Select one)" is longer than almost
		// everything these lists hold.
		//
		// So it works, and it works for a reason nobody chose. No screen in
		// this window ever shows that string - measured across all 49 stored
		// states, which is nought - so the width of four controls a person
		// does look at is set by a string nobody ever sees. What would make it
		// wrong: the toolkit rewording its placeholder, or a list of options
		// growing past it. B-261 carries the question of a slack of our own.
		Width:    paint.ChooserWidth(append([]string{r.chooser.PlaceHolder}, r.chooser.Options...)...),
		Unchosen: unchosen,
		Focused:  r.chooser.hasKeyboard(),
		Disabled: r.chooser.Disabled(),
		Hovered:  r.chooser.pointerIsHere,
		Pressed:  r.chooser.pointerIsDown,
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
