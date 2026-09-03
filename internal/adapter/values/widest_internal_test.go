package values

import (
	"strconv"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// This test is inside the package while its neighbours are beside it, and the
// reason is the whole point of what it guards. Nothing outside can call the
// function below with the argument that matters: the caller starts at six
// digits and only lowers. A test coming through the public door would be a test
// of that caller, which is the very thing that might change one day.

// 🔴 Asking for more digits than the number can hold used to walk off the end
// of the arithmetic rather than being refused. The limit is built by
// multiplying by ten, ten to the nineteenth does not fit in an int64, and what
// comes out is negative. A draw with a bound at or below zero panics, and this
// package produces values one at a time in the middle of writing somebody's
// dataset, so the panic would arrive with a file half written.
func TestADrawWiderThanTheNumbersItReturnsIsCappedRatherThanPanicking(t *testing.T) {
	source := New("pl-PL", "widest-test")
	request := generate.Request{
		Type: TypeInteger,
		At:   seed.Coordinate{Entity: "t", Row: 1, Field: "n", Purpose: "value"},
	}

	// Nineteen is the first width that overflows. Sixty four is past the point
	// where every power of ten is a multiple of two to the sixty fourth, so the
	// limit is not negative there but zero, and a bound of zero panics too.
	for _, digits := range []int{widestDrawn, widestDrawn + 1, 25, 64} {
		drawn := source.wholeNumber(request, digits)
		if drawn < 0 {
			t.Errorf("a draw of %d digits gave %d, which is not a number this build "+
				"would put in a column", digits, drawn)
		}
		if written := strconv.FormatInt(drawn, 10); len(written) > widestDrawn {
			t.Errorf("a draw of %d digits gave %s, which is wider than the ceiling of %d",
				digits, written, widestDrawn)
		}
	}
}
