package intl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// theAnchor is the calculation 49 CFR 565.15 works in Table VI, with the check
// digit the regulation arrives at put back in position nine.
//
// It is quoted rather than generated, for the reason the tax number keeps its
// own worked example: an anchor derived from our own arithmetic would agree
// with our own arithmetic and prove nothing about the source.
const theAnchor = "1G4AH59H45G118341"

// generateVIN produces the number for one row, failing the test rather than
// handing back an error nobody looks at.
func generateVIN(t *testing.T, source seed.Source, row int64) string {
	t.Helper()
	value, err := (intl.VIN{}).Generate(provider.Request{
		Source: source,
		At:     seed.Coordinate{Entity: "fleet", Row: row, Field: "chassis", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: the generator refused: %v", row, err)
	}
	return value
}

// The strongest anchor a source can give: it does the arithmetic itself. This
// runs the regulation's own example through our implementation, so a weight or
// a letter value typed wrongly is caught by the document rather than by us
// agreeing with ourselves.
func TestTheCalculationWorkedByTheRegulationComesOutTheSameHere(t *testing.T) {
	if verdict := (intl.VIN{}).Validate(theAnchor); !verdict.Valid {
		t.Fatalf("the number the regulation works out in Table VI was refused: %s", verdict.Reason)
	}
	rebuilt, err := (intl.VIN{}).WithCheckDigits(theAnchor)
	if err != nil {
		t.Fatalf("recomputing the check digit of the worked example failed: %v", err)
	}
	if rebuilt != theAnchor {
		t.Errorf("the worked example rebuilt with its own check digit became %q", rebuilt)
	}
	if findings := oracle.CheckVIN(theAnchor); len(findings) != 0 {
		t.Errorf("the independent validator refused the worked example: %v", findings)
	}
}

// Every value this build produces has to satisfy the validator written from the
// regulation rather than from the generator.
func TestEveryGeneratedVehicleNumberSatisfiesTheIndependentValidator(t *testing.T) {
	source := seed.New("vin-sweep-2026-09")
	for row := int64(0); row < 5000; row++ {
		value := generateVIN(t, source, row)
		if len(value) != 17 {
			t.Fatalf("row %d produced %q, which is %d characters", row, value, len(value))
		}
		if findings := oracle.CheckVIN(value); len(findings) != 0 {
			t.Fatalf("row %d produced %q and the independent validator says %v", row, value, findings)
		}
	}
}

// The regulation writes a letter for the residue that does not fit in a digit,
// where the tax number refuses the body and the statistical number writes
// nought. Three schemes, three answers, and this is the one that proves ours is
// the third rather than a copy of either.
func TestTheEleventhResidueIsWrittenAsTheLetterX(t *testing.T) {
	source := seed.New("vin-folded-2026-09")
	folded := 0
	for row := int64(0); row < 3000; row++ {
		value := generateVIN(t, source, row)
		if value[8] != 'X' {
			continue
		}
		folded++
		if findings := oracle.CheckVIN(value); len(findings) != 0 {
			t.Fatalf("%q carries the folded residue and was refused: %v", value, findings)
		}
	}
	if folded == 0 {
		t.Fatal("no value in the swept range left a remainder of ten, so this test checked nothing")
	}
}

// The three letters the regulation leaves out are left out so that nothing can
// be read as a one or a nought. A value carrying one of them is refused, and it
// is refused for that reason rather than for a wrong check digit.
func TestTheThreeLettersTheRegulationLeavesOutAreRefused(t *testing.T) {
	for _, letter := range []byte{'I', 'O', 'Q'} {
		for at := 0; at < 17; at++ {
			damaged := []byte(theAnchor)
			damaged[at] = letter
			verdict := (intl.VIN{}).Validate(string(damaged))
			if verdict.Valid {
				t.Errorf("%q carries the letter %s at position %d and was accepted",
					damaged, string(letter), at+1)
				continue
			}
			if !strings.Contains(verdict.Reason, "I, O and Q") {
				t.Errorf("%q was refused for the wrong reason: %s", damaged, verdict.Reason)
			}
			if findings := oracle.CheckVIN(string(damaged)); len(findings) == 0 {
				t.Errorf("the independent validator accepted %q", damaged)
			}
		}
	}
}

// A value this build calls valid has to survive being rebuilt with its own
// check digit. Anything else would mean the reader and the writer disagree
// about the same number.
func TestRecomputingTheCheckDigitNeverRefuses(t *testing.T) {
	source := seed.New("vin-rebuild-2026-09")
	for row := int64(0); row < 500; row++ {
		value := generateVIN(t, source, row)
		for at := 0; at < 17; at++ {
			if at == 8 {
				continue
			}
			for _, replacement := range "0123456789ABCDEFGHJKLMNPRSTUVWXYZ" {
				moved := []byte(value)
				moved[at] = byte(replacement)
				repaired, err := (intl.VIN{}).WithCheckDigits(string(moved))
				if err != nil {
					t.Fatalf("%q could not be completed: %v", moved, err)
				}
				if findings := oracle.CheckVIN(repaired); len(findings) != 0 {
					t.Fatalf("%q was repaired into %q and refused: %v", moved, repaired, findings)
				}
			}
		}
	}
}

// The degenerate value that is nonetheless correct. Seventeen noughts sum to
// nought, eleven divides that exactly, and the check digit written for a
// remainder of nought is a nought, which is what already stands in position
// nine. A validator that refused it would be refusing the arithmetic.
func TestSeventeenNoughtsAreACorrectNumberBecauseTheArithmeticSaysSo(t *testing.T) {
	const noughts = "00000000000000000"
	if verdict := (intl.VIN{}).Validate(noughts); !verdict.Valid {
		t.Errorf("seventeen noughts were refused: %s", verdict.Reason)
	}
	if findings := oracle.CheckVIN(noughts); len(findings) != 0 {
		t.Errorf("the independent validator refused seventeen noughts: %v", findings)
	}
}

// The shapes that are not this scheme at all, each refused by the rule it
// breaks rather than by whichever rule happens to be checked first.
func TestTheShapesThisSchemeDoesNotHaveAreRefused(t *testing.T) {
	refused := []struct {
		what  string
		value string
	}{
		{"nothing at all", ""},
		{"one character short", theAnchor[:16]},
		{"one character long", theAnchor + "1"},
		{"written in lower case", strings.ToLower(theAnchor)},
		{"broken up with hyphens", "1G4-AH59H4-5G118341"},
		{"digits of another script", "١٢٣٤٥٦٧٨٩٠١٢٣٤٥٦٧"},
		{"a wrong check digit", theAnchor[:8] + "5" + theAnchor[9:]},
	}
	for _, shape := range refused {
		verdict := (intl.VIN{}).Validate(shape.value)
		if verdict.Valid {
			t.Errorf("a value %s was accepted: %q", shape.what, shape.value)
			continue
		}
		if verdict.Reason == "" {
			t.Errorf("a value %s was refused without a reason", shape.what)
		}
		if findings := oracle.CheckVIN(shape.value); len(findings) == 0 {
			t.Errorf("the independent validator accepted a value %s: %q", shape.what, shape.value)
		}
	}
}

// The model year of Table VII is the one rule this generator satisfies and this
// validator does not enforce, so it needs a test of its own or nothing holds it.
//
// Three characters of the alphabet never appear as a year: nought, U and Z. A
// value carrying one of them in position ten is a value whose year cannot be
// read, which is a defect in a generator of test data even though it is not a
// defect in a number.
func TestEveryGeneratedNumberCarriesAModelYearThatCanBeRead(t *testing.T) {
	const codes = "123456789ABCDEFGHJKLMNPRSTVWXY"
	source := seed.New("vin-year-2026-09")
	for row := int64(0); row < 2000; row++ {
		value := generateVIN(t, source, row)
		if !strings.ContainsRune(codes, rune(value[9])) {
			t.Fatalf("row %d produced %q, whose position ten holds %s and no year is written that way",
				row, value, string(value[9]))
		}
	}
}

// 49 CFR 565.15(d) requires the last five characters to be numeric for
// passenger cars and the last four for every other vehicle, so five satisfies
// both. The example the regulation works in Table VI ends in five digits.
//
// This is the second rule the generator satisfies and the validator does not
// enforce, and it is here because the mutator that adds one to the final digit
// found it missing: it reported that a value ending in a letter has nothing at
// the last position to change. A guard about damage found a defect in
// generation.
func TestEveryGeneratedNumberEndsInTheDigitsTheRegulationRequires(t *testing.T) {
	source := seed.New("vin-tail-2026-09")
	for row := int64(0); row < 2000; row++ {
		value := generateVIN(t, source, row)
		for at := len(value) - 5; at < len(value); at++ {
			if value[at] < '0' || value[at] > '9' {
				t.Fatalf("row %d produced %q, whose position %d is not a digit", row, value, at+1)
			}
		}
	}
}

// The same request has to produce the same value, whatever else the process did
// first. This is the rule the whole product rests on, asked of one scheme.
func TestTheSameRequestProducesTheSameNumber(t *testing.T) {
	first := seed.New("vin-repeat-2026-09")
	second := seed.New("vin-repeat-2026-09")
	for row := int64(0); row < 200; row++ {
		if a, b := generateVIN(t, first, row), generateVIN(t, second, row); a != b {
			t.Fatalf("row %d produced %q and then %q", row, a, b)
		}
	}
}

// Where the check digit sits is a promise a mutator relies on, so it is asked
// rather than assumed.
func TestTheCheckDigitIsReportedAtPositionNine(t *testing.T) {
	positions, ok := (intl.VIN{}).CheckDigits(theAnchor)
	if !ok {
		t.Fatal("a well formed number was said to have no check digit")
	}
	if len(positions) != 1 || positions[0] != 8 {
		t.Errorf("the check digit was reported at %v rather than at the ninth character", positions)
	}
	if _, ok := (intl.VIN{}).CheckDigits(theAnchor[:16]); ok {
		t.Error("a value of the wrong length was said to carry a check digit")
	}
}
