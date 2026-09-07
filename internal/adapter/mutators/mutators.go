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
