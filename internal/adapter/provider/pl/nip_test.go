package pl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

func nipRequest(row int64) provider.Request {
	return provider.Request{
		Source: seed.New("release-2026-08"),
		At:     seed.Coordinate{Entity: "company", Row: row, Field: "nip", Purpose: "value"},
	}
}

func generateNIP(t *testing.T, row int64) string {
	t.Helper()
	value, err := pl.NIP{}.Generate(nipRequest(row))
	if err != nil {
		t.Fatalf("row %d: %v", row, err)
	}
	return value
}

func TestTheTaxNumberProviderMeetsItsContract(t *testing.T) {
	if findings := provider.Check(pl.NIP{}); len(findings) != 0 {
		t.Fatalf("the contract is not met: %v", findings)
	}
	if id := (pl.NIP{}).ID(); id != "PL.nip" {
		t.Errorf("the identifier is %q", id)
	}
}

// # The example the source works out by hand
//
// This is the check the PESEL never had. The country sheet the Ministry of
// Finance publishes on the European TIN Portal takes one number apart digit by
// digit and states the answer, so the weights in this repository are compared
// against arithmetic somebody else did rather than against a second telling of
// our own.
//
// Both the products and the total are restated here. A test asserting only the
// final digit would still pass with two compensating errors in the table, which
// is exactly the shape a mistyped weight takes.
func TestTheWorkedExampleFromTheSourceIsReproduced(t *testing.T) {
	const published = "2234567895"

	if verdict := (pl.NIP{}).Validate(published); !verdict.Valid {
		t.Fatalf("the number the source works out is refused by our validator: %s", verdict.Reason)
	}
	if findings := oracle.CheckNIP(published); len(findings) != 0 {
		t.Fatalf("the number the source works out is refused by the independent validator: %v", findings)
	}

	// 2*6=12, 2*5=10, 3*7=21, 4*2=8, 5*3=15, 6*4=24, 7*5=35, 8*6=48, 9*7=63
	products := []int{12, 10, 21, 8, 15, 24, 35, 48, 63}
	weights := []int{6, 5, 7, 2, 3, 4, 5, 6, 7}
	total := 0
	for position, want := range products {
		if got := int(published[position]-'0') * weights[position]; got != want {
			t.Errorf("position %d multiplies out to %d, the source says %d", position+1, got, want)
		}
		total += want
	}
	if total != 236 {
		t.Errorf("the products add up to %d, the source says 236", total)
	}
	if total%11 != 5 {
		t.Errorf("the remainder is %d, the source says 5", total%11)
	}
	if published[9] != '5' {
		t.Errorf("the check digit in the published number is %c, the source says 5", published[9])
	}
}

