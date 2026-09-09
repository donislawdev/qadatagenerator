package parts_test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// 🔴 The guard asks what was DRAWN, not whether the event arrived.
//
// That distinction is the whole lesson of DEC-237. The chooser went on
// receiving MouseIn through an entire regression: it set a flag of the library,
// refreshed itself and drew exactly the same picture, so a guard asking "does
// the pointer reach this control" would have been green while the window had no
// hover state at all.
func TestARowUnderThePointerDrawsSomethingElse(t *testing.T) {
	drawn := 0
	row := parts.NewTapRow(func(hovered bool) fyne.CanvasObject {
		drawn++
		if hovered {
			return canvas.NewText("under the pointer", nil)
		}
		return canvas.NewText("at rest", nil)
	}, nil)

	window := test.NewWindow(row)
	defer window.Close()

	if words := textIn(row); words != "at rest" {
		t.Fatalf("a row nobody is pointing at draws %q", words)
	}

	row.MouseIn(&desktop.MouseEvent{})
	if words := textIn(row); words != "under the pointer" {
		t.Errorf("a row under the pointer draws %q, so the state arrived and the picture did not", words)
	}

	row.MouseOut()
	if words := textIn(row); words != "at rest" {
		t.Errorf("a row the pointer left draws %q, so it is stuck in the hovered picture", words)
	}
}

// A row does what it is for when it is clicked, with the value it was built
// around rather than whatever the screen last held.
func TestClickingARowDoesWhatThatRowIsFor(t *testing.T) {
	tapped := ""
	row := parts.NewTapRow(
		func(bool) fyne.CanvasObject { return canvas.NewText("666-94-2223", nil) },
		func() { tapped = "666-94-2223" })

	row.Tapped(nil)

	if tapped != "666-94-2223" {
		t.Errorf("clicking the row did %q", tapped)
	}
}

// A row with nothing wired to it is silent rather than a crash, because the
// card is rebuilt whenever the values change and a row can outlive one draw.
func TestARowWithNothingWiredToItIsSilent(t *testing.T) {
	row := parts.NewTapRow(nil, nil)
	window := test.NewWindow(row)
	defer window.Close()
	row.Tapped(nil)
	row.MouseIn(&desktop.MouseEvent{})
	row.MouseOut()
}

// textIn reads whatever text the row is drawing right now, which is the only
// thing a person can see of it.
func textIn(row *parts.TapRow) string {
	found := ""
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		switch shape := object.(type) {
		case *canvas.Text:
			found = shape.Text
		case *fyne.Container:
			for _, child := range shape.Objects {
				walk(child)
			}
		}
	}
	for _, object := range test.WidgetRenderer(row).Objects() {
		walk(object)
	}
	return found
}
