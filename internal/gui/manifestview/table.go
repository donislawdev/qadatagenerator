package manifestview

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// columns are the headings, in column order, matching Row.Cells, each with the
// sentence that says what it holds.
//
// 🔴 One list rather than two, and that is the lesson of the register that let
// a fourteenth generator in without a type: two lists kept in the same order by
// hand are two lists that come apart the first time somebody adds to one of
// them, quietly and without anything going red. Here the pair is the entry, so
// a fifth column cannot be added without saying what it holds.
var columns = []struct{ Name, Holds i18n.Key }{
	{i18n.ManifestColumnMutator, i18n.ManifestWaitMutator},
	{i18n.ManifestColumnPlace, i18n.ManifestWaitPlace},
	{i18n.ManifestColumnExpect, i18n.ManifestWaitExpect},
	{i18n.ManifestColumnStatus, i18n.ManifestWaitStatus},
}

func (v *View) columnName(column int) string {
	if column < 0 || column >= len(columns) {
		return ""
	}
	return v.text.T(columns[column].Name)
}

// drawEntries puts the table in its slot, or the shape of one while there is
// no manifest to fill it with.
//
// The table is built from the rows it shows, so it is built here rather than
// once. This runs when a file is opened and at no other time: a read that is
// going on and a read that failed leave the entries where they are, because
// the file on the screen is still the file on the screen.
func (v *View) drawEntries() {
	if !v.hasPromise {
		v.entries = nil
		v.entriesBox.Objects = []fyne.CanvasObject{v.guide()}
	} else {
		v.entries = v.table(v.screen.Rows)
		v.entriesBox.Objects = []fyne.CanvasObject{
			paint.AtLeast(0, v.rowsAtLeast(), v.entries.Framed())}
	}
	v.entriesBox.Refresh()
}

// rowsAtLeast is the least height the table asks the window for: its heading
// and as many rows as the theme says, so that a window too short for them
// grows rather than handing the table a negative height.
func (v *View) rowsAtLeast() float32 {
	cell := cellHeight()
	between := palette.StepOf(palette.NameStepTight)
	rows := float32(palette.EntryRowsAtLeast)
	return cell + rows*cell + rows*between
}

// cellHeight is how tall one cell of the table is, asked of the toolkit rather
// than assumed, because a person may scale the interface and every value this
// view draws with belongs to the theme rather than to this file.
func cellHeight() float32 { return painted(Line{Text: "0"}).MinSize().Height }

// guide is the shape of the entries table before there are entries: the name
// of each column and what it holds.
//
// 🔴 This stands where the empty table stood in the first draft, and the
// difference was measured rather than argued. An entries table with headings
// and no rows took 40 per cent of the height of the render and 46 per cent of
// the real window, and what a person meets on opening this screen should not be
// a frame around nothing. The headings carry four words, and a word is not an
// answer: "Outcome" means nothing to somebody who has never run this tool,
// which is exactly who this state is drawn for.
//
// A card that hugs its sentences rather than a frame filling the slot, for the
// reason the take screen measured: an empty framed rectangle reads as a
// failure where free space on the ground reads as room.
func (v *View) guide() fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, 0, len(columns)+len(columns))
	for _, column := range columns {
		cells = append(cells, paint.Strong(v.text.T(column.Name), palette.TypeBody, palette.TextPrimary))
		cells = append(cells, v.paragraph(v.text.T(column.Holds)))
	}
	return container.NewVBox(paint.Card(palette.NameStepGroup, paint.Column(palette.NameStepRelated,
		v.paragraph(v.text.T(i18n.ManifestEntriesWait)),
		container.New(layout.NewFormLayout(), cells...))))
}

