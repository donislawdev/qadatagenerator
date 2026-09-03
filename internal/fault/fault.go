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

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

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
	// instead is what to do about this one, for the few that know it better
	// than the surface can work it out. Empty for nearly all of them, which is
	// what DoInstead below is about.
	instead string
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

// DoInstead attaches what to do about this failure and hands the failure back,
// so that it reads as one expression at the place the failure is made.
//
// # Why the surface does not work this out on its own, as it does for the rest
//
// It answers by kind, and that is right nearly always: a path that cannot be
// written needs different action from a recipe that cannot be satisfied, and
// one sentence per kind is the only version that cannot fall out of step with
// the messages. It is wrong where the kind is too coarse to say anything true.
// A destination that is already taken is an IO failure, and the sentence for an
// IO failure asks the reader to check that the path exists and that this
// account may write there, which is the one thing that is not the matter. A
// message whose last line contradicts its first is worse than one with no last
// line.
//
// # Why these words may not be a surface's words
//
// They travel to every surface, so they have to be true on all of them. The
// window has no flags and the command line has no tick boxes. What goes here
// says what to do and not which control does it, and a surface with better
// words of its own is free to use them.
func (e *Error) DoInstead(instead string) *Error {
	e.instead = instead
	return e
}

// InsteadOf returns what to do about a failure when the failure carries it, and
// nothing when it does not, which is the ordinary case.
func InsteadOf(err error) string {
	var carrier *Error
	if errors.As(err, &carrier) {
		return carrier.instead
	}
	return ""
}

// Says is the whole of what a failure has to say, as one piece of prose.
//
// It is for a surface that shows a failure in one place rather than in parts.
// The command line shows the four parts of a refusal on lines of their own and
// does not use this. The window shows one label, and a window showing only the
// message would drop the sentence that message used to carry.
func Says(err error) string {
	if err == nil {
		return ""
	}
	instead := InsteadOf(err)
	if instead == "" {
		return err.Error()
	}
	// The next move is written as a phrase rather than a sentence, because the
	// command line puts it after a label of its own. A surface joining it to
	// the message has to make a sentence of it.
	first, width := utf8.DecodeRuneInString(instead)
	return err.Error() + ". " + string(unicode.ToUpper(first)) + instead[width:]
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
		return carrier.Kind()
	}
	return Unknown
}