// Every generated value is checked by a validator written from the published
// description rather than from this generator. Two implementations by one hand
// in one hour agree with each other when the hand is wrong.
func TestEveryGeneratedTaxNumberSatisfiesTheIndependentValidator(t *testing.T) {
	for row := int64(0); row < 500; row++ {
		value := generateNIP(t, row)
		if findings := oracle.CheckNIP(value); len(findings) != 0 {
			t.Fatalf("row %d produced %s, which the independent validator refuses: %v", row, value, findings)
		}
		if verdict := (pl.NIP{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %s, which our own validator refuses: %s", row, value, verdict.Reason)
		}
		if len(value) != 10 {
			t.Fatalf("row %d produced %q, and the emitted form is ten bare digits", row, value)
		}
	}
}

func TestTheSameAddressAlwaysGivesTheSameTaxNumber(t *testing.T) {
	first := generateNIP(t, 42)
	for row := int64(0); row < 20; row++ {
		generateNIP(t, row)
	}
	if second := generateNIP(t, 42); first != second {
		t.Fatalf("the same place gave %s and then %s", first, second)
	}
}

// # The hole eleven leaves, walked in full
//
// Eleven residues do not fit into ten digits, so for some nine digit bodies no
// check digit exists at all. The claim this generator rests on is that the hole
// costs at most one choice of the ninth digit, never all ten, which is why the
// draw needs no retry loop.
//
// The claim depends on the running total only through its remainder, and there
// are eleven of those. So this walks every one of them rather than sampling
// bodies: where a domain is that small, exhaustive leaves nothing to draw.
func TestAtMostOneNinthDigitIsEverForbidden(t *testing.T) {
	const ninthWeight = 7

	for remainder := 0; remainder < 11; remainder++ {
		allowed := 0
		for digit := 0; digit <= 9; digit++ {
			if (remainder+digit*ninthWeight)%11 != 10 {
				allowed++
			}
		}
		if allowed < 9 {
			t.Fatalf("with the first eight digits leaving %d, only %d of ten ninth digits work", remainder, allowed)
		}
	}
}

// A body that no check digit can complete, refused by name rather than by a
// mismatch that would read as an ordinary typo.
//
// Nine hundred million is such a body: nine times the first weight is fifty
// four, and fifty four leaves ten.
func TestABodyNoCheckDigitCanCompleteIsRefusedByName(t *testing.T) {
	for digit := byte('0'); digit <= '9'; digit++ {
		value := "900000000" + string(digit)
		verdict := pl.NIP{}.Validate(value)
		if verdict.Valid {
			t.Fatalf("%s was accepted, and no NIP starts with those nine digits", value)
		}
		if !strings.Contains(verdict.Reason, "remainder of ten") {
			t.Errorf("%s was refused with %q, which does not name the reason", value, verdict.Reason)
		}
		if findings := oracle.CheckNIP(value); len(findings) == 0 {
			t.Errorf("the independent validator accepted %s", value)
		}
	}

	if _, err := (pl.NIP{}).WithCheckDigits("9000000000"); err == nil {
		t.Error("recomputing the check digit succeeded for a body that has none")
	}
}

// Every single digit error changes the answer. Eleven is prime and no weight is
// a multiple of it, so this is a property of the scheme rather than of the
// sample, and it holds for the check digit position too.
func TestEverySingleDigitErrorChangesTheTaxCheckDigit(t *testing.T) {
	for row := int64(0); row < 200; row++ {
		base := generateNIP(t, row)
		for position := 0; position < 10; position++ {
			for replacement := byte('0'); replacement <= '9'; replacement++ {
				if base[position] == replacement {
					continue
				}
				damaged := base[:position] + string(replacement) + base[position+1:]
				if findings := oracle.CheckNIP(damaged); len(findings) == 0 {
					t.Fatalf("changing position %d of %s to %c left a valid number", position, base, replacement)
				}
			}
		}
	}
}

// # The difference from the PESEL, stated as arithmetic rather than as a sample
//
// The catalog warns that some check digit algorithms miss a transposition
// entirely, and the PESEL is one of them: its weights repeat, so certain swaps
// of neighbouring digits leave the total alone. This scheme does not have that
// hole among the first nine, and the reason is that eleven is prime while every
// difference of neighbouring weights is small and non zero.
//
// Walked in full over positions and digit pairs, because eight hundred cases is
// nothing and a sampled version would prove only that the cases somebody chose
// happen to work.
func TestEveryAdjacentTranspositionInTheBodyIsCaught(t *testing.T) {
	weights := []int{6, 5, 7, 2, 3, 4, 5, 6, 7}
	checked := 0

	for position := 0; position < 8; position++ {
		difference := weights[position] - weights[position+1]
		for left := 0; left <= 9; left++ {
			for right := 0; right <= 9; right++ {
				// Swapping moves each digit onto the other's weight, so the
				// total shifts by the difference of the digits times the
				// difference of the weights.
				if shift := (left - right) * difference; left != right && shift%11 == 0 {
					t.Fatalf("swapping %d and %d at position %d leaves the total unchanged modulo eleven",
						left, right, position+1)
				}
				checked++
			}
		}
	}

	if checked != 800 {
		t.Fatalf("only %d pairs were walked, so the loop is not covering the scheme", checked)
	}
}

// The swap across the boundary between the body and the check digit is a
// different case, and it is measured rather than assumed. It is the one place
// where the catalog note about undetected transpositions could still bite.
func TestTheSwapAcrossTheCheckDigitIsMeasuredRatherThanAssumed(t *testing.T) {
	survived := 0
	for row := int64(0); row < 500; row++ {
		base := generateNIP(t, row)
		if base[8] == base[9] {
			continue
		}
		swapped := base[:8] + string(base[9]) + string(base[8])
		if findings := oracle.CheckNIP(swapped); len(findings) == 0 {
			survived++
		}
	}
	// Recorded rather than asserted to be zero. If a later change to the scheme
	// makes this number move, the log says so instead of a test failing for a
	// reason nobody wrote down.
	t.Logf("%d of 500 swaps across the check digit boundary survived", survived)
}

// Poland writes the same ten digits three ways, and all three are the same
// number. The shape a value arrives in survives, because reformatting it would
// be a second change nobody asked for.
func TestTheThreeWaysOfWritingOneNumberAreAllAccepted(t *testing.T) {
	const bare = "1234563218"
	forms := []string{bare, "123-456-32-18", "123-45-63-218"}

	for _, form := range forms {
		t.Run(form, func(t *testing.T) {
			if verdict := (pl.NIP{}).Validate(form); !verdict.Valid {
				t.Fatalf("%s was refused: %s", form, verdict.Reason)
			}
			if findings := oracle.CheckNIP(form); len(findings) != 0 {
				t.Fatalf("the independent validator refused %s: %v", form, findings)
			}

			positions, ok := (pl.NIP{}).CheckDigits(form)
			if !ok || len(positions) != 1 || positions[0] != len(form)-1 {
				t.Fatalf("the check digit of %s was placed at %v", form, positions)
			}

			rebuilt, err := pl.NIP{}.WithCheckDigits(form)
			if err != nil {
				t.Fatalf("recomputing the check digit of %s failed: %v", form, err)
			}
			if rebuilt != form {
				t.Errorf("%s came back as %s, so the written shape did not survive", form, rebuilt)
			}
		})
	}
}

// Separators anywhere other than where the two conventions put them are not a
// third convention, they are a value we do not accept. Said with a test so that
// a looser parser cannot arrive later without somebody noticing.
func TestSeparatorsInTheWrongPlaceAreRefused(t *testing.T) {
	for _, value := range []string{
		"1-2-3-4-5-6-7-8-9-0",
		"12-3456-32-18",
		"123-456-3218",
		"123 456 32 18",
		"123-456-32-1",
		"-1234563218",
		"1234563218-",
	} {
		if verdict := (pl.NIP{}).Validate(value); verdict.Valid {
			t.Errorf("%q was accepted", value)
		}
		if findings := oracle.CheckNIP(value); len(findings) == 0 {
			t.Errorf("the independent validator accepted %q", value)
		}
	}
}

func TestValidateRefusesWhatTheTaxSchemeForbids(t *testing.T) {
	good := generateNIP(t, 7)

	cases := []struct {
		name  string
		value string
		says  string
	}{
		{"too short", good[:9], "ten digits"},
		{"too long", good + "0", "ten digits"},
		{"empty", "", "ten digits"},
		{"not digits", good[:9] + "x", "ten digits"},
		{"broken check digit", flipLast(good), "check digit"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			verdict := pl.NIP{}.Validate(test.value)
			if verdict.Valid {
				t.Fatalf("%q was accepted", test.value)
			}
			if !strings.Contains(verdict.Reason, test.says) {
				t.Errorf("the reason was %q, expected it to mention %q", verdict.Reason, test.says)
			}
			if findings := oracle.CheckNIP(test.value); len(findings) == 0 {
				t.Errorf("the independent validator accepted %q", test.value)
			}
		})
	}
}

// The ninth digit is chosen rather than drawn freely, so the values it takes
// have to be spread over what is available instead of falling on one of them.
// A constrained draw that quietly collapses to a constant would leave every
// other test in this file green.
func TestTheConstrainedNinthDigitStillVaries(t *testing.T) {
	seen := make(map[byte]bool)
	for row := int64(0); row < 500; row++ {
		seen[generateNIP(t, row)[8]] = true
	}
	if len(seen) < 9 {
		t.Fatalf("the ninth digit took only %d different values over five hundred rows", len(seen))
	}
}
