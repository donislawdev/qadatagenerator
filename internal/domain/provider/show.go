package provider

import (
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Show renders a value for a message, quoted and bounded.
//
// # Why a scheme may not put a raw value in a sentence
//
// The values a scheme is handed come from a recipe or from somebody else's
// file, so they are untrusted input of unbounded length. A message built with
// the quoting verb of the formatting package carries whatever it was given,
// which turns a hostile value into a message the same size, on its way into
// somebody's build log.
//
// This is the second of three roads that class of failure takes. The first was
// the recipe reader, bounded on 2026-08-02, and the note left there asked how
// many roads there are. The third arrived with the reader of somebody else's
// file. All three are bounded by the same rule now, which lives in the fault
// package because that is the only layer every road may import.
//
// # Why this is a function rather than a habit
//
// Thirteen places across four schemes wrote the value straight into a sentence,
// and a fourteenth scheme would have written a fourteenth. A rule that has to be
// remembered at each call site is the guard somebody eventually does not write,
// so the guard is not this function: it is the check that no scheme may use the
// quoting verb at all. This function is what makes that check something a person
// can satisfy rather than merely obey.
//
// The order matters. The value is shortened first and quoted second, because
// quoting first can produce an escape sequence that shortening then cuts in
// half.
func Show(value string) string {
	return strconv.Quote(fault.Shorten(value))
}
