package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	toolkit "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryFloodedControlKeepsItsLabelReadable walks every state of every
// screen and asks each control the toolkit floods with the focus colour what
// its label will be painted with while the keyboard is on it.
//
// # Why this is a walk of the tree rather than a number off the render
//
// The render measures pixels, which is the stronger evidence, but it only
// measures what it can find: on 2026-09-05 it reported 1015 text nodes
// measured and 1554 it could not measure, and the value written across a
// chooser was among the second group. A control nothing photographs is a
// control nothing guards, so absence from the failure list of that probe is
// not a pass. This walk has no such gap, because it asks the control rather
// than the picture.
//
// # Why it also refuses a bare toolkit control
//
// A widget.Button or a widget.Select that is not one of ours is one nothing can
// dress, since the theme that repairs the label is given to a container built
// around the control and there is no container around a control the toolkit
// makes inside its own renderer. That is why the list of generators stopped
// being a widget.Accordion. A bare one appearing here again means the same
// defect has come back by another door.
func TestEveryFloodedControlKeepsItsLabelReadable(t *testing.T) {
	data := sampleWithSource(t)
	onFocus := palette.Named(palette.NameTextOnFocus).NRGBA()

	for _, state := range All() {
		content := Assemble(i18n.English, data, state)
		found := 0
		walkTree(content, false, func(object fyne.CanvasObject, dressed bool) {
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
			found++
			// 🔴 THE STRUCTURAL QUESTION, and it is the one that catches a
			// mistake. The colour question below cannot: measured on
			// 2026-09-05, a button handed to a layout WITHOUT OnScreen still
			// paints its label correctly, because NewThemeOverride registers
			// the whole subtree the moment it is built and cache.Renderer
			// makes the renderer to reach it. So the colour is right by a side
			// effect of construction, whatever ends up on screen.
			//
			// We do not build on that. What the toolkit promises is a
			// container whose children are themed, and a container that is not
			// on screen also never pushes its theme while the window lays
			// itself out. Leaning on the side effect would be leaning on
			// something nobody wrote down and nobody has to keep true.
			if !dressed {
				t.Errorf("%s: %s stands on screen outside any theme of its own. "+
					"A layout was given the control rather than its OnScreen",
					state.Name, name)
			}

			focusable, takesKeyboard := object.(fyne.Focusable)
			if !takesKeyboard {
				t.Errorf("%s: %s is flooded with the focus colour and cannot take the keyboard",
					state.Name, name)
				return
			}

			resting := labelColour(object)
			focusable.FocusGained()
			holding := labelColour(object)
			focusable.FocusLost()

			if holding != rgba(onFocus) {
				t.Errorf("%s: %s paints its label %v while the keyboard is on it, "+
					"where the palette says %v",
					state.Name, name, holding, rgba(onFocus))
			}
			if resting == holding {
				t.Errorf("%s: %s paints its label the same colour with and without the "+
					"keyboard, so nothing changed when it gained focus", state.Name, name)
			}
		})
		if found == 0 {
			t.Errorf("state %q holds no flooded control at all, so this walk passed by "+
				"walking nothing", state.Name)
		}
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
	case *parts.Button:
		return fmt.Sprintf("the button %q", typed.Text), true
	case *parts.Select:
		return fmt.Sprintf("the chooser %q", typed.Selected), true
	case *widget.Button:
		return fmt.Sprintf("a toolkit button %q, which nothing can dress", typed.Text), true
	case *widget.Select:
		return fmt.Sprintf("a toolkit chooser %q, which nothing can dress", typed.Selected), true
	}
	return "", false
}

// labelColour is what this control's own theme answers for the colour its
// label is painted with.
//
// The colour is asked of the theme of the control rather than read off a
// picture, because that is where the answer comes from: the renderer of a
// button sets a colour NAME on its text and leaves the text to resolve it
// against the scope it is in.
func labelColour(object fyne.CanvasObject) [4]uint32 {
	answer := toolkit.CurrentForWidget(object).Color(toolkit.ColorNameForeground, toolkit.VariantDark)
	return rgba(answer)
}

func rgba(from interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) [4]uint32 {
	r, g, b, a := from.RGBA()
	return [4]uint32{r, g, b, a}
}

// walkTree visits an object and everything drawn inside it, carrying whether
// the way down passed through a theme of its own.
//
// A widget is opened through its renderer, because the children of a widget are
// the objects its renderer holds and there is no other way in from outside the
// toolkit.
func walkTree(object fyne.CanvasObject, dressed bool, visit func(fyne.CanvasObject, bool)) {
	if object == nil {
		return
	}
	if _, own := object.(*container.ThemeOverride); own {
		dressed = true
	}
	visit(object, dressed)
	switch typed := object.(type) {
	case *fyne.Container:
		for _, child := range typed.Objects {
			walkTree(child, dressed, visit)
		}
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(typed).Objects() {
			walkTree(child, dressed, visit)
		}
	}
}
