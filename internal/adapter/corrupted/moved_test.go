package corrupted_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/adapter/corrupted"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

const damageLettered catalog.ID = "L2.string.lettered"

// lettered is damage made of letters, which finds a value holding none unfit.
type lettered struct{}

func (lettered) ID() catalog.ID { return damageLettered }

func (lettered) Damage(target mutator.Target) mutator.Outcome {
	if strings.IndexFunc(target.Value, unicode.IsLetter) < 0 {
		return mutator.Unfit("the value holds no letter")
	}
	return mutator.Done(target.Value + "!")
}

// codeAndName is a table of a column of digits and a column of letters, both
// text, so that damage made of letters covers both and can only land on one.
type codeAndName struct{}

func (codeAndName) EntityNames() []string { return []string{"person"} }
func (codeAndName) FieldsOf(string) []recipe.FieldInfo {
	return []recipe.FieldInfo{{Name: "code", Type: "string"}, {Name: "name", Type: "string"}}
}
func (codeAndName) ParentsOf(string) []recipe.ParentInfo { return nil }
func (codeAndName) RulesOf(string) []recipe.Rule         { return nil }
func (codeAndName) PrimaryKeyOf(string) []string         { return nil }

// keeping is the writer the records reach, and it keeps a copy of each.
type keeping struct{ records [][]string }

func (k *keeping) Begin(dataset.Table) error { return nil }
func (k *keeping) Write(record dataset.Record) error {
	k.records = append(k.records, slices.Clone(record.Values))
	return nil
}
func (k *keeping) End(dataset.Table) error { return nil }
func (k *keeping) Finish() error           { return nil }

// 🔴 A damaged value is written to the column the manifest names, also when the
// damage moved there because the column drawn for it declined the value.
//
// The engine names the column it changed, and the writer has to put the value
// there rather than in the column the order was drawn for. Twenty rows drawn
// between two columns leave none of them drawn for the column of digits only
// with a chance of one in a million.
func TestDamageThatMovedColumnsIsWrittenWhereTheManifestSays(t *testing.T) {
	lettering := entry(damageLettered, "Adds a letter to a value of letters.")
	lettering.AppliesTo.FieldTypes = []string{"string"}
	lettering.Requires = nil
	described, problems := catalog.New("0.1", "draft", []catalog.Entry{lettering})
	if len(problems) > 0 {
		t.Fatalf("the fixture catalog was refused: %v", problems)
	}
	selector, err := corrupt.NewSelector(recipe.Policy{
		Layers: []string{string(catalog.FieldShape)}, Rows: "100%",
	}, described, codeAndName{}, seed.New("a fixed seed"))
	if err != nil {
		t.Fatalf("the fixture policy was refused: %v", err)
	}
	engine := corrupt.New(described, registry{damageLettered: lettered{}}, noSchemes{},
		capability.NewSet(), seed.New("a fixed seed"))
	log := []string{}
	written := &recorder{name: "manifest", log: &log}
	kept := &keeping{}
	writer := corrupted.New(kept, engine, &selector, corrupted.Tee{written}, codeAndName{})

	table := dataset.Table{Entity: "person", Columns: []string{"code", "name"}, Rows: 20}
	if err := writer.Begin(table); err != nil {
		t.Fatalf("the run refused to begin: %v", err)
	}
	for row := 0; row < table.Rows; row++ {
		if err := writer.Write(dataset.Record{Values: []string{"4711", "Anna"}, Parent: dataset.NoParent}); err != nil {
			t.Fatalf("row %d was refused: %v", row, err)
		}
	}
	if err := writer.End(table); err != nil {
		t.Fatalf("the run refused to end: %v", err)
	}
	if problem := misplaced(kept.records, written.entries); problem != "" {
		t.Error(problem)
	}
	// The canary is the shape of the defect: the value in the column drawn.
	planted := []manifest.Defect{{Applied: true, ValueAfter: "Anna!",
		Location: manifest.Location{Entity: "person", Row: 1, Field: "name"}}}
	if misplaced([][]string{{"Anna!", "Anna"}}, planted) == "" {
		t.Fatal("the check does not notice a value written to another column")
	}
}

// misplaced says what is wrong when a row does not hold what its entry says:
// the damaged value in the named column, and every other column as it was.
func misplaced(records [][]string, entries []manifest.Defect) string {
	original := []string{"4711", "Anna"}
	columns := []string{"code", "name"}
	if len(entries) != len(records) {
		return fmt.Sprintf("%d rows were written with %d entries", len(records), len(entries))
	}
	for _, entry := range entries {
		if !entry.Applied {
			return fmt.Sprintf("row %d was refused: %s", entry.Location.Row, entry.FailureReason)
		}
		record := records[entry.Location.Row-1]
		for at, column := range columns {
			want := original[at]
			if column == entry.Location.Field {
				want = entry.ValueAfter
			}
			if record[at] != want {
				return fmt.Sprintf("row %d holds %q in %s, and the manifest says %s took %q",
					entry.Location.Row, record[at], column, entry.Location.Field, entry.ValueAfter)
			}
		}
	}
	return ""
}
