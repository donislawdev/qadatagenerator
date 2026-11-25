package produce

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/license"
)

// ranAt is the moment every run here says it happened at, so that two runs of
// one recipe can be compared byte for byte.
var ranAt = time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

// writeInto is a whole run of a recipe into a directory, started the way both
// surfaces start one.
func writeInto(rec recipe.Recipe, settings Settings) (Written, error) {
	prepared, err := Prepare(rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
	if err != nil {
		return Written{}, err
	}
	return prepared.Write(settings, generate.NeverStops{}, ranAt)
}

// damaged is the related recipe with a policy damaging every record of it, in
// the one layer this build carries out most of.
func damaged(children int) recipe.Recipe {
	rec := related(children)
	rec.Policy = recipe.Policy{Layers: []string{"L3"}, Rows: "100%", PerRow: "exactly_one"}
	return rec
}

// 🔴 The promise the whole screen rests on, held against a disk.
//
// Somebody looks at ten rows and presses a button. If what lands in the folder
// does not begin with those rows, every other guard in this package is about
// arithmetic nobody can see. The preview is the run's first rows by
// construction, and this is the sentence read back from the file the run wrote.
func TestWhatIsWrittenBeginsWithTheRowsThePreviewShowed(t *testing.T) {
	rec := related(60)
	source := values.New(rec.Locale, rec.Seed)

	sample, err := Preview(rec, source, 7, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the preview could not be made: %v", err)
	}

	dir := t.TempDir()
	written, err := writeInto(rec, Settings{Into: dir})
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

// A larger count adds rows at the end and leaves the first ones as they were,
// with the damage they carried, and a parent table that grows leaves the
// children of its first rows where they were. The guide to reproducible data
// on the website says so, because each value is drawn from its own place and
// not from what came before it.
//
// The canary asks the same comparison of a run with another seed and needs it
// to find a difference, so a comparison that reads nothing cannot pass.
func TestALargerCountLeavesTheFirstRowsAndTheirDamage(t *testing.T) {
	const table = `
version: "0.1"
seed: "%s"
locale: en-US
entities:
  customer:
    count: %d
    fields:
      first_name: { generator: first_name, required: true, max_length: 30 }
      last_name:  { generator: last_name, required: true, max_length: 30 }
      card:       { generator: card }
policy:
  layers: [L2, L3, REV]
  rows: 40%%
  per_row: exactly_one
`
	const relation = `
version: "0.1"
seed: "%s"
locale: en-US
entities:
  customer:
    count: %d
    primary_key: [customer_id]
    fields:
      customer_id: { type: integer, unique: true }
  order:
    belongs_to: { customer: 0-3 }
    fields:
      status: { values: [new, paid, shipped] }
`
	run := func(text string) string {
		dir := t.TempDir()
		if _, err := writeInto(decoded(t, text), Settings{Into: dir}); err != nil {
			t.Fatalf("the run failed: %v", err)
		}
		return dir
	}
	kept := func(fewer, more string, files ...string) []string {
		var moved []string
		for _, name := range files {
			small := readBack(t, filepath.Join(fewer, name))
			large := readBack(t, filepath.Join(more, name))
			if len(large) < len(small) {
				moved = append(moved, name+" has fewer rows in the larger run")
				continue
			}
			for at, row := range small {
				if strings.Join(row, "|") != strings.Join(large[at], "|") {
					moved = append(moved, fmt.Sprintf("%s line %d", name, at+1))
				}
			}
		}
		if smaller, larger := damageOf(t, fewer), damageOf(t, more); smaller != nil {
			rows := len(readBack(t, filepath.Join(fewer, files[0]))) - 1
			for key, value := range larger {
				if key.row <= rows && smaller[key] != value {
					moved = append(moved, fmt.Sprintf("%s in row %d, column %s", key.mutator, key.row, key.field))
				}
			}
			for key := range smaller {
				if _, ok := larger[key]; !ok {
					moved = append(moved, fmt.Sprintf("%s in row %d, column %s is gone", key.mutator, key.row, key.field))
				}
			}
		}
		return moved
	}

	fewer := run(fmt.Sprintf(table, "larger", 100))
	if moved := kept(fewer, run(fmt.Sprintf(table, "larger", 150)), "customer.csv"); len(moved) > 0 {
		t.Errorf("fifty more rows moved what the first hundred were: %v", moved)
	}
	if len(damageOf(t, fewer)) == 0 {
		t.Error("the run damaged nothing, so its damage was never compared")
	}
	if moved := kept(fewer, run(fmt.Sprintf(table, "another", 150)), "customer.csv"); len(moved) == 0 {
		t.Error("a run of another seed kept every row and its damage, so the comparison reads nothing")
	}

	parents := run(fmt.Sprintf(relation, "larger", 10))
	if moved := kept(parents, run(fmt.Sprintf(relation, "larger", 15)), "customer.csv", "order.csv"); len(moved) > 0 {
		t.Errorf("five more customers moved what the first ten and their orders were: %v", moved)
	}
}

// place is where a defect stands and which damage it is, since one cell can
// be ordered more than one.
type place struct {
	row     int
	field   string
	mutator string
}

// damageOf reads the manifest a run left, as whether each ordered defect was
// applied and what its value became, or nil when the run left no manifest.
func damageOf(t *testing.T, dir string) map[place]string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(dir, manifestjson.Name))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("cannot read the manifest: %v", err)
	}
	var read struct {
		Defects []struct {
			Location struct {
				Row   int    `json:"row"`
				Field string `json:"field"`
			} `json:"location"`
			Mutator string `json:"mutator"`
			Applied bool   `json:"applied"`
			After   string `json:"value_after"`
		} `json:"defects"`
	}
	if err := json.Unmarshal(content, &read); err != nil {
		t.Fatalf("the manifest cannot be read: %v", err)
	}
	found := make(map[place]string, len(read.Defects))
	for _, defect := range read.Defects {
		at := place{defect.Location.Row, defect.Location.Field, defect.Mutator}
		if _, twice := found[at]; twice {
			t.Fatalf("the manifest orders %s twice in row %d, column %s", at.mutator, at.row, at.field)
		}
		found[at] = fmt.Sprintf("%v %q", defect.Applied, defect.After)
	}
	return found
}

