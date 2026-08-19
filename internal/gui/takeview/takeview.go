// Package takeview builds the screen for the take and corrupt mode: somebody
// else's file goes in, and this says what is in it and what damaging it would
// do.
//
// # What this screen does not do yet
//
// It writes nothing, and it has no button that would. That is deliberate rather
// than unfinished: a control that does nothing is a promise, and a disabled one
// with no reason beside it is a puzzle. Writing arrives with the state that
// shows a run in progress and the way to stop it, because those three belong
// together and a screen that could start a run it could not stop would be worse
// than one that cannot start it.
//
// # Why the mode is asked rather than reimplemented
//
// Everything about what is refused, what a file was taken to hold and what a
// run would do comes from internal/take, which is also what the command line
// asks. The two surfaces therefore cannot disagree about any of it. What lives
// here is the drawing and the words, and the words are keys rather than
// sentences because this is the translated surface.
package takeview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// Opener reads somebody else's file and says what it holds.
//
// It is handed in for the same reason the manifest browser takes its readers
// that way: every state of this screen then has a way in that needs no disk,
// including the states worth checking, which are the ones nobody wants to keep
// a file around for.
type Opener func(path string, settings take.Settings) (*take.Opened, error)

// View is the screen and what it is currently looking at.
//
// Nothing is read until somebody chooses a file. A surface that went looking
// for something to open the moment it appeared would be acting on its own, and
// this mode is the one that writes beside other people's data.
type View struct {
	text      i18n.Catalogue
	window    fyne.Window
	described catalog.Catalog
	openFile  Opener

	chosen take.Settings
	opened *take.Opened
	path   string

	// The parts that change with what is on screen. They are kept rather than
	// rebuilt, because rebuilding a control throws away what somebody typed
	// into it and where the keyboard was standing.
	empty   *fyne.Container
	fileBox *fyne.Container
	// split holds the description apart from the settings, and is put away
	// while there is no file to describe.
	split       *container.Split
	settingsBox *fyne.Container
	planBox     *fyne.Container

	layers    *widget.CheckGroup
	share     *widget.Entry
	perRow    *widget.Select
	exclude   *widget.Entry
	delimiter *widget.Entry
	locale    *widget.Entry
	seed      *widget.Entry
	out       *widget.Entry
	manifest  *widget.Select
}

// New builds the screen. The window is what a file chooser has to appear over.
func New(text i18n.Catalogue, window fyne.Window, described catalog.Catalog,
	openFile Opener) *View {
	view := &View{
		text: text, window: window, described: described, openFile: openFile,
		chosen: take.Defaults(),
		empty:  container.NewVBox(), fileBox: container.NewVBox(),
		settingsBox: container.NewVBox(), planBox: container.NewVBox(),
	}
	view.buildControls()
	view.empty.Objects = []fyne.CanvasObject{
		wrapped(text.T(i18n.TakeEmpty)), wrapped(text.T(i18n.TakeDrop)),
	}
	view.redraw()
	return view
}

// Content is the whole screen.
//
// The parts are handed over in the order a person reads them, which is also the
// order the keyboard walks them. Building this as a border with the bar added
// last would draw the same and walk backwards, which was measured on this
// project once already.
func (v *View) Content() fyne.CanvasObject {
	choose := parts.NewButton(v.text.T(i18n.TakeChoose), v.choose)
	// What the file holds is on the left and what to do about it on the right,
	// each scrolling on its own.
	//
	// 🔴 A single column was the first shape and it does not survive a real
	// file. Rendered against five columns, the answer to what a run would do
	// had already fallen off the bottom of the window, and a file from the
	// people this mode is for carries dozens: the description grows with the
	// file while the settings and the answer do not, so the one thing somebody
	// came to read is pushed further away the more there is to read about.
	//
	// The toolkit's own split is used rather than a layout of ours. It also
	// gives a person the divider, so a wide description can be given room
	// without this screen deciding for them how much it deserves.
	answer := container.NewVBox(
		v.settingsBox,
		widget.NewSeparator(),
		v.planBox,
	)
	split := container.NewHSplit(
		container.NewVScroll(v.fileBox),
		container.NewVScroll(answer),
	)
	v.split = split
	body := container.NewVBox(container.NewHBox(choose), v.empty)
	return parts.Inset(container.NewBorder(body, nil, nil, nil, split))
}

// Open reads a file and puts the screen into the state that shows it.
//
// # Why this is exported
//
// Something outside this package has to reach that state without a file
// chooser: the guard holding what every screen draws, and the probe rendering
// them to pictures. It carries the whole state change rather than the drawing
// alone, which is the correction of a drift this project measured on the other
// screen, where an exported call that only drew reached a state no person can.
func (v *View) Open(path string) error {
	opened, err := v.openFile(path, v.chosen)
	if err != nil {
		v.opened, v.path = nil, ""
		v.showUnreadable(err)
		return err
	}
	v.opened, v.path = opened, path
	v.redraw()
	return nil
}

// Dropped takes a file dropped onto the window.
//
// Only the first is taken, and a drop of several says so by acting on one
// rather than by damaging whichever the system happened to list last. This mode
// works on one file at a time, which is a property of the mode rather than of
// the screen.
func (v *View) Dropped(dropped []fyne.URI) {
	if len(dropped) == 0 {
		return
	}
	_ = v.Open(dropped[0].Path())
}

func (v *View) choose() {
	dialog.ShowFileOpen(func(chosen fyne.URIReadCloser, err error) {
		if err != nil {
			v.showUnreadable(err)
			return
		}
		if chosen == nil {
			// A cancelled chooser leaves the screen as it was. Turning it into
			// a message would tell somebody something went wrong when they were
			// the one who decided nothing should.
			return
		}
		defer func() { _ = chosen.Close() }()
		_ = v.Open(chosen.URI().Path())
	}, v.window)
}

// showUnreadable puts the failure where the file would have been, and leaves
// the rest of the screen out of the way.
func (v *View) showUnreadable(err error) {
	v.empty.Objects = []fyne.CanvasObject{
		wrapped(v.text.T(i18n.TakeUnreadable) + " " + err.Error()),
		wrapped(v.text.T(i18n.TakeDrop)),
	}
	v.empty.Refresh()
	v.redraw()
}

// redraw shows the parts that belong to the state the screen is in.
//
// Hiding rather than rebuilding, so that a control keeps what was typed into it
// and the keyboard keeps its place.
func (v *View) redraw() {
	if v.opened == nil {
		v.empty.Show()
		v.hideSplit()
		return
	}
	v.empty.Hide()
	v.drawFile()
	v.drawPlan()
	v.fileBox.Show()
	v.settingsBox.Show()
	v.planBox.Show()
	if v.split != nil {
		v.split.Show()
	}
}

// hideSplit puts away everything that describes a file, for the state where
// there is none.
func (v *View) hideSplit() {
	v.fileBox.Hide()
	v.settingsBox.Hide()
	v.planBox.Hide()
	if v.split != nil {
		v.split.Hide()
	}
}

// wrapped is a paragraph that follows the width of the window rather than
// running off the edge of it.
func wrapped(words string) *widget.Label {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func heading(words string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(words, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// counted writes a number with the thing it counts, in the singular where there
// is one of it.
func counted(many int, thing string) string {
	if many == 1 {
		return "1 " + thing
	}
	return strconv.Itoa(many) + " " + thing + "s"
}
