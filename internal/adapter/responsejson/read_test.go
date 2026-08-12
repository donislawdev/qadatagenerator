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
	row, answered := record.ByRow(1)
	if !answered || row.Code != "E1" {
		t.Errorf("row one came back as %+v", row)
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

	looked, _ := record.ByRow(1)
	if !looked.HasStored || !looked.HasWarnings {
		t.Error("a producer that answered with an empty list was taken for one that said nothing")
	}
	if marked, known := looked.Marked(); marked || !known {
		t.Errorf("an empty warning list came back as marked=%t known=%t", marked, known)
	}

	silent, _ := record.ByRow(2)
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
		{"no rows at all", `{"spec_version":"0.1","rows":[]}`, "names no row"},
		{"a row numbered zero", `{"spec_version":"0.1","rows":[{"row":0,"verdict":"accepted"}]}`,
			"count from one"},
		{"one row answered twice",
			`{"spec_version":"0.1","rows":[{"row":1,"verdict":"accepted"},{"row":1,"verdict":"rejected"}]}`,
			"answered twice"},
		{"a verdict nobody knows",
			`{"spec_version":"0.1","rows":[{"row":1,"verdict":"maybe"}]}`, "verdict"},
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
