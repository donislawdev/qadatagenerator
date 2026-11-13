package manifestview

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// pairShown is a value and the one that replaced it, as the card writes them.
//
// # Why a pair is written out where a single value is not
//
// Damage that writes a value in another Unicode form leaves two strings that
// differ in every letter it decomposed and that the font draws the same,
// because an accent stored as a character of its own is drawn on the letter
// before it. Whether such an accent has to be written out cannot be told from
// one value. Accents stored apart are how Devanagari, Hebrew with its vowels
// and Thai are ordinarily written, and writing them out in every value would
// make an ordinary value unreadable on the one screen meant for reading it. It
// can be told from the pair: two values of different bytes that are the same
// text by the Unicode definition of canonical equivalence can only be told
// apart by their code points. Any other pair either looks different already or
// differs by a character that draws nothing, which visible writes out on its
// own.
//
// # What is written out
//
// The two values are walked together, piece by piece, where a piece is the
// shortest stretch of each that decomposes to the same text. A piece the same
// in both is drawn as it is, so a letter the damage left alone keeps its
// accent drawn. Of a piece that differs, the side with more code points is
// written out, and the after where both have as many. Its first character is
// drawn and every one after it is written as its code point, so a Z with a dot
// above stored as two characters reads Z<U+0307>. A piece of one character is
// written out whole, because a character replaced by another of the same look,
// such as the ohm sign by the Greek capital omega, has nothing else to show.
func pairShown(before, after string) (shownBefore, shownAfter string) {
	if !sameTextApart(before, after) {
		return visible(before), visible(after)
	}
	var builtBefore, builtAfter strings.Builder
	for _, one := range piecesOf(before, after) {
		switch {
		case one.before == one.after:
			builtBefore.WriteString(visible(one.before))
			builtAfter.WriteString(visible(one.after))
		case utf8.RuneCountInString(one.before) > utf8.RuneCountInString(one.after):
			builtBefore.WriteString(writtenOut(one.before))
			builtAfter.WriteString(visible(one.after))
		default:
			builtBefore.WriteString(visible(one.before))
			builtAfter.WriteString(writtenOut(one.after))
		}
	}
	return builtBefore.String(), builtAfter.String()
}

// sameTextApart reports two values of different bytes that are the same text,
// which is when they decompose to the same thing.
//
// # Why it asks whether each is already normalised first
//
// Each normalisation form has one way of writing a text, so two different
// values already in the same form are different text. Asking that copies
// nothing, and decomposing does. Nearly every pair a manifest carries is two
// values in the composed form, a value and what other damage made of it, and
// it is answered here without a copy.
//
// The question is put as a quick span rather than as IsNormalString, which
// copies too. A span that stops short of the end does not prove a value is not
// normalised, only that the quick answer is unsure, and such a pair goes on to
// be decomposed, so an unsure answer costs time and never a wrong screen.
func sameTextApart(before, after string) bool {
	if before == after {
		return false
	}
	if alreadyIn(norm.NFC, before) && alreadyIn(norm.NFC, after) {
		return false
	}
	if alreadyIn(norm.NFD, before) && alreadyIn(norm.NFD, after) {
		return false
	}
	return norm.NFD.String(before) == norm.NFD.String(after)
}

// alreadyIn reports a value the quick check finds in the form all the way to
// its end.
func alreadyIn(form norm.Form, value string) bool {
	return form.QuickSpanString(value) == len(value)
}

// piece is a stretch of each of two values, and the two decompose to the same
// text.
type piece struct{ before, after string }

// piecesOf cuts two values that are the same text into the shortest stretches
// that decompose to the same text, in order.
//
// Each value is decomposed one segment at a time, and a stretch ends where the
// two have produced as much decomposed text as each other. Both decompose to
// the same text, so as much is the same. A segment is not always as long on
// both sides: a Hangul syllable is one segment and its three letters stored
// apart are three, which is why the walk moves whichever side is behind rather
// than both at once.
//
// The last step always ends a stretch, because the two decomposed forms were
// compared before the walk. The line after the loop keeps the rest of either
// value should a library ever cut segments differently from how it normalises
// a whole string, so that no part of a value is dropped from the screen.
func piecesOf(before, after string) []piece {
	var walkBefore, walkAfter norm.Iter
	walkBefore.InitString(norm.NFD, before)
	walkAfter.InitString(norm.NFD, after)
	var pieces []piece
	fromBefore, fromAfter, grownBefore, grownAfter := 0, 0, 0, 0
	for !walkBefore.Done() || !walkAfter.Done() {
		if !walkBefore.Done() && (grownBefore <= grownAfter || walkAfter.Done()) {
			grownBefore += len(walkBefore.Next())
		} else {
			grownAfter += len(walkAfter.Next())
		}
		if grownBefore == grownAfter {
			pieces = append(pieces, piece{before[fromBefore:walkBefore.Pos()], after[fromAfter:walkAfter.Pos()]})
			fromBefore, fromAfter = walkBefore.Pos(), walkAfter.Pos()
		}
	}
	if fromBefore < len(before) || fromAfter < len(after) {
		pieces = append(pieces, piece{before[fromBefore:], after[fromAfter:]})
	}
	return pieces
}

// writtenOut draws the first character of a stretch and writes every one after
// it as its code point, or writes the stretch out whole when it is one
// character.
func writtenOut(stretch string) string {
	var built strings.Builder
	_, size := utf8.DecodeRuneInString(stretch)
	if size < len(stretch) {
		built.WriteString(visible(stretch[:size]))
		stretch = stretch[size:]
	}
	for _, r := range stretch {
		writeCodePoint(&built, r)
	}
	return built.String()
}
