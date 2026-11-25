package values_test

import (
	"fmt"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// records is a writer that keeps every record of a run as it arrived.
type records struct{ kept []dataset.Record }

func (r *records) Begin(dataset.Table) error { return nil }
func (r *records) End(dataset.Table) error   { return nil }
func (r *records) Finish() error             { return nil }

func (r *records) Write(record dataset.Record) error {
	r.kept = append(r.kept, record)
	return nil
}

// recordsOf plans and runs a recipe and returns its records, failing the test
// on either refusal.
func recordsOf(t *testing.T, rec recipe.Recipe) []dataset.Record {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	kept := &records{}
	if err := generate.Run(plan, rec, source, kept, generate.NeverStops{}); err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	return kept.kept
}

// Columns of the table below, in the order of its fields.
const (
	columnCoupon = iota
	columnNote
	columnName
)

// marksAgree says what is wrong with one record of the table below, or nothing.
//
// Every value this table draws has at least one character, so a cell is
// without a value exactly when its text is empty: the coupon in the rows its
// share leaves empty, and the note in every row, since nothing fills an
// expression yet.
func marksAgree(record dataset.Record) error {
	for column, value := range record.Values {
		if record.AbsentAt(column) != (value == "") {
			return fmt.Errorf("column %d holds %q and is marked absent: %v", column, value,
				record.AbsentAt(column))
		}
	}
	if !record.AbsentAt(columnNote) {
		return fmt.Errorf("the note, which nothing fills, is not marked absent")
	}
	if record.AbsentAt(columnName) {
		return fmt.Errorf("the required name is marked absent")
	}
	return nil
}

// A cell left without a value is marked as such, and a cell with a value is not.
//
// The mark is what lets a writer tell a cell with nothing in it from a value
// that is the empty text, the difference between NULL and an empty string in
// a database (DEC-341). The canary hands the same check a record whose empty
// cell is not marked, which is what this run produced before the mark existed.
func TestACellLeftWithoutAValueIsMarkedAndAValueIsNot(t *testing.T) {
	rec := recipe.Recipe{
		Version: "0.1", Seed: "absent", Locale: "en-US",
		Output: recipe.Output{Format: "csv"},
		Entities: []recipe.Entity{{Name: "account", Count: 200, HasCount: true, Fields: []recipe.Field{
			{Name: "coupon", Values: []string{"SPRING10", "WELCOME"}, EmptyShare: 50, HasEmptyShare: true},
			{Name: "note", Expression: "${coupon}"},
			{Name: "name", Generator: values.FirstName, Required: true},
		}}},
	}

	kept := recordsOf(t, rec)
	if len(kept) != 200 {
		t.Fatalf("the run wrote %d records of 200", len(kept))
	}
	empty, filled := 0, 0
	for row, record := range kept {
		if err := marksAgree(record); err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if record.AbsentAt(columnCoupon) {
			empty++
		} else {
			filled++
		}
	}
	if empty == 0 || filled == 0 {
		t.Errorf("a share of half left %d coupons empty and %d filled, so one side was not tried",
			empty, filled)
	}

	unmarked := dataset.Record{Values: []string{"", "", "Ann"}, Absent: []bool{false, true, false}}
	if marksAgree(unmarked) == nil {
		t.Error("canary: the check passed a record whose empty coupon is not marked absent")
	}
}

// A row in which every cell has a value carries no marks at all.
//
// Most rows of most recipes are like that, and a slice of marks for each of
// them would be memory spent on saying nothing.
func TestARowWithAValueInEveryCellCarriesNoMarks(t *testing.T) {
	rec := recipe.Recipe{
		Version: "0.1", Seed: "absent", Locale: "en-US",
		Output: recipe.Output{Format: "csv"},
		Entities: []recipe.Entity{{Name: "account", Count: 20, HasCount: true, Fields: []recipe.Field{
			{Name: "name", Generator: values.FirstName},
			{Name: "plan", Values: []string{"free", "pro"}},
		}}},
	}
	for row, record := range recordsOf(t, rec) {
		if record.Absent != nil {
			t.Fatalf("row %d of a table with a value in every cell carries marks %v", row, record.Absent)
		}
	}
}
