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
	save Saver) fyne.CanvasObject {
	sections := []fyne.CanvasObject{
		container.NewHBox(saveButton(text, report, save).OnScreen()),
		heading(text, i18n.AboutHeadingVersions),
		versionList(text, versions),
		widget.NewSeparator(),
		heading(text, i18n.AboutHeadingLicenses),
		licenseList(text, report),
		widget.NewSeparator(),
		heading(text, i18n.AboutHeadingComponents),
		componentList(text, report),
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
