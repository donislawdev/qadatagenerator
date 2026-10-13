package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// recipeAt writes a recipe into a directory and returns its path.
func recipeIn(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "recipe.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("cannot write the recipe: %v", err)
	}
	return path
}

// smallRecipe is a recipe with two entities, a relation and a rule binding an
// identifier to the columns it reads.
func smallRecipe(out string) string {
	return `version: "0.1"
seed: "test-seed"
locale: pl-PL
entities:
  customer:
    count: 3
    primary_key: [pesel]
    fields:
      first_name: { generator: first_name }
      gender:     { generator: gender }
      birth_date: { generator: birth_date }
      pesel:      { generator: pesel }
    consistency:
      - pesel matches: [birth_date, gender]
  order:
    count: 4
    belongs_to: { customer: 0-3 }
    fields:
      placed_at: { generator: birth_date }
output:
  format: csv
  dir: ` + strings.ReplaceAll(out, `\`, `/`) + `
  file_per_entity: true
`
}

// The verb exists and produces what the recipe asked for.
func TestGenerateWritesTheDatasetAndTheNoteBesideIt(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	got := run(commandGenerate, recipeIn(t, dir, smallRecipe(out)))

	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}
	if got.data != "" {
		t.Errorf("counts went to the data channel, which is for the dataset: %q", got.data)
	}
	for _, name := range []string{"customer.csv", "order.csv", produce.RunNoteName} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("no %s was written: %v", name, err)
		}
	}
}

// 🔴 The dataset has to say about itself that it describes nobody, because a
// README does not travel with a folder of records.
func TestTheNoteSaysTheValuesAreSynthetic(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	if got := run(commandGenerate, recipeIn(t, dir, smallRecipe(out))); got.code != 0 {
		t.Fatalf("the run ended with %d: %s", got.code, got.log)
	}

	read, err := os.ReadFile(filepath.Join(out, produce.RunNoteName))
	if err != nil {
		t.Fatalf("cannot read the note: %v", err)
	}
	var note produce.RunNote
	if err := json.Unmarshal(read, &note); err != nil {
		t.Fatalf("the note is not a document a pipeline can read: %v", err)
	}
	if note.SyntheticData != license.SyntheticDataNotice {
		t.Error("the note does not carry the sentence saying every value is synthetic")
	}
	for _, version := range []string{note.Tool, note.Catalog, note.Manifest, note.RecipeVersion} {
		if version == "" {
			t.Error("the note leaves out one of the four versions, which is the only trail back " +
				"to what changed when the same recipe gives a different answer")
		}
	}
	if len(note.Tables) != 2 || note.Tables[1].Parent != "customer" {
		t.Errorf("the note does not describe the tables and their relation: %+v", note.Tables)
	}
}

// A setting the command line overrode is named in the note beside the data.
//
// # Why this is not a detail
//
// 🔴 The note exists to answer one question: why two runs of one recipe gave
// different data. A seed from a flag is the commonest reason for that, and the
// note recorded the overridden value while saying nothing about where it came
// from, so the field that answers the question was empty in exactly the runs
// that raise it. The log said it and the log stays in the pipeline while the
// data travels, which is the whole reason this file exists.
//
// The second half asks for the empty list on a run with no overrides, because a
// null where a list was is a change in the document for whoever parses it.
func TestTheNoteSaysWhichSettingsTheCommandLineOverrode(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	path := recipeIn(t, dir, smallRecipe(out))

	if got := run(commandGenerate, "--seed", "OVERRIDDEN", path); got.code != 0 {
		t.Fatalf("the run ended with %d: %s", got.code, got.log)
	}
	note := noteIn(t, out)
	if len(note.Overrides) != 1 {
		t.Fatalf("one setting came from the command line and the note carries %+v", note.Overrides)
	}
	if note.Overrides[0].Setting != "seed" {
		t.Errorf("the note names %q as the overridden setting", note.Overrides[0].Setting)
	}
	if note.Overrides[0].FromFlag != "OVERRIDDEN" {
		t.Errorf("the note does not carry the value from the flag: %+v", note.Overrides[0])
	}
	if note.Overrides[0].FromRecipe == "" {
		t.Error("the note does not say what the recipe asked for, which is half the answer")
	}
	if note.Seed != "OVERRIDDEN" {
		t.Errorf("the note records the seed as %q rather than the one the run used", note.Seed)
	}

	plain := filepath.Join(dir, "plain")
	if got := run(commandGenerate, recipeIn(t, dir, smallRecipe(plain))); got.code != 0 {
		t.Fatalf("the run with nothing overridden ended with %d: %s", got.code, got.log)
	}
	read, err := os.ReadFile(filepath.Join(plain, produce.RunNoteName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(read), `"overrides": []`) {
		t.Error("a run with nothing overridden writes something other than an empty list")
	}
}

// A directory given on the command line is where the run writes, and the note
// records it even when the recipe names the same one.
//
// The window records its field on every run, because the line it shows for a
// terminal carries --out whenever the field holds anything, and the two notes
// have to say the same thing of one run (DEC-255). This pins the half of that
// promise that belongs to the command line: a flag that was given is a move,
// whatever it moved to.
//
// 🔴 The case of another directory also asks where the files went. Until
// 2026-10-13 no test did, and a mutation leaving the recipe pointing at its own
// directory was green on every test of this package.
func TestADirectoryFromTheCommandLineIsWhereTheRunWritesAndTheNoteSaysSo(t *testing.T) {
	for _, each := range []struct {
		name   string
		inside func(dir string) string
	}{
		{"the recipe names the same directory", func(dir string) string { return filepath.Join(dir, "data") }},
		{"the recipe names another directory", func(dir string) string { return filepath.Join(dir, "elsewhere") }},
	} {
		t.Run(each.name, func(t *testing.T) {
			dir := t.TempDir()
			out := filepath.ToSlash(filepath.Join(dir, "data"))
			named := filepath.ToSlash(each.inside(dir))
			path := recipeIn(t, dir, smallRecipe(named))

			if got := run(commandGenerate, "--out", out, path); got.code != 0 {
				t.Fatalf("the run ended with %d: %s", got.code, got.log)
			}
			note := noteIn(t, out)
			want := produce.Override{Setting: "output.dir", FromRecipe: named, FromFlag: out}
			if len(note.Overrides) != 1 || note.Overrides[0] != want {
				t.Errorf("the note carries %+v rather than %+v", note.Overrides, want)
			}
			if named != out {
				if _, err := os.Stat(named); err == nil {
					t.Errorf("the run wrote into %s, which the flag moved it away from", named)
				}
			}
		})
	}
}

// noteIn reads the record of the run a verb left beside its data.
func noteIn(t *testing.T, out string) produce.RunNote {
	t.Helper()
	read, err := os.ReadFile(filepath.Join(out, produce.RunNoteName))
	if err != nil {
		t.Fatalf("cannot read the note: %v", err)
	}
	var note produce.RunNote
	if err := json.Unmarshal(read, &note); err != nil {
		t.Fatalf("the note is not a document a pipeline can read: %v", err)
	}
	return note
}

// A second run into the same directory stops rather than overwriting, and says
// what to do instead.
func TestASecondRunIntoTheSameDirectoryIsRefused(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	path := recipeIn(t, dir, smallRecipe(out))

	if first := run(commandGenerate, path); first.code != 0 {
		t.Fatalf("the first run ended with %d: %s", first.code, first.log)
	}
	before, err := os.ReadFile(filepath.Join(out, "customer.csv"))
	if err != nil {
		t.Fatal(err)
	}

	second := run(commandGenerate, path)
	if second.code != int(codeIO) {
		t.Errorf("the second run ended with %d and the table says %d for a write failure",
			second.code, codeIO)
	}
	if !strings.Contains(second.log, "already exists") {
		t.Errorf("the refusal does not say what is in the way: %q", second.log)
	}
	after, err := os.ReadFile(filepath.Join(out, "customer.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("the refused run changed the file it refused to overwrite")
	}
	if left := temporaries(t, out); len(left) != 0 {
		t.Errorf("the refused run left its temporary files behind: %v", left)
	}
}

// The flag that allows overwriting has to actually allow it, or the refusal
// above would be a wall rather than a question.
func TestOverwritingIsAllowedWhenAskedFor(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	path := recipeIn(t, dir, smallRecipe(out))

	if first := run(commandGenerate, path); first.code != 0 {
		t.Fatalf("the first run ended with %d: %s", first.code, first.log)
	}
	if second := run(commandGenerate, "--overwrite", path); second.code != 0 {
		t.Fatalf("the second run was refused even with the flag: %s", second.log)
	}
}

// A dry run answers the same arithmetic and touches nothing.
func TestADryRunAnswersWithoutWritingAnything(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	got := run(commandGenerate, "--dry-run", "--json", recipeIn(t, dir, smallRecipe(out)))

	if got.code != 0 {
		t.Fatalf("the dry run ended with %d: %s", got.code, got.log)
	}
	var plan jsonPlan
	if err := json.Unmarshal([]byte(got.data), &plan); err != nil {
		t.Fatalf("the dry run did not answer with a document: %v", err)
	}
	if len(plan.Tables) != 2 || plan.Tables[0].Rows != 3 || plan.Tables[1].Rows != 4 {
		t.Errorf("the dry run does not describe what a real run would make: %+v", plan.Tables)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("the dry run created the output directory, so it touched the disk after all")
	}
}

// The summary of a run says what happened, and a trial run did not write.
//
// Two failures live on that one line and both were found by measuring rather
// than by reading it. A trial run said rows written, which sends somebody
// looking for files nobody made, and a single table said 1 tables, which is the
// exact sentence counted was written to prevent.
func TestTheSummarySaysWhatTheRunDidAndCountsAgreeWithTheirWords(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	path := recipeIn(t, dir, smallRecipe(out))

	trial := run(commandGenerate, "--dry-run", path)
	if trial.code != 0 {
		t.Fatalf("the trial run ended with %d: %s", trial.code, trial.log)
	}
	if !strings.Contains(trial.log, "2 tables, 7 rows would be written") {
		t.Errorf("the trial run does not say what a real one would write: %q", trial.log)
	}
	if strings.Contains(trial.log, "rows written") {
		t.Errorf("the trial run claims it wrote rows: %q", trial.log)
	}

	written := run(commandGenerate, path)
	if written.code != 0 {
		t.Fatalf("the run ended with %d: %s", written.code, written.log)
	}
	if !strings.Contains(written.log, "2 tables, 7 rows written") {
		t.Errorf("the run does not say what it wrote: %q", written.log)
	}

	// One table of one row is the case where every word of the line is
	// singular, and it is the case the plural rule gets wrong.
	single := filepath.Join(dir, "single")
	body := `version: "0.1"
seed: "test-seed"
entities:
  customer:
    count: 1
    fields:
      first_name: { generator: first_name }
output:
  format: csv
  dir: ` + strings.ReplaceAll(single, `\`, `/`) + `
  file_per_entity: true
`
	lone := run(commandGenerate, "--dry-run", recipeIn(t, t.TempDir(), body))
	if lone.code != 0 {
		t.Fatalf("the run of one entity ended with %d: %s", lone.code, lone.log)
	}
	if !strings.Contains(lone.log, "1 table, 1 row would be written") {
		t.Errorf("the counts do not agree with their words: %q", lone.log)
	}
}

// A generator this build does not have is refused by name, with the list of the
// ones it does, rather than producing a column of nothing.
func TestAGeneratorThisBuildLacksIsRefusedWithTheList(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	body := strings.Replace(smallRecipe(out),
		"first_name: { generator: first_name }", "first_name: { generator: nosuchthing }", 1)

	got := run(commandGenerate, recipeIn(t, dir, body))
	if got.code != int(codeConfig) {
		t.Errorf("an unknown generator ended with %d and the table says %d for a bad recipe",
			got.code, codeConfig)
	}
	if !strings.Contains(got.log, "nosuchthing") || !strings.Contains(got.log, "pesel") {
		t.Errorf("the refusal does not name the generator or list what exists: %q", got.log)
	}
}

// A flag that carries a value overrides the recipe and says so, and one that
// was not given leaves the recipe alone.
func TestAFlagThatOverridesTheRecipeSaysSo(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	path := recipeIn(t, dir, smallRecipe(out))

	plain := run(commandGenerate, "--dry-run", "--json", path)
	moved := run(commandGenerate, "--dry-run", "--json", "--seed", "another", path)

	var before, after jsonPlan
	if err := json.Unmarshal([]byte(plain.data), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(moved.data), &after); err != nil {
		t.Fatal(err)
	}
	if before.Seed != "test-seed" {
		t.Errorf("the seed of the recipe was changed by a flag nobody gave: %q", before.Seed)
	}
	if after.Seed != "another" {
		t.Errorf("the flag did not override the recipe: %q", after.Seed)
	}
	if len(after.Overrides) != 1 || after.Overrides[0].FromRecipe != "test-seed" {
		t.Errorf("the run does not record what the flag moved: %+v", after.Overrides)
	}
	if len(before.Overrides) != 0 {
		t.Errorf("a run with no flags recorded an override: %+v", before.Overrides)
	}
}

// temporaries returns the part files a run may have left behind.
func temporaries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var left []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".part") {
			left = append(left, entry.Name())
		}
	}
	return left
}

// The paths a recipe can fail on before a single row is produced. Each of them
// ends the run with a different code, and a pipeline branches on the code rather
// than on the message.
//
// These were the least covered lines of the verb until 2026-08-05, which is a
// weak reason to write a test and not the reason here: every one of them is a
// way somebody's build stops, and until now none of them had ever run.
func TestARecipeThatCannotBeUsedEndsWithTheCodeThatSaysWhy(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")

	for _, sample := range []struct {
		name string
		body string
		path string
		code exitCode
		says string
	}{
		{
			name: "a directory where a recipe was expected",
			path: dir,
			code: codeIO,
			says: "directory",
		},
		{
			name: "a recipe that is not there at all",
			path: filepath.Join(dir, "nothing.yaml"),
			code: codeIO,
			says: "cannot read",
		},
		{
			name: "a format this build does not write",
			body: strings.Replace(smallRecipe(out), "format: csv", "format: parquet", 1),
			code: codeConfig,
			says: "parquet",
		},
	} {
		t.Run(sample.name, func(t *testing.T) {
			path := sample.path
			if path == "" {
				path = recipeIn(t, t.TempDir(), sample.body)
			}
			got := run(commandGenerate, path)
			if got.code != int(sample.code) {
				t.Errorf("ended with %d and the table says %d: %s", got.code, sample.code, got.log)
			}
			if !strings.Contains(got.log, sample.says) {
				t.Errorf("the refusal does not say %q: %s", sample.says, got.log)
			}
		})
	}
}

// A field left empty on purpose is empty in the file, and the share is drawn at
// its own address so that changing what a field draws from does not move which
// rows are blank.
func TestAFieldWithAShareOfEmptyValuesLeavesSomeRowsBlank(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	body := strings.Replace(smallRecipe(out),
		"first_name: { generator: first_name }",
		`first_name: { generator: first_name, empty: "50%" }`, 1)

	if got := run(commandGenerate, recipeIn(t, dir, body)); got.code != 0 {
		t.Fatalf("the run ended with %d: %s", got.code, got.log)
	}
	written, err := os.ReadFile(filepath.Join(out, "customer.csv"))
	if err != nil {
		t.Fatal(err)
	}
	blank := 0
	for _, line := range strings.Split(strings.TrimSuffix(string(written), "\n"), "\n")[1:] {
		if strings.HasPrefix(line, ",") {
			blank++
		}
	}
	if blank == 0 {
		t.Errorf("no row was left blank although half of them were asked to be:\n%s", written)
	}
}
