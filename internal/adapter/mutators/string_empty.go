package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// stringEmptyID is the catalog entry this implements.
const stringEmptyID catalog.ID = "L2.string.empty"

// stringEmpty replaces a text value with an empty string.
//
// # Why this is the smallest mutator and still worth its own entry
//
// Empty and absent are the two things most often confused, in schemas, in
// forms and in databases, and telling them apart is the whole content of this
// damage. The expectation is soft on purpose: whether an empty string is
// allowed where a value is expected is a product decision, and a manifest
// claiming otherwise would be inventing somebody else's rule.
//
// It is also the one piece of damage whose result has to survive a journey it
// could easily be lost on. An empty value written to a file, read back and
// reported has to arrive as a value that is empty rather than as a field that
// was never there, and the manifest carries it as one.
type stringEmpty struct{}

// ID is the catalog identifier.
func (stringEmpty) ID() catalog.ID { return stringEmptyID }

// Damage empties the value.
func (stringEmpty) Damage(target mutator.Target) mutator.Outcome {
	if target.Value == "" {
		// Already empty, so nothing would change and an entry saying otherwise
		// would send a test looking for a difference that is not there. It is
		// an ordinary case rather than a fault: a column with a share of empty
		// values has some, by design.
		return mutator.Unfit("the value is already empty, so emptying it would change nothing")
	}
	return mutator.Done("")
}
