package mutators

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// stringInNumericID is the catalog entry this implements.
const stringInNumericID catalog.ID = "L2.type.string_in_numeric"

// replacementParameter is the knob the entry declares. Its default lives in the
// catalog and arrives already applied, so there is no copy of it here.
const replacementParameter = "value"

// stringInNumeric puts text where the schema declares a number.
//
// It is the first mutator, and it was chosen for that because it exercises the
// whole of a manifest entry while needing nothing but the schema: a value
// before, a value after, damage that is reversible, a hard expectation of
// rejection, and a way of not applying that arises naturally rather than being
// arranged.
//
// # What a real database does with this, measured rather than assumed
//
// Measured on 2026-08-05 against MariaDB, whose default sql_mode carries
// STRICT_TRANS_TABLES: text written into an integer column is refused outright
// with error 1292, which is the rejection the entry declares.
//
// Under a permissive sql_mode, which plenty of real deployments run, the same
// row is accepted and the value becomes nought. No error, one warning. That is
// worth knowing because the specification says acceptance means the value
// survives rather than merely that nothing complained, so a system quietly
// storing nought has failed this expectation as surely as one that crashed. It
// is also the reason the second pillar is worth building at all.
type stringInNumeric struct{}

// ID is the catalog identifier.
func (stringInNumeric) ID() catalog.ID { return stringInNumericID }

// Damage writes the replacement text over the number.
//
// Whether the field holds a number at all is not asked here. The entry declares
// the field types it covers and the engine has already checked them, so asking
// again would put a second copy of a published field in this file.
func (stringInNumeric) Damage(target mutator.Target) mutator.Outcome {
	replacement, ok := target.Params.Text(replacementParameter)
	if !ok {
		return mutator.Skipped(fmt.Sprintf(
			"no %s was given to write in place of the number", replacementParameter))
	}
	if replacement == "" {
		// An empty string is damage of a different kind and there is an entry
		// for it. Doing it here would put one mutator's work under another
		// one's name, and the manifest would then name the wrong expectation.
		return mutator.Skipped(
			"the replacement is empty, which is a different defect with an entry of its own")
	}
	if target.Value == replacement {
		// Nothing would change, and an entry claiming damage that left the
		// value alone would send a test looking for a difference that is not
		// there.
		return mutator.Skipped(fmt.Sprintf(
			"the value is already %q, so writing it again would change nothing", replacement))
	}
	return mutator.Done(replacement)
}