// A run that fails leaves nothing that looks like a dataset.
//
// The recipe asks for ten rows unique on a list of two values bound to the
// gender of the row. The plan leaves a field a rule binds to the run, so the
// run refuses it at the third row, after the engine has already written two.
// What matters here is what is on the disk afterwards, and it is asked twice:
// without a policy, and with one, where the manifest and its export have been
// opened beside the data before the failure.
//
// 🔴 Until 2026-10-27 the field was a gender asked to be unique, and the plan
// refuses that with the arithmetic since 2026-10-26, before a file is opened.
// Nothing was written, so there was nothing to leave behind, and the mutation
// that stops a failed run from throwing its files away survived on the commit
// before as well as after.
func TestARunThatFailsLeavesNothingBehind(t *testing.T) {
	for _, policy := range []recipe.Policy{
		{},
		{Layers: []string{"L3"}, Rows: "100%"},
	} {
		rec := recipe.Recipe{
			Version: "0.1", Seed: "sample", Locale: "pl-PL",
			Entities: []recipe.Entity{{
				Name: "customer", Count: 10, HasCount: true,
				Fields: []recipe.Field{{Name: "sex", Generator: "gender"},
					{Name: "code", Values: []string{"p", "q"}, Unique: true}},
				Consistency: []recipe.Rule{{Field: "code", Kind: recipe.Matches, With: []string{"sex"}}},
			}},
			Policy: policy,
		}
		dir := t.TempDir()

		if _, err := writeInto(rec, Settings{Into: dir, Manifest: "csv"}); err == nil {
			t.Fatal("a recipe asking for ten unique values out of two produced a dataset")
		}

		left, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("the directory cannot be read: %v", err)
		}
		for _, entry := range left {
			t.Errorf("a failed run with policy %v left %q behind", policy.Layers, entry.Name())
		}
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

	if _, err := writeInto(rec, Settings{Into: dir}); err != nil {
		t.Fatalf("the first run failed: %v", err)
	}
	if _, err := writeInto(rec, Settings{Into: dir}); err == nil {
		t.Error("a second run replaced the first one without being told to")
	}
	if _, err := writeInto(rec, Settings{Into: dir, Replace: true}); err != nil {
		t.Errorf("a run told to replace files refused anyway: %v", err)
	}
}

