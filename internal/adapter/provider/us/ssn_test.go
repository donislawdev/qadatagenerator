package us_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

func request(row int64) provider.Request {
	return provider.Request{
		Source: seed.New("release-2026-08"),
		At:     seed.Coordinate{Entity: "customer", Row: row, Field: "national_id", Purpose: "value"},
	}
}

func generate(t *testing.T, row int64) string {
	t.Helper()
	value, err := us.SSN{}.Generate(request(row))
	if err != nil {
		t.Fatalf("row %d: %v", row, err)
	}
	return value
}

// The provider has to satisfy the contract it declares, or a mutator will be
// told a capability is available and find out otherwise while corrupting.
func TestTheProviderMeetsItsContract(t *testing.T) {
	if findings := provider.Check(us.SSN{}); len(findings) != 0 {
		t.Fatalf("the contract is not met: %v", findings)
	}
	if id := (us.SSN{}).ID(); id != "US.ssn" {
		t.Errorf("the identifier is %q", id)
	}
}

// The absences are as much a part of this provider as the one capability it
// brings. An SSN carries no check digit, no date and no gender, so claiming
// any of them would hand a mutator something to work with that is not there.
func TestTheProviderClaimsNothingTheSchemeDoesNotHave(t *testing.T) {
	subject := us.SSN{}

	if got := len(subject.Capabilities()); got != 1 {
		t.Fatalf("expected one capability, got %v", subject.Capabilities())
	}
	if _, ok := interface{}(subject).(provider.CheckDigited); ok {
		t.Error("the provider offers check digits, and an SSN has none")
	}
	if _, ok := interface{}(subject).(provider.DateEncoder); ok {
		t.Error("the provider offers an encoded date, and an SSN encodes none")
	}
	if _, ok := interface{}(subject).(provider.GenderEncoder); ok {
		t.Error("the provider offers an encoded gender, and an SSN encodes none")
	}
}

