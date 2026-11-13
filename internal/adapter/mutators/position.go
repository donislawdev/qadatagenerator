package mutators

import (
	"fmt"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// The three places the position knob names for the entries that put something
// inside a value. The knob is declared beside the check digit entries, which
// read the same name with values of their own.
const (
	positionStart  = "start"
	positionMiddle = "middle"
	positionEnd    = "end"
)

// positionOf reads where the insertion goes, or the sentence saying why there
// is nothing to work with.
func positionOf(params mutator.Parameters) (string, string) {
	position, given := params.Text(positionParameter)
	if !given {
		return "", fmt.Sprintf("no %s was given, so nothing says where in the value to put it",
			positionParameter)
	}
	switch position {
	case positionStart, positionMiddle, positionEnd:
		return position, ""
	}
	return "", fmt.Sprintf("the %s %q is none of %s, %s or %s", positionParameter, position,
		positionStart, positionMiddle, positionEnd)
}

// placed puts an insertion into a value at a named place, or says why it
// cannot.
//
// The start and the end are always there. The middle is the place nearest the
// middle with an ordinary character on each side, and a value without one is
// refused rather than given the insertion at an edge, because an edge is a
// different test from the inside of a word.
func placed(value, insertion, position string) (string, string) {
	switch position {
	case positionStart:
		return insertion + value, ""
	case positionEnd:
		return value + insertion, ""
	}
	at, found := middleOf(value)
	if !found {
		return "", "the value has no place inside it with an ordinary letter, digit, space or " +
			"punctuation mark on each side, so there is no middle this build can be sure of"
	}
	return value[:at] + insertion + value[at:], ""
}

// middleOf returns the byte offset nearest the middle of a value at which a
// character can go in between two characters a reader sees, and false when
// there is no such place inside the value.
//
// # Why only between two ordinary characters
//
// Where one character ends and the next begins is a published algorithm with
// tables of its own, and this build does not carry them. Writing part of it
// here was tried first and missed rules the published one has: the conjuncts
// of twenty scripts, the characters that attach to what follows them, and a
// Thai vowel that is a letter and joins the letter before it. So this does not
// decide where characters meet. It takes a place only where both
// neighbours are ordinary, which the algorithm never joins to anything, and a
// test holds that claim against the published algorithm for every character
// Unicode defines.
//
// The price is a refusal for a value with no such place, such as a word in
// Thai or Korean or a row of emoji, and the refusal says so in the manifest.
func middleOf(value string) (int, bool) {
	var runes []rune
	var offsets []int
	for at, r := range value {
		runes = append(runes, r)
		offsets = append(offsets, at)
	}
	best, found := 0, false
	for i := 1; i < len(runes); i++ {
		if !ordinary(runes[i-1]) || !ordinary(runes[i]) {
			continue
		}
		// Twice the distance, so that the middle of an odd count is compared
		// in whole numbers. The earlier of two equally near places wins, which
		// keeps the answer the same on every run.
		if !found || distance(2*i, len(runes)) < distance(2*best, len(runes)) {
			best, found = i, true
		}
	}
	if !found {
		return 0, false
	}
	return offsets[best], true
}

// ordinary reports a character the segmentation of text never joins to a
// neighbour: a letter of a script whose letters stand alone, a digit, a
// punctuation mark, a mathematical or currency sign, or a space.
//
// What it leaves out, and why each is left out rather than handled:
//   - marks, joiners, variation selectors, skin tones and tags, which attach
//     to the character before them
//   - the scripts that build one character from several letters or join a
//     letter to the next through a virama, such as Devanagari, Thai and Korean
//     written as jamo
//   - other symbols, which hold the emoji, the regional indicators read in
//     pairs, and the modifiers
//   - control characters, which hold the carriage return that is one line
//     break with the line feed after it
func ordinary(r rune) bool {
	if unicode.IsLetter(r) {
		return unicode.In(r, standalone...)
	}
	return unicode.In(r, unicode.N, unicode.P, unicode.Sm, unicode.Sc, unicode.Zs)
}

// standalone are the scripts whose letters are each one character to a reader.
var standalone = []*unicode.RangeTable{
	unicode.Latin, unicode.Greek, unicode.Cyrillic, unicode.Armenian, unicode.Hebrew,
	unicode.Arabic, unicode.Han, unicode.Hiragana, unicode.Katakana,
}

func distance(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
