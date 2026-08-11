package catalogyaml_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/guard"
)

// The published examples are test cases, not illustrations.
//
// Every entry carries a before and after pair, and the architecture turns those
// pairs into tests so that the specification cannot drift away from the
// behaviour. Most of them cannot be produced and compared, because this build
// implements two of the fifty two kinds of damage described. What can be done
// for the rest is the half that needs only the providers: where an example is an
// identifier of a scheme this build can compute, the values are checked against
// that scheme.
//
// Where there is an implementation, the whole pair runs instead, beside the
// mutator rather than here. L3.checksum.off_by_one is the first of those, and
// its test produces the after value from the before one.
//
// The first run of this file found three entries with bad values, and two of
// them contradicted their own note. That is worth more than the values it
// fixed, because a pair taken as a test case teaches the engine whatever the
// pair happens to say.
//
// # Why each claim is written out rather than derived
//
// An entry does not record which scheme its example belongs to. applies_to
// names every provider the mutator is meaningful for, up to ten of them, and
// the example is one value from one of those schemes with nothing saying which.
// So the scheme is named here, per entry, and a check below makes sure this
// list cannot fall behind the catalog.

// providersThisBuildHas are the schemes with an implementation to check against.
var providersThisBuildHas = map[string]bool{
	"PL.pesel": true, "PL.nip": true, "PL.regon": true,
	"US.ssn": true, "US.aba": true, "US.postal": true, "iban": true,
}

// claim is what one entry's example has to satisfy.
type claim struct {
	// unchecked, when set, is why the values cannot be checked at all. It is
	// a sentence rather than a silence, because an entry nobody checks and
	// an entry nobody could check look the same from outside.
	unchecked string
	check     func(t *testing.T, e catalog.Entry)
}

var claims = map[catalog.ID]claim{
	"L3.checksum.off_by_one": {check: func(t *testing.T, e catalog.Entry) {
		requireValidNIP(t, "before", e.Example.Before)
		requireInvalidNIP(t, "after", e.Example.After)
	}},
	"L3.checksum.transposed_digits": {check: func(t *testing.T, e catalog.Entry) {
		requireValidNIP(t, "before", e.Example.Before)
		requireInvalidNIP(t, "after", e.Example.After)
	}},

	// The note promises that the check digit is recomputed so the value fails
	// only on the date, which is what separates this from a checksum mutator.
	// A value that also breaks the check digit is rejected by any validator
	// before the date is ever looked at, so the mutator would be testing
	// nothing it claims to test.
	"L3.national_id.impossible_date": {check: func(t *testing.T, e catalog.Entry) {
		requireValidPESEL(t, "before", e.Example.Before)
		after := e.Example.After
		if repaired, err := (pl.PESEL{}).WithCheckDigits(after); err != nil || repaired != after {
			t.Errorf("after %q does not carry the check digit its own digits require (%q), "+
				"so it fails on the checksum as well as the date, "+
				"and the note says it fails only on the date", after, repaired)
		}
		if _, ok := (pl.PESEL{}).EncodedDate(after); ok {
			t.Errorf("after %q encodes a date that exists, and the entry is about one that does not", after)
		}
	}},

	// Both values are correct on their own and only the pair is wrong, which
	// is what puts this at L4 rather than L3.
	"L4.coherence.national_id_vs_birthdate": {check: func(t *testing.T, e catalog.Entry) {
		requireValidPESEL(t, "before", e.Example.Before)
		requireValidPESEL(t, "after", e.Example.After)
		beforeDate, okBefore := (pl.PESEL{}).EncodedDate(e.Example.Before)
		afterDate, okAfter := (pl.PESEL{}).EncodedDate(e.Example.After)
		if okBefore && okAfter && beforeDate == afterDate {
			t.Errorf("both values encode %v, so the pair shows no disagreement about a birth date",
				beforeDate)
		}
	}},

	// Same shape, about the other thing PESEL encodes.
	"L4.coherence.national_id_vs_gender": {check: func(t *testing.T, e catalog.Entry) {
		requireValidPESEL(t, "before", e.Example.Before)
		requireValidPESEL(t, "after", e.Example.After)
		beforeGender, okBefore := (pl.PESEL{}).EncodedGender(e.Example.Before)
		afterGender, okAfter := (pl.PESEL{}).EncodedGender(e.Example.After)
		if okBefore && okAfter && beforeGender == afterGender {
			t.Errorf("both values encode %v, so the pair shows no disagreement about gender",
				beforeGender)
		}
	}},

	// The applicability condition says the base and the target sit in
	// different never issued sub ranges, and that the tool guarantees this
	// rather than checking for it.
	"L3.range.unassigned": {check: func(t *testing.T, e catalog.Entry) {
		if !(us.SSN{}).InReservedRange(e.Example.Before) {
			t.Errorf("before %q is not in a range that is never issued, "+
				"and the condition says the base value sits in one", e.Example.Before)
		}
		if !(us.SSN{}).InReservedRange(e.Example.After) {
			t.Errorf("after %q is not in a range that is never issued, "+
				"which is the whole of what this mutator does", e.Example.After)
		}
		// Different sub ranges rather than merely different values. Two
		// numbers out of one never issued block satisfy "not the same" and
		// show nothing, since the mutator exists to move a value from the
		// block our own generator uses into another one.
		if area(e.Example.Before) == area(e.Example.After) {
			t.Errorf("before %q and after %q sit in the same block %q, "+
				"and the condition says they must sit in different ones",
				e.Example.Before, e.Example.After, area(e.Example.Before))
		}
	}},

	// The reverse direction: a value that looks entirely ordinary is replaced
	// by one from a range that is never issued.
	"REV.identifier.reserved_range": {check: func(t *testing.T, e catalog.Entry) {
		if (us.SSN{}).InReservedRange(e.Example.Before) {
			t.Errorf("before %q already sits in a range that is never issued, "+
				"so the entry shows no change", e.Example.Before)
		}
		if !(us.SSN{}).InReservedRange(e.Example.After) {
			t.Errorf("after %q is not in a range that is never issued", e.Example.After)
		}
	}},

	"L4.coherence.postal_vs_city": {
		unchecked: "the example is a Polish pair, and there is no PL.postal in this build. " +
			"Where the Polish postal registry comes from is an open question, because the " +
			"only complete source claims rights to it",
	},
}

