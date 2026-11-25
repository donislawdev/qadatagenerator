package quickview

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
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
//
// # What is measured since 2026-11-25
//
// The lines a mask is drawn on (maskLines), because a mask wider than the panel
// breaks after its separators, which the UUID was the first to need. A group
// with no separator in it that is still too wide comes back whole and is
// refused here, and the canary below holds that this test sees one.
func TestEveryMaskFitsThePanelItIsDrawnIn(t *testing.T) {
	// The room a mask actually has: the panel less what the card puts either
	// side of it. Read from the theme rather than written down, so a change to
	// either moves this floor with it.
	room := palette.AsideWidth - 2*palette.StepOf(palette.NameStepGroup)
	if maskRoom() != room {
		t.Fatalf("the panel gives a mask %.1f and this test expects %.1f", maskRoom(), room)
	}
	wide := strings.Repeat(string(provider.MaskAlnum), 60)
	if lines := tooWide(maskLines(wide, maskFits), room); len(lines) == 0 {
		t.Fatalf("canary: a mask of 60 places with no separator was drawn as lines that fit: %v",
			maskLines(wide, maskFits))
	}

	for _, scheme := range values.New("en-US", "").Schemes() {
		shaped, saysSo := scheme.(provider.Shaped)
		if !saysSo {
			continue
		}
		// Every form, because each is drawn on its own lines and the widest of
		// them is the one that decides the width of the panel.
		for _, mask := range shaped.Masks() {
			lines := maskLines(mask, maskFits)
			if strings.Join(lines, "") != mask {
				t.Errorf("the mask of %s was broken into %q, which do not join back into %q", scheme.ID(), lines, mask)
			}
			for _, line := range tooWide(lines, room) {
				t.Errorf("a line of the mask of %s is %.1f wide where the panel leaves %.1f, so drawing it "+
					"widens the whole panel and narrows the values beside it: %q",
					scheme.ID(), width(line), room, line)
			}
		}
	}
}

// The UUID, the one mask wider than the panel today, goes on two lines, each
// ending where a group ends.
func TestTheMaskOfAUUIDGoesOnTwoLinesBrokenAfterAHyphen(t *testing.T) {
	mask := (intl.UUID{}).Masks()[0]
	lines := maskLines(mask, maskFits)
	if len(lines) != 2 || !strings.HasSuffix(lines[0], "-") {
		t.Errorf("the mask of a UUID was drawn as %q", lines)
	}
	if got := maskLines("000-00-0000", maskFits); len(got) != 1 {
		t.Errorf("a mask that fits was broken into %q", got)
	}
}

// width is how wide a line of a mask is drawn.
func width(line string) float32 {
	return fyne.MeasureText(line, palette.TypeSize(palette.TypeValue), fyne.TextStyle{Monospace: true}).Width
}

// tooWide are the lines wider than the room.
func tooWide(lines []string, room float32) []string {
	var wide []string
	for _, line := range lines {
		if width(line) > room {
			wide = append(wide, line)
		}
	}
	return wide
}
