package mutators

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// nationalIDVsGenderID is the catalog entry this implements.
const nationalIDVsGenderID catalog.ID = "L4.coherence.national_id_vs_gender"

// nationalIDVsGender leaves a correct identifier that encodes the other gender
// than the column beside it.
//
// # Why this one is worth more than the layer below it
//
// Nothing here is malformed. The identifier passes its own validator, the
// gender column holds a legal value, and only the pair is wrong. Applications
// almost always catch the layer three version, where the value itself is
// broken, and almost never this one, because catching it means checking two
// fields against each other rather than one field against a rule. That gap is
// the whole reason the second pillar exists, so the expectation here is soft:
// whether an application should reject the row is its own decision, and the
// manifest says so instead of pretending.
//
// # Why the provider does the writing
//
// Flipping the digit that carries the gender invalidates the check digit, so
// the value has to be rebuilt rather than edited. Only the scheme knows how,
// and doing it here would put the arithmetic of one country inside a mutator
// that is named after a property rather than a country. The same rule already
// keeps reserved ranges and postal pairs on the provider's side.
type nationalIDVsGender struct{}

// ID is the catalog identifier.
func (nationalIDVsGender) ID() catalog.ID { return nationalIDVsGenderID }

// Damage rebuilds the identifier around the other gender.
//
// The value it produces is correct on its own, check digit included. That is
// the point of it, and it is why this cannot be done by writing over a digit.
func (nationalIDVsGender) Damage(target mutator.Target) mutator.Outcome {
	// One assertion answers two questions. A column following no scheme at all
	// and a scheme that encodes no gender are the same thing to whoever reads
	// the manifest: nothing here can say what gender this value carries.
	encoder, encodes := target.Provider.(provider.GenderEncoder)
	if !encodes {
		return mutator.Skipped(
			"this column follows no numbering scheme that encodes a gender, so there is none to disagree with")
	}
	carried, readable := encoder.EncodedGender(target.Value)
	if !readable {
		return mutator.Unfit("no gender can be read out of this value, so there is none to change")
	}
	other, opposite := theOther(carried)
	if !opposite {
		return mutator.Unfit(fmt.Sprintf(
			"this value carries no gender to turn round, it reads as %s", carried))
	}
	rebuilt, err := encoder.WithGender(target.Value, other)
	if err != nil {
		return mutator.Skipped(fmt.Sprintf("the value cannot be rebuilt around %s: %v", other, err))
	}
	if rebuilt == target.Value {
		// The scheme reported one gender and rebuilt the value unchanged, so
		// the two halves of it disagree. An entry claiming damage here would
		// send a test looking for a difference that is not there.
		return mutator.Skipped(fmt.Sprintf(
			"the value reads as %s and rebuilding it as %s changed nothing", carried, other))
	}
	return mutator.Done(rebuilt)
}

// theOther returns the gender to write instead, and whether there is one.
//
// The set is closed and has two members plus the answer that means the scheme
// carries none. Turning the third one into either of the first two would be
// this tool deciding somebody's gender from a value that does not state it.
func theOther(carried provider.Gender) (provider.Gender, bool) {
	switch carried {
	case provider.Female:
		return provider.Male, true
	case provider.Male:
		return provider.Female, true
	default:
		return provider.Unknown, false
	}
}
