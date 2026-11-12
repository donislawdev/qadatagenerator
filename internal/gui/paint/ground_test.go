package paint_test

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryLookOfAFieldIsDrawnApart holds that each state a text field can be in
// reads differently on the ground and the edge alone, without the field, its
// words or its caret.
//
// # Why the ground and not the field
//
// The keyboard arriving in a field also brings the caret, so a picture of the
// whole field changes on focus even when the mark of the keyboard is gone. The
// first guard of the refused field measured the whole field, and it passed with
// the second edge of a refused field taken out (B-363, tried on 2026-11-12).
// The caret blinks on the real window, so a field whose only sign of the
// keyboard is the caret is seen half of the time, which this window does not
// accept anywhere else (tools/ognisko.py).
//
// Out of reach is one drawing whoever is on the field and whatever it holds,
// and it has to differ from every other.
func TestEveryLookOfAFieldIsDrawnApart(t *testing.T) {
	application := test.NewTempApp(t)
	application.Settings().SetTheme(fynetheme.Visual{})

	seen := map[string]paint.FieldLook{}
	both := []bool{false, true}
	for _, focused := range both {
		for _, hovered := range both {
			for _, invalid := range both {
				look := paint.FieldLook{Focused: focused, Hovered: hovered, Invalid: invalid}
				drawn := groundPixels(look)
				if earlier, same := seen[drawn]; same {
					t.Errorf("%+v is drawn exactly as %+v, so a person cannot tell the two apart",
						look, earlier)
				}
				seen[drawn] = look
			}
		}
	}

	out := groundPixels(paint.FieldLook{Disabled: true})
	if earlier, same := seen[out]; same {
		t.Errorf("a field out of reach is drawn exactly as %+v", earlier)
	}
	for _, look := range []paint.FieldLook{
		{Disabled: true, Focused: true, Hovered: true},
		{Disabled: true, Invalid: true},
	} {
		if groundPixels(look) != out {
			t.Errorf("%+v is drawn apart from a field out of reach, so something says the value "+
				"could or should be changed where nobody can change it", look)
		}
	}
}

// groundPixels is the ground of a field in one look, drawn on its own at the
// height of a control, as text so that drawings can be compared and kept.
func groundPixels(look paint.FieldLook) string {
	ground := paint.FieldGround(look)
	window := test.NewWindow(ground)
	defer window.Close()
	window.SetPadded(false)
	window.Resize(fyne.NewSize(theme.ControlHeight*4, theme.ControlHeight))
	captured := window.Canvas().Capture()
	bounds := captured.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, captured.At(x, y))
		}
	}
	return bounds.String() + string(rgba.Pix)
}
