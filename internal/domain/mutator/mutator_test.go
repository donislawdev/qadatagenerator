package mutator_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// The three types are not a convenience. Measured on 2026-08-11: the catalog
// reader hands whole numbers over as uint64, which is what the YAML library
// produces for an unsigned literal, a negative one arrives as int64, and a
// parameter built in a test is written as a plain int. A reader that knew only
// the one the catalog happens to use today would refuse a value somebody wrote
// by hand and look like a defect in their policy.
func TestAWholeNumberIsReadFromEachOfTheThreeTypesItCanArriveAs(t *testing.T) {
	written := []struct {
		how   string
		value any
	}{
		{"a plain int, which is how a parameter built by hand is written", 7},
		{"an int64, which is the type a signed literal arrives as", int64(7)},
		{"a uint64, which is what the catalog reader produces", uint64(7)},
	}
	for _, kind := range written {
		params := mutator.Parameters{"overshoot": kind.value}
		read, ok := params.Whole("overshoot")
		if !ok {
			t.Errorf("%s was refused, so a policy written that way would look faulty", kind.how)
			continue
		}
		if read != 7 {
			t.Errorf("%s was read as %d rather than 7", kind.how, read)
		}
	}
}

// A negative knob is a value somebody can write and the reader hands it over
// rather than refusing it. What a negative one means belongs to the mutator
// that was given it, and a reader deciding that here would be a second place
// where the same question is answered.
func TestANegativeWholeNumberReachesTheMutatorRatherThanBeingRefusedHere(t *testing.T) {
	params := mutator.Parameters{"drift_years": int64(-3)}
	read, ok := params.Whole("drift_years")
	if !ok {
		t.Fatal("a negative parameter was refused by the reader of the knob")
	}
	if read != -3 {
		t.Fatalf("a negative parameter was read as %d rather than -3", read)
	}
}

// A fraction is refused rather than truncated, because truncating 2.5 to 2
// would be this tool deciding how much damage to do. The refusal is about the
// type and not about the value, which is why a fraction that happens to be
// whole is refused as well: the catalog carries fractions for other parameters
// and the two kinds have to stay apart.
func TestAFractionIsRefusedRatherThanTruncated(t *testing.T) {
	written := []struct {
		how   string
		value any
	}{
		{"a fraction with something after the point", 2.5},
		{"a fraction written at single precision", float32(2.5)},
		{"a fraction that happens to land on a whole number", 2.0},
	}
	for _, kind := range written {
		params := mutator.Parameters{"overshoot": kind.value}
		if read, ok := params.Whole("overshoot"); ok {
			t.Errorf("%s was read as the whole number %d", kind.how, read)
		}
	}
}

// A mutator asked to use a knob it was never given has nothing to work from,
// and inventing one would be this tool deciding what damage to do. Both readers
// therefore answer twice: what was there, and whether anything was.
func TestAParameterNobodyWroteIsRefusedRatherThanDefaulted(t *testing.T) {
	empty := mutator.Parameters{}
	if read, ok := empty.Whole("overshoot"); ok {
		t.Errorf("a whole number nobody wrote was handed over as %d", read)
	}
	if read, ok := empty.Text("replacement"); ok {
		t.Errorf("a text nobody wrote was handed over as %q", read)
	}
}

// Neither reader converts. A policy naming the wrong kind of value is a policy
// somebody has to fix, and a reader that quietly made it work would hide the
// mistake until the damage came out wrong.
func TestAParameterOfTheWrongKindIsRefusedRatherThanConverted(t *testing.T) {
	params := mutator.Parameters{"replacement": 7, "overshoot": "7"}
	if read, ok := params.Text("replacement"); ok {
		t.Errorf("a number was read as the text %q", read)
	}
	if read, ok := params.Whole("overshoot"); ok {
		t.Errorf("text was read as the whole number %d", read)
	}
}

// Damage either happened or it did not, and either way something comes back
// saying so. The two constructors are the only writers of an outcome, so the
// combination that says both things at once cannot be written down.
func TestAnOutcomeSaysEitherWhatItWroteOrWhyItWroteNothing(t *testing.T) {
	done := mutator.Done("90041158021")
	if !done.Applied() {
		t.Error("damage that happened reported that it did not")
	}
	if done.Value() != "90041158021" {
		t.Errorf("the value that took the old one's place came back as %q", done.Value())
	}
	if done.Reason() != "" {
		t.Errorf("damage that happened also carried the refusal %q", done.Reason())
	}

	const why = "the value holds no separator to double"
	skipped := mutator.Skipped(why)
	if skipped.Applied() {
		t.Error("damage that did not happen reported that it did")
	}
	if skipped.Reason() != why {
		t.Errorf("the reason came back as %q rather than %q", skipped.Reason(), why)
	}
	if skipped.Value() != "" {
		t.Errorf("damage that did not happen also carried the value %q", skipped.Value())
	}
	if skipped.ValueUnfit() || done.ValueUnfit() {
		t.Error("only a value found unfit may say so, and a plain refusal or done damage did")
	}

	const unfitWhy = "the value is empty"
	unfit := mutator.Unfit(unfitWhy)
	if unfit.Applied() || !unfit.ValueUnfit() || unfit.Reason() != unfitWhy || unfit.Value() != "" {
		t.Errorf("an unfit value came back as applied %v, unfit %v, reason %q, value %q",
			unfit.Applied(), unfit.ValueUnfit(), unfit.Reason(), unfit.Value())
	}
}

// An outcome nobody built is a refusal without a reason rather than damage that
// wrote an empty value. That is the safer of the two, and it is worth naming
// because the zero value is what a mutator returns when a path forgets to.
func TestTheOutcomeNobodyBuiltIsARefusalRatherThanAnAppliedEmptyValue(t *testing.T) {
	var nothing mutator.Outcome
	if nothing.Applied() {
		t.Error("an outcome nobody built reported damage that happened")
	}
}
