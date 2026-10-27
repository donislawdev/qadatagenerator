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
//
// # The shape, since 2026-09-15 (DEC-245)
//
// A band at the top holds the file, how it is read and the one action of the
// screen. The body is two columns: what to damage on the left, and on the
// right the one card, saying what a run would do or did, over the panel
// saying what the file holds. A band at the foot holds the command that does
// the same in a terminal. It is the shape of the Quick screen, because the two
// are one program, and the settings went into a column rather than the band
// because there are ten of them: the band with all ten took half the window
// at the size the owner keeps it, measured on the mock before a line of this
// was written.
//
// 🔴 What the shape repairs is measured rather than felt. The old screen put
// the action, the state of a run and the result under the fold of a scrolling
// pane at that size (B-280), so a person pressed a button they could not see
// and then saw nothing happen.
package takeview

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// Opener reads somebody else's file and says what it holds.
//
// It is handed in for the same reason the manifest browser takes its readers
// that way: every state of this screen then has a way in that needs no disk,
// including the states worth checking, which are the ones nobody wants to keep
// a file around for.
//
// stale says that the screen has asked a newer question, and the reading ends
// at the next record once it does.
type Opener func(path string, settings take.Settings, stale corrupt.Halt) (*take.Opened, error)

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
	// command is that line, and the button beside it copies it and says, for a
	// moment, that it did.
	command     *widget.Label
	copyCommand *parts.Button
	terminal    fyne.CanvasObject
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
	// reads is the queue every read of this screen goes through, so that a
	// separator typed during a read stops it and waits for it rather than
	// walking the file beside it. Each character reads the whole file, and
	// before 2026-10-27 every one of them did so at once. The reasons stand on
	// background.Latest, which also drops the answer to a read asked again.
	//
	// 🔴 Superseding rather than refusing, and the separator is why. It is
	// typed one character at a time, so a keystroke arriving during a read is
	// ordinary. Refusing it would leave the file read under one separator while
	// the field on screen showed another, which is the screen describing a file
	// nobody has.
	reads *background.Latest
	// unread is what the last attempt to read a file ended with, kept so that a
	// caller assembling this screen without a person can say what went wrong
	// rather than draw an empty screen and call it the state it asked for.
	// tried is the file that attempt was about, so that the band can name it
	// after the path has been let go of.
	unread error
	tried  string

	// What the screen is doing and what the last run left. The stage decides
	// what is drawn and the two beside it carry what there is to say about it.
	stage  stage
	result take.Result
	failed error
	// halt is how the run in flight is asked to end, and nothing while there is
	// none.
	halt *background.Stopper
	// letGo gives back the window held for the run in flight, and is nothing
	// outside a run. It covers the same stretch as the halt above, because the
	// window and the system ask the same question as the person at the
	// keyboard.
	letGo func()
	// keys moves the keyboard off the button a run takes away and back when
	// the run ends. Shared with the recipe screen since 2026-09-15, and the
	// reasons stand on the type.
	keys parts.Handover

	// The parts that are refilled with what is on screen. They are kept rather
	// than rebuilt, because rebuilding a control throws away what somebody
	// typed into it and where the keyboard was standing.
	//
	// facts is the slot in the band beside the button: the name of the file
	// and its figures, or the sentence about there being none. action holds
	// the button that damages and the one that stops, of which one is shown.
	// planBox and runBox are the two parts of the one card, and fileBox is
	// what the panel holds, inside the scrolling area the keyboard can reach.
	facts       *fyne.Container
	action      *fyne.Container
	cardHeading *fyne.Container
	planBox     *fyne.Container
	runBox      *fyne.Container
	fileBox     *fyne.Container
	fileScroll  *parts.KeyScroll
	// panel is the slot the panel stands in: hugging its sentence while there
	// is no file, and filling the column with a scrolling area once there is
	// a description to scroll.
	panel   *fyne.Container
	content fyne.CanvasObject

	layers    kindBoxes
	share     *parts.FieldEntry
	perRow    *parts.Select
	exclude   *parts.FieldEntry
	delimiter *parts.FieldEntry
	locale    *parts.FieldEntry
	seed      *parts.FieldEntry
	out       *parts.FieldEntry
	manifest  *parts.Select
	overwrite *parts.Check
	// options folds away the settings a first run does not need (DEC-261).
	options *parts.Disclosure

	chooseFile *parts.Button
	// chooseOut asks the system for the directory the copy goes to, beside the
	// field a path can be typed or pasted into.
	chooseOut *parts.Button
	damage    *parts.Button
	stop      *parts.Button
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
		facts: container.NewStack(), action: container.NewStack(),
		cardHeading: container.NewStack(),
		planBox:     container.NewStack(), runBox: container.NewStack(),
		fileBox: container.NewStack(), panel: container.NewStack(),
		reads: background.NewLatest(work.Read),
	}
	view.buildControls()
	view.lay()
	view.redraw()
	return view
}

