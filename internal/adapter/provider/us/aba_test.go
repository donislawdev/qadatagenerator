package us_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

func abaRequest(row int64) provider.Request {
	return provider.Request{
		Source: seed.New("release-2026-08"),
		At:     seed.Coordinate{Entity: "account", Row: row, Field: "routing_number", Purpose: "value"},
	}
}

func generateABA(t *testing.T, row int64) string {
	t.Helper()
	value, err := us.ABA{}.Generate(abaRequest(row))
	if err != nil {
		t.Fatalf("row %d: %v", row, err)
	}
	return value
}

// The contract, checked in both directions. A declared capability without its
// interface fails, and an interface without its declaration fails too.
func TestTheRoutingNumberProviderSatisfiesTheContract(t *testing.T) {
	if findings := provider.Check(us.ABA{}); len(findings) != 0 {
		t.Fatalf("the provider breaks its contract: %v", findings)
	}
}

// # The absence that carries the ethics of this scheme
//
// identifier.reserved-range means the scheme has a harbour, a block nothing is
// ever issued from, which pillar one generates inside by default. This scheme
// has none: the registrar marks series "Reserved for future use" and says their
// use "will be determined by the Board", which is stock rather than sanctuary.
//
// Not declaring it is the machine readable way of saying so, and a report
// reading the registry learns from it that a dataset carrying these numbers has
// to be marked synthetic. So the absence is asserted rather than left to be
// noticed.
func TestTheRoutingNumberClaimsNoSafeHarbour(t *testing.T) {
	for _, declared := range (us.ABA{}).Capabilities() {
		if declared == capability.IdentifierReservedRange {
			t.Fatal("the provider claims a harbour, and this scheme has none. " +
				"Every series the registrar leaves spare is stock it says the Board may hand out.")
		}
	}
	// The write side is declared, because moving a value into a spare series is
	// a different question and one this scheme can answer.
	var found bool
	for _, declared := range (us.ABA{}).Capabilities() {
		if declared == capability.IdentifierUnassignedTarget {
			found = true
		}
	}
	if !found {
		t.Fatal("the provider does not offer the blocks a corruption moves a value into")
	}
}

