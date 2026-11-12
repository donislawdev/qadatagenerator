package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// TestNoScreenHoldsAControlTheToolkitFloods walks every state of every screen
// and refuses any control the toolkit paints over with the focus colour.
//
// # 🔴 Why this asks the opposite of what it asked until 2026-09-09
//
// It used to be TestEveryFloodedControlKeepsItsLabelReadable, and it checked
// that each such control sat inside a theme of its own that repaired the ink of
// its label. That was the right question while the window was built out of the
// toolkit's controls: the flood is 1.56 for the words written across it, no
// fill repairs it, and changing the ink was the only answer available.
//
// The chooser was the last control that needed it. Both controls of this window
// now draw their own face and their own focus ring, so nothing here is flooded
// and there is no ink to repair. Its own comment said what to do on that day
// and this is the half of it worth keeping: the question about the repair goes,
// the question about the DAMAGE stays.
//
// That is the stronger of the two, and it was always available. A control the
// toolkit floods can only be dressed when somebody remembers to hand a layout
// its OnScreen rather than the control, which is a thing to get right at every
// call site. Not having one on screen at all is a thing to get right once.
//
// # Why a bare toolkit control is the thing being refused
//
// A widget.Button or a widget.Select that is not one of ours is one nothing can
// dress, since the theme that repairs the label is given to a container built
// around the control and there is no container around a control the toolkit
// makes inside its own renderer. That is why the list of generators stopped
// being a widget.Accordion. One appearing here means the defect has come back
// by another door, and now it means something simpler as well: a screen went
// back to a control that does not look like the rest of the window.
func TestNoScreenHoldsAControlTheToolkitFloods(t *testing.T) {
	data := sampleWithSource(t)

	// 🔴 The canary, and it is what stops this guard passing by walking
	// nothing. The count it used to keep was of the controls it objected to,
	// which is now expected to be nought, so a broken walk and a clean window
	// would read alike. This counts the controls it walked PAST instead: our
	// own two, which every screen but About has several of. Nought of those
	// means the walk stopped reaching controls, whatever it reports.
	ours := 0
	for _, state := range All() {
		content := Assemble(i18n.English, data, state)
		walkTree(content, func(object fyne.CanvasObject) {
			switch object.(type) {
			case *parts.Button, *parts.Select:
				if object.Visible() {
					ours++
				}
			}

			name, flooded := floodedControl(object)
			if !flooded {
				return
			}
			// 🔴 A hidden control is skipped, and the reason is measured
			// rather than assumed. widget.NewForm builds a button to submit
			// and a button to cancel whether or not it was given either
			// action, and hides them when it was not. The form on the take
			// and corrupt screen has neither, so two toolkit buttons with no
			// label sit in every state of that screen. They paint nothing and
			// the toolkit's own focus walk passes over them, so the keyboard
			// cannot reach one.
			//
			// This is the only reason to skip anything here. A control that
			// is hidden in one state and shown in another is caught in the
			// state that shows it, and there are forty of them.
			if !object.Visible() {
				return
			}
			t.Errorf("%s: %s. The toolkit floods it with the focus colour when the keyboard "+
				"arrives, which measures 1.56 for the words written across it, and nothing "+
				"outside its own renderer can repair that. Use the one in parts",
				state.Name, name)
		})
	}
	if ours == 0 {
		t.Error("no state of any screen holds a control of ours, so this walk passed by " +
			"walking nothing rather than by finding a window built the way it should be")
	}
}

// floodedControl says whether the toolkit paints this control's whole
// background with the focus colour, and names it for a message.
//
// The list is the toolkit's rather than ours, read at widget/button.go:379 and
// widget/select.go:432 of v2.8.1. widget/menu_item.go:295 does the same and is
// left out because this window has no menu, which is checked by the branch for
// a bare toolkit control catching one if a menu ever arrives.
func floodedControl(object fyne.CanvasObject) (string, bool) {
	switch typed := object.(type) {
	// 🔴 Neither parts.Button nor parts.Select is here, and their absence is
	// the finding rather than an omission.
	//
	// This guard is about one thing the toolkit does: it FLOODS a focused
	// control with the focus colour, so whatever is written on that control
	// becomes unreadable at 1.56. That is a fact about controls the TOOLKIT
	// draws. Both of ours draw their own face and paint their own focus ring,
	// so neither is ever flooded, and listing one here would be reporting a
	// control for damage nobody does to it.
	//
	// parts.Select was listed until 2026-09-09 and came off on the day it began
	// drawing itself. It is the reason this guard now asks its question the
	// other way round.
	case *widget.Button:
		return fmt.Sprintf("a toolkit button %q, which nothing can dress", typed.Text), true
	case *widget.Select:
		return fmt.Sprintf("a toolkit chooser %q, which nothing can dress", typed.Selected), true
	}
	return "", false
}

func rgba(from interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) [4]uint32 {
	r, g, b, a := from.RGBA()
	return [4]uint32{r, g, b, a}
}

// walkTree visits an object and everything drawn inside it.
//
// A widget is opened through its renderer, because the children of a widget are
// the objects its renderer holds and there is no other way in from outside the
// toolkit.
func walkTree(object fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if object == nil {
		return
	}
	visit(object)
	switch typed := object.(type) {
	case *fyne.Container:
		for _, child := range typed.Objects {
			walkTree(child, visit)
		}
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(typed).Objects() {
			walkTree(child, visit)
		}
	}
}