// Every generated value is checked by a validator written from the published
// wording rather than from this generator. Two implementations by one hand in
// one hour agree with each other when the hand is wrong.
func TestEveryGeneratedValueSatisfiesTheIndependentValidator(t *testing.T) {
	for row := int64(0); row < 500; row++ {
		value := generate(t, row)
		if findings := oracle.CheckSSN(value); len(findings) != 0 {
			t.Fatalf("row %d produced %s, which the independent validator refuses: %v", row, value, findings)
		}
		if verdict := (us.SSN{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %s, which our own validator refuses: %s", row, value, verdict.Reason)
		}
	}
}

// The ethical default, checked rather than promised. A number outside area 666
// could be somebody's, and the point of the choice is that it takes a code
// change to leave the area rather than a forgotten flag.
func TestEveryGeneratedValueSitsInTheAreaThatIsNeverAssigned(t *testing.T) {
	for row := int64(0); row < 20000; row++ {
		value := generate(t, row)
		if !strings.HasPrefix(value, "666-") {
			t.Fatalf("row %d produced %s, which is outside the never assigned area", row, value)
		}
		if !(us.SSN{}).InReservedRange(value) {
			t.Fatalf("row %d produced %s, which the provider does not call reserved", row, value)
		}
		if !oracle.SSNNeverAssigned(value) {
			t.Fatalf("row %d produced %s, which the independent validator says could be issued", row, value)
		}
	}
}

// The two parts that start at one rather than nought. An off by one in either
// draw would produce a group of 00 or a serial of 0000, which the scheme calls
// invalid outright, and twenty thousand rows is far more than enough for a one
// in a hundred mistake to show up.
func TestTheGroupAndTheSerialNeverFallToNought(t *testing.T) {
	for row := int64(0); row < 20000; row++ {
		value := generate(t, row)
		if value[4:6] == "00" {
			t.Fatalf("row %d produced %s, whose group number the scheme calls invalid", row, value)
		}
		if value[7:11] == "0000" {
			t.Fatalf("row %d produced %s, whose serial number the scheme calls invalid", row, value)
		}
	}
}

// The same address has to give the same value, whatever else happened first.
func TestTheSameAddressAlwaysGivesTheSameValue(t *testing.T) {
	first := generate(t, 42)
	// Ask about other places in between, which is what would move a generator
	// holding a counter instead of deriving from the coordinate.
	for row := int64(0); row < 20; row++ {
		generate(t, row)
	}
	if second := generate(t, 42); first != second {
		t.Fatalf("the same place gave %s and then %s", first, second)
	}
}

// # Why this walks the whole space instead of sampling it
//
// The domain this generator can reach is ninety nine groups by nine thousand
// nine hundred and ninety nine serials inside one area, which is small enough
// to walk. Where a domain is countable, exhaustive beats random: a property
// based generator would draw a few thousand of these and call it evidence,
// and this leaves nothing to draw.
func TestEveryNumberTheGeneratorCanReachIsWellFormedAndNeverAssigned(t *testing.T) {
	subject := us.SSN{}
	checked := 0

	for group := 1; group <= 99; group++ {
		for serial := 1; serial <= 9999; serial++ {
			value := fmt.Sprintf("666-%02d-%04d", group, serial)
			if verdict := subject.Validate(value); !verdict.Valid {
				t.Fatalf("%s is refused by our own validator: %s", value, verdict.Reason)
			}
			if findings := oracle.CheckSSN(value); len(findings) != 0 {
				t.Fatalf("%s is refused by the independent validator: %v", value, findings)
			}
			if !subject.InReservedRange(value) {
				t.Fatalf("%s is not called reserved, so it could be somebody's", value)
			}
			checked++
		}
	}

	// A count that quietly fell to nothing would leave this passing while
	// walking no numbers at all, which is how a loop based test retires.
	if checked != 99*9999 {
		t.Fatalf("walked %d numbers, and the area holds %d", checked, 99*9999)
	}
	t.Logf("walked all %d numbers this generator can reach", checked)
}

func TestValidateRefusesWhatTheSchemeForbids(t *testing.T) {
	cases := []struct {
		name  string
		value string
		says  string
	}{
		{"too short", "66612345", "nine digits"},
		{"too long", "6661234567", "nine digits"},
		{"not digits", "666-12-345x", "nine digits"},
		{"separators in the wrong place", "6661-2-3456", "nine digits"},
		{"one separator only", "666-123456", "nine digits"},
		{"empty", "", "nine digits"},
		{"zero area", "000-12-3456", "area number"},
		{"zero group", "666-00-3456", "group number"},
		{"zero serial", "666-12-0000", "serial number"},
		{"zero area without separators", "000123456", "area number"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			verdict := us.SSN{}.Validate(test.value)
			if verdict.Valid {
				t.Fatalf("%q was accepted", test.value)
			}
			if !strings.Contains(verdict.Reason, test.says) {
				t.Errorf("the reason was %q, expected it to mention %q", verdict.Reason, test.says)
			}
			// The independent validator has to refuse it too, or the two
			// disagree about what the scheme says.
			if findings := oracle.CheckSSN(test.value); len(findings) == 0 {
				t.Errorf("the independent validator accepted %q", test.value)
			}
		})
	}
}

// Area 666 and the taxpayer identification space look like ordinary numbers
// and are handed out to nobody. A validator that refused them would refuse
// this tool's own output, so the two questions stay apart.
func TestTheAreasThatLookOrdinaryPassValidationAndAreStillNeverAssigned(t *testing.T) {
	for _, value := range []string{"666-12-3456", "900-12-3456", "999-99-9999", "912345678"} {
		if verdict := (us.SSN{}).Validate(value); !verdict.Valid {
			t.Errorf("%s was refused by shape: %s", value, verdict.Reason)
		}
		if !(us.SSN{}).InReservedRange(value) {
			t.Errorf("%s is not reported as never assigned", value)
		}
		if !oracle.SSNNeverAssigned(value) {
			t.Errorf("the independent validator says %s could be issued", value)
		}
	}

	// And one that is neither. An ordinary looking number outside every listed
	// block has to come back as issuable, or the check above proves nothing.
	if (us.SSN{}).InReservedRange("123-45-6789") {
		t.Error("an area outside every listed block is reported as never assigned")
	}
}

