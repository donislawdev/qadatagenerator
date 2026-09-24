package manifestview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Readers are the two documents this screen shows, as functions rather than as
// a dependency on any particular format.
//
// # Why they are handed in
//
// The screen then has no opinion about files at all, so every state of it can
// be reached in a test by handing in a reader that answers whatever the test
// needs, including a failure. A screen that opened its own files could only
// ever be checked against files that exist on a disk somewhere, and the states
// worth checking are exactly the ones nobody wants to keep on a disk.
type Readers struct {
	Manifest func(path string) (manifest.Manifest, error)
	Response func(path string) (response.Record, error)
}

// Clipboard is the one thing this screen puts outside itself that is not a
// file, and it is an interface so that a render can be handed nothing.
type Clipboard interface{ SetContent(string) }

// View is the manifest browser and what it currently holds.
//
// Nothing is loaded until somebody asks. That is the window half of DEC-35: a
// surface that reads a file the moment it is opened is a surface that acts on
// its own, and this tool writes to other people's databases.
//
// # The shape, since 2026-09-15 (DEC-247) and 2026-09-23 (DEC-249)
//
// A band at the top holds the two files, one under the other: the manifest,
// which is the one action of the screen and the only filled button, and the
// response record, which cannot be opened before it. The body is the strip a
// comparison adds about the whole file, the table of entries taking every unit
// of height the rest leaves, and under it one card saying one thing at a time:
// the entry somebody clicked or moved the keyboard onto, or the envelope of the
// run, with a heading over it saying which. A band at the foot holds the line
// that does the same comparison in a terminal.
//
// 🔴 The envelope stood in a section over the table until 2026-09-23, and the
// real window refused it: opened beside a comparison it asked for 1423 pixels
// of height where the owner's screen has 1392 to give (B-286). In the card it
// takes the height an entry takes, which is fixed, so opening it cannot grow
// the window at all.
type View struct {
	text      i18n.Catalogue
	window    fyne.Window
	read      Readers
	clipboard Clipboard
	// load is where a document is read, so that the thread which draws is free
	// while it happens.
	//
	// 🔴 It was read where the chooser answered until 2026-09-01, which is on
	// the thread that draws. This screen reads manifests of up to 48 MB and
	// then compares one against a record of what an application did, so the
	// window stopped repainting and stopped answering the keyboard for the
	// length of both. The file next door explains at length why a run may not
	// happen there. Nothing had applied the same sentence to a read.
	load background.Read

	// The band: the two buttons, kept because a read puts them out of reach
	// and something has to hold them to do it, and the slot of facts beside
	// each, refilled with what is known about the file.
	open          *parts.Button
	respond       *parts.Button
	manifestFacts *fyne.Container
	responseFacts *fyne.Container

	// The body: the strip of notes a comparison adds over the table, the tally
	// at the heading of the table, the slot the table or the shape of one
	// stands in, and the card under it with its heading, the button in that
	// heading which puts the run in the card and takes it out again, and the
	// title the heading was last drawn with.
	notesBox    *fyne.Container
	tally       *fyne.Container
	entriesBox  *fyne.Container
	entries     *parts.KeyTable
	cardHeading *fyne.Container
	about       *parts.Button
	cardTitle   i18n.Key
	detailBox   *fyne.Container
	detail      *parts.KeyScroll
	content     fyne.CanvasObject

	// The band at the foot: the terminal line, the button that copies it, the
	// mark saying that it did, and the sentence saying which part of the line
	// is an example.
	terminal      fyne.CanvasObject
	command       *widget.Label
	copyCommand   *parts.Button
	commandCopied *canvas.Text
	commandNote   fyne.CanvasObject

	// What is on the screen: the manifest, the response beside it, and the
	// names of the files they came from, which are empty for a document that
	// arrived already parsed.
	screen       Screen
	promised     manifest.Manifest
	hasPromise   bool
	hasResponse  bool
	manifestPath string
	responsePath string
	answered     int

	// reading is the file being read right now and which of the two it is,
	// and nothing while none is. It is a state of its own because point eight
	// of the GUI checklist asks for a sentence rather than an unchanged
	// screen: a window that has not answered yet and one that has nothing to
	// say look alike, and telling them apart is the one thing a person cannot
	// do. unread is what the last attempt to read ended with, and tried the
	// file it was about, so that the band can name it after the path has been
	// let go of.
	reading     string
	readingWhat document
	unread      error
	tried       string
	triedWhat   document
	// asked counts the reads, so that an answer which is no longer the latest
	// question is dropped rather than drawn over a newer one.
	asked int

	// Which entry the card reads out: the last one chosen, by a click or by the
	// keyboard moving onto it, and never one the pointer merely passed over
	// (DEC-249). The cell the keyboard stands on is kept beside it, because it
	// is worth marking only while the table is what the keyboard is talking to.
	selected     int
	hasSelected  bool
	highlight    widget.TableCellID
	hasHighlight bool
	tableFocused bool
	// showingRun is the card saying what the run was rather than reading out
	// an entry. Choosing an entry ends it, and so does the button that
	// started it.
	showingRun bool
}

