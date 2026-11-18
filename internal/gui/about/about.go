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

// Report is the licence report of this window: the register every build
// shares, and the notices of what the window itself links and carries, which
// the command line's report carries for the command line (DEC-327).
//
// # 🔴 Why the notices are read here, which is the whole point
//
// Measured on 2026-09-08 on the built binaries: an embedded text that no
// reachable code reads is pruned by the linker, so an embed directive alone
// ships nothing, and the tests beside it stay green because a test binary
// reads it. The window reaches this function, and through it the notices,
// which is what puts them into qadg-gui.
func Report() license.Report {
	report := license.Current()
	report.ThirdPartyNotices = license.WindowNotices
	return report
}

// Content is the whole screen, built from what this build carries.
//
// It takes the report and the versions rather than reading them itself, so the
// screen can be rendered in a test against a build that carries something else
// entirely. A view that fetches its own facts can only ever be checked against
// the one set of facts it fetches.
//
// The objects are handed to the layout in reading order, which is the order
// the keyboard walks them: the two buttons in the band, then the page.
//
// software says whether this window draws with the software renderer rather
// than the graphics driver (B-414). A person who wonders why the window is slow
// looks here, next to the version they would put in a report.
//
// kept is what the window keeps on this computer, or nil where there is no
// window keeping anything (see Kept).
func Content(text i18n.Catalogue, report license.Report, versions version.Set, software bool,
	save Saver, kept Kept) fyne.CanvasObject {
	top := band(text, report, save)
	// The scrolling area is the keyboard's rather than the toolkit's own, which
	// answers to the wheel and to nothing else. The page is text apart from one
	// button, so without that the only way through most of it is a mouse:
	// measured on 2026-08-24 at 3169 pixels of content in a window showing 836
	// of them, with every key leaving the offset at zero (DEC-187).
	page := parts.NewKeyScroll(sections(text, report, versions, software, kept)).Framed()
	return container.New(layout.NewBorderLayout(top, nil, nil, nil), top, page)
}

// band is what the screen is for and the two files it can write.
//
// # Why the buttons are here rather than beside what they describe
//
// The band does not scroll. The report is for a compliance department (DEC-30),
// and its button standing beside the sections it describes would have put the
// second one, the notices, under the fold. Both are quiet: this screen is read
// rather than acted on, so nothing on it is the one action it is for (DEC-250).
func band(text i18n.Catalogue, report license.Report, save Saver) fyne.CanvasObject {
	return paint.Band(paint.BandBelow, palette.NameStepGroup, palette.NameStepMajor,
		paint.Column(palette.NameStepGroup,
			quiet(text.T(i18n.PurposeAbout)),
			paint.Row(palette.NameStepRelated,
				saveButton(text, report, save).OnScreen(),
				noticeButton(text, report, save).OnScreen())))
}

// quiet is a sentence in the second ink that wraps to the width it is given.
//
// A label rather than text of the canvas, because a sentence as canvas text is
// drawn on one line whatever the width, and so decides how narrow the window
// may be. Placed in a column, where the width is the column's. The label itself
// comes back rather than a canvas object, because the line under Forget changes
// its words once the button is pressed.
func quiet(words string) *widget.Label {
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

// noticeButton offers the third party notices of the window as one file, the
// same text as THIRD-PARTY-NOTICES-GUI.md beside it in the archive. It is the
// path to them a person has who received the binary alone, the one
// `qadg --license` is for the command line.
//
// # Why a file rather than the text on this screen
//
// The licences inside the window, the Open Font License, ISC, MIT and BSD among
// them, ask the same thing, that the notice is kept with what it covers and
// reaches whoever received it. Thousands of lines of licence laid into this
// column would satisfy that and bury everything else on the screen, and
// somebody who wants the text wants to keep it rather than scroll it. It is the
// same argument that put the machine readable report behind a button beside
// this one.
//
// # Why it is disabled rather than hidden when there is nothing to write
//
// A control that is not there says nothing about why. A control that cannot be
// pressed says the report carries no notices, which is a fact worth showing,
// and it loses its edge as every disabled control here does.
func noticeButton(text i18n.Catalogue, report license.Report, save Saver) *parts.Button {
	button := parts.NewButton(text.T(i18n.AboutSaveNotices), func() {
		save([]byte(report.ThirdPartyNotices))
	})
	button.ShowIcon(paint.IconShield)
	if save == nil || report.ThirdPartyNotices == "" {
		button.Disable()
	}
	return button
}
