package pl_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

func request(row int64, agreed map[string]string) provider.Request {
	return provider.Request{
		Source: seed.New("release-2026-08"),
		At:     seed.Coordinate{Entity: "customer", Row: row, Field: "pesel", Purpose: "value"},
		Agrees: provider.NewAgreement(agreed),
	}
}

func generate(t *testing.T, row int64, agreed map[string]string) string {
	t.Helper()
	value, err := pl.PESEL{}.Generate(request(row, agreed))
	if err != nil {
		t.Fatalf("row %d: %v", row, err)
	}
	return value
}

// The provider has to satisfy the contract it declares, or a mutator will be
// told a capability is available and find out otherwise while corrupting.
func TestTheProviderMeetsItsContract(t *testing.T) {
	if findings := provider.Check(pl.PESEL{}); len(findings) != 0 {
		t.Fatalf("the contract is not met: %v", findings)
	}
	if id := (pl.PESEL{}).ID(); id != "PL.pesel" {
		t.Errorf("the identifier is %q", id)
	}
}

// Every generated value is checked by a validator written from the published
// description rather than from this generator. Two implementations by one hand
// in one hour agree with each other when the hand is wrong, and this is the
// arrangement that refuses to let them.
func TestEveryGeneratedValueSatisfiesTheIndependentValidator(t *testing.T) {
	for row := int64(0); row < 500; row++ {
		value := generate(t, row, nil)
		if findings := oracle.CheckPESEL(value); len(findings) != 0 {
			t.Fatalf("row %d produced %s, which the independent validator refuses: %v", row, value, findings)
		}
		if verdict := (pl.PESEL{}).Validate(value); !verdict.Valid {
			t.Fatalf("row %d produced %s, which our own validator refuses: %s", row, value, verdict.Reason)
		}
	}
}

// The same address has to give the same value, whatever else happened first.
func TestTheSameAddressAlwaysGivesTheSameValue(t *testing.T) {
	first := generate(t, 42, nil)
	// Ask about other places in between, which is what would move a
	// generator holding a counter instead of deriving from the coordinate.
	for row := int64(0); row < 20; row++ {
		generate(t, row, nil)
	}
	if second := generate(t, 42, nil); first != second {
		t.Fatalf("the same place gave %s and then %s", first, second)
	}
}

// A recipe binding the date of birth and the gender gets a number that agrees
// with them. This is the whole reason the identifier is a bridge between the
// two pillars.
func TestABoundDateAndGenderSurviveIntoTheNumber(t *testing.T) {
	cases := []struct {
		date   string
		gender provider.Gender
		want   provider.Date
	}{
		{"1990-04-11", provider.Female, provider.Date{Year: 1990, Month: 4, Day: 11}},
		{"2004-12-31", provider.Male, provider.Date{Year: 2004, Month: 12, Day: 31}},
		{"1800-01-01", provider.Female, provider.Date{Year: 1800, Month: 1, Day: 1}},
		{"2299-12-31", provider.Male, provider.Date{Year: 2299, Month: 12, Day: 31}},
		// The leap day that exists and the one that does not, a century
		// apart. Nineteen hundred is not a leap year and two thousand is.
		{"2000-02-29", provider.Female, provider.Date{Year: 2000, Month: 2, Day: 29}},
	}

	for _, test := range cases {
		t.Run(test.date, func(t *testing.T) {
			value := generate(t, 1, map[string]string{
				provider.RoleBirthDate: test.date,
				provider.RoleGender:    string(test.gender),
			})

			got, ok := (pl.PESEL{}).EncodedDate(value)
			if !ok || got != test.want {
				t.Fatalf("%s encodes %+v, expected %+v", value, got, test.want)
			}
			if gender, ok := (pl.PESEL{}).EncodedGender(value); !ok || gender != test.gender {
				t.Errorf("%s encodes gender %q, expected %q", value, gender, test.gender)
			}
			if findings := oracle.CheckPESEL(value); len(findings) != 0 {
				t.Errorf("%s is refused by the independent validator: %v", value, findings)
			}
		})
	}
}