// document says which of the two files a read or a failure was about.
type document int

const (
	theManifest document = iota
	theResponse
)

// New builds the screen.
//
// The window is what a file chooser has to appear over, the place of work is
// where a document is read, and the clipboard is where the terminal line goes,
// or nothing for a screen built to be rendered rather than shown.
func New(text i18n.Catalogue, window fyne.Window, read Readers, load background.Read,
	clipboard Clipboard) *View {
	view := &View{
		text: text, window: window, read: read, load: load, clipboard: clipboard,
		manifestFacts: container.NewStack(), responseFacts: container.NewStack(),
		notesBox: container.NewStack(), tally: container.NewStack(),
		entriesBox: container.NewStack(), cardHeading: container.NewStack(),
		detailBox: container.NewStack(),
	}
	view.buildBand()
	view.buildCommand()
	view.lay()
	view.draw()
	return view
}

// Content is the whole screen.
func (v *View) Content() fyne.CanvasObject { return v.content }

// lay puts the parts on the screen, apart from making them.
//
// 🔴 Every layout here is given its objects in reading order, which is the
// order the keyboard walks them. container.NewBorder appends its edges after
// its middle, and measured on 2026-08-18 that put the table in front of the
// button above it. Three screens have paid for the same walk since.
func (v *View) lay() {
	heading := headed(v.text.T(i18n.ManifestHeadingEntries), paint.Middle(v.tally))
	middle := container.New(layout.NewBorderLayout(heading, nil, nil, nil), heading, v.entriesBox)

	// The card under the table is a fixed height, so that reading out one
	// entry after another never moves the foot of the table, and scrolls
	// inside, because a value somebody damaged can be as long as they like.
	// The envelope of the run takes the same card and the same height, which
	// is what keeps the window from growing when it is opened (DEC-249).
	//
	// 🔴 The heading is refilled as the card changes what it says, and the
	// button in it is one object moved from one heading to the next, so that a
	// keyboard standing on it is still standing on it after the press.
	v.about = parts.NewButton(v.text.T(i18n.ManifestHeadingRun), v.toggleRun)
	v.about.Plain = true
	v.detail = parts.NewKeyScroll(v.detailBox)
	card := paint.AtLeast(0, palette.EntryDetailHeight,
		paint.Card(palette.NameStepGroup, v.detail.Framed()))
	foot := container.New(layout.NewBorderLayout(v.cardHeading, nil, nil, nil), v.cardHeading, card)

	body := paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepMajor, palette.NameStepMajor,
		container.New(layout.NewBorderLayout(v.notesBox, foot, nil, nil), v.notesBox, middle, foot))

	band := v.fileBand()
	v.terminal = v.commandBand()
	v.content = container.New(layout.NewBorderLayout(band, v.terminal, nil, nil),
		band, body, v.terminal)
}

