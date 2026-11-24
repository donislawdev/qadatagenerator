package parts

import (
	"image/color"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// What the recipe editor draws beside and under its text: the number of every
// line, a mark at a line the reader found a problem on, and a band across the
// line the caret is on (DEC-337).
//
// # Why the rows come from the toolkit rather than from a count of our own
//
// The editor folds a long line at a word (NewEditor says why), so one line of
// the recipe can be several rows on the screen, and the number belongs beside
// the first of them only. Where the toolkit folds depends on the width, the
// face, and the measure of every character, CJK and emoji included, and its
// own algorithm is private (wrapWordLines in widget/richtext.go of v2.8.2). A
// second copy of it here would agree with the toolkit until the first release
// that moves a fold by one character, and then put numbers beside the wrong
// rows with nothing to say so. So every row is asked of the toolkit through
// what it exports: where the row begins in the text (CursorTextOffset at a row
// and a column of nought, the same road caretTo takes), and where the first
// row stands (CursorPosition, which is where the toolkit draws the caret on
// it), every row after it being one caret's height lower (rows says why).
//
// # Why the field's cursor is moved to ask, and why that is safe
//
// The toolkit answers both questions only about the row and column its cursor
// is on, so each question moves the two exported fields and the answer is read
// at once. The fields are put back before anything else runs: nothing in the
// toolkit reads them between the two lines, nothing is redrawn, and the second
// copy the toolkit keeps for drawing a selection is never touched. The cost is a
// constant per row (textPosFromRowCol in widget/selectable.go), and only the rows
// in sight are asked about.
//
// # Why a number is counted by newlines
//
// The reader names a problem by the line of the FILE it is on, and a line of
// the file ends at a newline whatever the window does with its width. Numbering
// the rows on the screen instead would put "line 7" of a message beside some
// other line than the seventh.

// lineMap is where each line of a text begins, counted in characters, so that a
// place in the text can be named by its line without walking the text again.
type lineMap struct {
	text  string
	runes []rune
	// starts holds where each line begins: starts[0] is the first line.
	starts []int
}

// newLineMap reads where the lines of a text begin.
func newLineMap(text string) lineMap {
	runes := []rune(text)
	starts := []int{0}
	for at, r := range runes {
		if r == '\n' {
			starts = append(starts, at+1)
		}
	}
	return lineMap{text: text, runes: runes, starts: starts}
}

// count is how many lines the text has. A text ending in a newline has an
// empty last line, which is where the caret goes after Enter, so it is counted.
func (m lineMap) count() int { return len(m.starts) }

// lineOf is the line, counted from one, that holds a place in the text.
func (m lineMap) lineOf(place int) int {
	return sort.Search(len(m.starts), func(i int) bool { return m.starts[i] > place })
}

// begins reports whether a place in the text is the first character of a line.
func (m lineMap) begins(place int) bool {
	return place == 0 || (place <= len(m.runes) && m.runes[place-1] == '\n')
}

// textStart is where the words of a line begin, past its indentation, for a
// line counted from one that the text has.
func (m lineMap) textStart(line int) int {
	at := m.starts[line-1]
	end := len(m.runes)
	if line < len(m.starts) {
		end = m.starts[line] - 1
	}
	for at < end && m.runes[at] == ' ' {
		at++
	}
	return at
}

// lineView is what the column and the band draw at one moment: the rows in
// sight that begin a line, and where the line the caret is on stands.
type lineView struct {
	// rows are the rows in sight that begin a line, from the top.
	rows []viewRow
	// height is the height of one row.
	height float32
	// current is the line the caret is on, or nought when the keyboard is
	// elsewhere, because the caret is drawn only while it is in the field.
	current int
	// band is where the band across that line stands, and shown says whether
	// there is one at all.
	band span
}

// viewRow is a row in sight that begins a line.
type viewRow struct {
	// top is where the row stands, from the top of the editor.
	top float32
	// line is the line it begins, counted from one.
	line int
}

// span is a stretch of the editor from top to bottom.
type span struct {
	top, bottom float32
	shown       bool
}

// lineMap is the map of the text in the field now, read again only when the
// text has changed since the last question.
func (e *Editor) lineMap() lineMap {
	if e.lines.starts == nil || e.lines.text != e.Text {
		e.lines = newLineMap(e.Text)
	}
	return e.lines
}

// rowHeight is the height of one row of the field, measured the way the toolkit
// measures the height of its caret: the letter M in the face and size of the
// field (charMinSize in widget/richtext.go of v2.8.2).
func (e *Editor) rowHeight() float32 {
	return fyne.MeasureText("M", e.Theme().Size(theme.SizeNameText), e.TextStyle).Height
}

// rows says where the rows of the field stand in its scrolling content: the
// top of the first, and the height every row has. Where each row BEGINS in the
// text is still asked of the toolkit row by row (rowStart), which costs it
// nothing, because that is where it folded the line.
//
// # Why the toolkit is asked about the first row only
//
// The toolkit places its caret on a row at the row's index times the height
// of the caret (CursorPosition), and lays its rows out one caret's height
// apart, so one answer and the height give every other row. Asking row by row
// cost the whole text each time instead: CursorPosition turns the whole text
// of the field into characters for every question (lineSizeToColumn in
// widget/richtext.go of v2.8.2), which measured 23 ms for the rows in sight of
// a recipe of 256 KB on 2026-11-24, as much on a scroll as on a key press.
//
// It moves the cursor to ask, and the caller puts it back.
func (e *Editor) rows() (first, height float32) {
	e.CursorRow, e.CursorColumn = 0, 0
	return e.CursorPosition().Y, e.rowHeight()
}

// view works out what the column and the band draw now, and false when the
// field's scrolling area could not be found, in which case nothing is drawn: a
// number beside the wrong row is worse than no number.
func (e *Editor) view() (lineView, bool) {
	area := e.area
	if area == nil || area.Content == nil {
		return lineView{}, false
	}
	lines := e.lineMap()
	row, column := e.CursorRow, e.CursorColumn
	caret := e.CursorTextOffset()
	defer func() { e.CursorRow, e.CursorColumn = row, column }()

	first, height := e.rows()
	shown := lineView{height: height}
	inSight := area.Position().Y
	// shift moves a place in the scrolling content to a place in the editor.
	shift := inSight + first - area.Offset.Y
	bottom := inSight + area.Size().Height
	// The first row in sight is worked out from the height of a row rather than
	// found by asking every row above it, which on a recipe of thousands of
	// lines would be thousands of questions per key press.
	from := 0
	if height > 0 {
		from = max(int((inSight-shift)/height)-1, 0)
	}
	for at := from; ; at++ {
		begin, there := e.rowStart(at)
		y := shift + float32(at)*height
		if !there || y >= bottom {
			break
		}
		if y+height > inSight && lines.begins(begin) {
			shown.rows = append(shown.rows, viewRow{top: y, line: lines.lineOf(begin)})
		}
	}
	if e.keyboard && !e.Disabled() {
		shown.current = lines.lineOf(caret)
		shown.band = e.bandAround(row, lines, shift, height, inSight, bottom)
	}
	return shown, true
}

// bandAround is where the band across the line holding a row stands: from the
// first row of that line to the last, cut at the edges of what is in sight.
//
// There is no band while anything is selected. The band and a selection are
// both grounds under the text, and the selection is the one a person made on
// purpose, so it is never drawn on another ground (ThresholdSelection).
func (e *Editor) bandAround(row int, lines lineMap, shift, height, inSight, bottom float32) span {
	if e.SelectedText() != "" {
		return span{}
	}
	first := row
	for first > 0 {
		begin, _ := e.rowStart(first)
		if lines.begins(begin) {
			break
		}
		first--
	}
	last := row
	for {
		begin, there := e.rowStart(last + 1)
		if !there || lines.begins(begin) {
			break
		}
		last++
	}
	band := span{top: max(shift+float32(first)*height, inSight),
		bottom: min(shift+float32(last+1)*height, bottom)}
	band.shown = band.bottom > band.top
	return band
}

// redrawLines puts the numbers, the marks and the band where the text is now.
//
// It is called whenever the rows can have moved: the caret moved or the text
// changed (OnCursorChanged, which the toolkit calls after it scrolled to the
// caret), the area was scrolled by the wheel or its bar (OnScrolled), the field
// was laid out at another width, and the keyboard arrived or left.
func (e *Editor) redrawLines() {
	if e.marks == nil {
		return
	}
	shown, found := e.view()
	if !found {
		shown = lineView{}
	}
	e.marks.show(shown, e.problems)
}

// MarkLines marks the lines, counted from one, that the reader found a problem
// on, in place of the lines marked before. A line the text no longer has is
// left unmarked: the answer was about an older text, and the next one replaces
// it.
func (e *Editor) MarkLines(lines []int) {
	e.problems = make(map[int]bool, len(lines))
	for _, line := range lines {
		e.problems[line] = true
	}
	e.redrawLines()
}

// Marked is the lines marked now, from the top, for a caller with nobody
// looking at the screen.
func (e *Editor) Marked() []int {
	marked := make([]int, 0, len(e.problems))
	for line := range e.problems {
		marked = append(marked, line)
	}
	sort.Ints(marked)
	return marked
}

// GoToLine puts the caret where the words of a line begin, counted from one,
// gives the field the keyboard and brings the line into sight.
//
// A line past the end of the text goes to the last line rather than nowhere:
// it comes from an answer about an older text, and the end is the nearest
// place that still exists. Whatever was selected stops being selected, because
// the caret goes somewhere else.
func (e *Editor) GoToLine(line int) {
	lines := e.lineMap()
	line = min(max(line, 1), lines.count())
	if e.SelectedText() != "" {
		// Left with no Shift ends the selection, as in outdent.
		e.Entry.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	e.caretTo(lines.textStart(line))
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	if canvas := app.Driver().CanvasForObject(e); canvas != nil {
		canvas.Focus(e)
	}
}

// LineOfCaret is the line the caret is on, counted from one.
func (e *Editor) LineOfCaret() int { return e.lineMap().lineOf(e.CursorTextOffset()) }

// indentAfter is how many spaces begin the line after one that Enter ended:
// the spaces that began the line, or, when the line opens a block by ending in
// a colon, one indent deeper than where its words begin.
//
// Where the words begin counts the dash of a list item as indentation, because
// the keys of a mapping inside an item line up with the first key after the
// dash, not with the dash. A comment after the colon does not stop the line
// from opening a block, and a colon inside quotes does not open one.
func indentAfter(line string) int {
	spaces := len(line) - len(strings.TrimLeft(line, " "))
	words := strings.TrimRight(withoutComment(line), " \t")
	if !strings.HasSuffix(words, ":") {
		return spaces
	}
	column := spaces
	rest := line[spaces:]
	for strings.HasPrefix(rest, "- ") {
		trimmed := strings.TrimLeft(rest[1:], " ")
		column += len(rest) - len(trimmed)
		rest = trimmed
	}
	return column + len(indent)
}

// withoutComment is a line of YAML with its comment taken off: a hash that
// begins the line or follows a space or a tab, outside quotes.
func withoutComment(line string) string {
	var quote rune
	escaped := false
	before := ' '
	for at, r := range line {
		switch {
		case escaped:
			escaped = false
		case quote == '"' && r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '#' && (before == ' ' || before == '\t'):
			return line[:at]
		}
		before = r
	}
	return line
}

// indentNewLine types the indentation of the line Enter just began.
//
// The line above the caret is the one Enter ended, and what stands on it is
// what was before the caret when the key was pressed, a selection already
// taken away by the toolkit. When the toolkit did not type a new line, which
// is what it does when a field is submitted, nothing is typed. The spaces go
// through the toolkit's own handling of a key, like the indent of Tab, so
// Ctrl+Z takes them back one at a time.
func (e *Editor) indentNewLine() {
	text := []rune(e.Text)
	at := min(max(e.CursorTextOffset(), 0), len(text))
	if at == 0 || text[at-1] != '\n' {
		return
	}
	start := at - 1
	for start > 0 && text[start-1] != '\n' {
		start--
	}
	for range indentAfter(string(text[start : at-1])) {
		e.Entry.TypedRune(' ')
	}
}

// lineMarks is everything the editor draws beside and under its text: the
// column of numbers at its leading edge, the marks in that column, and the
// band across the line of the caret under the text.
//
// # Why one widget under the whole field rather than one per thing
//
// The column and the band are drawn from the same answer about the rows
// (view), at the same moments, and both have to be cut where the field is cut.
// One widget laid under the column and the text holds both, and the text of
// the toolkit lies on top of it, so the band is a ground under the words. The
// column is a gap left by the layout beside the text (room), with this widget
// showing through it. That keeps this to one entry in the register of what the
// window builds for itself rather than three.
//
// It answers the pointer where the column is, the way the text beside it does,
// so that the ground of the field stays lit while the pointer crosses into the
// column, and it hands the wheel to the field's area there, so that the column
// is not a strip of the field the wheel does nothing over. Over the text the
// toolkit's field lies on top and answers first.
type lineMarks struct {
	widget.BaseWidget
	editor *Editor
	// holder carries every piece, placed by hand: the pieces are put inside one
	// container rather than handed out as the renderer's own objects, for the
	// reason tapRowRenderer gives.
	holder *fyne.Container
	// room is the gap the layout leaves for the column, as wide as width.
	room    *canvas.Rectangle
	band    *canvas.Rectangle
	at      span
	numbers []*canvas.Text
	marks   []*fyne.Container
}

var (
	_ fyne.Widget       = (*lineMarks)(nil)
	_ fyne.Scrollable   = (*lineMarks)(nil)
	_ desktop.Hoverable = (*lineMarks)(nil)
)

func newLineMarks(editor *Editor) *lineMarks {
	band := canvas.NewRectangle(palette.Ink(palette.BgCurrent))
	band.Hide()
	marks := &lineMarks{editor: editor, holder: container.NewWithoutLayout(band), band: band,
		room: canvas.NewRectangle(color.Transparent)}
	marks.ExtendBaseWidget(marks)
	marks.room.SetMinSize(fyne.NewSize(marks.width(), 0))
	return marks
}

// digits is how many digits the column is wide enough for now.
func (m *lineMarks) digits() int {
	return max(palette.GutterDigitsAtLeast, len(strconv.Itoa(m.editor.lineMap().count())))
}

// width is how wide the column is: a mark, a step, the digits, and a step
// before the text.
func (m *lineMarks) width() float32 {
	digit := fyne.MeasureText("0", m.editor.Theme().Size(theme.SizeNameText), m.editor.TextStyle)
	return palette.IconSmall + palette.StepOf(palette.NameStepHairline) +
		float32(m.digits())*digit.Width + palette.StepOf(palette.NameStepRelated)
}

// show puts a number beside every row in sight that begins a line, in the ink
// of the line the caret is on or of any other, a mark beside the lines the
// reader found a problem on, and the band where the view says.
//
// The gap for the column is given its width again first, which changes only
// when the count of lines passes a power of ten past the least digits. The
// window lays the field out again on its next frame when the gap asks for
// another width, the way it does for any object whose least size changed.
func (m *lineMarks) show(shown lineView, problems map[int]bool) {
	width := m.width()
	if m.room.MinSize().Width != width {
		m.room.SetMinSize(fyne.NewSize(width, 0))
	}
	size := m.editor.Theme().Size(theme.SizeNameText)
	right := width - palette.StepOf(palette.NameStepRelated)
	objects := []fyne.CanvasObject{m.band}
	marked := 0
	for at, row := range shown.rows {
		number := m.number(at)
		number.Text = strconv.Itoa(row.line)
		number.TextSize = size
		number.Color = palette.Ink(palette.TextSecondary)
		if row.line == shown.current {
			number.Color = palette.Ink(palette.TextPrimary)
		}
		number.Resize(number.MinSize())
		number.Move(fyne.NewPos(right-number.MinSize().Width, row.top))
		objects = append(objects, number)
		if problems[row.line] {
			// The mark stands in a box as tall as the row, which centres it on
			// the row the way the number beside it is.
			mark := m.mark(marked)
			mark.Resize(fyne.NewSize(palette.IconSmall, shown.height))
			mark.Move(fyne.NewPos(0, row.top))
			objects = append(objects, mark)
			marked++
		}
	}
	m.holder.Objects = objects
	m.at = shown.band
	m.placeBand(m.Size().Width)
	m.holder.Refresh()
}

// placeBand lays the band across the text, from the end of the column to the
// far edge of a width.
func (m *lineMarks) placeBand(width float32) {
	if !m.at.shown {
		m.band.Hide()
		return
	}
	column := m.room.MinSize().Width
	m.band.Move(fyne.NewPos(column, m.at.top))
	m.band.Resize(fyne.NewSize(width-column, m.at.bottom-m.at.top))
	m.band.Show()
}

// number is the text of the n-th number from the top, kept between frames so
// that a key press allocates nothing for the rows that stayed in sight.
func (m *lineMarks) number(n int) *canvas.Text {
	for len(m.numbers) <= n {
		text := canvas.NewText("", palette.Ink(palette.TextSecondary))
		text.TextStyle = fyne.TextStyle{Monospace: true}
		m.numbers = append(m.numbers, text)
	}
	return m.numbers[n]
}

// mark is the n-th mark from the top, kept the same way.
func (m *lineMarks) mark(n int) *fyne.Container {
	for len(m.marks) <= n {
		icon := paint.Icon(paint.IconAlert, palette.InkDanger, palette.IconSmall)
		m.marks = append(m.marks, container.NewCenter(icon))
	}
	return m.marks[n]
}

// Shown says whether the band is drawn, and where, for a test that would
// otherwise have to read pixels.
func (m *lineMarks) Shown() (top, bottom float32, shown bool) {
	return m.at.top, m.at.bottom, m.at.shown && m.band.Visible()
}

// Scrolled hands the wheel to the field's area, the way wheelLayer does over
// the text.
func (m *lineMarks) Scrolled(event *fyne.ScrollEvent) {
	if area := m.editor.area; area != nil {
		area.Scrolled(wheelTurned(event, area.Size().Height))
	}
}

// MouseIn lights the field, as the pointer on its text does.
func (m *lineMarks) MouseIn(event *desktop.MouseEvent) { m.editor.MouseIn(event) }

// MouseMoved changes nothing the ground shows.
func (m *lineMarks) MouseMoved(*desktop.MouseEvent) {}

// MouseOut puts the ground back, as the pointer leaving the text does.
func (m *lineMarks) MouseOut() { m.editor.MouseOut() }

// CreateRenderer draws the holder, cut at the field's edges, and lays the band
// across the width whenever that changes.
func (m *lineMarks) CreateRenderer() fyne.WidgetRenderer {
	return &clipped{holder: m.holder, laid: m.placeBand}
}

// clipped is a renderer of one holder whose pieces are placed by hand, which
// the driver cuts at the widget's own edges.
//
// # Why the cut matters, and how it is asked for
//
// A number half scrolled out of the field has to be half drawn, as the text
// beside it is, and Fyne does not cut anything by default: a piece placed above
// the widget is drawn over whatever is there. The driver cuts the children of
// a widget whose renderer has a method named IsClip, internal/driver/util.go:209
// of v2.8.2, which is the toolkit's own way of asking for it.
type clipped struct {
	holder *fyne.Container
	// laid is told the width whenever the widget is laid out, for pieces that
	// stretch across it.
	laid func(float32)
}

func (r *clipped) Layout(size fyne.Size) {
	r.holder.Resize(size)
	r.laid(size.Width)
}

// MinSize is nothing: the widget takes the size it is laid out at, and the
// pieces it holds are placed rather than measured.
func (r *clipped) MinSize() fyne.Size           { return fyne.Size{} }
func (r *clipped) Refresh()                     { r.holder.Refresh() }
func (r *clipped) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.holder} }
func (r *clipped) Destroy()                     {}

// IsClip is the method the driver looks for. It does nothing: being there is
// the whole of what it says.
func (r *clipped) IsClip() {}