// # Why this walks every date instead of sampling them
//
// The five cases above are hand picked, which means they cover the boundaries
// somebody thought of. This covers all of them, both genders included, and
// the domain is small enough that there is no reason to sample: the scheme
// expresses five centuries, and walking the lot costs a fraction of a second.
//
// Where a domain is bounded, exhaustive beats random. A property based
// generator would draw a few thousand of these and call it evidence. This
// leaves nothing to draw.
func TestEveryDateTheSchemeCanExpressSurvivesTheRoundTrip(t *testing.T) {
	subject := pl.PESEL{}
	checked := 0

	for year := 1800; year <= 2299; year++ {
		for month := 1; month <= 12; month++ {
			for day := 1; day <= daysInMonth(year, month); day++ {
				want := provider.Date{Year: year, Month: month, Day: day}
				for _, gender := range []provider.Gender{provider.Female, provider.Male} {
					value, err := subject.Generate(request(1, map[string]string{
						provider.RoleBirthDate: iso(want),
						provider.RoleGender:    string(gender),
					}))
					if err != nil {
						t.Fatalf("%s %s: %v", iso(want), gender, err)
					}
					if got, ok := subject.EncodedDate(value); !ok || got != want {
						t.Fatalf("%s became %s, which reads back as %+v", iso(want), value, got)
					}
					if got, ok := subject.EncodedGender(value); !ok || got != gender {
						t.Fatalf("%s %s became %s, which reads back as %q", iso(want), gender, value, got)
					}
					if findings := oracle.CheckPESEL(value); len(findings) != 0 {
						t.Fatalf("%s %s became %s, refused by the independent validator: %v",
							iso(want), gender, value, findings)
					}
					checked++
				}
			}
		}
	}

	// A count that quietly fell to nothing would leave this passing while
	// checking no dates at all, which is the way a loop based test retires.
	if checked < 300000 {
		t.Fatalf("only %d combinations were walked, so the loop is not covering the scheme", checked)
	}
	t.Logf("walked %d date and gender combinations", checked)
}

func daysInMonth(year, month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%400 == 0 || (year%4 == 0 && year%100 != 0) {
			return 29
		}
		return 28
	default:
		return 31
	}
}

func iso(date provider.Date) string {
	pad := func(value, width int) string {
		digits := make([]byte, width)
		for i := width - 1; i >= 0; i-- {
			digits[i] = byte('0' + value%10)
			value /= 10
		}
		return string(digits)
	}
	return pad(date.Year, 4) + "-" + pad(date.Month, 2) + "-" + pad(date.Day, 2)
}

// A date the scheme cannot express is refused rather than folded into some
// other century, which is what makes the boundary a boundary.
func TestADateOutsideTheSchemeIsRefused(t *testing.T) {
	for _, date := range []string{"1799-12-31", "2300-01-01"} {
		_, err := pl.PESEL{}.Generate(request(1, map[string]string{provider.RoleBirthDate: date}))
		if err == nil {
			t.Errorf("%s was accepted, and the scheme covers 1800 to 2299", date)
		}
	}
}

// The case a straight subtraction from ten gets wrong. When the weighted total
// ends in nought the answer is nought and not ten, and ten is not a digit.
func TestACheckDigitOfNoughtIsProducedRatherThanTen(t *testing.T) {
	found := ""
	for row := int64(0); row < 2000 && found == ""; row++ {
		value := generate(t, row, nil)
		if strings.HasSuffix(value, "0") {
			found = value
		}
	}
	// 🔴 A failure and not a skip (B-319). About one value in ten ends in
	// nought, so a sample of two thousand without one says the generator stopped
	// producing that digit, which is the fault this test is about.
	if found == "" {
		t.Fatal("no value with a check digit of nought turned up in 2000 rows, where about " +
			"one in ten should")
	}
	if findings := oracle.CheckPESEL(found); len(findings) != 0 {
		t.Fatalf("%s ends in nought and the independent validator refuses it: %v", found, findings)
	}
}

func TestValidateRefusesWhatTheSchemeForbids(t *testing.T) {
	// A value known good, then broken one way at a time.
	good := generate(t, 7, map[string]string{provider.RoleBirthDate: "1985-06-15"})

	cases := []struct {
		name  string
		value string
		says  string
	}{
		{"too short", good[:10], "digits"},
		{"too long", good + "0", "digits"},
		{"not digits", "9007199254740991"[:10] + "x", "digits"},
		{"month in no century", "85136000000", "century"},
		{"impossible date", replaceAt(good, 2, "0230"), "does not exist"},
		{"broken check digit", flipLast(good), "check digit"},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			verdict := pl.PESEL{}.Validate(test.value)
			if verdict.Valid {
				t.Fatalf("%s was accepted", test.value)
			}
			if !strings.Contains(verdict.Reason, test.says) {
				t.Errorf("the reason was %q, expected it to mention %q", verdict.Reason, test.says)
			}
			// The independent validator has to refuse it too, or the two
			// disagree about what the scheme says.
			if findings := oracle.CheckPESEL(test.value); len(findings) == 0 {
				t.Errorf("the independent validator accepted %s", test.value)
			}
		})
	}
}

