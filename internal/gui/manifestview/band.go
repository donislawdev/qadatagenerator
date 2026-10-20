package manifestview

import (
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The band along the top: the two files, one under the other.
//
// # Why two rows rather than one
//
// One row was drawn first and the render at the size the owner keeps the
// window refused it: the two buttons with a name and a few facts beside each
// asked for 1244 units of 1113, and the second button was drawn over the facts
// of the first. Two rows cost 46 units of height and give the names room, and
// names of files are long (contacts.corrupted.manifest.json is one this tool
// writes itself). They also read as the two steps they are: a manifest first,
// then what an application did about the dataset it describes.
//
// Both buttons are as wide as the wider label, so that the facts beside them
// start in one column.

// buildBand makes the controls once.
//
// Once rather than on each draw, because a control rebuilt under somebody is a
// control that loses where their keyboard was standing.
func (v *View) buildBand() {
	// The filled button carries no icon, as no filled button in this window
	// does: its fill is what marks it out, and it stands 4.99 against the
	// ground of the window.
	v.open = parts.NewPrimaryButton(v.text.T(i18n.ManifestOpen), v.chooseManifest)
	v.respond = parts.NewButton(v.text.T(i18n.ManifestOpenResponse), v.chooseResponse)
	v.respond.ShowIcon(paint.IconChecklist)
	// Nothing to compare a response against until a manifest is open, and a
	// control that cannot do anything yet says so by being unavailable rather
	// than by failing after the fact. The words beside it say why.
	v.respond.Disable()
}

// fileBand is the band: what the screen is for, then the two files.
func (v *View) fileBand() fyne.CanvasObject {
	// The two buttons measured rather than a width chosen for them, so that a
	// translation with a longer label widens both.
	width := v.open.MinSize().Width
	if wider := v.respond.OnScreen().MinSize().Width; wider > width {
		width = wider
	}
	// Each row leads with the number of its step (DEC-261), so that the order
	// is said before a word is read: a manifest first, then the answer.
	row := func(step int, button, facts fyne.CanvasObject) fyne.CanvasObject {
		return paint.Lead(paint.Middle(paint.StepMark(step)),
			paint.PadLeft(palette.NameStepTight,
				paint.Lead(paint.AtLeast(width, 0, button),
					paint.PadLeft(palette.NameStepRelated,
						paint.AtLeast(0, palette.ControlHeight, paint.Middle(facts))))))
	}
	// Numbered by their place in the list, for the reason paint.Steps gives.
	files := []struct{ button, facts fyne.CanvasObject }{
		{v.open, v.manifestFacts},
		{v.respond.OnScreen(), v.responseFacts},
	}
	rows := make([]fyne.CanvasObject, 0, len(files))
	for at, file := range files {
		rows = append(rows, row(at+1, file.button, file.facts))
	}
	// The sentence wraps, because it is the longest line on this screen and a
	// piece of canvas text would otherwise decide how narrow the window may be.
	return paint.Band(paint.BandBelow, palette.NameStepGroup, palette.NameStepMajor,
		paint.Column(palette.NameStepGroup,
			v.paragraph(v.text.T(i18n.PurposeManifest)),
			paint.Column(palette.NameStepTight, rows...)))
}

// drawBand fills the two slots of facts with what is known about each file:
// its name and figures, that it is being read, that it could not be, or that
// there is none yet.
//
// 🔴 The name and not the path, for two reasons that happen to agree. A person
// knows their file by its name, and the whole path is a line of noise around
// it. And the stored tree of a state would otherwise carry a directory of the
// machine it was rendered on.
func (v *View) drawBand() {
	v.manifestFacts.Objects = []fyne.CanvasObject{v.manifestFactsNow()}
	v.manifestFacts.Refresh()
	v.responseFacts.Objects = []fyne.CanvasObject{v.responseFactsNow()}
	v.responseFacts.Refresh()
}

func (v *View) manifestFactsNow() fyne.CanvasObject {
	switch {
	case v.reading != "" && v.readingWhat == theManifest:
		return v.named(v.reading, v.text.T(i18n.ManifestFactsReading), palette.TextSecondary)
	case v.unread != nil && v.triedWhat == theManifest:
		return v.named(v.tried, v.text.T(i18n.ManifestFactsUnreadable), palette.StateDanger)
	case !v.hasPromise:
		return v.cut(v.text.T(i18n.ManifestEmpty))
	}
	run := v.promised.Run()
	words := v.text.Counted(len(v.screen.Rows), i18n.ManifestFactsEntry, i18n.ManifestFactsEntries) +
		between + v.text.T(i18n.ManifestFactsSeed) + " " + run.Seed +
		between + run.Locale
	return v.named(v.manifestPath, words, palette.TextSecondary)
}

func (v *View) responseFactsNow() fyne.CanvasObject {
	switch {
	case v.reading != "" && v.readingWhat == theResponse:
		return v.named(v.reading, v.text.T(i18n.ManifestFactsReading), palette.TextSecondary)
	case v.unread != nil && v.triedWhat == theResponse:
		return v.named(v.tried, v.text.T(i18n.ManifestFactsUnreadable), palette.StateDanger)
	case !v.hasPromise:
		return v.cut(v.text.T(i18n.ManifestResponseWait))
	case !v.hasResponse:
		// Beside a button that can be pressed there is nothing to say, and a
		// sentence saying "not open yet" would be furniture.
		return paint.Spacer()
	}
	words := strconv.Itoa(v.answered) + " " + v.text.T(i18n.ManifestFactsOf) + " " +
		strconv.Itoa(v.promised.Run().RowsTotal) + " " + v.text.T(i18n.ManifestFactsAnswered)
	facts := v.named(v.responsePath, words, palette.TextSecondary)
	if verdict := v.fileVerdict(); verdict != nil {
		return paint.Row(palette.NameStepTight, facts, verdict)
	}
	return facts
}

// fileVerdict is what the comparison says about the response as a whole, in
// the few words the slot beside the file has room for: how many records it
// never answered, or that it disagreed with nothing. Nothing when neither is
// true, because then the tally at the heading of the table says it.
//
// # Why here and not in a strip over the table (DEC-251)
//
// The strip stood between the band and the table and was as tall as what it
// said, so opening a response made the window taller than the one it opened
// in (B-299). This slot is one line of fixed height whether it holds a count or
// not. The shape goes with the words, since the two colours differ in hue far
// more than in lightness and a hue on its own reaches nobody who does not
// separate them (docs/17, 4.2).
func (v *View) fileVerdict() fyne.CanvasObject {
	switch {
	case v.screen.NeverAnswered > 0:
		return verdictFact(paint.IconAlert, palette.StateDanger,
			strconv.Itoa(v.screen.NeverAnswered)+" "+v.text.T(i18n.ManifestFactsNeverAnswered))
	case v.screen.Agreed:
		return verdictFact(paint.IconCheck, palette.StateSuccess, v.text.T(i18n.ManifestFactsAgreed))
	}
	return nil
}

// verdictFact is one more fact after the dot, with the shape of its verdict
// in front of the words.
func verdictFact(icon, ink, words string) fyne.CanvasObject {
	return paint.Row(palette.NameStepHairline,
		paint.Middle(paint.Text(dot, palette.TypeBody, palette.TextSecondary)),
		paint.Middle(paint.Icon(icon, ink, palette.IconSmall)),
		paint.Middle(paint.Text(words, palette.TypeBody, ink)))
}

// named is the name of a file with a few words after it, or the words alone
// for a document that arrived without a file.
func (v *View) named(path, words, ink string) fyne.CanvasObject {
	if path == "" {
		return paint.Text(words, palette.TypeBody, ink)
	}
	return paint.Row(palette.NameStepTight,
		paint.Strong(filepath.Base(path), palette.TypeBody, palette.TextPrimary),
		paint.Text(dot+" "+words, palette.TypeBody, ink))
}

// dot stands between two facts about a file, as it does between the version
// and the promise at the foot of the sidebar, and between is the dot with the
// same room on both sides of it.
const (
	dot     = "·"
	between = " " + dot + " "
)

func (v *View) chooseManifest() {
	dialog.ShowFileOpen(func(chosen fyne.URIReadCloser, err error) {
		path, ok := v.pathOf(chosen, err)
		if !ok {
			return
		}
		v.startReading(path, theManifest, func() func() {
			promised, readErr := v.read.Manifest(path)
			return func() {
				if readErr != nil {
					v.hasPromise = false
					v.respond.Disable()
					v.unreadable(readErr)
					return
				}
				v.Open(promised, path)
			}
		})
	}, v.window)
}

func (v *View) chooseResponse() {
	dialog.ShowFileOpen(func(chosen fyne.URIReadCloser, err error) {
		path, ok := v.pathOf(chosen, err)
		if !ok || !v.hasPromise {
			return
		}
		v.startReading(path, theResponse, func() func() {
			actual, readErr := v.read.Response(path)
			return func() {
				if readErr != nil {
					v.unreadable(readErr)
					return
				}
				v.Compare(actual, path)
			}
		})
	}, v.window)
}

// startReading hands a read to the place of work, says on screen that it is
// going on, and puts the controls out of reach until it answers.
//
// # Why the answer is checked against a counter before it is used
//
// The buttons are disabled for the length of a read, so two reads cannot be
// asked for by anybody clicking. The counter is here anyway, because that
// sentence is about the controls and this one is about the screen: the day
// something else opens a document, an answer about the document before last
// would otherwise arrive second and be drawn over a newer one.
//
// The read itself builds what to do about it, on the goroutine that read, out
// of values it made there. Nothing is shared between the two threads.
func (v *View) startReading(path string, what document, read func() func()) {
	v.asked++
	wanted := v.asked
	v.reading, v.readingWhat = path, what
	v.tried, v.triedWhat, v.unread = path, what, nil
	v.freeze()
	v.drawBand()
	v.drawDetail()
	v.load(func() func() {
		act := read()
		return func() {
			if wanted != v.asked {
				return
			}
			v.reading = ""
			v.thaw()
			act()
		}
	}, func(err error) func() {
		// A read that came apart goes where a read that failed goes. The
		// band names the file, and the failure names itself as a defect of
		// this build, so the two together say whose fault it is without this
		// screen having to decide.
		return func() {
			if wanted != v.asked {
				return
			}
			v.reading = ""
			v.thaw()
			v.unreadable(err)
		}
	})
}

// Reading reports whether a document is being read, for a caller with nobody
// looking at the screen.
func (v *View) Reading() bool { return v.reading != "" }

// unreadable puts a failed read in the band and in the card under the table.
func (v *View) unreadable(err error) {
	v.unread = err
	v.drawBand()
	v.drawDetail()
	// The card is where the failure is said in full, so it is shown even with
	// no manifest open, when it is otherwise put away.
	v.foot.Show()
}

// freeze puts the two buttons out of reach for the length of a read.
//
// A control that cannot do anything yet says so by being unavailable rather
// than by being pressable and ignored. The button for a response record goes
// back to whatever it was, because whether it may be used at all is decided by
// there being a manifest open and not by this.
func (v *View) freeze() {
	v.open.Disable()
	v.respond.Disable()
}

func (v *View) thaw() {
	v.open.Enable()
	if v.hasPromise {
		v.respond.Enable()
	}
}

// pathOf reads a choice, and reports whether there is anything to do about it.
//
// A cancelled chooser hands back nothing and no error, and it leaves the screen
// exactly as it was. Turning a cancellation into an error message would tell a
// person something went wrong when they were the one who decided nothing
// should.
func (v *View) pathOf(chosen fyne.URIReadCloser, err error) (string, bool) {
	if err != nil {
		v.tried, v.triedWhat = "", theManifest
		v.unreadable(err)
		return "", false
	}
	if chosen == nil {
		return "", false
	}
	// The reader is closed and its own failure ignored on purpose: the path is
	// all that is wanted from it, nothing was written through it, and a
	// complaint about closing a file somebody only picked would say nothing a
	// person could act on.
	defer func() { _ = chosen.Close() }()
	return chosen.URI().Path(), true
}

// Unread is what the last attempt to read a document ended with, or nothing.
func (v *View) Unread() error { return v.unread }
