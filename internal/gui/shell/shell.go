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
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
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
	// Icon is the name of the icon drawn beside this screen's name in the
	// column, or empty for a screen that shows none.
	//
	// It is the NAME of one of this window's icons rather than a resource,
	// because the colour of an icon on a button depends on where the
	// keyboard is, and a resource is a colour already decided. ShowIcon
	// beside the button says what that costs when it is got wrong.
	Icon string
	// Group is the heading this screen sits under in the column, and screens
	// naming the same one sit together beneath it.
	//
	// A screen naming NO group is not work somebody came here to do, so it goes
	// to the foot of the column, apart from the screens that are. There is one
	// today and it is About, which answers a question nobody has while working.
	Group string
	// OnDropped is what this screen does with a file dropped onto the window,
	// or nothing for a screen that takes none.
	//
	// It is declared here rather than on the screen itself because dropping is
	// a property of the WINDOW: the toolkit takes one callback for the whole of
	// it, so a screen registering its own would be overwriting whatever the
	// last screen registered. Which screen is open is something this frame
	// already knows, so the routing belongs here and can be checked without a
	// window at all.
	OnDropped func([]fyne.URI)
}

// Brand is the product's name at the head of the column and its promise at the
// foot, which the accepted drawing has and the window did not until 2026-09-09.
//
// # Why the frame is told these rather than reaching for them
//
// The version lives in a package the window has no business importing, and the
// words are a translator's. Handing both in keeps this frame buildable in a
// test with neither, which is the same reason every screen is handed its
// readers rather than opening its own files.
//
// # Why they are worth the room they take
//
// The name says which program this is, which a window with one title bar and
// four screens does not otherwise say once anybody maximises it. The promise
// under it is the one sentence about this tool that is true before a person
// presses anything, and the foot of the column is where somebody looks while
// deciding whether to trust a window at all.
type Brand struct {
	// Name is the product's name and Mark the icon beside it.
	Name string
	Mark string
	// Version is what this build is, and Promise what it does not do.
	Version string
	Promise string
}

// Shell is the frame and which screen is showing.
type Shell struct {
	screens []Screen
	brand   Brand
	buttons []*parts.Button
	items   []*item
	body    *fyne.Container
	// title is the name of the screen being read, drawn above it.
	//
	// One label rewritten rather than one per screen, for the reason parts.Title
	// gives: the frame is the only thing here that knows which screen is open,
	// and a title each screen drew for itself is a title one screen would
	// eventually not draw.
	title *widget.Label
}

