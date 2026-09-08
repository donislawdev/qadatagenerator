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
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
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
	work      background.Work

	chosen take.Settings
	// clipboard is where the line that would do this in a terminal goes, or
	// nothing when this screen is being rendered rather than shown.
	clipboard Clipboard
	// command is that line, and the button beside it copies it.
	command     *widget.Label
	copyCommand *parts.Button
	opened      *take.Opened
	path        string

	// reading is the file being read right now, and nothing while none is.
	//
	// # Why this is not one more value of stage
	//
	// stage says what the last RUN left, and reading a file must not throw that
	// away. The separator is read again on every keystroke, and a person who
	// corrects a comma to a semicolon after a run has not undone the run.
	reading string
	// asked counts the reads asked for, so that an answer which is no longer
	// the latest question is dropped rather than drawn.
	//
	// 🔴 Superseding rather than refusing, and the separator is why. It is
	// typed one character at a time and each character reads the whole file, so
	// a keystroke arriving during a read is ordinary. Refusing it would leave
	// the file read under one separator while the field on screen showed
	// another, which is the screen describing a file nobody has.
	asked int
	// unread is what the last attempt to read a file ended with, kept so that a
	// caller assembling this screen without a person can say what went wrong
	// rather than draw an empty screen and call it the state it asked for.
	unread error

	// What the screen is doing and what the last run left. The stage decides
	// what is drawn and the two beside it carry what there is to say about it.
	stage  stage
	result take.Result
	failed error
	// halt is how the run in flight is asked to end, and nothing while there is
	// none.
	halt *background.Stopper
	// releaseWatch stops noticing the system asking this program to end, and is
	// nothing outside a run. The watch covers the same stretch as the halt
	// above, because it is the same question asked by somebody other than the
	// person at the keyboard.
	releaseWatch func()
	// closing records that somebody asked for the window while a run was going
	// on, so that the window can be given to them once the run has stopped.
	closing bool
	// hadKeyboard records that the keyboard was on this screen when a run
	// began, so that the end of the run knows whether it owes anybody a place
	// to stand. Asked then rather than at the end, because by the end a person
	// working by keyboard and a person working by mouse look the same.
	hadKeyboard bool

	// The parts that change with what is on screen. They are kept rather than
	// rebuilt, because rebuilding a control throws away what somebody typed
	// into it and where the keyboard was standing.
	empty   *fyne.Container
	fileBox *fyne.Container
	// split holds the description apart from the settings, and is put away
	// while there is no file to describe. The two scrolling areas inside it are
	// kept because each of them is a place the keyboard can stand.
	split        *container.Split
	fileScroll   *parts.KeyScroll
	answerScroll *parts.KeyScroll

	settingsBox *fyne.Container
	// settingsWait says why the settings are dead, and stands only while they are.
	settingsWait *widget.Label
	planBox      *fyne.Container
	runBox       *fyne.Container
	// 🔴 The rules between the three boxes above are held rather than left in
	// the column, because a rule outlives what it divides. The first version of
	// the empty state hid every box and left the split showing, which drew two
	// separators across a half screen dividing nothing from nothing. Hiding a
	// box now hides the rule with it.
	answerRules []*widget.Separator
	// fileWait says what the left pane will hold, so that a person sees the
	// shape of this screen before choosing anything rather than a blank half.
	fileWait *widget.Label

	layers    *widget.CheckGroup
	share     *widget.Entry
	perRow    *parts.Select
	exclude   *widget.Entry
	delimiter *widget.Entry
	locale    *widget.Entry
	seed      *widget.Entry
	out       *widget.Entry
	manifest  *parts.Select
	overwrite *widget.Check

	chooseFile *parts.Button
	damage     *parts.Button
	stop       *parts.Button
}

// Clipboard is the one thing this screen puts outside itself that is not a
// file, and it is an interface so that a render can be handed nothing.
type Clipboard interface{ SetContent(string) }

