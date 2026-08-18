package manifestview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
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

// View is the manifest browser and what it currently holds.
//
// Nothing is loaded until somebody asks. That is the window half of DEC-35: a
// surface that reads a file the moment it is opened is a surface that acts on
// its own, and this tool writes to other people's databases.
type View struct {
	text    i18n.Catalogue
	window  fyne.Window
	read    Readers
	body    *fyne.Container
	respond *widget.Button

	promised   manifest.Manifest
	hasPromise bool
}

// New builds the screen. The window is what a file chooser has to appear over.
func New(text i18n.Catalogue, window fyne.Window, read Readers) *View {
	view := &View{text: text, window: window, read: read, body: container.NewStack()}
	view.respond = widget.NewButton(text.T(i18n.ManifestOpenResponse), view.chooseResponse)
	// Nothing to compare a response against until a manifest is open, and a
	// control that cannot do anything yet says so by being unavailable rather
	// than by failing after the fact.
	view.respond.Disable()
	view.show(sentence(text.T(i18n.ManifestEmpty)))
	return view
}

// Content is the whole screen.
func (v *View) Content() fyne.CanvasObject {
	open := widget.NewButton(v.text.T(i18n.ManifestOpen), v.chooseManifest)
	bar := container.NewHBox(open, v.respond)
	return inset(container.NewBorder(bar, nil, nil, nil, v.body))
}

// inset keeps content off the edges of the window by the margin of the scale.
//
// The first drawing had none, so every word began at the frame. A margin is not
// decoration here: it is what separates the window from the thing inside it,
// and its size is the one the scale already names for exactly this. It is
// applied once, around the whole screen, rather than around each part, because
// two of them stacked read as a gap somebody forgot to close.
func inset(content fyne.CanvasObject) fyne.CanvasObject {
	margin := palette.StepOf(palette.NameStepSection)
	return container.New(layout.NewCustomPaddedLayout(margin, margin, margin, margin), content)
}

func (v *View) show(content fyne.CanvasObject) {
	v.body.Objects = []fyne.CanvasObject{content}
	v.body.Refresh()
}

// sentence is the screen when there is nothing to show.
//
// A sentence rather than an empty panel, which is point eight of the GUI
// checklist. An empty rectangle reads as a fault in the program, and a person
// meeting one has no way to tell it from a screen that is still loading.
func sentence(words string) fyne.CanvasObject {
	label := widget.NewLabel(words)
	label.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(label)
}

func (v *View) chooseManifest() {
	dialog.ShowFileOpen(func(chosen fyne.URIReadCloser, err error) {
		path, ok := v.pathOf(chosen, err)
		if !ok {
			return
		}
		promised, readErr := v.read.Manifest(path)
		if readErr != nil {
			v.hasPromise = false
			v.respond.Disable()
			v.show(sentence(v.text.T(i18n.ManifestUnreadable) + " " + readErr.Error()))
			return
		}
		v.promised, v.hasPromise = promised, true
		v.respond.Enable()
		v.show(v.rendered(Build(v.text, promised, nil)))
	}, v.window)
}

func (v *View) chooseResponse() {
	dialog.ShowFileOpen(func(chosen fyne.URIReadCloser, err error) {
		path, ok := v.pathOf(chosen, err)
		if !ok || !v.hasPromise {
			return
		}
		actual, readErr := v.read.Response(path)
		if readErr != nil {
			v.show(sentence(v.text.T(i18n.ManifestUnreadable) + " " + readErr.Error()))
			return
		}
		comparison := Comparison{Record: actual, Report: verify.Compare(v.promised, actual)}
		v.show(v.rendered(Build(v.text, v.promised, &comparison)))
	}, v.window)
}

