package sourcecsv_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/sourcecsv"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// written puts a file on the disk exactly as given, bytes and all.
//
// The bytes matter here more than anywhere else in this project: this is the one
// package whose whole job is to accept what somebody else wrote, so a helper
// that normalised line endings on the way in would test a file nobody has.
func written(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write the file this test reads: %v", err)
	}
	return path
}

// allRows walks a file to the end and returns what came out.
func allRows(t *testing.T, file *sourcecsv.File) [][]string {
	t.Helper()
	var rows [][]string
	for {
		record, more, err := file.Next()
		if err != nil {
			t.Fatalf("reading stopped: %v", err)
		}
		if !more {
			return rows
		}
		rows = append(rows, record.Values)
	}
}

func TestAFileIsReadIntoATableAndItsRecords(t *testing.T) {
	path := written(t, "contacts.csv", "id,name\n1,Ada\n2,Grace\n")
	file, err := sourcecsv.Open(path, ',')
	if err != nil {
		t.Fatalf("cannot open an ordinary file: %v", err)
	}
	defer func() { _ = file.Close() }()

	table := file.Table()
	if table.Entity != "contacts" {
		t.Errorf("the records were called %q rather than after the file", table.Entity)
	}
	if want := []string{"id", "name"}; strings.Join(table.Columns, ",") != strings.Join(want, ",") {
		t.Errorf("the columns came out as %v", table.Columns)
	}
	if table.Rows != 2 {
		t.Errorf("2 records went in and the table says %d", table.Rows)
	}
	if rows := allRows(t, file); len(rows) != 2 || rows[1][1] != "Grace" {
		t.Errorf("the records came out as %v", rows)
	}
}

