package verify_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// promised builds a manifest through its own model, so that a test cannot
// compare against an envelope the tool would never have written. It is written
// to 0.1 and lists no entity, which is a dataset of the one table its entries
// stand in.
func promised(t *testing.T, rows int, defects ...manifest.Defect) manifest.Manifest {
	t.Helper()
	return built(t, "0.1", rows, nil, defects...)
}

// listing builds a manifest of 0.2, which lists every table with its rows.
func listing(t *testing.T, tables []manifest.Entity, defects ...manifest.Defect) manifest.Manifest {
	t.Helper()
	rows := 0
	for _, table := range tables {
		rows += table.Rows
	}
	return built(t, "0.2", rows, tables, defects...)
}

func built(t *testing.T, version string, rows int, tables []manifest.Entity,
	defects ...manifest.Defect) manifest.Manifest {
	t.Helper()
	made, problems := manifest.New(version, manifest.Run{
		GeneratedAt: "2026-08-12T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: rows, RowsMutated: len(defects), Entities: tables,
	}, defects)
	if len(problems) > 0 {
		t.Fatalf("the test built a manifest the model refuses: %v", problems)
	}
	return made
}

func damage(row int, expectation catalog.Expectation, confidence catalog.Confidence) manifest.Defect {
	return damageIn("customer", row, expectation, confidence)
}

func damageIn(entity string, row int, expectation catalog.Expectation,
	confidence catalog.Confidence) manifest.Defect {
	return manifest.Defect{
		ID:            entity + "-" + strconv.Itoa(row),
		Location:      manifest.Location{Entity: entity, Row: row, Field: "pesel"},
		Mutator:       "L3.checksum.off_by_one",
		Layer:         catalog.Layer("L3"),
		Applied:       true,
		ValueAfter:    "sent",
		HasValueAfter: true,
		Expectation:   expectation,
		Confidence:    confidence,
		Rationale:     "the check digit no longer agrees",
	}
}

func answered(rows ...response.Row) response.Record {
	return response.Record{SpecVersion: "0.1", Rows: rows}
}

// compare is a comparison the test expects to be made.
func compare(t *testing.T, promised manifest.Manifest, actual response.Record) verify.Report {
	t.Helper()
	report, err := verify.Compare(promised, actual)
	if err != nil {
		t.Fatalf("the comparison was refused: %v", err)
	}
	return report
}

func kindsIn(report verify.Report) []verify.Kind {
	var kinds []verify.Kind
	for _, finding := range report.Findings {
		kinds = append(kinds, finding.Kind)
	}
	return kinds
}

func has(report verify.Report, kind verify.Kind) bool {
	for _, found := range kindsIn(report) {
		if found == kind {
			return true
		}
	}
	return false
}

// The four disagreements are told apart, because they mean different things to
// whoever reads the report.
func TestTheFourDisagreementsAreToldApart(t *testing.T) {
	for _, given := range []struct {
		what        string
		expectation catalog.Expectation
		answer      response.Row
		want        verify.Kind
	}{
		{"a value that should have been refused was taken", catalog.Rejection,
			response.Row{Number: 1, Verdict: response.Accepted}, verify.Hole},
		{"a value that should have been taken was refused", catalog.Acceptance,
			response.Row{Number: 1, Verdict: response.Rejected}, verify.Overzealous},
		{"a value expecting a mark was taken in silence", catalog.Warning,
			response.Row{Number: 1, Verdict: response.Accepted, HasWarnings: true}, verify.Silent},
	} {
		t.Run(given.what, func(t *testing.T) {
			report := compare(t,
				promised(t, 1, damage(1, given.expectation, catalog.Hard)),
				answered(given.answer))
			if !has(report, given.want) {
				t.Errorf("wanted %q and the report has %v", given.want, kindsIn(report))
			}
			if !report.Failed() {
				t.Error("a hard disagreement did not fail the run")
			}
		})
	}

	// The fourth is the one no comparison of a row against a verdict finds,
	// because both sides of such a comparison assume the row was seen.
	report := compare(t, promised(t, 2, damage(1, catalog.Rejection, catalog.Hard)),
		answered(response.Row{Number: 2, Verdict: response.Accepted}))
	if !has(report, verify.Missing) {
		t.Errorf("a row nobody answered for was not reported missing: %v", kindsIn(report))
	}
}

