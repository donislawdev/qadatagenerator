// Package mutators holds the damage this build knows how to do.
//
// The catalog says what each one is. This says what each one does, and the two
// meet at an identifier and nowhere else. Adding damage means adding an entry
// there and an implementation here, and touching nothing in between.
//
// # Why the list of what is implemented lives here
//
// It was in the domain for a day, while there were no implementations and an
// identifier was the only certain thing about one. Now that the implementations
// exist, a list of identifiers over there would be a second copy of the keys of
// the map below, and two lists part company. The closure that holds this
// against the catalog takes the identifiers as an argument, so it did not have
// to change when they moved.
package mutators

import (
	"sort"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// registry is every mutator this build carries, by catalog identifier.
//
// The key is written out rather than taken from the value, so that a mutator
// registered under the wrong identifier is a line somebody can see in a review
// instead of a value quietly filed in the wrong place. A test holds the two to
// each other.
var registry = map[catalog.ID]mutator.Mutator{
	stringInNumericID:          stringInNumeric{},
	checksumOffByOneID:         checksumOffByOne{},
	checksumTransposedDigitsID: checksumTransposedDigits{},
	nationalIDVsGenderID:       nationalIDVsGender{},
	nationalIDVsBirthdateID:    nationalIDVsBirthdate{},
	rangeUnassignedID:          rangeUnassigned{},
	exceedsMaxLengthID:         exceedsMaxLength{},
	stringEmptyID:              stringEmpty{},
	emojiBasicID:               emojiBasic{},
	emojiZWJSequenceID:         emojiZWJSequence{},
	zeroWidthSpaceID:           zeroWidthSpace{},
	nonIdempotentID:            nonIdempotent{},
}

// Brings is what the damage in this build carries with it, so that every run
// has it whatever the recipe, the file or the writer.
//
// # Why this is asked of the damage rather than written into each mode
//
// It was written into each mode: the run from a recipe and the run over
// somebody else's file each added the same capability by hand, with the same
// sentence beside it. A second capability arriving meant a second pair of
// lines, and a mode that missed one would have had every entry needing it
// refused there and applied in the other mode, for a reason nobody chose. The
// corpus it names lives in this package, so this package is the one that knows
// it is here.
//
// # What each one stands on
//
// The Unicode corpus is the handful of characters in unicode_corpus.go. The
// normalisation is the engine's own claim, made before any entry needed it: no
// code in this build normalises yet, and the entry about two forms of the same
// text, the first that would, is not implemented. The entry that does run under
// it supplies a value from a fixed table and normalises nothing.
func Brings() []capability.Name {
	return []capability.Name{capability.UnicodeNormalization, capability.CorpusUnicode}
}

// Lookup returns the mutator for an identifier and whether this build has one.
//
// The second result matters. Damage nobody implemented is a manifest entry
// saying so, not a run that quietly does less than it was asked.
func Lookup(id catalog.ID) (mutator.Mutator, bool) {
	found, ok := registry[id]
	return found, ok
}

// IDs returns every identifier this build implements, sorted.
//
// Sorted because it reaches the closure against the catalog and the messages
// that come out of it, and Go randomises map iteration on purpose.
func IDs() []catalog.ID {
	ids := make([]catalog.ID, 0, len(registry))
	for id := range registry {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
