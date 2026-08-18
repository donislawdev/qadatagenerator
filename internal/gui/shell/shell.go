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
	"fyne.io/fyne/v2/widget"

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
	buttons []*widget.Button
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
func (s *Shell) button(index int) *widget.Button {
	return widget.NewButton(s.screens[index].Name, func() { s.Open(index) })
}

// Open shows one screen and marks its name as the one being read.
//
// The mark is weight rather than colour alone. The three meaning colours of
// this palette differ from one another by a contrast of 1.18 to 1.41, so a
// colour on its own is a difference somebody who does not separate hues cannot
// see, and this project holds that as a measurement rather than as a guideline.
func (s *Shell) Open(index int) {
	if index < 0 || index >= len(s.screens) {
		return
	}
	for at, button := range s.buttons {
		button.Importance = widget.MediumImportance
		if at == index {
			button.Importance = widget.HighImportance
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
func (s *Shell) Content() fyne.CanvasObject {
	names := make([]fyne.CanvasObject, 0, len(s.buttons))
	for _, button := range s.buttons {
		names = append(names, button)
	}
	column := container.New(fixedWidth{width: palette.SidebarWidth}, container.NewVBox(names...))
	return container.NewBorder(nil, nil, column, nil, s.body)
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
