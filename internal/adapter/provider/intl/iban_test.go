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

// generated produces one account number at a stated address.
//
// Every correct value in this file comes from here rather than from a literal.
// This scheme has no block that is never issued, so a correct value cannot be
// shown not to be somebody's, and the answer to that is to publish none.
func generated(t *testing.T, source string, row int64) string {
	t.Helper()
	value, err := intl.IBAN{}.Generate(provider.Request{
		Source: seed.New(source),
		At:     seed.Coordinate{Entity: "account", Row: row, Field: "iban", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: the generator refused to produce a value: %v", row, err)
	}
	return value
}

// The first question asked of every provider here. Two implementations written
// from one reading agree by construction, so this is worth less than it looks,
// which is why the register of anchors exists beside it.
//
// It is worth more than usual in this one case: the two are not one description
// written twice. The act sets out two methods and calls them equivalent, and the
// provider takes one while the independent validator takes the other.
func TestEveryGeneratedAccountNumberSatisfiesTheIndependentValidator(t *testing.T) {
	for row := int64(0); row < 2000; row++ {
		value := generated(t, "iban conformance", row)
		if findings := oracle.CheckIBAN(value); len(findings) != 0 {
			t.Fatalf("row %d produced %s, which the independent validator refuses: %v",
				row, value, findings)
		}
		if verdict := (intl.IBAN{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %s, which this provider refuses: %s",
				row, value, verdict.Reason)
		}
	}
}

// Pillar one produces a value whose correctness can be checked in full, and for
// this scheme that means a Polish one.
//
// Producing a value from a country whose length we never read would be handing
// somebody data this tool cannot say anything about, which is the defect it
// sells rather than the product.
func TestEveryGeneratedAccountNumberIsOneThisBuildChecksInFull(t *testing.T) {
	for row := int64(0); row < 500; row++ {
		value := generated(t, "iban in full", row)
		if !oracle.IBANCheckedInFull(value) {
			t.Fatalf("row %d produced %s, which the independent validator only half checks", row, value)
		}
		if len(value) != 28 || !strings.HasPrefix(value, "PL") {
			t.Fatalf("row %d produced %s, which is not the shape the act in force describes", row, value)
		}
	}
}

// The countries the provider names are the countries of the values it produces,
// both ways round: a country named and never produced would shelve the kind
// under a promise, and one produced and not named would leave a surface saying
// less than the values do (DEC-281).
func TestTheCountriesTheAccountNumberNamesAreTheOnesItProduces(t *testing.T) {
	named := intl.IBAN{}.Countries()
	if len(named) == 0 {
		t.Fatal("the provider names no country, which the contract forbids")
	}
	produced := map[string]bool{}
	for row := int64(0); row < 500; row++ {
		produced[generated(t, "iban countries", row)[:2]] = true
	}
	for _, country := range named {
		if !produced[country] {
			t.Errorf("the provider names %s and 500 values carried none from it", country)
		}
		delete(produced, country)
	}
	for country := range produced {
		t.Errorf("a value from %s was produced and the provider does not name that country", country)
	}
}

// The settlement number names a bank, and the act in force gives it a check
// digit of its own. A generated value failing that rule would be one the central
// bank could not have assigned, and this provider would then be refusing its own
// output one file away.
func TestEveryGeneratedSettlementNumberClosesUnderTheActInForce(t *testing.T) {
	weights := []int{3, 9, 7, 1, 3, 9, 7, 1}
	for row := int64(0); row < 500; row++ {
		value := generated(t, "iban settlement", row)
		total := 0
		for position, weight := range weights {
			total += int(value[4+position]-'0') * weight
		}
		if total%10 != 0 {
			t.Fatalf("row %d produced %s, whose settlement number leaves %d", row, value, total%10)
		}
	}
}

// 🔴 The anchor of this provider, and the reason its arithmetic is not a
// measurement the way the routing number's is.
//
// The act does not merely describe the rule. It prints thirty weights it worked
// out, and they are the powers of ten modulo ninety seven. Reproducing all
// thirty forces the modulus rather than agreeing with it: this walks every
// modulus up to a thousand and finds one.
func TestTheWeightTableThePolishActPublishesIsReproduced(t *testing.T) {
	published := oracle.IBANPublishedWeights()
	if len(published) != 30 {
		t.Fatalf("the published table came apart into %d weights, not thirty", len(published))
	}

	for position, weight := range published {
		power := 1
		for turn := 0; turn < position; turn++ {
			power = power * 10 % 97
		}
		if power != weight {
			t.Errorf("position %d carries %d and ten to that power modulo ninety seven is %d",
				position+1, weight, power)
		}
	}

	var solutions []int
	for modulus := 2; modulus < 1000; modulus++ {
		if reproducesTable(published, modulus) {
			solutions = append(solutions, modulus)
		}
	}
	if len(solutions) != 1 || solutions[0] != 97 {
		t.Fatalf("the table is reproduced by the moduli %v, so it does not force ninety seven",
			solutions)
	}
}

func reproducesTable(published []int, modulus int) bool {
	power := 1 % modulus
	for _, weight := range published {
		if power != weight {
			return false
		}
		power = power * 10 % modulus
	}
	return true
}

// The source still has to say the three things the arithmetic rests on. A
// constant beside a sentence that stopped supporting it is the drift this whole
// package is built to catch.
func TestTheSourceStillFixesTheRotationAndTheAcceptedRemainder(t *testing.T) {
	if !oracle.IBANSourceSaysFourCharactersMove() {
		t.Error("the quoted sentence no longer says four characters move to the end")
	}
	if !oracle.IBANSourceAcceptsRemainderOne() {
		t.Error("the quoted sentence no longer makes one the accepting remainder")
	}
	if !oracle.NRBLengthFromSource() {
		t.Error("the act in force no longer says the national number is twenty six digits")
	}
}

// 🔴 The act names its own numbers wrongly and this records it, because the next
// session will read that sentence too.
//
// It calls twenty five and twenty one the decimal ASCII values of P and L. ASCII
// puts those letters at eighty and seventy six. The numbers are the letter values
// of the international scheme, where A is ten, and taking the justification
// literally for any other country would break every value.
func TestTheActsOwnNameForItsLetterValuesIsWrongAndTheNumbersAreRight(t *testing.T) {
	forP, forL, ok := oracle.IBANLetterValuesFromSource()
	if !ok {
		t.Fatal("the quoted sentence no longer carries both letter values")
	}
	if forP == int('P') || forL == int('L') {
		t.Fatal("the sentence now carries the actual ASCII values, so the act was corrected " +
			"and this test is the place to find out")
	}
	if want := int('P'-'A') + 10; forP != want {
		t.Errorf("the source gives P the value %d and the letter offset of ten gives %d", forP, want)
	}
	if want := int('L'-'A') + 10; forL != want {
		t.Errorf("the source gives L the value %d and the letter offset of ten gives %d", forL, want)
	}
}

// 🔴 Check digits outside 02 to 98 are refused, and the reason is a derivation
// from the quoted rule rather than an extra assumption.
//
// The check number is ninety eight minus a remainder modulo ninety seven, so it
// runs from two to ninety eight. The congruence does not catch either end on its
// own, because a number and that number plus ninety seven leave the same
// remainder: 00 pairs with 97, 01 with 98 and 99 with 02.
//
// 🔴 The high end was missing until a fuzzer found it in fifteen seconds. Both
// ends are tested here for that reason, and the pairs above are named so the
// next reader can see why one guard is not two thirds of the work.
func TestCheckDigitsNoCalculationCanReachAreRefused(t *testing.T) {
	value := generated(t, "iban impossible", 3)
	for _, impossible := range []string{"00", "01", "99"} {
		broken := value[:2] + impossible + value[4:]
		verdict := intl.IBAN{}.Validate(broken)
		if verdict.Valid || verdict.Unsupported {
			t.Errorf("check digits %s were accepted and no calculation produces them", impossible)
		}
		if !strings.Contains(verdict.Reason, "02 to 98") {
			t.Errorf("check digits %s were refused for the wrong reason: %s", impossible, verdict.Reason)
		}
		if findings := oracle.CheckIBAN(broken); len(findings) == 0 {
			t.Errorf("the independent validator accepted check digits %s", impossible)
		}
	}
}

// foreign builds an account number for a country code ISO 3166 sets aside for
// private use, so it cannot be anybody's account, and gives it correct check
// digits with our own arithmetic.
func foreign(t *testing.T, national string) string {
	t.Helper()
	rebuilt, err := intl.IBAN{}.WithCheckDigits("ZZ00" + national)
	if err != nil {
		t.Fatalf("cannot build a foreign value: %v", err)
	}
	return rebuilt
}

// 🔴 The whole of the decision this provider was built around.
//
// A value from another country whose check digits agree is not approved, because
// the length of an account number there was never read at a source this project
// can use. It is answered as unchecked, and the answer names the country so that
// a report can say which one.
func TestAnAccountNumberFromAnotherCountryIsUnsupportedRatherThanApproved(t *testing.T) {
	value := foreign(t, "12345678901234")

	verdict := intl.IBAN{}.Validate(value)
	if verdict.Valid {
		t.Fatal("a value whose length this build never checked was approved")
	}
	if !verdict.Unsupported {
		t.Fatalf("a value with agreeing check digits was faulted: %s", verdict.Reason)
	}
	if !strings.Contains(verdict.Reason, "ZZ") {
		t.Errorf("the answer does not name the country it is about: %s", verdict.Reason)
	}
	if oracle.IBANCheckedInFull(value) {
		t.Error("the independent validator claims it checked a foreign value in full")
	}
}

// The other half of that decision, and the half that keeps it from being a way
// of never saying anything. Check digits that disagree are a defect we can name
// whatever the country is, so they are a fault rather than an omission.
func TestABrokenAccountNumberFromAnotherCountryIsStillInvalid(t *testing.T) {
	value := foreign(t, "12345678901234")
	broken := value[:4] + string(rotateDigit(value[4])) + value[5:]

	verdict := intl.IBAN{}.Validate(broken)
	if verdict.Unsupported {
		t.Fatalf("a value with wrong check digits was excused as unchecked: %s", verdict.Reason)
	}
	if verdict.Valid {
		t.Fatal("a value with wrong check digits was approved")
	}
	if findings := oracle.CheckIBAN(broken); len(findings) == 0 {
		t.Error("the independent validator accepted a value whose check digits disagree")
	}
}

func rotateDigit(digit byte) byte {
	if digit == '9' {
		return '0'
	}
	return digit + 1
}

// A Polish value whose settlement number could not have been assigned is told
// that, rather than being told its check digits disagree, which they no longer
// do because they were recomputed.
//
// The two are different defects with different fixes, and reporting the second
// for the first is the same silence this project keeps finding elsewhere.
func TestASettlementNumberTheBankCouldNotHaveAssignedIsNamedSeparately(t *testing.T) {
	value := generated(t, "iban settlement fault", 11)
	moved := value[:4] + string(rotateDigit(value[4])) + value[5:]

	repaired, err := intl.IBAN{}.WithCheckDigits(moved)
	if err != nil {
		t.Fatalf("cannot recompute the check digits: %v", err)
	}

	verdict := intl.IBAN{}.Validate(repaired)
	if verdict.Valid {
		t.Fatal("a settlement number that does not close was approved")
	}
	if !strings.Contains(verdict.Reason, "settlement number") {
		t.Errorf("the defect was reported as something else: %s", verdict.Reason)
	}
	if findings := oracle.CheckIBAN(repaired); len(findings) == 0 {
		t.Error("the independent validator accepted a settlement number that does not close")
	}
}

// A Polish value of the wrong length is told about its length. The check digits
// would also disagree, and reporting those instead would name the symptom.
func TestAPolishAccountNumberOfTheWrongLengthIsToldAboutItsLength(t *testing.T) {
	value := generated(t, "iban length", 5)

	verdict := intl.IBAN{}.Validate(value[:len(value)-1])
	if verdict.Valid || verdict.Unsupported {
		t.Fatal("a Polish value one character short was not faulted")
	}
	if !strings.Contains(verdict.Reason, "twenty eight") {
		t.Errorf("the length defect was reported as something else: %s", verdict.Reason)
	}
}

// Any spacing is a way of writing the same number, because the act says the
// spaces are removed before the arithmetic and says nothing about where they
// were. What comes back has to be the text that arrived, not a regrouped one.
func TestTheSpacingOfAnAccountNumberSurvivesBeingRebuilt(t *testing.T) {
	value := generated(t, "iban spacing", 9)
	spellings := []string{value, inGroupsOf(value, 4), inGroupsOf(value, 7), value[:1] + " " + value[1:]}

	for _, spelling := range spellings {
		verdict := intl.IBAN{}.Validate(spelling)
		if !verdict.Valid {
			t.Errorf("%q is the same number and was refused: %s", spelling, verdict.Reason)
			continue
		}
		rebuilt, err := intl.IBAN{}.WithCheckDigits(spelling)
		if err != nil {
			t.Errorf("%q could not be rebuilt: %v", spelling, err)
			continue
		}
		if rebuilt != spelling {
			t.Errorf("%q came back as %q, which is a second change nobody asked for",
				spelling, rebuilt)
		}
	}
}

func inGroupsOf(value string, size int) string {
	var built strings.Builder
	for index := 0; index < len(value); index++ {
		if index > 0 && index%size == 0 {
			built.WriteByte(' ')
		}
		built.WriteByte(value[index])
	}
	return built.String()
}

// The offsets answer about the value as it was written. A value carrying spaces
// holds its check digits further along, and returning two and three for it would
// point a corruption at the wrong bytes.
func TestTheCheckDigitPositionsFollowTheSpacing(t *testing.T) {
	value := generated(t, "iban positions", 2)

	bare, ok := intl.IBAN{}.CheckDigits(value)
	if !ok || len(bare) != 2 || bare[0] != 2 || bare[1] != 3 {
		t.Fatalf("the check digits of a bare value are at %v", bare)
	}

	spaced := value[:1] + " " + value[1:]
	moved, ok := intl.IBAN{}.CheckDigits(spaced)
	if !ok || len(moved) != 2 {
		t.Fatalf("a spaced value answered %v", moved)
	}
	if spaced[moved[0]] != value[2] || spaced[moved[1]] != value[3] {
		t.Errorf("the offsets %v of %q do not land on its check digits", moved, spaced)
	}
}

// Recomputing never refuses for arithmetic reasons, because the rule is a
// subtraction rather than a lookup. That is the difference from the Polish tax
// number, where about one body in eleven has no completing digit.
func TestRecomputingCheckDigitsNeverRefuses(t *testing.T) {
	for row := int64(0); row < 300; row++ {
		value := generated(t, "iban recompute", row)
		rebuilt, err := intl.IBAN{}.WithCheckDigits(value)
		if err != nil {
			t.Fatalf("%s could not be rebuilt with its own check digits: %v", value, err)
		}
		if rebuilt != value {
			t.Fatalf("%s rebuilt with its own check digits became %s", value, rebuilt)
		}
	}
}

// The contract says a declared capability has to be backed by the interface it
// implies, in both directions. This is that check for this provider.
func TestTheProviderMeetsItsContract(t *testing.T) {
	if findings := provider.Check(intl.IBAN{}); len(findings) != 0 {
		t.Fatalf("the provider does not meet the contract it declares: %v", findings)
	}
	if id := (intl.IBAN{}).ID(); id != "iban" {
		t.Fatalf("the identifier is %q and an international standard carries no country prefix", id)
	}
}

// A scheme with no block that is never issued must not claim one. The absence of
// the declaration is the machine readable way of saying a dataset carrying these
// has to be marked synthetic, and a manifest reader would believe the claim.
//
// It asks about the declared list as well as the interfaces, because the
// declaration is the half a report reads. Asking only about the interfaces would
// leave the claim breakable in the place where it is actually made.
func TestTheAccountNumberClaimsNoSafeHarbour(t *testing.T) {
	for _, claimed := range (intl.IBAN{}).Capabilities() {
		if claimed == capability.IdentifierReservedRange {
			t.Error("this scheme has no range that is never issued and the provider declares one")
		}
		if claimed == capability.IdentifierUnassignedTarget {
			t.Error("this scheme has no block a value could be moved into and the provider declares one")
		}
	}
	if _, claims := interface{}(intl.IBAN{}).(provider.RangeAware); claims {
		t.Error("the provider implements the reserved range interface for a scheme with no harbour")
	}
	if _, claims := interface{}(intl.IBAN{}).(provider.UnassignedWritable); claims {
		t.Error("the provider implements the unassigned target interface for a scheme with no spare block")
	}
}

// Values that are not account numbers are refused rather than answered about.
func TestValuesThatAreNotAccountNumbersAreRefused(t *testing.T) {
	for _, value := range []string{
		"", "P", "PL", "PL6", "PL61", "pl61109010140000071219812874",
		"1161109010140000071219812874", "PLXX109010140000071219812874",
		"PL61-1090-1014", "PL61\t1090", "PL61109010140000071219812.74",
	} {
		verdict := intl.IBAN{}.Validate(value)
		if verdict.Valid || verdict.Unsupported {
			t.Errorf("%q was not refused", value)
		}
		if verdict.Reason == "" {
			t.Errorf("%q was refused without a reason", value)
		}
	}
}