// Moving the date while keeping everything else is the corruption an
// application is least likely to catch: the number is faultless and disagrees
// with the rest of the record.
func TestTheDateCanBeMovedAndTheNumberStaysWellFormed(t *testing.T) {
	original := generate(t, 3, map[string]string{provider.RoleBirthDate: "1990-04-11"})

	moved, err := pl.PESEL{}.WithDate(original, provider.Date{Year: 1991, Month: 4, Day: 11})
	if err != nil {
		t.Fatalf("moving the date failed: %v", err)
	}
	if moved == original {
		t.Fatal("the value did not change")
	}
	if findings := oracle.CheckPESEL(moved); len(findings) != 0 {
		t.Fatalf("%s is not well formed after the move: %v", moved, findings)
	}
	// The ordinal number, gender digit included, has to survive. Changing it
	// as well would make the corruption a different one.
	if original[6:10] != moved[6:10] {
		t.Errorf("the ordinal number changed from %s to %s", original[6:10], moved[6:10])
	}
}

func TestTheGenderCanBeFlippedAndTheNumberStaysWellFormed(t *testing.T) {
	original := generate(t, 4, map[string]string{
		provider.RoleBirthDate: "1990-04-11",
		provider.RoleGender:    string(provider.Female),
	})

	flipped, err := pl.PESEL{}.WithGender(original, provider.Male)
	if err != nil {
		t.Fatalf("flipping the gender failed: %v", err)
	}
	if gender, _ := (pl.PESEL{}).EncodedGender(flipped); gender != provider.Male {
		t.Errorf("%s still reads as %q", flipped, gender)
	}
	if findings := oracle.CheckPESEL(flipped); len(findings) != 0 {
		t.Fatalf("%s is not well formed after the flip: %v", flipped, findings)
	}
	if _, err := (pl.PESEL{}).WithGender(original, provider.Unknown); err == nil {
		t.Error("the scheme encodes two values and unknown was accepted as a third")
	}
}

// # The worked example the government description never gave
//
// For a day this file said that no worked example existed at the source, so the
// weights rested on its prose alone with nothing to compare them against. That
// was true of the page the rules were read from and false of the world: the
// country sheet the Ministry of Finance of Poland publishes on the European TIN
// Portal, under its own copyright notice, takes one number apart digit by digit.
//
// It is quoted here in full, because it is the only number in this repository
// whose arithmetic somebody else did. Every intermediate value the source
// prints is asserted, not only the final digit: a test checking the answer
// alone would still pass with two compensating errors in the table, which is
// the shape a mistyped weight actually takes.
//
// The sheet describes the sum its own way, adding only the last digit of each
// product, so both readings are checked. They agree, which is the point.
func TestTheWorkedExampleFromTheMinistryIsReproduced(t *testing.T) {
	const published = "02070803628"

	if findings := oracle.CheckPESEL(published); len(findings) != 0 {
		t.Fatalf("the number the source works out is refused by the independent validator: %v", findings)
	}
	if verdict := (pl.PESEL{}).Validate(published); !verdict.Valid {
		t.Fatalf("the number the source works out is refused by our validator: %s", verdict.Reason)
	}

	// The source says: a person born on the 8 of July 1902, sex female.
	date, ok := pl.PESEL{}.EncodedDate(published)
	if !ok || date != (provider.Date{Year: 1902, Month: 7, Day: 8}) {
		t.Errorf("%s reads as %+v, the source says the 8 of July 1902", published, date)
	}
	if gender, _ := (pl.PESEL{}).EncodedGender(published); gender != provider.Female {
		t.Errorf("%s reads as %q, the source says female", published, gender)
	}

	// 0*1=0, 2*3=6, 0*7=0, 7*9=63, 0*1=0, 8*3=24, 0*7=0, 3*9=27, 6*1=6, 2*3=6
	products := []int{0, 6, 0, 63, 0, 24, 0, 27, 6, 6}
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9, 1, 3}
	lastDigitTotal := 0
	for position, want := range products {
		if got := int(published[position]-'0') * weights[position]; got != want {
			t.Errorf("position %d multiplies out to %d, the source says %d", position+1, got, want)
		}
		lastDigitTotal += want % 10
	}
	// 0+6+0+3+0+4+0+7+6+6=32, then 10-2=8.
	if lastDigitTotal != 32 {
		t.Errorf("the last digits of the products add up to %d, the source says 32", lastDigitTotal)
	}
	if got := (10 - lastDigitTotal%10) % 10; got != 8 {
		t.Errorf("the check digit works out to %d, the source says 8", got)
	}
	if published[10] != '8' {
		t.Errorf("the check digit in the published number is %c, the source says 8", published[10])
	}
}

