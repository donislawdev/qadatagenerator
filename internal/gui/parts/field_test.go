package parts

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	toolkit "fyne.io/fyne/v2/theme"

	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// 🔴 The words of a field start where the words of a chooser start.
//
// That was the owner's first report about this control and it is the reason the
// space around the field exists. The renderer adds its own inner padding to
// whatever room it is given, so the guard measures the SUM rather than either
// half: what a person sees is where the text begins, not which of the two
// numbers put it there.
func TestAFieldStartsItsWordsWhereAChooserDoes(t *testing.T) {
	window := fyne.CurrentApp().Settings().Theme()
	around := palette.StepOf(palette.NameStepTight)
	fromRenderer := (bareField{}).Size(toolkit.SizeNameInnerPadding)
	if got, want := around+fromRenderer, palette.StepOf(palette.NameStepRelated); got != want {
		t.Errorf("the words of a field start %v in, where a chooser starts its own at %v", got, want)
	}
	// And the inner padding is still the window's, so this is space put around
	// the field rather than a second palette living inside it.
	if fromRenderer != window.Size(toolkit.SizeNameInnerPadding) {
		t.Error("a field answers the inner padding itself, which would be a value the window does not know about")
	}
}

func sameInk(first, second color.Color) bool {
	fr, fg, fb, fa := first.RGBA()
	sr, sg, sb, sa := second.RGBA()
	return fr == sr && fg == sg && fb == sb && fa == sa
}