// table is the entries, as a table.
//
// # Why the toolkit's table rather than a list of rows built by hand
//
// The first two attempts built one: a list whose rows were containers, then a
// list whose rows were a grid of equal columns. The first had nothing lining up
// between one entry and the next. The second lined up and then ran over itself
// in a narrow window, because a piece of canvas text draws its whole string
// wherever it is put and pays no attention to the width it was given.
//
// A table answers all of that and was there the whole time: columns have widths
// that are set rather than divided, the heading row is part of it, and cells are
// built only for what can be seen, which is what keeps a manifest of a hundred
// thousand entries from costing what it would in a plain column.
//
// # How the card under it follows the table
//
// Two ways in, because two kinds of person use this screen. The keyboard moves
// a highlight from cell to cell, and the card reads out the entry under it as
// it moves, so that browsing with the arrow keys is browsing. The pointer
// selects an entry by clicking it, and the card reads that one out until
// another is chosen.
func (v *View) table(rows []Row) *parts.KeyTable {
	entries := parts.NewKeyTable(
		func() (int, int) { return len(rows), palette.EntryColumns() },
		func() fyne.CanvasObject { return painted(Line{}) },
		func(cell widget.TableCellID, object fyne.CanvasObject) {
			if text, ok := object.(*widget.RichText); ok && cell.Row < len(rows) {
				set(text, rows[cell.Row].Cell(cell.Col), v.cellStyle(cell))
			}
		},
		v.tableFocus,
	)
	entries.ShowHeaderColumn = false
	entries.CreateHeader = func() fyne.CanvasObject { return bold(Line{Color: palette.TextSecondary}) }
	entries.UpdateHeader = func(cell widget.TableCellID, object fyne.CanvasObject) {
		if text, ok := object.(*widget.RichText); ok {
			set(text, Line{Text: v.columnName(cell.Col), Color: palette.TextSecondary},
				fyne.TextStyle{Bold: true})
		}
	}
	// The toolkit redraws the cell it is leaving and the cell it is arriving at
	// before it says where it arrived, so the mark has to be moved by hand here.
	// Without these two refreshes the mark trails one key press behind.
	entries.OnHighlighted = func(cell widget.TableCellID) {
		previous, had := v.highlight, v.hasHighlight
		v.highlight, v.hasHighlight = cell, true
		if had && previous != cell {
			entries.RefreshItem(previous)
		}
		entries.RefreshItem(cell)
		v.readOut(cell.Row)
	}
	entries.OnSelected = func(cell widget.TableCellID) { v.readOut(cell.Row) }
	entries.Widths(palette.EntryColumnWidths, palette.EntryStretchColumn)
	return entries
}

// tableFocus records whether the table is what the keyboard is talking to, and
// reads out the entry it is standing on when it arrives.
func (v *View) tableFocus(has bool) {
	v.tableFocused = has
	if has && v.hasHighlight {
		v.readOut(v.highlight.Row)
	}
}

// readOut makes the card under the table about one entry.
func (v *View) readOut(row int) {
	if row < 0 || row >= len(v.screen.Rows) {
		return
	}
	v.selected, v.hasSelected = row, true
	v.drawDetail()
}

// drawTally fills the trailing end of the table's heading: how much of what
// was ordered happened, or once a response is open, how much could be judged
// and with what result, or before any file the one line explaining the mark
// on a soft expectation.
func (v *View) drawTally() {
	var said fyne.CanvasObject
	switch {
	case !v.hasPromise:
		said = paint.Text(v.text.T(i18n.ManifestLegendMark), palette.TypeLabel, palette.TextSecondary)
	case v.hasResponse:
		said = v.tallied(
			tallyPart{strconv.Itoa(v.screen.Compared) + " " + v.text.T(i18n.ManifestFactsOf) + " " +
				strconv.Itoa(len(v.screen.Rows)) + " " + v.text.T(i18n.ManifestTallyCompared), palette.TextSecondary},
			tallyPart{strconv.Itoa(v.screen.Disagreed) + " " + v.text.T(i18n.ManifestTallyDisagree),
				attention(v.screen.Disagreed, palette.StateDanger)},
			tallyPart{strconv.Itoa(len(v.screen.Unsettled)) + " " + v.text.T(i18n.ManifestTallyNotSettled),
				attention(len(v.screen.Unsettled), palette.StateWarning)})
	default:
		notApplied := len(v.promised.Unapplied())
		said = v.tallied(
			tallyPart{strconv.Itoa(v.promised.Requested()) + " " + v.text.T(i18n.ManifestTallyOrdered), palette.TextSecondary},
			tallyPart{strconv.Itoa(v.promised.Applied()) + " " + v.text.T(i18n.ManifestTallyApplied), palette.TextSecondary},
			tallyPart{strconv.Itoa(notApplied) + " " + v.text.T(i18n.ManifestTallyNotApplied),
				attention(notApplied, palette.StateWarning)})
	}
	v.tally.Objects = []fyne.CanvasObject{said}
	v.tally.Refresh()
}

// tallyPart is one count of the tally and the ink it is written in.
type tallyPart struct{ words, ink string }

// attention is the ink of a count that matters when it is not nought, and the
// quiet ink when it is: "0 disagree" in red would be an alarm about nothing.
func attention(count int, ink string) string {
	if count == 0 {
		return palette.TextSecondary
	}
	return ink
}