// The names come from the writer rather than from the recipe, so a form that
// writes one file for the whole dataset is reported as one file, with the note
// beside it.
func TestTheNamesReportedAreTheNamesTheWriterAskedFor(t *testing.T) {
	rec := related(3)
	rec.Output.Format = "json"
	dir := t.TempDir()

	written, err := writeInto(rec, Settings{Into: dir})
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if len(written.Files) != 2 || !strings.HasSuffix(written.Files[0], ".json") ||
		written.Files[1] != RunNoteName {
		t.Errorf("the object form writes one file for the whole dataset and the note beside "+
			"it, and this run reported %v", written.Files)
	}
	for _, name := range written.Files {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("the run reported %q and it is not there: %v", name, err)
		}
	}
}

// 🔴 A recipe with a policy is damaged on its way to the disk, with a manifest
// and the export asked for beside the data.
//
// This is B-300 held against a disk. The recipe screen wrote such a recipe
// clean, with no manifest, while the command line wrote it damaged, because the
// policy was carried out only by the command line. The run both surfaces call
// is now this one, so the question is asked of it rather than of either of them.
func TestARecipeWithAPolicyIsDamagedOnTheWayAndLeavesAManifest(t *testing.T) {
	dir := t.TempDir()
	written, err := writeInto(damaged(3), Settings{Into: dir, Manifest: "csv"})
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}

	if !written.Damaged {
		t.Fatal("a recipe with a policy ran as if it had none")
	}
	if written.Counts.Requested == 0 || written.Counts.Applied == 0 {
		t.Errorf("every record was ordered damage in a layer this build carries out, and "+
			"the run counted %+v", written.Counts)
	}
	for _, name := range damage.Files("csv") {
		if !slices.Contains(written.Files, name) {
			t.Errorf("the run did not report %s among %v", name, written.Files)
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("the run did not leave %s behind: %v", name, err)
		}
	}

	clean := t.TempDir()
	if _, err := writeInto(related(3), Settings{Into: clean}); err != nil {
		t.Fatalf("the clean run failed: %v", err)
	}
	if same(t, filepath.Join(dir, "customer.csv"), filepath.Join(clean, "customer.csv")) {
		t.Error("every customer was ordered damage and the file is the one a clean run writes")
	}
}

// A run of a recipe without a policy writes no manifest, whatever form was
// asked for, and FormUnused is what lets a surface say so.
func TestAnExtraFormWithoutAPolicyWritesNothingAndSaysSo(t *testing.T) {
	dir := t.TempDir()
	settings := Settings{Into: dir, Manifest: "csv"}
	written, err := writeInto(related(3), settings)
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if written.Damaged {
		t.Error("a recipe without a policy reported damage")
	}
	if _, err := os.Stat(filepath.Join(dir, manifestjson.Name)); err == nil {
		t.Error("a recipe without a policy left a manifest")
	}
	if !settings.FormUnused(related(3)) {
		t.Error("an extra form on a run with nothing to export is not reported as unused")
	}
	if settings.FormUnused(damaged(3)) {
		t.Error("an extra form on a run with a policy is reported as unused")
	}
	if (Settings{}).FormUnused(related(3)) {
		t.Error("a run that asked for no extra form is reported as leaving one unused")
	}
}

