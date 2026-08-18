// Package shell is the frame the screens sit in.
//
// # Why there is a frame at all, and why it is a column down the side
//
// Until 2026-08-17 the window was one screen with a title around it. docs/08
// names seven, so something had to say how a person moves between them, and
// docs/17 says outright that it does not settle the layout of screens. The
// owner settled it that day: a column of names down the left, content to the
// right.
//
// What was turned down, with the reason, because a decision without one invites
// the next session to reopen it. Tabs across the top are cheaper today and come
// with the toolkit, and they narrow as they are added: at seven the name of the
// screen a person is on stops being fully readable, which is precisely when it
// starts to matter. A window per screen was turned down as seven windows to
// size, place and close.
package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Screen is one thing a person can open, and what it shows.
//
// The content is built once and kept, rather than rebuilt on each visit. A
// screen holding what somebody opened is a screen that still holds it when they
// come back, and rebuilding would quietly throw that away.
type Screen struct {
	Name    string
	Content fyne.CanvasObject
}

// Shell is the frame and which screen is showing.
type Shell struct {
	screens []Screen
	buttons []*parts.Button
	body    *fyne.Container
}

// New builds the frame around the screens, showing the first one.
func New(screens []Screen) *Shell {
	frame := &Shell{screens: screens, body: container.NewStack()}
	for index := range screens {
		frame.buttons = append(frame.buttons, frame.button(index))
	}
	if len(screens) > 0 {
		frame.Open(0)
	}
	return frame
}

// button is one name in the column.
//
// A button rather than a row of a list, and the reason is the keyboard. Point
// nine of the GUI checklist asks that everything reachable with a mouse be
// reachable without one, and a button is what this toolkit gives focus to and
// draws a focus ring around. That ring is measured: the focus colour stands at
// 9.02 against the window, where the guideline asks for three.
func (s *Shell) button(index int) *parts.Button {
	return parts.NewButton(s.screens[index].Name, func() { s.Open(index) })
}

// Open shows one screen and marks its name as the one being read.
//
// # Why the mark is a shape and not the colour it used to be
//
// It was the toolkit's high importance, which paints the name in the primary
// colour. This project answers for the primary colour and for the focus colour
// with the same measured value, so the name of the screen a person was on and
// the name the keyboard was on were painted identically. Seen on a real window
// on 2026-08-18: the first press of Tab changed nothing anybody could see, and
// the second made two names look equally current.
//
// Reading the rendered tree says the same thing in words, and shows the trap in
// guarding it by name: the two rectangles are reported as fillColor="primary"
// and fillColor="focus", two names for one value. So the guard compares what
// the theme answers, not what the tree calls it.
//
// The mark is therefore a shape, which docs/17 asks for anyway: a colour that
// carries meaning on its own is a difference somebody who does not separate
// hues cannot see. The colour is left to mean one thing only, which is where
// the keyboard is.
func (s *Shell) Open(index int) {
	if index < 0 || index >= len(s.screens) {
		return
	}
	for at, button := range s.buttons {
		button.Importance = widget.MediumImportance
		button.SetIcon(nil)
		if at == index {
			button.SetIcon(theme.NavigateNextIcon())
		}
		button.Refresh()
	}
	s.body.Objects = []fyne.CanvasObject{s.screens[index].Content}
	s.body.Refresh()
}

// Showing is the index of the screen on display, and reports whether there is
// one at all.
func (s *Shell) Showing() (int, bool) {
	if len(s.body.Objects) == 0 {
		return 0, false
	}
	for index, screen := range s.screens {
		if screen.Content == s.body.Objects[0] {
			return index, true
		}
	}
	return 0, false
}

// Content is the whole frame.
//
// # Why the column is handed over before the body
//
// The keyboard walks the objects of a container in the order they were given,
// and the convenient way of building a border appends the edges after the
// middle. Measured on 2026-08-18: Tab reached the screen first and the column of
// names, which stands down the left and is read first, came last. The layout is
// therefore told where the parts go, and the parts are handed over in the order
// a person meets them.
func (s *Shell) Content() fyne.CanvasObject {
	names := make([]fyne.CanvasObject, 0, len(s.buttons))
	for _, button := range s.buttons {
		names = append(names, button)
	}
	column := container.New(fixedWidth{width: palette.SidebarWidth}, container.NewVBox(names...))
	return container.New(layout.NewBorderLayout(nil, nil, column, nil), column, s.body)
}

// fixedWidth gives what it holds one width whatever is in it.
//
// # Why the column does not simply take the width of its longest name
//
// It would then change width when a screen is added, when a name is translated
// and when the desktop scales the interface, so the content beside it would
// start at a different place on every build. A person moving between two
// screens would see everything shift sideways for no reason they caused.
type fixedWidth struct{ width float32 }

func (f fixedWidth) MinSize(objects []fyne.CanvasObject) fyne.Size {
	height := float32(0)
	for _, object := range objects {
		if tall := object.MinSize().Height; tall > height {
			height = tall
		}
	}
	return fyne.NewSize(f.width, height)
}

func (f fixedWidth) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Resize(fyne.NewSize(f.width, size.Height))
		object.Move(fyne.NewPos(0, 0))
	}
}
