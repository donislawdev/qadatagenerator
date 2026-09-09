package parts

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The chooser began drawing its own face on 2026-09-09, and these are the
// questions that came with it.
//
// # Why they are here rather than beside the other choosers
//
// Two of them were asked over every state of every screen until that day, by
// the walk in internal/gui/screens that found the rectangle this package used
// to lay BESIDE the control: what colour the edge is drawn in and how thick it
// is. There is no such rectangle now, and the walk over screens cannot reach
// inside a renderer without becoming a second copy of one.
//
// So they moved to where the renderer can be reached, which is here, beside the
// same questions already asked of the button. What stayed over there is the
// structural one that only a whole screen can answer: whether the mark is on
// every control that should carry it, in every state.

// chooserFace is what the renderer of this chooser would put on the screen.
func chooserFace(chooser *Select) fyne.CanvasObject {
	return chooser.CreateRenderer().(*chooserRenderer).holder
}

// strokedIn is every rectangle in a drawing that paints a stroke.
//
// The whole list rather than the first, because a focused control draws two -
// its edge and its ring - and a question about either has to be able to tell
// them apart by colour rather than by which came first.
func strokedIn(object fyne.CanvasObject) []*canvas.Rectangle {
	switch shape := object.(type) {
	case *canvas.Rectangle:
		if shape.StrokeWidth > 0 {
			return []*canvas.Rectangle{shape}
		}
	case *fyne.Container:
		found := []*canvas.Rectangle{}
		for _, child := range shape.Objects {
			found = append(found, strokedIn(child)...)
		}
		return found
	}
	return nil
}

// TestTheChooserIsOutlinedFromTheTokens holds the colour and the thickness the
// walk over screens used to hold.
//
// The thickness is not a preference. Below two the toolkit draws the line
// between two rows of pixels and every pixel of it is a half blend, so an edge
// measured at 3.29 in the palette arrives on screen at 2.12.
func TestTheChooserIsOutlinedFromTheTokens(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
	chooser.SetSelected("pesel")

	strokes := strokedIn(chooserFace(chooser))
	if len(strokes) != 1 {
		t.Fatalf("a resting chooser draws %d strokes, where it draws one: its edge", len(strokes))
	}
	edge := strokes[0]
	if !sameColour(edge.StrokeColor, palette.Ink(palette.BorderControl)) {
		t.Errorf("the chooser is outlined %v, where the token says %v",
			edge.StrokeColor, palette.Ink(palette.BorderControl))
	}
	if edge.StrokeWidth != palette.BorderThickness {
		t.Errorf("the chooser is outlined %v thick, where the palette says %v",
			edge.StrokeWidth, palette.BorderThickness)
	}
}

// TestTheEdgeOfAFaceIsDrawnInsideTheControl is the rule TestTheEdgeIsDrawn
// InsideTheControlItOutlines holds for parts.Outlined, asked of the drawing
// that replaced it.
//
// # 🔴 Why it had to be asked again, and how the fault was found
//
// The toolkit centres a stroke on the path it is given, so a rectangle at the
// full size of a control leaves half its edge outside, drawn over whatever the
// control stands on and arriving as a blend of two colours rather than as the
// one asked for. parts.Outlined knew this and inset its rectangle. The face
// that replaced it did not, for as long as it took to render one screen.
//
// Neither a test nor a reading found it. What found it was the chooser standing
// in one row beside a text field the toolkit still draws, and the two being
// compared in the stored tree: the toolkit puts the border of a field at 1,1
// and one thickness in from each edge, ours sat at 0,0 and the full size. Two
// controls in one row with two different edges is the fault DEC-215 settled,
// and it is invisible until something is put next to it.
func TestTheEdgeOfAFaceIsDrawnInsideTheControl(t *testing.T) {
	size := fyne.NewSize(220, palette.ControlHeight)
	faces := map[string]fyne.CanvasObject{
		"the chooser": chooserFace(func() *Select {
			chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
			chooser.SetSelected("pesel")
			return chooser
		}()),
		"the quiet button": drawnFace(NewButton("Choose a file", func() {})),
	}

	for name, face := range faces {
		t.Run(name, func(t *testing.T) {
			face.Resize(size)
			strokes := strokedIn(face)
			if len(strokes) != 1 {
				t.Fatalf("%s draws %d strokes at rest, where it draws one: its edge", name, len(strokes))
			}
			edge := strokes[0]

			wantAt := fyne.NewPos(palette.OutlineInset, palette.OutlineInset)
			if edge.Position() != wantAt {
				t.Errorf("%s draws its edge at %v, where half of its own stroke is %v. "+
					"Anywhere else and half the line falls outside the control, onto whatever "+
					"it stands on", name, edge.Position(), wantAt)
			}
			wantSize := fyne.NewSize(size.Width-palette.BorderThickness, size.Height-palette.BorderThickness)
			if edge.Size() != wantSize {
				t.Errorf("%s draws its edge at %v rather than %v, so the far side of the "+
					"stroke lands somewhere other than the near side does",
					name, edge.Size(), wantSize)
			}
		})
	}
}

// TestADisabledChooserShowsNoEdge is the button's rule, asked of the control
// beside it.
//
// A chooser has no notion of importance, so this is its only quiet state. The
// ground stays either way, because a chooser nobody can touch still has to show
// where it is.
func TestADisabledChooserShowsNoEdge(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
	chooser.SetSelected("pesel")
	if len(strokedIn(chooserFace(chooser))) == 0 {
		t.Fatal("a fresh chooser draws no edge, so the rest of this proves nothing")
	}

	chooser.Disable()
	if len(strokedIn(chooserFace(chooser))) != 0 {
		t.Error("a disabled chooser still draws the edge that says it can be used")
	}

	chooser.Enable()
	if len(strokedIn(chooserFace(chooser))) == 0 {
		t.Error("a chooser that became usable again did not get its edge back")
	}
}

