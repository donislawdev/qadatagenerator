package paint_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	theme "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Words cut to a width fit it, end on an ellipsis, keep every character they
// keep whole, and are left alone when they fit already (DEC-338).
func TestWordsCutToAWidthFitItAndStayWhole(t *testing.T) {
	test.NewTempApp(t)
	size := theme.TypeSize(theme.TypeBody)
	width := func(words string) float32 { return fyne.MeasureText(words, size, fyne.TextStyle{}).Width }
	words := "Zażółć gęślą jaźń, the kind of value such as a whole number"
	full := width(words)
	if got := paint.CutToWidth(words, theme.TypeBody, full); got != words {
		t.Errorf("words that fit were cut to %q", got)
	}
	for _, share := range []float32{0.25, 0.5, 0.9} {
		room := full * share
		got := paint.CutToWidth(words, theme.TypeBody, room)
		if !strings.HasSuffix(got, "…") || !utf8.ValidString(got) {
			t.Errorf("cut to %v of %v: %q does not end on an ellipsis or is not whole characters", room, full, got)
		}
		if width(got) > room {
			t.Errorf("cut to %v: %q is %v wide", room, got, width(got))
		}
		kept := strings.TrimSuffix(got, "…")
		if !strings.HasPrefix(words, kept) {
			t.Errorf("cut to %v: %q is not the start of the words", room, got)
		}
		if next := []rune(words)[utf8.RuneCountInString(kept)]; width(kept+string(next)+"…") <= room {
			t.Errorf("cut to %v: %q stops a character short of what fits", room, got)
		}
	}
	if got := paint.CutToWidth(words, theme.TypeBody, 0); got != "" {
		t.Errorf("no room at all left %q", got)
	}
}
