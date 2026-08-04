package cli

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var updateDataset = flag.Bool("update", false,
	"rewrite the frozen dataset after a deliberate change to what the tool produces")

// datasetDir is where the recipe and the frozen answer live.
func datasetDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("testdata", "dataset"))
	if err != nil {
		t.Fatalf("cannot find the frozen dataset: %v", err)
	}
	return path
}

// A whole dataset, frozen against the past.
//
// # Why this exists next to the tests that check the rules
//
// Every other test here asks whether the output follows a rule. All of them
// would stay green if every byte moved at once, because they check shape,
// agreement and arithmetic rather than identity. Somebody holding a dataset
// made with this seed was promised these exact bytes, and this is the only
// thing that notices when the promise breaks.
//
// # Why it goes through the verb rather than through the engine
//
// The engine already has its own tests. What is frozen here is what a person
// actually gets: the files, their names, the order of the columns, the key a
// child carries and the note beside the data. A freeze of the engine alone
// would leave the half of the run that assembles all of that unprotected.
//
// 🔴 A change here is a change to everybody's data. Rewriting the file with
// -update is a major version and a changelog entry, and it is only trustworthy
// after tools/wyrocznia_zestawu.py has recomputed the same bytes from the
// coordinates, in another language, without reading any of this.
func TestTheGeneratedDatasetHasNotMoved(t *testing.T) {
	frozen := datasetDir(t)
	made := t.TempDir()
	// The recipe writes into the working directory, so that nothing in the
	// frozen bytes depends on where the test happened to run. Naming the output
	// with a flag would put a temporary path into the note.
	t.Chdir(made)

	got := run(commandGenerate, filepath.Join(frozen, "recipe.yaml"))
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	produced := filesIn(t, made)
	if *updateDataset {
		writeFrozen(t, frozen, made, produced)
		return
	}
	compareFrozen(t, frozen, made, produced)
}

// filesIn returns the names of the files a run left behind, sorted so that the
// comparison below is the same on every system.
func filesIn(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read what the run produced: %v", err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names
}

// writeFrozen replaces the frozen answer with what the run just made.
func writeFrozen(t *testing.T, frozen, made string, produced []string) {
	t.Helper()
	expected := filepath.Join(frozen, "expected")
	if err := os.RemoveAll(expected); err != nil {
		t.Fatalf("cannot clear the frozen dataset: %v", err)
	}
	if err := os.MkdirAll(expected, 0o755); err != nil {
		t.Fatalf("cannot create the frozen dataset: %v", err)
	}
	for _, name := range produced {
		content, err := os.ReadFile(filepath.Join(made, name))
		if err != nil {
			t.Fatalf("cannot read %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(expected, name), content, 0o644); err != nil {
			t.Fatalf("cannot freeze %s: %v", name, err)
		}
	}
	t.Logf("the frozen dataset was rewritten with %d files. Run tools/wyrocznia_zestawu.py "+
		"before trusting it.", len(produced))
}

// compareFrozen checks the run against the frozen answer, in both directions.
func compareFrozen(t *testing.T, frozen, made string, produced []string) {
	t.Helper()
	expected := filepath.Join(frozen, "expected")
	stored := filesIn(t, expected)

	if strings.Join(stored, " ") != strings.Join(produced, " ") {
		t.Fatalf("the run no longer makes the same files.\nfrozen:   %v\nproduced: %v", stored, produced)
	}
	for _, name := range stored {
		want, err := os.ReadFile(filepath.Join(expected, name))
		if err != nil {
			t.Fatalf("cannot read the frozen %s: %v", name, err)
		}
		got, err := os.ReadFile(filepath.Join(made, name))
		if err != nil {
			t.Fatalf("cannot read the produced %s: %v", name, err)
		}
		if string(want) != string(got) {
			t.Errorf("%s has moved.\nEvery dataset made with this seed now differs from the one "+
				"somebody has on disk.\nIf that was not meant, this is a defect. If it was meant, it "+
				"is a major version and a changelog entry, and the file is rewritten with -update and "+
				"checked by tools/wyrocznia_zestawu.py.\n\nfrozen:\n%s\nproduced:\n%s",
				name, want, got)
		}
	}
}
