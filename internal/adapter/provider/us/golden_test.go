package us_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

var update = flag.Bool("update", false, "rewrite the golden file after a deliberate change to the provider")

// sample is one generated number together with the address it came from, so
// that a script in another language can recompute the same answer without
// reading any Go.
type sample struct {
	Seed    string `json:"seed"`
	Entity  string `json:"entity"`
	Row     int64  `json:"row"`
	Field   string `json:"field"`
	Purpose string `json:"purpose"`
	Value   string `json:"value"`
	// Corrupted is the same number moved into the block the catalog entry
	// aims at by default. It is here because the pair is the thing that has to
	// stay stable: a change to either side alone would go unnoticed.
	Corrupted string `json:"corrupted"`
}

func goldenPath() string { return filepath.Join("testdata", "ssn.json") }

// The values this provider produces are a promise to everybody holding a
// dataset made with a given seed. This freezes them against the past, which is
// a different guarantee from the tests that check them against the rules:
// those would stay green if every byte moved at once.
func TestTheGeneratedNumbersHaveNotMoved(t *testing.T) {
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
		at := seed.Coordinate{Entity: "customer", Row: test.row, Field: "national_id", Purpose: "value"}
		value, err := us.SSN{}.Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		corrupted, err := us.SSN{}.WithUnassigned(value, us.TargetDefault)
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

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("cannot create testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath(), append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Log("golden file rewritten. Run tools/wyrocznia_ssn.py before trusting it.")
		return
	}

	stored, err := os.ReadFile(goldenPath())
	if err != nil {
		t.Fatalf("cannot read the golden file, run with -update to create it: %v", err)
	}
	if string(stored) != string(append(encoded, '\n')) {
		t.Fatal("the generated numbers have moved.\n" +
			"Every dataset made with these seeds now differs from the one somebody has on disk.\n" +
			"If that was not meant, this is a defect. If it was meant, it is a major version and a\n" +
			"changelog entry, and the file is rewritten with -update and checked by\n" +
			"tools/wyrocznia_ssn.py.")
	}
}
