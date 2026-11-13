package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// nameApostropheID is the catalog entry this implements.
const nameApostropheID catalog.ID = "REV.name.apostrophe"

// nameApostrophe puts a real surname with an apostrophe in place of a surname,
// and a whole name with one in place of a whole name.
//
// # What the application is expected to do
//
// Take the surname and keep it as it was sent. The defect is a defence against
// injection that refuses the apostrophe, or one that escapes it on the way in
// and stores O\'Brien. The catalog marks the expectation hard, because the name
// is real and nothing about it is unusual to the people who carry it.
//
// # Why only a column of surnames
//
// The entry is about a surname, and the engine lets it land only where a
// column is known to hold surnames, alone or in a whole name, from a
// generator in a recipe or from a header in somebody else's file. Why a real
// name rather than an apostrophe put into the value is written beside the
// names.
type nameApostrophe struct{}

// ID is the catalog identifier.
func (nameApostrophe) ID() catalog.ID { return nameApostropheID }

// Damage puts one of the names with an apostrophe in place of the value.
func (nameApostrophe) Damage(target mutator.Target) mutator.Outcome {
	return replacedByName(target, nameLists{surnames: apostropheSurnames, whole: apostropheFullNames}.
		For(target.Field.NamePart), ":name-apostrophe")
}