// 🔴 A soft disagreement is reported and does not fail the run.
//
// Soft means the correct reaction depends on rules belonging to the
// application rather than to the data. Failing on one would be this tool
// asserting something it was never entitled to assert, and every first run at
// anybody's site would be red for a reason nobody can act on.
func TestASoftDisagreementIsReportedAndDoesNotFailTheRun(t *testing.T) {
	report := compare(t,
		promised(t, 1, damage(1, catalog.Rejection, catalog.Soft)),
		answered(response.Row{Number: 1, Verdict: response.Accepted}))

	if len(report.Findings) != 1 {
		t.Fatalf("the disagreement was not reported at all: %v", report.Findings)
	}
	if report.Failed() {
		t.Error("a soft disagreement failed the run, so the first run anywhere is red")
	}
	// And the other direction, so that a build failing on nothing would be
	// caught as surely as one failing on everything.
	hard := compare(t,
		promised(t, 1, damage(1, catalog.Rejection, catalog.Hard)),
		answered(response.Row{Number: 1, Verdict: response.Accepted}))
	if !hard.Failed() {
		t.Error("a hard disagreement did not fail the run")
	}
}

// 🔴 A value taken and quietly changed is told from one taken as it was sent.
//
// Measured rather than imagined: a widely deployed database configuration takes
// a row, turns text into zero, cuts an overlong string to the column width and
// reports no error at all. A system holding something other than what it was
// sent did not take the value, it took a different one.
func TestAValueQuietlyChangedIsToldFromOneTakenAsSent(t *testing.T) {
	for _, given := range []struct {
		what   string
		stored map[string]string
		says   string
	}{
		{"held exactly as sent", map[string]string{"pesel": "sent"}, "exactly as sent"},
		{"quietly changed", map[string]string{"pesel": "0"}, "is not what was sent"},
		{"nobody looked", nil, "does not say what is held now"},
	} {
		t.Run(given.what, func(t *testing.T) {
			answer := response.Row{Number: 1, Verdict: response.Accepted,
				Stored: given.stored, HasStored: given.stored != nil}
			report := compare(t,
				promised(t, 1, damage(1, catalog.Rejection, catalog.Hard)),
				answered(answer))
			if len(report.Findings) != 1 {
				t.Fatalf("expected one finding, got %v", report.Findings)
			}
			if !strings.Contains(report.Findings[0].Detail, given.says) {
				t.Errorf("the detail is %q and should say %q",
					report.Findings[0].Detail, given.says)
			}
		})
	}
}

// 🔴 A question the response record cannot answer is left unsettled rather than
// counted as a pass.
//
// An entry expecting the application to complain cannot be judged by a file
// that never says whether anything was said. Calling that a pass would be
// inventing the answer, which is the one thing this report exists not to do.
func TestAQuestionTheRecordCannotAnswerIsLeftUnsettled(t *testing.T) {
	report := compare(t,
		promised(t, 1, damage(1, catalog.Warning, catalog.Hard)),
		answered(response.Row{Number: 1, Verdict: response.Accepted}))

	if len(report.Findings) != 0 {
		t.Errorf("an unanswerable question was reported as a disagreement: %v", report.Findings)
	}
	if len(report.Unsettled) != 1 {
		t.Fatalf("an unanswerable question was not recorded as unsettled: %v", report.Unsettled)
	}
	if report.Failed() {
		t.Error("a question nobody could answer failed the run")
	}
	if len(report.Limits) == 0 {
		t.Error("the report does not say what it could not see")
	}
}

// Records the application never reacted to are counted from the dataset, not
// from the damaged rows, because a receiver dropping everything past a limit
// drops undamaged records just as readily.
func TestRecordsNobodyReactedToAreCountedFromTheDataset(t *testing.T) {
	report := compare(t,
		promised(t, 60, damage(1, catalog.Rejection, catalog.Hard)),
		answered(response.Row{Number: 1, Verdict: response.Rejected}))

	if !has(report, verify.Missing) {
		t.Fatalf("fifty nine records went unanswered and nothing said so: %v", kindsIn(report))
	}
	if !report.Failed() {
		t.Error("a dataset that arrived incomplete did not fail the run")
	}
	// The one damaged row was answered correctly, so the finding is about the
	// dataset rather than about that row.
	for _, finding := range report.Findings {
		if finding.Kind == verify.Missing && finding.Row != 0 {
			t.Errorf("the shortfall was attributed to row %d rather than to the dataset",
				finding.Row)
		}
	}
}

