package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// emojiZWJSequenceID is the catalog entry this implements.
const emojiZWJSequenceID catalog.ID = "UNI.emoji.zwj_sequence"

// emojiZWJSequence puts one character built from seven code points at the end
// of a text value.
//
// # Why at the end and nowhere else
//
// The entry declares no position, and its example puts the family after the
// name with a space, the way the single emoji goes. The end is also where a
// length limit cuts, so a limit that cuts the value cuts into the family before
// it reaches the name, which is the failure this entry exists to show.
type emojiZWJSequence struct{}

// ID is the catalog identifier.
func (emojiZWJSequence) ID() catalog.ID { return emojiZWJSequenceID }

// Damage puts the family after the value.
func (emojiZWJSequence) Damage(target mutator.Target) mutator.Outcome {
	return withEmoji(target.Value, family, positionEnd)
}
