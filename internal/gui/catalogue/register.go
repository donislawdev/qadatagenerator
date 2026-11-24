package catalogue

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/gui/quickview"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// control is one type of internal/gui/parts as the catalogue shows it: the
// states it is shown in, one per column, and its variants, one per row.
type control struct {
	// kind is the name of the type in internal/gui/parts. The guard beside
	// this file reads which types the screens build out of the source, and
	// closes this register against them in both directions.
	kind     string
	name     i18n.Key
	states   []state
	variants []variant
}

// variant is one row: the control built one of the ways a screen builds it.
type variant struct {
	name i18n.Key
	// alone gives this row a cell size of its own. A variant carrying much
	// more than the rest, a very long label, would otherwise make every cell
	// of the control as wide as its own and spread the other rows across the
	// page.
	alone bool
	// fills gives the control the whole width of the row it stands in, which
	// is how a screen lays out a control that is a row itself, and it is
	// honoured only for a variant standing alone. Drawn at the width it asks
	// for, a row of a value puts the word under the pointer against the value,
	// where the screen puts it at the far edge.
	fills bool
	// build makes a fresh control and what stands on the page for it, which
	// is the control dressed the way a screen dresses it. The states are put
	// on the first and the second is what is drawn.
	build builder
}

type builder func(text i18n.Catalogue, samples Samples) (any, fyne.CanvasObject)

// The states each family is shown in. Which states a control answers to is
// not a choice made here: the guard beside this file draws every control in
// every state it takes, and refuses both a state missing from these lists and
// a state on them that changes nothing.
var (
	pressable = []state{rest, pointer, pressed, focus, disabled}
	typeable  = []state{rest, pointer, focus, disabled}
	hoverable = []state{rest, pointer}
	focusable = []state{rest, focus}
	still     = []state{rest}
)

// buttons is the page of the one control most screens are built out of.
func buttons() []control {
	return []control{{
		kind: "Button", name: i18n.CatalogueButton, states: pressable,
		variants: []variant{
			{name: i18n.CatalogueOutlined, build: button(false, "")},
			{name: i18n.CatalogueIcon, build: button(false, paint.IconCopy)},
			{name: i18n.CatalogueFilled, build: button(true, "")},
			{name: i18n.CataloguePlace, build: place(false)},
			{name: i18n.CatalogueCurrent, build: place(true)},
			{name: i18n.CatalogueLongLabel, alone: true, build: longButton},
		},
	}}
}

// button is an action, outlined or filled, with or without an icon, built by
// the constructor the screens call for it.
func button(filled bool, icon string) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewButton(samples.Word, nil)
		if filled {
			made = parts.NewPrimaryButton(samples.Word, nil)
		}
		if icon != "" {
			made.ShowIcon(icon)
		}
		return made, made.OnScreen()
	}
}

// place is a name in the column of screens, which is also how the heading of a
// section that opens is drawn: plain, from the leading edge, with an icon.
func place(current bool) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewButton(samples.Word, nil)
		made.ShowIcon(paint.IconBolt)
		made.Leading, made.Plain, made.Current = true, true, current
		return made, made.OnScreen()
	}
}

func longButton(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made := parts.NewButton(samples.Long, nil)
	return made, made.OnScreen()
}

// choosers is the page of the control a person picks one value with.
func choosers() []control {
	return []control{{
		kind: "Select", name: i18n.CatalogueSelect, states: pressable,
		variants: []variant{
			{name: i18n.CatalogueNothing, build: chooser(false)},
			{name: i18n.CatalogueChosen, build: chooser(true)},
			{name: i18n.CatalogueLongValue, alone: true, build: longChooser},
		},
	}}
}

// checks is the page of the control a person ticks.
func checks() []control {
	return []control{{
		kind: "Check", name: i18n.CatalogueCheck, states: pressable,
		variants: []variant{
			{name: i18n.CatalogueOff, build: check(false)},
			{name: i18n.CatalogueOn, build: check(true)},
			{name: i18n.CatalogueLongLabel, alone: true, build: longCheck},
		},
	}}
}

