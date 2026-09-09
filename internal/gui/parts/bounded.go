package parts

import (
	"strings"

	"fyne.io/fyne/v2/widget"
)

// Bounded keeps a field inside what this product will accept, as somebody
// types or pastes into it.
//
// # Why the field refuses rather than the screen
//
// A screen refusing afterwards is the older answer and it is still there: the
// mode says whether a request is carried out, and it says so in four parts.
// What it cannot do is stop a person believing, for the seconds before they
// press the button, that what they typed is a thing this tool takes.
//
// 🔴 The owner found the gap by typing into the real window on 2026-09-09: a
// number of thirty seven digits went into how many, the screen answered "How
// many is a whole number, such as 10", and the line at the foot offered
// qadg quick --count 1000000000000000000000000000000000000 to copy. Three
// things were wrong at once and only one of them was the refusal: the sentence
// named the wrong fault, because the text IS a whole number and what it is not
// is a number this machine can hold, and the command bridge offered a line the
// screen had just refused to carry out.
//
// A field that will not take the thirty seventh digit answers all three, and it
// answers them before anybody has pressed anything.
//
// # What this is NOT
//
// It is not validation and it does not replace any refusal. Every rule the mode
// holds - at least one value, no more than the screen draws, a date this
// product reads - stays where it is, because a limit on what can be TYPED says
// nothing about a request arriving from the command line. This narrows one
// surface's field to what that surface can carry, which is the smallest claim
// that closes the gap the owner found.
//
// # Why it is silent
//
// A character that never appears is its own message, and it arrives at the
// moment the key is pressed rather than a sentence later. This is the one place
// in the product where silence is not the ban on silence being broken: nothing
// was refused, because nothing was asked for.
func Bounded(entry *widget.Entry, longest int, takes func(rune) bool) {
	previous := entry.OnChanged
	entry.OnChanged = func(typed string) {
		kept := bound(typed, longest, takes)
		if kept != typed {
			// 🔴 Guarded, because SetText calls OnChanged again and an
			// ungurded write here is an endless loop on the thread that draws
			// - which is to say a frozen window rather than a wrong value.
			entry.SetText(kept)
			return
		}
		if previous != nil {
			previous(typed)
		}
	}
}

// bound is what survives of the text, and it is separate so that a test can ask
// it directly rather than through a widget.
func bound(text string, longest int, takes func(rune) bool) string {
	var kept strings.Builder
	for _, letter := range text {
		if takes != nil && !takes(letter) {
			continue
		}
		// Counted in RUNES rather than in bytes, because a person pasting a
		// seed out of somebody else's log may bring anything with them, and a
		// limit counted in bytes would cut a character in half and leave the
		// field holding something no longer readable as text.
		if longest > 0 && len([]rune(kept.String())) >= longest {
			break
		}
		kept.WriteRune(letter)
	}
	return kept.String()
}

// Digits is the rule for a field holding a count: nothing but the ten digits.
//
// 🔴 A minus sign is NOT among them, and that is a decision rather than an
// omission. A count below one is refused by the mode, on both surfaces, with a
// sentence saying what would have been taken instead - so a person who types
// -5 on the command line is told why. Here the character simply never appears,
// which says the same thing sooner. The two answers agree because the mode
// still holds the rule, and what differs is only how early this surface stops.
func Digits(letter rune) bool { return letter >= '0' && letter <= '9' }

// DigitsAndDashes is the rule for a field holding a date written YYYY-MM-DD.
//
// The shape itself is the mode's DateShape and is refused there, so this takes
// the characters that shape is made of and nothing else. A person typing a
// month name gets no characters rather than a refusal three fields later.
func DigitsAndDashes(letter rune) bool { return Digits(letter) || letter == '-' }
