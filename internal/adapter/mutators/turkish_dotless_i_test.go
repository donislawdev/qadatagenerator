package mutators_test

import (
	"strings"
	"testing"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

// The two Turkish letters, written as escapes because each is one dot away from
// the letter it replaces.
const (
	dotlessSmallI  = "\U00000131"
	dottedCapitalI = "\U00000130"
)

// Every expected value is written out code point by code point. The swap is a
// table of two letters, and what the tests have to hold is where it applies:
// every plain i and I, and no letter that carries an accent, whichever way the
// accent is written.
func TestEveryIBecomesTheOtherTurkishLetterOfItsCase(t *testing.T) {
	for _, each := range []struct{ name, before, after string }{
		{"a name with both", "Irina", dottedCapitalI + "r" + dotlessSmallI + "na"},
		{"small letters only", "Kinga", "K" + dotlessSmallI + "nga"},
		{"capitals only", "INFO", dottedCapitalI + "NFO"},
		{"two words", "Iwona Kowalik", dottedCapitalI + "wona Kowal" + dotlessSmallI + "k"},
		{"an accented capital written as one character", "\U000000CDsis", "\U000000CDs" + dotlessSmallI + "s"},
		{"an accent written after the letter", "I\U00000301ri", "I\U00000301r" + dotlessSmallI},
		{"a Turkish letter already there", "I" + dotlessSmallI + "k", dottedCapitalI + dotlessSmallI + "k"},
		{"bytes that are not text", "i\xffI", dotlessSmallI + "\xff" + dottedCapitalI},
	} {
		t.Run(each.name, func(t *testing.T) {
			outcome := withDefaults(t, turkishI, each.before)
			if !outcome.Applied() {
				t.Fatalf("%+q was left alone: %s", each.before, outcome.Reason())
			}
			if outcome.Value() != each.after {
				t.Errorf("%+q became %+q, want %+q", each.before, outcome.Value(), each.after)
			}
		})
	}
}

// A letter with an accent is left alone whether the accent is part of one
// character or follows the letter, so a value and its decomposed form come out
// as the two forms of one result. The normalisation library is an oracle here:
// the mutator does not use it. A decomposed İ is an I followed by a combining
// dot, and swapping that I would give a capital with two dots.
func TestTheTwoCanonicalFormsOfAValueComeOutAsOneResult(t *testing.T) {
	for _, value := range []string{
		dottedCapitalI + "zmir", "\U000000CDtalo Iris", "Ren\U000000E9e Ibi\U000000E9", "\U000000EFnis",
		dottedCapitalI + "stanbul",
	} {
		composed := withDefaults(t, turkishI, norm.NFC.String(value))
		decomposed := withDefaults(t, turkishI, norm.NFD.String(value))
		if composed.Applied() != decomposed.Applied() {
			t.Errorf("%+q was damaged in one form and refused in the other: %q and %q",
				value, composed.Reason(), decomposed.Reason())
			continue
		}
		if !composed.Applied() {
			continue
		}
		if back := norm.NFC.String(decomposed.Value()); back != composed.Value() {
			t.Errorf("%+q gives %+q composed and %+q decomposed", value, composed.Value(), back)
		}
	}
}

// 🔴 A value with nothing to swap is unfit rather than skipped, so the engine can
// try another text column of the row. A code, a date and a name without an i
// are all like that, and so is a value whose only i carries an accent.
func TestAValueWithNoPlainIIsUnfit(t *testing.T) {
	for _, value := range []string{
		"", "Ewa", "Ola 123", "1990-04-11", dottedCapitalI + dotlessSmallI, "\U000000CD", "I\U00000301",
		dottedCapitalI + "stanbul",
	} {
		outcome := withDefaults(t, turkishI, value)
		if outcome.Applied() {
			t.Errorf("%+q was damaged into %+q, and it has no plain i or I", value, outcome.Value())
			continue
		}
		if !outcome.ValueUnfit() {
			t.Errorf("%+q was refused for a reason that does not move the order on: %s", value, outcome.Reason())
		}
		if !strings.Contains(outcome.Reason(), "no letter i or I") {
			t.Errorf("%+q was refused with %q, which does not name the letter it looked for", value, outcome.Reason())
		}
	}
}

// The catalog note and the corpus say how the two letters change case, and
// those sentences are a measurement of the Unicode library rather than of this
// build. Held here, a library that started converting them another way would
// fail this test instead of leaving the note describing a behaviour nobody has.
func TestTheTwoLettersChangeCaseAsTheCatalogNoteSays(t *testing.T) {
	for _, tag := range []language.Tag{
		language.Und, language.Turkish, language.Azerbaijani, language.Lithuanian,
		language.Polish, language.German, language.Greek,
	} {
		if got := cases.Upper(tag).String(dotlessSmallI); got != "I" {
			t.Errorf("upper casing the dotless i in %s gives %+q, not a plain I", tag, got)
		}
	}
	if got := strings.ToUpper(dotlessSmallI); got != "I" {
		t.Errorf("the standard library upper cases the dotless i to %+q, not a plain I", got)
	}
	if got := cases.Lower(language.Und).String("I"); got != "i" {
		t.Errorf("lower casing I outside a Turkish locale gives %+q", got)
	}
	for _, each := range []struct {
		where string
		lower func(string) string
		want  string
	}{
		{"Turkish", cases.Lower(language.Turkish).String, "i"},
		{"Azerbaijani", cases.Lower(language.Azerbaijani).String, "i"},
		{"the standard library", strings.ToLower, "i"},
		{"no locale", cases.Lower(language.Und).String, "i\U00000307"},
		{"Polish", cases.Lower(language.Polish).String, "i\U00000307"},
		{"Lithuanian", cases.Lower(language.Lithuanian).String, "i\U00000307\U00000307"},
	} {
		if got := each.lower(dottedCapitalI); got != each.want {
			t.Errorf("lower casing the dotted capital I in %s gives %+q, want %+q", each.where, got, each.want)
		}
	}
}
