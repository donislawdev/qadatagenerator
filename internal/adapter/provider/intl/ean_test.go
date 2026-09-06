package intl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// theBookAnchor is the calculation the ISBN Users' Manual works in Appendix
// A1.1, with the check digit the manual arrives at put back in last place.
//
// The manual writes it as 978-0-11-000222-4. An ISBN-13 is a GTIN-13 carrying
// the prefix 978, so this is an article number and this scheme reads it.
const theBookAnchor = "9780110002224"

// theBookAnchorSum is the total the manual prints at step 1, before it takes
// the sum anywhere.
//
// Held separately from the check digit on purpose. A worked example that only
// gives an answer lets a wrong sum and a wrong fold cancel each other out. This
// one publishes the number in the middle, so both halves are pinned.
const theBookAnchorSum = 56

// theStandardsOwnNumbers are article numbers GS1 prints in the current release
// of its own specification.
//
// They are quoted rather than generated, for the reason the tax number keeps
// its worked example: values derived from our arithmetic would agree with our
// arithmetic and prove nothing about the source. They carry a second meaning
// here as well. Every one of them sits in the prefix this scheme treats as its
// safe harbour, which is the behaviour that supports the harbour far better
// than the table row does.
var theStandardsOwnNumbers = []string{
	"9521234500001",
	"9524000059109",
	"9520123456788",
	"9521234500018",
}

