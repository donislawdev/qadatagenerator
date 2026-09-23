package about

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/license"
)

// The three lists at the foot of the screen - the reference data, the fonts and
// icons, and the modules - are one component drawn three times, because a
// shape drawn a third time is a component rather than a copy (rule 2 of the
// window). Each is a head over four columns and a row per thing: the thing, its
// version, its license and the day that license was read.
//
// # Why the three columns on the right are measured per list
//
// Measured on the mock on 2026-09-23 rather than guessed. One set of widths for
// all three lists would line them up under each other, and it does not fit: the
// widest license is a plain statement of public domain at about 190 units and
// the widest version a pseudo version at about 290, which leaves the name of a
// module under 200 where the longest needs about 225. So each list takes the
// widths of its own values, and each has a head of its own saying what they are.
//
// # Why the name takes the rest of the width
//
// It is what somebody scans for. The three columns keep the width of their
// widest value, measured on the very objects that draw them, so they start at
// the same place in every row, and the name is given what is left.

// entry is one row of a list.
type entry struct {
	name, version, license, checked string
	// lists hangs under the row in the second ink and wraps, because the
	// number of lists read from one source grows with the years.
	lists string
	// address hangs under that, cut with an ellipsis where it does not fit: it
	// is data, and text of the canvas would decide the width of the window.
	address string
}

// trailing are the values of the three measured columns, in the order of the
// heads over them.
func (e entry) trailing() []string { return []string{e.version, e.license, e.checked} }

// list draws a head and a row per entry, with a hairline between rows, or the
// sentence saying there is nothing to list.
func list(text i18n.Catalogue, first, empty i18n.Key, entries []entry) fyne.CanvasObject {
	if len(entries) == 0 {
		return quiet(text.T(empty))
	}
	heads := []string{
		text.T(i18n.AboutColumnVersion), text.T(i18n.AboutColumnLicense), text.T(i18n.AboutColumnChecked),
	}
	widths := columnWidths(heads, entries)

	headCells := make([]fyne.CanvasObject, 0, len(heads))
	for _, words := range heads {
		headCells = append(headCells, headText(words))
	}
	lines := []fyne.CanvasObject{
		paint.PadAll(palette.NameStepHairline, line(widths, headText(text.T(first)), headCells)),
		paint.Hairline(palette.LineQuiet),
	}
	for at, one := range entries {
		if at > 0 {
			lines = append(lines, paint.Hairline(palette.LineQuiet))
		}
		lines = append(lines, row(widths, one))
	}
	return container.NewVBox(lines...)
}

// columnWidths is the widest head or value of each measured column.
func columnWidths(heads []string, entries []entry) []float32 {
	widths := make([]float32, len(heads))
	for column, words := range heads {
		widths[column] = headText(words).MinSize().Width
	}
	for _, one := range entries {
		for column, words := range one.trailing() {
			if width := valueText(words, palette.TextPrimary).MinSize().Width; width > widths[column] {
				widths[column] = width
			}
		}
	}
	return widths
}

// line puts the name on the left, taking the rest of the width, and the
// measured cells on the right at the widths of their columns.
func line(widths []float32, name fyne.CanvasObject, cells []fyne.CanvasObject) fyne.CanvasObject {
	placed := make([]fyne.CanvasObject, 0, len(cells))
	for column, cell := range cells {
		placed = append(placed, paint.AtLeast(widths[column], 0, cell))
	}
	return paint.Trail(name, paint.Row(palette.NameStepGroup, placed...))
}

// row is one entry: the line of four values, and under it what hangs there.
func row(widths []float32, one entry) fyne.CanvasObject {
	lines := []fyne.CanvasObject{line(widths, valueText(one.name, palette.TextPrimary), []fyne.CanvasObject{
		valueText(one.version, palette.TextSecondary),
		valueText(one.license, palette.TextPrimary),
		valueText(one.checked, palette.TextSecondary),
	})}
	if one.lists != "" {
		lines = append(lines, quiet(one.lists))
	}
	if one.address != "" {
		lines = append(lines, address(one.address))
	}
	return paint.Pad(palette.NameStepTight, palette.NameStepTight,
		palette.NameStepHairline, palette.NameStepHairline, container.NewVBox(lines...))
}

func headText(words string) fyne.CanvasObject {
	return paint.Text(words, palette.TypeLabel, palette.TextSecondary)
}

func valueText(words, ink string) fyne.CanvasObject {
	return paint.Text(words, palette.TypeBody, ink)
}

// address is a place the terms can be read, on one line and cut where it does
// not fit. The whole address is in the report the band saves.
func address(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Truncation = fyne.TextTruncateEllipsis
	label.Importance = widget.LowImportance
	return label
}

// componentList is the modules this build links.
//
// # Why without the address
//
// On every one of the thirty six modules the address is the name of the module
// with https:// in front of it, counted on 2026-09-23, so on screen it said the
// name twice. It stays in the report the band saves (DEC-250).
func componentList(text i18n.Catalogue, report license.Report) fyne.CanvasObject {
	entries := make([]entry, 0, len(report.Components))
	for _, one := range report.Components {
		entries = append(entries, entry{name: one.Module, version: one.Version, license: one.License,
			checked: one.Checked})
	}
	return list(text, i18n.AboutColumnModule, i18n.AboutEmptyComponents, entries)
}

// embeddedList is the typeface and the icon set, which are neither modules nor
// reference data and were named on neither surface until 2026-09-08.
func embeddedList(text i18n.Catalogue, report license.Report) fyne.CanvasObject {
	entries := make([]entry, 0, len(report.Embedded))
	for _, one := range report.Embedded {
		entries = append(entries, entry{name: one.Asset, version: one.Version, license: one.License,
			checked: one.Checked, address: one.Source})
	}
	return list(text, i18n.AboutColumnAsset, i18n.AboutEmptyEmbedded, entries)
}

// dataList is the reference data, one row per source with its lists under it,
// grouped by the same rule the command line prints them with.
//
// 🔴 Missing from this screen from its first commit until 2026-09-23, while
// `qadg --license` printed it all along and the heading waited in the
// catalogue. The window was a second, shorter answer to one question.
func dataList(text i18n.Catalogue, report license.Report) fyne.CanvasObject {
	sources := license.BySource(report.Data)
	entries := make([]entry, 0, len(sources))
	for _, one := range sources {
		entries = append(entries, entry{name: one.Project, version: one.Version, license: one.License,
			checked: one.Checked, lists: strings.Join(one.Lists, between), address: one.Source})
	}
	return list(text, i18n.AboutColumnSource, i18n.AboutEmptyData, entries)
}

// dataCount says how many lists came from how many sources, or nothing when
// there are none, for the reason howMany gives.
func dataCount(text i18n.Catalogue, report license.Report) string {
	if len(report.Data) == 0 {
		return ""
	}
	return text.Counted(len(report.Data), i18n.AboutCountList, i18n.AboutCountLists) + between +
		text.Counted(len(license.BySource(report.Data)), i18n.AboutCountSource, i18n.AboutCountSources)
}

// between stands between two facts on one line, the dot the band of the
// manifest browser and the foot of the sidebar use, with room on both sides.
// It carries no words, so it is not a key.
const between = "  ·  "
