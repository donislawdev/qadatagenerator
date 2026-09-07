package intl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

func isbnGoldenPath() string { return filepath.Join("testdata", "isbn.json") }

// The values this provider produces are a promise to everybody holding a
// dataset made with a given seed. This freezes them against the past, which is
// a different guarantee from the tests that check them against the manual:
// those would stay green if every digit moved at once.
func TestTheGeneratedBookNumbersHaveNotMoved(t *testing.T) {
	cases := []struct {
		seed string
		row  int64
	}{
		{"release-2026-09", 0},
		{"release-2026-09", 1},
		{"release-2026-09", 2},
		{"release-2026-09", 7},
		{"release-2026-09", 42},
		{"release-2026-09", 999},
		{"release-2026-09", 1000000},
		{"another-seed", 0},
		{"another-seed", 1},
		{"release-2026-09", 5},
	}

	produced := make([]sample, 0, len(cases))
	for _, test := range cases {
		at := seed.Coordinate{Entity: "library", Row: test.row, Field: "isbn", Purpose: "value"}
		value, err := (intl.ISBN{}).Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		produced = append(produced, sample{
			Seed: test.seed, Entity: at.Entity, Row: at.Row, Field: at.Field,
			Purpose: at.Purpose, Value: value, Broken: withWrongBookCheckDigit(value),
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
		if err := os.WriteFile(isbnGoldenPath(), append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Log("golden file rewritten. Run tools/wyrocznia_isbn.py before trusting it.")
		return
	}

	stored, err := os.ReadFile(isbnGoldenPath())
	if err != nil {
		t.Fatalf("cannot read the golden file, run with -update to create it: %v", err)
	}
	if string(stored) != string(append(encoded, '\n')) {
		t.Fatal("the generated book numbers have moved.\n" +
			"Every dataset made with these seeds now differs from the one somebody has on disk.\n" +
			"If that was not meant, this is a defect. If it was meant, it is a major version and a\n" +
			"changelog entry, and the file is rewritten with -update and checked by\n" +
			"tools/wyrocznia_isbn.py.")
	}
}

// withWrongBookCheckDigit moves the check digit on by one, which is the
// corruption the catalog aims at a check digit.
//
// Thirteen digits with the check digit last, which is the article number's
// shape rather than the equipment identity's. It is still its own function
// rather than a shared one, because sharing it would tie two schemes together
// through a test helper and the shapes agree today by coincidence rather than
// by any rule saying they must.
func withWrongBookCheckDigit(value string) string {
	broken := []byte(value)
	broken[12] = '0' + (broken[12]-'0'+1)%10
	return string(broken)
}