// # Why this test still earns its place
//
// The worked example above pins the table to one official answer. This asks a
// different question, about every number rather than one: whether the table has
// the property a check digit table is supposed to have, that every single digit
// error changes the answer. A weight sharing a factor with ten would silently
// fail to, and one published example would not show it.
func TestEverySingleDigitErrorChangesTheCheckDigit(t *testing.T) {
	base := generate(t, 11, map[string]string{provider.RoleBirthDate: "1975-03-08"})

	for position := 0; position < 10; position++ {
		for replacement := byte('0'); replacement <= '9'; replacement++ {
			if base[position] == replacement {
				continue
			}
			damaged := base[:position] + string(replacement) + base[position+1:]
			if findings := oracle.CheckPESEL(damaged); len(findings) == 0 {
				t.Fatalf("changing position %d of %s to %c left a valid number", position, base, replacement)
			}
		}
	}
}

// The limitation of the same table, measured rather than assumed, and it is
// the reason the catalog carries a transposition mutator of its own.
//
// Adjacent weights differ by two, four, two and eight, and none of those is
// coprime with ten. So some swaps of neighbouring digits leave the total
// unchanged and pass every check. An application trusting the check digit to
// catch a typed transposition is trusting something the scheme never promised.
func TestSomeAdjacentTranspositionsSlipPastTheCheckDigit(t *testing.T) {
	// The swap is put inside the ordinal number, positions seven and eight,
	// on purpose. A swap across the date would change the date as well, and
	// then the date rule would refuse the value and this test would pass
	// while proving nothing about the check digit. That is the shape of a
	// test that reports success for the wrong reason.
	base := generate(t, 12, map[string]string{provider.RoleBirthDate: "1975-03-08"})

	// Digits differing by exactly five. The weights at these two positions
	// differ by eight, and eight times five is forty, which leaves the total
	// unchanged modulo ten.
	withPair, err := pl.PESEL{}.WithCheckDigits(base[:7] + "16" + base[9:])
	if err != nil {
		t.Fatalf("building the sample failed: %v", err)
	}
	if findings := oracle.CheckPESEL(withPair); len(findings) != 0 {
		t.Fatalf("the sample %s is not valid to begin with: %v", withPair, findings)
	}

	swapped := withPair[:7] + "61" + withPair[9:]
	if findings := oracle.CheckPESEL(swapped); len(findings) != 0 {
		t.Fatalf("swapping 1 and 6 in %s was caught, so the limitation this documents is gone: %v", withPair, findings)
	}
	t.Logf("%s and %s differ by a swap of two neighbouring digits and both are valid", withPair, swapped)

	// Stated as arithmetic as well, so the claim is about the scheme rather
	// than about this one number.
	weights := []int{1, 3, 7, 9, 1, 3, 7, 9, 1, 3}
	for position := 0; position < 9; position++ {
		difference := weights[position] - weights[position+1]
		if difference < 0 {
			difference = -difference
		}
		if difference%2 == 0 {
			return // At least one neighbouring pair cannot detect every swap.
		}
	}
	t.Fatal("every neighbouring pair of weights is coprime with ten, so this limitation no longer holds and the comment above is wrong")
}

func replaceAt(value string, at int, with string) string {
	return value[:at] + with + value[at+len(with):]
}

func flipLast(value string) string {
	last := value[len(value)-1]
	if last == '0' {
		return value[:len(value)-1] + "1"
	}
	return value[:len(value)-1] + "0"
}