// The count of records nobody reacted to agrees with its verb, for one record
// as for many.
//
// 🔴 The real window read "so 1 were never reacted to" on 2026-09-23 (B-289),
// and so did the command line, since both print the same sentence.
func TestOneRecordNobodyReactedToIsSaidInTheSingular(t *testing.T) {
	for _, tc := range []struct {
		rows int
		want string
	}{
		{2, "so 1 was never reacted to"},
		{3, "so 2 were never reacted to"},
	} {
		report := compare(t,
			promised(t, tc.rows, damage(1, catalog.Rejection, catalog.Hard)),
			answered(response.Row{Number: 1, Verdict: response.Rejected}))
		said := ""
		for _, finding := range report.Findings {
			if finding.Kind == verify.Missing && finding.Row == 0 {
				said = finding.Detail
			}
		}
		if !strings.HasSuffix(said, tc.want) {
			t.Errorf("%d records and one answer read %q where it should end %q", tc.rows, said, tc.want)
		}
		// The same number as a number, for a surface that says it in fewer
		// words. One source, so the two cannot come apart.
		if report.NeverReacted != tc.rows-1 {
			t.Errorf("%d records and one answer counted %d never reacted to, want %d",
				tc.rows, report.NeverReacted, tc.rows-1)
		}
	}
}

// A dataset answered in full has nothing never reacted to, and the count says
// nought rather than being left for a reader to work out.
func TestADatasetAnsweredInFullCountsNothingNeverReactedTo(t *testing.T) {
	report := compare(t,
		promised(t, 1, damage(1, catalog.Rejection, catalog.Hard)),
		answered(response.Row{Number: 1, Verdict: response.Rejected}))
	if report.NeverReacted != 0 {
		t.Errorf("one record answered out of one counted %d never reacted to", report.NeverReacted)
	}
}

// Damage that was never applied has nothing in the dataset to react to, so it
// is not compared and does not count against the application.
func TestDamageThatNeverHappenedIsNotComparedAgainstAnybody(t *testing.T) {
	unapplied := damage(1, catalog.Rejection, catalog.Hard)
	unapplied.Applied = false
	unapplied.HasValueAfter = false
	unapplied.ValueAfter = ""
	unapplied.FailureReason = "this build carries no implementation of this damage"
	unapplied.FailureKind = manifest.Unimplemented

	report := compare(t, promised(t, 1, unapplied),
		answered(response.Row{Number: 1, Verdict: response.Accepted}))

	if report.Compared != 0 {
		t.Errorf("%d entries were compared and none should have been", report.Compared)
	}
	if len(report.Findings) != 0 {
		t.Errorf("damage that never happened was held against the application: %v", report.Findings)
	}
	if report.Failed() {
		t.Error("a run whose damage was never applied failed because of the application")
	}
}

// twoTables is a dataset of a customer table and an order table, each with a
// row 1 carrying damage that expects the opposite verdict of the other.
func twoTables(t *testing.T, customers, orders int) manifest.Manifest {
	t.Helper()
	return listing(t, []manifest.Entity{{Name: "customer", Rows: customers}, {Name: "order", Rows: orders}},
		damageIn("customer", 1, catalog.Rejection, catalog.Hard),
		damageIn("order", 1, catalog.Acceptance, catalog.Hard))
}

// 🔴 A row is judged by the answer about its own table (R22, B-192).
//
// Row 1 of customers was rightly refused and row 1 of orders rightly taken.
// Looked up by number alone, both entries met the first answer for row 1, so
// the order was judged against the customer's refusal and a run in which the
// application did everything right failed with an overzealous rejection.
func TestARowIsJudgedByTheAnswerAboutItsOwnTable(t *testing.T) {
	report := compare(t, twoTables(t, 1, 1), response.Record{SpecVersion: "0.2", Rows: []response.Row{
		{Entity: "customer", Number: 1, Verdict: response.Rejected},
		{Entity: "order", Number: 1, Verdict: response.Accepted},
	}})
	if len(report.Findings) != 0 || report.Failed() {
		t.Errorf("an application that did what both tables expected was reported as %v", report.Findings)
	}
	if len(report.Entities) != 2 {
		t.Errorf("a dataset of two tables was reported as %v", report.Entities)
	}
}

