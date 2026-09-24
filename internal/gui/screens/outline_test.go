package screens

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryPressableControlDrawsItsOwnEdge walks every state of every screen and
// asks each control whether the thing that makes it visible is on screen with it.
//
// # Why the question is about the tree and not about the colour
//
// A test that asks what colour a control will be painted passes even when the
// control was handed to a layout wrongly, because the toolkit resolves colours
// from the moment a scope is built rather than from where the object ends up.
// That cost a whole guard on 2026-09-08. So this asks the shape of what is
// drawn instead: the face the renderer of the control puts on screen has to
// carry a stroke, or not carry one, according to what the control is.
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
// The filled parts.Button has no edge either, and it never reaches this walk at
// all: it is handed to a layout bare and its face carries no edge for the walk
// to find. That is deliberate and it is why the guard below exists. A control
// this one silently passes over is a control the promise in this name is no
// longer true of, and an exception nobody checks is the same thing as no rule.
//
// # 🔴 Why there is one walk here since 2026-09-09, where there were two
//
// There were two because the two controls answered this question in two
// different places: the button drew its own face, so its edge was inside its
// renderer, while the chooser was the toolkit's with a rectangle laid BESIDE
// it, so its edge was a sibling in the layout. The comment on the second walk
// said what to do on the day the chooser drew itself too, and this is it: the
// walk over siblings goes, and this one covers both.
//
// What went with it is the colour and the thickness of the edge, which the
// walk over siblings also asked. Those moved to parts, where the renderer of
// each control can be reached and where the same questions are already asked of
// the button. Here the question is the structural one, over every state of
// every screen: whether the mark is there at all, and whether the two controls
// that go without are the two that were decided.
func TestEveryPressableControlDrawsItsOwnEdge(t *testing.T) {
	data := sampleWithSource(t)

	for _, state := range All() {
		content := Assemble(i18n.English, data, state)
		found := 0
		walkTree(content, false, func(object fyne.CanvasObject, _ bool) {
			control, name := ourControl(object)
			if control == nil || !object.Visible() {
				return
			}
			found++

			drawn := edgesInFace(test.WidgetRenderer(object.(fyne.Widget)).Objects(), control.focusIsNoEdge())
			// 🔴 A chooser has no notion of importance, so the quiet cases are
			// asked of the control rather than assumed of a button. Reading
			// Importance off something that has none would have been a
			// compile error here and a wrong rule anywhere it was not.
			quiet := control.quiet()
			if drawn == quiet {
				t.Errorf("%s: %s draws an edge: %v, where a quiet one is %v. "+
					"An edge says a control can be used, so a disabled one and a heading go without",
					state.Name, name, drawn, quiet)
			}
		})

		if found == 0 {
			t.Errorf("state %q holds no pressable control at all, so this walk passed by walking nothing",
				state.Name)
		}
	}
}

// edgesInFace says whether anything in this drawing paints a stroke, leaving
// out the mark of focus when the control draws that mark without an edge.
//
// The same question edgeIn asks of a layout, asked of what a renderer holds.
// Kept separate rather than folded into that one, because that one deliberately
// stops two levels down to avoid mistaking a control's own fill for an edge,
// and a face is a tree of our own shapes where every level is ours.
//
// # 🔴 Why the mark of focus is left out for a plain button, added 2026-09-25
//
// A plain button has no edge, and in focus it draws an outline in mark.focus,
// because focus has to show on every control that can hold the keyboard
// (paint.ButtonFace, the branch for Plain). That outline is the mark of focus
// and not an edge. This walk had never met one: it builds each screen without
// a keyboard, so no control it walked was in focus until the catalogue of
// controls (DEC-253) put one there, and it went red on exactly the two plain
// buttons shown with the keyboard on them. Leaving the mark out for every
// quiet control would have been simpler and would have hidden a disabled
// control drawing a mark of focus, which it must never do, so it is left out
// only for a control that says it draws one without an edge.
func edgesInFace(objects []fyne.CanvasObject, focusIsNoEdge bool) bool {
	mark := rgba(palette.Ink(palette.MarkFocus))
	for _, object := range objects {
		switch shape := object.(type) {
		case *canvas.Rectangle:
			if shape.StrokeWidth > 0 && !(focusIsNoEdge && rgba(shape.StrokeColor) == mark) {
				return true
			}
		case *fyne.Container:
			if edgesInFace(shape.Objects, focusIsNoEdge) {
				return true
			}
		}
	}
	return false
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
	// focusIsNoEdge says the control marks focus with an outline where it
	// has no edge, so a stroke in mark.focus on it is that mark.
	focusIsNoEdge() bool
}

