package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
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
// This works on the file rather than on a run, because producing a manifest
// past the limit means a dataset wide enough to take minutes, and what is
// being guarded is the sentence rather than the arithmetic that reaches it.
func TestTheWarningAboutAManifestTooLargeToReadBackNamesBothCauses(t *testing.T) {
	dir := t.TempDir()
	oversized := filepath.Join(dir, "expectation-manifest.json")
	file, err := os.Create(oversized)
	if err != nil {
		t.Fatalf("cannot make the oversized manifest this test needs: %v", err)
	}
	if err := file.Truncate(manifestjson.MaxSize + 1); err != nil {
		t.Fatalf("cannot give the oversized manifest its length: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("cannot close the oversized manifest: %v", err)
	}

	lines := tooLargeToReadBack(dir, formatCSV)
	if len(lines) != 1 {
		t.Fatalf("a manifest one byte past the limit produced %d warnings, want 1: %v",
			len(lines), lines)
	}
	for _, cause := range []string{"record", "column"} {
		if !strings.Contains(lines[0], cause) {
			t.Errorf("the warning does not name %ss as something that makes a manifest grow: %q",
				cause, lines[0])
		}
	}
}

// A manifest inside the limit is not warned about, so that the warning means
// something when it does arrive.
func TestAManifestInsideTheLimitIsNotWarnedAbout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "expectation-manifest.json"),
		[]byte(`{"spec_version":"0.1"}`), 0o600); err != nil {
		t.Fatalf("cannot write the manifest this test needs: %v", err)
	}
	if lines := tooLargeToReadBack(dir, formatCSV); len(lines) != 0 {
		t.Fatalf("a manifest well inside the limit was warned about: %v", lines)
	}
}