// New builds the screen.
//
// The window is what a file chooser has to appear over and what a run takes
// over while it is going on. The work is where a run happens, where a file is
// read and how the system asking this program to stop reaches here, all handed
// in so that every state of this screen can be reached without a goroutine.
// The clipboard is handed in like everything else, and may be nothing at all.
// A screen built to be rendered rather than shown has no application behind it,
// and a picture that reached the clipboard of the machine taking it would
// change something outside itself.
func New(text i18n.Catalogue, window fyne.Window, described catalog.Catalog,
	openFile Opener, work background.Work, clipboard Clipboard) *View {
	view := &View{
		text: text, window: window, described: described, openFile: openFile,
		work: work, clipboard: clipboard, chosen: take.Defaults(),
		empty: container.NewVBox(), fileBox: container.NewVBox(),
		settingsBox: container.NewVBox(), planBox: container.NewVBox(),
		runBox: container.NewVBox(),
	}
	view.buildControls()
	view.sayNothingIsChosen()
	view.redraw()
	return view
}

// Content is the whole screen.
//
// The parts are handed over in the order a person reads them, which is also the
// order the keyboard walks them.
//
// 🔴 That sentence stood here before it was true. container.NewBorder appends
// its edges after its middle, so the convenient call put the whole of the right
// hand pane in front of the bar holding the button that chooses a file, and the
// button standing at the very top of the screen was the nineteenth and last
// thing the keyboard reached. Measured on 2026-08-24 by walking a real canvas.
// Nothing about the drawing said so, which is the same failure the manifest
// browser paid for on 2026-08-18 and the reason this comment was written in the
// first place. A comment is not a guard, so what holds it now is a test.
//
// The layout is therefore built with the objects named in reading order rather
// than through the call that reorders them, which is what the browser next door
// already does.
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
	v.answerRules = []*widget.Separator{widget.NewSeparator(), widget.NewSeparator()}
	answer := container.NewVBox(
		v.settingsBox,
		v.answerRules[0],
		v.planBox,
		v.answerRules[1],
		v.runBox,
	)
	// 🔴 Both panes scroll with the keyboard rather than with the toolkit's own
	// scrolling area, which answers to the wheel and to nothing else. This
	// screen is the taller of the two that had one: the description of a file
	// grows with the number of columns in it, and the settings are ten rows
	// whatever the window is. Measured on the About screen on 2026-08-24, where
	// the same absence left 73.6 percent of the content reachable only with a
	// mouse.
	//
	// The cost is stated rather than discovered: each of these is one more
	// place for the keyboard to stand, so the walk of this screen is two longer
	// than it was and the stored states that count presses of Tab moved with
	// it.
	v.fileScroll = parts.NewKeyScroll(v.fileBox)
	v.answerScroll = parts.NewKeyScroll(answer)
	split := container.NewHSplit(v.fileScroll.Framed(), v.answerScroll.Framed())
	v.split = split
	// The purpose comes before the control, because somebody opening this
	// window for the first time reads the name in the column, learns that the
	// screen is called Take and corrupt, and still does not know whether it is
	// about to write over the file they are about to choose.
	purpose := wrapped(v.text.T(i18n.PurposeTake))
	purpose.Importance = widget.LowImportance
	// 🔴 What the screen says before a file is chosen stays UNDER the button and
	// is not centred in the space below it. Tried on 2026-09-08 and turned down
	// by measurement, which is written here because the idea is an obvious one
	// and the next session will have it again.
	//
	// The screen with no file is three lines in the top left of a window the
	// owner runs at 1670 by 1210, so putting them in the middle of what is left
	// looks like the answer. Spacers above and below made it WORSE: rendered,
	// the first sentence came out as a label 1366 by 122 for a single line of
	// text 19 high, and the two sentences ended up 131 apart.
	//
	// The cause is that a wrapping label reports a MinSize computed against
	// whatever width it has at the time of asking, and in a column it is asked
	// before it has been given one. A column then reserves that wrong height.
	// It happens not to bite in the border edge above, because there the label
	// is given the width first.
	//
	// So vertical centring of wrapping text is not a thing to reach for here.
	// The emptiness on this screen is a shortage of CONTENT before a file is
	// chosen, and the answer to it is showing something, not moving three lines
	// into the middle of the nothing.
	body := container.NewVBox(purpose, container.NewHBox(v.chooseFile.OnScreen()), v.empty)
	// 🔴 Drawn again now that the split exists. The screen was first drawn while
	// this was still being built, so the state with no file had hidden every box
	// and left the split itself showing: two separator lines across an otherwise
	// empty half of the screen, dividing nothing from nothing. Seen in the
	// stored tree rather than by anybody looking, which is what that guard is
	// for.
	v.redraw()
	return parts.Inset(container.New(layout.NewBorderLayout(body, nil, nil, nil), body, split))
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
//
// # Why it answers nothing
//
// 🔴 It used to hand back the failure of the read, and it cannot any more,
// because the read no longer happens here. Reading a file this mode is for
// means walking all of it: measured at 11.7 seconds for a hundred thousand
// rows, which on the thread that draws is a window that stops repainting, stops
// answering the keyboard and offers nothing to do about it. So the read is
// handed to the place of work and what it found arrives later. What went wrong
// is on the screen, and Unread carries it for a caller with nobody looking.
func (v *View) Open(path string) {
	if v.busy() {
		// Nothing may take the file away from a run that is walking it. The
		// button that chooses one is disabled for the length of a run, and a
		// file dropped onto the window arrives by another road entirely, which
		// is why the rule lives here rather than on the control.
		return
	}
	v.startReading(path, func(opened *take.Opened, err error) {
		if err != nil {
			v.opened, v.path = nil, ""
			v.forgetLastRun()
			v.showUnreadable(err)
			return
		}
		if path != v.path {
			// A different file, so what the last run did is about something
			// else.
			v.forgetLastRun()
		}
		v.opened, v.path = opened, path
		v.redraw()
	})
}

