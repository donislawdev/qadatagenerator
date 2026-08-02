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

func taxGoldenPath() string { return filepath.Join("testdata", "nip.json") }

func statisticalGoldenPath() string { return filepath.Join("testdata", "regon.json") }

// writeOrCompare is the whole of what a golden test does, and both providers do
// it the same way. Keeping one copy means a change to how the file is written
// cannot reach one identifier and miss the other.
func writeOrCompare(t *testing.T, path string, produced []sample, oracleScript string) {
	t.Helper()

	encoded, err := json.MarshalIndent(produced, "", "  ")
	if err != nil {
		t.Fatalf("cannot encode: %v", err)
	}
	encoded = append(encoded, '\n')

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("cannot create testdata: %v", err)
		}
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Logf("golden file rewritten. Run %s before trusting it.", oracleScript)
		return
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the golden file, run with -update to create it: %v", err)
	}
	if string(stored) != string(encoded) {
		t.Fatalf("the generated numbers have moved.\n"+
			"Every dataset made with these seeds now differs from the one somebody has on disk.\n"+
			"If that was not meant, this is a defect. If it was meant, it is a major version and a\n"+
			"changelog entry, and the file is rewritten with -update and checked by\n%s.", oracleScript)
	}
}

// The tax numbers are frozen against the past for the same reason the personal
// ones are: somebody holding a dataset made with a given seed was promised
// these bytes. The tests checking them against the rules would stay green if
// every byte moved at once.
func TestTheGeneratedTaxNumbersHaveNotMoved(t *testing.T) {
	cases := []struct {
		seed string
		row  int64
	}{
		{"release-2026-08", 0},
		{"release-2026-08", 1},
		{"release-2026-08", 2},
		{"release-2026-08", 999},
		{"release-2026-08", 123456},
		{"another-seed", 0},
		{"another-seed", 1},
	}

	produced := make([]sample, 0, len(cases))
	for _, test := range cases {
		at := seed.Coordinate{Entity: "company", Row: test.row, Field: "nip", Purpose: "value"}
		value, err := pl.NIP{}.Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		produced = append(produced, sample{
			Seed: test.seed, Entity: at.Entity, Row: at.Row, Field: at.Field,
			Purpose: at.Purpose, Value: value,
		})
	}

	writeOrCompare(t, taxGoldenPath(), produced, "tools/wyrocznia_nip.py")
}

// The statistical numbers are frozen for the same reason as the other two. This
// one has no bound date or gender to vary, because the regulation forbids its
// digits from meaning anything about the entity, so the cases vary only the
// address the value is derived from.
func TestTheGeneratedStatisticalNumbersHaveNotMoved(t *testing.T) {
	cases := []struct {
		seed string
		row  int64
	}{
		{"release-2026-08", 0},
		{"release-2026-08", 1},
		{"release-2026-08", 2},
		{"release-2026-08", 999},
		{"release-2026-08", 123456},
		{"another-seed", 0},
		{"another-seed", 1},
	}

	produced := make([]sample, 0, len(cases))
	for _, test := range cases {
		at := seed.Coordinate{Entity: "company", Row: test.row, Field: "regon", Purpose: "value"}
		value, err := pl.REGON{}.Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		produced = append(produced, sample{
			Seed: test.seed, Entity: at.Entity, Row: at.Row, Field: at.Field,
			Purpose: at.Purpose, Value: value,
		})
	}

	writeOrCompare(t, statisticalGoldenPath(), produced, "tools/wyrocznia_regon.py")
}

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

	writeOrCompare(t, goldenPath(), produced, "tools/wyrocznia_pesel.py")
}
