package verify_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
)

// promised builds a manifest through its own model, so that a test cannot
// compare against an envelope the tool would never have written.
func promised(t *testing.T, rows int, defects ...manifest.Defect) manifest.Manifest {
	t.Helper()
	built, problems := manifest.New("0.1", manifest.Run{
		GeneratedAt: "2026-08-12T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: rows, RowsMutated: len(defects),
	}, defects)
	if len(problems) > 0 {
		t.Fatalf("the test built a manifest the model refuses: %v", problems)
	}
	return built
}

func damage(row int, expectation catalog.Expectation, confidence catalog.Confidence) manifest.Defect {
	return manifest.Defect{
		ID:            "case-" + string(rune('a'+row)),
		Location:      manifest.Location{Entity: "customer", Row: row, Field: "pesel"},
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
			report := verify.Compare(
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
	report := verify.Compare(promised(t, 1, damage(1, catalog.Rejection, catalog.Hard)),
		answered(response.Row{Number: 7, Verdict: response.Accepted}))
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
	report := verify.Compare(
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
	hard := verify.Compare(
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
			report := verify.Compare(
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
	report := verify.Compare(
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
	report := verify.Compare(
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
		report := verify.Compare(
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

	report := verify.Compare(promised(t, 1, unapplied),
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