// tallied writes the counts with a dot between each pair.
func (v *View) tallied(parts ...tallyPart) fyne.CanvasObject {
	var drawn []fyne.CanvasObject
	for at, part := range parts {
		if at > 0 {
			drawn = append(drawn, paint.Text(dot, palette.TypeBody, palette.TextSecondary))
		}
		drawn = append(drawn, paint.Text(part.words, palette.TypeBody, part.ink))
	}
	return paint.Row(palette.NameStepTight, drawn...)
}

// painted is one phrase in the colour the model asked for, trimmed to whatever
// width it is given.
//
// 🔴 Why rich text rather than a piece of canvas text, which is the obvious
// choice for a coloured phrase: canvas text draws its whole string wherever it
// is put, ignoring the size it was laid out at. Measured on 2026-08-18 in a
// window 1342 pixels wide, that meant the identifier of a damage ran straight
// through the column beside it and two facts about different things were drawn
// in the same pixels. Rich text trims to its column and marks the cut, which is
// rule 4.7 of docs/17, and it takes a colour by name because the theme answers
// for the tokens of this window as well as for the toolkit's own names.
func painted(line Line) *widget.RichText {
	text := widget.NewRichText(segmentOf(line))
	text.Truncation = fyne.TextTruncateEllipsis
	return text
}

// plain is a phrase that asks for as much width as it needs.
func plain(line Line, heavy bool) *widget.RichText {
	segment := segmentOf(line)
	if heavy {
		segment.Style.TextStyle = fyne.TextStyle{Bold: true}
	}
	return widget.NewRichText(segment)
}

// bold is a phrase carrying weight, which is how one thing is made to stand out
// without spending a second colour on it.
func bold(line Line) *widget.RichText {
	text := painted(line)
	if segment, ok := text.Segments[0].(*widget.TextSegment); ok {
		segment.Style.TextStyle = fyne.TextStyle{Bold: true}
	}
	return text
}

func segmentOf(line Line) *widget.TextSegment {
	name := line.Color
	if name == "" {
		name = palette.TextPrimary
	}
	return &widget.TextSegment{
		Text:  line.Text,
		Style: widget.RichTextStyle{ColorName: fyne.ThemeColorName(name), Inline: true},
	}
}

// set writes one phrase into a piece of text that is already on screen.
//
// The refresh is what has the new words rasterised: assigning to the segment
// changes what the object holds, and the screen goes on showing what was last
// drawn until something asks for it again, which matters here because every
// cell of a table was already drawn as some other cell. The weight and the
// underline are written on every update rather than only when there is
// something to mark. Cells are drawn from a pool and handed out again as the
// table scrolls, so a style left behind on one would reappear under a row that
// has nothing to do with it.
func set(target *widget.RichText, line Line, style fyne.TextStyle) {
	name := line.Color
	if name == "" {
		name = palette.TextPrimary
	}
	// The length is asked about before the position is taken. Nothing in this
	// package builds a rich text without a segment, so this cannot happen
	// today, and the cells are drawn from a pool the toolkit hands back and
	// forth. A panic in a screen takes the window with it, and one line is
	// cheaper than the belief that the pool will always hold what it held.
	if len(target.Segments) == 0 {
		return
	}
	segment, ok := target.Segments[0].(*widget.TextSegment)
	if !ok {
		return
	}
	segment.Text = line.Text
	segment.Style.ColorName = fyne.ThemeColorName(name)
	segment.Style.TextStyle = style
	target.Refresh()
}

// cellStyle marks the one cell the keyboard is standing on.
//
// # Why weight and an underline rather than a colour behind the cell
//
// The toolkit marks the cell by moving its hover rectangle, which this project
// paints in the colour of a row under the pointer. Measured on 2026-08-18 that
// rectangle stands at 1.18 against the surface behind it, where docs/17 asks
// three for anything carrying meaning without words, so as an indicator it is
// not there at all.
//
// Repainting that rectangle brighter cannot fix it, and the reason is arithmetic
// rather than taste: to reach three against the surface the fill needs a
// luminance of at least 0.153, and for the text of this product to keep four and
// a half on top of it the same fill may be at most 0.140. The two do not meet.
// A cell filled brightly enough to show where the keyboard is would be a cell
// whose words can no longer be read.
//
// So the mark is the text itself. It keeps the colour that says what the entry
// means, which a fill would have taken away, and it adds weight and a line under
// the words. That is a shape as well as a colour, which is what docs/17 asks of
// anything carrying meaning, and the text it is drawn in already stands at 8.13
// against the surface.
func (v *View) cellStyle(cell widget.TableCellID) fyne.TextStyle {
	if v.tableFocused && v.hasHighlight && v.highlight == cell {
		return fyne.TextStyle{Bold: true, Underline: true}
	}
	return fyne.TextStyle{}
}
