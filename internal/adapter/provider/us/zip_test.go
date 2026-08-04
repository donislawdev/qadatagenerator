package us_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// zip is the provider under test. It is a package level value because a
// composite literal cannot stand in the init clause of an if without
// parentheses, and scattering those through every check reads worse than
// naming the thing once.
var zip = us.ZIP{}

// The registry is the product here, more than the code around it. These walk
// all of it rather than sampling, because it is finite and cheap: the project
// decided once that where a domain can be enumerated, exhaustive beats random.

func TestEveryPairInTheRegistryIsWellFormed(t *testing.T) {
	lines, ok := corpus.Load("en-US", "postal-places")
	if !ok {
		t.Fatal("the postal registry is not in the binary")
	}
	if len(lines) < 10000 {
		t.Fatalf("the registry holds %d pairs, which is too few to have come out right", len(lines))
	}

	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			t.Fatalf("%q is not code, place and region", line)
		}
		code, place, region := parts[0], parts[1], parts[2]
		if findings := oracle.CheckPostalCode(code); len(findings) != 0 {
			t.Errorf("%q: the independent check says %v", code, findings)
		}
		if place == "" || region == "" {
			t.Errorf("%q carries an empty place or region", line)
		}
		if place != strings.TrimSpace(place) {
			t.Errorf("%q has space around the place name", line)
		}
		if len(region) != 2 {
			t.Errorf("%q: the region is not a two letter abbreviation", line)
		}
	}
}

// The registry ships bare place names, which means a term was removed from
// nearly every one of them. A removal rule that is slightly wrong truncates
// names quietly, so the check uses the list held in the oracle rather than the
// one the extraction used.
func TestNoPlaceNameStillCarriesTheTermTheCensusAppends(t *testing.T) {
	lines, _ := corpus.Load("en-US", "postal-places")
	for _, line := range lines {
		place := strings.Split(line, "|")[1]
		if term, carries := oracle.PlaceNameCarriesATerm(place); carries {
			t.Errorf("%q still ends in %q, so the reduction did not happen", place, term)
		}
	}
}

// The forward direction. Every code the registry carries has to answer, because
// a lookup returning nothing for data we shipped ourselves would be a defect
// nobody could see from outside.
func TestEveryCodeInTheRegistryResolvesToItsPlace(t *testing.T) {
	lines, _ := corpus.Load("en-US", "postal-places")
	for _, line := range lines {
		parts := strings.Split(line, "|")
		place, ok := zip.PlaceFor(parts[0])
		if !ok {
			t.Fatalf("%q is in the registry and did not resolve", parts[0])
		}
		if place.Name != parts[1] || place.Region != parts[2] {
			t.Errorf("%q resolved to %v, expected %s in %s", parts[0], place, parts[1], parts[2])
		}
	}
}

// The reverse direction, which the capability promises as well.
func TestAPlaceAnswersWithTheCodesItHolds(t *testing.T) {
	lines, _ := corpus.Load("en-US", "postal-places")
	first := strings.Split(lines[0], "|")
	place := provider.Place{Name: first[1], Region: first[2]}

	codes := zip.CodesFor(place)
	if len(codes) == 0 {
		t.Fatalf("%v holds no codes, and its own line says otherwise", place)
	}
	found := false
	for _, code := range codes {
		if code == first[0] {
			found = true
		}
		if back, ok := zip.PlaceFor(code); !ok || back != place {
			t.Errorf("%q came back for %v and resolves to %v", code, place, back)
		}
	}
	if !found {
		t.Errorf("%v does not list its own code %q", place, first[0])
	}
	if zip.CodesFor(provider.Place{Name: "Nowhere", Region: "ZZ"}) != nil {
		t.Error("a place that is not in the registry answered with codes")
	}
}

