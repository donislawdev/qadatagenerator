package responsejson_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/responsejson"
)

func written(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "response.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write the file this test reads: %v", err)
	}
	return path
}

const sound = `{"spec_version":"0.1","dataset":{"rows_total":2},
 "rows":[{"row":1,"verdict":"rejected","code":"E1"},{"row":2,"verdict":"accepted"}]}`

func TestASoundRecordIsRead(t *testing.T) {
	record, err := responsejson.Load(written(t, sound))
	if err != nil {
		t.Fatalf("a sound record was refused: %v", err)
	}
	if len(record.Rows) != 2 || !record.HasRowsTotal || record.RowsTotal != 2 {
		t.Fatalf("the record came back as %+v", record)
	}
	if row := record.Rows[0]; row.Number != 1 || row.Code != "E1" || row.Entity != "" {
		t.Errorf("row one came back as %+v", row)
	}
}

// 🔴 An empty list is read as an answer and a list left out or written as null
// is refused (DEC-271). The two came out of the reader as one record of no rows
// until 2026-11-03, so the refusal could not tell them apart and took both
// (B-338).
func TestAnEmptyListOfRowsIsReadAndNoListIsRefused(t *testing.T) {
	record, err := responsejson.Load(written(t, `{"spec_version":"0.2","rows":[]}`))
	if err != nil {
		t.Fatalf("a record saying the application reacted to nothing was refused: %v", err)
	}
	if !record.HasRows || len(record.Rows) != 0 {
		t.Errorf("an empty list came back as %+v", record)
	}
	for _, given := range []struct{ what, content string }{
		{"left out", `{"spec_version":"0.2"}`},
		{"written as null", `{"spec_version":"0.2","rows":null}`},
	} {
		t.Run(given.what, func(t *testing.T) {
			_, err := responsejson.Load(written(t, given.content))
			if err == nil {
				t.Fatalf("a record whose list of rows is %s was read", given.what)
			}
			if !strings.Contains(err.Error(), "carries no list of rows") {
				t.Errorf("the refusal of a list %s says %q", given.what, err)
			}
		})
	}
}

// A row of version 0.2 says which entity it belongs to, and the same number
// under two entities is two rows of a dataset of two tables rather than one
// row answered twice.
func TestAnEntityIsReadWithItsRow(t *testing.T) {
	record, err := responsejson.Load(written(t, `{"spec_version":"0.2","rows":[
	  {"entity":"customer","row":1,"verdict":"accepted"},
	  {"entity":"order","row":1,"verdict":"rejected"}]}`))
	if err != nil {
		t.Fatalf("two tables with a row 1 each were refused: %v", err)
	}
	if record.Rows[0].Entity != "customer" || record.Rows[1].Entity != "order" {
		t.Errorf("the entities came back as %q and %q", record.Rows[0].Entity, record.Rows[1].Entity)
	}
}

// 🔴 A member this build does not know is a typo in a version it knows and a
// later addition in a version it does not (DEC-266). Refusing both made every
// newer record unreadable before its version was looked at, and skipping both
// would compare against evidence somebody believes they supplied.
func TestAnUnknownMemberIsRefusedOrSkippedByTheVersion(t *testing.T) {
	for _, given := range []struct {
		what    string
		version string
		read    bool
		says    string
	}{
		{"the version this build speaks", "0.2", false, "member names"},
		{"an earlier version", "0.1", false, "member names"},
		{"a later minor version", "0.3", true, ""},
		{"a later major version", "1.0", false, "this build speaks"},
	} {
		t.Run(given.what, func(t *testing.T) {
			record, err := responsejson.Load(written(t, `{"spec_version":"`+given.version+
				`","rows":[{"row":1,"verdict":"accepted","confidence":"high"}],"producer":"x"}`))
			switch {
			case given.read && err != nil:
				t.Fatalf("a record of %s was refused for members it may add: %v", given.version, err)
			case given.read && (len(record.Rows) != 1 || record.Rows[0].Verdict != "accepted"):
				t.Errorf("a record of %s came back as %+v", given.version, record)
			case !given.read && err == nil:
				t.Fatalf("a record of %s with a member nobody knows was accepted", given.version)
			case !given.read && !strings.Contains(err.Error(), given.says):
				t.Errorf("the refusal of %s says %q and should mention %q", given.version, err, given.says)
			}
		})
	}
}

