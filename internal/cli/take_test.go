package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// somebodyElsesFile writes a file this tool did not make, in a directory of its
// own, and returns the path.
//
// The identifiers in it are built by the providers rather than typed out. A
// correct identifier written into a test is a correct identifier nobody checked,
// and this whole verb turns on whether a validator recognises a column.
func somebodyElsesFile(t *testing.T, rows int) string {
	t.Helper()
	source := values.New("pl-PL", "take-test")
	lines := []string{"id,first_name,pesel,aba,ssn,note"}
	for row := 0; row < rows; row++ {
		lines = append(lines, strings.Join([]string{
			builtValue(t, source, values.Pesel, row)[:6],
			builtValue(t, source, values.FirstName, row),
			builtValue(t, source, values.Pesel, row),
			builtValue(t, source, values.ABA, row),
			builtValue(t, source, values.SSN, row),
			"a note",
		}, ","))
	}
	made := t.TempDir()
	path := filepath.Join(made, "contacts.csv")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("cannot write the file this test damages: %v", err)
	}
	return path
}

func builtValue(t *testing.T, source values.Source, generator string, row int) string {
	t.Helper()
	value, err := source.Produce(generate.Request{
		Generator: generator,
		At:        seed.Coordinate{Entity: "contacts", Row: int64(row), Field: generator, Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("cannot produce a %s: %v", generator, err)
	}
	return value
}

// brokenFile writes a file this build will refuse to read, so that the refusals
// coming up from below can be exercised as well as the ones written by hand.
func brokenFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "given.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write the file this test refuses: %v", err)
	}
	return path
}

// damaged runs the verb into a directory of its own and reads what came out.
func damaged(t *testing.T, source string, extra ...string) (map[string]any, outcome, string) {
	t.Helper()
	into := t.TempDir()
	got := run(append([]string{commandCorrupt, "--out", into}, append(extra, source)...)...)
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}
	raw, err := os.ReadFile(filepath.Join(into, manifestjson.Name))
	if err != nil {
		t.Fatalf("the run left no manifest: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("the manifest does not parse: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(into, "contacts"+corruptedSuffix))
	if err != nil {
		t.Fatalf("the run left no damaged file: %v", err)
	}
	return document, got, string(written)
}

func TestAFileNobodyDescribedIsDamagedAndTheManifestSaysWhat(t *testing.T) {
	document, got, written := damaged(t, somebodyElsesFile(t, 40), "--layers", "L3", "--rows", "100%")

	envelope := document["run"].(map[string]any)
	if envelope["source"] != "contacts.csv" && !strings.HasSuffix(envelope["source"].(string), "contacts.csv") {
		t.Errorf("the envelope names the source as %v", envelope["source"])
	}
	if envelope["rows_total"].(float64) != 40 {
		t.Errorf("40 rows went in and the envelope says %v", envelope["rows_total"])
	}
	if envelope["defects_applied"].(float64) < 1 {
		t.Fatalf("nothing at all was damaged, and the log said %q", got.log)
	}
	if lines := strings.Count(written, "\n"); lines != 41 {
		t.Errorf("41 lines were expected in the output and there are %d", lines)
	}
}