// The write side, which is what the corruption needs. The value it returns has
// to be real and its place has to differ, or the corruption produces something
// the manifest describes wrongly.
func TestADifferentPlaceIsRealAndActuallyDifferent(t *testing.T) {
	lines, _ := corpus.Load("en-US", "postal-places")
	// Walking all of them is affordable and catches the wrap around at the end
	// of the registry, which a sample would miss precisely because it is one
	// case out of thirty thousand.
	for _, line := range lines {
		parts := strings.Split(line, "|")
		code := parts[0]
		other, err := zip.WithDifferentPlace(code)
		if err != nil {
			t.Fatalf("%q: %v", code, err)
		}
		here := provider.Place{Name: parts[1], Region: parts[2]}
		there, ok := zip.PlaceFor(other)
		if !ok {
			t.Fatalf("%q produced %q, which is not in the registry", code, other)
		}
		if there == here {
			t.Errorf("%q produced %q, which belongs to the same place %v", code, other, here)
		}
	}
}

// The same input has to give the same answer. This is the promise the whole
// project rests on, and a corruption is the easiest place to break it.
func TestADifferentPlaceIsTheSameAnswerEveryTime(t *testing.T) {
	for _, code := range []string{"01001", "10001", "90210", "99929"} {
		first, err := zip.WithDifferentPlace(code)
		if err != nil {
			t.Fatalf("%q: %v", code, err)
		}
		for i := 0; i < 5; i++ {
			again, err := zip.WithDifferentPlace(code)
			if err != nil || again != first {
				t.Fatalf("%q gave %q then %q", code, first, again)
			}
		}
	}
}

// The refusal. A code the registry does not carry cannot be corrupted into a
// mismatched pair, and saying so is what keeps it out of the manifest as a
// corruption that happened.
func TestACodeTheRegistryDoesNotCarryIsRefusedRatherThanInvented(t *testing.T) {
	for _, code := range []string{"00000", "99999", "abcde", "", "1234", "123456"} {
		if _, err := zip.WithDifferentPlace(code); err == nil {
			t.Errorf("%q was corrupted although the registry cannot place it", code)
		}
	}
}

// Shape only, and the omission is the point. A real code missing from the
// registry must not be reported as a defect, because the source says outright
// that the registry does not cover the scheme.
func TestValidationAnswersAboutShapeAndNotAboutMembership(t *testing.T) {
	unlisted := "99999"
	if _, ok := zip.PlaceFor(unlisted); ok {
		t.Fatalf("%s was chosen as an unlisted code and the registry carries it", unlisted)
	}
	if verdict := zip.Validate(unlisted); !verdict.Valid {
		t.Errorf("a well formed code outside the registry was faulted: %s", verdict.Reason)
	}
	if !strings.Contains(oracle.RegistryIsNotThePostalRegister(), "Not all valid ZIP Codes") {
		t.Error("the sentence this leniency rests on is no longer the one being quoted")
	}
	if !strings.Contains(oracle.RegistryApproximatesThePostalScheme(), "Tabulation Areas") {
		t.Error("the definition the registry is built on is no longer the one being quoted")
	}

	for _, bad := range []string{"", "1234", "123456", "12345-", "12345-123", "1234a", "12345 6789"} {
		if verdict := zip.Validate(bad); verdict.Valid {
			t.Errorf("%q passed validation", bad)
		}
		if findings := oracle.CheckPostalCode(bad); len(findings) == 0 {
			t.Errorf("%q passed the independent check", bad)
		}
	}
	for _, good := range []string{"01001", "12345-6789"} {
		if verdict := zip.Validate(good); !verdict.Valid {
			t.Errorf("%q was faulted: %s", good, verdict.Reason)
		}
	}
}

// The longer written form names the same place, because the four digits after
// the hyphen narrow a route inside it rather than move it.
func TestTheLongerWrittenFormResolvesToTheSamePlace(t *testing.T) {
	lines, _ := corpus.Load("en-US", "postal-places")
	parts := strings.Split(lines[0], "|")
	short, _ := zip.PlaceFor(parts[0])
	long, ok := zip.PlaceFor(parts[0] + "-6789")
	if !ok || short != long {
		t.Errorf("%s resolved to %v and %s-6789 to %v", parts[0], short, parts[0], long)
	}
}