func chooser(chosen bool) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewSelect(samples.Values, nil)
		if chosen && len(samples.Values) > 0 {
			made.SetSelected(samples.Values[0])
		}
		return made, made.OnScreen()
	}
}

func longChooser(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made := parts.NewSelect(append(append([]string{}, samples.Values...), samples.Long), nil)
	made.SetSelected(samples.Long)
	return made, made.OnScreen()
}

func check(on bool) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewCheck(samples.Word, nil)
		made.SetChecked(on)
		return made, made.OnScreen()
	}
}

func longCheck(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made := parts.NewCheck(samples.Long, nil)
	return made, made.OnScreen()
}

// valueRow is a row of the list of values on the quick screen, drawn by that
// screen, because the row itself holds behaviour and no picture. The index
// says which row it is, and the screen bands every second one.
func valueRow(at int) builder {
	return func(text i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		value := samples.Word
		if len(samples.Values) > 0 {
			value = samples.Values[0]
		}
		made := parts.NewTapRow(func(hovered bool) fyne.CanvasObject {
			return quickview.RowFace(at, 1, value, hovered, rowHint(text, hovered))
		}, nil)
		return made, made
	}
}

// rowHint is what a row of values says under the pointer and nothing
// otherwise, as on the quick screen.
func rowHint(text i18n.Catalogue, hovered bool) string {
	if hovered {
		return text.T(i18n.QuickCopy)
	}
	return ""
}

// valueList is the list of values on the quick screen, built by that screen,
// holding more values than it shows: the samples once for every row of a
// control's height the area stands, so several times what it shows.
func valueList(text i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	var values []string
	for range palette.CatalogueAreaRows {
		values = append(values, samples.Values...)
	}
	digits := len(strconv.Itoa(len(values)))
	made := quickview.NewValueList()
	made.Hold(len(values), 0, func(row fyne.CanvasObject, at int) {
		if tap, isRow := row.(*parts.TapRow); isRow {
			tap.Rebind(func(hovered bool) fyne.CanvasObject {
				return quickview.RowFace(at, digits, values[at], hovered, rowHint(text, hovered))
			}, nil)
		}
	})
	return made, paint.AtLeast(0, areaHeight(), made.Framed())
}

// texts is the page of the controls a person types into.
func texts() []control {
	return []control{{
		kind: "FieldEntry", name: i18n.CatalogueField, states: typeable,
		variants: []variant{
			{name: i18n.CatalogueEmpty, build: field(nothing)},
			// Alone, because the field is as wide as its hint, and sharing a
			// cell would widen every row of the field until the states no
			// longer stood side by side and the page ran past the window.
			{name: i18n.CatalogueHinted, alone: true, build: hinted},
			// One word with a note under it, and below it the same field and
			// note in the colour of a refusal, which is the pair the count of
			// the quick screen shows (DEC-335). One row rather than a row of a
			// bare word and a row with a note: the page gained one row with the
			// note and ran past the foot of the window it opens in, and a bare
			// field holding words is drawn by the two rows of long values.
			{name: i18n.CatalogueNoted, build: notedField},
			// A row rather than a column, because a screen marks a field as
			// refused for what it holds, which is what a row is (B-363). The
			// guard beside this file holds it apart from the row above.
			{name: i18n.CatalogueRefused, build: refusedField},
			{name: i18n.CatalogueExtreme, build: field(extreme)},
			{name: i18n.CatalogueLongValue, build: field(long)},
		},
	}}
}

// What a text control holds, one sample each.
func nothing(Samples) string        { return "" }
func word(s Samples) string         { return s.Word }
func extreme(s Samples) string      { return s.Extreme }
func long(s Samples) string         { return s.Long }
func severalLines(s Samples) string { return s.Lines }

