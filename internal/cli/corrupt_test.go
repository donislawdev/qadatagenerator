package cli

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/damage"
)

// The warning about a manifest too large to read back names both of the things
// that make one grow.
//
// 🔴 It named one of them until 2026-09-03, and not the one that gets there
// first. Measured that day: a file of 200 000 columns and three rows produced a
// manifest of 38.6 MB carrying no entries at all, because a manifest describes
// every column whether anything was damaged in it or not. A reader told to
// damage fewer rows would have changed the one number that was already zero.
//
// This works on the sentence rather than on a run, because producing a manifest
// past the limit means a dataset wide enough to take minutes, and what is being
// guarded is the sentence rather than the arithmetic that reaches it. Which
// files reach it is guarded where the size is measured, in internal/damage.
func TestTheWarningAboutAManifestTooLargeToReadBackNamesBothCauses(t *testing.T) {
	line := tooLargeSentence(manifestjson.Name, damage.ReadBackLimit+1)
	for _, cause := range []string{"record", "column"} {
		if !strings.Contains(line, cause) {
			t.Errorf("the warning does not name %ss as something that makes a manifest grow: %q",
				cause, line)
		}
	}
	if !strings.Contains(line, manifestjson.Name) {
		t.Errorf("the warning does not name the file it is about: %q", line)
	}
}
