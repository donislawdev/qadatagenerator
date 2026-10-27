package version_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/version"
)

// What a build can do with a version it was handed, case by case.
//
// 🔴 The case that made this function necessary is the third one below. Until
// 2026-08-17 a response record claiming version 99.0 was compared entry by
// entry and the run ended saying the application had behaved as promised. The
// field was checked for being present and never for what it said.
func TestWhatThisBuildCanDoWithAVersion(t *testing.T) {
	for _, given := range []struct {
		name    string
		speaks  string
		claimed string
		want    version.Reading
	}{
		{"the version this build writes", "0.1", "0.1", version.AsWritten},
		{"a later minor version", "0.1", "0.9", version.Later},
		{"an earlier minor version", "0.9", "0.1", version.Earlier},
		{"a minor version with a patch part", "0.1", "0.1.4", version.Later},
		{"an earlier minor version beside a patch part", "0.2", "0.1.4", version.Earlier},
		// Numbers, not text. As text 0.10 sorts before 0.9, and a build
		// speaking 0.9 would have read a newer record as an older one and said
		// nothing about what it left unread.
		{"a minor version of two digits", "0.9", "0.10", version.Later},
		{"an earlier minor version of one digit", "0.10", "0.9", version.Earlier},
		{"a trailing nought", "0.1", "0.1.0", version.AsWritten},
		{"a minor part that is a word", "0.2", "0.x", version.Later},
		{"a minor part with a leading nought", "0.2", "0.01", version.Later},
		{"a later major version", "0.1", "99.0", version.Unreadable},
		{"an earlier major version", "2.0", "1.9", version.Unreadable},
		{"a word", "0.1", "banana", version.Unreadable},
		{"a number with a word stuck to it", "0.1", "0beta.1", version.Unreadable},
		{"spaces around the version", "0.1", "  0.1  ", version.AsWritten},
		{"a version that is only spaces", "0.1", "   ", version.Unreadable},
		{"nothing at all", "0.1", "", version.Unreadable},
		// The recipe reader refuses a leading nought in a count, and a version
		// gets the same answer for the same reason: 01 and 1 would be two
		// spellings of one thing with nothing to say which was meant.
		{"a major version written with a leading nought", "1.0", "01.0", version.Unreadable},
	} {
		t.Run(given.name, func(t *testing.T) {
			if got := version.Compatibility(given.speaks, given.claimed); got != given.want {
				t.Errorf("a build speaking %q handed %q: got %v, wanted %v",
					given.speaks, given.claimed, got, given.want)
			}
		})
	}
}

// The formats this build reads name themselves, so nothing has to be typed
// twice at the places that check a document.
func TestTheReadableFormatsCarryTheirVersion(t *testing.T) {
	for name, spoken := range map[string]string{
		"manifest": version.Manifest,
		"response": version.Response,
	} {
		if version.Compatibility(spoken, spoken) != version.AsWritten {
			t.Errorf("this build does not recognise its own %s version %q", name, spoken)
		}
	}
}