// Generation has to stay inside the registry, or pillar one produces a code
// nothing can pair with a place.
func TestEveryGeneratedCodeIsOneTheRegistryCanPlace(t *testing.T) {
	source := seed.New("release-2026-08")
	for row := int64(0); row < 2000; row++ {
		at := seed.Coordinate{Entity: "customer", Row: row, Field: "postal_code", Purpose: "value"}
		value, err := zip.Generate(provider.Request{Source: source, At: at})
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if _, ok := zip.PlaceFor(value); !ok {
			t.Fatalf("row %d produced %q, which the registry cannot place", row, value)
		}
		if verdict := zip.Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %q, which fails our own validator: %s", row, value, verdict.Reason)
		}
	}
}

// The contract, in both directions. A declared capability whose interface is
// missing, or an interface nobody declared, is caught here rather than at run
// time by a mutator that never applies.
func TestTheProviderMeetsItsOwnDeclaration(t *testing.T) {
	if findings := provider.Check(zip); len(findings) != 0 {
		t.Errorf("US.zip breaks its contract: %v", findings)
	}
	declared := zip.Capabilities()
	if len(declared) != 1 || declared[0] != capability.ReferencePostalToPlace {
		t.Errorf("US.zip declares %v, and it brings exactly the postal registry", declared)
	}
}

// The values this provider produces are a promise to everybody holding a
// dataset made with a given seed. This freezes them against the past, which is
// a different guarantee from the tests that check them against the rules.
//
// The codes written here are real, and unlike every other golden file in this
// project that is not a problem. A postal code names an area rather than a
// person or an institution, so it cannot be traced to anybody, and the whole
// registry is real by necessity: a capability pairing codes with places cannot
// be built from invented codes. The rule that keeps real identifiers out of this
// repository is about values that could belong to a subject, and this is the one
// scheme here where they cannot.
func TestTheGeneratedPostalCodesHaveNotMoved(t *testing.T) {
	type postalSample struct {
		Seed    string `json:"seed"`
		Row     int64  `json:"row"`
		Value   string `json:"value"`
		Place   string `json:"place"`
		Region  string `json:"region"`
		Foreign string `json:"foreign"`
	}

	rows := []struct {
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

	produced := make([]postalSample, 0, len(rows))
	for _, test := range rows {
		at := seed.Coordinate{Entity: "customer", Row: test.row, Field: "postal_code", Purpose: "value"}
		value, err := zip.Generate(provider.Request{Source: seed.New(test.seed), At: at})
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		place, ok := zip.PlaceFor(value)
		if !ok {
			t.Fatalf("seed %q row %d: %q has no place", test.seed, test.row, value)
		}
		foreign, err := zip.WithDifferentPlace(value)
		if err != nil {
			t.Fatalf("seed %q row %d: %v", test.seed, test.row, err)
		}
		produced = append(produced, postalSample{
			Seed: test.seed, Row: test.row, Value: value,
			Place: place.Name, Region: place.Region, Foreign: foreign,
		})
	}

	encoded, err := json.MarshalIndent(produced, "", "  ")
	if err != nil {
		t.Fatalf("cannot encode: %v", err)
	}
	path := filepath.Join("testdata", "zip.json")

	if *update {
		if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
			t.Fatalf("cannot write the golden file: %v", err)
		}
		t.Log("golden file rewritten. Run tools/wyrocznia_zip.py before trusting it.")
		return
	}

	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the golden file, run with -update to create it: %v", err)
	}
	if string(stored) != string(append(encoded, '\n')) {
		t.Fatal("the generated postal codes have moved.\n" +
			"Every dataset made with these seeds now differs from the one somebody has on disk.\n" +
			"If that was not meant, this is a defect. If it was meant, it is a major version and a\n" +
			"changelog entry, and the file is rewritten with -update and checked by\n" +
			"tools/wyrocznia_zip.py.")
	}
}
