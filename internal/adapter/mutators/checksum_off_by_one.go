package mutators

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// checksumOffByOneID is the catalog entry this implements.
const checksumOffByOneID catalog.ID = "L3.checksum.off_by_one"

// positionParameter is the knob the entry declares, and lastPosition is the one
// value of it this build can carry out. The default lives in the catalog and
// arrives already applied, so there is no copy of it here.
const (
	positionParameter = "position"
	lastPosition      = "last"
)

// checksumOffByOne changes the final digit, so the value keeps its shape and
// stops satisfying the arithmetic of its scheme.
//
// It is the second mutator and it was chosen for what it opens rather than for
// being easy. It is the first damage that is meaningful for some numbering
// schemes and meaningless for others, which is the question the engine could
// not ask until now: every national identifier is a field of type identifier,
// and the schemes disagree about whether they carry a check digit at all.
//
// # Why only the last digit, when the entry says position
//
// Because it is the only position where the expectation the entry declares is
// true of every scheme, and that expectation is hard.
//
// Where the check digit sits at the end, which is so for the national
// identifier, the tax number, the statistical number and the routing number,
// the correct digit is a function of the digits before it. Those did not move,
// so exactly one digit is right and any other one is wrong. Where the check
// digits sit elsewhere, as in the international account number, the final digit
// is part of what they are computed over, and the modulus of that scheme is
// coprime to ten: changing the digit by one changes the remainder, because
// ninety seven divides neither one nor nine nor any power of ten.
//
// Any other position is a different question with a different answer per
// scheme, and one of them is already measured: the statistical number folds two
// remainders onto one digit, so some single digit changes in its payload leave
// a value its own validator still accepts. A mutator aiming there would write a
// hard expectation of rejection over data that is not rejected, which is the
// silent disagreement this product exists to find. So this refuses any other
// position by name rather than doing something that looks similar.
//
// That reasoning is an argument, and an argument is not a measurement. What
// makes the claim safe to publish is the test beside it, which takes a correct
// value from every scheme this build carries, damages it here, and asks that
// scheme's own validator whether the result is still acceptable.
type checksumOffByOne struct{}

// ID is the catalog identifier.
func (checksumOffByOne) ID() catalog.ID { return checksumOffByOneID }

// Damage adds one to the final digit, wrapping nine round to nought.
//
// Whether the field is an identifier, and whether its scheme is one this damage
// covers, are not asked here. The entry declares both and the engine has
// already checked them, so asking again would put a second copy of a published
// field in this file.
func (checksumOffByOne) Damage(target mutator.Target) mutator.Outcome {
	where, given := target.Params.Text(positionParameter)
	if !given {
		return mutator.Skipped(fmt.Sprintf(
			"no %s was given, so nothing says which digit to change", positionParameter))
	}
	if where != lastPosition {
		return mutator.Skipped(fmt.Sprintf(
			"%q is not a position this build can change, because only %q is known to break the "+
				"check of every scheme it applies to", where, lastPosition))
	}
	if target.Value == "" {
		return mutator.Skipped("the value is empty, so there is no digit to change")
	}
	at := len(target.Value) - 1
	// 🔴 Compared as a byte against the ten ASCII digits, and deliberately not
	// with unicode.IsDigit. That function is true of the digits of a dozen other
	// scripts, and adding one to an Arabic-Indic nine here would produce a
	// character from arithmetic that means nothing. A value ending in any
	// character outside ASCII ends in a byte above this range, so it is refused
	// rather than mangled.
	digit := target.Value[at]
	if digit < '0' || digit > '9' {
		return mutator.Skipped(
			"the value does not end in a digit, so there is nothing at the last position to add one to")
	}
	return mutator.Done(target.Value[:at] + string(rune(next(digit))))
}

// next is the digit after this one, with nine wrapping round to nought.
//
// Wrapping rather than counting down at nine, so there is one rule instead of
// two. A digit that sometimes went up and sometimes down would be a second thing
// to explain in the manifest for no gain: what the entry promises is a value one
// away from the right one, and both directions are that.
func next(digit byte) byte {
	if digit == '9' {
		return '0'
	}
	return digit + 1
}
