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
	writer := manifestjson.New(file, "0.2")
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
		Entities: []manifest.Entity{{Name: "customer", Rows: 1}},
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

// emptyManifest writes the manifest of a dataset of no rows with the real
// writer, which is what a recipe asking for none produces.
func emptyManifest(t *testing.T, into string) string {
	t.Helper()
	path := filepath.Join(into, "empty-manifest.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("cannot write the manifest this test reads: %v", err)
	}
	err = manifestjson.New(file, "0.2").Finish(manifest.Run{
		GeneratedAt: "2026-11-03T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", Entities: []manifest.Entity{{Name: "customer"}},
	}, 0, 0)
	if err != nil {
		t.Fatalf("the manifest did not close: %v", err)
	}
	_ = file.Close()
	return path
}

// 🔴 What an empty list of rows ends with, read through both real readers
// (DEC-271). A dataset of no rows agrees with it, a dataset with rows went
// missing whole, and a record that leaves the list out is refused as it always
// was. Until 2026-11-03 the first two ended with the code for a broken file as
// well, so a dataset of no rows could not be verified at all (B-338).
func TestAnEmptyListOfRowsEndsAsWhatItSays(t *testing.T) {
	const none = `{"spec_version":"0.2","rows":[]}`
	for _, given := range []struct {
		what     string
		manifest func(t *testing.T, into string) string
		response string
		want     int
		says     string
	}{
		{"a dataset of no rows answered by an empty list", emptyManifest, none, 0, "compared 0 entries"},
		{"a dataset of one row answered by an empty list", func(t *testing.T, into string) string {
			return manifestFor(t, into, catalog.Rejection, catalog.Hard)
		}, none, 4, "answers for 0, so 1 was never reacted to"},
		{"a dataset of no rows answered without a list", emptyManifest, `{"spec_version":"0.2"}`, 3,
			"carries no list of rows"},
	} {
		t.Run(given.what, func(t *testing.T) {
			into := t.TempDir()
			got := run(commandVerify, given.manifest(t, into), responseFor(t, into, given.response))
			if got.code != given.want || !strings.Contains(got.log, given.says) {
				t.Errorf("%s ended with %d, want %d, and should say %q. It said %q",
					given.what, got.code, given.want, given.says, got.log)
			}
		})
	}
}

// The detail of a finding is a sentence of its own, and a sentence begins with
// a capital.
//
// 🔴 The line read "accepted although it should have been rejected. the value
// was taken" (B-298), because the detail is written to follow other words.
func TestTheDetailOfAFindingBeginsItsOwnSentence(t *testing.T) {
	into := t.TempDir()
	got := run(commandVerify,
		manifestFor(t, into, catalog.Rejection, catalog.Hard),
		responseFor(t, into, `{"spec_version":"0.1","dataset":{"rows_total":1},
		 "rows":[{"row":1,"verdict":"accepted"}]}`))
	if !strings.Contains(got.log, "should have been rejected. The value was taken") {
		t.Errorf("the detail does not begin a sentence of its own. It said %q", got.log)
	}

	// A finding without a detail keeps its kind and gains no stray full stop.
	if said := followedBy("kind", ""); said != "kind" {
		t.Errorf("a finding without a detail reads %q, want %q", said, "kind")
	}
}

// manifestAtVersion writes a sound manifest that names some other version of
// the format, so that the version is the only thing under test.
func manifestAtVersion(t *testing.T, into, spec string) string {
	t.Helper()
	path := filepath.Join(into, "other-version.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("cannot write the manifest this test reads: %v", err)
	}
	writer := manifestjson.New(file, spec)
	writer.Add(manifest.Defect{
		ID:       "case",
		Location: manifest.Location{Entity: "customer", Row: 1, Field: "pesel"},
		Mutator:  "L3.checksum.off_by_one", Layer: catalog.Layer("L3"),
		Applied: true, ValueAfter: "sent", HasValueAfter: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the check digit no longer agrees",
	})
	err = writer.Finish(manifest.Run{
		GeneratedAt: "2026-08-17T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 1, RowsMutated: 1,
		Entities: []manifest.Entity{{Name: "customer", Rows: 1}},
	}, 1, 1)
	if err != nil {
		t.Fatalf("the manifest did not close: %v", err)
	}
	_ = file.Close()
	return path
}