// pathOf reads a choice, and reports whether there is anything to do about it.
//
// A cancelled chooser hands back nothing and no error, and it leaves the screen
// exactly as it was. Turning a cancellation into an error message would tell a
// person something went wrong when they were the one who decided nothing
// should.
func (v *View) pathOf(chosen fyne.URIReadCloser, err error) (string, bool) {
	if err != nil {
		v.show(sentence(v.text.T(i18n.ManifestUnreadable) + " " + err.Error()))
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

// rendered draws a screen value.
func (v *View) rendered(screen Screen) fyne.CanvasObject {
	head := container.NewVBox(
		panel(fields(screen.Run, screen.Counts)),
		lines(screen.Notes),
		heading(v.text.T(i18n.ManifestHeadingEntries)),
	)
	return container.NewBorder(head, v.foot(screen), nil, nil, v.table(screen.Rows))
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
func (v *View) table(rows []Row) fyne.CanvasObject {
	entries := widget.NewTableWithHeaders(
		func() (int, int) { return len(rows), palette.EntryColumns() },
		func() fyne.CanvasObject { return painted(Line{}) },
		func(cell widget.TableCellID, object fyne.CanvasObject) {
			if text, ok := object.(*widget.RichText); ok && cell.Row < len(rows) {
				set(text, rows[cell.Row].Cell(cell.Col))
			}
		})
	entries.ShowHeaderColumn = false
	entries.CreateHeader = func() fyne.CanvasObject { return bold(Line{Color: palette.NameTextSubdued}) }
	entries.UpdateHeader = func(cell widget.TableCellID, object fyne.CanvasObject) {
		if text, ok := object.(*widget.RichText); ok {
			set(text, Line{Text: v.columnName(cell.Col), Color: palette.NameTextSubdued})
		}
	}
	for column, width := range palette.EntryColumnWidths {
		entries.SetColumnWidth(column, width)
	}
	return entries
}

// columnNames are the headings, in column order, matching Row.Cells.
var columnNames = []i18n.Key{
	i18n.ManifestColumnMutator, i18n.ManifestColumnPlace, i18n.ManifestColumnExpect,
	i18n.ManifestColumnConfidence, i18n.ManifestColumnStatus, i18n.ManifestColumnDetail,
}

func (v *View) columnName(column int) string {
	if column < 0 || column >= len(columnNames) {
		return ""
	}
	return v.text.T(columnNames[column])
}

// panel lifts what it holds off the background.
//
// 🔴 Depth in a dark theme is carried by light rather than by shadow: a shadow
// against a dark ground is invisible, so a surface that sits above another one
// is drawn lighter. This project measured such a surface into its palette
// before any window existed and then drew its first screen without using it at
// all, so everything sat flat on one tone and nothing said where one part of
// the screen ended and the next began.
func panel(content fyne.CanvasObject) fyne.CanvasObject {
	background := canvas.NewRectangle(palette.Named(palette.NameSurface).NRGBA())
	padding := palette.StepOf(palette.NameStepTight)
	padded := container.New(layout.NewCustomPaddedLayout(padding, padding, padding, padding), content)
	return container.NewStack(background, padded)
}

// fields lays the envelope out as label and value pairs.
//
// # Why a form rather than columns of a fixed width
//
// A form column is as wide as the widest label in it and no wider, worked out
// from the labels themselves. That is what the envelope needs and what the
// previous attempt got wrong twice: first a single row of eleven pairs, which
// reached the frame of a window two thousand pixels wide and would have been
// cut off in anything narrower, then cells of a fixed size, which trimmed every
// value in the envelope down to an ellipsis.
//
// Two forms side by side rather than one, because eleven pairs in one column is
// a tall block of a screen whose subject is the table under it.
func fields(run, counts []Field) fyne.CanvasObject {
	all := append(append([]Field{}, run...), counts...)
	half := (len(all) + 1) / palette.EnvelopeColumns
	return container.NewHBox(form(all[:half]), form(all[half:]))
}

// form is one column of label and value pairs.
//
// 🔴 The pairs are built out of text that does NOT trim itself, and that is the
// whole of why this function exists rather than reusing the cells of the table.
// Read in the toolkit's source on 2026-08-18: text that may truncate is cut to
// the width the layout gave it, and a form works out the width of its label
// column from the smallest size those same objects report. The two together are
// a circle, and the screen it produced showed an envelope of nothing but
// ellipses. A cell of a table is the opposite case, where the column width is
// set and the text has to fit it.
func form(fields []Field) fyne.CanvasObject {
	cells := make([]fyne.CanvasObject, 0, len(fields)*palette.EnvelopeColumns)
	for _, field := range fields {
		cells = append(cells, plain(Line{Text: field.Label, Color: palette.NameTextSubdued}, false))
		cells = append(cells, plain(Line{Text: field.Value}, true))
	}
	return container.New(layout.NewFormLayout(), cells...)
}

// plain is a phrase that asks for as much width as it needs.
func plain(line Line, heavy bool) *widget.RichText {
	segment := segmentOf(line)
	if heavy {
		segment.Style.TextStyle = fyne.TextStyle{Bold: true}
	}
	return widget.NewRichText(segment)
}

// foot is what the comparison could not settle and could not see.
//
// Below the table rather than above it, because it is read after the entries
// and because it is empty in the first state. It carries no scrolling of its
// own, which is a limit: a comparison that settles nothing at all would push
// the table up. Recorded rather than guessed at.
func (v *View) foot(screen Screen) fyne.CanvasObject {
	// The legend says once what the mark on a soft expectation means. It used
	// to be spelled out in every row, which cost the width of a column to say
	// the same sentence sixty times and told a reader nothing about how the
	// entries differ.
	parts := []fyne.CanvasObject{wrapped(Line{
		Text: v.text.T(i18n.ManifestLegendSoft), Color: palette.NameTextSubdued,
	})}
	if len(screen.Unsettled) > 0 {
		parts = append(parts, heading(v.text.T(i18n.ManifestHeadingUnsettled)), lines(screen.Unsettled))
	}
	if len(screen.Limits) > 0 {
		parts = append(parts, heading(v.text.T(i18n.ManifestHeadingLimits)), lines(screen.Limits))
	}
	return panel(container.NewVBox(parts...))
}

func heading(words string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(words, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func lines(all []Line) fyne.CanvasObject {
	if len(all) == 0 {
		return container.NewVBox()
	}
	drawn := make([]fyne.CanvasObject, 0, len(all))
	for _, line := range all {
		drawn = append(drawn, wrapped(line))
	}
	return container.NewVBox(drawn...)
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
// rule 4.7 of docs/17, and it takes a colour by name because the theme now
// answers for the names of this palette as well as for the toolkit's own.
func painted(line Line) *widget.RichText {
	text := widget.NewRichText(segmentOf(line))
	text.Truncation = fyne.TextTruncateEllipsis
	return text
}

// wrapped is a phrase that may take as many lines as it needs.
//
// The sections below the table are sentences rather than cells, so trimming
// them would drop the half that says what could not be settled.
func wrapped(line Line) *widget.RichText {
	text := widget.NewRichText(segmentOf(line))
	text.Wrapping = fyne.TextWrapWord
	return text
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
		name = palette.NameText
	}
	return &widget.TextSegment{
		Text:  line.Text,
		Style: widget.RichTextStyle{ColorName: fyne.ThemeColorName(name), Inline: true},
	}
}

// set writes one phrase into a piece of text that is already on screen.
//
// The weight of the segment is kept rather than rewritten, so a heading stays a
// heading as cells are reused. The refresh is what has the new words rasterised:
// assigning to the segment changes what the object holds, and the screen goes on
// showing what was last drawn until something asks for it again, which matters
// here because every cell of a table was already drawn as some other cell.
func set(target *widget.RichText, line Line) {
	name := line.Color
	if name == "" {
		name = palette.NameText
	}
	segment, ok := target.Segments[0].(*widget.TextSegment)
	if !ok {
		return
	}
	segment.Text = line.Text
	segment.Style.ColorName = fyne.ThemeColorName(name)
	target.Refresh()
}