// 🔴 The note beside the data is written by the run both surfaces call, so the
// window leaves it too (B-209).
//
// Its last field is the reason it matters: a folder of national identifiers
// has to say from the folder alone that the values describe nobody.
func TestTheNoteBesideTheDataSaysTheValuesAreSynthetic(t *testing.T) {
	dir := t.TempDir()
	applied := []Override{{Setting: "seed", FromRecipe: "a", FromFlag: "b"}}
	if _, err := writeInto(related(3), Settings{Into: dir, Overrides: applied}); err != nil {
		t.Fatalf("the run failed: %v", err)
	}

	read, err := os.ReadFile(filepath.Join(dir, RunNoteName))
	if err != nil {
		t.Fatalf("the run left no note: %v", err)
	}
	var note RunNote
	if err := json.Unmarshal(read, &note); err != nil {
		t.Fatalf("the note is not a document a pipeline can read: %v", err)
	}
	if note.SyntheticData != license.SyntheticDataNotice {
		t.Error("the note does not carry the sentence saying every value is synthetic")
	}
	if len(note.Tables) != 2 || note.Tables[1].Parent != "customer" {
		t.Errorf("the note does not describe the tables and their relation: %+v", note.Tables)
	}
	if len(note.Overrides) != 1 || note.Overrides[0] != applied[0] {
		t.Errorf("the note carries overrides %+v rather than the one handed in", note.Overrides)
	}
}

// readNote is the record a run left beside its data.
func readNote(t *testing.T, dir string) RunNote {
	t.Helper()
	read, err := os.ReadFile(filepath.Join(dir, RunNoteName))
	if err != nil {
		t.Fatalf("the run left no note: %v", err)
	}
	var note RunNote
	if err := json.Unmarshal(read, &note); err != nil {
		t.Fatalf("the note is not a document a pipeline can read: %v", err)
	}
	return note
}

// decoded is a recipe read the way every surface reads one.
func decoded(t *testing.T, text string) recipe.Recipe {
	t.Helper()
	result := recipeyaml.Decode([]byte(text))
	if !result.Valid() {
		t.Fatalf("the recipe was refused: %+v", result.Problems)
	}
	return result.Recipe
}

