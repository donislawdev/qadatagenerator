package i18n

import (
	"strings"
	"testing"
)

// The register has to close from both sides.
//
// A key with no entry renders as its own name on screen. An entry no screen
// asks for is a sentence somebody translates into every language for nothing,
// and it is also how a key survives the removal of the control it belonged to.
func TestEveryKeyAndEveryEntryAnswerToEachOther(t *testing.T) {
	entries := map[Key]bool{}
	for _, name := range English.Names() {
		entries[name] = true
	}
	if len(entries) == 0 {
		t.Fatal("the English catalogue is empty, so every check here passes by checking nothing")
	}

	declared := map[Key]bool{}
	for _, key := range Keys {
		declared[key] = true
		if !entries[key] {
			t.Errorf("the constant %q has no entry, so a screen asking for it shows the key itself", key)
		}
	}
	for name := range entries {
		if !declared[name] {
			t.Errorf("the catalogue carries %q and no constant asks for it", name)
		}
	}
}

// The embedded file has to be readable, because the fallback for an unreadable
// one is a catalogue that answers with keys, and a window full of dotted
// identifiers is the failure this test exists to keep out of a release.
func TestTheEmbeddedCatalogueIsReadable(t *testing.T) {
	if English.Language() != "en" {
		t.Fatalf("the English catalogue reports language %q", English.Language())
	}
	if got := English.T(WindowTitle); got == string(WindowTitle) {
		t.Fatal("the title resolved to its own key, so the embedded file did not parse")
	}
}

// Text that reaches a person follows the punctuation rule of this repository.
// It is checked here rather than trusted, because the audit script reads Go
// and Markdown and this catalogue is neither.
func TestNoEntryUsesPunctuationThisRepositoryRefuses(t *testing.T) {
	semicolon := ";"             // punctuation: test data
	dashes := []string{"—", "–"} // punctuation: test data
	for _, name := range English.Names() {
		value := English.T(name)
		if strings.Contains(value, semicolon) {
			t.Errorf("%q uses a semicolon, and a full stop is what replaces it here: %q", name, value)
		}
		for _, dash := range dashes {
			if strings.Contains(value, dash) {
				t.Errorf("%q uses a dash this repository does not use: %q", name, value)
			}
		}
		if strings.TrimSpace(value) == "" {
			t.Errorf("%q has no text, so the control it labels is blank", name)
		}
	}
}

// A missing key names itself rather than rendering as nothing, because a
// control with no label reads as a layout fault and sends whoever sees it
// looking in the wrong place.
func TestAMissingKeyNamesItself(t *testing.T) {
	if got := English.T(Key("no.such.key")); got != "no.such.key" {
		t.Errorf("a missing key resolved to %q instead of naming itself", got)
	}
}

// One of a thing takes the first form and every other number the second,
// nought included: "0 modules" is English and "0 module" is not. The words come
// from the catalogue, so a missing key would show up here as its own name.
func TestACountTakesTheFormOfTheWordItsNumberAsksFor(t *testing.T) {
	for _, one := range []struct {
		many int
		want string
	}{
		{0, "0 modules"},
		{1, "1 module"},
		{36, "36 modules"},
	} {
		if got := English.Counted(one.many, AboutCountModule, AboutCountModules); got != one.want {
			t.Errorf("%d counted as %q, want %q", one.many, got, one.want)
		}
	}
}

func TestAnUnreadableCatalogueIsReportedRatherThanGuessed(t *testing.T) {
	if _, err := Parse("xx", []byte("this: [is not\n  valid yaml")); err == nil {
		t.Fatal("a broken catalogue parsed without complaint")
	}
}
