package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// nameCJKID is the catalog entry this implements.
const nameCJKID catalog.ID = "REV.name.cjk"

// nameCJK puts a real name written in Chinese, Japanese or Korean characters in
// place of a name of the same part.
//
// # What the application is expected to do
//
// Take the name and keep it as it was sent. Two defects meet here. A minimum
// length of three refuses a name that is complete in two characters, and a
// column in a character set of one byte a letter, or one whose limit counts
// bytes, refuses or cuts it, because each of these characters takes three bytes
// in UTF-8. The catalog marks the expectation hard.
//
// # Why the same part of a name
//
// For the reason written beside the entry about diacritics: a given name stays
// a given name, so the report asks one question.
type nameCJK struct{}

// ID is the catalog identifier.
func (nameCJK) ID() catalog.ID { return nameCJKID }

// Damage puts one of the names in these writings in place of the value.
func (nameCJK) Damage(target mutator.Target) mutator.Outcome {
	return replacedByName(target,
		nameLists{cjkGivenNames, cjkSurnames, cjkFullNames}.For(target.Field.NamePart), ":name-cjk")
}