// TestTheKeyboardOnTheChooserIsVisible asks the question the toolkit answers by
// flooding the control.
//
// 🔴 It is asked as a DIFFERENCE rather than as a colour, for the reason the
// focus ring of the primary button records: a mark painted in the colour of
// what is under it changes pixels and says nothing, and a guard reading only
// the colour passes on one that cannot be seen.
func TestTheKeyboardOnTheChooserIsVisible(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
	chooser.SetSelected("pesel")

	resting := strokedIn(chooserFace(chooser))
	if len(resting) == 0 {
		t.Fatal("a resting chooser draws no edge at all, so there is nothing for focus to change")
	}
	chooser.FocusGained()
	holding := strokedIn(chooserFace(chooser))

	// 🔴 The mark is the EDGE of the control rather than a second shape inside
	// it, changed on 2026-09-09. So the guard asks whether the drawing CHANGED
	// and what it changed to, not how many strokes there are: counting shapes
	// was the right question when focus added one and is the wrong one now,
	// and it would have gone on passing while the colour drifted anywhere.
	if len(holding) != len(resting) {
		t.Fatalf("focus changed the number of strokes from %d to %d, where it is meant to "+
			"recolour the edge the chooser already has", len(resting), len(holding))
	}

	marked := false
	for _, stroke := range holding {
		if sameColour(stroke.StrokeColor, palette.Ink(palette.MarkFocus)) {
			marked = true
		}
	}
	if !marked {
		t.Errorf("a chooser with the keyboard on it draws no edge in mark.focus %v, so nothing "+
			"on screen says where the keyboard is", palette.Ink(palette.MarkFocus))
	}
	for _, stroke := range resting {
		if sameColour(stroke.StrokeColor, palette.Ink(palette.MarkFocus)) {
			t.Error("a chooser nobody is on already draws the focus edge, so the mark says nothing")
		}
	}

	chooser.FocusLost()
	for _, stroke := range strokedIn(chooserFace(chooser)) {
		if sameColour(stroke.StrokeColor, palette.Ink(palette.MarkFocus)) {
			t.Error("the mark stayed after the keyboard left")
		}
	}
}

// TestTheChooserDoesNotChangeWidthWhenSomethingIsChosen is the whole reason
// paint.ChooserWidth exists.
//
// 🔴 A control as wide as what it currently says is a control that resizes
// under the hand using it, and a row of four settings would shuffle sideways
// every time one of them was answered. The toolkit avoids this by an accident -
// selectRenderer.MinSize throws the width of the value away and uses the width
// of the placeholder - and its own renderer is gone, so nothing would avoid it
// here without being asked to.
//
// The options are deliberately far apart in width. Two of a similar length
// would pass on a control that had no such rule at all.
func TestTheChooserDoesNotChangeWidthWhenSomethingIsChosen(t *testing.T) {
	chooser := NewSelect([]string{"pl", "a considerably longer option than that"}, func(string) {})

	chooser.SetSelected("pl")
	narrow := chooser.MinSize().Width
	chooser.SetSelected("a considerably longer option than that")
	wide := chooser.MinSize().Width

	if narrow != wide {
		t.Errorf("the chooser asks for %v holding a short value and %v holding a long one, "+
			"so the row it stands in moves when somebody chooses", narrow, wide)
	}
}

// TestTheChooserOpensOnEnterAsWellAsOnSpace holds the key the toolkit ignores.
//
// # Why the pop up is looked for on the canvas
//
// Because that is the only place it can be seen from outside the toolkit: the
// menu is kept in a private field of widget.Select and put on screen as an
// overlay of the canvas the chooser is on. An overlay is public, so the
// question "did the list open" has a public answer, and it does not depend on
// anything about how the list is built.
func TestTheChooserOpensOnEnterAsWellAsOnSpace(t *testing.T) {
	for _, key := range []fyne.KeyName{fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace} {
		t.Run(string(key), func(t *testing.T) {
			chooser := NewSelect([]string{"pesel", "nip"}, func(string) {})
			chooser.SetSelected("pesel")
			window := test.NewWindow(chooser)
			defer window.Close()

			if window.Canvas().Overlays().Top() != nil {
				t.Fatal("something was already over the window, so this proves nothing")
			}
			chooser.TypedKey(&fyne.KeyEvent{Name: key})
			if window.Canvas().Overlays().Top() == nil {
				t.Errorf("%s left the chooser closed, so a person who pressed it got silence "+
					"and no way to tell that from a program that had stopped answering", key)
			}
		})
	}
}

// TestTheChooserStillWalksItsOptionsWithTheArrows holds what the key above must
// not have cost.
//
// TypedKey is overridden to add one key, and the way to add one key wrongly is
// to answer that one and drop the rest. The toolkit's chooser walks its options
// with Left and Right, and that is behaviour this project did not write and
// must not quietly take away.
func TestTheChooserStillWalksItsOptionsWithTheArrows(t *testing.T) {
	chooser := NewSelect([]string{"pesel", "nip", "regon"}, func(string) {})
	chooser.SetSelected("pesel")

	chooser.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if chooser.Selected != "nip" {
		t.Errorf("the right arrow moved the chooser to %q, where the next option is %q",
			chooser.Selected, "nip")
	}
	chooser.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if chooser.Selected != "pesel" {
		t.Errorf("the left arrow moved the chooser to %q, where the previous option is %q",
			chooser.Selected, "pesel")
	}
}