// Unread is what the last attempt to read a file ended with, or nothing.
//
// It exists for the callers that assemble this screen with nobody looking at
// it. They ask for a state built on a file, and a screen that quietly drew the
// empty state instead would be a picture of something nobody asked for.
func (v *View) Unread() error { return v.unread }

// startReading hands a read to the place of work and shows that it is going on.
//
// # Why the answer is checked against a counter before it is used
//
// Reads are superseded rather than refused, because the separator is read again
// on every keystroke and a person types faster than a large file is walked.
// Without the counter the answer about the file as it was two keystrokes ago
// would arrive last and win, and the screen would describe a file read under a
// separator nobody has on screen. The counter is read and written only on the
// thread that draws, which is what lets it be a plain number.
func (v *View) startReading(path string, answer func(*take.Opened, error)) {
	v.asked++
	wanted := v.asked
	v.reading = path
	v.unread = nil
	v.drawReading()
	settings := v.chosen
	v.work.Read(func() func() {
		opened, err := v.openFile(path, settings)
		return func() {
			if wanted != v.asked {
				// A later read has already been asked for, so this answer is
				// about a question nobody is asking any more.
				return
			}
			v.reading = ""
			v.unread = err
			answer(opened, err)
		}
	}, func(err error) func() {
		// A read that came apart takes the same road as a read that failed,
		// with nothing opened. Every caller of this already has to handle a
		// failure with no file behind it, so this adds a cause rather than a
		// case.
		return func() {
			if wanted != v.asked {
				return
			}
			v.reading = ""
			v.unread = err
			answer(nil, err)
		}
	})
}

// readingNow reports whether a file is being read right now.
//
// It is apart from busy, which is about a run. The two never overlap, and they
// are asked about separately because they mean different things to a person: a
// run may be stopped and a read may not, and a run draws a way out where a read
// draws a sentence.
func (v *View) readingNow() bool { return v.reading != "" }

