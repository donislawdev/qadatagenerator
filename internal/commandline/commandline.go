// Package commandline names the verbs and flags of the command line, and
// assembles one line out of them.
//
// # Why this exists, and why it is not in internal/cli
//
// docs/08 asks every screen that produces a dataset to carry "Copy as a CLI
// command", and says why in the same breath: without it the window and the
// terminal become two products. Measured on 2026-09-08, the whole time the
// window has existed, nothing in internal/gui contained the words. The rule was
// written down and nothing held it, so it was never built.
//
// The window cannot import internal/cli, and it must not keep a second copy of
// the flag names either, because a second copy of a register is how registers
// drift. So the names live here, in a package with no dependencies, that both
// surfaces may use. That the names here are the ones the command line really
// takes is closed by a test inside internal/cli, which reads its own help
// register and compares.
//
// # What this package is NOT
//
// It does not run anything and it never will. It writes a line for a person to
// read and to paste. A window that could run the command line would be a second
// way to do everything, and the two would answer differently the first time one
// of them changed.
package commandline

import "strings"

// The name of the tool and the verbs a window has an equivalent for.
const (
	Tool = "qadg"

	Quick    = "quick"
	Generate = "generate"
	Corrupt  = "corrupt"
)

// The flags those verbs take.
const (
	BirthDate     = "--birth-date"
	Count         = "--count"
	Delimiter     = "--delimiter"
	DryRun        = "--dry-run"
	ExcludeFields = "--exclude-fields"
	Format        = "--format"
	Gender        = "--gender"
	Layers        = "--layers"
	Locale        = "--locale"
	Manifest      = "--manifest"
	Out           = "--out"
	Overwrite     = "--overwrite"
	PerRow        = "--per-row"
	Rows          = "--rows"
	Seed          = "--seed"
)

// Verbs is every verb named above, so a test can close the register from the
// side the constants are on.
var Verbs = []string{Quick, Generate, Corrupt}

// Flags is every flag named above, for the same reason.
var Flags = []string{
	BirthDate, Count, Delimiter, DryRun, ExcludeFields, Format, Gender, Layers,
	Locale, Manifest, Out, Overwrite, PerRow, Rows, Seed,
}

// Line is a command being written out.
//
// The order things are added is the order they appear, because a person reads
// this to understand what the window did, and a line whose flags arrive in a
// different order on every draw is one they have to read again each time. Go
// randomises map iteration by decision, so nothing here is built from a map.
type Line struct {
	verb  string
	parts []string
}

// New starts a line for one verb.
func New(verb string) *Line { return &Line{verb: verb} }

// Flag adds a flag with a value, and adds NOTHING when the value is empty.
//
// Empty means the person left the setting alone, and a flag carrying an empty
// value is not the same request: --seed with nothing after it is an error,
// while no --seed at all is what the window actually did.
func (l *Line) Flag(name, value string) *Line {
	if value == "" {
		return l
	}
	l.parts = append(l.parts, name, quote(value))
	return l
}

// Switch adds a flag that carries no value, when it is on.
func (l *Line) Switch(name string, on bool) *Line {
	if on {
		l.parts = append(l.parts, name)
	}
	return l
}

// Argument adds what the verb acts on, which goes last.
func (l *Line) Argument(value string) *Line {
	if value == "" {
		return l
	}
	l.parts = append(l.parts, quote(value))
	return l
}

// String is the line as somebody would type it.
func (l *Line) String() string {
	return strings.Join(append([]string{Tool, l.verb}, l.parts...), " ")
}

// quote wraps a value that would otherwise arrive as more than one argument.
//
// A path with a space in it is the ordinary case on this desktop, not an edge
// one: the folder a person writes to is under Documents more often than not.
// A value already carrying a double quote is left alone rather than escaped,
// because the escaping differs between the shells this line may be pasted into
// and a line that is wrong in one of them is worse than a line that is plainly
// unquoted.
func quote(value string) string {
	if !strings.ContainsAny(value, " \t") || strings.Contains(value, `"`) {
		return value
	}
	return `"` + value + `"`
}
