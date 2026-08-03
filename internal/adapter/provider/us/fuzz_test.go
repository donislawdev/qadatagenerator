package us_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// seeds are the shapes worth starting from: correct values, values broken one
// rule at a time, and the shapes that are not numbers at all.
var seeds = []string{
	"666-12-3456",
	"666123456",
	"123-45-6789",
	"000-12-3456",
	"666-00-3456",
	"666-12-0000",
	"900-70-1234",
	"899-70-1234",
	"999-99-9999",
	"665-12-3456",
	"667-12-3456",
	"",
	"0",
	"000000000",
	"999999999",
	"66612345",   // one short
	"6661234567", // one long
	"666-12-345x",
	"6661-2-3456", // separators in the wrong place
	"666-123456",  // one separator only
	"666 12 3456",
	"           ",
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00",
	"٦٦٦١٢٣٤٥٦", // digits of another script
	"666-12-345é",
}

// The two validators are written from one pair of sentences by one hand, so
// the interesting question is not whether they agree about the values we
// produced. It is whether anything at all makes them disagree.
//
// A differential fuzz is the only way to ask that without imagining the answer
// first. Four states of a path are written by somebody who has to think of
// them. A fuzzer thinks of nothing and therefore finds different things.
func FuzzValidateAgreesWithTheIndependentValidator(f *testing.F) {
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		ours := us.SSN{}.Validate(value)
		theirs := oracle.CheckSSN(value)

		if ours.Valid != (len(theirs) == 0) {
			t.Fatalf("the two validators disagree about %q: ours says valid=%v (%s), the independent one says %v",
				value, ours.Valid, ours.Reason, theirs)
		}
		if !ours.Valid && ours.Reason == "" {
			t.Fatalf("%q was refused without a reason, which is the silence this tool argues against", value)
		}
	})
}

// The second question, fuzzed separately because it has a separate answer. The
// two validators reach it by different routes: one compares numbers, the other
// compares the digits of a value against the digits printed in the quoted
// sentence, spans included.
func FuzzTheNeverAssignedAnswerAgreesWithTheIndependentValidator(f *testing.F) {
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		ours := us.SSN{}.InReservedRange(value)
		if theirs := oracle.SSNNeverAssigned(value); ours != theirs {
			t.Fatalf("the two disagree about whether %q could ever be issued: ours says %v, the independent one says %v",
				value, ours, theirs)
		}
	})
}

// Nothing a person can put in a file may take the process down. A recipe comes
// from somebody else's repository, so the values it names are untrusted input
// rather than our own data.
func FuzzTheReadersNeverPanic(f *testing.F) {
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		subject := us.SSN{}

		// Each of these takes a value from outside and has to answer rather
		// than fail. The results are deliberately unused: the property under
		// test is that the call returns at all.
		_ = subject.InReservedRange(value)
		for _, target := range subject.UnassignedTargets() {
			_, _ = subject.WithUnassigned(value, target)
		}
		_, _ = subject.WithUnassigned(value, value)

		// A value this build calls well formed has to survive being moved into
		// every block and come back refused. Anything else would mean the
		// corruption can report success while leaving a number an ordinary
		// validator still accepts.
		if !subject.Validate(value).Valid {
			return
		}
		for _, target := range subject.UnassignedTargets() {
			moved, err := subject.WithUnassigned(value, target)
			if err != nil {
				t.Fatalf("%q is well formed and moving it into %s failed: %v", value, target, err)
			}
			if subject.Validate(moved).Valid {
				t.Fatalf("%q moved into %s became %q, which is still well formed", value, target, moved)
			}
		}
	})
}
