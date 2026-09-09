package quickview

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// TestEveryMaskFitsThePanelItIsDrawnIn is the guard over a fault this window
// has already paid for once, arriving through a new door.
//
// # Why a mask is dangerous where a sentence is not
//
// The sentences in this panel are labels and they WRAP, so a long one makes the
// panel taller. A mask is canvas text and canvas text does not wrap: it asks
// for the width of its whole string, and a column is as wide as its widest
// child. So a mask too long for the panel does not overflow visibly - it makes
// the entire panel wider, and the card of values beside it narrower, on a
// screen nobody was looking at when the scheme was added.
//
// That is the same family as the fault measured on the take screen on
// 2026-09-02: one paragraph without wrapping pushed four settings off the
// window, with every test green, because the tree had the content in it and
// only the SIZES were wrong.
//
// # Why the longest mask is not obvious
//
// The account number is twenty eight characters where every other scheme here
// is nine to seventeen, and it is the one whose length is not a property of the
// scheme at all but of the country this build generates for. A second country
// would change it without anybody editing this window.
func TestEveryMaskFitsThePanelItIsDrawnIn(t *testing.T) {
	// The room a mask actually has: the panel less what the card and the
	// column put either side of it. Read from the theme rather than written
	// down, so a change to either moves this floor with it.
	room := palette.AsideWidth - 2*palette.StepOf(palette.NameStepGroup)

	for _, scheme := range values.New("en-US", "").Schemes() {
		shaped, saysSo := scheme.(provider.Shaped)
		if !saysSo {
			continue
		}
		mask := shaped.Mask()
		drawn := fyne.MeasureText(mask, palette.TypeSize(palette.TypeValue),
			fyne.TextStyle{Monospace: true}).Width
		if drawn > room {
			t.Errorf("the mask of %s is %.1f wide where the panel leaves %.1f, so drawing it "+
				"widens the whole panel and narrows the values beside it: %q",
				scheme.ID(), drawn, room, mask)
		}
	}
}