// 🔴 Whether the producer looked is a different answer from what they found.
//
// An empty object means they read the row back and it was unchanged. No member
// at all means nobody read anything. Treating the second as the first is
// exactly the silence this format exists to prevent, so the reader keeps them
// apart and the model carries both.
func TestLookingAndFindingNothingIsNotTheSameAsNotLooking(t *testing.T) {
	record, err := responsejson.Load(written(t,
		`{"spec_version":"0.1","rows":[
		  {"row":1,"verdict":"accepted","stored":{},"warnings":[]},
		  {"row":2,"verdict":"accepted"}]}`))
	if err != nil {
		t.Fatalf("a sound record was refused: %v", err)
	}

	looked := record.Rows[0]
	if !looked.HasStored || !looked.HasWarnings {
		t.Error("a producer that answered with an empty list was taken for one that said nothing")
	}
	if marked, known := looked.Marked(); marked || !known {
		t.Errorf("an empty warning list came back as marked=%t known=%t", marked, known)
	}

	silent := record.Rows[1]
	if silent.HasStored || silent.HasWarnings {
		t.Error("a producer that said nothing was taken for one that answered")
	}
	if _, known := silent.Marked(); known {
		t.Error("a row with no warnings member was treated as an answer about marking")
	}
}

// Everything a record cannot be, refused with a reason rather than read
// half way.
func TestARecordThatCannotBeComparedIsRefusedWithAReason(t *testing.T) {
	for _, given := range []struct {
		what    string
		content string
		says    string
	}{
		{"no version", `{"rows":[{"row":1,"verdict":"accepted"}]}`, "which version"},
		// An empty list is an answer since DEC-271, and a list left out is not.
		{"no list of rows", `{"spec_version":"0.1"}`, "carries no list of rows"},
		{"a row numbered zero", `{"spec_version":"0.1","rows":[{"row":0,"verdict":"accepted"}]}`,
			"count from one"},
		{"one row answered twice",
			`{"spec_version":"0.1","rows":[{"row":1,"verdict":"accepted"},{"row":1,"verdict":"rejected"}]}`,
			"answered twice"},
		{"a verdict nobody knows",
			`{"spec_version":"0.1","rows":[{"row":1,"verdict":"maybe"}]}`, "verdict"},
		// Two problems are two sentences, and the second begins with a capital
		// (B-352). It read ". row 1 has the verdict" until 2026-11-03.
		{"two problems at once",
			`{"spec_version":"0.1","rows":[{"row":0,"verdict":"accepted"},{"row":1,"verdict":"maybe"}]}`,
			`because the header is not a record. Row 1 has the verdict "maybe"`},
		// A misspelled member is the likely mistake in a file somebody writes
		// by hand, and skipping it would mean comparing against evidence the
		// producer believes they supplied.
		{"a misspelled member",
			`{"spec_version":"0.1","rows":[{"row":1,"verdict":"accepted","store":{}}]}`, "member names"},
		{"two documents in one file",
			`{"spec_version":"0.1","rows":[{"row":1,"verdict":"accepted"}]}{"spec_version":"0.1"}`,
			"more than one"},
		{"not a document at all", `nonsense`, "response record"},
	} {
		t.Run(given.what, func(t *testing.T) {
			_, err := responsejson.Load(written(t, given.content))
			if err == nil {
				t.Fatalf("%s was accepted", given.what)
			}
			if !strings.Contains(err.Error(), given.says) {
				t.Errorf("the refusal for %s says %q and should mention %q",
					given.what, err, given.says)
			}
		})
	}
}

// 🔴 Every fault at once, because whoever wrote this file will run the tool
// again after fixing what it named, and a reader stopping at the first fault
// turns one mistake into as many runs as there are mistakes.
func TestEveryFaultInARecordIsNamedAtOnce(t *testing.T) {
	_, err := responsejson.Load(written(t,
		`{"rows":[{"row":0,"verdict":"maybe"},{"row":0,"verdict":"accepted"}]}`))
	if err == nil {
		t.Fatal("a record with three faults was accepted")
	}
	for _, wanted := range []string{"which version", "count from one", "verdict"} {
		if !strings.Contains(err.Error(), wanted) {
			t.Errorf("the refusal does not mention %q: %v", wanted, err)
		}
	}
}

// The read is bounded, because this file comes from outside and an unbounded
// one is a way to end a run by handing it a large file.
func TestARecordLargerThanTheLimitIsRefused(t *testing.T) {
	var wide strings.Builder
	wide.WriteString(`{"spec_version":"0.1","rows":[{"row":1,"verdict":"accepted","message":"`)
	wide.WriteString(strings.Repeat("x", responsejson.MaxSize))
	wide.WriteString(`"}]}`)

	_, err := responsejson.Load(written(t, wide.String()))
	if err == nil {
		t.Fatal("a record past the limit was read")
	}
	if !strings.Contains(err.Error(), "larger than") {
		t.Errorf("the refusal does not say the file is too large: %v", err)
	}
}