func TestTheExamplesSatisfyWhatTheirEntriesClaim(t *testing.T) {
	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for id, one := range claims {
		entry, ok := loaded.Lookup(id)
		if !ok {
			t.Errorf("%s has a claim here and no entry in the catalog", id)
			continue
		}
		if one.unchecked != "" {
			t.Logf("%s: not checked, because %s", id, one.unchecked)
			continue
		}
		t.Run(string(id), func(t *testing.T) { one.check(t, entry) })
	}
}

// The list above is written by hand, so it can fall behind the catalog. This is
// the half that stops that: an entry naming a scheme this build can compute,
// with nobody saying what its example has to satisfy, fails here and asks for a
// claim rather than being quietly unexamined.
func TestNoEntryOfAKnownSchemeGoesUnclaimed(t *testing.T) {
	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, entry := range loaded.Entries() {
		if _, claimed := claims[entry.ID]; claimed {
			continue
		}
		for _, named := range entry.AppliesTo.Providers {
			if providersThisBuildHas[named] {
				t.Errorf("%s applies to %s, which this build can compute, "+
					"and nothing here says what its example must satisfy", entry.ID, named)
				break
			}
		}
	}
}

// deliberatelyDamaged are the values in the published manifest example that are
// meant to be invalid, because a manifest describes damage and a damaged value
// is the point of the entry carrying it.
//
// The list is short and closed in both directions below, so it cannot quietly
// grow into a way of excusing whatever happens to be wrong.
var deliberatelyDamaged = map[string]string{
	"85011412346": "the result of L3.checksum.off_by_one, which is invalid by construction",
}

// The published manifest example carries national identifiers, and they are
// read by whoever implements this format.
//
// This is here because the same defect was found in three places on one day.
// The catalog had it, the internal example had it, and this file had it: a
// value presented as correct that no validator would accept, and an entry
// saying two values are each right on their own while one of them was not. Two
// of those three are prose nothing checks. This one ships, so it gets a check.
func TestTheIdentifiersInThePublishedManifestExampleAreSound(t *testing.T) {
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	path := filepath.Join(root, "spec", "expectation-manifest.md")
	published, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read spec/expectation-manifest.md, so it is unchecked rather than sound: %v", err)
	}

	found := map[string]bool{}
	for _, value := range regexp.MustCompile(`\b\d{11}\b`).FindAllString(string(published), -1) {
		found[value] = true
		if why, damaged := deliberatelyDamaged[value]; damaged {
			t.Logf("%s is invalid on purpose: %s", value, why)
			continue
		}
		if verdict := (pl.PESEL{}).Validate(value); !verdict.Valid {
			t.Errorf("the manifest example carries %q as a sound value, and it is not one: %s",
				value, verdict.Reason)
		}
	}
	if len(found) == 0 {
		t.Fatal("no identifier was found in the example at all, so this check read nothing")
	}
	for value := range deliberatelyDamaged {
		if !found[value] {
			t.Errorf("%q is excused here and no longer appears in the example", value)
		}
	}
}

// area is the first group of a social security number, which is what names the
// block a value belongs to.
func area(value string) string {
	digits := regexp.MustCompile(`\d`).FindAllString(value, 3)
	return strings.Join(digits, "")
}

func requireValidPESEL(t *testing.T, which, value string) {
	t.Helper()
	if verdict := (pl.PESEL{}).Validate(value); !verdict.Valid {
		t.Errorf("%s %q is not a valid PESEL: %s", which, value, verdict.Reason)
	}
}

func requireValidNIP(t *testing.T, which, value string) {
	t.Helper()
	if verdict := (pl.NIP{}).Validate(value); !verdict.Valid {
		t.Errorf("%s %q is not a valid NIP: %s", which, value, verdict.Reason)
	}
}

func requireInvalidNIP(t *testing.T, which, value string) {
	t.Helper()
	if verdict := (pl.NIP{}).Validate(value); verdict.Valid {
		t.Errorf("%s %q is a valid NIP, and the damage should have made it invalid", which, value)
	}
}
