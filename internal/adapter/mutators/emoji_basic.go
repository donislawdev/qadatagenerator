package mutators

import (
	"strings"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// emojiBasicID is the catalog entry this implements.
const emojiBasicID catalog.ID = "UNI.emoji.basic"

// emojiBasic puts one emoji into a text value.
//
// # Why the emoji is a word of its own at either edge
//
// The entry shows Maria becoming Maria with a smiling face after a space. At an
// edge the emoji is therefore kept apart by a space, and in the middle it goes
// in as it is, because a space there would split one word into two, which is a
// second change the entry does not describe.
//
// # Why an empty value is refused
//
// The emoji would then be the whole value rather than part of one, and a field
// holding nothing but an emoji is a test of whether the field may be empty of
// text, which is not what this entry says.
//
// # Why a value without a letter is refused
//
// The entry expects the emoji to be accepted, and its note says why that holds
// for text: a name or a comment. A value with no letter in it is a code, such
// as a number, a date or a social security number held as text, and an emoji
// in a code is a value its application should refuse. Measured on a file where
// nothing said which column held what: the joined family landed on a column of
// social security numbers and the manifest expected it accepted. What the
// value shows is enough to tell the two apart, so it is refused here, and a
// column of names beside it still gets the damage.
type emojiBasic struct{}

// ID is the catalog identifier.
func (emojiBasic) ID() catalog.ID { return emojiBasicID }

// Damage puts the emoji where the entry says.
func (emojiBasic) Damage(target mutator.Target) mutator.Outcome {
	position, refusal := positionOf(target.Params)
	if refusal != "" {
		return mutator.Skipped(refusal)
	}
	return withEmoji(target.Value, smilingFace, position)
}

// withEmoji puts an emoji into a value at a position, kept apart by a space at
// either edge.
//
// It is shared by the two entries that supply an emoji, so that a single
// character and a joined sequence land the same way and differ only in what
// they are.
func withEmoji(value, emoji, position string) mutator.Outcome {
	if value == "" {
		return mutator.Unfit("the value is empty, so the emoji would be the whole value rather " +
			"than part of one")
	}
	if strings.IndexFunc(value, unicode.IsLetter) < 0 {
		return mutator.Unfit("the value holds no letter, so it reads as a code such as a number " +
			"or a date rather than as text, and the acceptance this damage expects is about text")
	}
	insertion := emoji
	switch position {
	case positionStart:
		insertion = emoji + " "
	case positionEnd:
		insertion = " " + emoji
	}
	damaged, refusal := placed(value, insertion, position)
	if refusal != "" {
		return mutator.Skipped(refusal)
	}
	return mutator.Done(damaged)
}
