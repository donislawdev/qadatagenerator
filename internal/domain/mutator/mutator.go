// Package mutator is the contract every piece of damage implements.
//
// The catalog says what a mutator is and what a test should conclude from it.
// This says what a mutator does. Nothing joins the two except the identifier,
// which is what lets the catalog grow without the engine being touched.
//
// # Why this was not written until there was a mutator to write it from
//
// An interface designed before its first user is a guess about what that user
// needs. This project has already paid for one: an interface for reserved
// ranges was written a day ahead of the mutator that wanted it, and the mutator
// turned out to need a side the interface did not have. So this was left
// deliberately unwritten while the catalog landed, and it is shaped by the
// first entry implemented rather than by what seemed likely.
//
// The first entry taught one thing that changed the shape. Whether a mutator
// suits a field is already written down in the catalog, in applies_to, so
// asking each mutator to decide it again would put a second copy of a published
// field inside the code, once per mutator. The engine asks the entry, and by
// the time a mutator is called the question is settled. What is left for the
// mutator is the part only it can know.
//
// # Silence cannot be returned from here
//
// Damage either happened or it did not, and either way something comes back
// saying so. There is no third answer and no error: a mutator that could return
// nothing would be a mutator whose result the engine could drop, and a dropped
// result is a defect that never reaches the manifest. That is the one failure
// this whole product exists to make impossible.
package mutator

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Target is one value a mutator has been asked to damage, with everything it
// is allowed to know about it.
type Target struct {
	// Value is what is there now.
	Value string
	// Field is what the schema says about the column it sits in. It arrives
	// the same way whether a recipe declared it or it was worked out from
	// somebody else's file, which is why the engine takes a schema view and
	// never a recipe.
	Field recipe.FieldInfo
	// At is the address of this value. Randomness is derived from it, so a
	// mutator that chooses between variants chooses the same one every time
	// for one seed.
	At seed.Coordinate
	// Provider is the numbering scheme behind the column, nil when the column
	// follows none or when this build has no implementation of the one it
	// names.
	//
	// # Why the whole provider and not the one method a mutator wants
	//
	// The schemes differ in what they can be asked. One encodes a date, another
	// a gender, another has blocks nobody is issued from, and the contract
	// carries a separate interface for each of those. A mutator asks for the
	// one it needs with a type assertion, which is the same question the
	// provider contract already answers in both directions when a provider is
	// checked.
	//
	// That assertion also settles the nil case without a second check: a column
	// following no scheme fails it exactly as a scheme that cannot do what was
	// asked, and both are the same refusal to whoever reads the manifest.
	//
	// # Why the engine resolves it rather than the mutator
	//
	// A mutator may not know where implementations live. The same rule put the
	// registry of damage behind a port, and this is the second one through the
	// same door.
	Provider provider.Provider
	// Params are the knobs, with the catalog's defaults already applied.
	//
	// The catalog is the only thing that fills them. This said "and the
	// policy's overrides on top" until 2026-09-06, and nothing in this build
	// does that: a policy carries the layers to attack, the share of rows, the
	// columns to leave alone and the condition a row has to meet, and it has
	// nowhere to put a knob. A sentence promising a setting nobody can write
	// is worse than no sentence, because it reads as a feature to whoever goes
	// looking for one.
	Params Parameters
}

// Parameters are a mutator's knobs.
//
// The values are untyped because the catalog carries several kinds of scalar.
// Readers are added here as a mutator needs one, rather than in advance, so
// that the ones that exist are the ones something uses.
type Parameters map[string]any

// Text reads a parameter as text and says whether it was there and was text.
//
// The second result is not decoration. A mutator asked to write a replacement
// it was never given has nothing to write, and inventing one would be this tool
// deciding what damage to do.
func (p Parameters) Text(name string) (string, bool) {
	written, ok := p[name].(string)
	return written, ok
}

// Whole reads a parameter as a whole number and says whether it was there and
// was one.
//
// # Why three types and not one
//
// Measured on 2026-08-11: the catalog reader hands whole numbers over as
// uint64, which is what the YAML library produces for an unsigned literal. A
// negative one would arrive as int64, and a parameter built in a test is
// written as a plain int. All three mean the same thing to a mutator, and a
// reader that knew only the one the catalog happens to use today would refuse a
// value somebody wrote by hand and look like a defect in their policy.
//
// A fraction is deliberately not one of them. Truncating 2.5 to 2 would be this
// tool deciding how much damage to do, and the catalog carries fractions for
// other parameters, so the two kinds have to stay apart.
func (p Parameters) Whole(name string) (int, bool) {
	switch value := p[name].(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case uint64:
		return int(value), true
	default:
		return 0, false
	}
}

// Outcome is what a mutator did. It is built by one of two constructors, so
// the combination that says both things at once cannot be written down.
type Outcome struct {
	value   string
	applied bool
	reason  string
}

// Done reports damage that happened, carrying the value that took the old
// one's place.
func Done(value string) Outcome { return Outcome{value: value, applied: true} }

// Skipped reports damage that did not happen, carrying why.
//
// The reason reaches the manifest and from there a person reading a report, so
// it is a sentence about the data rather than about the code.
func Skipped(reason string) Outcome { return Outcome{reason: reason} }

// Applied reports whether the damage happened.
func (o Outcome) Applied() bool { return o.applied }

// Value is what took the old value's place, meaningful only when the damage
// happened.
func (o Outcome) Value() string { return o.value }

// Reason is why the damage did not happen, meaningful only when it did not.
func (o Outcome) Reason() string { return o.reason }

// Mutator is one piece of damage.
type Mutator interface {
	// ID is the catalog identifier this implements. The entry beside it
	// carries everything else, including what a test should expect.
	ID() catalog.ID

	// Damage returns what happened. It is called only after the engine has
	// found the entry applicable, so it may assume the field suits it and
	// refuse only for reasons the catalog cannot express.
	Damage(target Target) Outcome
}
