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
package about

import (
	"encoding/json"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
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
	// A screen with no way to write is a screen where this button would do
	// nothing at all, and a control that does nothing is a promise. Disabled,
	// it also loses its edge, which is what says a control is there.
	if save == nil {
		button.Disable()
	}
	return button
}

// Content is the whole screen, built from what this build carries.
//
// It takes the report and the versions rather than reading them itself, so the
// screen can be rendered in a test against a build that carries something else
// entirely. A view that fetches its own facts can only ever be checked against
// the one set of facts it fetches.
func Content(text i18n.Catalogue, report license.Report, versions version.Set,
	save Saver, notices Notices) fyne.CanvasObject {
	sections := []fyne.CanvasObject{
		container.NewHBox(
			saveButton(text, report, save).OnScreen(),
			noticeButton(text, report, save, notices).OnScreen()),
		heading(text, i18n.AboutHeadingVersions),
		versionList(text, versions),
		widget.NewSeparator(),
		heading(text, i18n.AboutHeadingLicenses),
		licenseList(text, report),
		widget.NewSeparator(),
		heading(text, i18n.AboutHeadingComponents),
		componentList(text, report),
		widget.NewSeparator(),
		// The fonts and the icons, which are neither modules nor reference
		// lists and were named on neither surface until 2026-09-08. This
		// section is the window's half of the pair, and the command line
		// prints the same facts under a heading of its own.
		heading(text, i18n.AboutHeadingEmbedded),
		embeddedList(text, report),
		widget.NewSeparator(),
		heading(text, i18n.AboutHeadingNotices),
		noticeList(report),
	}
	// The scrolling area is the keyboard's rather than the toolkit's own, which
	// answers to the wheel and to nothing else. This screen is one long column
	// and has no control on it at all, so without that the only way through it
	// is a mouse: measured on 2026-08-24 at 3169 pixels of content in a window
	// showing 836 of them, with every key leaving the offset at zero.
	return parts.Inset(parts.NewKeyScroll(container.NewVBox(sections...)).Framed())
}

func heading(text i18n.Catalogue, key i18n.Key) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text.T(key), fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

// pair is a label and the value beside it. Values are data rather than
// interface text, so they are not keys.
func pair(label, value string) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	return container.NewHBox(name, widget.NewLabel(value))
}

func versionList(text i18n.Catalogue, versions version.Set) fyne.CanvasObject {
	return container.NewVBox(
		pair(text.T(i18n.AboutVersionTool), versions.Tool),
		pair(text.T(i18n.AboutVersionCatalog), versions.Catalog),
		pair(text.T(i18n.AboutVersionManifest), versions.Manifest),
		pair(text.T(i18n.AboutVersionRecipe), versions.Recipe),
	)
}

func licenseList(text i18n.Catalogue, report license.Report) fyne.CanvasObject {
	rows := []fyne.CanvasObject{
		pair(text.T(i18n.AboutLicenseCode), report.Code.License),
		pair(text.T(i18n.AboutLicenseSpec), report.Spec.License),
	}
	// The scope is filled only where the license does not cover everything
	// this binary compiles, and without it the two names above read as a
	// contradiction.
	if report.Spec.Scope != "" {
		rows = append(rows, widget.NewLabel(report.Spec.Scope))
	}
	return container.NewVBox(rows...)
}

func componentList(text i18n.Catalogue, report license.Report) fyne.CanvasObject {
	if len(report.Components) == 0 {
		return widget.NewLabel(text.T(i18n.AboutEmptyComponents))
	}
	rows := make([]fyne.CanvasObject, 0, len(report.Components))
	for _, component := range report.Components {
		rows = append(rows, componentRow(text, component))
	}
	return container.NewVBox(rows...)
}

// componentRow describes one linked module in two lines rather than five.
//
// # Why two rather than one field per line
//
// Looking at the first build showed the reason: with a label on its own line
// for every field, thirty seven modules became one hundred and eighty five
// lines that all look the same. Squinting at it picked out nothing, which is
// the first question the checklist in docs/17 asks of a screen. The module
// name leads because it is what somebody scans for, and the rest of the facts
// follow it on the same line.
//
// The day the license was read stays on screen beside the license itself,
// because a license is a property of a version rather than of a project, and a
// name with no date is a fact somebody remembered rather than one they checked.
func componentRow(text i18n.Catalogue, component license.Component) fyne.CanvasObject {
	name := widget.NewLabelWithStyle(component.Module, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	terms := widget.NewLabel(component.Version + "   " + component.License)

	source := widget.NewLabel(component.Source)
	source.Importance = widget.LowImportance
	checked := widget.NewLabel(text.T(i18n.AboutColumnChecked) + " " + component.Checked)
	checked.Importance = widget.LowImportance

	return container.NewVBox(
		container.NewHBox(name, terms),
		container.NewHBox(source, checked),
	)
}

func noticeList(report license.Report) fyne.CanvasObject {
	generated := widget.NewLabel(report.Notices.GeneratedData)
	generated.Wrapping = fyne.TextWrapWord
	synthetic := widget.NewLabel(report.Notices.SyntheticData)
	synthetic.Wrapping = fyne.TextWrapWord
	return container.NewVBox(generated, synthetic)
}

// embeddedList names the typeface and the icon set this build carries.
//
// The shape follows componentRow above rather than inventing a second one: the
// asset leads because it is what somebody scans for, the terms follow on the
// same line, and the day they were read stays beside them because a licence is
// a property of a version rather than of a project.
func embeddedList(text i18n.Catalogue, report license.Report) fyne.CanvasObject {
	if len(report.Embedded) == 0 {
		return widget.NewLabel(text.T(i18n.AboutEmptyEmbedded))
	}
	rows := make([]fyne.CanvasObject, 0, len(report.Embedded))
	for _, asset := range report.Embedded {
		name := widget.NewLabelWithStyle(asset.Asset, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		terms := widget.NewLabel(asset.Version + "   " + asset.License)

		source := widget.NewLabel(asset.Source)
		source.Importance = widget.LowImportance
		checked := widget.NewLabel(text.T(i18n.AboutColumnChecked) + " " + asset.Checked)
		checked.Importance = widget.LowImportance

		rows = append(rows, container.NewVBox(
			container.NewHBox(name, terms),
			container.NewHBox(source, checked),
		))
	}
	return container.NewVBox(rows...)
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