// 🔴 Records nobody reacted to are counted table by table (R21, B-192).
//
// The shortfall used to be the sum of every table less the rows of the record,
// so a record about the customers alone could be balanced by rows of orders it
// never had, and one about orders alone was a shortfall blamed on nobody.
func TestRecordsNobodyReactedToAreCountedTableByTable(t *testing.T) {
	report := compare(t, twoTables(t, 2, 3), response.Record{SpecVersion: "0.2", Rows: []response.Row{
		{Entity: "customer", Number: 1, Verdict: response.Rejected},
		{Entity: "customer", Number: 2, Verdict: response.Accepted},
		{Entity: "order", Number: 1, Verdict: response.Accepted},
	}})
	var short []verify.Finding
	for _, finding := range report.Findings {
		if finding.Kind == verify.Missing && finding.Row == 0 {
			short = append(short, finding)
		}
	}
	if len(short) != 1 || short[0].Entity != "order" ||
		!strings.Contains(short[0].Detail, "3 records of order and the response record answers for 1, so 2 were") {
		t.Fatalf("one table answered in full and one short by two came back as %+v", short)
	}
	if report.NeverReacted != 2 {
		t.Errorf("two records never reacted to were counted as %d", report.NeverReacted)
	}
}

// 🔴 A row that cannot be placed in the dataset refuses the record (DEC-266).
//
// Each of these makes a verdict about some other record than the one the
// application saw. Past the end is the usual sign of every row shifted by one,
// because the producer counted the header, and until 2026-10-27 such a row also
// made up for one that never arrived (R21).
func TestARowThatCannotBePlacedRefusesTheRecord(t *testing.T) {
	single := promised(t, 60, damage(1, catalog.Rejection, catalog.Hard))
	for _, given := range []struct {
		what     string
		promised manifest.Manifest
		rows     []response.Row
		says     string
	}{
		{"no entity in a dataset of two", twoTables(t, 1, 1),
			[]response.Row{{Number: 1, Verdict: response.Accepted}},
			"row 1 names no entity, and the dataset has 2, customer and order"},
		{"an entity the manifest does not list", twoTables(t, 1, 1),
			[]response.Row{{Entity: "invoice", Number: 1, Verdict: response.Accepted}},
			`row 1 names the entity "invoice", which the manifest does not list. It lists customer and order`},
		{"an entity beside the only table", single,
			[]response.Row{{Entity: "order", Number: 1, Verdict: response.Accepted}},
			`names the entity "order", which the manifest does not list. It lists customer`},
		{"past the end of the dataset", single,
			[]response.Row{{Number: 61, Verdict: response.Accepted}},
			"row 61 is past the end of the dataset, which has 60 rows"},
		{"past the end of one table", twoTables(t, 1, 3),
			[]response.Row{{Entity: "customer", Number: 2, Verdict: response.Accepted}},
			"row 2 is past the end of customer, which has 1 row"},
		{"one place named with and without its entity", single,
			[]response.Row{{Number: 1, Verdict: response.Accepted},
				{Entity: "customer", Number: 1, Verdict: response.Rejected}},
			"row 1 is answered twice in the dataset, once with the entity named and once without it"},
	} {
		t.Run(given.what, func(t *testing.T) {
			_, err := verify.Compare(given.promised, response.Record{SpecVersion: "0.2", Rows: given.rows})
			if err == nil {
				t.Fatal("the record was compared")
			}
			if !strings.Contains(err.Error(), given.says) {
				t.Errorf("the refusal says %q and should say %q", err, given.says)
			}
			if fault.KindOf(err) != fault.Config {
				t.Errorf("the refusal is %v, and a record that does not fit is a mistake in the input",
					fault.KindOf(err))
			}
		})
	}
}

