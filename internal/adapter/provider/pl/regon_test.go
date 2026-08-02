package pl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// generateREGON produces the number for one row, failing the test rather than
// returning an error nobody looks at.
func generateREGON(t *testing.T, source seed.Source, row int64) string {
	t.Helper()
	value, err := pl.REGON{}.Generate(provider.Request{
		Source: source,
		At:     seed.Coordinate{Entity: "company", Row: row, Field: "regon", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: %v", row, err)
	}
	return value
}

// The generator and the judge must not be the same arithmetic. This one asks a
// validator written from the description, in another file, that never sees what
// the provider produces.
func TestEveryGeneratedStatisticalNumberSatisfiesTheIndependentValidator(t *testing.T) {
	source := seed.New("release-2026-08")
	for row := int64(0); row < 5000; row++ {
		value := generateREGON(t, source, row)
		if len(value) != 9 {
			t.Fatalf("row %d: %q is not nine digits", row, value)
		}
		if findings := oracle.CheckREGON(value); len(findings) != 0 {
			t.Fatalf("row %d: the independent validator rejects %q: %v", row, value, findings)
		}
		if verdict := (pl.REGON{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d: the provider rejects its own output %q: %s", row, value, verdict.Reason)
		}
	}
}

// The one rule that must not be carried over from the tax number next door.
//
// Both schemes divide by eleven and both meet the residue that does not fit in
// a digit. The tax number calls such a number invalid. This one writes nought
// and the number stands. Assuming they agree would produce a provider that
// refused about one body in eleven for no reason.
func TestTheEleventhResidueIsWrittenAsNoughtRatherThanRefusingTheNumber(t *testing.T) {
	weights := [8]int{8, 9, 2, 3, 4, 5, 6, 7}
	folded := 0

	// Exhaustive over a counted domain rather than sampled: every body whose
	// first six digits are nought, which still reaches all eleven residues.
	for seventh := 0; seventh <= 9; seventh++ {
		for eighth := 0; eighth <= 9; eighth++ {
			body := []int{0, 0, 0, 0, 0, 0, seventh, eighth}
			sum := 0
			for position, digit := range body {
				sum += digit * weights[position]
			}
			if sum%11 != 10 {
				continue
			}
			folded++

			bare := digitsOf(body) + "0"
			if verdict := (pl.REGON{}).Validate(bare); !verdict.Valid {
				t.Fatalf("%q leaves a remainder of ten and nought should complete it: %s", bare, verdict.Reason)
			}
			if findings := oracle.CheckREGON(bare); len(findings) != 0 {
				t.Fatalf("the independent validator disagrees about %q: %v", bare, findings)
			}
		}
	}

	if folded == 0 {
		// Without this the test could pass by never reaching the case at all,
		// which is the shape of a test that measures nothing.
		t.Fatal("no body in the swept range left a remainder of ten, so this test checked nothing")
	}
	t.Logf("bodies reaching the folded residue in the swept range: %d", folded)
}

// Recomputing never refuses, which is the other half of the same property.
//
// The tax number has to refuse for one body in eleven because no digit
// completes it. Here every body completes, so a mutator that moved a digit can
// always ask for a consistent number back.
func TestRecomputingTheCheckDigitNeverRefuses(t *testing.T) {
	source := seed.New("release-2026-08")
	for row := int64(0); row < 2000; row++ {
		value := generateREGON(t, source, row)
		for position := 0; position < 8; position++ {
			for replacement := '0'; replacement <= '9'; replacement++ {
				damaged := []byte(value)
				damaged[position] = byte(replacement)

				repaired, err := (pl.REGON{}).WithCheckDigits(string(damaged))
				if err != nil {
					t.Fatalf("row %d: recomputing %q refused: %v", row, damaged, err)
				}
				if findings := oracle.CheckREGON(repaired); len(findings) != 0 {
					t.Fatalf("row %d: %q was rebuilt into %q, which the validator rejects: %v",
						row, damaged, repaired, findings)
				}
			}
		}
	}
}

// What this check digit does not catch, stated as an exact property rather than
// as a percentage.
//
// Folding two residues onto one digit means the map from remainder to digit is
// not one to one, so some single digit errors survive. They survive in exactly
// one circumstance and nowhere else: when the remainder moves between nought
// and ten, the two residues sharing a digit. Anything relying on a corruption
// being noticed has to know this, so it is a test rather than a sentence.
func TestASingleDigitErrorSurvivesExactlyWhenTheResidueFoldsOntoNought(t *testing.T) {
	weights := [8]int{8, 9, 2, 3, 4, 5, 6, 7}
	remainderOf := func(body []int) int {
		sum := 0
		for position, digit := range body {
			sum += digit * weights[position]
		}
		return sum % 11
	}

	source := seed.New("release-2026-08")
	survived, examined := 0, 0

	for row := int64(0); row < 1500; row++ {
		value := generateREGON(t, source, row)
		body := make([]int, 8)
		for index := 0; index < 8; index++ {
			body[index] = int(value[index] - '0')
		}
		before := remainderOf(body)

		for position := 0; position < 8; position++ {
			original := body[position]
			for replacement := 0; replacement <= 9; replacement++ {
				if replacement == original {
					continue
				}
				examined++
				body[position] = replacement
				after := remainderOf(body)
				body[position] = original

				unnoticed := foldsToSameDigit(before, after)
				damaged := []byte(value)
				damaged[position] = byte('0' + replacement)
				accepted := len(oracle.CheckREGON(string(damaged))) == 0

				if accepted != unnoticed {
					t.Fatalf("%q with position %d set to %d: validator accepted=%v, but the residues %d and %d say %v",
						value, position, replacement, accepted, before, after, unnoticed)
				}
				if unnoticed {
					survived++
				}
			}
		}
	}

	if survived == 0 {
		t.Fatal("no single digit error survived, so this test never reached the case it describes")
	}
	t.Logf("single digit errors surviving the check digit: %d of %d (%.3f%%)",
		survived, examined, 100*float64(survived)/float64(examined))
}

// foldsToSameDigit says whether two remainders are written as the same digit.
// Only nought and ten are, and that is the whole of the weakness.
func foldsToSameDigit(before, after int) bool {
	if before == after {
		return false
	}
	return (before == 0 || before == 10) && (after == 0 || after == 10)
}

// A well formed local unit number is refused for the right reason.
//
// It is a real REGON, so saying it is malformed would report a defect in a
// value that has none. The reason names the limit of this build instead.
func TestALocalUnitNumberIsRefusedByNameRatherThanAsMalformed(t *testing.T) {
	verdict := (pl.REGON{}).Validate("12345678512347")
	if verdict.Valid {
		t.Fatal("a fourteen digit number was accepted, but this build does not check that form")
	}
	if !strings.Contains(verdict.Reason, "fourteen") || !strings.Contains(verdict.Reason, "local unit") {
		t.Fatalf("the reason should name the local unit form rather than call the value malformed, got %q", verdict.Reason)
	}
}

// Separators are refused, and the contrast with the tax number is deliberate.
// That one is written with hyphens on invoices and this one is not, so there is
// no second shape to accept.
func TestTheStatisticalNumberIsRefusedWhenWrittenWithSeparators(t *testing.T) {
	for _, written := range []string{"123-456-785", "12345678 5", "1234567 85", "123456785 "} {
		if verdict := (pl.REGON{}).Validate(written); verdict.Valid {
			t.Fatalf("%q was accepted, but this number is written as bare digits", written)
		}
	}
}

// The contract, in both directions. A capability declared without its interface
// and an interface without its declaration are both defects.
func TestTheContractHoldsForTheStatisticalNumber(t *testing.T) {
	if findings := provider.Check(pl.REGON{}); len(findings) != 0 {
		t.Fatalf("the provider does not meet the contract: %v", findings)
	}
	if id := (pl.REGON{}).ID(); id != "PL.regon" {
		t.Fatalf("the identifier is %q and the catalog names PL.regon", id)
	}
}

// Where the check digit sits, asked of a value that has one and of one that
// does not.
func TestTheStatisticalNumberSaysWhereItsCheckDigitSits(t *testing.T) {
	positions, ok := (pl.REGON{}).CheckDigits("123456785")
	if !ok || len(positions) != 1 || positions[0] != 8 {
		t.Fatalf("expected the ninth digit, got %v (ok=%v)", positions, ok)
	}
	if _, ok := (pl.REGON{}).CheckDigits("1234"); ok {
		t.Fatal("a value too short to have a check digit reported that it has one")
	}
}

// digitsOf writes digits back out, so the test does not depend on a helper
// inside the package it is testing.
func digitsOf(digits []int) string {
	written := make([]byte, 0, len(digits))
	for _, digit := range digits {
		written = append(written, byte('0'+digit))
	}
	return string(written)
}
