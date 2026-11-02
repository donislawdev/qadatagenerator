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

func cardGoldenPath() string { return filepath.Join("testdata", "card.json") }

// The numbers this provider draws are a promise to everybody holding a dataset
// made with a given seed. This freezes which of the published numbers each row
// gets, which the tests against the list would not notice moving: every number
// would still be published.
func TestTheGeneratedCardsHaveNotMoved(t *testing.T) {
	cases := []struct {
		seed string
		row  int64
	}{
		{"release-2026-11", 0},
		{"release-2026-11", 1},
		{"release-2026-11", 2},
		{"release-2026-11", 7},
		{"release-2026-11", 42},
		{"release-2026-11", 999},
		{"release-2026-11", 1000000},
		{"another-seed", 0},
		{"another-seed", 1},
		{"release-2026-11", 5},
	}

	produced := make([]sample, 0, len(cases))
	for _, test := range cases {
		at := seed.Coordinate{Entity: "payment", Row: test.row, Field: "card", Purpose: "value"}
		value, err := (intl.Card{}).Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		produced = append(produced, sample{
			Seed: test.seed, Entity: at.Entity, Row: at.Row, Field: at.Field,
			Purpose: at.Purpose, Value: value, Broken: withWrongCardCheckDigit(value),
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
		if err := os.WriteFile(cardGoldenPath(), append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Log("golden file rewritten. Run tools/wyrocznia_karta.py before trusting it.")
		return
	}

	stored, err := os.ReadFile(cardGoldenPath())
	if err != nil {
		t.Fatalf("cannot read the golden file, run with -update to create it: %v", err)
	}
	if string(stored) != string(append(encoded, '\n')) {
		t.Fatal("the generated card numbers have moved.\n" +
			"Every dataset made with these seeds now differs from the one somebody has on disk.\n" +
			"If that was not meant, this is a defect. If it was meant, it is a major version and a\n" +
			"changelog entry, and the file is rewritten with -update and checked by\n" +
			"tools/wyrocznia_karta.py.")
	}
}

// withWrongCardCheckDigit moves the check digit on by one, which is the
// corruption the catalog aims at a check digit.
//
// Its own function rather than the equipment identity's, because the check
// digit of a card sits at the last position of a value that is fifteen or
// sixteen long rather than at a fixed one.
func withWrongCardCheckDigit(value string) string {
	broken := []byte(value)
	last := len(broken) - 1
	broken[last] = '0' + (broken[last]-'0'+1)%10
	return string(broken)
}
