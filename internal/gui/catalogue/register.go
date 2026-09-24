package catalogue

import (
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
// screen, because the row itself holds behaviour and no picture.
func valueRow(banded bool) builder {
	return func(text i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		value := samples.Word
		if len(samples.Values) > 0 {
			value = samples.Values[0]
		}
		made := parts.NewTapRow(func(hovered bool) fyne.CanvasObject {
			return quickview.RowFace(1, 1, value, banded, hovered, text.T(i18n.QuickCopy))
		}, nil)
		return made, made
	}
}

// texts is the page of the controls a person types into.
func texts() []control {
	return []control{{
		kind: "FieldEntry", name: i18n.CatalogueField, states: typeable,
		variants: []variant{
			{name: i18n.CatalogueEmpty, build: field(nothing)},
			{name: i18n.CatalogueShort, build: field(word)},
			{name: i18n.CatalogueExtreme, build: field(extreme)},
			{name: i18n.CatalogueLongValue, build: field(long)},
		},
	}, {
		kind: "Editor", name: i18n.CatalogueEditor, states: typeable,
		variants: []variant{
			{name: i18n.CatalogueEmpty, build: editor(nothing)},
			{name: i18n.CatalogueLines, build: editor(severalLines)},
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

// editor is the field of several lines, sized like the single line one so
// that its lines are clipped the way the recipe editor clips a long one.
func editor(value func(Samples) string) builder {
	return func(_ i18n.Catalogue, samples Samples) (any, fyne.CanvasObject) {
		made := parts.NewEditor(nil)
		made.SetText(value(samples))
		return made, paint.AtLeast(paint.FieldWidth(samples.Word), 0, made.Framed())
	}
}

// lists is the page of the controls that show values a row at a time.
//
// The table stands alone, on its side, because one table is wider than half
// the page and two of them side by side would run past its edge.
func lists() []control {
	return []control{{
		kind: "TapRow", name: i18n.CatalogueTapRow, states: hoverable,
		variants: []variant{
			{name: i18n.CatalogueUnbanded, alone: true, fills: true, build: valueRow(false)},
			{name: i18n.CatalogueBanded, alone: true, fills: true, build: valueRow(true)},
		},
	}, {
		kind: "KeyTable", name: i18n.CatalogueTable, states: focusable,
		variants: []variant{{name: i18n.CatalogueRows, alone: true, build: table}},
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
