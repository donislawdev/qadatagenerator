package manifest_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

func run() manifest.Run {
	return manifest.Run{
		GeneratedAt:    "2026-08-05T10:14:22Z",
		Seed:           "release-2026-08",
		Locale:         "pl-PL",
		CatalogVersion: "0.1",
		RowsTotal:      13,
		RowsMutated:    1,
	}
}

// applied is an entry with nothing wrong with it, for a test to break in one
// place at a time.
func applied() manifest.Defect {
	return manifest.Defect{
		ID:             "d1",
		Location:       manifest.Location{Entity: "customer", Row: 1, Field: "nip"},
		Mutator:        "L3.checksum.off_by_one",
		Layer:          catalog.Domain,
		Applied:        true,
		ValueBefore:    "1234563218",
		HasValueBefore: true,
		ValueAfter:     "1234563219",
		HasValueAfter:  true,
		Expectation:    catalog.Rejection,
		Confidence:     catalog.Hard,
		Rationale:      "The format is still valid and the check digit no longer matches.",
	}
}

// refused is the other kind of entry, and the one this whole format exists for.
func refused() manifest.Defect {
	return manifest.Defect{
		ID:            "d2",
		Location:      manifest.Location{Entity: "customer", Row: 2, Field: "ssn"},
		Mutator:       "L3.checksum.off_by_one",
		Layer:         catalog.Domain,
		Applied:       false,
		Expectation:   catalog.Rejection,
		Confidence:    catalog.Hard,
		Rationale:     "The identifier carries no check digit, so there is none to break.",
		FailureReason: "US.ssn has no check digit",
	}
}

func TestAWholeManifestIsAccepted(t *testing.T) {
	built, problems := manifest.New("0.1", run(), []manifest.Defect{applied(), refused()})
	if len(problems) > 0 {
		t.Fatalf("a manifest with nothing wrong with it was refused: %v", problems)
	}
	if built.SpecVersion() != "0.1" {
		t.Errorf("format version is %q", built.SpecVersion())
	}
	if built.Run().Seed != "release-2026-08" {
		t.Error("the envelope did not survive")
	}
}

// The counts are derived rather than carried, so that the summary cannot
// disagree with the list under it. A summary saying forty were ordered while
// the list holds twenty seven is the exact lie this format exists to stop, and
// nobody would check it by hand.
func TestTheCountsComeFromTheEntriesAndNotFromWhoeverBuiltThem(t *testing.T) {
	// Two applied and one refused rather than one of each. With one of each,
	// a count of the applied and a count of the refused are the same number,
	// so this test passed against a build that counted the wrong ones. The
	// mutation that inverted the condition went unnoticed until the counts
	// were made to differ, which is why the second applied entry is here.
	second := applied()
	second.ID = "d3"
	second.Location.Field = "regon"

	built, problems := manifest.New("0.1", run(), []manifest.Defect{applied(), refused(), second})
	if len(problems) > 0 {
		t.Fatalf("refused: %v", problems)
	}
	if built.Requested() != 3 {
		t.Errorf("requested is %d, and three damages were ordered", built.Requested())
	}
	if built.Applied() != 2 {
		t.Errorf("applied is %d, and two of the three happened", built.Applied())
	}
	if built.Complete() {
		t.Error("a run with one damage refused calls itself complete")
	}
	if missing := built.Unapplied(); len(missing) != 1 || missing[0].ID != "d2" {
		t.Errorf("the unapplied entries are %v", missing)
	}
}

// An envelope claiming more mutated rows than the entries name would describe a
// dataset that is not the one beside it.
func TestAnEnvelopeCannotClaimMoreMutatedRowsThanTheEntriesShow(t *testing.T) {
	envelope := run()
	envelope.RowsMutated = 0
	_, problems := manifest.New("0.1", envelope, []manifest.Defect{applied()})
	requireSays(t, problems, "the entries name 1")
}

func TestAnEnvelopeCannotMutateMoreRowsThanItHas(t *testing.T) {
	envelope := run()
	envelope.RowsTotal = 1
	envelope.RowsMutated = 5
	_, problems := manifest.New("0.1", envelope, []manifest.Defect{})
	requireSays(t, problems, "says 5 of 1 rows were mutated")
}

func TestAnEnvelopeMissingWhatMakesARunReadableIsRefused(t *testing.T) {
	cases := map[string]func(*manifest.Run){
		"no timestamp":       func(r *manifest.Run) { r.GeneratedAt = "" },
		"no seed":            func(r *manifest.Run) { r.Seed = " " },
		"no locale":          func(r *manifest.Run) { r.Locale = "" },
		"no catalog version": func(r *manifest.Run) { r.CatalogVersion = "" },
	}
	for what, breakIt := range cases {
		t.Run(what, func(t *testing.T) {
			envelope := run()
			breakIt(&envelope)
			if _, problems := manifest.New("0.1", envelope, nil); len(problems) == 0 {
				t.Errorf("an envelope with %s was accepted", what)
			}
		})
	}
}

// This is the rule the product rests on. An entry that did not happen and does
// not say why is worse than no entry, because the manifest looks complete.
func TestAnEntryThatDidNotHappenHasToSayWhy(t *testing.T) {
	silent := refused()
	silent.FailureReason = " "
	_, problems := manifest.New("0.1", run(), []manifest.Defect{silent})
	requireSays(t, problems, "which is the silence this format exists to prevent")
}

// The other half of the same rule. Values on an entry that did not happen would
// describe a change nobody made.
func TestAnEntryThatDidNotHappenCannotCarryValues(t *testing.T) {
	lying := refused()
	lying.HasValueAfter = true
	lying.ValueAfter = "1234563219"
	_, problems := manifest.New("0.1", run(), []manifest.Defect{lying})
	requireSays(t, problems, "describes a change nobody made")
}