// # The regression this test exists for
//
// Pillar one generates inside a never issued block, so InReservedRange is true
// of every correct value. A corruption keying its applicability off that
// question would therefore never apply to anything, and the whole of layer L3
// for this identifier would be permanently unapplied while every test stayed
// green.
//
// The split is what makes it work: the base value sits in a block that looks
// ordinary, the corruption moves it into one that looks broken.
func TestTheCorruptionTargetsADifferentBlockFromTheGeneratedValue(t *testing.T) {
	subject := us.SSN{}
	base := generate(t, 8)

	// The trap, stated as an assertion so that nobody reintroduces it.
	if !subject.InReservedRange(base) {
		t.Fatalf("%s is not in a reserved range, so this test no longer guards anything", base)
	}

	for _, target := range subject.NeverIssuedTargets() {
		t.Run(target, func(t *testing.T) {
			moved, err := subject.WithNeverIssued(base, target)
			if err != nil {
				t.Fatalf("moving %s into %s failed: %v", base, target, err)
			}
			if moved == base {
				t.Fatalf("moving %s into %s changed nothing", base, target)
			}
			if verdict := subject.Validate(moved); verdict.Valid {
				t.Errorf("%s is still well formed, so an ordinary validator would accept it", moved)
			}
			if findings := oracle.CheckSSN(moved); len(findings) == 0 {
				t.Errorf("the independent validator accepts %s", moved)
			}
		})
	}
}

// The entry in the published catalog carries this pair as its example, and the
// examples are meant to become tests. Until the catalog is wired to the engine
// this stands in for that, so the specification cannot drift away from the
// behaviour without something going red.
func TestTheCatalogExampleIsReproduced(t *testing.T) {
	const (
		before = "666-12-3456"
		after  = "000-12-3456"
	)
	got, err := us.SSN{}.WithNeverIssued(before, us.TargetDefault)
	if err != nil {
		t.Fatalf("the catalog example failed: %v", err)
	}
	if got != after {
		t.Fatalf("the catalog says %s becomes %s, and it became %s", before, after, got)
	}
}

// Moving a value into the block it already occupies is the one case the
// manifest has to record as not applied with a reason. Answering it quietly
// with the value unchanged would be the silence this tool argues against.
func TestMovingAValueIntoTheBlockItAlreadyOccupiesIsRefused(t *testing.T) {
	cases := map[string]string{
		"000-12-3456": us.TargetZeroArea,
		"666-00-3456": us.TargetZeroGroup,
		"666-12-0000": us.TargetZeroSerial,
	}

	for value, target := range cases {
		_, err := us.SSN{}.WithNeverIssued(value, target)
		if err == nil {
			t.Errorf("%s was moved into %s, where it already sits", value, target)
			continue
		}
		if !strings.Contains(err.Error(), "change nothing") {
			t.Errorf("the refusal of %s says %q, which does not say what happened", value, err)
		}
	}
}

// A name the scheme does not have is refused by name, with the choices, rather
// than falling back to some default. A corruption that quietly does something
// other than what it was asked is worse than one that does not run.
func TestAnUnknownTargetIsRefusedWithTheChoices(t *testing.T) {
	_, err := us.SSN{}.WithNeverIssued("666-12-3456", "zero_everything")
	if err == nil {
		t.Fatal("an unknown block was accepted")
	}
	for _, target := range (us.SSN{}).NeverIssuedTargets() {
		if !strings.Contains(err.Error(), target) {
			t.Errorf("the refusal does not name the choice %q: %v", target, err)
		}
	}
}

// The corruption hands back the shape it was given. Reformatting on the way
// through would show up in a diff as though it had done more than it did.
func TestBothWaysOfWritingTheNumberSurviveTheCorruption(t *testing.T) {
	cases := map[string]string{
		"666-12-3456": "000-12-3456",
		"666123456":   "000123456",
	}

	for before, after := range cases {
		if verdict := (us.SSN{}).Validate(before); !verdict.Valid {
			t.Errorf("%s was refused: %s", before, verdict.Reason)
		}
		got, err := us.SSN{}.WithNeverIssued(before, us.TargetZeroArea)
		if err != nil {
			t.Errorf("%s: %v", before, err)
			continue
		}
		if got != after {
			t.Errorf("%s became %s, expected %s", before, got, after)
		}
	}
}

// The report has to be able to say why an application rejected data this tool
// calls correct, and the reason is a consequence of the ethical default rather
// than a defect. A limitation the interface does not say out loud is one the
// user meets as a bug.
func TestTheNeverAssignedBlocksAreDescribedForTheReport(t *testing.T) {
	described := strings.Join((us.SSN{}).ReservedRanges(), " | ")

	for _, mentioned := range []string{"000", "666", "900", "00", "0000"} {
		if !strings.Contains(described, mentioned) {
			t.Errorf("the description does not mention %s: %s", mentioned, described)
		}
	}
	if !strings.Contains(described, "ordinary") {
		t.Errorf("the description does not say that the generated block looks ordinary: %s", described)
	}
}
