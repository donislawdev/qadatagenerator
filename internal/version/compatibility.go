package version

import "strings"

// Reading is what this build can do with a document written to some version of
// a format it knows.
//
// # Why this exists at all
//
// Both readable formats carry the version they were written to, and until
// 2026-08-13 that field was checked for being PRESENT and never for what it
// said. A response record claiming version 99.0, or banana, was compared entry
// by entry and the run ended saying the application had behaved as promised.
// That is a verdict about somebody else's system, drawn from a document this
// build cannot claim to understand.
type Reading int

const (
	// AsWritten means the document names the version this build speaks.
	AsWritten Reading = iota
	// WithinTheLine means the same major version and a different minor one.
	// The format grows by adding, so such a document stays readable and the
	// only honest thing to do is read it and say so.
	WithinTheLine
	// Unreadable means a different major version, or a string naming no
	// version at all. A major version is what changes the meaning of what is
	// already there, so a reader meeting one cannot know what it is holding.
	Unreadable
)

// Compatibility says what a build speaking one version can do with a document
// claiming another.
//
// # Why the major version is the line and not a list of known versions
//
// The specification of the manifest tells consumers to ignore fields they do
// not know, so that a document from a newer producer stays readable. That is a
// rule about growth by adding, which is what a minor version is. It says
// nothing about a version whose major part moved, and a major move is by
// definition the one that changes what an existing field means. Both halves of
// this function come from that single sentence rather than from two opinions.
//
// An empty claimed version is Unreadable here, though in practice both models
// refuse it earlier with a sentence of their own, which is the better message
// because it can say what the field is for.
func Compatibility(speaks, claimed string) Reading {
	speaks, claimed = strings.TrimSpace(speaks), strings.TrimSpace(claimed)
	if claimed == speaks && speaks != "" {
		return AsWritten
	}
	spoken, ok := major(speaks)
	if !ok {
		return Unreadable
	}
	given, ok := major(claimed)
	if !ok || given != spoken {
		return Unreadable
	}
	return WithinTheLine
}

// major returns the part before the first point, and refuses anything that is
// not a plain decimal number.
//
// The refusal covers the whole part rather than its first character, because
// 9beta names no version this build could reason about and reading it as nine
// would be the guess this function exists to avoid. A leading nought is refused
// for the same reason the recipe reader refuses one in a count: 01 and 1 would
// otherwise be two spellings of one version, and nothing would say which was
// meant.
func major(version string) (string, bool) {
	first, _, _ := strings.Cut(version, ".")
	if first == "" || (len(first) > 1 && first[0] == '0') {
		return "", false
	}
	for _, digit := range first {
		if digit < '0' || digit > '9' {
			return "", false
		}
	}
	return first, true
}
