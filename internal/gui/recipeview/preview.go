package recipeview

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// preview is the first rows of every table under the verdict.
//
// # Why rows at all, when the verdict already says the recipe is usable
//
// Because usable is not the same as right. A recipe that reads correctly can
// still describe a column of the wrong shape, a relation pointing the wrong way
// or a locale that produces nothing recognisable, and none of that is a problem
// a validator can see. The rows are how somebody checks the thing they actually
// meant, before anything is written.
//
// The rows are the run's own first rows rather than an illustration of them, and
// what makes that true is in internal/produce rather than here.
func (v *View) preview() []fyne.CanvasObject {
	var blocks []fyne.CanvasObject
	for _, table := range v.answer.Sample.Tables {
		head := paint.Row(palette.NameStepTight,
			paint.Strong(table.Entity, palette.TypeBody, palette.TextPrimary),
			paint.Text(v.rowsShown(table), palette.TypeBody, palette.TextSecondary))
		var body fyne.CanvasObject
		switch {
		case len(table.Columns) == 0:
			body = v.paragraph(v.text.T(i18n.RecipeNoFields))
		case len(table.Rows) == 0:
			// 🔴 A sentence rather than an empty rectangle, which is point eight
			// of the GUI checklist. A count of nought is an ordinary thing to
			// ask for while a test is being written, and a table drawn with a
			// heading and nothing under it says the tool failed.
			body = v.paragraph(v.text.T(i18n.RecipeNoRows))
		default:
			body = v.rows(table)
		}
		blocks = append(blocks, paint.Column(palette.NameStepTight, head, body))
	}
	return append(blocks, v.writtenText()...)
}

// writtenText shows the file itself for a form that is not a table.
//
// The object form nests children inside parents, so there is no row to put in a
// column. Showing the bytes is the honest answer: it is what the file will
// contain, and this screen has nothing truer to say about it.
func (v *View) writtenText() []fyne.CanvasObject {
	if len(v.answer.Sample.Tables) > 0 || len(v.answer.Sample.Files) == 0 {
		return nil
	}
	blocks := []fyne.CanvasObject{v.paragraph(v.text.T(i18n.RecipeAsText))}
	for _, file := range v.answer.Sample.Files {
		blocks = append(blocks, paint.Column(palette.NameStepTight,
			paint.Strong(file.Name, palette.TypeBody, palette.TextPrimary),
			written(file.Text)))
	}
	return blocks
}

// rowsShown says how much of a table is on screen.
//
// A table shorter than the sample and a table cut down to it are different
// things and read differently, so the sentence differs rather than the number
// alone. Somebody looking at three rows has seen the whole table.
func (v *View) rowsShown(table produce.Table) string {
	if !table.Cut() {
		return v.text.Counted(table.Total, i18n.RecipeRow, i18n.RecipeRows)
	}
	return v.text.T(i18n.RecipeShowing) + " " + strconv.Itoa(len(table.Rows)) + " " +
		v.text.T(i18n.RecipeOf) + " " + v.text.Counted(table.Total, i18n.RecipeRow, i18n.RecipeRows)
}

// rows is the table itself.
//
// # Why the toolkit's table and not a column of rows
//
// The same answer the manifest browser arrived at after building two of its
// own: columns line up because their widths are set rather than divided, the
// heading is part of it, and cells exist only for what can be seen. The type
// used here is the extension that says when the keyboard arrives and leaves,
// which is shared with that screen rather than copied from it.
func (v *View) rows(table produce.Table) fyne.CanvasObject {
	shown := table
	built := parts.NewKeyTable(
		func() (int, int) { return len(shown.Rows), len(shown.Columns) },
		func() fyne.CanvasObject { return v.cell() },
		func(cell widget.TableCellID, object fyne.CanvasObject) {
			if text, ok := object.(*widget.RichText); ok {
				v.setCell(text, v.cellText(shown, cell.Row, cell.Col))
			}
		},
		nil,
	)
	built.ShowHeaderColumn = false
	built.CreateHeader = func() fyne.CanvasObject { return v.cell() }
	built.UpdateHeader = func(cell widget.TableCellID, object fyne.CanvasObject) {
		if text, ok := object.(*widget.RichText); ok && cell.Col < len(shown.Columns) {
			v.setCell(text, shown.Columns[cell.Col])
		}
	}
	for column := range shown.Columns {
		built.SetColumnWidth(column, v.widthOf(shown, column))
	}
	// 🔴 The height is given and the width is NOT, and the difference is what
	// rendering showed. A table has no height of its own, so inside a column
	// that scrolls it collapses to nothing. Given both, it stops at the width it
	// was given and leaves the rest of a widened window empty, which is what the
	// first drawing did.
	//
	// So the height arrives as a spacer beside it, which the layout counts and
	// the table does not have to obey. The arithmetic of the height is the
	// toolkit's own rather than a guess at it: rows sit one cell apart plus the
	// padding between them, and the heading is one cell with nothing under it.
	// Written as anything else it comes out a few pixels short and the table
	// grows a scrollbar of its own inside a pane that already scrolls. Both
	// numbers come from the theme, the cell by measuring itself.
	cell := cellHeight()
	between := palette.StepOf(palette.NameStepTight)
	rows := float32(len(shown.Rows))
	tall := canvas.NewRectangle(color.Transparent)
	tall.SetMinSize(fyne.NewSize(0, cell+rows*cell+(rows-1)*between))
	// Framed, so that a keyboard standing on the table shows the one mark of
	// focus every control here wears. The frame is over the table alone, inside
	// the height the spacer gives the row, so it hugs the cells rather than the
	// whole width of the widened card.
	return container.NewBorder(nil, nil, tall, nil, built.Framed())
}