// generateEAN produces the number for one row, failing the test rather than
// handing back an error nobody looks at.
func generateEAN(t *testing.T, source seed.Source, row int64) string {
	t.Helper()
	value, err := (intl.EAN{}).Generate(provider.Request{
		Source: source,
		At:     seed.Coordinate{Entity: "catalogue", Row: row, Field: "barcode", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: the generator refused: %v", row, err)
	}
	return value
}

// The strongest anchor a source can give: it does the arithmetic itself. Two
// documents do it here, written independently of one another, and one of them
// publishes the number it reaches halfway through.
func TestTheExamplesTheStandardsWorkComeOutTheSameHere(t *testing.T) {
	// The intermediate value first, because an answer that agrees while the
	// working disagrees is the case a single comparison would miss.
	if sum := weightedSumOf(theBookAnchor[:12]); sum != theBookAnchorSum {
		t.Errorf("the manual prints a sum of %d at step 1 and the same weights give %d here",
			theBookAnchorSum, sum)
	}

	for _, anchor := range append([]string{theBookAnchor}, theStandardsOwnNumbers...) {
		if verdict := (intl.EAN{}).Validate(anchor); !verdict.Valid {
			t.Errorf("the number %s is printed by its standard and was refused: %s", anchor, verdict.Reason)
		}
		rebuilt, err := (intl.EAN{}).WithCheckDigits(anchor)
		if err != nil {
			t.Errorf("recomputing the check digit of %s failed: %v", anchor, err)
			continue
		}
		if rebuilt != anchor {
			t.Errorf("the published number %s rebuilt with its own check digit became %s", anchor, rebuilt)
		}
		if findings := oracle.CheckEAN(anchor); len(findings) != 0 {
			t.Errorf("the independent validator refused the published number %s: %v", anchor, findings)
		}
	}
}

// weightedSumOf is step 1 of the published calculation, written here so the
// test can compare against the number the manual prints rather than only
// against its final answer.
func weightedSumOf(body string) int {
	sum := 0
	for back := 0; back < len(body); back++ {
		weight := 1
		if back%2 == 0 {
			weight = 3
		}
		sum += int(body[len(body)-1-back]-'0') * weight
	}
	return sum
}

// Every value this build produces has to satisfy the validator written from the
// standard rather than from the generator.
func TestEveryGeneratedArticleNumberSatisfiesTheIndependentValidator(t *testing.T) {
	source := seed.New("ean-sweep-2026-09")
	for row := int64(0); row < 5000; row++ {
		value := generateEAN(t, source, row)
		if len(value) != 13 {
			t.Fatalf("row %d produced %q, which is %d characters", row, value, len(value))
		}
		if findings := oracle.CheckEAN(value); len(findings) != 0 {
			t.Fatalf("row %d produced %q and the independent validator says %v", row, value, findings)
		}
	}
}

// The claim this scheme makes that most schemes here cannot: nothing it writes
// can belong to a real article.
//
// This is the guard on that claim. It is checked on the value rather than on
// the capability list, because a capability declared by a provider that then
// generates outside its own harbour would be worse than declaring nothing.
func TestEveryGeneratedArticleNumberSitsInTheDemonstrationPrefix(t *testing.T) {
	source := seed.New("ean-harbour-2026-09")
	for row := int64(0); row < 5000; row++ {
		value := generateEAN(t, source, row)
		if !(intl.EAN{}).InReservedRange(value) {
			t.Fatalf("row %d produced %q, which the provider does not call reserved", row, value)
		}
		if !strings.HasPrefix(value, oracle.EANDemonstrationPrefix()) {
			t.Fatalf("row %d produced %q, which is outside the prefix the standard sets aside", row, value)
		}
	}
}

// The prefix the provider generates into has to be the prefix the standard
// publishes, and the comparison has to be against the published row rather than
// against a second copy of the same three characters.
func TestTheDemonstrationPrefixIsTheOneTheStandardPublishes(t *testing.T) {
	published := oracle.EANDemonstrationPrefix()
	if published == "" {
		t.Fatal("the published prefix row did not come apart into anything")
	}
	value := generateEAN(t, seed.New("ean-prefix-2026-09"), 0)
	if !strings.HasPrefix(value, published) {
		t.Errorf("the standard sets aside %q and this build generates %q", published, value)
	}
	// A number one digit outside the harbour is still a correct article number.
	// If this ever fails, the harbour has been confused with validity.
	outside := "9531234500009"
	if (intl.EAN{}).InReservedRange(outside) {
		t.Errorf("%s is outside the demonstration prefix and was called reserved", outside)
	}
}

// Validate is deliberately wider than Generate. A real article number from
// anywhere in the numbering space is not wrong, and calling it wrong would make
// this tool report a defect in somebody's correct data.
func TestValidateAcceptsCorrectNumbersOutsideTheDemonstrationPrefix(t *testing.T) {
	for _, value := range []string{theBookAnchor, "4006381333931", "0000000000000"} {
		if verdict := (intl.EAN{}).Validate(value); !verdict.Valid {
			t.Errorf("%s is a well formed article number and was refused: %s", value, verdict.Reason)
		}
	}
}

// Thirteen noughts is a correct article number, and the arithmetic is the whole
// reason: the sum is nought, the nearest equal or higher multiple of ten is
// nought, and the check digit is nought. The test confirms it rather than
// stepping around it.
func TestThirteenNoughtsAreACorrectNumberBecauseTheArithmeticSaysSo(t *testing.T) {
	const noughts = "0000000000000"
	if verdict := (intl.EAN{}).Validate(noughts); !verdict.Valid {
		t.Errorf("thirteen noughts were refused: %s", verdict.Reason)
	}
	if findings := oracle.CheckEAN(noughts); len(findings) != 0 {
		t.Errorf("the independent validator refused thirteen noughts: %v", findings)
	}
}

// The shapes this scheme does not have, each refused with a rule named rather
// than with a bare no.
func TestTheShapesAnArticleNumberDoesNotHaveAreRefused(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"twelve digits", "952123450000"},
		{"fourteen digits", "95212345000012"},
		{"a letter in the middle", "95212345X0001"},
		{"a space", "952123450 001"},
		{"a sign", "-952123450001"},
		{"a multibyte character", "952123450001é"},
		{"the check digit moved on by one", "9521234500002"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			verdict := (intl.EAN{}).Validate(test.value)
			if verdict.Valid {
				t.Fatalf("%q was accepted", test.value)
			}
			if verdict.Reason == "" {
				t.Error("refused without saying why, which is the silence this project does not allow")
			}
			if findings := oracle.CheckEAN(test.value); len(findings) == 0 {
				t.Errorf("the independent validator accepted %q", test.value)
			}
		})
	}
}

