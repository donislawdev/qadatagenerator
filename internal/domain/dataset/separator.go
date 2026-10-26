package dataset

import "unicode/utf8"

// UsableSeparator reports whether the text can separate the fields of a
// delimited table, and gives the character when it can.
//
// # One rule for reading, writing and the recipe
//
// 🔴 Until 2026-10-26 every place asked only whether the text was one
// character. A line feed is one character, and a recipe giving it produced a
// file in which the end of a field and the end of a record were the same byte:
// two records of two numbers came out as six lines, and both verbs ended with
// success. Found by an outside review on 2026-10-22. A quote was accepted the
// same way and gave a file nothing could read back. The recipe, the writer and
// the reader of somebody else's file now ask this one question, so the three
// cannot come to disagree about which characters a table may be split on.
//
// # Why exactly these characters
//
// They are the ones the reader of this format refuses as a separator: a quote
// and a line end already mean something in the format, a zero byte ends text
// for many readers, and the replacement character is what an unreadable byte
// becomes. Allowing one of them on writing would produce a file this tool's
// own reader cannot open.
//
// The empty text is refused here as well. A caller with a default for it, such
// as the comma, applies the default before asking.
func UsableSeparator(given string) (rune, bool) {
	if utf8.RuneCountInString(given) != 1 {
		return 0, false
	}
	separator, _ := utf8.DecodeRuneInString(given)
	switch separator {
	case 0, '"', '\r', '\n', utf8.RuneError:
		return 0, false
	}
	return separator, true
}
