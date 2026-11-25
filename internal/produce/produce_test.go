package produce

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// related is a recipe with the one shape a preview can get wrong: a child whose
// parent is decided by sharing the children out among the parents, which needs
// to know how many children there are.
func related(children int) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1", Seed: "preview-seed", Locale: "pl-PL",
		Entities: []recipe.Entity{
			{
				Name: "customer", Count: 40, HasCount: true, PrimaryKey: []string{"pesel"},
				Fields: []recipe.Field{
					{Name: "first_name", Generator: "first_name"},
					{Name: "gender", Generator: "gender"},
					{Name: "birth_date", Generator: "birth_date"},
					{Name: "pesel", Generator: "pesel"},
				},
				Consistency: []recipe.Rule{
					{Field: "pesel", Kind: recipe.Matches, With: []string{"birth_date", "gender"}},
				},
			},
			{
				Name: "order", Count: children, HasCount: true,
				Fields:  []recipe.Field{{Name: "placed_at", Generator: "birth_date"}},
				Parents: []recipe.Parent{{Entity: "customer", Min: 0, Max: 8}},
			},
		},
	}
}

// wholeRun writes the whole dataset the way a real run writes it, into memory,
// and reads the bytes back the same way the sample does.
//
// It is the oracle here, and it is a good one for the reason the tooling gives
// for reading artefacts with somebody else's parser: the second form of one run
// answers about the first without sharing its arithmetic. A hand written table
// of expected values would freeze whatever this build happens to produce today,
// including its mistakes.
func wholeRun(t *testing.T, rec recipe.Recipe) map[string][][]string {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the whole run could not be planned: %v", err)
	}
	held := newHeld()
	writer, err := format.For(held, rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
	if err != nil {
		t.Fatalf("the whole run has no writer: %v", err)
	}
	if err := generate.Run(plan, rec, source, writer, generate.NeverStops{}); err != nil {
		t.Fatalf("the whole run failed: %v", err)
	}

	read := map[string][][]string{}
	for _, file := range held.files() {
		reader := csv.NewReader(strings.NewReader(file.Text))
		reader.FieldsPerRecord = -1
		lines, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("the whole run wrote %s and it cannot be read back: %v", file.Name, err)
		}
		read[strings.TrimSuffix(file.Name, ".csv")] = lines
	}
	return read
}

// 🔴 The whole worth of this package is in this one test.
//
// A preview that shows rows the run will not write is worse than no preview,
// because it looks like an answer. Measured on 2026-09-02: the cheap way of
// building one, lowering the count in the recipe, agrees on every value of
// every field and disagrees on every key naming the parent, which is a wrong
// answer nobody can see. So the sample is held against the run itself.
func TestASampleIsTheFirstRowsTheRunItselfWouldWrite(t *testing.T) {
	rec := related(60)
	whole := wholeRun(t, rec)

	sample, err := Preview(rec, values.New(rec.Locale, rec.Seed), 7, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the sample could not be made: %v", err)
	}
	if len(sample.Tables) != 2 {
		t.Fatalf("expected both tables, got %d", len(sample.Tables))
	}

	for _, table := range sample.Tables {
		lines, wrote := whole[table.Entity]
		if !wrote {
			t.Fatalf("the run wrote no %s at all", table.Entity)
		}
		if strings.Join(table.Columns, "|") != strings.Join(lines[0], "|") {
			t.Errorf("%s: the sample has columns %v and the file has %v",
				table.Entity, table.Columns, lines[0])
		}
		if len(table.Rows) != 7 {
			t.Errorf("%s: asked for 7 rows and the sample carries %d", table.Entity, len(table.Rows))
		}
		for at, row := range table.Rows {
			want := lines[at+1]
			if strings.Join(row, "|") != strings.Join(want, "|") {
				t.Errorf("%s row %d: the sample says %v and the run writes %v",
					table.Entity, at, row, want)
			}
		}
	}
}

// The column naming the parent is the one the cheap preview gets wrong, so it
// is named here rather than left to the comparison above. A test that fails
// without saying which column moved sends whoever reads it back to the diff.
func TestTheKeyNamingTheParentIsTheOneTheRunWrites(t *testing.T) {
	rec := related(60)
	whole := wholeRun(t, rec)

	sample, err := Preview(rec, values.New(rec.Locale, rec.Seed), 5, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the sample could not be made: %v", err)
	}

	var child Table
	for _, table := range sample.Tables {
		if table.Entity == "order" {
			child = table
		}
	}
	column := -1
	for at, name := range child.Columns {
		if name == "customer" {
			column = at
		}
	}
	if column < 0 {
		t.Fatalf("the child carries no column naming its parent, only %v", child.Columns)
	}
	// 🔴 The sample says WHICH column that is, and it has to be this one. A
	// surface uses it to tell an empty key from an empty value, and pointing at
	// the wrong column would turn ordinary blank values into a sentence about
	// parents.
	if child.ParentColumn != column {
		t.Errorf("the sample says the parent is column %d and it is column %d",
			child.ParentColumn, column)
	}
	for _, table := range sample.Tables {
		if table.Entity == "customer" && table.ParentColumn != -1 {
			t.Errorf("a table belonging to nobody names column %d as its parent",
				table.ParentColumn)
		}
	}

	for at, row := range child.Rows {
		want := whole["order"][at+1][column]
		if row[column] != want {
			t.Errorf("row %d belongs to %q in the sample and to %q in the file",
				at, row[column], want)
		}
	}
}