// Recomputing the check digit never refuses for arithmetic reasons. Ten
// residues fit in ten digits, so every twelve digit body has a completion and a
// mutator that moved a digit always gets a consistent value back.
func TestRecomputingTheArticleCheckDigitNeverRefuses(t *testing.T) {
	source := seed.New("ean-rebuild-2026-09")
	for row := int64(0); row < 2000; row++ {
		value := generateEAN(t, source, row)
		for at := 0; at < 12; at++ {
			moved := []byte(value)
			moved[at] = '0' + (moved[at]-'0'+1)%10
			rebuilt, err := (intl.EAN{}).WithCheckDigits(string(moved))
			if err != nil {
				t.Fatalf("row %d with position %d moved: %v", row, at, err)
			}
			if verdict := (intl.EAN{}).Validate(rebuilt); !verdict.Valid {
				t.Fatalf("row %d rebuilt to %q which is invalid: %s", row, rebuilt, verdict.Reason)
			}
		}
	}
}

// Moving any one digit of the body has to break the check digit, because that
// is the corruption the catalog aims at this scheme and a mutator whose damage
// went unnoticed would be worse than no mutator.
func TestMovingASingleDigitBreaksTheCheckDigit(t *testing.T) {
	source := seed.New("ean-damage-2026-09")
	for row := int64(0); row < 500; row++ {
		value := generateEAN(t, source, row)
		for at := 0; at < 12; at++ {
			damaged := []byte(value)
			damaged[at] = '0' + (damaged[at]-'0'+1)%10
			if verdict := (intl.EAN{}).Validate(string(damaged)); verdict.Valid {
				t.Fatalf("row %d: moving position %d of %q left a value the scheme still accepts",
					row, at, value)
			}
		}
	}
}

// The same address has to give the same number, on every platform and every
// run. Without it a corrupted dataset cannot be used in continuous integration,
// which is the whole point of the tool.
func TestTheSameRequestProducesTheSameArticleNumber(t *testing.T) {
	for row := int64(0); row < 200; row++ {
		first := generateEAN(t, seed.New("ean-repeat"), row)
		second := generateEAN(t, seed.New("ean-repeat"), row)
		if first != second {
			t.Fatalf("row %d gave %q and then %q", row, first, second)
		}
	}
}

// The check digit is reported where the standard puts it, and only for a value
// of the right length.
func TestTheCheckDigitIsReportedAtTheLastPosition(t *testing.T) {
	value := generateEAN(t, seed.New("ean-positions"), 3)
	positions, ok := (intl.EAN{}).CheckDigits(value)
	if !ok {
		t.Fatal("a generated number has no check digit position")
	}
	if len(positions) != 1 || positions[0] != 12 {
		t.Errorf("the check digit was reported at %v", positions)
	}
	if _, ok := (intl.EAN{}).CheckDigits("952"); ok {
		t.Error("a value that is not an article number was given a check digit position")
	}
}

// The provider has to satisfy the contract every provider satisfies, including
// that each capability it declares is backed by the interface that capability
// requires.
func TestTheProviderKeepsTheContract(t *testing.T) {
	if findings := provider.Check(intl.EAN{}); len(findings) != 0 {
		t.Errorf("the provider breaks its contract: %v", findings)
	}
	if id := (intl.EAN{}).ID(); id != "ean" {
		t.Errorf("the catalog published the name ean before this file existed and the provider answers %q", id)
	}
	if ranges := (intl.EAN{}).ReservedRanges(); len(ranges) == 0 {
		t.Error("the provider declares a reserved range and describes none")
	}
}