// New builds the frame around the screens, showing the first one.
func New(screens []Screen, brand Brand) *Shell {
	frame := &Shell{screens: screens, brand: brand, body: container.NewStack(), title: parts.Title("")}
	for index := range screens {
		button := frame.button(index)
		frame.buttons = append(frame.buttons, button)
		frame.items = append(frame.items, newItem(button))
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
	button := parts.NewButton(s.screens[index].Name, func() { s.Open(index) })
	if name := s.screens[index].Icon; name != "" {
		// The icon leads the words, which is the order every column of
		// places to go is read in and the order the accepted drawing has.
		button.ShowIcon(name)
	}
	// 🔴 The names line up down the left edge rather than being centred, which
	// is what the toolkit does when nobody says otherwise.
	//
	// Seen on a real window on 2026-09-08, the first time anybody looked at
	// one: centring moves every name by half the difference between its own
	// length and the width of the column, so five names of five different
	// lengths started at five different places and the column read as ragged
	// rather than as a list. The accepted drawing aligns them left, and so does
	// every list of places to go that anybody has seen before.
	button.Leading = true
	// No edge either. Five boxes down a column read as five boxes rather than
	// as a list, and the mark of the current screen is what says where you are.
	button.Plain = true
	return button
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
	// # Why low importance, which is what takes the edge off these five
	//
	// Changed on 2026-09-08 with DEC-225. They carried the edge every pressable
	// button carries, and the column read as five identical boxes stacked on
	// one another, with nothing saying which of them mattered. Squinting at the
	// window, which is point two of the checklist in docs/17, found nothing
	// standing out anywhere.
	//
	// This is not a new mechanism and not an exception written for this
	// column. parts.Button.markEdge already leaves the edge off a button of low
	// importance, and the owner chose that on 2026-09-05 for the heading of the
	// list of generators, with the number beside it: that heading measures 1.00
	// against the panel and the arrow beside its words is the answer to it. The
	// same answer is used here, and the arrow is already below.
	//
	// What this does NOT change is the one thing worth protecting: these stay
	// buttons. The comment on button above says why, and it is the keyboard.
	for at, button := range s.buttons {
		s.items[at].show(at == index)
		button.Refresh()
	}
	// The name of the screen, written above it as well as in the column. Not a
	// repetition: the column says where a person can go and this says where
	// they are, and until 2026-09-08 the screens answered the second question
	// with nothing at all.
	s.title.SetText(s.screens[index].Name)
	s.body.Objects = []fyne.CanvasObject{s.screens[index].Content}
	s.body.Refresh()
}

// Dropped hands a dropped file to the screen a person is looking at.
//
// # Why the screen underneath rather than every screen
//
// A drop is aimed at what is on display. Handing it to all of them would have
// the manifest browser open something somebody dropped while reading About, and
// handing it to a fixed screen would mean the window quietly switching under
// them. A screen that takes no files ignores the drop, which is what a person
// dropping one onto the About screen expects.
//
// The position is taken and not used. The toolkit reports where the drop landed
// and no screen here places anything by pixel, so using it would be inventing a
// meaning for a number nobody set.
func (s *Shell) Dropped(_ fyne.Position, dropped []fyne.URI) {
	index, showing := s.Showing()
	if !showing || len(dropped) == 0 {
		return
	}
	if takes := s.screens[index].OnDropped; takes != nil {
		takes(dropped)
	}
}

// Showing is the index of the screen on display, and reports whether there is
// one at all.
// Reveal opens the screen with this name, and does nothing if there is none.
//
// By name rather than by position, because the caller is a keyboard shortcut
// living outside this package and a number written there would be a second
// register of an order this frame already holds. The neighbouring list of
// states paid for exactly that on 2026-09-08: it pointed at screens by index,
// the order changed, and every one of them silently moved.
func (s *Shell) Reveal(name string) {
	for index, screen := range s.screens {
		if screen.Name == name {
			s.Open(index)
			return
		}
	}
}

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
	column := s.column()
	// 🔴 A rule between the column and the screen, which the accepted drawing
	// has and the window did not. Without it the two meet on the same flat
	// ground and the column reads as words floating beside the content rather
	// than as a part of the frame. It is a separator rather than a rectangle of
	// our own so that its colour and its thickness are the theme's answer, the
	// same as every other line in the window.
	//
	// A row lays it out beside the column, and a row gives each child the full
	// height, which is what turns a separator into a vertical rule.
	side := container.NewHBox(column, widget.NewSeparator())
	// The title carries the margin of the screen below it on three sides and
	// none at the foot, so that the name lines up with the content under it and
	// the gap between them is the screen's own top margin rather than two
	// margins meeting.
	margin := palette.StepOf(palette.NameStepSection)
	titled := container.New(layout.NewCustomPaddedLayout(margin, 0, margin, margin), s.title)
	screen := container.New(layout.NewBorderLayout(titled, nil, nil, nil), titled, s.body)
	return container.New(layout.NewBorderLayout(nil, nil, side, nil), side, screen)
}