func TestAnEntryCannotBeAppliedAndFailedAtOnce(t *testing.T) {
	both := applied()
	both.FailureReason = "the field had no check digit"
	_, problems := manifest.New("0.1", run(), []manifest.Defect{both})
	requireSays(t, problems, "says both things at once")
}

func TestEveryWayAnEntryCanBeUnreadableIsRefused(t *testing.T) {
	cases := []struct {
		what    string
		breakIt func(*manifest.Defect)
		says    string
	}{
		{"no identifier", func(d *manifest.Defect) { d.ID = "" }, "no test can refer to it"},
		{"no mutator", func(d *manifest.Defect) { d.Mutator = " " }, "nothing says what was done"},
		{"unknown layer", func(d *manifest.Defect) { d.Layer = "L9" }, "not one this build knows"},
		{"unknown expectation",
			func(d *manifest.Defect) { d.Expectation = "shrug" }, "not one of rejection"},
		{"unknown confidence",
			func(d *manifest.Defect) { d.Confidence = "probably" }, "neither hard nor soft"},
		{"no rationale", func(d *manifest.Defect) { d.Rationale = "" }, "somebody who was not here"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			entry := applied()
			one.breakIt(&entry)
			_, problems := manifest.New("0.1", run(), []manifest.Defect{entry})
			requireSays(t, problems, one.says)
		})
	}
}

func TestTwoEntriesUnderOneIdentifierAreRefused(t *testing.T) {
	_, problems := manifest.New("0.1", run(), []manifest.Defect{applied(), applied()})
	requireSays(t, problems, "two entries carry this identifier")
}

// Each layer damages something different, so each needs a different place
// named. A byte offset says nothing about a rule spanning two records.
func TestALocationHasToSayEnoughForItsLayer(t *testing.T) {
	nowhere := manifest.Location{}
	cases := []struct {
		what     string
		layer    catalog.Layer
		location manifest.Location
		accepted bool
	}{
		{"damage to a whole file names no place", catalog.Bytes, nowhere, true},
		{"a container defect naming a row", catalog.Container,
			manifest.Location{Row: 4}, true},
		{"a container defect naming a path", catalog.Container,
			manifest.Location{Path: "$.items[2]"}, true},
		{"a container defect naming nothing", catalog.Container, nowhere, false},
		{"a value defect naming entity, row and field", catalog.FieldShape,
			manifest.Location{Entity: "customer", Row: 1, Field: "nip"}, true},
		{"a value defect naming only a row", catalog.FieldShape,
			manifest.Location{Row: 1}, false},
		{"a cross cutting module names a place the same way", catalog.Reverse,
			manifest.Location{Entity: "customer", Row: 1, Field: "email"}, true},
		{"set integrity with the records it spans", catalog.SetLevel,
			manifest.Location{Entity: "customer", Row: 1, Field: "pesel",
				Related: []manifest.Location{{Entity: "customer", Row: 7, Field: "pesel"}}}, true},
		{"set integrity naming one record only", catalog.SetLevel,
			manifest.Location{Entity: "customer", Row: 1, Field: "pesel"}, false},
	}

	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			entry := applied()
			entry.Layer = one.layer
			entry.Location = one.location
			_, problems := manifest.New("0.1", run(), []manifest.Defect{entry})
			if one.accepted && len(problems) > 0 {
				t.Errorf("refused: %v", problems)
			}
			if !one.accepted && len(problems) == 0 {
				t.Error("accepted, and the location does not say enough for this layer")
			}
		})
	}
}

// A manifest is a promise, and a caller that could reach into it after it was
// made could change what was promised.
func TestTheEntriesCannotBeChangedThroughWhatTheCallerIsHanded(t *testing.T) {
	built, problems := manifest.New("0.1", run(), []manifest.Defect{applied()})
	if len(problems) > 0 {
		t.Fatalf("refused: %v", problems)
	}
	entries := built.Defects()
	entries[0].Rationale = "something else entirely"
	if built.Defects()[0].Rationale == "something else entirely" {
		t.Error("a caller can rewrite the manifest through the slice it was handed")
	}
}

func TestAManifestWithAnyProblemIsNotHandedBack(t *testing.T) {
	silent := refused()
	silent.FailureReason = ""
	built, problems := manifest.New("0.1", run(), []manifest.Defect{applied(), silent})
	if len(problems) == 0 {
		t.Fatal("a manifest with a silent entry was accepted")
	}
	if built.Requested() != 0 {
		t.Errorf("%d entries came back with the problems", built.Requested())
	}
}

// A run that damaged nothing is a real answer. A policy asking for a layer no
// mutator covers produces exactly this, and it has to be expressible.
func TestARunThatDamagedNothingIsAManifestRatherThanAnError(t *testing.T) {
	envelope := run()
	envelope.RowsMutated = 0
	built, problems := manifest.New("0.1", envelope, nil)
	if len(problems) > 0 {
		t.Fatalf("a run with no damage was refused: %v", problems)
	}
	if !built.Complete() {
		t.Error("a run that was asked for nothing and did nothing is not complete")
	}
	if built.Requested() != 0 || built.Applied() != 0 {
		t.Errorf("requested %d applied %d", built.Requested(), built.Applied())
	}
}

func requireSays(t *testing.T, problems []string, want string) {
	t.Helper()
	if len(problems) == 0 {
		t.Fatalf("this was accepted, and it should have been refused for %q", want)
	}
	if !strings.Contains(strings.Join(problems, "\n"), want) {
		t.Errorf("the refusal does not say %q: %v", want, problems)
	}
}
