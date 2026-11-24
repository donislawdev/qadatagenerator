package dataset

import "strings"

// UsableEncoding reports whether a recipe may ask for this encoding, which is
// whether it names UTF-8, the one encoding every form is written in.
//
// # One rule for the recipe and the writer
//
// The reader of recipes and the writer of the comma separated form both ask
// this, so validate cannot call a recipe usable that generate then refuses. A
// recipe asking for windows-1250 was told both on 2026-11-24, the same
// disagreement UsableSeparator settles for the delimiter.
//
// The empty text is usable, because a recipe that names no encoding gets UTF-8.
func UsableEncoding(name string) bool {
	switch strings.ToLower(name) {
	case "", "utf-8", "utf8":
		return true
	}
	return false
}
