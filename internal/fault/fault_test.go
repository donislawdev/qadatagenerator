package fault_test

import (
	"errors"
	"io"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

func TestKindOfFindsTheKindThroughWrapping(t *testing.T) {
	deep := fault.New(fault.Config, "unknown mutator identifier")
	wrapped := fault.Wrap(fault.IO, deep, "cannot read the recipe")

	// The outermost kind wins. The place that added context knew more about
	// how the failure should be reported than the place below it.
	if got := fault.KindOf(wrapped); got != fault.IO {
		t.Fatalf("expected IO, got %v", got)
	}
	if !errors.Is(wrapped, deep) {
		t.Fatal("wrapping must keep the original reachable")
	}
}

func TestKindOfSaysUnknownRatherThanGuessing(t *testing.T) {
	// An error from elsewhere carries no kind of ours. Answering with any
	// real kind here would be the guess that gives every new failure mode
	// the same code by accident.
	if got := fault.KindOf(io.EOF); got != fault.Unknown {
		t.Fatalf("expected Unknown, got %v", got)
	}
	if got := fault.KindOf(nil); got != fault.Unknown {
		t.Fatalf("expected Unknown for no error at all, got %v", got)
	}
}

func TestZeroValueIsUnknown(t *testing.T) {
	var kind fault.Kind
	if kind != fault.Unknown {
		t.Fatal("the zero value has to be the one that means nothing was set")
	}
	if kind.String() != "UNKNOWN" {
		t.Fatalf("expected UNKNOWN, got %s", kind.String())
	}
}

func TestMessageCarriesBothHalves(t *testing.T) {
	wrapped := fault.Wrap(fault.Data, errors.New("five values for a thousand rows"), "uniqueness cannot be satisfied")
	const want = "uniqueness cannot be satisfied: five values for a thousand rows"
	if wrapped.Error() != want {
		t.Fatalf("expected %q, got %q", want, wrapped.Error())
	}
	bare := fault.New(fault.Data, "uniqueness cannot be satisfied")
	if bare.Error() != "uniqueness cannot be satisfied" {
		t.Fatalf("a failure with no cause repeats nothing, got %q", bare.Error())
	}
	if bare.Unwrap() != nil {
		t.Fatal("a failure with no cause unwraps to nothing")
	}
}

func TestAFailureWithNothingToSayStillNamesItself(t *testing.T) {
	// An empty message would reach the user as a prefix with nothing after
	// it, which reads like the tool lost track of its own failure.
	const want = "unspecified RUNTIME failure"
	if got := fault.New(fault.Runtime, "").Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if got := fault.Wrap(fault.IO, io.EOF, "").Error(); got != io.EOF.Error() {
		t.Fatalf("a cause with no message of its own speaks for itself, got %q", got)
	}
}

func TestEveryKindHasItsOwnName(t *testing.T) {
	kinds := []fault.Kind{
		fault.Unknown, fault.Runtime, fault.Usage, fault.Config,
		fault.Expectation, fault.Data, fault.IO, fault.Interrupted, fault.Terminated,
	}
	seen := make(map[string]bool, len(kinds))
	for _, kind := range kinds {
		name := kind.String()
		if name == "" {
			t.Fatalf("kind %d has no name, and the name is what a person reads in a log", kind)
		}
		if seen[name] {
			t.Fatalf("two kinds answer to %s", name)
		}
		seen[name] = true
	}
}
