package mutators

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// nonIdempotentID is the catalog entry this implements.
const nonIdempotentID catalog.ID = "UNI.normalization.non_idempotent"

// nonIdempotent rewrites the Latin letters of a value as raised capitals, so
// that normalising it twice gives a different answer than normalising it once.
//
// # What it supplies and what it does not claim
//
// The value is stable under compatibility normalisation alone and under lower
// casing alone. It changes between the first and second pass of a pipeline that
// lower cases and then normalises, which is the order the documented takeover
// ran in. A pipeline that normalises first and lower cases second is stable on
// it, and that is why the entry's confidence is soft: whether this value finds
// anything depends on the order inside the application, which the manifest
// cannot see.
//
// # Why every letter that has a form, and not one
//
// The entry's example turns every letter of bigbird. One letter would do for
// the arithmetic, and the whole word is what an attacker registers, because it
// is what reads as the other person's name once it has been through the
// pipeline.
//
// Only the Latin letters A to Z are rewritten. A letter with a diacritic has no
// raised capital, and a letter of another script whose upper case happens to be
// Latin, such as the Turkish dotless i, is a different entry's business.
type nonIdempotent struct{}

// ID is the catalog identifier.
func (nonIdempotent) ID() catalog.ID { return nonIdempotentID }

// Damage rewrites the letters that have a raised capital.
func (nonIdempotent) Damage(target mutator.Target) mutator.Outcome {
	var built strings.Builder
	built.Grow(len(target.Value) * utf8.UTFMax)
	changed := false
	for at := 0; at < len(target.Value); {
		r, size := utf8.DecodeRuneInString(target.Value[at:])
		// Bytes that are not text are copied as they were rather than read as
		// a character. Decoding them hands back the replacement character, and
		// writing that would change the value in a place the manifest does not
		// describe.
		if r <= unicode.MaxASCII {
			if raised, has := capitalModifiers[unicode.ToUpper(r)]; has {
				built.WriteRune(raised)
				changed = true
				at += size
				continue
			}
		}
		built.WriteString(target.Value[at : at+size])
		at += size
	}
	if !changed {
		return mutator.Unfit("the value holds no Latin letter that has a raised capital form, " +
			"such as the B of bigbird, so nothing in it changes between two normalisations")
	}
	return mutator.Done(built.String())
}
