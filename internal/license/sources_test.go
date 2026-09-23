package license

import (
	"reflect"
	"testing"
)

// Rows of one project at one version become one source with their lists in the
// order of the register, and the same project at another version is another
// source, because terms belong to a version rather than to a project.
func TestReferenceDataIsGroupedByProjectAndVersionInTheOrderOfTheRegister(t *testing.T) {
	rows := []DataSource{
		{Corpus: "en-US/a", Project: "p", Version: "1", License: "MIT", Source: "s", Checked: "2026-01-01"},
		{Corpus: "en-US/b", Project: "q", Version: "1", License: "ISC", Source: "t", Checked: "2026-01-02"},
		{Corpus: "pl-PL/c", Project: "p", Version: "1", License: "MIT", Source: "s", Checked: "2026-01-01"},
		{Corpus: "pl-PL/d", Project: "p", Version: "2", License: "MIT", Source: "s", Checked: "2026-01-03"},
	}
	want := []Source{
		{Project: "p", Version: "1", License: "MIT", Source: "s", Checked: "2026-01-01",
			Lists: []string{"en-US/a", "pl-PL/c"}},
		{Project: "q", Version: "1", License: "ISC", Source: "t", Checked: "2026-01-02",
			Lists: []string{"en-US/b"}},
		{Project: "p", Version: "2", License: "MIT", Source: "s", Checked: "2026-01-03",
			Lists: []string{"pl-PL/d"}},
	}
	if got := BySource(rows); !reflect.DeepEqual(got, want) {
		t.Errorf("grouped as\n%+v\nwant\n%+v", got, want)
	}
	if got := BySource(nil); len(got) != 0 {
		t.Errorf("no reference data grouped into %d sources", len(got))
	}
}

// A source shows one day beside its license, taken from its first row. That is
// honest only while every row of the group was read on that day, so the
// register is held to it here rather than trusted.
func TestEveryListOfOneSourceWasReadOnTheSameDay(t *testing.T) {
	first := map[string]string{}
	for _, row := range Data {
		key := row.Project + " " + row.Version
		if day, seen := first[key]; seen && day != row.Checked {
			t.Errorf("%s lists %s as read on %s and an earlier list of the same source on %s, "+
				"so the one day shown for the source would hide the other", key, row.Corpus, row.Checked, day)
		}
		if _, seen := first[key]; !seen {
			first[key] = row.Checked
		}
	}
	if len(first) == 0 {
		t.Fatal("the register holds no reference data, so this test checks nothing")
	}
}