// manifestOfTwo writes a manifest of a customer table and an order table of two
// rows each, with row 1 of the orders expected to be refused.
func manifestOfTwo(t *testing.T, into string) string {
	t.Helper()
	path := filepath.Join(into, "two-tables.json")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("cannot write the manifest this test reads: %v", err)
	}
	writer := manifestjson.New(file, "0.2")
	writer.Add(manifest.Defect{
		ID:       "case",
		Location: manifest.Location{Entity: "order", Row: 1, Field: "amount"},
		Mutator:  "L3.checksum.off_by_one", Layer: catalog.Layer("L3"),
		Applied: true, ValueAfter: "sent", HasValueAfter: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the check digit no longer agrees",
	})
	err = writer.Finish(manifest.Run{
		GeneratedAt: "2026-10-27T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 4, RowsMutated: 1,
		Entities: []manifest.Entity{{Name: "customer", Rows: 2}, {Name: "order", Rows: 2}},
	}, 1, 1)
	if err != nil {
		t.Fatalf("the manifest did not close: %v", err)
	}
	_ = file.Close()
	return path
}

// 🔴 A record about two tables is held to each of them, end to end (DEC-266,
// B-192).
//
// The finding names the table it is in, since row 1 alone could be in either.
// A record that leaves the entity out of a dataset of two is refused with the
// code for an input that is not valid, because comparing it would be guessing
// which table each row belongs to.
func TestARecordAboutTwoTablesIsHeldToEachOfThem(t *testing.T) {
	into := t.TempDir()
	twoTables := manifestOfTwo(t, into)

	got := run(commandVerify, twoTables, responseFor(t, into, `{"spec_version":"0.2","rows":[
	  {"entity":"customer","row":1,"verdict":"rejected"},{"entity":"customer","row":2,"verdict":"accepted"},
	  {"entity":"order","row":1,"verdict":"accepted"},{"entity":"order","row":2,"verdict":"accepted"}]}`))
	if got.code != int(codeExpectation) || !strings.Contains(got.log, "row 1 of order, amount: accepted") {
		t.Errorf("an order taken although it should have been refused ended with %d: %q", got.code, got.log)
	}

	got = run(commandVerify, twoTables, responseFor(t, into,
		`{"spec_version":"0.2","rows":[{"row":1,"verdict":"rejected"}]}`))
	if got.code != int(codeConfig) || !strings.Contains(got.log, "names no entity") {
		t.Errorf("a row naming no entity in a dataset of two ended with %d: %q", got.code, got.log)
	}
}

// A document from a version this build cannot read is refused rather than
// compared.
//
// 🔴 This is the sharpest thing measured on 2026-08-17. A complete response
// record differing ONLY in its version, claiming 99.0 or banana or a couple of
// spaces, ended with exit code nought and the sentence that the application had
// done what the manifest said. That is a verdict on somebody else's system
// drawn from a document this build cannot claim to understand, which is rule
// five of the project moved one step outwards, onto the document itself.
func TestADocumentFromAVersionThisBuildCannotReadIsRefused(t *testing.T) {
	const sound = `{"spec_version":"0.1","rows":[{"row":1,"verdict":"rejected","code":"E1"}]}`

	for _, given := range []struct {
		what     string
		manifest func(t *testing.T, into string) string
		response string
	}{
		{"a manifest from a later major version",
			func(t *testing.T, into string) string { return manifestAtVersion(t, into, "99.0") },
			sound},
		{"a response record from a later major version",
			func(t *testing.T, into string) string {
				return manifestFor(t, into, catalog.Rejection, catalog.Hard)
			},
			`{"spec_version":"99.0","rows":[{"row":1,"verdict":"rejected","code":"E1"}]}`},
		{"a response record whose version is a word",
			func(t *testing.T, into string) string {
				return manifestFor(t, into, catalog.Rejection, catalog.Hard)
			},
			`{"spec_version":"banana","rows":[{"row":1,"verdict":"rejected","code":"E1"}]}`},
	} {
		t.Run(given.what, func(t *testing.T) {
			into := t.TempDir()
			got := run(commandVerify, given.manifest(t, into), responseFor(t, into, given.response))
			if got.code != int(codeConfig) {
				t.Errorf("%s ended with %d, want %d. It said %q",
					given.what, got.code, codeConfig, got.log)
			}
			if !strings.Contains(got.log, "version") {
				t.Errorf("the refusal for %s does not say the version is the problem: %q",
					given.what, got.log)
			}
		})
	}
}

