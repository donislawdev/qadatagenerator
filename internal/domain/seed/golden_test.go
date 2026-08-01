package seed_test

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// The golden file answers a different question than every other test here.
//
// Comparing two runs of the same build can only ever find randomness that
// crept in. It stays green when the derivation changes completely, because
// both halves of the comparison change together. These values were written
// down once and committed, so they compare today's bytes with the past.
//
// They are also what checks that the derivation does not depend on the
// machine. The same file has to produce the same bytes on Windows, macOS and
// Linux, so until there is a build matrix this test is the whole of that
// promise.
var update = flag.Bool("update", false, "rewrite the golden file after a deliberate change of the derivation")

type goldenCase struct {
	Seed    string `json:"seed"`
	Entity  string `json:"entity"`
	Row     int64  `json:"row"`
	Field   string `json:"field"`
	Purpose string `json:"purpose"`
	Digest  string `json:"digest"`
	Number  uint64 `json:"number"`
	Bound   int64  `json:"bound"`
	Bounded int64  `json:"bounded"`
}

// The places below are chosen to pin every part of the message: an empty
// everything, text outside ASCII, a length that would matter if the framing
// were dropped, both ends of the row range, and bounds that do and do not
// divide evenly.
var goldenPlaces = []struct {
	seed  string
	place seed.Coordinate
	bound int64
}{
	{"", seed.Coordinate{}, 1},
	{"run-1", seed.Coordinate{Entity: "customer", Row: 0, Field: "first_name", Purpose: "value"}, 2},
	{"run-1", seed.Coordinate{Entity: "customer", Row: 1, Field: "first_name", Purpose: "value"}, 7},
	{"run-1", seed.Coordinate{Entity: "customer", Row: 0, Field: "last_name", Purpose: "value"}, 100},
	{"run-1", seed.Coordinate{Entity: "customer", Row: 0, Field: "first_name", Purpose: "corrupt"}, 256},
	{"run-2", seed.Coordinate{Entity: "customer", Row: 0, Field: "first_name", Purpose: "value"}, 1000},
	{"release-2026-08", seed.Coordinate{Entity: "order", Row: 4812903, Field: "total", Purpose: "value"}, 999983},
	{"zażółć", seed.Coordinate{Entity: "kontrahent", Row: 12, Field: "nazwa", Purpose: "value"}, 3},
	{"s", seed.Coordinate{Entity: "ab", Row: 0, Field: "c", Purpose: ""}, 5},
	{"s", seed.Coordinate{Entity: "a", Row: 0, Field: "bc", Purpose: ""}, 5},
	{"s", seed.Coordinate{Entity: "e", Row: -1, Field: "f", Purpose: ""}, 10},
	{"s", seed.Coordinate{Entity: "e", Row: 9223372036854775807, Field: "f", Purpose: ""}, 10},
	{"s", seed.Coordinate{Entity: "e", Row: 5, Field: "f", Purpose: ""}, 5000000000000000000},
}

func currentGolden() []goldenCase {
	cases := make([]goldenCase, 0, len(goldenPlaces))
	for _, wanted := range goldenPlaces {
		source := seed.New(wanted.seed)
		digest := source.Bytes(wanted.place)
		cases = append(cases, goldenCase{
			Seed:    wanted.seed,
			Entity:  wanted.place.Entity,
			Row:     wanted.place.Row,
			Field:   wanted.place.Field,
			Purpose: wanted.place.Purpose,
			Digest:  hex.EncodeToString(digest[:]),
			Number:  source.Uint64(wanted.place),
			Bound:   wanted.bound,
			Bounded: source.IntN(wanted.place, wanted.bound),
		})
	}
	return cases
}

func goldenPath() string {
	return filepath.Join("testdata", "derivation.json")
}

func TestTheDerivationHasNotMoved(t *testing.T) {
	current := currentGolden()

	if *update {
		encoded, err := json.MarshalIndent(current, "", "  ")
		if err != nil {
			t.Fatalf("cannot encode the golden values: %v", err)
		}
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("cannot create testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath(), append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Log("golden file rewritten, which is a major version change and an entry in the changelog")
		return
	}

	raw, err := os.ReadFile(goldenPath())
	if err != nil {
		t.Fatalf("cannot read the golden file: %v", err)
	}
	var recorded []goldenCase
	if err := json.Unmarshal(raw, &recorded); err != nil {
		t.Fatalf("cannot read the golden values: %v", err)
	}

	if len(recorded) != len(current) {
		t.Fatalf("the golden file holds %d places and the test asks about %d", len(recorded), len(current))
	}
	for index, want := range recorded {
		got := current[index]
		if got == want {
			continue
		}
		t.Errorf("place %d moved\n got %+v\nwant %+v", index, got, want)
	}
	if t.Failed() {
		t.Log("this is the derivation itself changing, so every dataset ever generated now differs. " +
			"If it was meant, it is a major version and a changelog entry, and the file is rewritten with -update.")
	}
}