// ourControl is the control this object is, if it is one of the two this window
// draws itself, named for a message.
//
// 🔴 It replaced a helper that looked for the CONTAINER a control had been
// dressed in, and the difference is the whole point of the change it came with.
// That one could only find a control somebody had wrapped, so a control handed
// to a layout bare left the walk without a word - which is how a guard about
// every pressable control came to be silent about the one this project cares
// most about. Asking the object what it is has no such gap.
//
// The filled button is left out, and not as a quiet one. It goes without an
// edge at rest because its fill says it can be pressed, and gains one in focus,
// so neither answer of quiet() describes it. That exception has a guard of its
// own below. Until 2026-09-24 it was a type of its own and this switch never
// saw it, so leaving it out here keeps what this walk has always covered.
func ourControl(object fyne.CanvasObject) (outlined, string) {
	switch typed := object.(type) {
	case *parts.Button:
		if typed.Filled {
			return nil, ""
		}
		return quietButton{typed}, fmt.Sprintf("the button %q", typed.Text)
	case *parts.Select:
		return quietSelect{typed}, fmt.Sprintf("the chooser %q", typed.Selected)
	}
	return nil, ""
}

// quietButton and quietSelect say when each control goes without an edge.
type quietButton struct{ *parts.Button }

func (b quietButton) quiet() bool {
	// Plain rather than low importance, since 2026-09-09. The button draws its
	// own face and says in one field what it used to say by borrowing a word
	// from the toolkit that also meant several other things.
	return b.Disabled() || b.Plain
}

// focusIsNoEdge is a plain button somebody can use. A disabled one never holds
// the keyboard, so a mark of focus on it is a fault this walk has to see.
func (b quietButton) focusIsNoEdge() bool { return b.Plain && !b.Disabled() }

type quietSelect struct{ *parts.Select }

func (s quietSelect) quiet() bool { return s.Disabled() }

// focusIsNoEdge is never true of a chooser: it has an edge whenever it can be
// used, and focus recolours that edge.
func (s quietSelect) focusIsNoEdge() bool { return false }

// edgeIn is the outlining rectangle of this container.
//
// Two levels are read rather than one, because an edge is put inside a padded
// container of the toolkit to hold the stroke inside the control - by the faces
// in paint today, and by parts.Outlined until it was deleted with DEC-253. The walk stops there rather than going the whole way down, so that
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
// The walk above finds controls by looking for a container holding a drawn
// edge. The filled parts.Button is handed to a layout bare, so that walk passes over it
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
			if filledAction(object) != nil {
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
			// identified as one parts.Outlined built. Asked of any container
			// at all it also answers yes for the ground of a Card, and the
			// first version here did ask that and reported four states that
			// were perfectly correct. What Outlined produced, and what an edge
			// beside a control would still be, is a stack holding the control
			// and the edge as SIBLINGS, so that is what is looked for.
			var slab *parts.Button
			for _, child := range group.Objects {
				if filled := filledAction(child); filled != nil {
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
			"Either a screen lost its one action or no button is filled, and the second is what the palette entry was added to stop")
	}
}

// filledAction is the object as the one action of a screen, or nil when it is
// anything else, an ordinary button included.
func filledAction(object fyne.CanvasObject) *parts.Button {
	if button, is := object.(*parts.Button); is && button.Filled {
		return button
	}
	return nil
}