// 🔴 The two passes are the whole shape of DEC-155, so the count has to be
// right before a single record comes out. A caller building a manifest envelope
// asks for it before the first row is damaged.
func TestTheRowCountIsKnownBeforeTheFirstRecord(t *testing.T) {
	path := written(t, "contacts.csv", "a\n1\n2\n3\n4\n5\n")
	file, err := sourcecsv.Open(path, ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	defer func() { _ = file.Close() }()
	if file.Table().Rows != 5 {
		t.Fatalf("the count is %d before anything was read", file.Table().Rows)
	}
	if rows := allRows(t, file); len(rows) != file.Table().Rows {
		t.Errorf("the count promised %d rows and the walk gave %d", file.Table().Rows, len(rows))
	}
}

// Quoting, embedded delimiters and records spanning lines are the reason this
// side uses a real parser rather than splitting on a character.
func TestQuotedFieldsSurviveTheWayTheyWereWritten(t *testing.T) {
	path := written(t, "notes.csv", "id,note\n1,\"a, b\"\n2,\"line\nbreak\"\n3,\"say \"\"hi\"\"\"\n")
	file, err := sourcecsv.Open(path, ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	defer func() { _ = file.Close() }()
	rows := allRows(t, file)
	want := []string{"a, b", "line\nbreak", `say "hi"`}
	for at, expected := range want {
		if rows[at][1] != expected {
			t.Errorf("row %d carried %q rather than %q", at+1, rows[at][1], expected)
		}
	}
}

// A file whose quoting is broken is refused by name. Reading it into whatever
// the salvage produces would be damage this tool did not order and does not
// describe.
func TestBrokenQuotingIsRefusedRatherThanSalvaged(t *testing.T) {
	path := written(t, "broken.csv", "id,note\n1,\"unclosed\n2,fine\n")
	if _, err := sourcecsv.Open(path, ','); err == nil {
		t.Fatal("a file with broken quoting was read as though it were fine")
	}
}

// A row with the wrong number of fields is refused rather than read into
// columns that have quietly shifted.
func TestARaggedRowIsRefused(t *testing.T) {
	path := written(t, "ragged.csv", "a,b,c\n1,2,3\n4,5\n")
	_, err := sourcecsv.Open(path, ',')
	if err == nil {
		t.Fatal("a row short of a field was accepted")
	}
	if !strings.Contains(err.Error(), "ragged.csv") {
		t.Errorf("the refusal has to name the file and says %q", err)
	}
}

func TestAFileWithNothingInItIsRefused(t *testing.T) {
	if _, err := sourcecsv.Open(written(t, "empty.csv", ""), ','); err == nil {
		t.Error("an empty file was accepted")
	}
}

// A header with no rows under it is its own refusal, because there is nothing
// to damage and a run that wrote an empty file with a manifest beside it would
// look like a run that did something.
func TestAHeaderWithNoRecordsIsRefused(t *testing.T) {
	_, err := sourcecsv.Open(written(t, "headings.csv", "a,b\n"), ',')
	if err == nil {
		t.Fatal("a file of headings alone was accepted")
	}
	if !strings.Contains(err.Error(), "at least one row") {
		t.Errorf("the refusal has to say what would work, and says %q", err)
	}
}

// A column with no name cannot be pointed at in a manifest, and two columns of
// one name make every location ambiguous. Both are refused rather than repaired,
// because repairing invents a name that is not in the file.
func TestAHeaderThatCannotNameAColumnIsRefused(t *testing.T) {
	for _, given := range []struct {
		name    string
		content string
		says    string
	}{
		{"blank.csv", "a,,c\n1,2,3\n", "no name"},
		{"twice.csv", "a,b,a\n1,2,3\n", "two columns"},
	} {
		_, err := sourcecsv.Open(written(t, given.name, given.content), ',')
		if err == nil {
			t.Errorf("%s was accepted", given.name)
			continue
		}
		if !strings.Contains(err.Error(), given.says) {
			t.Errorf("%s was refused with %q, which does not say %q", given.name, err, given.says)
		}
	}
}

// A byte order mark is taken off the first column name and reported, rather
// than left to become part of it. A column whose name still began with one would be a column
// no policy could exclude and no manifest reader could match.
func TestAByteOrderMarkIsTakenOffAndReported(t *testing.T) {
	path := written(t, "excel.csv", "\ufeffid,name\n1,Ada\n")
	file, err := sourcecsv.Open(path, ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	defer func() { _ = file.Close() }()
	if got := file.Table().Columns[0]; got != "id" {
		t.Errorf("the first column is called %q", got)
	}
	if !file.HadByteOrderMark() {
		t.Error("the file began with a byte order mark and nothing says so, " +
			"so the output would lose it in silence")
	}
}

// Blanks are counted rather than sampled. A column of national identifiers with
// gaps in it is still one, and a blank is evidence of nothing either way.
//
// 🔴 A blank cell and a blank LINE are different things, and this test is where
// that was measured rather than assumed. A line with nothing on it at all is not
// a record for a comma separated file, so the parser steps over it and it never
// reaches a sample, a count or the file that gets written back. A line holding a
// space is a record whose one field is blank, and that one is counted.
//
// The consequence is not confined to this package: a file with blank lines in it
// comes back without them, which is a change nobody ordered. It is stated in
// what a run says about itself, beside the encoding and the line endings.
func TestABlankCellIsCountedAndABlankLineIsNotEvenARecord(t *testing.T) {
	path := written(t, "gaps.csv", "a\n1\n\n2\n \n3\n")
	file, err := sourcecsv.Open(path, ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	defer func() { _ = file.Close() }()

	if file.Table().Rows != 4 {
		t.Errorf("6 lines carried 4 records and the count says %d", file.Table().Rows)
	}
	sample := file.Samples()[0]
	if sample.NonEmpty != 3 || sample.Empty != 1 {
		t.Errorf("3 values and 1 blank cell went in, and the sample says %d and %d",
			sample.NonEmpty, sample.Empty)
	}
	if len(sample.Values) != 3 {
		t.Errorf("the kept values are %v", sample.Values)
	}
}

// The sample is capped and the counts are not, which is what lets a report say
// how much of a column was looked at rather than leave it assumed.
func TestTheSampleIsCappedAndTheCountIsNot(t *testing.T) {
	var file strings.Builder
	file.WriteString("a\n")
	rows := sourcecsv.SampleCap + 25
	for row := 0; row < rows; row++ {
		file.WriteString("x\n")
	}
	opened, err := sourcecsv.Open(written(t, "long.csv", file.String()), ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	defer func() { _ = opened.Close() }()

	sample := opened.Samples()[0]
	if len(sample.Values) != sourcecsv.SampleCap {
		t.Errorf("the cap is %d and %d values were kept", sourcecsv.SampleCap, len(sample.Values))
	}
	if sample.NonEmpty != rows {
		t.Errorf("%d values went in and the count says %d", rows, sample.NonEmpty)
	}
	if opened.Table().Rows != rows {
		t.Errorf("the row count is capped too, at %d", opened.Table().Rows)
	}
}

// Another delimiter is read as asked. This is exactly where an import falls
// over, so it is a setting rather than a detail.
func TestAnotherDelimiterIsRead(t *testing.T) {
	file, err := sourcecsv.Open(written(t, "euro.csv", "a;b\n1;2\n"), ';') // punctuation: test data
	if err != nil {
		t.Fatalf("cannot open a semicolon separated file: %v", err)
	}
	defer func() { _ = file.Close() }()
	if rows := allRows(t, file); len(rows) != 1 || rows[0][1] != "2" {
		t.Errorf("the row came out as %v", rows)
	}
}

// A record carries no parent, because one file is one table. A writer handed
// something else would put a column of relations into a flat file.
func TestARecordFromAFileBelongsToNobody(t *testing.T) {
	file, err := sourcecsv.Open(written(t, "flat.csv", "a\n1\n"), ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	defer func() { _ = file.Close() }()
	record, _, err := file.Next()
	if err != nil {
		t.Fatalf("cannot read: %v", err)
	}
	if record.Parent != dataset.NoParent {
		t.Errorf("a record read from a flat file says its parent is row %d", record.Parent)
	}
}

// Closing twice is the ordinary case rather than a mistake: a walk that reaches
// the end closes itself, and the caller that opened the file closes it too.
func TestClosingTwiceIsAllowed(t *testing.T) {
	file, err := sourcecsv.Open(written(t, "short.csv", "a\n1\n"), ',')
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	allRows(t, file)
	if err := file.Close(); err != nil {
		t.Errorf("closing a file whose walk already ended failed: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Errorf("closing twice failed: %v", err)
	}
}

func TestTheRecordsAreNamedAfterTheFileWithoutItsExtension(t *testing.T) {
	for path, want := range map[string]string{
		"contacts.csv":           "contacts",
		"/tmp/a/b/customers.CSV": "customers",
		"no-extension":           "no-extension",
		"two.dots.csv":           "two.dots",
	} {
		if got := sourcecsv.EntityOf(path); got != want {
			t.Errorf("%s gave the entity %q rather than %q", path, got, want)
		}
	}
}
