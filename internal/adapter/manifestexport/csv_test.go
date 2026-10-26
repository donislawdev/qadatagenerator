package manifestexport_test

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
)

// read parses the file back with the standard library, so that what is asserted
// is what a reader gets rather than what the writer meant.
func read(t *testing.T, written string) []map[string]string {
	t.Helper()
	// The reader checks that every record has as many fields as the heading, so
	// a row broken by a value is a failure here rather than a wrong assertion
	// further down.
	rows, err := csv.NewReader(strings.NewReader(written)).ReadAll()
	if err != nil {
		t.Fatalf("the file does not read back as CSV: %v\n%s", err, written)
	}
	if len(rows) == 0 {
		t.Fatalf("the file has no heading:\n%s", written)
	}
	var out []map[string]string
	for _, row := range rows[1:] {
		named := map[string]string{}
		for at, cell := range row {
			named[rows[0][at]] = cell
		}
		out = append(out, named)
	}
	return out
}

// 🔴 Every ordered entry has a row here, and this is the decision that separates
// this form from the Gherkin one.
//
// The Gherkin form drops what was not applied because a row there is an
// instruction to a test. This form is read by a person reviewing what a run
// did, and a table showing only what succeeded reads as the whole account of
// the run. When seven of sixty ordered defects were applied, that reading is
// wrong in the direction that matters most.
func TestEveryOrderedEntryHasARowAndSaysWhetherItHappened(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewCSV(&out)
	table.Add(applied(10, "pesel", "L3.checksum.off_by_one"))
	table.Add(refused(11, "ssn", "L3.range.unassigned"))
	if err := table.Finish(); err != nil {
		t.Fatalf("the file did not close: %v", err)
	}

	rows := read(t, out.String())
	if len(rows) != 2 {
		t.Fatalf("two entries were ordered and the file has %d rows:\n%s", len(rows), out.String())
	}
	if rows[0]["applied"] != "yes" || rows[1]["applied"] != "no" {
		t.Errorf("the file does not say which one happened: %v", rows)
	}
	if rows[1]["failure_reason"] == "" || rows[1]["failure_kind"] == "" {
		t.Errorf("the entry that did not happen does not say why: %v", rows[1])
	}
	// The other direction. A row carrying a reason for damage that did happen
	// would be describing a failure that is not there.
	if rows[0]["failure_reason"] != "" || rows[0]["failure_kind"] != "" {
		t.Errorf("the entry that happened carries a failure anyway: %v", rows[0])
	}
}

// 🔴 A value a spreadsheet would run is written so that it shows instead.
//
// The catalog names this class as the one injection aimed at the file an
// application exports, and this file is exactly that. The column name comes out
// of somebody else's file, so it is whatever they wrote.
//
// Both directions, because a writer that prefixed every value would pass a test
// looking only at the hostile one, and would corrupt every ordinary name in the
// report.
func TestAValueASpreadsheetWouldRunIsWrittenSoItShows(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewCSV(&out)
	// The line feed and the four full width forms since 2026-10-26, after an
	// outside review wrote "\n=1+1" and "＝1+1" through as column names.
	opening := []string{
		`=cmd|" /C calc"!A0`, "+1+1", "-2+3", "@SUM(A1:A9)",
		"\n=1+1", "＝1+1", "＋1+1", "－2+3", "＠SUM(A1:A9)",
	}
	for at, field := range append(append([]string{}, opening...), "pesel", "id") {
		table.Add(applied(at+1, field, "L3.checksum.off_by_one"))
	}
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	rows := read(t, out.String())
	if len(rows) != len(opening)+2 {
		t.Fatalf("%d entries were ordered and the file has %d rows", len(opening)+2, len(rows))
	}
	for _, row := range rows[:len(opening)] {
		if !strings.HasPrefix(row["field"], "'") {
			t.Errorf("%q opens a formula and was written as it stands", row["field"])
		}
	}
	for _, row := range rows[len(opening):] {
		if strings.HasPrefix(row["field"], "'") {
			t.Errorf("%q is an ordinary name and was altered anyway", row["field"])
		}
	}
}

// 🔴 The values a mutator produced are not in this file, and that is deliberate.
//
// They are in the canonical manifest. They are also the one part of an entry
// that can be an attack payload rather than a description of one, because the
// catalog has a module whose purpose is to produce hostile values. This form
// exists to be opened in a spreadsheet, so a payload written here would be
// handed the one target it wants.
func TestTheValuesThemselvesAreNotInTheSpreadsheetForm(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewCSV(&out)
	entry := applied(1, "pesel", "L3.checksum.off_by_one")
	entry.ValueBefore, entry.HasValueBefore = "=WEBSERVICE(\"http://example.invalid\")", true
	entry.ValueAfter, entry.HasValueAfter = "=WEBSERVICE(\"http://example.invalid\")", true
	table.Add(entry)
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out.String(), "WEBSERVICE") {
		t.Errorf("the payload of a mutator reached the spreadsheet form:\n%s", out.String())
	}
}

// A separator or a quote in a column name does not break the record.
//
// The standard library does the quoting, so this asserts that nothing is
// written around it that would undo the work.
func TestASeparatorInAColumnNameDoesNotBreakTheRecord(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewCSV(&out)
	table.Add(applied(1, `a,b`, "L3.checksum.off_by_one"))
	table.Add(applied(2, `say "no"`, "L3.checksum.off_by_one"))
	table.Add(applied(3, "two\nlines", "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	rows := read(t, out.String())
	want := []string{`a,b`, `say "no"`, "two\nlines"}
	if len(rows) != len(want) {
		t.Fatalf("three entries were ordered and the file reads back as %d rows", len(rows))
	}
	for at, expected := range want {
		if rows[at]["field"] != expected {
			t.Errorf("row %d came back as %q, want %q", at, rows[at]["field"], expected)
		}
	}
}