// Four hundred rows without an entity are one sentence, since the mistake that
// leaves the entity out leaves it out of every row.
func TestRowsRefusedForOneReasonAreOneSentence(t *testing.T) {
	rows := make([]response.Row, 0, 400)
	for number := 1; number <= 400; number++ {
		rows = append(rows, response.Row{Number: number, Verdict: response.Accepted})
	}
	_, err := verify.Compare(twoTables(t, 400, 400), response.Record{SpecVersion: "0.2", Rows: rows})
	if err == nil {
		t.Fatal("four hundred rows without an entity were compared")
	}
	said := err.Error()
	if !strings.Contains(said, "rows 1, 2, 3 and 397 more name no entity") || len(said) > 600 {
		t.Errorf("four hundred rows without an entity were refused in %d characters: %s", len(said), said)
	}
}

// A manifest written to 0.1 lists no entity, so its tables are where its entries
// stand. With one, the dataset is that table and a row may name it or not. With
// several, a row is placed by name and held to the dataset as a whole, and the
// report says it could do no better.
func TestAManifestOfTheEarlierVersionIsReadByWhatItCanSay(t *testing.T) {
	one := promised(t, 2, damage(1, catalog.Rejection, catalog.Hard))
	report := compare(t, one, response.Record{SpecVersion: "0.2", Rows: []response.Row{
		{Entity: "customer", Number: 1, Verdict: response.Rejected},
		{Number: 2, Verdict: response.Accepted},
	}})
	if len(report.Findings) != 0 {
		t.Errorf("a table of 0.1 answered in full came back as %v", report.Findings)
	}

	several := promised(t, 3, damageIn("customer", 1, catalog.Rejection, catalog.Hard),
		damageIn("order", 1, catalog.Acceptance, catalog.Hard))
	report = compare(t, several, response.Record{SpecVersion: "0.2", Rows: []response.Row{
		{Entity: "customer", Number: 1, Verdict: response.Rejected},
		{Entity: "order", Number: 1, Verdict: response.Accepted},
	}})
	if report.NeverReacted != 1 {
		t.Errorf("three records of which two were answered counted %d never reacted to", report.NeverReacted)
	}
	if !strings.Contains(strings.Join(report.Limits, ". "), "how many rows each entity had") {
		t.Errorf("the report did not say the tables of 0.1 have no sizes: %v", report.Limits)
	}
	if _, err := verify.Compare(several, response.Record{SpecVersion: "0.2", Rows: []response.Row{
		{Entity: "order", Number: 4, Verdict: response.Accepted},
	}}); err == nil || !strings.Contains(err.Error(), "3 rows across its entities") {
		t.Errorf("a row past every table of 0.1 came back as %v", err)
	}
}

// A document written to a newer minor version is read as far as this build
// understands it, and the report says so. An older one says nothing, because
// everything it can say is something this build knows.
func TestANewerDocumentIsSaidAmongTheLimits(t *testing.T) {
	for _, given := range []struct {
		version string
		said    bool
	}{
		{"0.1", false},
		{"0.2", false},
		{"0.3", true},
	} {
		report := compare(t, promised(t, 1, damage(1, catalog.Rejection, catalog.Hard)),
			response.Record{SpecVersion: given.version, Rows: []response.Row{{Number: 1, Verdict: response.Rejected}}})
		said := strings.Contains(strings.Join(report.Limits, ". "),
			"the response record is written to version "+given.version)
		if said != given.said {
			t.Errorf("a record of %s: said among the limits %t, want %t (%v)",
				given.version, said, given.said, report.Limits)
		}
	}
}

// The window asks the report what the application said about an entry, so it
// is answered from the places the comparison judged with, and a row nobody
// answered is absent rather than accepted.
func TestAnEntryIsAnsweredFromItsOwnTable(t *testing.T) {
	order := damageIn("order", 1, catalog.Acceptance, catalog.Hard)
	report := compare(t, twoTables(t, 2, 1), response.Record{SpecVersion: "0.2", Rows: []response.Row{
		{Entity: "customer", Number: 1, Verdict: response.Rejected},
		{Entity: "order", Number: 1, Verdict: response.Accepted, Code: "OK"},
	}})
	if answer, found := report.Answer(order); !found || answer.Code != "OK" {
		t.Errorf("the order entry was answered with %+v (found %t)", answer, found)
	}
	if _, found := report.Answer(damageIn("customer", 2, catalog.Rejection, catalog.Hard)); found {
		t.Error("a row the application never answered was found")
	}
}
