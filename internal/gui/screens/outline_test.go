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
//
// # The third case, which this walk cannot see, and where it is held instead
//
// parts.PrimaryButton has no edge either, and it never reaches this walk at
// all: it is handed to a layout bare rather than through the container Outlined
// builds, so controlDressedIn never finds it. That is deliberate and it is why
// the guard below exists. A control this one silently passes over is a control
// the promise in this name is no longer true of, and an exception nobody checks
// is the same thing as no rule.
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
//
// 🔴 A rectangle counts only if it DRAWS a stroke, and that qualification was
// missing until 2026-09-09.
//
// Without it this answered yes for any rectangle at all, which was true of
// every container it was asked about on the day it was written and stopped
// being true the moment controls gained a floor under their height. That floor
// is a fully transparent rectangle with no stroke, put there to give a control
// a minimum size, and it made this report an edge on twelve states where no
// edge is drawn - including on the one control this project deliberately leaves
// without one.
//
// The name was right the whole time and the test was reading something else.
// Asking for a stroke is what makes the question match the name: an edge is a
// mark a person can see, and a rectangle that paints nothing is not one.
func edgeIn(group *fyne.Container) *canvas.Rectangle {
	drawn := func(shape *canvas.Rectangle) bool { return shape.StrokeWidth > 0 }

	for _, child := range group.Objects {
		if edge, is := child.(*canvas.Rectangle); is && drawn(edge) {
			return edge
		}
		nested, isGroup := child.(*fyne.Container)
		if !isGroup {
			continue
		}
		for _, grandchild := range nested.Objects {
			if edge, is := grandchild.(*canvas.Rectangle); is && drawn(edge) {
				return edge
			}
		}
	}
	return nil
}

// TestTheFilledActionIsTheNamedExceptionToTheEdgeRule holds the one pressable
// control that goes without an edge on purpose.
//
// # Why it needs a guard of its own rather than a sentence
//
// The walk above finds controls by looking for the container Outlined builds.
// parts.PrimaryButton is handed to a layout bare, so that walk passes over it
// without a word and stays green whatever happens to it. A rule with an
// exception nobody checks is a rule that has quietly stopped covering the thing
// it names, which in this repository has already happened to a register of
// generators and to a register of guards.
//
// # What is actually asked here
//
// Two things, and both of them are about the exception rather than about the
// colour. Whether the slab is painted from the palette is asked next door in
// parts, where the renderer can be reached.
//
//   - that a filled action exists somewhere, because an exception describing
//     nothing is worth deleting rather than keeping
//   - that no filled action is ALSO dressed in an outline. The fill is what
//     makes this control findable, standing 4.99 against the window ground
//     where 3.0 is asked. An edge around it as well would put two marks on one
//     control and say that the loudest thing on the screen needed help being
//     seen.
func TestTheFilledActionIsTheNamedExceptionToTheEdgeRule(t *testing.T) {
	data := sampleWithSource(t)
	slabs := 0

	for _, state := range All() {
		content := Assemble(i18n.English, data, state)

		walkTree(content, false, func(object fyne.CanvasObject, _ bool) {
			if _, filled := object.(*parts.PrimaryButton); filled {
				slabs++
			}
		})

		walkTree(content, false, func(object fyne.CanvasObject, _ bool) {
			group, isGroup := object.(*fyne.Container)
			if !isGroup {
				return
			}
			// 🔴 Direct children only, and that is the whole correctness of
			// this check. edgeIn answers about any rectangle among a
			// container's children or grandchildren, which is exactly right
			// where it was written, right after a container has been
			// identified as one Outlined built. Asked of any container at all
			// it also answers yes for the ground of a Card, and the first
			// version here did ask that and reported four states that were
			// perfectly correct. What Outlined actually produces is a stack
			// holding the control and the edge as SIBLINGS, so that is what is
			// looked for.
			var slab *parts.PrimaryButton
			for _, child := range group.Objects {
				if filled, is := child.(*parts.PrimaryButton); is {
					slab = filled
					break
				}
			}
			if slab == nil || edgeIn(group) == nil {
				return
			}
			t.Errorf("%s: the filled action %q stands beside an outline of its own, "+
				"so one control carries two marks saying it is a control",
				state.Name, slab.Text)
		})
	}

	if slabs == 0 {
		t.Error("no filled action stands on any screen, so the exception this file names describes nothing. " +
			"Either a screen lost its one action or parts.PrimaryButton is unused, and the second is what the palette entry was added to stop")
	}
}
