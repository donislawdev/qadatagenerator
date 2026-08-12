package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

// manifestFor writes a manifest with the real writer, so that this test cannot
// pass against a file the tool would never produce.
func manifestFor(t *testing.T, into string, expectation catalog.Expectation,
	confidence catalog.Confidence) string {
	t.Helper()
	path := filepath.Join(into, "expectation-manifest.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("cannot write the manifest this test reads: %v", err)
	}
	writer := manifestjson.New(file, "0.1")
	writer.Add(manifest.Defect{
		ID:            "case",
		Location:      manifest.Location{Entity: "customer", Row: 1, Field: "pesel"},
		Mutator:       "L3.checksum.off_by_one",
		Layer:         catalog.Layer("L3"),
		Applied:       true,
		ValueAfter:    "sent",
		HasValueAfter: true,
		Expectation:   expectation,
		Confidence:    confidence,
		Rationale:     "the check digit no longer agrees",
	})
	err = writer.Finish(manifest.Run{
		GeneratedAt: "2026-08-12T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 1, RowsMutated: 1,
	}, 1, 1)
	if err != nil {
		t.Fatalf("the manifest did not close: %v", err)
	}
	_ = file.Close()
	return path
}

func responseFor(t *testing.T, into, content string) string {
	t.Helper()
	path := filepath.Join(into, "response.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write the response record: %v", err)
	}
	return path
}

// 🔴 The expectation code is returned by this verb and by nothing else, and it
// is returned for a hard disagreement only.
//
// The code has been in the table since the first day with nothing able to
// return it. What matters as much as returning it is not returning it for a
// soft disagreement, where the right reaction belongs to the application rather
// than to the data.
func TestOnlyAHardDisagreementReturnsTheExpectationCode(t *testing.T) {
	const took = `{"spec_version":"0.1","dataset":{"rows_total":1},
	 "rows":[{"row":1,"verdict":"accepted"}]}`
	const refused = `{"spec_version":"0.1","dataset":{"rows_total":1},
	 "rows":[{"row":1,"verdict":"rejected","code":"E1"}]}`

	for _, given := range []struct {
		what       string
		confidence catalog.Confidence
		response   string
		want       int
	}{
		{"a hard disagreement", catalog.Hard, took, 4},
		{"a soft disagreement", catalog.Soft, took, 0},
		{"the application did what it was told", catalog.Hard, refused, 0},
	} {
		t.Run(given.what, func(t *testing.T) {
			into := t.TempDir()
			got := run(commandVerify,
				manifestFor(t, into, catalog.Rejection, given.confidence),
				responseFor(t, into, given.response))
			if got.code != given.want {
				t.Errorf("%s ended with %d, want %d. It said %q",
					given.what, got.code, given.want, got.log)
			}
		})
	}
}

// Every refusal of this verb is complete and never sends the reader to a recipe
// it does not have.
//
// 🔴 The advice for a data fault used to name a recipe for every verb that was
// not the one verb listed as recipe free. This verb has no recipe either, and
// inherited advice about somebody else's file.
func TestEveryRefusalOfVerifyIsCompleteAndMentionsNoRecipe(t *testing.T) {
	into := t.TempDir()
	sound := manifestFor(t, into, catalog.Rejection, catalog.Hard)

	for _, given := range []struct {
		what string
		args []string
	}{
		{"no arguments", []string{commandVerify}},
		{"one argument", []string{commandVerify, sound}},
		{"a manifest that is not there",
			[]string{commandVerify, filepath.Join(into, "absent.json"), sound}},
		{"a response record that is not a record",
			[]string{commandVerify, sound, responseFor(t, into, "nonsense")}},
	} {
		got := run(given.args...)
		if got.code == 0 {
			t.Errorf("%s was accepted", given.what)
			continue
		}
		if strings.Contains(got.log, "recipe") {
			t.Errorf("the refusal for %s sends the reader to a recipe they do not have: %q",
				given.what, got.log)
		}
		if !strings.Contains(got.log, "try:") {
			t.Errorf("the refusal for %s does not say what to do instead: %q", given.what, got.log)
		}
	}
}

// The report says what it could not see, whatever else it found.
func TestTheReportAlwaysSaysWhatItCouldNotSee(t *testing.T) {
	into := t.TempDir()
	got := run(commandVerify,
		manifestFor(t, into, catalog.Rejection, catalog.Hard),
		responseFor(t, into, `{"spec_version":"0.1","dataset":{"rows_total":1},
		 "rows":[{"row":1,"verdict":"rejected"}]}`))

	if got.code != 0 {
		t.Fatalf("a run that agreed ended with %d: %q", got.code, got.log)
	}
	if !strings.Contains(got.log, "not settled here") {
		t.Errorf("a clean report does not say what it could not see: %q", got.log)
	}
}