// Every generated value has to satisfy the validator written from the
// registrar's own sentences, which states the check digit rule as a property of
// the whole number rather than as an arithmetic producing the last digit.
func TestEveryGeneratedRoutingNumberSatisfiesTheIndependentValidator(t *testing.T) {
	for row := int64(0); row < 2000; row++ {
		value := generateABA(t, row)
		if findings := oracle.CheckABA(value); len(findings) != 0 {
			t.Fatalf("row %d produced %s, which the independent validator refuses: %v", row, value, findings)
		}
		if verdict := (us.ABA{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %s, which its own validator refuses: %s", row, value, verdict.Reason)
		}
	}
}

// # The ethical rule of this provider, and it is the reverse of the SSN one
//
// This scheme has no harbour, so correct values sit in the series the policy
// assigns to banks. Every one of them could belong to a real institution, which
// is why the dataset has to be marked synthetic elsewhere. What must never
// happen is generating inside a spare series: that would be picking a block
// because it looks free, which is the mistake this project made once already.
func TestEveryGeneratedRoutingNumberSitsInAnIssuedSeries(t *testing.T) {
	for row := int64(0); row < 2000; row++ {
		value := generateABA(t, row)
		if !oracle.ABAIssuedToInstitutions(value) {
			use, _ := oracle.ABASeriesUse(value)
			t.Fatalf("row %d produced %s, whose series is for %q rather than for the transactions of banks", row, value, use)
		}
		if oracle.ABAInReservedSeries(value) {
			t.Fatalf("row %d produced %s, which sits in a series the registrar has not handed out", row, value)
		}
	}
}

// The source has to keep saying that spare series are stock rather than
// sanctuary. If it ever stopped, the reason for generating inside an issued
// series would be gone, and so would the obligation to mark the dataset.
func TestTheSpareSeriesAreStockRatherThanSanctuary(t *testing.T) {
	if !oracle.ABASpareSeriesAreNotAHarbour() {
		t.Fatal("the quoted note no longer says the Board decides the use of spare series")
	}
	if !oracle.ABAStructureSaysNineDigits() {
		t.Fatal("the quoted structure no longer says nine digits")
	}
}

// The rule is modulo ten, so every body completes. There is no hole to fall
// into and therefore no refusal, which is the difference from the Polish tax
// number and worth a test rather than a sentence.
func TestRecomputingARoutingCheckDigitNeverRefuses(t *testing.T) {
	for row := int64(0); row < 500; row++ {
		value := generateABA(t, row)
		for position := 0; position < 8; position++ {
			damaged := []byte(value)
			damaged[position] = '0' + byte((int(damaged[position]-'0')+1)%10)

			repaired, err := us.ABA{}.WithCheckDigits(string(damaged))
			if err != nil {
				t.Fatalf("recomputing %s refused: %v", damaged, err)
			}
			if findings := oracle.CheckABA(repaired); len(findings) != 0 {
				t.Fatalf("recomputing %s gave %s, which the validator refuses: %v", damaged, repaired, findings)
			}
		}
	}
}

// # The corruption, and the rule from DEC-109 about testing one
//
// A test claiming that a corruption is caught by rule Y has to check that the
// value passes every other rule as well. Without it a mutator can be green
// because it broke something else entirely, which happened here once before.
//
// So the moved value has to be well formed, keep its length, keep its check
// digit consistent, and be refused only by the allocation table.
func TestMovingIntoASpareSeriesLeavesAWellFormedNumberTheTableRefuses(t *testing.T) {
	for _, target := range (us.ABA{}).UnassignedTargets() {
		for row := int64(0); row < 200; row++ {
			value := generateABA(t, row)

			moved, err := us.ABA{}.WithUnassigned(value, target)
			if err != nil {
				t.Fatalf("row %d: moving %s into %s failed: %v", row, value, target, err)
			}
			if len(moved) != len(value) {
				t.Fatalf("row %d: moving %s into %s changed its length to %d", row, value, target, len(moved))
			}
			// Every rule except the one the corruption aims at still holds.
			if findings := oracle.CheckABA(moved); len(findings) != 0 {
				t.Fatalf("row %d: %s is not well formed after the move: %v", row, moved, findings)
			}
			if verdict := (us.ABA{}).Validate(moved); !verdict.Valid {
				t.Fatalf("row %d: %s fails its own validator after the move: %s", row, moved, verdict.Reason)
			}
			// And the one it aims at does not.
			if !oracle.ABAInReservedSeries(moved) {
				t.Fatalf("row %d: %s did not land in a spare series", row, moved)
			}
			if oracle.ABAIssuedToInstitutions(moved) {
				t.Fatalf("row %d: %s still sits in an issued series", row, moved)
			}
		}
	}
}

// Moving a value where it already sits is refused rather than reported as a
// corruption that changed nothing. Unlike the SSN this can only reach a value
// somebody else supplied, because pillar one never generates in a spare series.
func TestMovingARoutingNumberIntoTheSeriesItAlreadyOccupiesIsRefused(t *testing.T) {
	value := generateABA(t, 3)
	moved, err := us.ABA{}.WithUnassigned(value, us.TargetSpareLow)
	if err != nil {
		t.Fatalf("the first move failed: %v", err)
	}
	if _, err := (us.ABA{}).WithUnassigned(moved, us.TargetSpareLow); err == nil {
		t.Fatal("moving a value into the series it already occupies was accepted, so a corruption that changed nothing would be reported as applied")
	}
}

// An unknown target names the choices rather than failing silently.
func TestAnUnknownTargetNamesTheChoices(t *testing.T) {
	_, err := us.ABA{}.WithUnassigned(generateABA(t, 1), "somewhere_else")
	if err == nil {
		t.Fatal("an unknown target was accepted")
	}
	for _, target := range (us.ABA{}).UnassignedTargets() {
		if !strings.Contains(err.Error(), target) {
			t.Fatalf("the refusal does not name %q: %v", target, err)
		}
	}
}

// Federal law prints these numbers grouped, and that is where the measurement
// behind the weights came from, so a value copied out of the regulation has to
// be readable. The shape a value arrived in survives being rebuilt.
func TestBothWaysOfWritingARoutingNumberAreAccepted(t *testing.T) {
	bare := generateABA(t, 11)
	grouped := bare[0:4] + "-" + bare[4:8] + "-" + bare[8:9]

	for _, written := range []string{bare, grouped} {
		if verdict := (us.ABA{}).Validate(written); !verdict.Valid {
			t.Fatalf("%q was refused: %s", written, verdict.Reason)
		}
		if findings := oracle.CheckABA(written); len(findings) != 0 {
			t.Fatalf("%q was refused by the independent validator: %v", written, findings)
		}
		rebuilt, err := us.ABA{}.WithCheckDigits(written)
		if err != nil {
			t.Fatalf("%q could not be rebuilt: %v", written, err)
		}
		if rebuilt != written {
			t.Fatalf("%q came back as %q, so the shape did not survive", written, rebuilt)
		}
		positions, ok := (us.ABA{}).CheckDigits(written)
		if !ok || len(positions) != 1 || positions[0] != len(written)-1 {
			t.Fatalf("%q reported its check digit at %v", written, positions)
		}
	}
}

// A value that is not a routing number is refused rather than repaired.
func TestValuesThatAreNotRoutingNumbersAreRefused(t *testing.T) {
	for _, value := range []string{"", "12345678", "1234567890", "12345678a", "123-45-6789", "         "} {
		if verdict := (us.ABA{}).Validate(value); verdict.Valid {
			t.Errorf("%q was accepted", value)
		}
		if _, ok := (us.ABA{}).CheckDigits(value); ok {
			t.Errorf("%q was given a check digit position", value)
		}
	}
}

// The same coordinate has to give the same number, and a different one a
// different number. This is the property the whole tool rests on.
func TestARoutingNumberIsAFunctionOfItsPlace(t *testing.T) {
	first := generateABA(t, 40)
	if again := generateABA(t, 40); again != first {
		t.Fatalf("the same place gave %s and then %s", first, again)
	}
	distinct := map[string]bool{}
	for row := int64(0); row < 500; row++ {
		distinct[generateABA(t, row)] = true
	}
	if len(distinct) < 400 {
		t.Fatalf("five hundred places gave only %d different numbers, so the draw is collapsing", len(distinct))
	}
}

// The values this provider produces are a promise to everybody holding a
// dataset made with a given seed. This freezes them against the past.
func TestTheGeneratedRoutingNumbersHaveNotMoved(t *testing.T) {
	cases := []struct {
		seed string
		row  int64
	}{
		{"release-2026-08", 0},
		{"release-2026-08", 1},
		{"release-2026-08", 2},
		{"release-2026-08", 7},
		{"release-2026-08", 42},
		{"release-2026-08", 999},
		{"release-2026-08", 1000000},
		{"another-seed", 0},
		{"another-seed", 1},
		{"release-2026-08", 5},
	}

	produced := make([]sample, 0, len(cases))
	for _, test := range cases {
		at := seed.Coordinate{Entity: "account", Row: test.row, Field: "routing_number", Purpose: "value"}
		value, err := us.ABA{}.Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		corrupted, err := us.ABA{}.WithUnassigned(value, us.TargetABADefault)
		if err != nil {
			t.Fatalf("seed %q row %d: corrupting %s failed: %v", test.seed, test.row, value, err)
		}
		produced = append(produced, sample{
			Seed: test.seed, Entity: at.Entity, Row: at.Row, Field: at.Field,
			Purpose: at.Purpose, Value: value, Corrupted: corrupted,
		})
	}

	encoded, err := json.MarshalIndent(produced, "", "  ")
	if err != nil {
		t.Fatalf("cannot encode: %v", err)
	}
	path := filepath.Join("testdata", "aba.json")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("cannot create testdata: %v", err)
		}
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Log("golden file rewritten. Run tools/wyrocznia_aba.py before trusting it.")
		return
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the golden file, run with -update to create it: %v", err)
	}
	if string(stored) != string(append(encoded, '\n')) {
		t.Fatal("the generated routing numbers have moved.\n" +
			"Every dataset made with these seeds now differs from the one somebody has on disk.\n" +
			"If that was not meant, this is a defect. If it was meant, it is a major version and a\n" +
			"changelog entry, and the file is rewritten with -update and checked by\n" +
			"tools/wyrocznia_aba.py.")
	}
}
