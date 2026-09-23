package about

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// sections is the page under the band, in the order a person reads it: what
// this build is, what that means for their data, and then what went into it.
//
// # Why the lists go shortest first
//
// Two sources of reference data and two embedded assets say something at a
// glance, and thirty six modules are an appendix. Put first, the modules pushed
// both of the short lists three screens down, where nobody scrolled to them.
func sections(text i18n.Catalogue, report license.Report, versions version.Set) fyne.CanvasObject {
	return paint.Pad(palette.NameStepSection, palette.NameStepSection,
		palette.NameStepMajor, palette.NameStepMajor,
		paint.Column(palette.NameStepSection,
			buildCard(text, report, versions),
			section(text.T(i18n.AboutHeadingNotices), "", noticeLines(report)),
			section(text.T(i18n.AboutHeadingData), dataCount(text, report), dataList(text, report)),
			section(text.T(i18n.AboutHeadingEmbedded),
				howMany(text, len(report.Embedded), i18n.AboutCountAsset, i18n.AboutCountAssets),
				embeddedList(text, report)),
			section(text.T(i18n.AboutHeadingComponents),
				howMany(text, len(report.Components), i18n.AboutCountModule, i18n.AboutCountModules),
				componentList(text, report))))
}

// howMany is the count at the end of a heading, or nothing for an empty list,
// where the sentence under the heading already says there is nothing and a
// nought beside it would say it twice.
func howMany(text i18n.Catalogue, many int, one, rest i18n.Key) string {
	if many == 0 {
		return ""
	}
	return text.Counted(many, one, rest)
}

// section is a heading with a rule to the far edge and how many things are in
// it at the end of the rule, the same heading that stands over the table of
// the manifest browser.
func section(heading, count string, body fyne.CanvasObject) fyne.CanvasObject {
	var aside fyne.CanvasObject = paint.Spacer()
	if count != "" {
		aside = paint.Middle(paint.Text(count, palette.TypeLabel, palette.TextSecondary))
	}
	return paint.Column(palette.NameStepRelated, paint.SectionHeading(heading, aside), body)
}

// buildCard is the one card on the screen: the versions and the licenses side
// by side, each half a column of labels and a column of values.
//
// # Why this is the card
//
// A screen has one framed thing and it is what the screen is for. These six
// facts are what somebody puts in a bug report or a compliance answer, and all
// the rest of the page is either a consequence of them or an inventory.
//
// The number of columns is how many halves there are, which reads as what it
// is and is not a number written into the view.
func buildCard(text i18n.Catalogue, report license.Report, versions version.Set) fyne.CanvasObject {
	versionHalf := paint.Column(palette.NameStepTight,
		paint.Caption(text.T(i18n.AboutHeadingVersions)),
		form([]pair{
			{text.T(i18n.AboutVersionTool), versions.Tool},
			{text.T(i18n.AboutVersionCatalog), versions.Catalog},
			{text.T(i18n.AboutVersionManifest), versions.Manifest},
			{text.T(i18n.AboutVersionRecipe), versions.Recipe},
		}))
	licenseHalf := []fyne.CanvasObject{
		paint.Caption(text.T(i18n.AboutHeadingLicenses)),
		form([]pair{
			{text.T(i18n.AboutLicenseCode), report.Code.License},
			{text.T(i18n.AboutLicenseSpec), report.Spec.License},
		}),
	}
	// The scope is filled only where the license does not cover everything
	// this binary compiles, and without it the two names above read as a
	// contradiction.
	if report.Spec.Scope != "" {
		licenseHalf = append(licenseHalf, quiet(report.Spec.Scope))
	}
	halves := []fyne.CanvasObject{versionHalf, paint.Column(palette.NameStepTight, licenseHalf...)}
	return paint.Card(palette.NameStepGroup, container.NewGridWithColumns(len(halves), halves...))
}

// pair is a label and the value beside it. Values are data rather than
// interface text, so they are not keys.
type pair struct{ label, value string }

// form lays pairs out as a column of labels and a column of values, so the
// labels line up under each other (rule 13 of the window).
//
// # Why the label carries room on its right
//
// The form takes the distance between its two columns from the padding of the
// theme, and that padding is nought by decision. The first drawing of this card
// set "Corruption catalog" against its value with no room between them.
func form(pairs []pair) fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, 0, len(pairs)+len(pairs))
	for _, one := range pairs {
		cells = append(cells,
			paint.Pad(palette.NameStepHairline, palette.NameStepHairline,
				palette.NameStepHairline, palette.NameStepGroup,
				paint.Text(one.label, palette.TypeBody, palette.TextSecondary)),
			paint.Middle(paint.Strong(one.value, palette.TypeBody, palette.TextPrimary)))
	}
	return container.New(layout.NewFormLayout(), cells...)
}

// noticeLines are the two sentences about the user's data, each with its sign.
//
// The same two `qadg --license` prints. The third notice of the report, about
// the examples in the catalog, is in the file and on neither surface as text,
// because it describes this project's documents rather than anybody's data.
func noticeLines(report license.Report) fyne.CanvasObject {
	return paint.Column(palette.NameStepRelated,
		paint.Marked(paint.IconCheck, palette.TextSecondary, parts.Sentence(report.Notices.GeneratedData)),
		paint.Marked(paint.IconShield, palette.TextSecondary, parts.Sentence(report.Notices.SyntheticData)))
}
