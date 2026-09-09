package parts_test

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// A field that takes anything takes a file, and that is how the owner found
// this on 2026-09-09: thirty seven digits went into a field asking how many
// values to produce.
func TestABoundedFieldStopsAtTheLengthItWasGiven(t *testing.T) {
	field := widget.NewEntry()
	parts.Bounded(field, 5, parts.Digits)

	field.SetText("1234567890")

	if field.Text != "12345" {
		t.Errorf("a field bounded at five characters holds %q, want %q", field.Text, "12345")
	}
}

// The characters that could never be part of the answer never appear, which
// says what is taken sooner than any sentence could.
func TestABoundedFieldKeepsOnlyTheCharactersItTakes(t *testing.T) {
	field := widget.NewEntry()
	parts.Bounded(field, 0, parts.Digits)

	field.SetText("1a2-3 4")

	if field.Text != "1234" {
		t.Errorf("a field taking digits holds %q, want %q", field.Text, "1234")
	}
}

// 🔴 The guard that would have caught the loop rather than the value.
//
// SetText inside OnChanged calls OnChanged again. Without the guard in Bounded
// this is not a wrong value but a window that stops answering, on the thread
// that draws - and a test asserting only the text would pass right up until the
// moment it hangs forever.
func TestABoundedFieldSettlesInsteadOfCallingItselfForever(t *testing.T) {
	field := widget.NewEntry()
	calls := 0
	field.OnChanged = func(string) { calls++ }
	parts.Bounded(field, 3, parts.Digits)

	field.SetText("123456789")

	if field.Text != "123" {
		t.Fatalf("the field holds %q, want %q", field.Text, "123")
	}
	// One call for the text that was already within bounds. More than a
	// handful means each correction is provoking another round.
	if calls > 2 {
		t.Errorf("the callback ran %d times for one write, so the correction is feeding itself", calls)
	}
}

// Text already inside the bounds reaches whatever the screen wired up first,
// because Bounded narrows a field rather than taking it over.
func TestABoundedFieldStillTellsTheScreenWhatWasTyped(t *testing.T) {
	field := widget.NewEntry()
	heard := ""
	field.OnChanged = func(typed string) { heard = typed }
	parts.Bounded(field, 10, parts.Digits)

	field.SetText("42")

	if heard != "42" {
		t.Errorf("the screen heard %q, want %q", heard, "42")
	}
}

// 🔴 Counted in runes, not bytes.
//
// A seed is pasted out of somebody's log and may carry anything. A limit
// counted in bytes cuts a multi byte character in half, and half a character is
// not text: the field would then hold something that cannot be read back or
// typed again, which is the one thing a seed has to survive.
func TestABoundedFieldCountsCharactersRatherThanBytes(t *testing.T) {
	field := widget.NewEntry()
	parts.Bounded(field, 4, nil)

	field.SetText("łódka")

	if field.Text != "łódk" {
		t.Errorf("a field bounded at four characters holds %q, want %q", field.Text, "łódk")
	}
	if strings.ContainsRune(field.Text, '�') {
		t.Error("the field holds a replacement character, so a character was cut in half")
	}
}

// A minus sign is not a digit here, and the reason is written beside Digits:
// the mode still refuses a count below one, in words, on both surfaces.
func TestTheDigitRuleTakesNoSign(t *testing.T) {
	for _, letter := range []rune{'-', '+', '.', ',', ' '} {
		if parts.Digits(letter) {
			t.Errorf("the digit rule takes %q, and a count is written in digits alone", letter)
		}
	}
	for letter := '0'; letter <= '9'; letter++ {
		if !parts.Digits(letter) {
			t.Errorf("the digit rule refuses %q", letter)
		}
	}
}

// A date is written YYYY-MM-DD, so the field takes what that shape is made of
// and nothing else. Whether the digits form a date is still the mode's answer.
func TestTheDateRuleTakesDigitsAndDashesOnly(t *testing.T) {
	if !parts.DigitsAndDashes('-') || !parts.DigitsAndDashes('7') {
		t.Error("the date rule refuses a character the shape YYYY-MM-DD is made of")
	}
	for _, letter := range []rune{'/', 'a', ' ', ':'} {
		if parts.DigitsAndDashes(letter) {
			t.Errorf("the date rule takes %q, which the shape YYYY-MM-DD does not contain", letter)
		}
	}
}
