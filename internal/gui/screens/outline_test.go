package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryPressableControlDrawsItsOwnEdge walks every state of every screen and
// asks each control whether the thing that makes it visible is on screen with it.
//
// # Why the question is about the tree and not about the colour
//
// The lesson is a day old and it cost a whole guard. A test that asks what
// colour a control will be painted passes even when the control was handed to a
// layout wrongly, because the toolkit resolves colours from the moment a scope
// is built rather than from where the object ends up. So this asks the shape of
// the tree instead: the layout has to have been given the container Outlined
// builds, with the edge in it beside the control.
//
// # Why it is not a number off the render either
//
// The render is the stronger evidence and it is measured separately, but it can
// only report what it can find, and an edge is not a text node. The probe that
// reads pixels says so about itself in its own first paragraph.
//
// # What it also holds
//
// That the two cases with no edge are the two that were decided, rather than
// two that drifted. A disabled button and the one low importance heading go
// without, everything else has one, and a button that quietly loses its edge
// because somebody set an importance fails here rather than on somebody's
// screen.
func TestEveryPressableControlDrawsItsOwnEdge(t *testing.T) {
	data := sampleWithSource(t)
	want := rgba(palette.Named(palette.NameControlOutline).NRGBA())

	for _, state := range All() {
		content := Assemble(i18n.English, data, state)
		found := 0
		walkTree(content, false, func(object fyne.CanvasObject, _ bool) {
			group, isGroup := object.(*fyne.Container)
			if !isGroup {
				return
			}
			control, name := controlDressedIn(group)
			if control == nil {
				return
			}
			found++

			edge := edgeIn(group)
			if edge == nil {
				t.Errorf("%s: %s stands on screen with no edge beside it. "+
					"A layout was given the dressed control rather than what Outlined returns",
					state.Name, name)
				return
			}
			if got := rgba(edge.StrokeColor); got != want {
				t.Errorf("%s: %s is outlined %v, where the palette says %v",
					state.Name, name, got, want)
			}
			if edge.StrokeWidth != palette.BorderThickness {
				t.Errorf("%s: %s is outlined %v thick, where the palette says %v. "+
					"Below that the toolkit draws the line between two rows of pixels "+
					"and every pixel of it is a half blend",
					state.Name, name, edge.StrokeWidth, palette.BorderThickness)
			}

			// 🔴 A chooser has no notion of importance, so the quiet cases are
			// asked of the control rather than assumed of a button. Reading
			// Importance off something that has none would have been a
			// compile error here and a wrong rule anywhere it was not.
			quiet := control.quiet()
			if edge.Visible() == quiet {
				t.Errorf("%s: %s shows its edge %v where it should be %v. "+
					"An edge says a control can be used, so a disabled one and a heading go without",
					state.Name, name, edge.Visible(), !quiet)
			}
		})
		if found == 0 {
			t.Errorf("state %q holds no button at all, so this walk passed by walking nothing",
				state.Name)
		}
	}
}

// outlined is a control this window draws an edge around, asked whether this
// is one of its quiet moments.
//
// An interface rather than a type switch at the call site, because the two
// controls answer that question from different fields: a button is quiet when
// it is disabled or a heading, and a chooser has no notion of importance at
// all.
type outlined interface {
	quiet() bool
}

// controlDressedIn is the control this container was built around, if it was.
//
// Only the direct children are read. A container holding such a container
// somewhere below it is an ordinary layout and owes nothing, and reading the
// whole subtree would make every ancestor of every control a finding.
func controlDressedIn(group *fyne.Container) (outlined, string) {
	for _, child := range group.Objects {
		dressed, own := child.(*container.ThemeOverride)
		if !own {
			continue
		}
		var found outlined
		var name string
		walkTree(dressed, false, func(object fyne.CanvasObject, _ bool) {
			if found != nil {
				return
			}
			switch typed := object.(type) {
			case *parts.Button:
				found, name = quietButton{typed}, fmt.Sprintf("the button %q", typed.Text)
			case *parts.Select:
				found, name = quietSelect{typed}, fmt.Sprintf("the chooser %q", typed.Selected)
			}
		})
		if found != nil {
			return found, name
		}
	}
	return nil, ""
}

// quietButton and quietSelect say when each control goes without an edge.
type quietButton struct{ *parts.Button }

func (b quietButton) quiet() bool {
	return b.Disabled() || b.Importance == widget.LowImportance
}

type quietSelect struct{ *parts.Select }

func (s quietSelect) quiet() bool { return s.Disabled() }

// edgeIn is the outlining rectangle of this container.
//
// Two levels are read rather than one, because Outlined puts the rectangle
// inside a padded container of the toolkit to hold the stroke inside the
// control. The walk stops there rather than going the whole way down, so that
// a rectangle belonging to the control itself, which every button has for its
// own fill, is never mistaken for an edge.
func edgeIn(group *fyne.Container) *canvas.Rectangle {
	for _, child := range group.Objects {
		if edge, is := child.(*canvas.Rectangle); is {
			return edge
		}
		nested, isGroup := child.(*fyne.Container)
		if !isGroup {
			continue
		}
		for _, grandchild := range nested.Objects {
			if edge, is := grandchild.(*canvas.Rectangle); is {
				return edge
			}
		}
	}
	return nil
}
