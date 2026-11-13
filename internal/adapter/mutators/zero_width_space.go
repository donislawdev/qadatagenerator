package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// zeroWidthSpaceID is the catalog entry this implements.
const zeroWidthSpaceID catalog.ID = "UNI.zero_width.space"

// zeroWidthSpace puts an invisible character inside a text value.
//
// # Why the middle is the default and the edges are only allowed
//
// The default comes from the catalog, and its example puts the character
// inside the domain of an address, between two letters a reader sees side by
// side. The value then reads as one word and compares as another, which is the
// impersonation route the entry describes. The edges are honoured because the
// entry names the knob.
//
// It is not whitespace, by the Unicode definition, so code that trims spaces
// from the ends of a value leaves it where it is, at an edge as much as in the
// middle.
type zeroWidthSpace struct{}

// ID is the catalog identifier.
func (zeroWidthSpace) ID() catalog.ID { return zeroWidthSpaceID }

// Damage puts the character where the entry says.
func (zeroWidthSpace) Damage(target mutator.Target) mutator.Outcome {
	position, refusal := positionOf(target.Params)
	if refusal != "" {
		return mutator.Skipped(refusal)
	}
	if target.Value == "" {
		return mutator.Unfit("the value is empty, and an invisible character on its own is a " +
			"value that only looks empty, which is a different test")
	}
	damaged, refusal := placed(target.Value, zeroWidthSpaceText, position)
	if refusal != "" {
		return mutator.Skipped(refusal)
	}
	return mutator.Done(damaged)
}