// 🔴 What the run guessed travels with the data, per column and with its
// evidence. Everything else in a manifest is something this tool did, and this
// is the one thing it worked out about data it did not produce.
func TestTheManifestCarriesTheEvidenceForEveryColumn(t *testing.T) {
	document, _, _ := damaged(t, somebodyElsesFile(t, 40), "--layers", "L3")

	envelope := document["run"].(map[string]any)
	inference, carried := envelope["schema_inference"].(map[string]any)
	if !carried {
		t.Fatal("the run guessed what the columns hold and the manifest does not say so")
	}
	if inference["method"] != inferenceMethod {
		t.Errorf("the method is recorded as %v", inference["method"])
	}
	columns := inference["columns"].([]any)
	if len(columns) != 6 {
		t.Fatalf("6 columns went in and %d were reported", len(columns))
	}
	byField := map[string]map[string]any{}
	for _, column := range columns {
		entry := column.(map[string]any)
		byField[entry["field"].(string)] = entry
		if entry["basis"] == "" || entry["basis"] == nil {
			t.Errorf("the conclusion about %v has no evidence behind it", entry["field"])
		}
	}
	if byField["pesel"]["numbering_scheme"] != "PL.pesel" {
		t.Errorf("the column of PESEL values was taken for %v", byField["pesel"]["numbering_scheme"])
	}
	if byField["aba"]["numbering_scheme"] != "US.aba" {
		t.Errorf("the column of routing numbers was taken for %v", byField["aba"]["numbering_scheme"])
	}
	// The expensive half of DEC-156, end to end: a validator answering about
	// shape must not name a column, even when the values really are of that
	// scheme.
	if scheme, named := byField["ssn"]["numbering_scheme"]; named && scheme != "" {
		t.Errorf("a scheme with no check digit was read off values: %v", scheme)
	}
	if len(inference["not_inferred"].([]any)) == 0 {
		t.Error("the manifest says what it recognised and not what it never reads, " +
			"so it reads as a complete account of the file")
	}
}

// What a file cannot say is refused per column, with the reason, rather than
// left as damage that silently did not happen.
func TestDamageNeedingSomethingAFileCannotSayIsRefusedWithAReason(t *testing.T) {
	document, _, _ := damaged(t, somebodyElsesFile(t, 40),
		"--layers", "L2,L4", "--rows", "100%")

	wanted := map[string]bool{
		"L2.string.empty":                       false,
		"L2.string.exceeds_maxlength":           false,
		"L4.coherence.national_id_vs_gender":    false,
		"L4.coherence.national_id_vs_birthdate": false,
	}
	for _, entry := range document["defects"].([]any) {
		defect := entry.(map[string]any)
		name, _ := defect["mutator"].(string)
		if _, watched := wanted[name]; !watched {
			continue
		}
		wanted[name] = true
		if defect["applied"].(bool) {
			t.Errorf("%s was applied, and nothing in a file declares what it needs", name)
		}
		if defect["failure_kind"] != "run_lacks_capability" {
			t.Errorf("%s was refused as %v rather than for the capability it wanted",
				name, defect["failure_kind"])
		}
		if defect["failure_reason"] == "" {
			t.Errorf("%s was refused without saying why", name)
		}
	}
	for name, seen := range wanted {
		if !seen {
			t.Errorf("%s was never ordered, so nothing said it could not happen", name)
		}
	}
}

// Only the ordered damage changes. Everything else comes back as it went in,
// which is what makes the manifest a description of the difference rather than
// a description of some of it.
func TestNothingChangesExceptWhatTheManifestNames(t *testing.T) {
	source := somebodyElsesFile(t, 40)
	before, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("cannot read the source back: %v", err)
	}
	document, _, written := damaged(t, source, "--layers", "L3", "--rows", "100%")

	wasLine := strings.Split(strings.TrimRight(string(before), "\n"), "\n")
	nowLine := strings.Split(strings.TrimRight(written, "\n"), "\n")
	if len(wasLine) != len(nowLine) {
		t.Fatalf("%d lines went in and %d came out", len(wasLine), len(nowLine))
	}
	changed := map[int]bool{}
	for at := range wasLine {
		if wasLine[at] != nowLine[at] {
			changed[at] = true
		}
	}
	named := map[int]bool{}
	for _, entry := range document["defects"].([]any) {
		defect := entry.(map[string]any)
		if !defect["applied"].(bool) {
			continue
		}
		where := defect["location"].(map[string]any)
		named[int(where["row"].(float64))] = true
	}
	for at := range changed {
		if !named[at] {
			t.Errorf("line %d changed and no manifest entry names that row", at)
		}
	}
	if len(changed) == 0 {
		t.Error("nothing changed at all, so this test proved nothing")
	}
}

