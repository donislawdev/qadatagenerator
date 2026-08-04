package values_test

import (
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

func at(field string, row int64) seed.Coordinate {
	return seed.Coordinate{Entity: "t", Row: row, Field: field, Purpose: "value"}
}

func ask(generator, field string, row int64) generate.Request {
	return generate.Request{Generator: generator, At: at(field, row)}
}

// A Polish surname inflects with the gender of the bearer, so a woman must
// never be given the masculine form.
//
// 🔴 What this test does NOT assert, and the reason is a property of the data
// rather than of the code. The corpus carries 200 masculine surnames, half of
// them the inflecting kind, and 3690 that do not inflect. It carries no
// feminine inflecting forms at all, so no woman in a generated dataset ever
// receives one. Building them by rewriting the masculine ending would be a rule
// this project invented rather than took from anywhere, so it is recorded as a
// limit instead of coded. Asserting a feminine form here would have frozen an
// expectation the data cannot meet.
func TestAPolishSurnameNeverGivesAWomanTheMasculineForm(t *testing.T) {
	source := values.New("pl-PL", "surnames")
	masculine := 0

	for row := int64(0); row < 400; row++ {
		female := ask(values.LastName, "last_name", row)
		female.Agrees = provider.NewAgreement(map[string]string{
			provider.RoleGender: string(provider.Female)})
		drawn, err := source.Produce(female)
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if inflectsMasculine(drawn) {
			t.Errorf("row %d gave a woman the masculine form %q", row, drawn)
		}

		male := ask(values.LastName, "last_name", row)
		male.Agrees = provider.NewAgreement(map[string]string{
			provider.RoleGender: string(provider.Male)})
		if drawn, err = source.Produce(male); err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if inflectsMasculine(drawn) {
			masculine++
		}
	}
	if masculine == 0 {
		t.Error("no man received an inflecting surname in four hundred draws, so this test would " +
			"pass against a build that never reaches the masculine list at all")
	}
}

func inflectsMasculine(name string) bool {
	return strings.HasSuffix(name, "ski") || strings.HasSuffix(name, "cki") ||
		strings.HasSuffix(name, "dzki")
}

// An English surname does not inflect, so the same request works without a
// gender and must not fall through to a list that does not exist.
func TestAnEnglishSurnameNeedsNoGender(t *testing.T) {
	drawn, err := values.New("en-US", "surnames").Produce(ask(values.LastName, "last_name", 1))
	if err != nil {
		t.Fatalf("a surname was refused: %v", err)
	}
	if drawn == "" {
		t.Error("a surname came back empty")
	}
}

// A postal code exists for one locale and this build says so about the other
// rather than inventing one.
func TestAPostalCodeIsRefusedWhereThereIsNoRegister(t *testing.T) {
	if _, err := values.New("en-US", "codes").Produce(ask(values.PostalCode, "zip", 0)); err != nil {
		t.Errorf("a postal code was refused where the register exists: %v", err)
	}

	_, err := values.New("pl-PL", "codes").Produce(ask(values.PostalCode, "kod", 0))
	if err == nil {
		t.Fatal("a Polish postal code was produced and this build has no register of them")
	}
	if !strings.Contains(err.Error(), "pl-PL") {
		t.Errorf("the refusal does not say which locale it is about: %v", err)
	}
}

// A limit on length is honoured by choosing an entry that fits, not by cutting
// one that does not. A truncated name is a value this tool invented.
func TestALimitIsHonouredByChoosingRatherThanCutting(t *testing.T) {
	source := values.New("pl-PL", "short")
	for row := int64(0); row < 40; row++ {
		request := ask(values.FirstName, "first_name", row)
		request.MaxLength, request.HasMaxLength, request.Unit = 5, true, "characters"
		drawn, err := source.Produce(request)
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if len([]rune(drawn)) > 5 {
			t.Fatalf("row %d gave %q, which is longer than the field accepts", row, drawn)
		}
	}
}

// A limit nothing can meet is a failure with the arithmetic in it rather than a
// value that does not fit.
func TestALimitNothingCanMeetIsRefused(t *testing.T) {
	request := ask(values.FirstName, "first_name", 0)
	request.MaxLength, request.HasMaxLength, request.Unit = 1, true, "characters"
	_, err := values.New("pl-PL", "impossible").Produce(request)
	if err == nil {
		t.Fatal("a name of one character was produced from a list that has none")
	}
	if !strings.Contains(err.Error(), "first_name") {
		t.Errorf("the refusal does not name the field: %v", err)
	}
}

// A scheme of fixed width cannot choose a shorter value, so it says so instead.
func TestAFixedWidthValueTooLongForItsFieldIsRefused(t *testing.T) {
	request := ask(values.Pesel, "pesel", 0)
	request.MaxLength, request.HasMaxLength, request.Unit = 5, true, "characters"
	_, err := values.New("pl-PL", "narrow").Produce(request)
	if err == nil {
		t.Fatal("an eleven digit number was accepted into a field of five")
	}
	if !strings.Contains(err.Error(), "pesel") {
		t.Errorf("the refusal does not name the generator: %v", err)
	}
}

// Bytes and characters disagree on exactly the values this tool produces, which
// is the whole reason the unit is carried rather than assumed.
func TestBytesAndCharactersAreCountedDifferently(t *testing.T) {
	// 🔴 One draw was the whole of this test until 2026-08-04, and one draw
	// cannot tell the two units apart: the name that came up happened to be
	// short in both, so a build counting characters everywhere passed. Found by
	// mutation, which reported the collapse of the two units as unnoticed.
	source := values.New("pl-PL", "units")
	multibyte := 0

	for row := int64(0); row < 200; row++ {
		inBytes := ask(values.FirstName, "first_name", row)
		inBytes.MaxLength, inBytes.HasMaxLength, inBytes.Unit = 6, true, "bytes"
		drawn, err := source.Produce(inBytes)
		if err != nil {
			t.Fatalf("row %d: a name of six bytes was refused: %v", row, err)
		}
		if len(drawn) > 6 {
			t.Fatalf("row %d gave %q, which is %d bytes and the field accepts six",
				row, drawn, len(drawn))
		}
		if len(drawn) != len([]rune(drawn)) {
			multibyte++
		}
	}

	if multibyte == 0 {
		t.Error("no drawn name carried a letter costing more than one byte in two hundred draws, " +
			"so this test would pass against a build that counts the two units alike")
	}
}

// A name this build does not have is refused rather than answered with nothing,
// and the registry can list what it does have.
func TestAnUnknownGeneratorIsRefusedAndTheKnownOnesCanBeListed(t *testing.T) {
	source := values.New("en-US", "names")
	if source.Known("nosuchthing") {
		t.Error("a generator that does not exist was reported as known")
	}
	if _, err := source.Produce(ask("nosuchthing", "x", 0)); err == nil {
		t.Error("an unknown generator produced a value")
	}
	names := source.Names()
	if len(names) < 10 {
		t.Errorf("the registry lists %d generators, which is fewer than it has", len(names))
	}
	for index := 1; index < len(names); index++ {
		if names[index-1] >= names[index] {
			t.Fatalf("the list is not sorted, so two runs could disagree about it: %v", names)
		}
	}
}

// Only the generators an identifier can read play a role, and the role comes
// from the generator rather than from the name of the column.
func TestTheRoleComesFromTheGeneratorAndOnlyTwoPlayOne(t *testing.T) {
	source := values.New("pl-PL", "roles")
	for generator, wanted := range map[string]string{
		values.BirthDate: provider.RoleBirthDate,
		values.Gender:    provider.RoleGender,
	} {
		role, plays := source.RoleOf(generator)
		if !plays || role != wanted {
			t.Errorf("%s plays %q and should play %q", generator, role, wanted)
		}
	}
	if _, plays := source.RoleOf(values.City); plays {
		t.Error("a city plays a role no identifier reads")
	}
}

// A date has to be a real one, whatever month it lands in.
func TestEveryDrawnDateIsARealDay(t *testing.T) {
	source := values.New("en-US", "dates")
	seen := map[string]bool{}
	for row := int64(0); row < 500; row++ {
		drawn, err := source.Produce(ask(values.BirthDate, "born", row))
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if len(drawn) != 10 || drawn[4] != '-' || drawn[7] != '-' {
			t.Fatalf("row %d gave %q, which is not a date written the one stable way", row, drawn)
		}
		// 🔴 The shape above was the whole of this test until 2026-08-04, and it
		// is not what the name promises: a thirtieth of February is written the
		// same stable way as any other day. Found by asking what mutation would
		// make this go red and finding that breaking the length of a month made
		// no test anywhere fail.
		if _, err := time.Parse("2006-01-02", drawn); err != nil {
			t.Fatalf("row %d gave %q, which is not a day that exists: %v", row, drawn, err)
		}
		seen[drawn[5:7]] = true
	}
	if len(seen) != 12 {
		t.Errorf("only %d months came up in five hundred draws", len(seen))
	}
}
