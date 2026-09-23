// Package about builds the screen that answers what this build is and under
// what terms.
//
// # Why this screen came first
//
// It needs no new product logic. The versions and the license register already
// exist and are already what the two flags print, so the first window measures
// the window rather than a feature written at the same time. It also closes a
// promise that was made in one document and carried by no other: the parity
// rule says every function has to be describable from both surfaces, and the
// license flags had no path here until this screen existed.
//
// # How it is laid out, settled as DEC-250
//
// A person comes here for one of four things: the version to put in a report,
// the license of the code and of the specification, whether the data they
// generate is theirs, or the file a compliance department reads. All four stand
// above the fold. The band at the top holds the purpose and the two ways to save
// a file, and it does not scroll, so the file is found without looking for it.
// Under it: one card with the versions and the licenses side by side, the
// sentences about the user's data, and then the three lists of what went into
// the build, the short ones first and the thirty six modules last.
//
// It had shown 29.1 percent of its content, with the versions and the first of
// the modules above the fold (B-280), and it had never shown the reference data
// the command line prints, which the rule of parity does not allow.
package about

import (
	"encoding/json"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Saver writes the machine readable report where a person chooses to put it.
//
// It is handed in rather than reached for, and it may be nothing: a screen
// being rendered into a picture has no window to open a chooser over, and a
// picture that wrote a file would leave one behind.
type Saver func(contents []byte)

// Notices hands over the licence text of one embedded asset, and says whether
// this build carries it at all.
//
// # Why the text arrives through a port rather than being read here
//
// The same reason the report does. This screen is rendered in tests against a
// build that carries something else entirely, and a register naming a notice
// that this binary does not embed is a legitimate state there rather than a
// fault. Reaching into the asset package would make that state a panic.
//
// # 🔴 Why anything reads these bytes at all, which is the whole point
//
// Measured on 2026-09-08 on the built binaries, after the embed directive was
// added and the tests beside it went green: the notices were STILL not in
// qadg-gui.exe. A go:embed variable that no reachable code reads is pruned by
// the linker, so the directive alone ships nothing. The tests passed because a
// test binary reads them, and a test binary is not the artefact anybody
// receives.
//
// One reachable read is enough for the whole set, which was measured the same
// way: with a single notice read, both appear in the binary. So this port is
// not a convenience on top of the fix. It IS the fix, and the register and the
// printed section are what describe it.
type Notices func(name string) (string, bool)

// Content is the whole screen, built from what this build carries.
//
// It takes the report and the versions rather than reading them itself, so the
// screen can be rendered in a test against a build that carries something else
// entirely. A view that fetches its own facts can only ever be checked against
// the one set of facts it fetches.
//
// The objects are handed to the layout in reading order, which is the order
// the keyboard walks them: the two buttons in the band, then the page.
func Content(text i18n.Catalogue, report license.Report, versions version.Set,
	save Saver, notices Notices) fyne.CanvasObject {
	top := band(text, report, save, notices)
	// The scrolling area is the keyboard's rather than the toolkit's own, which
	// answers to the wheel and to nothing else. The page holds no control, so
	// without that the only way through it is a mouse: measured on 2026-08-24 at
	// 3169 pixels of content in a window showing 836 of them, with every key
	// leaving the offset at zero (DEC-187).
	page := parts.NewKeyScroll(sections(text, report, versions)).Framed()
	return container.New(layout.NewBorderLayout(top, nil, nil, nil), top, page)
}

// band is what the screen is for and the two files it can write.
//
// # Why the buttons are here rather than beside what they describe
//
// The band does not scroll. The report is for a compliance department (DEC-30),
// and its button standing beside the sections it describes would have put the
// second one, the notices of the fonts, under the fold. Both are quiet: this
// screen is read rather than acted on, so nothing on it is the one action it is
// for (DEC-250).
func band(text i18n.Catalogue, report license.Report, save Saver, notices Notices) fyne.CanvasObject {
	return paint.Band(paint.BandBelow, palette.NameStepGroup, palette.NameStepMajor,
		paint.Column(palette.NameStepGroup,
			quiet(text.T(i18n.PurposeAbout)),
			paint.Row(palette.NameStepRelated,
				saveButton(text, report, save).OnScreen(),
				noticeButton(text, report, save, notices).OnScreen())))
}

// quiet is a sentence in the second ink that wraps to the width it is given.
//
// A label rather than text of the canvas, because a sentence as canvas text is
// drawn on one line whatever the width, and so decides how narrow the window
// may be. Placed in a column, where the width is the column's.
func quiet(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	label.Importance = widget.LowImportance
	return label
}

// saveButton offers the report in the form a machine reads.
//
// # Why a file rather than a panel showing the same thing
//
// DEC-30 names the reader and it is a compliance department, whose tool reads
// documents rather than paragraphs. Everything on this screen is already here
// for a person to read, so showing the same facts again as JSON would answer
// nobody. The register of what the window cannot do says this in its own
// words: the window has to PRODUCE the file.
//
// What is written is exactly what `qadg --license --json` writes, because both
// encode the same report with the same tags. That report carries the version of
// this build, so the one file answers which build as well as under what terms.
func saveButton(text i18n.Catalogue, report license.Report, save Saver) *parts.Button {
	button := parts.NewButton(text.T(i18n.AboutSaveJSON), func() {
		encoded, err := json.Marshal(report)
		if err != nil {
			// Nothing in the report is unencodable, so this cannot fire. It is
			// dropped rather than reported for the one reason a silence is
			// allowed here: the same call at the other surface says the same
			// thing on a channel this screen does not have.
			return
		}
		save(encoded)
	})
	button.ShowIcon(paint.IconDoc)
	// A screen with no way to write is a screen where this button would do
	// nothing at all, and a control that does nothing is a promise. Disabled,
	// it also loses its edge, which is what says a control is there.
	if save == nil {
		button.Disable()
	}
	return button
}

// noticeButton offers the licence texts of the embedded assets as one file.
//
// # Why a file rather than the text on this screen
//
// The Open Font License and ISC both ask the same thing, that the notice is
// kept with what it covers and reaches whoever received it. Several kilobytes
// of licence laid into this column would satisfy that and bury everything else
// on the screen, and somebody who wants the text wants to keep it rather than
// scroll it. It is the same argument that put the machine readable report
// behind a button beside this one.
//
// # Why it is disabled rather than hidden when there is nothing to write
//
// A control that is not there says nothing about why. A control that cannot be
// pressed says the build carries no notices, which is a fact worth showing, and
// it loses its edge as every disabled control here does.
func noticeButton(text i18n.Catalogue, report license.Report, save Saver,
	notices Notices) *parts.Button {
	document := noticeDocument(text, report, notices)
	button := parts.NewButton(text.T(i18n.AboutSaveNotices), func() {
		save([]byte(document))
	})
	button.ShowIcon(paint.IconShield)
	if save == nil || document == "" {
		button.Disable()
	}
	return button
}

// noticeDocument is every notice this build carries, under the facts about the
// asset it covers.
//
// An asset the register names and this build does not carry is left out rather
// than written as a heading with nothing under it. A heading over an absent
// licence reads as a licence somebody deleted, and the register is closed
// against the embedded set by a test, so in this build the case cannot arise.
func noticeDocument(text i18n.Catalogue, report license.Report, notices Notices) string {
	if notices == nil {
		return ""
	}
	var out strings.Builder
	for _, asset := range report.Embedded {
		body, carried := notices(asset.Notice)
		if !carried {
			continue
		}
		if out.Len() == 0 {
			out.WriteString(text.T(i18n.AboutNoticesHeader) + "\n")
		}
		out.WriteString("\n" + noticeRule + "\n")
		out.WriteString(asset.Asset + " " + asset.Version + "\n")
		out.WriteString(asset.License + "  " + asset.Source + "\n")
		out.WriteString(noticeRule + "\n\n")
		out.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			out.WriteString("\n")
		}
	}
	return out.String()
}

// noticeRule separates one notice from the next in that file.
//
// A row of characters rather than a blank line, because a licence text is
// itself full of blank lines and a reader scrolling one has to be able to see
// where the next begins. It is not interface text and is not translated: it
// carries no words.
const noticeRule = "----------------------------------------------------------------"
