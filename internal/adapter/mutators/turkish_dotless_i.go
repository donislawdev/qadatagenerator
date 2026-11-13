package mutators

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// turkishDotlessIID is the catalog entry this implements.
const turkishDotlessIID catalog.ID = "UNI.case.turkish_dotless_i"

// turkishDotlessI swaps every i for the dotless ı and every I for the dotted İ,
// the other Turkish letter of the same case. Why these two letters find the
// defect on any machine is written beside them in the corpus.
//
// # What the application is expected to do
//
// Take the value and keep it as it was sent. The defect is a case conversion
// somewhere in the application that quietly gives back a different value, such
// as a plain I where the value had ı, or a value one or two characters longer.
// Whether the field accepts Turkish letters at all can be a rule of the product,
// which is why the catalog marks the expectation soft.
//
// # Why a letter with an accent is left alone
//
// An i or I with an accent is a different letter, and it is left alone the same
// whether the accent is written into one character or follows the letter as a
// combining mark. A decomposed İ is an I followed by a combining dot, and
// swapping that I would give a capital with two dots. Treating both writings
// alike also means the two canonical forms of one value come out as the two
// forms of one result.
//
// # Why every i and not one
//
// As with the raised capitals of the entry about normalising twice, the whole
// word is what reads as somebody else's name once it has been through a case
// conversion, so every letter that can swap does.
type turkishDotlessI struct{}

// ID is the catalog identifier.
func (turkishDotlessI) ID() catalog.ID { return turkishDotlessIID }

// Damage swaps each i and I that carries no accent for its Turkish counterpart.
func (turkishDotlessI) Damage(target mutator.Target) mutator.Outcome {
	value := target.Value
	var built strings.Builder
	// Each swapped letter grows from one byte to two in UTF-8.
	built.Grow(2 * len(value))
	changed := false
	for at := 0; at < len(value); {
		r, size := utf8.DecodeRuneInString(value[at:])
		// Bytes that are not text are copied as they were rather than read as
		// a character, so the value changes only where the manifest says.
		counterpart, swaps := turkishCounterparts[r]
		if swaps && !markFollows(value[at+size:]) {
			built.WriteRune(counterpart)
			changed = true
		} else {
			built.WriteString(value[at : at+size])
		}
		at += size
	}
	if !changed {
		return mutator.Unfit("the value has no letter i or I without an accent, so there is " +
			"nothing to swap for the Turkish dotless i or dotted capital I")
	}
	return mutator.Done(built.String())
}

// markFollows reports whether the text begins with a combining mark, which
// belongs to the letter before it.
func markFollows(rest string) bool {
	next, size := utf8.DecodeRuneInString(rest)
	return size > 0 && unicode.IsMark(next)
}