// cell is one piece of text in the table, on one line and cut with an ellipsis.
//
// 🔴 Cut rather than wrapped, and never mid character. Values here are
// deliberately multi byte: this tool generates for pl-PL and a whole module of
// damage exists to produce characters outside the basic plane. Cutting by byte
// puts invalid UTF-8 on screen, which this project has already repaired once in
// a report.
func (v *View) cell() *widget.RichText {
	text := widget.NewRichTextWithText("")
	text.Truncation = fyne.TextTruncateEllipsis
	text.Wrapping = fyne.TextWrapOff
	return text
}

// cellText is what one cell says.
//
// 🔴 An empty key naming a parent is not an empty value, it is a parent this
// sample does not reach. Drawn as nothing the two would be one cell, and one of
// them would be the tool failing to answer without saying so. Everywhere else
// an empty value is an ordinary thing for generated data to be, and stays
// empty.
func (v *View) cellText(table produce.Table, row, column int) string {
	if row >= len(table.Rows) || column >= len(table.Rows[row]) {
		return ""
	}
	if column == table.ParentColumn && table.ParentUnknown(row) {
		return v.text.T(i18n.RecipeParentOutside)
	}
	return table.Rows[row][column]
}

func (v *View) setCell(text *widget.RichText, value string) {
	text.Segments = []widget.RichTextSegment{&widget.TextSegment{Text: value}}
	text.Refresh()
}

// written is a file shown as itself, in the one face that keeps columns of
// characters under each other.
func written(text string) fyne.CanvasObject {
	label := widget.NewLabelWithStyle(text, fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})
	label.Wrapping = fyne.TextWrapOff
	return label
}

// widthOf measures a column and holds it between the two ends the theme names.
//
// The measuring is the toolkit's own, on the face it will actually draw with,
// so the width is of the text rather than of a guess about characters. Only the
// ends of the range are ours, and they live in the theme with everything else
// this view is not allowed to decide.
func (v *View) widthOf(table produce.Table, column int) float32 {
	widest := measure(table.Columns[column])
	// 🔴 Measured on what the cell will SAY rather than on the value behind it,
	// and the two differ. A cell whose parent lies outside the sample carries
	// words instead of a key, and the first drawing sized that column for the
	// empty string it replaced, so the words came out truncated. One function
	// answers what is in a cell, and both the measuring and the drawing ask it.
	for row := range table.Rows {
		if width := measure(v.cellText(table, row, column)); width > widest {
			widest = width
		}
	}
	// 🔴 One step more than the widget says it needs, and the step is measured
	// rather than chosen. A cell reports the width at which it exactly fits, and
	// the table takes its own spacing out of the column before handing the rest
	// to the cell, so a column sized to the reported minimum truncates the value
	// it was sized for. Rendering showed it: an eleven digit number in a column
	// wide enough for eleven digits came out as nine and an ellipsis.
	widest += palette.StepOf(palette.NameStepTight)
	if widest < palette.PreviewColumnMin {
		return palette.PreviewColumnMin
	}
	if widest > palette.PreviewColumnMax {
		return palette.PreviewColumnMax
	}
	return widest
}

// measure asks the toolkit how wide a piece of text is on the face it will
// actually draw with, padding included, rather than counting characters.
func measure(text string) float32 {
	// 🔴 Measured with the widget the cell is actually made of, not with a
	// label. Rendering showed why: a label and a rich text pad themselves
	// differently, so a column sized from a label cut an eleven digit number
	// down to nine and drew an ellipsis in a column with room to spare.
	rich := widget.NewRichTextWithText(text)
	rich.Wrapping = fyne.TextWrapOff
	return rich.MinSize().Width
}

// cellHeight is how tall one cell of a table is, asked of the toolkit rather
// than assumed, because a person may scale the interface and every value this
// view draws with belongs to the theme rather than to this file.
func cellHeight() float32 { return widget.NewLabel("0").MinSize().Height }
