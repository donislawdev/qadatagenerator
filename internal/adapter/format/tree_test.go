package format_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// The tree puts every child under the parent it names, in the order the table
// holds them, whatever order the parents come in.
//
// 🔴 The children of a row are found through an index built once for the whole
// tree since 2026-10-27, where they used to be found by walking every row of
// the child table for every parent (outside review, 2026-10-22). A plan always
// puts the children of one parent together, and an index that relied on that
// would be right for every dataset a plan makes and wrong for the first records
// that arrive otherwise. So they arrive otherwise here, and one of them names a
// parent the table above does not hold, which the tree has never written.
func TestTheTreePutsEveryChildUnderItsParentWhateverOrderTheyArriveIn(t *testing.T) {
	rec := recipe.Recipe{Seed: "s", Locale: "en-US",
		Output: recipe.Output{Format: format.NameJSON},
		Entities: []recipe.Entity{
			{Name: "p", Count: 2, HasCount: true, Fields: []recipe.Field{{Name: "a"}}},
			{Name: "c", Parents: []recipe.Parent{{Entity: "p", Min: 0, Max: 4}},
				Fields: []recipe.Field{{Name: "b"}}},
		}}
	files := newHeld()
	writer, err := format.NewJSON(files, rec, generate.NeverStops{})
	if err != nil {
		t.Fatal(err)
	}
	parents := dataset.Table{Entity: "p", Columns: []string{"a"}, Rows: 2}
	children := dataset.Table{Entity: "c", Columns: []string{"b"}, Rows: 5, Parent: "p"}
	steps := []func() error{
		func() error { return writer.Begin(parents) },
		func() error { return writer.Write(dataset.Record{Values: []string{"p0"}, Parent: dataset.NoParent}) },
		func() error { return writer.Write(dataset.Record{Values: []string{"p1"}, Parent: dataset.NoParent}) },
		func() error { return writer.End(parents) },
		func() error { return writer.Begin(children) },
		func() error { return writer.Write(dataset.Record{Values: []string{"c0"}, Parent: 1}) },
		func() error { return writer.Write(dataset.Record{Values: []string{"c1"}, Parent: 0}) },
		func() error { return writer.Write(dataset.Record{Values: []string{"c2"}, Parent: 7}) },
		func() error { return writer.Write(dataset.Record{Values: []string{"c3"}, Parent: 1}) },
		func() error { return writer.Write(dataset.Record{Values: []string{"c4"}, Parent: 0}) },
		func() error { return writer.End(children) },
		writer.Finish,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
	}

	var tree struct {
		P []struct {
			A string `json:"a"`
			C []struct {
				B string `json:"b"`
				P string `json:"p"`
			} `json:"c"`
		} `json:"p"`
	}
	text := files.written["dataset.json"].String()
	if err := json.Unmarshal([]byte(text), &tree); err != nil {
		t.Fatalf("the tree does not read: %v\n%s", err, text)
	}
	got := map[string][]string{}
	for _, parent := range tree.P {
		for _, child := range parent.C {
			got[parent.A] = append(got[parent.A], child.B+" names "+child.P)
		}
	}
	want := map[string][]string{"p0": {"c1 names 0", "c4 names 0"}, "p1": {"c0 names 1", "c3 names 1"}}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("the tree holds %v rather than %v", got, want)
	}
}
