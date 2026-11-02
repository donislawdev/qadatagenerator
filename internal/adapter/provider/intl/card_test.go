package intl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// generateCard produces the number for one row, failing the test rather than
// handing back an error nobody looks at.
func generateCard(t *testing.T, source seed.Source, row int64) string {
	t.Helper()
	value, err := (intl.Card{}).Generate(provider.Request{
		Source: source,
		At:     seed.Coordinate{Entity: "payment", Row: row, Field: "card", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: the generator refused: %v", row, err)
	}
	return value
}

// zeros is a value of nought digits, which satisfies the check digit at every
// length and is nobody's card.
//
// The tests below need correct values of lengths this build does not write,
// and a number built to satisfy the formula is exactly what the handbook keeps
// out of a dataset, because it can be somebody's card. A run of noughts is the
// one such value that cannot be.
func zeros(n int) string { return strings.Repeat("0", n) }

// 🔴 The anchor of the arithmetic. Eight numbers published as test numbers by
// at least two of four payment providers all satisfy it, which a value of the
// same length does one time in ten, so eight in a row by chance happen about
// once in a hundred million. Both validators are asked, because they compute
// the same rule two different ways.
func TestEveryPublishedTestNumberSatisfiesTheCheckDigit(t *testing.T) {
	published := (intl.Card{}).Published()
	if len(published) != 8 {
		t.Fatalf("the list holds %d numbers, and DEC-268 admitted eight", len(published))
	}
	for _, number := range published {
		if verdict := (intl.Card{}).Validate(number); !verdict.Valid {
			t.Errorf("the published test number %s was refused: %s", number, verdict.Reason)
		}
		if findings := oracle.CheckCard(number); len(findings) != 0 {
			t.Errorf("the independent validator refused the published test number %s: %v", number, findings)
		}
	}
}

// 🔴 The published numbers decide against the words of the one source that
// spells the rule out. FAQ 1137 of the PCI Security Standards Council says to
// take nine off a doubled value greater than ten, which leaves a doubled five
// at ten. The Mastercard number three providers publish is refused that way and
// accepted by the rule both validators here follow.
func TestTheMastercardNumberRefusesTheLiteralWordingOfTheSource(t *testing.T) {
	const mastercard = "5555555555554444"
	if total := oracle.CardWeightedTotal(mastercard); total%10 != 0 {
		t.Fatalf("the published Mastercard number gives a total of %d, which does not divide by ten", total)
	}
	literal := 0
	for back := 0; back < len(mastercard); back++ {
		digit := int(mastercard[len(mastercard)-1-back] - '0')
		if back%2 == 1 {
			digit *= 2
			if digit > 10 {
				digit -= 9
			}
		}
		literal += digit
	}
	if literal%10 == 0 {
		t.Error("the literal wording of the FAQ accepts the Mastercard number, so the case that " +
			"tells the two readings apart is no longer this one")
	}
}

// Every value this build produces is one of the published numbers, and every
// published number is produced. A number never drawn is a list entry nothing
// exercises, and a value off the list is a number nobody published.
func TestEveryGeneratedCardIsAPublishedTestNumber(t *testing.T) {
	published := map[string]bool{}
	for _, number := range (intl.Card{}).Published() {
		published[number] = false
	}
	source := seed.New("card-sweep-2026-11")
	for row := int64(0); row < 2000; row++ {
		value := generateCard(t, source, row)
		drawn, known := published[value]
		if !known {
			t.Fatalf("row %d produced %q, which is not a published test number", row, value)
		}
		if !drawn {
			published[value] = true
		}
		if findings := oracle.CheckCard(value); len(findings) != 0 {
			t.Fatalf("row %d produced %q and the independent validator says %v", row, value, findings)
		}
	}
	for number, drawn := range published {
		if !drawn {
			t.Errorf("%s is on the list and two thousand rows never drew it", number)
		}
	}
}

// The claim this provider deliberately does not make.
func TestTheCardClaimsNoSafeHarbour(t *testing.T) {
	for _, claimed := range (intl.Card{}).Capabilities() {
		if claimed == capability.IdentifierReservedRange {
			t.Error("nothing says no issuer has given a test number to a real card, and the provider declares a reserved range")
		}
		if claimed == capability.IdentifierUnassignedTarget {
			t.Error("this scheme has no block a value could be moved into and the provider declares one")
		}
	}
	if _, claims := interface{}(intl.Card{}).(provider.RangeAware); claims {
		t.Error("the provider implements the reserved range interface for a scheme with no harbour")
	}
}

// 🔴 The regression this provider would have caused. An equipment identity is
// fifteen digits under the same formula, and the mode that reads somebody
// else's file refuses to choose between two schemes accepting a whole column.
// A card validator approving it would make every column of IMEI values
// unrecognisable.
func TestAnEquipmentIdentityIsNotACardThisBuildChecks(t *testing.T) {
	source := seed.New("card-against-imei")
	for row := int64(0); row < 500; row++ {
		value, err := (intl.IMEI{}).Generate(provider.Request{
			Source: source,
			At:     seed.Coordinate{Entity: "handset", Row: row, Field: "imei", Purpose: "value"},
		})
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if verdict := (intl.Card{}).Validate(value); verdict.Valid {
			t.Fatalf("the equipment identity %s was approved as a card number", value)
		}
	}
	// The equipment identities people actually hold begin with a reporting
	// body such as 35 or 49 rather than the test block, so the same question is
	// asked of those shapes.
	for _, value := range []string{"490154203237518", "356938035643809", "011234567890126"} {
		if verdict := (intl.Card{}).Validate(value); verdict.Valid {
			t.Errorf("the equipment identity %s was approved as a card number", value)
		}
	}
}

// Other lengths are card numbers too, and they are passed over rather than
// called wrong, whatever their check digit says.
func TestOtherFormsArePassedOverRatherThanCalledWrong(t *testing.T) {
	for n := 1; n <= 19; n++ {
		value := zeros(n)
		verdict := (intl.Card{}).Validate(value)
		switch {
		case n == 16:
			if !verdict.Valid {
				t.Errorf("sixteen noughts satisfy the check digit and were refused: %s", verdict.Reason)
			}
		case !verdict.Unsupported:
			t.Errorf("%d noughts were answered valid=%v unsupported=%v (%s), where a form this build "+
				"does not check is passed over", n, verdict.Valid, verdict.Unsupported, verdict.Reason)
		case verdict.Reason == "":
			t.Errorf("%d noughts were passed over without a reason", n)
		}
	}
	// Fifteen digits are checked only behind the two American Express
	// beginnings, whatever the check digit says. This one fails the formula,
	// so it is nobody's card either.
	if verdict := (intl.Card{}).Validate("100000000000000"); !verdict.Unsupported {
		t.Errorf("a fifteen digit value outside American Express was answered valid=%v: %s",
			verdict.Valid, verdict.Reason)
	}
}

// The shapes no card number has are refused, with a reason.
func TestTheShapesACardNumberDoesNotHaveAreRefused(t *testing.T) {
	for _, value := range []string{
		"",
		"4242 4242 4242 4242",
		"424242424242424X",
		"4242-4242-4242-4242",
		zeros(20),
		"٤٢٤٢٤٢٤٢٤٢٤٢٤٢٤٢",
	} {
		verdict := (intl.Card{}).Validate(value)
		if verdict.Valid || verdict.Unsupported {
			t.Errorf("%q was answered valid=%v unsupported=%v, where it has to be refused",
				value, verdict.Valid, verdict.Unsupported)
		}
		if verdict.Reason == "" {
			t.Errorf("%q was refused without a reason", value)
		}
		if findings := oracle.CheckCard(value); len(findings) == 0 {
			t.Errorf("the independent validator accepted %q", value)
		}
	}
}

// Moving one digit has to break the check digit, which is the claim the catalog
// entry for this scheme rests on. Moving one of the first two digits of an
// American Express number can leave a form this build does not check instead,
// and that is passed over rather than approved.
func TestMovingASingleDigitBreaksTheCardCheckDigit(t *testing.T) {
	for _, number := range (intl.Card{}).Published() {
		for at := 0; at < len(number); at++ {
			for shift := 1; shift < 10; shift++ {
				moved := []byte(number)
				moved[at] = '0' + byte((int(moved[at]-'0')+shift)%10)
				verdict := (intl.Card{}).Validate(string(moved))
				if verdict.Valid {
					t.Fatalf("changing position %d of %s to %s left it valid", at, number, string(moved))
				}
				if verdict.Unsupported && !(len(number) == 15 && at < 2) {
					t.Fatalf("changing position %d of %s to %s was passed over rather than refused",
						at, number, string(moved))
				}
				// The agreement is about approval. The independent validator
				// says a form is outside the two it checks where the provider
				// passes it over, so both say not approved in their own way.
				if findings := oracle.CheckCard(string(moved)); verdict.Valid != (len(findings) == 0) {
					t.Fatalf("the two validators disagree about %s: ours valid=%v unsupported=%v, the other %v",
						string(moved), verdict.Valid, verdict.Unsupported, findings)
				}
			}
		}
	}
}

// The modulus is ten, so every body has a completion and this never refuses for
// arithmetic reasons in a form this build checks.
func TestRecomputingTheCardCheckDigitNeverRefuses(t *testing.T) {
	for _, number := range (intl.Card{}).Published() {
		// The first two digits of an American Express number are its form, so
		// the walk starts after them for that brand.
		from := 0
		if len(number) == 15 {
			from = 2
		}
		for at := from; at < len(number)-1; at++ {
			moved := []byte(number)
			moved[at] = '0' + (moved[at]-'0'+1)%10
			rebuilt, err := (intl.Card{}).WithCheckDigits(string(moved))
			if err != nil {
				t.Fatalf("%s position %d: %v", number, at, err)
			}
			if verdict := (intl.Card{}).Validate(rebuilt); !verdict.Valid {
				t.Fatalf("%s position %d rebuilt to %q, which is not valid: %s",
					number, at, rebuilt, verdict.Reason)
			}
		}
	}
	if _, err := (intl.Card{}).WithCheckDigits(zeros(14)); err == nil {
		t.Error("a form this build does not check was given a recomputed check digit")
	}
}

// The check digit is reported at the last position, and only for a form this
// build checks.
func TestTheCardCheckDigitIsReportedAtTheLastPosition(t *testing.T) {
	for _, number := range (intl.Card{}).Published() {
		positions, ok := (intl.Card{}).CheckDigits(number)
		if !ok || len(positions) != 1 || positions[0] != len(number)-1 {
			t.Errorf("the check digit of %s was reported at %v, ok=%v", number, positions, ok)
		}
	}
	for _, value := range []string{zeros(14), zeros(19), "424242424242424X"} {
		if _, ok := (intl.Card{}).CheckDigits(value); ok {
			t.Errorf("%q is not a form this build checks and was given a check digit position", value)
		}
	}
}

// The same address gives the same number, which is what a dataset rebuilt from
// a seed depends on.
func TestTheSameRequestProducesTheSameCard(t *testing.T) {
	for row := int64(0); row < 200; row++ {
		first := generateCard(t, seed.New("card-repeat"), row)
		second := generateCard(t, seed.New("card-repeat"), row)
		if first != second {
			t.Fatalf("row %d gave %q and then %q", row, first, second)
		}
	}
}

// The provider has to satisfy the contract every provider satisfies.
func TestTheCardHonoursItsContract(t *testing.T) {
	if findings := provider.Check(intl.Card{}); len(findings) != 0 {
		t.Errorf("the provider breaks its contract: %v", findings)
	}
	if id := (intl.Card{}).ID(); id != "card" {
		t.Errorf("the catalog published the name card before this file existed and the provider answers %q", id)
	}
}
