package produce

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// 🔴 The promise the whole screen rests on, held against a disk.
//
// Somebody looks at ten rows and presses a button. If what lands in the folder
// does not begin with those rows, every other guard in this package is about
// arithmetic nobody can see. The preview is the run's first rows by
// construction, and this is the sentence read back from the file the run wrote.
func TestWhatIsWrittenBeginsWithTheRowsThePreviewShowed(t *testing.T) {
	rec := related(60)
	source := values.New(rec.Locale, rec.Seed)

	sample, err := Preview(rec, source, 7)
	if err != nil {
		t.Fatalf("the preview could not be made: %v", err)
	}

	dir := t.TempDir()
	written, err := Write(rec, values.New(rec.Locale, rec.Seed), dir, false, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if written.Rows != 60+40 {
		t.Errorf("this recipe has forty customers and sixty orders, and the run counted %d rows",
			written.Rows)
	}

	for _, table := range sample.Tables {
		lines := readBack(t, filepath.Join(dir, table.Entity+".csv"))
		if strings.Join(lines[0], "|") != strings.Join(table.Columns, "|") {
			t.Errorf("%s: the preview shows columns %v and the file has %v",
				table.Entity, table.Columns, lines[0])
		}
		for at, row := range table.Rows {
			if strings.Join(row, "|") != strings.Join(lines[at+1], "|") {
				t.Errorf("%s row %d: the preview showed %v and the file holds %v",
					table.Entity, at, row, lines[at+1])
			}
		}
	}
}

// A run that fails leaves nothing that looks like a dataset.
//
// The recipe asks for ten rows unique on a field with two possible values,
// which is refused with the arithmetic after the engine has already written
// part of the table. What matters here is what is on the disk afterwards.
func TestARunThatFailsLeavesNothingBehind(t *testing.T) {
	rec := recipe.Recipe{
		Version: "0.1", Seed: "sample", Locale: "pl-PL",
		Entities: []recipe.Entity{{
			Name: "customer", Count: 10, HasCount: true,
			Fields: []recipe.Field{{Name: "gender", Generator: "gender", Unique: true}},
		}},
	}
	dir := t.TempDir()

	if _, err := Write(rec, values.New(rec.Locale, rec.Seed), dir, false,
		generate.NeverStops{}); err == nil {
		t.Fatal("a recipe asking for ten unique values out of two produced a dataset")
	}

	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("the directory cannot be read: %v", err)
	}
	for _, entry := range left {
		t.Errorf("a failed run left %q behind", entry.Name())
	}
}

// Replacing somebody's files takes saying so.
//
// The second run refuses rather than overwriting, and says so, which is the
// same refusal the command line makes and for the same reason: of the two
// things colliding, only one of them is ours.
func TestASecondRunRefusesToReplaceFilesUnlessItIsTold(t *testing.T) {
	rec := related(3)
	dir := t.TempDir()

	if _, err := Write(rec, values.New(rec.Locale, rec.Seed), dir, false,
		generate.NeverStops{}); err != nil {
		t.Fatalf("the first run failed: %v", err)
	}
	if _, err := Write(rec, values.New(rec.Locale, rec.Seed), dir, false,
		generate.NeverStops{}); err == nil {
		t.Error("a second run replaced the first one without being told to")
	}
	if _, err := Write(rec, values.New(rec.Locale, rec.Seed), dir, true,
		generate.NeverStops{}); err != nil {
		t.Errorf("a run told to replace files refused anyway: %v", err)
	}
}

// The names come from the writer rather than from the recipe, so a form that
// writes one file for the whole dataset is reported as one file.
func TestTheNamesReportedAreTheNamesTheWriterAskedFor(t *testing.T) {
	rec := related(3)
	rec.Output.Format = "json"
	dir := t.TempDir()

	written, err := Write(rec, values.New(rec.Locale, rec.Seed), dir, false,
		generate.NeverStops{})
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if len(written.Files) != 1 || !strings.HasSuffix(written.Files[0], ".json") {
		t.Errorf("the object form writes one file for the whole dataset, and this run "+
			"reported %v", written.Files)
	}
	for _, name := range written.Files {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("the run reported %q and it is not there: %v", name, err)
		}
	}
}

func readBack(t *testing.T, path string) [][]string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the run did not write %s: %v", path, err)
	}
	reader := csv.NewReader(strings.NewReader(string(content)))
	reader.FieldsPerRecord = -1
	lines, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("%s cannot be read back: %v", path, err)
	}
	return lines
}