// Content is the whole screen.
func (v *View) Content() fyne.CanvasObject { return v.content }

// lay puts the controls on the screen, apart from making them.
//
// 🔴 Every layout here is given its objects in reading order, which is the
// order the keyboard walks them. container.NewBorder appends its edges after
// its middle, so the convenient call once put the whole right hand pane in
// front of the bar holding the button that chooses a file, and the button at
// the very top of the screen was the nineteenth and last thing the keyboard
// reached. Measured on 2026-08-24 by walking a real canvas, and held by a test
// since.
func (v *View) lay() {
	v.fileScroll = parts.NewKeyScroll(v.fileBox)
	v.terminal = v.commandBand()

	// The panel hugs the width of an aside and takes the height the card
	// leaves, scrolling inside it: what a file holds grows with the number of
	// its columns and the window does not.
	// 🔴 ONE heading object per border, named and given to the layout twice.
	// A border finds its edges by COMPARING the objects it was given, so a
	// heading built twice would be laid out as a second centre and drawn
	// across the card. The quick screen paid for that with a button once.
	// The card shows the plan of the next run OR what the last one did,
	// never both: together they were taller than the column at the size the
	// owner keeps the window, and the panel under them was handed a negative
	// height, which the render's rasteriser refuses with a panic. The heading
	// over the card says which of the two it is, so it is refilled with the
	// card rather than built once.
	card := paint.Card(palette.NameStepGroup,
		paint.Column(palette.NameStepRelated, v.planBox, v.runBox))
	result := container.New(layout.NewBorderLayout(v.cardHeading, nil, nil, nil),
		v.cardHeading, container.NewVBox(card))
	right := container.New(layout.NewBorderLayout(result, nil, nil, nil),
		result, paint.Pad(palette.NameStepSection, palette.NameStepHairline,
			palette.NameStepHairline, palette.NameStepHairline,
			paint.AtLeast(palette.AsideWidth, 0, v.panel)))

	// The steps scroll inside their column, because opening the settings a
	// first run does not need makes them taller than the window the owner
	// keeps, and a column that grew instead would push the window's minimum
	// past its size.
	// Framed, so that the keyboard standing on the column shows a ring like
	// every other scrolling area of the window does.
	steps := parts.NewKeyScroll(v.stepsColumn()).Framed()

	aside := paint.PadLeft(palette.NameStepSection, right)
	body := paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepMajor, palette.NameStepMajor,
		container.New(layout.NewBorderLayout(nil, nil, nil, aside), steps, aside))

	band := v.purposeBand()
	v.content = container.New(layout.NewBorderLayout(band, v.terminal, nil, nil),
		band, body, v.terminal)
}

// heading is the title over the card, which says whether it holds the plan of
// the next run or what the last one did.
func (v *View) heading(key i18n.Key) fyne.CanvasObject {
	return paint.Pad(palette.NameStepHairline, palette.NameStepRelated,
		palette.NameStepHairline, palette.NameStepHairline,
		paint.SectionHeading(v.text.T(key), paint.Spacer()))
}

// purposeBand is the header under the name of the screen: what the screen is
// for, and nothing else.
//
// # Why the file and the button left it on 2026-10-21 (DEC-261)
//
// The band held the file, how it is read and the one action of the screen, in
// its top right corner. A person seeing the window for the first time read the
// band as the header it looks like and never found the button in it, and the
// action stood above the settings it acts on. The steps under it now carry all
// three, in the order they are done.
func (v *View) purposeBand() fyne.CanvasObject {
	// The sentence wraps, because it is the longest line on this screen and a
	// piece of canvas text would otherwise decide how narrow the window may be.
	return paint.Band(paint.BandBelow, palette.NameStepGroup, palette.NameStepMajor,
		v.paragraph(v.text.T(i18n.PurposeTake)))
}