// A run damaging nothing still leaves a manifest, because damage that did not
// happen is the answer rather than the absence of one.
func TestARunThatDamagesNothingStillLeavesAManifest(t *testing.T) {
	document, _, _ := damaged(t, somebodyElsesFile(t, 40), "--layers", "L1", "--rows", "100%")
	if len(document["defects"].([]any)) == 0 {
		t.Fatal("a layer this build implements nothing in left no entries at all")
	}
	if document["run"].(map[string]any)["defects_applied"].(float64) != 0 {
		t.Error("this build implements damage at L1 now, so this test needs another layer")
	}
}

// The dry run says what was worked out and writes nothing. It is the first
// phase without the second rather than a separate estimator.
func TestADryRunSaysWhatItWorkedOutAndWritesNothing(t *testing.T) {
	into := t.TempDir()
	got := run(commandCorrupt, "--out", into, "--layers", "L3", "--dry-run",
		somebodyElsesFile(t, 40))
	if got.code != 0 {
		t.Fatalf("a dry run ended with %d and said %q", got.code, got.log)
	}
	left, err := os.ReadDir(into)
	if err != nil {
		t.Fatalf("cannot look at the output directory: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("a dry run left %d files behind", len(left))
	}
	for _, wanted := range []string{"PL.pesel", "would write"} {
		if !strings.Contains(got.log, wanted) {
			t.Errorf("a dry run did not say %q, and said %q", wanted, got.log)
		}
	}
}

// Every refusal of this verb has the four parts and never sends the reader to a
// recipe, because this is the verb that has none.
//
// 🔴 The list covers both roads a refusal takes, and it did not at first. The
// refusals written out by hand here are one road. The ones that come up from
// below carry only a kind, and the sentence saying what to do about a kind used
// to name a recipe for every verb, because every verb had one when it was
// written. A list of hand written refusals alone left that road untested, and a
// mutation turning the second sentence back to the old one went unnoticed.
func TestEveryRefusalOfThisVerbIsCompleteAndMentionsNoRecipe(t *testing.T) {
	source := somebodyElsesFile(t, 40)
	for _, given := range []struct {
		what string
		args []string
	}{
		{"no layers", []string{commandCorrupt, source}},
		{"a layer nobody describes", []string{commandCorrupt, "--layers", "L9", source}},
		{"a share nobody can read", []string{commandCorrupt, "--layers", "L3", "--rows", "banana", source}},
		// A share that is shaped like a percentage and is not a number. It
		// reached the engine until this line existed, where it chose no row at
		// all and the run ended with success.
		{"a share that is not a number",
			[]string{commandCorrupt, "--layers", "L3", "--rows", "NaN%", source}},
		// Asked before the file is opened, so that a typo in the form costs a
		// refusal rather than a pass over a file the user chose the size of.
		{"a manifest form this build does not write",
			[]string{commandCorrupt, "--layers", "L3", "--manifest", "markdown", source}},
		{"a per row setting nobody knows", []string{commandCorrupt, "--layers", "L3", "--per-row", "two", source}},
		{"a delimiter of two characters", []string{commandCorrupt, "--layers", "L3", "--delimiter", "||", source}},
		{"no file at all", []string{commandCorrupt, "--layers", "L3"}},
		{"two files", []string{commandCorrupt, "--layers", "L3", source, source}},
		// These three come up from below carrying a kind rather than being
		// written out here, which is the road the two above do not travel.
		{"a file whose quoting is broken",
			[]string{commandCorrupt, "--layers", "L3", brokenFile(t, "a,b\n1,\"unclosed\n")}},
		{"a file of headings alone",
			[]string{commandCorrupt, "--layers", "L3", brokenFile(t, "a,b\n")}},
		{"a file that is not there",
			[]string{commandCorrupt, "--layers", "L3", filepath.Join(t.TempDir(), "absent.csv")}},
	} {
		// 🔴 Every case writes into a directory of its own, and that is not
		// tidiness. Without it these runs write into the package directory,
		// where the file one case left made the next one fail for a completely
		// different reason: the store refuses an existing file. A mutation
		// removing the refusal being tested here then looked as though it had
		// been caught, because something else refused instead. Measured.
		into := t.TempDir()
		got := run(append([]string{given.args[0], "--out", into}, given.args[1:]...)...)
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
		if left, err := os.ReadDir(into); err == nil && len(left) > 0 {
			t.Errorf("the run refused for %s and still left %d files behind", given.what, len(left))
		}
	}
}

// 🔴 The one thing this verb could do that nothing can take back: write the
// damaged records over the records they were made from.
//
// The run below points the output at the very directory the source sits in,
// which is the ordinary way somebody would type this. It has to succeed and
// leave the source exactly as it was, and both halves matter. The suffix on the
// output name is what makes that true, so a build that dropped the suffix fails
// this test twice over: the run is refused by the guard in front of the
// irreversible thing, and the source would otherwise have been replaced.
func TestDamagingAFileInItsOwnDirectoryLeavesTheSourceAlone(t *testing.T) {
	made := t.TempDir()
	source := filepath.Join(made, "contacts.csv")
	original := "id,note\n1,first\n2,second\n3,third\n"
	if err := os.WriteFile(source, []byte(original), 0o600); err != nil {
		t.Fatalf("cannot write the source: %v", err)
	}

	got := run(commandCorrupt, "--out", made, "--layers", "L3", source)
	if got.code != 0 {
		t.Fatalf("damaging a file where it sits was refused with %d: %q", got.code, got.log)
	}
	back, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("the source is gone: %v", err)
	}
	if string(back) != original {
		t.Errorf("the file that was read came back as %q rather than as %q", string(back), original)
	}
	if _, err := os.Stat(filepath.Join(made, "contacts"+corruptedSuffix)); err != nil {
		t.Errorf("the damaged records were not written beside the source: %v", err)
	}
}