// column is the names down the side, with the headings that group them.
//
// # Why the headings, added on 2026-09-08 with DEC-225
//
// Five names in a row said what the screens ARE. The headings say what a
// person came to DO, and that is the same change the order itself is: docs/08
// calls its first scenario the path that builds adoption and its second the
// shortest way to the first wow, and both are the manual tester this window is
// hardest for.
//
// A heading is not a control and takes no focus. A heading somebody tabs onto
// is a stop that goes nowhere, and point nine of the checklist in docs/17 is
// about reaching what a mouse reaches rather than about reaching more.
func (s *Shell) column() fyne.CanvasObject {
	// No capacity hinted, and that is not an oversight. The count is the names
	// plus however many headings they turn out to need, and writing it as a
	// sum with a number in it is what the guard over literals in this window
	// refuses, correctly: it cannot tell a slice hint from a distance.
	var items []fyne.CanvasObject
	group := ""
	footed := false
	for at := range s.buttons {
		switch name := s.screens[at].Group; {
		case name == "" && !footed:
			// The rule above the foot of the column, which the accepted
			// drawing puts there. It says that what follows is not more of the
			// work above it, which is the whole reason a screen naming no
			// group is sent down here in the first place.
			items = append(items, layout.NewSpacer(), widget.NewSeparator())
			footed = true
		case name != "" && name != group:
			items = append(items, heading(name))
		}
		group = s.screens[at].Group
		items = append(items, s.items[at].onScreen)
	}
	// The promise goes under the last name rather than beside it, because it
	// belongs to the program and not to the screen above it.
	if s.brand.Promise != "" {
		items = append(items, paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
			palette.NameStepTight, palette.NameStepHairline,
			paint.Text(s.footnote(), palette.TypeLabel, palette.TextSecondary)))
	}
	column := container.NewVBox(items...)
	// 🔴 A border rather than another entry in the column, because the mark has
	// to stay at the top while the names below it grow. A column of the toolkit
	// gives every child its minimum height, so the spacer that pushes About to
	// the foot would have nothing to push against if the mark were one of them.
	whole := fyne.CanvasObject(column)
	if mark := s.brandMark(); mark != nil {
		whole = container.New(layout.NewBorderLayout(mark, nil, nil, nil), mark, column)
	}
	return container.New(fixedWidth{width: palette.SidebarWidth}, whole)
}

// brandMark is the product's name at the head of the column, or nothing when
// the frame was not given one.
//
// The tile carries the accent, which is the one place on this screen where the
// accent says "this is the program" rather than "this is the action". It is the
// same shape the accepted drawing has: a rounded square of the fill colour with
// the mark on it, and the name beside it at the weight a name is read first at.
func (s *Shell) brandMark() fyne.CanvasObject {
	if s.brand.Name == "" {
		return nil
	}
	row := []fyne.CanvasObject{}
	if s.brand.Mark != "" {
		tile := paint.On(paint.Fill(palette.FillPrimary, palette.RadiusControl),
			container.NewCenter(paint.Icon(s.brand.Mark, palette.TextOnFill, palette.IconSmall)))
		row = append(row, paint.AtLeast(palette.ControlHeight, palette.ControlHeight, tile))
	}
	row = append(row, paint.Middle(paint.Strong(s.brand.Name, palette.TypeBody, palette.TextPrimary)))
	return paint.Pad(palette.NameStepHairline, palette.NameStepSection,
		palette.NameStepHairline, palette.NameStepHairline,
		paint.Row(palette.NameStepRelated, row...))
}

// footnote is the version and the promise on one line.
//
// Joined here rather than by whoever built the frame, because the separator
// between them is a matter of how the line is drawn and not of what it says. A
// translator writing it into the words would have every language carrying this
// window's punctuation.
func (s *Shell) footnote() string {
	if s.brand.Version == "" {
		return s.brand.Promise
	}
	return s.brand.Version + brandSeparator + s.brand.Promise
}

// brandSeparator is what stands between the version and the promise. A middle
// dot rather than a dash, since DEC-34 leaves this window one dash and it is
// the one inside words.
const brandSeparator = "  ·  "

// heading is one group name, drawn quieter than the names it groups.
//
// A RichText rather than a Label, because a label takes the ordinary text
// colour and a heading that shouts as loudly as its own contents has grouped
// nothing. The colour and the size are asked for by name, so the one place
// that answers for either is the palette.
func heading(name string) fyne.CanvasObject {
	// 🔴 Upper case, which is what the accepted drawing asks for and what
	// separates a label naming a group from the names inside it. The drawing
	// also spaces the letters out, and the toolkit has no way to say that, so
	// the case is the part of it that can be carried.
	//
	// Done here rather than in the catalogue, because it is a matter of how the
	// word is drawn and not of what it says. A translator writing a group name
	// in capitals in the file would have it shouted twice on a surface that
	// chose not to do this, and the command line is such a surface.
	segment := &widget.TextSegment{Text: strings.ToUpper(name)}
	segment.Style.ColorName = fyne.ThemeColorName(palette.NameTextSubdued)
	segment.Style.SizeName = theme.SizeNameCaptionText
	return widget.NewRichText(segment)
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