// A recipe that names no seed and asks for damage is damaged, and the record
// beside the data names the seed and the locale the run used.
//
// Reproduced by an outside review on 2026-10-22: validate called such a recipe
// usable and generate ended as an internal defect after producing every row,
// because the manifest envelope refused an empty seed.
func TestARecipeWithoutASeedIsDamagedWithTheSeedItIsReadWith(t *testing.T) {
	rec := decoded(t, `
version: "0.1"
entities:
  t:
    count: 2
    fields:
      n: { type: integer }
policy:
  layers: [L2]
  rows: 100%
`)
	dir := t.TempDir()
	if _, err := writeInto(rec, Settings{Into: dir}); err != nil {
		t.Fatalf("a recipe without a seed could not be damaged: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, manifestjson.Name)); err != nil {
		t.Errorf("the run left no manifest: %v", err)
	}
	note := readNote(t, dir)
	if note.Seed != recipe.DefaultSeed || note.Locale != recipe.DefaultLocale {
		t.Errorf("the note says seed %q and locale %q, and the run drew with %q and %q",
			note.Seed, note.Locale, recipe.DefaultSeed, recipe.DefaultLocale)
	}
}

// A setting the reader carried without reading is in the record beside the
// data, not only in what validate printed.
//
// Reproduced by an outside review on 2026-10-22: a date of birth limited to one
// year got dates from other decades, validate warned about the two settings,
// and the note said nothing was left undone.
func TestTheNoteNamesWhatTheReaderCarriedWithoutReading(t *testing.T) {
	rec := decoded(t, `
version: "0.1"
seed: "review"
entities:
  t:
    count: 2
    fields:
      born: { generator: birth_date, min: "2000-01-01", max: "2000-12-31" }
`)
	dir := t.TempDir()
	if _, err := writeInto(rec, Settings{Into: dir}); err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	var said []string
	for _, entry := range readNote(t, dir).NotDone {
		said = append(said, entry.Where)
	}
	for _, where := range []string{"entities.t.fields.born.max", "entities.t.fields.born.min"} {
		if !slices.Contains(said, where) {
			t.Errorf("the note leaves out %s, and lists %v", where, said)
		}
	}
}

// Moving the directory records both sides and leaves the recipe pointing where
// it was moved, which is what the settings of a run promise of the recipe handed
// in: it already carries the values that won.
func TestMovingTheDirectoryRecordsBothSidesAndMovesTheRecipe(t *testing.T) {
	rec := related(1)
	rec.Output.Dir = "from-the-recipe"

	got := MoveDir(&rec, "from-outside")

	want := Override{Setting: "output.dir", FromRecipe: "from-the-recipe", FromFlag: "from-outside"}
	if got != want {
		t.Errorf("the record of the move is %+v rather than %+v", got, want)
	}
	if rec.Output.Dir != "from-outside" {
		t.Errorf("the recipe still points at %q after the move", rec.Output.Dir)
	}
}

// 🔴 A setting the form asked for ignores is noted, and only when the recipe
// wrote it.
//
// The note about file_per_entity was said whenever the setting was false, and
// false is also what a recipe that never mentions it carries, so every comma
// separated recipe was told about a line nobody had written (B-301). A
// delimiter under the object form was ignored without a word until 2026-11-24.
// Both halves are held here for both settings: the note that is owed and the
// note that is not.
func TestASettingNobodyWroteIsNotReportedAsChangingNothing(t *testing.T) {
	for _, given := range []struct {
		what    string
		setting string
		output  recipe.Output
		noted   bool
	}{
		{"not written at all", "output.file_per_entity", recipe.Output{}, false},
		{"written false for the comma separated form", "output.file_per_entity",
			recipe.Output{FilePerEntityGiven: true}, true},
		{"written false for the form that reads it", "output.file_per_entity",
			recipe.Output{Format: "json", FilePerEntityGiven: true}, false},
		{"written true", "output.file_per_entity",
			recipe.Output{FilePerEntity: true, FilePerEntityGiven: true}, false},
		{"a delimiter for the form that reads it", "output.delimiter",
			recipe.Output{Delimiter: ";"}, false}, // punctuation: test data
		{"a delimiter for the object form", "output.delimiter",
			recipe.Output{Format: "json", Delimiter: ";"}, true}, // punctuation: test data
		{"no delimiter for the object form", "output.delimiter",
			recipe.Output{Format: "json"}, false},
		{"tables asked of the comma separated form", "output.create_tables",
			recipe.Output{CreateTables: true}, true},
		{"tables asked of the form that creates them", "output.create_tables",
			recipe.Output{Format: "sql", Dialect: "sqlite", CreateTables: true}, false},
		{"no tables asked of the object form", "output.create_tables",
			recipe.Output{Format: "json"}, false},
	} {
		rec := related(3)
		rec.Output = given.output
		prepared, err := Prepare(rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
		if err != nil {
			t.Fatalf("%s: the plan was refused: %v", given.what, err)
		}
		noted := false
		for _, note := range prepared.Plan().Notes {
			if note.Where == given.setting {
				noted = true
			}
		}
		if noted != given.noted {
			t.Errorf("%s %s: noted is %v, want %v", given.setting, given.what, noted, given.noted)
		}
	}
}

// same reports whether two files hold the same bytes.
func same(t *testing.T, left, right string) bool {
	t.Helper()
	one, err := os.ReadFile(left)
	if err != nil {
		t.Fatalf("cannot read %s: %v", left, err)
	}
	two, err := os.ReadFile(right)
	if err != nil {
		t.Fatalf("cannot read %s: %v", right, err)
	}
	return string(one) == string(two)
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