// A table shorter than the sample is not a cut table, and the difference is
// what the screen puts on it. Total says how many rows the run writes, so a
// count of nought is a table that says so rather than an empty rectangle.
func TestATableSaysWhetherItWasCutOrIsSimplyShort(t *testing.T) {
	rec := related(3)
	sample, err := Preview(rec, values.New(rec.Locale, rec.Seed), DefaultRows, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the sample could not be made: %v", err)
	}

	for _, table := range sample.Tables {
		switch table.Entity {
		case "customer":
			if !table.Cut() || table.Total != 40 {
				t.Errorf("forty rows shown ten at a time is a cut table, got total %d and %d rows",
					table.Total, len(table.Rows))
			}
		case "order":
			if table.Cut() || len(table.Rows) != 3 {
				t.Errorf("three rows shown ten at a time is the whole table, got total %d and %d rows",
					table.Total, len(table.Rows))
			}
		}
	}
}

// 🔴 The promise this screen makes before anybody presses anything.
//
// DEC-35: the answer on screen leaves no trace. It is a promise about a disk, so
// it is held against a disk rather than against the type that was chosen to keep
// it. The directory this runs in is empty before and has to be empty after.
//
// 🔴 It is deliberately NOT in the register of proven guards, and the reason is
// worth more than the entry would be. The register admits one of three states
// and the unproven one is capped at nought, so an entry here would have to claim
// a substitution that turns this red. There is none that compiles: the promise
// is kept by the type that receives the bytes rather than by a branch, and every
// way of breaking it needs an import the file does not have. Claiming a proof
// that does not exist is worse than an honest absence, which is the whole point
// of that register.
func TestASampleLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if _, err := Preview(related(60), values.New("pl-PL", "preview-seed"), DefaultRows,
		generate.NeverStops{}); err != nil {
		t.Fatalf("the sample could not be made: %v", err)
	}

	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("the directory cannot be read: %v", err)
	}
	if len(left) != 0 {
		names := []string{}
		for _, entry := range left {
			names = append(names, entry.Name())
		}
		t.Errorf("a preview left %v behind", names)
	}
}

// counting is a Halt that never stops and counts how often it was asked.
type counting struct{ asked int }

func (c *counting) Stopped() error {
	c.asked++
	return nil
}

// 🔴 The stop of a preview reaches the plan, which is the part of a preview
// that is not short: it shares the children of every parent out before a row
// is cut. Until 2026-10-27 the plan was handed a stop that never answers, so a
// check of the recipe screen that a newer keystroke had made stale planned to
// the end (B-341).
//
// The run of a cut preview asks once a row, at most twice DefaultRows here, so
// a count above the forty parents can only come from the plan asking once a
// parent.
func TestTheStopOfAPreviewReachesThePlan(t *testing.T) {
	stop := &counting{}
	if _, err := Preview(related(60), values.New("pl-PL", "preview-seed"), DefaultRows,
		stop); err != nil {
		t.Fatalf("the sample could not be made: %v", err)
	}
	if stop.asked <= 40 {
		t.Errorf("a preview of forty parents asked whether to stop %d times, so the plan was "+
			"not asked at all", stop.asked)
	}
}

// A recipe the plan refuses is refused here, with the failure the plan gave,
// rather than turning into an empty sample. An empty sample would read on
// screen as a recipe that produces nothing, which is a different sentence.
func TestARecipeThePlanRefusesIsRefusedHere(t *testing.T) {
	rec := related(60)
	rec.Entities[1].Fields = append(rec.Entities[1].Fields,
		recipe.Field{Name: "customer", Generator: "first_name"})

	if _, err := Preview(rec, values.New(rec.Locale, rec.Seed), DefaultRows,
		generate.NeverStops{}); err == nil {
		t.Fatal("a recipe whose relation and field want one column produced a sample")
	} else if !strings.Contains(err.Error(), "customer") {
		t.Errorf("the refusal does not name the column both things want: %v", err)
	}
}
