package manifestview

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// styleOf asks the table to fill a fresh cell and reports how it was drawn.
//
// It goes through the table's own update rather than through the decision
// beside it, so what is checked is the path a drawn cell actually takes.
func styleOf(t *testing.T, entries *parts.KeyTable, cell widget.TableCellID) fyne.TextStyle {
	t.Helper()
	object := entries.CreateCell()
	entries.UpdateCell(cell, object)
	text, ok := object.(*widget.RichText)
	if !ok {
		t.Fatalf("a cell of the table is %T rather than a phrase", object)
	}
	segment, ok := text.Segments[0].(*widget.TextSegment)
	if !ok {
		t.Fatalf("a cell of the table holds %T rather than a piece of text", text.Segments[0])
	}
	return segment.Style.TextStyle
}

func marked(style fyne.TextStyle) bool { return style.Bold && style.Underline }

// The keyboard can be seen in the table: the cell it stands on is marked, the
// mark moves with it, and it leaves when the keyboard does.
//
// 🔴 The last of those three is the one worth the file. The obvious way to know
// whether the table has focus is to ask the canvas, and measured on 2026-08-18
// that answer is wrong exactly when it matters: the table redraws every cell
// from inside FocusLost, and at that moment the canvas still names the table as
// focused. A mark drawn on that answer stays behind on a table nobody is in,
// pointing at a place the keyboard has left.
func TestTheCellTheKeyboardStandsOnIsMarkedAndTheMarkLeavesWithIt(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	rows := []Row{sample(), sample(), sample()}
	entries, ok := view.table(rows).(*parts.KeyTable)
	if !ok {
		t.Fatal("the entries are no longer shown in a table that says when it has focus")
	}

	elsewhere := widget.NewButton("somewhere else", func() {})
	window := test.NewWindow(container.NewBorder(elsewhere, nil, nil, nil, entries))
	defer window.Close()
	window.Resize(fyne.NewSize(1342, 600))
	canvas := window.Canvas()

	first := widget.TableCellID{Row: 0, Col: 0}
	second := widget.TableCellID{Row: 1, Col: 0}

	if got := styleOf(t, entries, first); marked(got) {
		t.Error("a cell is marked before the keyboard has reached the table at all")
	}

	canvas.Focus(entries)
	if got := styleOf(t, entries, first); !marked(got) {
		t.Errorf("the table has focus and the cell the keyboard stands on is drawn %+v, "+
			"so there is nothing on the screen saying where a person is", got)
	}
	if got := styleOf(t, entries, second); marked(got) {
		t.Error("a cell the keyboard is not on carries the mark, so the mark says nothing")
	}

	entries.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if got := styleOf(t, entries, second); !marked(got) {
		t.Errorf("the keyboard moved down and the cell it moved to is drawn %+v", got)
	}
	if got := styleOf(t, entries, first); marked(got) {
		t.Error("the keyboard moved on and the cell it left is still marked, so two places claim it")
	}

	canvas.FocusNext()
	if focused := canvas.Focused(); focused == fyne.Focusable(entries) {
		t.Fatal("focus did not leave the table, so this test proves nothing about leaving")
	}
	if got := styleOf(t, entries, second); marked(got) {
		t.Error("focus left the table and the mark stayed behind, which is the failure this guard exists for")
	}
}

// The mark is drawn rather than merely recorded.
//
// 🔴 A style that no pixel carries is the failure family this screen has already
// paid for four times: the test reads the model, the model is right, and the
// window shows something else. So this one renders the same phrase twice and
// asks whether the marked one puts more ink on the screen. Weight and an
// underline both add pixels, so fewer or equal means the mark never arrived.
func TestTheMarkOfTheKeyboardReachesThePixels(t *testing.T) {
	line := Line{Text: "should be rejected", Color: palette.NameRejection}

	plainCell := painted(Line{})
	set(plainCell, line, fyne.TextStyle{})
	markedCell := painted(Line{})
	set(markedCell, line, fyne.TextStyle{Bold: true, Underline: true})

	bare := inkOf(t, plainCell)
	withMark := inkOf(t, markedCell)
	if withMark <= bare {
		t.Errorf("the marked phrase paints %d pixels and the plain one %d, so the mark "+
			"is in the widget and not on the screen", withMark, bare)
	}
}

// inkOf renders one object and counts the pixels that differ from the ground.
func inkOf(t *testing.T, object fyne.CanvasObject) int {
	t.Helper()
	window := test.NewWindow(object)
	defer window.Close()
	window.Resize(fyne.NewSize(600, 120))

	image := window.Canvas().Capture()
	bounds := image.Bounds()
	groundR, groundG, groundB, _ := image.At(bounds.Min.X, bounds.Min.Y).RGBA()
	ink := 0
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			r, g, b, _ := image.At(x, y).RGBA()
			if r != groundR || g != groundG || b != groundB {
				ink++
			}
		}
	}
	return ink
}

// The keyboard meets the controls of the screen before it meets the entries.
//
// 🔴 Measured on 2026-08-18 before this was true: the first press of Tab landed
// on the table, and the button that opens a manifest, which stands above it,
// came second. Nothing about the drawing said so, because the convenient way of
// building a border hands the middle over first and the edges after it.
func TestTheKeyboardMeetsTheButtonsBeforeTheEntries(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	content := view.Content()
	view.show(view.rendered(Screen{Rows: []Row{sample(), sample()}}))

	window := test.NewWindow(content)
	defer window.Close()
	window.Resize(fyne.NewSize(1342, 600))

	canvas := window.Canvas()
	canvas.FocusNext()
	button, ok := canvas.Focused().(*parts.Button)
	if !ok {
		t.Fatalf("the first thing the keyboard reaches is %T rather than a button", canvas.Focused())
	}
	if button.Text != i18n.English.T(i18n.ManifestOpen) {
		t.Errorf("the first thing the keyboard reaches is %q rather than the button that opens a manifest",
			button.Text)
	}

	canvas.FocusNext()
	if _, isTable := canvas.Focused().(*parts.KeyTable); !isTable {
		t.Errorf("after the button the keyboard reaches %T rather than the entries", canvas.Focused())
	}
}