// A document from the same major line and a minor version this build has never
// seen is read, and the run says so.
//
// The format grows by adding, so such a document is readable and refusing it
// would make every upgrade a reason to keep old files unreadable. What must not
// happen is reading it in silence, because whatever the newer version added is
// invisible here by definition.
func TestADocumentFromANewerMinorVersionIsReadAndSaidSo(t *testing.T) {
	into := t.TempDir()
	got := run(commandVerify, manifestAtVersion(t, into, "0.9"),
		responseFor(t, into, `{"spec_version":"0.1","rows":[{"row":1,"verdict":"rejected","code":"E1"}]}`))

	if got.code != 0 {
		t.Fatalf("a readable manifest was refused with %d: %s", got.code, got.log)
	}
	if !strings.Contains(got.log, "0.9") {
		t.Errorf("the run does not say which version it read: %q", got.log)
	}
	if !strings.Contains(got.log, "anything the newer version added is not part of this comparison") {
		t.Errorf("the run does not say what reading it costs: %q", got.log)
	}

	// A response record of a newer minor version may carry members this build
	// has never heard of, and they are skipped rather than refused (DEC-266).
	got = run(commandVerify, manifestFor(t, into, catalog.Rejection, catalog.Hard),
		responseFor(t, into, `{"spec_version":"0.3","producer":"ci",`+
			`"rows":[{"row":1,"verdict":"rejected","code":"E1","latency_ms":12}]}`))
	if got.code != 0 || !strings.Contains(got.log, "the response record is written to version 0.3") {
		t.Errorf("a response record of a newer minor version ended with %d: %q", got.code, got.log)
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

// A file past the size this build reads back is refused with a move the reader
// can actually make.
//
// 🔴 Both halves of this were live on 2026-09-03 and both are the family B-206
// named. The manifest carried no move of its own, so it inherited the one for a
// Data failure: change the file or the flags describing it. The file is one a
// run wrote and no flag decides how large a manifest comes out, so the sentence
// sent the reader looking for a control that does not exist. The response
// record was worse. It carried a move inside its message and then took the
// inherited one on top, so the reader was told to split the run and, on the
// next line, to change the flags describing it.
//
// The files here are made by length rather than by content, because the
// refusal is decided on the length and never reads them.
func TestAFileTooLargeToReadBackIsRefusedWithAMoveThatCanBeMade(t *testing.T) {
	for _, given := range []struct {
		what     string
		oversize func(t *testing.T, into string) (manifest, response string)
		saying   string
	}{
		{"a manifest", func(t *testing.T, into string) (string, string) {
			return tooLarge(t, into, "manifest.json"),
				responseFor(t, into, `{"spec_version":"0.1","rows":[{"row":1,"verdict":"rejected"}]}`)
		}, "fewer columns"},
		{"a response record", func(t *testing.T, into string) (string, string) {
			return manifestFor(t, into, catalog.Rejection, catalog.Hard),
				tooLarge(t, into, "response.json")
		}, "smaller datasets"},
	} {
		t.Run(given.what, func(t *testing.T) {
			into := t.TempDir()
			manifest, response := given.oversize(t, into)
			got := run(commandVerify, manifest, response)

			if got.code != int(codeData) {
				t.Fatalf("%s over the limit ended with %d, want %d. It said %q",
					given.what, got.code, codeData, got.log)
			}
			if strings.Contains(got.log, doAboutDataCorrupt) {
				t.Errorf("the refusal for %s inherited the move for its kind, which points at "+
					"flags that do not decide a file's size: %q", given.what, got.log)
			}
			if !strings.Contains(got.log, given.saying) {
				t.Errorf("the refusal for %s carries no move the reader can make: %q",
					given.what, got.log)
			}
		})
	}
}

// tooLarge writes a file one byte past what this build reads back, by length
// rather than by content, and hands back its path.
func tooLarge(t *testing.T, into, name string) string {
	t.Helper()
	path := filepath.Join(into, name)
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("cannot make the oversized file this test needs: %v", err)
	}
	defer func() { _ = file.Close() }()
	if err := file.Truncate(manifestjson.MaxSize + 1); err != nil {
		t.Fatalf("cannot give the oversized file its length: %v", err)
	}
	return path
}
