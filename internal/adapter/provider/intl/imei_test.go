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

// theEquipmentAnchor is the calculation annex B.3 of 3GPP TS 23.003 works, with
// the check digit the specification arrives at put back in last place.
//
// The annex prints the fourteen digits in two groups, an allocation code of
// 2 6 0 5 3 1 7 9 and a serial number of 3 1 1 3 8 3, and reaches 7.
const theEquipmentAnchor = "260531793113837"

// theEquipmentAnchorTotal is the number the annex prints at step 2, before it
// takes the total anywhere.
//
// Held separately from the check digit on purpose, for the reason the article
// number holds its own intermediate total: a worked example that gives only an
// answer lets a wrong sum and a wrong fold cancel each other out. This
// specification publishes the number in the middle precisely so they cannot.
const theEquipmentAnchorTotal = 53

// generateIMEI produces the identity for one row, failing the test rather than
// handing back an error nobody looks at.
func generateIMEI(t *testing.T, source seed.Source, row int64) string {
	t.Helper()
	value, err := (intl.IMEI{}).Generate(provider.Request{
		Source: source,
		At:     seed.Coordinate{Entity: "handset", Row: row, Field: "imei", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: the generator refused: %v", row, err)
	}
	return value
}

// The strongest anchor a source can give: it does the arithmetic itself, and
// this one prints the number it reaches halfway through.
func TestTheExampleTheSpecificationWorksComesOutTheSameHere(t *testing.T) {
	// The intermediate value first, because an answer that agrees while the
	// working disagrees is the case a single comparison would miss.
	total, ok := oracle.IMEIPublishedTotal(theEquipmentAnchor[:14])
	if !ok {
		t.Fatal("the published body was not read as fourteen digits")
	}
	if total != theEquipmentAnchorTotal {
		t.Errorf("the annex prints a total of %d at step 2 and the same steps give %d here",
			theEquipmentAnchorTotal, total)
	}

	if verdict := (intl.IMEI{}).Validate(theEquipmentAnchor); !verdict.Valid {
		t.Errorf("the identity %s is worked by its own specification and was refused: %s",
			theEquipmentAnchor, verdict.Reason)
	}
	rebuilt, err := (intl.IMEI{}).WithCheckDigits(theEquipmentAnchor)
	if err != nil {
		t.Fatalf("recomputing the check digit of %s failed: %v", theEquipmentAnchor, err)
	}
	if rebuilt != theEquipmentAnchor {
		t.Errorf("the published identity %s rebuilt with its own check digit became %s",
			theEquipmentAnchor, rebuilt)
	}
	if findings := oracle.CheckIMEI(theEquipmentAnchor); len(findings) != 0 {
		t.Errorf("the independent validator refused the published identity %s: %v",
			theEquipmentAnchor, findings)
	}
}

// Every value this build produces has to satisfy the validator written from the
// specification rather than from the generator.
func TestEveryGeneratedEquipmentIdentitySatisfiesTheIndependentValidator(t *testing.T) {
	source := seed.New("imei-sweep-2026-09")
	for row := int64(0); row < 5000; row++ {
		value := generateIMEI(t, source, row)
		if len(value) != 15 {
			t.Fatalf("row %d produced %q, which is %d characters", row, value, len(value))
		}
		if findings := oracle.CheckIMEI(value); len(findings) != 0 {
			t.Fatalf("row %d produced %q and the independent validator says %v", row, value, findings)
		}
	}
}

// The narrowing this scheme does even though it has no harbour to claim.
//
// It is checked on the value rather than on the constant, because the promise
// is about what comes out and not about what the provider meant to do.
func TestEveryGeneratedEquipmentIdentitySitsInTheTestBlock(t *testing.T) {
	source := seed.New("imei-block-2026-09")
	for row := int64(0); row < 2000; row++ {
		value := generateIMEI(t, source, row)
		if !strings.HasPrefix(value, "00") {
			t.Fatalf("row %d produced %q, which is outside the block GSMA gives to test equipment", row, value)
		}
	}
}

// The claim this provider deliberately does not make.
//
// Generating inside the test block is worth something and it is not a safe
// harbour, because codes are issued inside that block to real manufacturers. A
// later reader who takes the narrowing for a harbour would turn a smaller
// promise into a bigger one, and this is what stops that.
func TestTheIdentityClaimsNoSafeHarbour(t *testing.T) {
	for _, claimed := range (intl.IMEI{}).Capabilities() {
		if claimed == capability.IdentifierReservedRange {
			t.Error("codes are issued inside the test block, so this scheme has no range that is never assigned, and the provider declares one")
		}
		if claimed == capability.IdentifierUnassignedTarget {
			t.Error("this scheme has no block a value could be moved into and the provider declares one")
		}
	}
	if _, claims := interface{}(intl.IMEI{}).(provider.RangeAware); claims {
		t.Error("the provider implements the reserved range interface for a scheme with no harbour")
	}
}

// Generation is narrow and checking is wide. A well formed identity outside the
// test block is somebody's real value, and calling it wrong would report a
// defect against data that has none.
func TestValidateAcceptsCorrectIdentitiesOutsideTheTestBlock(t *testing.T) {
	for _, value := range []string{
		theEquipmentAnchor,
		"490154203237518",
		"356938035643809",
		"011234567890126",
	} {
		if verdict := (intl.IMEI{}).Validate(value); !verdict.Valid {
			t.Errorf("the correct identity %s was refused: %s", value, verdict.Reason)
		}
		if findings := oracle.CheckIMEI(value); len(findings) != 0 {
			t.Errorf("the independent validator refused the correct identity %s: %v", value, findings)
		}
	}
}

// Fourteen noughts total nought, the total already ends in nought, so the check
// digit is nought. The specification names that case out loud rather than
// leaving it to be inferred, and the test confirms it rather than stepping
// around it.
func TestFourteenNoughtsAreACorrectIdentityBecauseTheArithmeticSaysSo(t *testing.T) {
	const allNoughts = "000000000000000"
	if verdict := (intl.IMEI{}).Validate(allNoughts); !verdict.Valid {
		t.Errorf("the arithmetic makes %s correct and it was refused: %s", allNoughts, verdict.Reason)
	}
	if findings := oracle.CheckIMEI(allNoughts); len(findings) != 0 {
		t.Errorf("the independent validator refused %s: %v", allNoughts, findings)
	}
}

// The sixteen digit form has no check digit at all, so it is recognised and
// passed over rather than approved or faulted.
//
// The verdict matters beyond the wording. The corruption engine treats an
// unsupported value as a refusal, which is what stops a check digit failure
// being aimed at a value that has no check digit to fail.
func TestTheSixteenDigitFormIsRecognisedRatherThanCalledWrong(t *testing.T) {
	// The published body with a software version number in place of the check
	// digit, which is what clause 6.2.2 describes.
	const withSoftwareVersion = "2605317931138301"

	verdict := (intl.IMEI{}).Validate(withSoftwareVersion)
	if verdict.Valid {
		t.Error("the sixteen digit form was approved, and nothing in it was checked")
	}
	if !verdict.Unsupported {
		t.Errorf("the sixteen digit form was called wrong rather than passed over: %s", verdict.Reason)
	}
	if !strings.Contains(verdict.Reason, "software version") {
		t.Errorf("the reason does not name the form it passed over: %s", verdict.Reason)
	}
	// Sixteen characters that are not all digits are not that form, and have to
	// fall through to the ordinary refusal.
	if verdict := (intl.IMEI{}).Validate("26053179311383XY"); verdict.Unsupported {
		t.Error("a sixteen character value that is not numeric was taken for the software version form")
	}
}

// Values that are not equipment identities are refused rather than answered
// about.
func TestTheShapesAnEquipmentIdentityDoesNotHaveAreRefused(t *testing.T) {
	for _, value := range []string{
		"", "2", "26053179311383", "2605317931138370",
		"26053179311383X", "X6053179311383 7", "26053179311383-7",
		"26053179311383٧",
	} {
		if verdict := (intl.IMEI{}).Validate(value); verdict.Valid {
			t.Errorf("%q was accepted as an equipment identity", value)
		}
	}
}

// A mutator has to be able to move a digit and ask for a consistent value back.
// The modulus is ten, so every body has a completion and this never refuses for
// arithmetic reasons.
func TestRecomputingTheEquipmentCheckDigitNeverRefuses(t *testing.T) {
	source := seed.New("imei-rebuild")
	for row := int64(0); row < 500; row++ {
		value := generateIMEI(t, source, row)
		for at := 0; at < 14; at++ {
			moved := []byte(value)
			moved[at] = '0' + (moved[at]-'0'+1)%10
			rebuilt, err := (intl.IMEI{}).WithCheckDigits(string(moved))
			if err != nil {
				t.Fatalf("row %d position %d: %v", row, at, err)
			}
			if verdict := (intl.IMEI{}).Validate(rebuilt); !verdict.Valid {
				t.Fatalf("row %d position %d rebuilt to %q, which is not valid: %s",
					row, at, rebuilt, verdict.Reason)
			}
		}
	}
}

// Moving one digit has to break the check digit, which is the claim the catalog
// entry for this scheme rests on.
func TestMovingASingleDigitBreaksTheEquipmentCheckDigit(t *testing.T) {
	source := seed.New("imei-single-digit")
	for row := int64(0); row < 200; row++ {
		value := generateIMEI(t, source, row)
		for at := 0; at < 15; at++ {
			for shift := 1; shift < 10; shift++ {
				moved := []byte(value)
				moved[at] = '0' + byte((int(moved[at]-'0')+shift)%10)
				if verdict := (intl.IMEI{}).Validate(string(moved)); verdict.Valid {
					t.Fatalf("changing position %d of %q to %q left it valid",
						at, value, string(moved))
				}
			}
		}
	}
}

// 🔴 The limitation whoever writes the transposed digits mutator has to know
// about, measured rather than argued.
//
// A Luhn sum is blind to an adjacent transposition exactly when the two digits
// are nought and nine, and to no other pair. The reason is arithmetic: doubling
// and folding maps a digit to itself plus a fixed offset, and only nought and
// nine share an offset. The article number is blind to five pairs and this one
// is blind to one, so the two schemes cannot be given the same expectation.
//
// This test walks every pair rather than sampling, so it is a statement about
// the scheme and not about a run.
func TestTheOnlyAdjacentTranspositionThisSchemeMissesIsNoughtAgainstNine(t *testing.T) {
	blind := map[string]bool{}
	source := seed.New("imei-transpositions")
	for row := int64(0); row < 300; row++ {
		value := generateIMEI(t, source, row)
		for at := 0; at < 13; at++ {
			if value[at] == value[at+1] {
				continue
			}
			swapped := []byte(value)
			swapped[at], swapped[at+1] = swapped[at+1], swapped[at]
			if verdict := (intl.IMEI{}).Validate(string(swapped)); verdict.Valid {
				pair := []byte{value[at], value[at+1]}
				if pair[0] > pair[1] {
					pair[0], pair[1] = pair[1], pair[0]
				}
				blind[string(pair)] = true
			}
		}
	}
	if len(blind) != 1 || !blind["09"] {
		t.Errorf("the pairs this scheme cannot tell apart when transposed are %v, and the arithmetic says it should be nought against nine alone", blind)
	}
}

// The same address gives the same identity, which is what a dataset rebuilt
// from a seed depends on.
func TestTheSameRequestProducesTheSameEquipmentIdentity(t *testing.T) {
	for row := int64(0); row < 200; row++ {
		first := generateIMEI(t, seed.New("imei-repeat"), row)
		second := generateIMEI(t, seed.New("imei-repeat"), row)
		if first != second {
			t.Fatalf("row %d gave %q and then %q", row, first, second)
		}
	}
}

// The check digit is reported where the specification puts it, and only for a
// value of the right length.
func TestTheEquipmentCheckDigitIsReportedAtTheLastPosition(t *testing.T) {
	value := generateIMEI(t, seed.New("imei-positions"), 3)
	positions, ok := (intl.IMEI{}).CheckDigits(value)
	if !ok {
		t.Fatal("a generated identity has no check digit position")
	}
	if len(positions) != 1 || positions[0] != 14 {
		t.Errorf("the check digit was reported at %v", positions)
	}
	if _, ok := (intl.IMEI{}).CheckDigits("00"); ok {
		t.Error("a value that is not an equipment identity was given a check digit position")
	}
}

// The provider has to satisfy the contract every provider satisfies, including
// that each capability it declares is backed by the interface that capability
// requires.
func TestTheProviderHonoursItsContract(t *testing.T) {
	if findings := provider.Check(intl.IMEI{}); len(findings) != 0 {
		t.Errorf("the provider breaks its contract: %v", findings)
	}
	if id := (intl.IMEI{}).ID(); id != "imei" {
		t.Errorf("the catalog published the name imei before this file existed and the provider answers %q", id)
	}
}