// The file this build refuses to read is refused before anything is written, so
// the output directory is left as it was found.
func TestAFileThatCannotBeReadLeavesNothingBehind(t *testing.T) {
	made := t.TempDir()
	source := filepath.Join(made, "broken.csv")
	if err := os.WriteFile(source, []byte("a,b\n1,\"unclosed\n"), 0o600); err != nil {
		t.Fatalf("cannot write the source: %v", err)
	}
	into := t.TempDir()
	if got := run(commandCorrupt, "--out", into, "--layers", "L3", source); got.code == 0 {
		t.Fatal("a file with broken quoting was damaged as though it were fine")
	}
	left, err := os.ReadDir(into)
	if err != nil {
		t.Fatalf("cannot look at the output directory: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("a run that refused the file left %d files behind", len(left))
	}
}

// The same file and the same seed give the same damage, which is what makes
// this usable in a pipeline at all.
func TestTheSameFileAndSeedGiveTheSameDamage(t *testing.T) {
	source := somebodyElsesFile(t, 40)
	_, _, first := damaged(t, source, "--layers", "L3", "--rows", "100%", "--seed", "repeat")
	_, _, second := damaged(t, source, "--layers", "L3", "--rows", "100%", "--seed", "repeat")
	if first != second {
		t.Error("two runs of one file and one seed produced different bytes")
	}
	_, _, other := damaged(t, source, "--layers", "L3", "--rows", "100%", "--seed", "different")
	if first == other {
		t.Error("two different seeds produced the same damage, so the seed decides nothing")
	}
}

// The providers are named here so that a build dropping one fails this file
// rather than quietly recognising fewer columns.
func TestTheSchemesThisVerbLeansOnAreStillInThisBuild(t *testing.T) {
	if (pl.PESEL{}).ID() != "PL.pesel" || (us.ABA{}).ID() != "US.aba" {
		t.Fatal("the schemes these tests recognise columns by have been renamed")
	}
}
