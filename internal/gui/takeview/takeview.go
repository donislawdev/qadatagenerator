// Package takeview builds the screen for the take and corrupt mode: somebody
// else's file goes in, and this says what is in it, what damaging it would do,
// and then does it.
//
// # Why the button, the state in progress and the way to stop arrived together
//
// The screen was a preview first, on purpose. A control that does nothing is a
// promise and a disabled one with no reason beside it is a puzzle, so nothing
// offered to start a run until all three of these existed: a screen that could
// begin a run it could not stop would be worse than one that could not begin
// one at all.
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
	"errors"
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
	run       Background

	chosen take.Settings
	opened *take.Opened
	path   string

	// What the screen is doing and what the last run left. The stage decides
	// what is drawn and the two beside it carry what there is to say about it.
	stage  stage
	result take.Result
	failed error
	// halt is how the run in flight is asked to end, and nothing while there is
	// none.
	halt *stopper
	// closing records that somebody asked for the window while a run was going
	// on, so that the window can be given to them once the run has stopped.
	closing bool

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
	runBox      *fyne.Container

	layers    *widget.CheckGroup
	share     *widget.Entry
	perRow    *widget.Select
	exclude   *widget.Entry
	delimiter *widget.Entry
	locale    *widget.Entry
	seed      *widget.Entry
	out       *widget.Entry
	manifest  *widget.Select
	overwrite *widget.Check

	chooseFile *parts.Button
	damage     *parts.Button
	stop       *parts.Button
}

// New builds the screen.
//
// The window is what a file chooser has to appear over and what a run takes
// over while it is going on. The background is where a run happens, handed in
// so that every state of this screen can be reached without a goroutine.
func New(text i18n.Catalogue, window fyne.Window, described catalog.Catalog,
	openFile Opener, run Background) *View {
	view := &View{
		text: text, window: window, described: described, openFile: openFile,
		run: run, chosen: take.Defaults(),
		empty: container.NewVBox(), fileBox: container.NewVBox(),
		settingsBox: container.NewVBox(), planBox: container.NewVBox(),
		runBox: container.NewVBox(),
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
		widget.NewSeparator(),
		v.runBox,
	)
	split := container.NewHSplit(
		container.NewVScroll(v.fileBox),
		container.NewVScroll(answer),
	)
	v.split = split
	body := container.NewVBox(container.NewHBox(v.chooseFile), v.empty)
	// 🔴 Drawn again now that the split exists. The screen was first drawn while
	// this was still being built, so the state with no file had hidden every box
	// and left the split itself showing: two separator lines across an otherwise
	// empty half of the screen, dividing nothing from nothing. Seen in the
	// stored tree rather than by anybody looking, which is what that guard is
	// for.
	v.redraw()
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
	if v.busy() {
		// Nothing may take the file away from a run that is walking it. The
		// button that chooses one is disabled for the length of a run, and a
		// file dropped onto the window arrives by another road entirely, which
		// is why the rule lives here rather than on the control.
		return errRunInProgress
	}
	opened, err := v.openFile(path, v.chosen)
	if err != nil {
		v.opened, v.path = nil, ""
		v.forgetLastRun()
		v.showUnreadable(err)
		return err
	}
	if path != v.path {
		// A different file, so what the last run did is about something else.
		v.forgetLastRun()
	}
	v.opened, v.path = opened, path
	v.redraw()
	return nil
}

// errRunInProgress is what Open answers while a run is walking the file.
//
// It never reaches a screen, so it is a sentence rather than a key: the two
// roads into Open are a control that is disabled and a file dropped onto the
// window, and both of them ignore what comes back.
var errRunInProgress = errors.New("a run is in progress, so the file cannot be changed")

// forgetLastRun clears what this screen says about a run that is now about
// another file.
func (v *View) forgetLastRun() {
	v.stage, v.result, v.failed = stageReady, take.Result{}, nil
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
	v.drawRun()
	v.fileBox.Show()
	v.settingsBox.Show()
	v.planBox.Show()
	v.runBox.Show()
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
	v.runBox.Hide()
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

// beside is a mark with a paragraph that follows the width of the window.
//
// 🔴 A row of the toolkit gives each thing the width it asks for and no more,
// and a wrapping label asks for the width of its whole sentence. Put in a row it
// therefore does not wrap at all, it runs off the right edge, and the wrapping
// that was set on it does nothing. Seen on the render of the state that shows a
// run going on: the sentence saying what stopping leaves behind was cut mid
// word at the edge of the screen. A border gives its middle whatever is left
// over after the edges, which is the width wrapping needs.
func beside(mark fyne.Resource, words fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewIcon(mark), nil, words)
}

// wrappedPair is a label and a value long enough to need wrapping.
//
// Paths and lists of file names are as long as somebody's directory, so the
// value beside the label is the one thing on this screen whose length nobody
// here decides.
func wrappedPair(label, value string) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewBorder(nil, nil, name, nil, wrapped(value))
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