// stepsColumn is the three steps of the screen, read from the top down: the
// file, what to damage, where the damaged copy goes and the button that writes
// it.
func (v *View) stepsColumn() fyne.CanvasObject {
	// 🔴 The facts take what the button leaves, rather than standing in a row
	// with it. In a row they were given the width they asked for, and the
	// sentence about there being no file asked for more than there was - seen
	// on the first render of the state the screen opens in, when the band held
	// it.
	facts := paint.AtLeast(0, palette.ControlHeight, paint.Middle(v.facts))
	file := paint.Lead(v.chooseFile.OnScreen(), paint.PadLeft(palette.NameStepRelated, facts))
	damage := paint.Column(palette.NameStepRelated, v.kindsGrid(), v.options.OnScreen())
	// The button stands at its own width at the start of the line, where the
	// eye arrives after the last setting, rather than stretched across it.
	write := paint.Column(palette.NameStepRelated,
		paint.Trail(labelled(v.text.T(i18n.TakeSettingOut), parts.Standing(parts.Field(v.out))),
			paint.PadLeft(palette.NameStepRelated, paint.Unlabelled(v.chooseOut.OnScreen()))),
		paint.AtLeast(0, palette.ControlHeight, v.overwrite.OnScreen()),
		paint.Lead(v.action, paint.Spacer()))
	// Room on the right for the bar of the scrolling area, and a hairline on
	// the left for the edge of a field, which is drawn outside its bounds and
	// was cut by the edge of the scrolling area on the first render.
	return paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
		palette.NameStepHairline, palette.NameStepGroup,
		paint.Steps(palette.NameStepMajor,
			paint.Step{Words: v.text.T(i18n.TakeStepFile), Body: file},
			paint.Step{Words: v.text.T(i18n.TakeHeadingSettings), Body: damage},
			paint.Step{Words: v.text.T(i18n.TakeStepWrite), Body: write}))
}

// chooseDirectory asks for the directory the damaged copy goes to, the way the
// recipe screen asks for the one its dataset goes to.
func (v *View) chooseDirectory() {
	if v.window == nil || v.busy() {
		return
	}
	dialog.ShowFolderOpen(func(chosen fyne.ListableURI, err error) {
		if err != nil || chosen == nil {
			return
		}
		v.out.SetText(chosen.Path())
	}, v.window)
}

// Open reads a file and puts the screen into the state that shows it.
//
// # Why this is exported
//
// Something outside this package has to reach that state without a file
// chooser: the guard holding what every screen draws, and the probe rendering
// them. It carries the whole state change rather than the drawing alone, which
// is the correction of a drift this project measured on the other screen,
// where an exported call that only drew reached a state no person can.
//
// # Why it answers nothing
//
// 🔴 It used to hand back the failure of the read, and it cannot any more,
// because the read no longer happens here. Reading a file this mode is for
// means walking all of it, about 0.6 seconds for three million rows measured
// on 2026-10-27 and longer for a larger file, which on the thread that draws
// is a window that stops repainting, stops answering the keyboard and offers
// nothing to do about it. So the read is handed to the place of work and what
// it found arrives later. What went wrong is on the screen, and Unread carries
// it for a caller with nobody looking.
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
// 🔴 The answer is applied only if it is still the answer to the latest
// question. The queue keeps one read in flight, and a separator typed while it
// goes on makes that read an answer to a question nobody is asking any more:
// the field already shows the next character, and the read about it is the one
// waiting. The queue tells the read in flight to stop and drops its answer.
// Until 2026-10-27 this screen kept a count of its own for the dropping, and
// from that day the count could never drop anything the queue had not, because
// every read it counted was a question asked of the queue.
func (v *View) startReading(path string, answer func(*take.Opened, error)) {
	v.reading, v.tried = path, path
	v.unread = nil
	v.drawReading()
	settings := v.chosen
	v.reads.Ask(func(stale *background.Stopper) func() {
		opened, err := v.openFile(path, settings, stale)
		return func() {
			v.reading = ""
			v.unread = err
			answer(opened, err)
		}
	}, func(err error) func() {
		// The place of work could not even start the read, which is a
		// failure of the same kind as a file that cannot be read, and is
		// shown the same way.
		return func() {
			v.reading = ""
			v.unread = err
			answer(nil, err)
		}
	})
}

// readingNow reports whether a file is being read.
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
	v.drawFacts()
	v.planBox.Objects = []fyne.CanvasObject{v.paragraph(v.text.T(i18n.TakeReading))}
	v.planBox.Refresh()
	v.drawRun()
}