// drawReading says that a file is being read, in place of what is known about
// the last one.
//
// A sentence rather than nothing, which is point eight of the GUI checklist: an
// unchanged screen and a screen still working look identical, and the one thing
// a person cannot do is tell them apart.
func (v *View) drawReading() {
	// The name of the file first, because it is what tells a person which read
	// they are waiting for when they have just changed the separator on one
	// file after another.
	//
	// 🔴 The name and not the path, for two reasons that happen to agree. A
	// person knows their file by its name, and the whole path is a line of
	// noise around it. And the stored tree of this state would otherwise carry
	// the directory the sample was written into, which is a fresh temporary
	// directory on every run, so the guard comparing those trees would fail at
	// random. A guard that fails at random is a guard somebody turns off.
	said := []fyne.CanvasObject{
		wrapped(filepath.Base(v.reading)), wrapped(v.text.T(i18n.TakeReading)),
	}
	// Into whichever of the two is on screen, and never into both. The one that
	// is hidden keeps whatever it last said, and a stored tree of this screen
	// carries hidden objects as well as shown ones, so a sentence left in the
	// wrong box would show up there as a screen saying two things at once.
	if v.opened == nil {
		v.empty.Objects = said
		v.empty.Refresh()
	} else {
		v.fileBox.Objects = said
		v.fileBox.Refresh()
	}
	// The button that would start a run goes with it. A run begun now would use
	// the file as it was read a moment ago, under settings the screen has since
	// been told to change, and would say so in a manifest.
	v.drawRun()
}

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
	v.Open(dropped[0].Path())
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
		v.Open(chosen.URI().Path())
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
		v.showWaiting()
		return
	}
	v.empty.Hide()
	v.settingsWait.Hide()
	v.fileWait.Hide()
	for _, rule := range v.answerRules {
		rule.Show()
	}
	// Given back, because the state above took them away. Guarded by the run
	// in flight rather than left to where this happens to be called from: no
	// caller today redraws during a run, and a rule that holds only while
	// nobody adds a caller is not a rule. Thawing mid run would put the
	// settings of the next run under the fingers of somebody watching this one.
	if !v.busy() {
		v.thaw()
	}
	// Put back what the empty box says, because a read that has just answered
	// left the sentence about reading in it. It is hidden from here on, so this
	// is not about what anybody sees: it is about the box holding what it
	// means, so that the next time it is shown it is not a sentence about a
	// file that was read minutes ago.
	v.sayNothingIsChosen()
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

// sayNothingIsChosen is what the screen says before anybody has chosen a file.
func (v *View) sayNothingIsChosen() {
	v.empty.Objects = []fyne.CanvasObject{
		wrapped(v.text.T(i18n.TakeEmpty)), wrapped(v.text.T(i18n.TakeDrop)),
	}
	v.empty.Refresh()
}

// hideSplit puts away everything that describes a file, for the state where
// there is none.
func (v *View) hideSplit() {
	v.fileBox.Hide()
	v.settingsBox.Hide()
	v.planBox.Hide()
	v.runBox.Hide()
	for _, rule := range v.answerRules {
		rule.Hide()
	}
	if v.split != nil {
		v.split.Hide()
	}
}

// showWaiting is the screen before a file is chosen: the shape of it, with the
// settings in place and dead, and each half saying what will stand there.
//
// 🔴 # Why the whole shape rather than a bare button
//
// Measured on 2026-09-08 with tools/pustka.py at the size the window is kept:
// this screen stood 64.0 per cent empty in the state it OPENS in, and the state
// a program opens in is the one no drawing shows and the first a person meets.
// What it said about this tool was that the tool does almost nothing.
//
// Nothing here is new content invented to fill space. The settings existed and
// were hidden, and what the two sentences add is the name of what each half is
// for, which is the question somebody opening this screen actually has.
//
// # What stays away, and why it is not the same answer
//
// The plan and the result describe a PARTICULAR file: what a run would do to it
// and what a run of it produced. With no file each is a heading over nothing,
// which is the box somebody forgot to fill rather than a screen that is
// waiting. The settings differ because they are true before there is a file:
// they are what a run WOULD use, and they are what somebody wants to set while
// looking for one.
func (v *View) showWaiting() {
	if v.split == nil {
		// Called once from New, before Content has built the split. There is
		// nothing on screen yet to put into a state.
		v.hideSplit()
		return
	}
	v.split.Show()
	v.fileBox.Show()
	v.fileWait.Show()
	v.settingsBox.Show()
	v.settingsWait.Show()
	v.planBox.Hide()
	v.runBox.Hide()
	for _, rule := range v.answerRules {
		rule.Hide()
	}
	v.previewSettings()
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

// heading names one part of this screen, with the rule the accepted drawing
// puts under it.
//
// Five of them, and every one is a section rather than a line of emphasis: what
// the file holds, what to damage, what would happen, the result, and the
// terminal line. Until 2026-09-08 they were bold text and nothing else, so
// nothing said where one part of this screen stopped and the next began, which
// on the busiest screen in the window is where it hurts most.
func heading(words string) fyne.CanvasObject {
	return parts.Section(words)
}

// counted writes a number with the thing it counts, in the singular where there
// is one of it.
func counted(many int, thing string) string {
	if many == 1 {
		return "1 " + thing
	}
	return strconv.Itoa(many) + " " + thing + "s"
}
