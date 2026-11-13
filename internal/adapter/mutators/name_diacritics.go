package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// nameDiacriticsID is the catalog entry this implements.
const nameDiacriticsID catalog.ID = "REV.name.diacritics"

// nameDiacritics puts a real name with a letter that carries a mark in place of
// a name of the same part: a given name for a given name, a surname for a
// surname, a whole name for a whole name.
//
// # What the application is expected to do
//
// Take the name and keep it as it was sent. The defect is a validator that
// allows only the letters of the English alphabet, or a database or a file
// written in a character set that has no Ł, which gives back Lukasz or a
// question mark. The catalog marks the expectation hard.
//
// # Why the same part of a name
//
// A surname in a column of given names is a value the column was not meant to
// hold, and a test reading the report would have to ask itself whether the
// application was right to refuse it for that. Keeping the part keeps the one
// question this entry asks.
type nameDiacritics struct{}

// ID is the catalog identifier.
func (nameDiacritics) ID() catalog.ID { return nameDiacriticsID }

// Damage puts one of the names with a marked letter in place of the value.
func (nameDiacritics) Damage(target mutator.Target) mutator.Outcome {
	return replacedByName(target,
		nameLists{diacriticGivenNames, diacriticSurnames, diacriticFullNames}.For(target.Field.NamePart),
		":name-diacritics")
}
