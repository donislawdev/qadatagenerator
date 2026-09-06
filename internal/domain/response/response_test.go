package response_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/response"
)

// contains reports whether any problem carries the fragment, so that a test can
// name the fault it means without freezing the whole sentence.
func contains(problems []string, fragment string) bool {
	for _, problem := range problems {
		if strings.Contains(problem, fragment) {
			return true
		}
	}
	return false
}

// Whoever produced this file is going to run the tool again after fixing what
// it named, so a reader that stopped at the first fault would turn one mistake
// into as many runs as there are mistakes.
func TestEveryFaultOfARecordIsNamedInOnePass(t *testing.T) {
	record := response.Record{
		Rows: []response.Row{
			{Number: 0, Verdict: response.Accepted},
			{Number: 2, Verdict: "maybe"},
			{Number: 3, Verdict: response.Rejected},
			{Number: 3, Verdict: response.Rejected},
		},
	}

	problems := record.Problems()
	named := map[string]string{
		"the version of the format nobody wrote": "which version of the format",
		"a row numbered below one":               "rows count from one",
		"a verdict that is neither of the two":   `"maybe"`,
		"a row answered twice":                   "answered twice",
	}
	for fault, fragment := range named {
		if !contains(problems, fragment) {
			t.Errorf("%s was not named: %v", fault, problems)
		}
	}
	if len(problems) != len(named) {
		t.Errorf("a record with %d faults named %d problems: %v",
			len(named), len(problems), problems)
	}
}

// Two documents describing one run have to be comparable line by line, which is
// the same rule the rest of this project follows about iteration order.
func TestTheProblemsComeBackInOneOrderRatherThanTheOrderTheyWereFound(t *testing.T) {
	record := response.Record{
		Rows: []response.Row{
			{Number: 9, Verdict: "no idea"},
			{Number: 1, Verdict: "also no idea"},
			{Number: 0, Verdict: response.Accepted},
		},
	}
	problems := record.Problems()
	if !sort.StringsAreSorted(problems) {
		t.Errorf("the problems came back in the order they were found: %v", problems)
	}
}

// A run where the application reacted to nothing is a run where every row went
// missing, and saying that needs the rows it did react to.
func TestARecordThatNamesNoRowIsRefused(t *testing.T) {
	record := response.Record{SpecVersion: "1.0", Rows: nil}
	problems := record.Problems()
	if !contains(problems, "names no row at all") {
		t.Fatalf("a record answering about nothing was not refused: %v", problems)
	}
	if len(problems) != 1 {
		t.Errorf("a record whose only fault is having no rows named %d problems: %v",
			len(problems), problems)
	}
}

// The other side of the same rule. A record this reader accepts has to come
// back with nothing to say, or the faults above would be true of every file.
func TestARecordWithNothingWrongWithItNamesNoProblem(t *testing.T) {
	record := response.Record{
		SpecVersion: "1.0",
		Rows: []response.Row{
			{Number: 1, Verdict: response.Accepted},
			{Number: 2, Verdict: response.Rejected, Code: "22001", Message: "value too long"},
		},
	}
	if problems := record.Problems(); len(problems) != 0 {
		t.Fatalf("a usable record was refused: %v", problems)
	}
}

// A row that is not numbered has no number to have been answered twice about,
// so two of them are two faults about numbering rather than one duplicate. The
// order of the two checks is what decides this, which is why it is named.
func TestARowWithNoUsableNumberIsNotAlsoCountedAsAnsweredTwice(t *testing.T) {
	record := response.Record{
		SpecVersion: "1.0",
		Rows: []response.Row{
			{Number: 0, Verdict: response.Accepted},
			{Number: 0, Verdict: response.Accepted},
		},
	}
	problems := record.Problems()
	if contains(problems, "answered twice") {
		t.Errorf("two unnumbered rows were reported as one row answered twice: %v", problems)
	}
	if len(problems) != 2 {
		t.Errorf("two unnumbered rows produced %d problems: %v", len(problems), problems)
	}
}

// A field the producer did not mention is not a field that survived. It is a
// field nobody looked at, and the difference is the measured one: a database
// taking a row, turning text into zero and cutting a string to the column width
// reports no error at all. Recorded as DEC-166.
func TestAFieldNobodyReadBackIsNotAFieldThatSurvived(t *testing.T) {
	const sent = "90041158021"

	unread := response.Row{Number: 1, Verdict: response.Accepted}
	if changed, known := unread.Changed("national_id", sent); known {
		t.Errorf("a producer that read nothing back was taken to have answered (changed=%v)", changed)
	}

	elsewhere := response.Row{
		Number: 1, Verdict: response.Accepted,
		Stored: map[string]string{"email": "somebody@example.test"}, HasStored: true,
	}
	if changed, known := elsewhere.Changed("national_id", sent); known {
		t.Errorf("a field the producer never mentioned was answered for (changed=%v)", changed)
	}

	held := response.Row{
		Number: 1, Verdict: response.Accepted,
		Stored: map[string]string{"national_id": sent}, HasStored: true,
	}
	if changed, known := held.Changed("national_id", sent); !known || changed {
		t.Errorf("a field held as it was sent came back changed=%v known=%v", changed, known)
	}

	cut := response.Row{
		Number: 1, Verdict: response.Accepted,
		Stored: map[string]string{"national_id": "9004115802"}, HasStored: true,
	}
	if changed, known := cut.Changed("national_id", sent); !known || !changed {
		t.Errorf("a field cut to the column width came back changed=%v known=%v", changed, known)
	}
}

// The same pair for anything the application said without refusing the row.
// Treating silence as agreement is what this file exists to prevent.
func TestSilenceAboutWarningsIsNotTheAbsenceOfWarnings(t *testing.T) {
	silent := response.Row{Number: 1, Verdict: response.Accepted}
	if marked, known := silent.Marked(); known {
		t.Errorf("a producer that answered nothing about warnings was read as answering (marked=%v)", marked)
	}

	none := response.Row{Number: 1, Verdict: response.Accepted, HasWarnings: true}
	if marked, known := none.Marked(); !known || marked {
		t.Errorf("a producer that said there were no warnings came back marked=%v known=%v", marked, known)
	}

	some := response.Row{
		Number: 1, Verdict: response.Accepted,
		HasWarnings: true, Warnings: []string{"the value was cut to the column width"},
	}
	if marked, known := some.Marked(); !known || !marked {
		t.Errorf("a producer that named a warning came back marked=%v known=%v", marked, known)
	}
}

// A row the application never answered is absent rather than accepted, which is
// what a receiver silently dropping everything past a limit actually produces.
func TestARowTheApplicationNeverAnsweredIsAbsentRatherThanAccepted(t *testing.T) {
	record := response.Record{
		SpecVersion: "1.0",
		Rows:        []response.Row{{Number: 2, Verdict: response.Rejected}},
	}

	if _, reacted := record.ByRow(1); reacted {
		t.Error("a row the application never answered was found")
	}
	row, reacted := record.ByRow(2)
	if !reacted {
		t.Fatal("a row the application answered was not found")
	}
	if row.Verdict != response.Rejected {
		t.Errorf("the verdict of row 2 came back as %q", row.Verdict)
	}
}
