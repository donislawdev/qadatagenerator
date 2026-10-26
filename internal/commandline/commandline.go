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

import (
	"runtime"
	"strings"
)

// The name of the tool and the verbs a window has an equivalent for.
const (
	Tool = "qadg"

	Quick    = "quick"
	Generate = "generate"
	Corrupt  = "corrupt"
	// Verify is the one verb a window has an equivalent for without producing
	// anything: the manifest browser with a response record open is the same
	// comparison qadg verify prints, and the line it shows names both files.
	Verify = "verify"
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
var Verbs = []string{Quick, Generate, Corrupt, Verify}

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

// quote writes a value so that the shell it is pasted into reads it as one
// argument and as nothing else.
//
// # Why every value outside a short list is quoted, not only one with a space
//
// 🔴 Until 2026-10-26 only a space or a tab was quoted, in double quotes, and a
// value carrying a double quote was left bare. The value is very often the name
// of a file somebody else handed over, and a name is allowed to carry what a
// shell reads as code: in PowerShell a name holding a command separator and
// Get-Date ran a second command, and "sample$(Write-Output SAFE).csv" ran one
// inside the quotes, because double quotes there still expand. Found by an outside review on 2026-10-22, which
// gave the line to the parser of PowerShell and counted the commands. Nothing
// here runs the line, so the harm needed a person to paste it, and the window
// offers it for exactly that.
//
// # Why single quotes in both shells, and what differs between them
//
// The owner chose on 2026-10-26 the syntax of the shell a desktop opens by
// default: PowerShell on Windows, sh elsewhere (DEC-266). In both, text in
// single quotes is taken as it stands. The one character that has to be
// written differently is the single quote itself, so that is the only part
// that follows the system. A line pasted into cmd.exe gets the quotes as part
// of the value and fails to find the file, which runs nothing.
//
// What may stay bare is what neither shell gives a meaning to. The comma is not
// on the list although sh would allow it: PowerShell reads an unquoted comma
// as a list, which is the same trap rule 16 of the project notes for an equals
// sign beside one. Keeping one list for both systems is also what keeps the
// recorded lines of the screens the same on every system.
func quote(value string) string {
	return quoteFor(value, runtime.GOOS == "windows")
}

// quoteFor is quote for a named system, so that a test on one system can read
// what the other gets.
func quoteFor(value string, windows bool) string {
	if bare(value, windows) {
		return value
	}
	if windows {
		// PowerShell takes five characters as a single quote, and a quote
		// inside a quoted value is written twice.
		return "'" + powerShellQuotes.Replace(value) + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// powerShellQuotes doubles every character PowerShell reads as a single quote:
// the plain one and four typographic forms of it.
//
// Measured on 2026-10-26 with the parser of PowerShell 7 rather than taken from
// memory: each of the four, left single inside quotes, closed them and gave the
// line a second command, and a double quote there did not.
var powerShellQuotes = strings.NewReplacer(
	"'", "''", "‘", "‘‘", "’", "’’",
	"‚", "‚‚", "‛", "‛‛")

// bare reports whether a value means only itself in both shells as it stands.
//
// ASCII letters and digits and eight marks, with the backslash as well on
// Windows, where it separates the parts of a path and means nothing to
// PowerShell. Everything else is quoted, including every character outside
// ASCII, because PowerShell reads some of them as quotes and dashes.
func bare(value string, windows bool) bool {
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z', character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9':
		case strings.ContainsRune("_-./:=+%", character):
		case character == '\\' && windows:
		default:
			return false
		}
	}
	return true
}
