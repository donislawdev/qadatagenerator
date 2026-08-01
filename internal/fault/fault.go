// Package fault carries the kind of a failure from the place where it happens
// to the place that has to report it.
//
// The kind travels with the error instead of being chosen where the program
// ends. A switch at the top gives every new failure mode the default code 1
// and quietly breaks the promise that every ending has a code from the table.
// The breakage is invisible, because the run still ends with a number.
//
// This package knows nothing about numbers. Turning a kind into an exit code
// belongs to the command line adapter, which is the only layer that owns that
// contract.
package fault

import "errors"

// Kind says what class of failure happened. It is deliberately small: a kind
// exists only when somebody outside the program has to tell two failures
// apart.
type Kind int

const (
	// Unknown is the zero value and means that no kind was ever set.
	//
	// It is first on purpose. A struct built without a kind must not look
	// like a real one, because that is exactly how a failure mode acquires
	// a wrong code without anybody noticing.
	Unknown Kind = iota

	// Runtime is a tool failure that belongs nowhere else.
	Runtime

	// Usage is a wrong invocation: bad syntax, an unknown flag, flags that
	// contradict each other.
	Usage

	// Config is a recipe or a corruption policy that is not valid: bad
	// syntax, an unknown mutator identifier, contradictory constraints.
	Config

	// Expectation means the data was produced but the application under
	// test did not behave the way the expectation manifest says.
	//
	// It describes the application under test rather than this tool, and it
	// can only arise in verify. Nothing else has a way to know what the
	// application did.
	Expectation

	// Data means the recipe is valid but cannot be satisfied. Uniqueness on
	// a field with five possible values across a thousand rows is the
	// standing example.
	Data

	// IO is a failed read or write, no space left, no permission.
	IO

	// Interrupted means the user stopped the run.
	Interrupted

	// Terminated means the system stopped the run.
	Terminated
)

// String returns the name used in logs and in the exit code table. The names
// are part of what a person reads in a CI log, so they do not change lightly.
func (k Kind) String() string {
	switch k {
	case Runtime:
		return "RUNTIME"
	case Usage:
		return "USAGE"
	case Config:
		return "CONFIG"
	case Expectation:
		return "EXPECTATION"
	case Data:
		return "DATA"
	case IO:
		return "IO"
	case Interrupted:
		return "INTERRUPTED"
	case Terminated:
		return "TERMINATED"
	default:
		return "UNKNOWN"
	}
}

// Error is a failure that knows its own kind.
type Error struct {
	kind    Kind
	message string
	wrapped error
}

// New reports a failure that has no underlying cause worth keeping.
func New(kind Kind, message string) *Error {
	return &Error{kind: kind, message: message}
}

// Wrap attaches a kind and context to a failure that came from elsewhere. The
// original stays reachable through errors.Is and errors.As.
func Wrap(kind Kind, wrapped error, message string) *Error {
	return &Error{kind: kind, message: message, wrapped: wrapped}
}

func (e *Error) Error() string {
	switch {
	case e.message == "" && e.wrapped == nil:
		// A failure with nothing to say is still a failure. Naming the
		// kind beats an empty line, which would reach the user as a
		// prefix with nothing after it.
		return "unspecified " + e.kind.String() + " failure"
	case e.wrapped == nil:
		return e.message
	case e.message == "":
		return e.wrapped.Error()
	default:
		return e.message + ": " + e.wrapped.Error()
	}
}

// Unwrap keeps the chain intact for errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.wrapped }

// Kind returns the kind this failure was created with.
func (e *Error) Kind() Kind { return e.kind }

// KindOf returns the kind carried by err, or Unknown when nothing in the chain
// carries one.
//
// Unknown is a real answer here and the caller has to handle it. An error that
// reached the surface without a kind is a defect in this program, and saying so
// out loud is cheaper than guessing a code for it.
func KindOf(err error) Kind {
	var carrier *Error
	if errors.As(err, &carrier) {
		return carrier.kind
	}
	return Unknown
}
