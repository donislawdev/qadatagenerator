package version

import "strings"

// Reading is what this build can do with a document written to some version of
// a format it knows.
//
// # Why this exists at all
//
// Both readable formats carry the version they were written to, and until
// 2026-08-17 that field was checked for being PRESENT and never for what it
// said. A response record claiming version 99.0, or banana, was compared entry
// by entry and the run ended saying the application had behaved as promised.
// That is a verdict about somebody else's system, drawn from a document this
// build cannot claim to understand.
type Reading int

const (
	// AsWritten means the document names the version this build speaks.
	AsWritten Reading = iota
	// Earlier means the same major version and an earlier minor one. The
	// format grows by adding, so everything such a document can say is
	// something this build knows, and it is read by the rules of its own
	// version without a word about it.
	Earlier
	// Later means the same major version and a minor one this build has never
	// seen. The format grows by adding, so such a document stays readable and
	// the only honest thing to do is read it and say so.
	//
	// # Why Earlier and Later are two answers and not one
	//
	// They were one until 2026-10-27, when the response format moved to 0.2
	// and a record written to 0.1 would have been read with the note that
	// anything the newer version added was left out. That sentence is false
	// about an older document, and a reader who believed it would go looking
	// for a part of their file that was never missing.
	Later
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
	switch order(minors(speaks), minors(claimed)) {
	case 0:
		return AsWritten
	case 1:
		return Earlier
	}
	return Later
}

// minors is the part of a version after its major number, one number for each
// part, and nil when any part is not a plain decimal number.
//
// A part that cannot be read as a number cannot be placed before or after the
// version this build speaks. Such a document is read as Later, because that is
// the answer that tells its reader something may have been left unread, and
// the other one would tell them nothing was.
func minors(version string) []int {
	_, rest, found := strings.Cut(version, ".")
	if !found {
		return []int{}
	}
	parts := strings.Split(rest, ".")
	numbers := make([]int, 0, len(parts))
	for _, part := range parts {
		number, ok := plain(part)
		if !ok {
			return nil
		}
		numbers = append(numbers, number)
	}
	return numbers
}

// plain reads a part of a version that is a plain decimal number, refusing a
// leading nought for the reason major gives.
func plain(part string) (int, bool) {
	if part == "" || len(part) > 9 || (len(part) > 1 && part[0] == '0') {
		return 0, false
	}
	number := 0
	for _, digit := range part {
		if digit < '0' || digit > '9' {
			return 0, false
		}
		number = number*10 + int(digit-'0')
	}
	return number, true
}

// order compares the minor parts of two versions: 0 when they are the same, 1
// when the claimed one comes first and -1 when it comes after. A missing part
// counts as nought, so 0.1 and 0.1.0 are one version. A claimed version whose
// parts could not be read comes after, and so does one the build speaking
// could not read, which only a build with a broken constant would be.
func order(speaks, claimed []int) int {
	if speaks == nil || claimed == nil {
		return -1
	}
	for at := 0; at < len(speaks) || at < len(claimed); at++ {
		spoken, given := partAt(speaks, at), partAt(claimed, at)
		switch {
		case given < spoken:
			return 1
		case given > spoken:
			return -1
		}
	}
	return 0
}

func partAt(parts []int, at int) int {
	if at < len(parts) {
		return parts[at]
	}
	return 0
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
