package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// nameNullSurnameID is the catalog entry this implements.
const nameNullSurnameID catalog.ID = "REV.name.null_surname"

// nameNullSurname puts the surname Null in place of a surname.
//
// # What the application is expected to do
//
// Take Null as a surname, keep it, and show it back as Null. The defect is a
// layer that reads the text Null as the absence of a value: the record is lost,
// the form says the field is required, or the surname comes back empty. The
// catalog marks the expectation hard, because people are called Null.
//
// # Why written Null and nothing else
//
// The entry names that surname. NULL in capitals is the same trap for the same
// layers, and it is left to the catalog to add as an entry of its own rather
// than drawn here under a name that does not say so.
type nameNullSurname struct{}

// ID is the catalog identifier.
func (nameNullSurname) ID() catalog.ID { return nameNullSurnameID }

// Damage puts Null in place of the value.
func (nameNullSurname) Damage(target mutator.Target) mutator.Outcome {
	return replacedByName(target, nullSurname, ":name-null-surname")
}