// field is a text field sized for one word, which is how a screen sizes one:
// for what it usually holds, so that what is longer is clipped, and the
// clipping is what the longer rows are for.
func field(value func(Samples) string) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewFieldEntry()
		made.SetText(value(samples))
		return made, paint.AtLeast(paint.FieldWidth(samples.Word), 0, parts.Standing(parts.Field(made)))
	}
}

// notedField is the field of one word with the note the quick screen stands
// under its count, in the words that screen uses.
func notedField(text i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made := parts.NewFieldEntry()
	made.SetText(word(samples))
	return made, paint.AtLeast(paint.FieldWidth(samples.Word), 0,
		parts.Noted(made, quickview.CountNote(text)))
}

// refusedField is the noted field marked as holding what its screen would
// refuse, which is how the count of the quick screen stands while somebody
// types a number Generate would not take: the edge and the note both red.
func refusedField(text i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made, drawn := notedField(text, samples)
	made.(*parts.FieldEntry).SetInvalid(true)
	return made, drawn
}

// hinted is an empty field showing what leaving it empty means, which is how
// the take, recipe and quick screens each show one of theirs. The hint is one
// the window really shows, and the field is as wide as it, so the row is the
// drawing a person meets rather than a hint cut to the width of one word.
func hinted(text i18n.Catalogue, _ Samples) (any, fyne.CanvasObject) {
	hint := text.T(i18n.TakeSettingOutHint)
	made := parts.NewFieldEntry()
	made.SetPlaceHolder(hint)
	return made, paint.AtLeast(paint.FieldWidth(hint), 0, parts.Standing(parts.Field(made)))
}

// editor is the field of several lines, its text as wide as the single line
// one, so that its lines fold the way the recipe editor folds a long one.
func editor(value func(Samples) string) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewEditor(nil)
		made.SetText(value(samples))
		return made, editorSized(made, samples)
	}
}

// markedEditor is the same editor with its first line marked the way the
// recipe screen marks a line the reader found a problem on (DEC-337), so the
// mark is seen in every state of the field: under the pointer, with the
// keyboard and the band across the caret's line, and out of reach. The first
// line, because it is the one in sight in a field this small.
func markedEditor(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made := parts.NewEditor(nil)
	made.SetText(samples.Lines)
	made.MarkLines([]int{1})
	return made, editorSized(made, samples)
}

// editorSized is an editor framed with room for its text as wide as one word,
// beside the column of line numbers.
//
// 🔴 The column is added to the width rather than taken out of it. Sized like
// the field of one line, as it was until 2026-11-24, the field gave the column
// most of its width once the column existed, and the render showed the sample
// folded one letter to a row (DEC-337).
func editorSized(made *parts.Editor, samples Samples) fyne.CanvasObject {
	framed := made.Framed()
	return paint.AtLeast(paint.FieldWidth(samples.Word)+framed.MinSize().Width, 0, framed)
}

// lists is the page of the controls that show records a row at a time.
//
// The table stands alone, on its side, because one table is wider than half
// the page and two of them side by side would run past its edge.
func lists() []control {
	return []control{{
		kind: "KeyTable", name: i18n.CatalogueTable, states: focusable,
		variants: []variant{{name: i18n.CatalogueRows, alone: true, build: table}},
	}}
}

// values is the page of the list of values on the quick screen: one row under
// the pointer, which no other picture shows, and the list holding more values
// than it shows.
func values() []control {
	return []control{{
		kind: "TapRow", name: i18n.CatalogueTapRow, states: hoverable,
		variants: []variant{
			{name: i18n.CatalogueUnbanded, alone: true, fills: true, build: valueRow(0)},
			{name: i18n.CatalogueBanded, alone: true, fills: true, build: valueRow(1)},
		},
	}, {
		kind: "ScrolledRows", name: i18n.CatalogueValueList, states: focusable,
		variants: []variant{{name: i18n.CatalogueRows, alone: true, fills: true, build: valueList}},
	}}
}

