package pl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// # The quietest defect this project had
//
// A bound gender the provider could not read returned the same answer as no
// binding at all, which means draw one. So a recipe declaring that the personal
// number agrees with the gender column, and writing that column as kobieta or
// other, received a number carrying a drawn gender that contradicted the row,
// with nothing said anywhere.
//
// It matters more here than it would elsewhere. This tool sells the ability to
// produce an identifier that is well formed and disagrees with its record as a
// deliberate corruption named in the manifest. Producing one by accident and
// reporting nothing is that promise inverted.
//
// The date half of the same agreement already failed loudly for a value it
// could not read. This is the two halves answering alike.
func TestAGenderThisSchemeCannotReadIsRefusedRatherThanDrawn(t *testing.T) {
	unreadable := []string{"kobieta", "mężczyzna", "other", "1", "F.", "", " "}

	for _, written := range unreadable {
		t.Run(written, func(t *testing.T) {
			_, err := pl.PESEL{}.Generate(request(1, map[string]string{
				provider.RoleGender: written,
			}))
			if err == nil {
				t.Fatalf("a gender written %q was accepted and quietly drawn instead", written)
			}
			if !strings.Contains(err.Error(), "is not a gender this scheme can encode") {
				t.Fatalf("the refusal did not say what was wrong: %v", err)
			}
			if !strings.Contains(err.Error(), "female") || !strings.Contains(err.Error(), "male") {
				t.Fatalf("the refusal did not name what would be accepted: %v", err)
			}
		})
	}
}

// The spellings the scheme does read have to keep working, or the refusal above
// is rejecting the format rather than a mistake.
func TestTheGenderSpellingsTheSchemeReadsStillWork(t *testing.T) {
	cases := map[string]provider.Gender{
		"female": provider.Female,
		"FEMALE": provider.Female,
		" f ":    provider.Female,
		"k":      provider.Female,
		"male":   provider.Male,
		"M":      provider.Male,
	}
	for written, want := range cases {
		t.Run(written, func(t *testing.T) {
			value, err := pl.PESEL{}.Generate(request(7, map[string]string{
				provider.RoleGender: written,
			}))
			if err != nil {
				t.Fatalf("a gender written %q was refused: %v", written, err)
			}
			got, ok := pl.PESEL{}.EncodedGender(value)
			if !ok || got != want {
				t.Fatalf("a gender written %q produced %v", written, got)
			}
		})
	}
}

// Binding nothing still draws, because defaulting would make every unbound
// record the same sex and that is a dataset nobody wants to test with.
func TestAnUnboundGenderIsStillDrawnRatherThanRefused(t *testing.T) {
	if _, err := (pl.PESEL{}).Generate(request(3, nil)); err != nil {
		t.Fatalf("a request binding no gender was refused: %v", err)
	}
}

// # Where the check digit sits is a question about a number
//
// This answered from the length alone, so eleven characters of anything got an
// offset back. A mutator would then have written into a value the scheme does
// not describe. The statistical number already asked the stricter question.
func TestTheCheckDigitPositionIsRefusedForValuesThatAreNotDigits(t *testing.T) {
	refused := []string{"abcdefghijk", "1234567890x", "  123456789", "1234567890"}
	for _, value := range refused {
		if _, ok := (pl.PESEL{}).CheckDigits(value); ok {
			t.Errorf("%q was given a check digit position", value)
		}
	}
	if positions, ok := (pl.PESEL{}).CheckDigits("02070803628"); !ok || len(positions) != 1 || positions[0] != 10 {
		t.Errorf("a real number lost its check digit position: %v %v", positions, ok)
	}
}

// # Three answers, where there used to be two
//
// The fourteen digit form of the statistical number is a real number this build
// does not check. Answering that with the same verdict as a broken value put a
// correct number and a faulty one in the same bucket, and the difference lived
// only in prose that nothing reads programmatically. In the mode where the
// column arrives from somebody else's file, that is reporting a defect in a
// value that has none.
func TestALocalUnitNumberIsUnsupportedRatherThanInvalid(t *testing.T) {
	verdict := pl.REGON{}.Validate("12345678512347")

	if verdict.Valid {
		t.Fatal("a form this build does not check was called valid")
	}
	if !verdict.Unsupported {
		t.Fatal("a real number this build does not check was reported as a defect")
	}
	if !strings.Contains(verdict.Reason, "local unit") {
		t.Fatalf("the reason did not name the form: %q", verdict.Reason)
	}
}

// A value that is genuinely wrong stays wrong, or the state above has become a
// way of passing everything.
func TestABrokenStatisticalNumberIsStillInvalidRatherThanUnsupported(t *testing.T) {
	for _, value := range []string{"123456780", "12345", "12345678a", "123456789012"} {
		verdict := pl.REGON{}.Validate(value)
		if verdict.Valid {
			t.Errorf("%q was called valid", value)
		}
		if verdict.Unsupported {
			t.Errorf("%q was excused as a form this build does not check", value)
		}
	}
}

// A correct nine digit number is still plainly valid, and carries neither of
// the other two states.
func TestACorrectStatisticalNumberIsPlainlyValid(t *testing.T) {
	value, err := pl.REGON{}.Generate(request(11, nil))
	if err != nil {
		t.Fatalf("generating failed: %v", err)
	}
	verdict := pl.REGON{}.Validate(value)
	if !verdict.Valid || verdict.Unsupported || verdict.Reason != "" {
		t.Fatalf("a generated number got %+v", verdict)
	}
}
