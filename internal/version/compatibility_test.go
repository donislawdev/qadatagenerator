package version_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/version"
)

// What a build can do with a version it was handed, case by case.
//
// 🔴 The case that made this function necessary is the third one below. Until
// 2026-08-13 a response record claiming version 99.0 was compared entry by
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
		{"a later minor version", "0.1", "0.9", version.WithinTheLine},
		{"an earlier minor version", "0.9", "0.1", version.WithinTheLine},
		{"a minor version with a patch part", "0.1", "0.1.4", version.WithinTheLine},
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