// editors is the page of the text of several lines.
//
// A page of its own since 2026-11-11, when the field of one line gained the
// row of an empty field with a hint: the two together ran past the foot of the
// window, and so did the editor on the page of areas. The numbers stand where
// the pages are counted.
func editors() []control {
	return []control{{
		kind: "Editor", name: i18n.CatalogueEditor, states: typeable,
		variants: []variant{
			{name: i18n.CatalogueEmpty, build: editor(nothing)},
			{name: i18n.CatalogueLines, build: editor(severalLines)},
			{name: i18n.CatalogueMarkedLine, build: markedEditor},
		},
	}}
}

// areas is the page of the controls that hold more than they show, or show
// it only when asked.
func areas() []control {
	return []control{{
		kind: "KeyScroll", name: i18n.CatalogueScroll, states: focusable,
		variants: []variant{{name: i18n.CatalogueOverflowed, alone: true, build: scroll}},
	}, {
		kind: "Disclosure", name: i18n.CatalogueDisclosure, states: still,
		// Alone, because the two are of very different heights, and sharing
		// a cell would give the closed one the height of the open one.
		variants: []variant{
			{name: i18n.CatalogueClosed, alone: true, build: disclosure(false)},
			{name: i18n.CatalogueOpened, alone: true, build: disclosure(true)},
		},
	}}
}

// areaHeight is how tall a table or a scrolling area stands: a few rows, fewer
// than the samples hold.
func areaHeight() float32 { return palette.ControlHeight * palette.CatalogueAreaRows }

// table is the samples in a table, each column as wide as its widest cell and
// the whole no taller than a few rows.
func table(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	columns := 0
	if len(samples.Table) > 0 {
		columns = len(samples.Table[0])
	}
	cell := func(row, column int) string {
		if row < len(samples.Table) && column < len(samples.Table[row]) {
			return samples.Table[row][column]
		}
		return ""
	}
	made := parts.NewKeyTable(
		func() (int, int) { return max(len(samples.Table)-1, 0), columns },
		func() fyne.CanvasObject { return paint.TableCell() },
		func(at widget.TableCellID, object fyne.CanvasObject) {
			paint.SetTableCell(object, cell(at.Row+1, at.Col))
		},
		nil)
	made.ShowHeaderColumn = false
	made.CreateHeader = func() fyne.CanvasObject { return paint.TableCell() }
	made.UpdateHeader = func(at widget.TableCellID, object fyne.CanvasObject) {
		paint.SetTableCell(object, cell(0, at.Col))
	}
	width := float32(0)
	for column := 0; column < columns; column++ {
		widest := float32(0)
		for row := range samples.Table {
			widest = max(widest, paint.FieldWidth(cell(row, column)))
		}
		made.SetColumnWidth(column, widest)
		width += widest
	}
	return made, paint.AtLeast(width, areaHeight(), made.Framed())
}

// scroll is a scrolling area holding more lines than it shows, inside the
// margin a screen gives what it scrolls, so that the mark of focus around the
// area does not touch the words.
func scroll(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
	made := parts.NewKeyScroll(paint.PadAll(palette.NameStepTight, lines(samples.Lines)))
	return made, paint.AtLeast(0, areaHeight(), made.Framed())
}

// disclosure is a heading with the values of the samples under it.
func disclosure(open bool) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewDisclosure(samples.Word, lines(strings.Join(samples.Values, "\n")))
		if open {
			made.Open()
		}
		return made, made.OnScreen()
	}
}

// lines is text of several lines, one piece of canvas text each, since canvas
// text draws one line and a scrolling area needs something taller than itself.
func lines(words string) fyne.CanvasObject {
	each := strings.Split(words, "\n")
	drawn := make([]fyne.CanvasObject, 0, len(each))
	for _, line := range each {
		drawn = append(drawn, paint.Text(line, palette.TypeBody, palette.TextPrimary))
	}
	return paint.Column(palette.NameStepTight, drawn...)
}