// drawFacts fills the slot beside the button with what is known about the
// file: its name and figures, that it is being read, that it could not be, or
// that there is none yet.
//
// 🔴 The name and not the path, for two reasons that happen to agree. A person
// knows their file by its name, and the whole path is a line of noise around
// it. And the stored tree of this state would otherwise carry the directory
// the sample was written into, which is a fresh temporary directory on every
// run, so the guard comparing those trees would fail at random. A guard that
// fails at random is a guard somebody turns off.
func (v *View) drawFacts() {
	var said fyne.CanvasObject
	switch {
	case v.readingNow():
		said = v.named(v.reading, v.text.T(i18n.TakeFileReading), palette.TextSecondary)
	case v.unread != nil:
		said = v.named(v.tried, v.text.T(i18n.TakeFileUnreadable), palette.StateDanger)
	case v.opened == nil:
		// A label cut with an ellipsis rather than a piece of canvas text,
		// because this is a sentence and the slot is whatever the band has
		// left: a piece of canvas text would ask for its whole width and take
		// it from the fields beside it.
		said = v.cut(v.text.T(i18n.TakeFileNone))
	default:
		said = v.named(v.path, v.shape(), palette.TextSecondary)
	}
	v.facts.Objects = []fyne.CanvasObject{said}
	v.facts.Refresh()
}

// named is the name of a file with a few words after it.
func (v *View) named(path, words, ink string) fyne.CanvasObject {
	return paint.Row(palette.NameStepTight,
		paint.Strong(filepath.Base(path), palette.TypeBody, palette.TextPrimary),
		paint.Text(dot+" "+words, palette.TypeBody, ink))
}

// dot stands between two facts about a file, as it does between the version
// and the promise at the foot of the sidebar.
const dot = "·"

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
			// The dialog was dismissed, which is a person changing their
			// mind rather than anything to say.
			return
		}
		defer func() { _ = chosen.Close() }()
		v.Open(chosen.URI().Path())
	}, v.window)
}

// showUnreadable puts the failure where the plan would have been, and says in
// the band that the file could not be read.
func (v *View) showUnreadable(err error) {
	v.unread = err
	v.redraw()
	v.planBox.Objects = []fyne.CanvasObject{paint.Column(palette.NameStepTight,
		v.paragraph(v.text.T(i18n.TakeUnreadable)+" "+err.Error()),
		v.paragraph(v.text.T(i18n.TakeDrop)))}
	v.planBox.Refresh()
}

// redraw draws the parts that belong to the state the screen is in.
//
// The controls are refilled rather than rebuilt, so that a control keeps what
// was typed into it and the keyboard keeps its place.
func (v *View) redraw() {
	v.drawFacts()
	v.drawFile()
	v.drawCommand()
	if v.opened == nil {
		v.planBox.Objects = []fyne.CanvasObject{v.paragraph(v.text.T(i18n.TakePlanWait))}
		v.planBox.Refresh()
		v.drawRun()
		return
	}
	// The controls come back to life after a run. Nothing else touches them,
	// since every setting is alive from the moment the screen opens.
	if !v.busy() {
		v.thaw()
	}
	v.drawPlan()
	v.drawRun()
}

// labelled names a control, above it rather than beside it.
//
// It is one call to the drawing next door, named here so that a row of
// settings reads as a list of settings rather than as layout instructions.
func labelled(name string, control fyne.CanvasObject) fyne.CanvasObject {
	return paint.FieldLabel(name, control)
}

// cut is a sentence that takes the width it is given and ends in an ellipsis
// where it does not fit, for the one slot of this screen that is a single
// line beside other things.
func (v *View) cut(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapOff
	label.Truncation = fyne.TextTruncateEllipsis
	label.Importance = widget.LowImportance
	label.SizeName = fyne.ThemeSizeName(palette.TypeBody)
	return label
}

// paragraph is a sentence that wraps to the width it is given, in the type of
// the body and the ink of an explanation. It is the drawing next door, named
// here so that the card reads as what it says.
func (v *View) paragraph(words string) fyne.CanvasObject {
	return paint.Paragraph(words)
}

// notice is a sentence with the mark of a warning before it, for the things a
// person is better off knowing before they press the button.
func (v *View) notice(key i18n.Key) fyne.CanvasObject {
	return v.marked(paint.IconWarn, palette.StateWarning, v.paragraph(v.text.T(key)))
}

// marked puts an icon before something, level with its first line. It is the
// drawing next door, named here so that the card reads as what it says.
func (v *View) marked(icon, ink string, words fyne.CanvasObject) fyne.CanvasObject {
	return paint.Marked(icon, ink, words)
}

// shape says how many records and columns the file that was read holds. The
// band and the panel both say it, so it is written once.
//
// Until 2026-09-24 the two words were written in the code and made plural by
// adding a letter, which is text a translation could never reach and the rule
// the recipe screen was bitten by with "2 entitys" (B-294).
func (v *View) shape() string {
	table := v.opened.Table()
	return v.text.Counted(table.Rows, i18n.TakeCountRecord, i18n.TakeCountRecords) + " " + dot + " " +
		v.text.Counted(len(table.Columns), i18n.TakeCountColumn, i18n.TakeCountColumns)
}
