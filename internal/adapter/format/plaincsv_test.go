package format_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

func plainTable(columns ...string) dataset.Table {
	return dataset.Table{Entity: "contacts", Columns: columns, Rows: 2}
}

func TestATableReadFromAFileIsWrittenBackUnderTheNameItWasGiven(t *testing.T) {
	files := newHeld()
	writer, err := format.NewPlainCSV(files, "contacts.corrupted.csv", ",")
	if err != nil {
		t.Fatalf("cannot make the writer: %v", err)
	}
	table := plainTable("id", "name")
	if err := writer.Begin(table); err != nil {
		t.Fatalf("cannot begin: %v", err)
	}
	for _, values := range [][]string{{"1", "Ada"}, {"2", "Grace"}} {
		if err := writer.Write(dataset.Record{Values: values, Parent: dataset.NoParent}); err != nil {
			t.Fatalf("cannot write: %v", err)
		}
	}
	if err := writer.End(table); err != nil {
		t.Fatalf("cannot end: %v", err)
	}
	if err := writer.Finish(); err != nil {
		t.Fatalf("cannot finish: %v", err)
	}

	// 🔴 The name of the file and the name of the records differ on purpose in
	// this mode, and this is where that is held. The records keep the name of
	// the source so a person finds them in the manifest, and the file carries a
	// suffix so that reading and writing cannot meet on one path.
	if _, made := files.written["contacts.corrupted.csv"]; !made {
		t.Fatalf("the file was written as %v rather than under the name it was given",
			files.written)
	}
	if _, madeAfterEntity := files.written["contacts.csv"]; madeAfterEntity {
		t.Error("the file was named after the entity, which is the name of the source")
	}
	if got := files.written["contacts.corrupted.csv"].String(); got != "id,name\n1,Ada\n2,Grace\n" {
		t.Errorf("the bytes came out as %q", got)
	}
}

// The quoting and the line ending are the promise, and they are the same
// promise as the writer that lays out a recipe. That is why both go through one
// piece of code, and this asks that they still do.
func TestTheBytesAreQuotedTheWayTheRestOfThisPackageQuotes(t *testing.T) {
	files := newHeld()
	writer, err := format.NewPlainCSV(files, "out.csv", ",")
	if err != nil {
		t.Fatalf("cannot make the writer: %v", err)
	}
	table := plainTable("note")
	_ = writer.Begin(table)
	for _, value := range []string{"a, b", "say \"hi\"", "line\nbreak", "plain"} {
		_ = writer.Write(dataset.Record{Values: []string{value}, Parent: dataset.NoParent})
	}
	_ = writer.End(table)

	written := files.written["out.csv"].String()
	for _, expected := range []string{`"a, b"`, `"say ""hi"""`, "\"line\nbreak\"", "\nplain\n"} {
		if !strings.Contains(written, expected) {
			t.Errorf("the output does not carry %q, and reads %q", expected, written)
		}
	}
	if strings.Contains(written, "\r") {
		t.Error("a carriage return reached the output, and the line ending is a promise")
	}
}

// One file, one table. A second one would be written over the first, and this
// mode has no name to give it, so it is refused rather than resolved.
func TestASecondTableIsRefused(t *testing.T) {
	files := newHeld()
	writer, err := format.NewPlainCSV(files, "out.csv", ",")
	if err != nil {
		t.Fatalf("cannot make the writer: %v", err)
	}
	if err := writer.Begin(plainTable("a")); err != nil {
		t.Fatalf("the first table was refused: %v", err)
	}
	err = writer.Begin(dataset.Table{Entity: "orders", Columns: []string{"a"}})
	if err == nil {
		t.Fatal("a second table was accepted and would have replaced the first")
	}
	if !strings.Contains(err.Error(), "orders") {
		t.Errorf("the refusal has to name the table it would not take, and says %q", err)
	}
}

func TestADelimiterOfMoreThanOneCharacterIsRefused(t *testing.T) {
	if _, err := format.NewPlainCSV(newHeld(), "out.csv", "||"); err == nil {
		t.Error("a delimiter of two characters was accepted")
	}
	if _, err := format.NewPlainCSV(newHeld(), "out.csv", ""); err != nil {
		t.Errorf("no delimiter has to mean a comma, and was refused: %v", err)
	}
}

func TestAnotherDelimiterIsWritten(t *testing.T) {
	files := newHeld()
	writer, _ := format.NewPlainCSV(files, "out.csv", ";") // punctuation: test data
	table := plainTable("a", "b")
	_ = writer.Begin(table)
	_ = writer.Write(dataset.Record{Values: []string{"1", "2"}, Parent: dataset.NoParent})
	_ = writer.End(table)
	if got := files.written["out.csv"].String(); got != "a;b\n1;2\n" { // punctuation: test data
		t.Errorf("the output came out as %q", got)
	}
}
