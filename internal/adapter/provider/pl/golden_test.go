package pl_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

var update = flag.Bool("update", false, "rewrite the golden file after a deliberate change to the provider")

// sample is one generated number together with the address it came from, so
// that a script in another language can recompute the same answer without
// reading any Go.
type sample struct {
	Seed      string `json:"seed"`
	Entity    string `json:"entity"`
	Row       int64  `json:"row"`
	Field     string `json:"field"`
	Purpose   string `json:"purpose"`
	BirthDate string `json:"birth_date,omitempty"`
	Gender    string `json:"gender,omitempty"`
	Value     string `json:"value"`
}

func goldenPath() string { return filepath.Join("testdata", "pesel.json") }

// The values this provider produces are a promise to everybody holding a
// dataset made with a given seed. This freezes them against the past, which is
// a different guarantee from the tests that check them against the rules: those
// would stay green if every byte moved at once.
func TestTheGeneratedNumbersHaveNotMoved(t *testing.T) {
	cases := []struct {
		seed      string
		row       int64
		birthDate string
		gender    string
	}{
		{"release-2026-08", 0, "", ""},
		{"release-2026-08", 1, "", ""},
		{"release-2026-08", 999, "", ""},
		{"release-2026-08", 1, "1990-04-11", "female"},
		{"release-2026-08", 1, "1990-04-11", "male"},
		{"release-2026-08", 2, "2004-12-31", "male"},
		{"release-2026-08", 3, "1800-01-01", "female"},
		{"release-2026-08", 4, "2299-12-31", "male"},
		{"release-2026-08", 5, "2000-02-29", "female"},
		{"another-seed", 1, "1990-04-11", "female"},
	}

	produced := make([]sample, 0, len(cases))
	for _, test := range cases {
		agreed := map[string]string{}
		if test.birthDate != "" {
			agreed[provider.RoleBirthDate] = test.birthDate
		}
		if test.gender != "" {
			agreed[provider.RoleGender] = test.gender
		}
		at := seed.Coordinate{Entity: "customer", Row: test.row, Field: "pesel", Purpose: "value"}
		value, err := pl.PESEL{}.Generate(provider.Request{
			Source: seed.New(test.seed),
			At:     at,
			Agrees: provider.NewAgreement(agreed),
		})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		produced = append(produced, sample{
			Seed: test.seed, Entity: at.Entity, Row: at.Row, Field: at.Field,
			Purpose: at.Purpose, BirthDate: test.birthDate, Gender: test.gender,
			Value: value,
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
		t.Log("golden file rewritten. Run tools/wyrocznia_pesel.py before trusting it.")
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
			"tools/wyrocznia_pesel.py.")
	}
}