// Open puts a manifest on the screen, with everything that goes with it.
//
// # Why this is exported, and why it is not merely a draw
//
// Something outside this package has to be able to reach the states of this
// screen without a file chooser, which is what the rendering probe in tools
// does. The readers were already injectable, so this was the one step that
// could not be reached from outside.
//
// It carries the whole state change rather than the drawing alone, and that is
// the correction of a real drift measured on 2026-08-18: an exported call that
// only drew left the button for a response record disabled, so the probe
// rendered a screen no person can reach by opening a manifest. Two paths into
// one state had come apart within the hour. The chooser calls this too.
//
// The path is where the manifest came from, for the band and the terminal
// line, and may be empty for a document that arrived already parsed.
func (v *View) Open(promised manifest.Manifest, path string) {
	v.promised, v.hasPromise, v.manifestPath = promised, true, path
	v.hasResponse, v.responsePath, v.answered = false, "", 0
	v.screen = Build(v.text, promised, nil)
	v.respond.Enable()
	v.forgetSelection()
	v.entries = nil
	v.draw()
}

// Compare puts the answer of an application beside the manifest already open.
//
// A record without a manifest to judge it against is refused rather than drawn,
// because the comparison has nothing to say and a screen showing it would be
// saying it anyway.
//
// 🔴 # Why the chosen entry survives
//
// The entries are the same entries: the manifest did not change, and a
// comparison adds what the application did to each of them. Until 2026-09-23
// this let go of the entry the card was reading out and built the table
// again, so a person who had chosen an entry to see what their application did
// about it found the card back at the sentence asking them to choose one, and
// the table back at its top (B-288, rule 3 of the window: nothing resets on a
// refresh). The table is refreshed in place instead, so the selection, the cell
// the keyboard stands on and how far it was scrolled all stay where they were.
func (v *View) Compare(actual response.Record, path string) bool {
	if !v.hasPromise {
		return false
	}
	comparison := Comparison{Record: actual, Report: verify.Compare(v.promised, actual)}
	v.screen = Build(v.text, v.promised, &comparison)
	v.hasResponse, v.responsePath, v.answered = true, path, len(actual.Rows)
	v.draw()
	return true
}

// ShowRun puts the envelope of the run in the card, which is what the button in
// the card's heading does, and it is exported for a picture of that state.
//
// The table lets go of its selection while the card is about the run. The
// entry stays chosen here, which is what the button that brings it back reads
// out, but a click on the very entry that was selected would otherwise be
// ignored by the toolkit - it selects nothing that is already selected - and
// the card would go on saying what the run was under a person clicking an
// entry to read it.
func (v *View) ShowRun() {
	v.showingRun = true
	if v.entries != nil {
		v.entries.UnselectAll()
	}
	v.detail.ScrollToTop()
	v.drawDetail()
}

// showEntry takes the run out of the card and puts back the entry chosen before
// it, selected again in the table so that the violet and the card agree.
func (v *View) showEntry() {
	v.showingRun = false
	if v.entries != nil && v.hasSelected {
		column := 0
		if v.hasHighlight && v.highlight.Row == v.selected {
			column = v.highlight.Col
		}
		v.entries.Select(widget.TableCellID{Row: v.selected, Col: column})
	}
	v.detail.ScrollToTop()
	v.drawDetail()
}

// toggleRun is the press of the button in the card's heading.
func (v *View) toggleRun() {
	if v.showingRun {
		v.showEntry()
		return
	}
	v.ShowRun()
}

// draw puts on the screen everything that depends on its state.
//
// The parts are refilled rather than rebuilt, so that a control keeps where the
// keyboard was standing. The table is the exception: it is built from the rows
// of a manifest, so it is built again when a manifest is opened and at no other
// time. A comparison refreshes it in place.
func (v *View) draw() {
	v.drawBand()
	v.drawNotes()
	v.drawTally()
	v.drawEntries()
	v.drawDetail()
	v.drawCommand()
}

// forgetSelection lets go of the entry the card was reading out, because the
// rows it belonged to are being replaced.
func (v *View) forgetSelection() {
	v.selected, v.hasSelected = 0, false
	v.hasHighlight, v.tableFocused = false, false
}
