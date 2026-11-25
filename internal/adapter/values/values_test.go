package values_test

import (
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
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

// Drawing under a limit leaves the shared word list as it was.
//
// # What this is really about
//
// 🔴 The lists are read once and handed to every later caller, so the slice a
// generator receives is the one the next row will receive. Honouring a maximum
// length means keeping the entries that fit, and doing that in place would move
// them to the front of the shared array and shorten what everybody sees
// afterwards. Every row drawn after the first limited one would then come from
// a list nobody chose, silently, and the dataset would still look ordinary.
//
// The code is correct because it filters through a zero capacity reslice, which
// forces the append to allocate a new array. That is one character away from
// being wrong, no compiler notices the difference, and the only symptom is
// wrong data. So the invariant is asserted rather than described.
func TestDrawingUnderALimitLeavesTheSharedListAlone(t *testing.T) {
	const list = "cities"
	before, found := corpus.Load("pl-PL", list)
	if !found {
		t.Fatalf("this build carries no %s for pl-PL, so this test proves nothing", list)
	}
	kept := append([]string(nil), before...)
	if len(kept) < 2 {
		t.Fatalf("a list of %d entries cannot show this", len(kept))
	}

	source := values.New("pl-PL", "shared")
	for row := int64(0); row < 20; row++ {
		request := ask(values.City, "city", row)
		request.MaxLength, request.HasMaxLength, request.Unit = 6, true, "characters"
		if _, err := source.Produce(request); err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
	}

	after, _ := corpus.Load("pl-PL", list)
	if len(after) != len(kept) {
		t.Fatalf("the shared list held %d entries and now holds %d", len(kept), len(after))
	}
	for index := range kept {
		if after[index] != kept[index] {
			t.Fatalf("entry %d of the shared list was %q and is now %q",
				index, kept[index], after[index])
		}
	}
}

// What this build says it can produce is what it produces, for every generator
// in every locale.
//
// # Why this test exists at all
//
// 🔴 The answer the plan gives is a second copy of the knowledge held by the
// generators themselves: which word lists each one draws on, and how that
// depends on the locale. A second copy is a place for the two to part company,
// and this project has paid for that shape before. Nothing in the compiler
// notices when a generator starts drawing on a list its answer does not name,
// and the failure would be the exact one the answer was added to prevent: a
// recipe called usable that dies on the first row, or a recipe refused that
// would have worked.
//
// # Why the answer is checked for every gender rather than once
//
// Which list a name is drawn from depends on the gender the row carries, so a
// build holding one list and not the other produces some rows and fails on
// others. The promise the plan makes is about the whole run, so a yes has to
// hold for every binding and a no has to be a real failure for at least one.
func TestWhatThisBuildSaysItCanProduceIsWhatItProduces(t *testing.T) {
	bindings := []struct {
		what   string
		agrees provider.Agreement
	}{
		{"no gender", provider.Agreement{}},
		{"a woman", provider.NewAgreement(map[string]string{
			provider.RoleGender: string(provider.Female)})},
		{"a man", provider.NewAgreement(map[string]string{
			provider.RoleGender: string(provider.Male)})},
	}
	// A locale this build does not carry stands beside the ones it does, so
	// that the refusing half of the answer is exercised and not only the
	// agreeing half.
	for _, locale := range []string{"en-US", "pl-PL", "xx-XX"} {
		source := values.New(locale, "can-produce")
		for _, generator := range source.Names() {
			fine, why := source.CanProduce(generator, locale)
			failed := ""
			for _, binding := range bindings {
				request := ask(generator, "f", 1)
				request.Agrees = binding.agrees
				if _, err := source.Produce(request); err != nil {
					failed = binding.what + ": " + err.Error()
					break
				}
			}
			switch {
			case fine && failed != "":
				t.Errorf("%s in %s is said to be producible and is not, for %s",
					generator, locale, failed)
			case !fine && failed == "":
				t.Errorf("%s in %s is refused as %q and produces a value for every gender",
					generator, locale, why)
			case !fine && why == "":
				t.Errorf("%s in %s is refused without saying why", generator, locale)
			}
		}
	}
}

// A Polish surname inflects with the gender of the bearer, so a woman must
// never be given the masculine form, and must be given the feminine one.
//
// 🔴 Until 2026-10-21 this test asserted the first half only and said why: the
// corpus carries no feminine forms, and building them was called a rule this
// project would invent. It is the rule of the language, Kowalski and Kowalska,
// and the header of the masculine list said from the start that the feminine
// form is derived by rule. Measured that day before the change: no woman in a
// thousand rows had a surname ending in ska, and about one in a hundred had a
// masculine adjective such as Mały or Brudny.
func TestAPolishSurnameNeverGivesAWomanTheMasculineForm(t *testing.T) {
	source := values.New("pl-PL", "surnames")
	masculine, feminine := 0, 0
	listedMasculine, found := corpus.Load("pl-PL", "last-names-male")
	if !found {
		t.Fatal("this build carries no masculine surnames for pl-PL")
	}
	masculineList := make(map[string]bool, len(listedMasculine))
	for _, name := range listedMasculine {
		masculineList[name] = true
	}

	for row := int64(0); row < 2000; row++ {
		female := ask(values.LastName, "last_name", row)
		female.Agrees = provider.NewAgreement(map[string]string{
			provider.RoleGender: string(provider.Female)})
		drawn, err := source.Produce(female)
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if inflectsMasculine(drawn) || strings.HasSuffix(drawn, "y") || strings.HasSuffix(drawn, "i") {
			t.Errorf("row %d gave a woman the masculine form %q", row, drawn)
		}
		// Counted against the masculine list rather than by ending, because the
		// list of surnames that do not inflect carries a few ending in ska of
		// its own, Kuska and Gryska, and counting by ending let a build without
		// a single feminine form pass. Found by a mutation that removed them.
		if trimmed, isFeminine := strings.CutSuffix(drawn, "a"); isFeminine && masculineList[trimmed+"i"] {
			feminine++
		}

		neither := ask(values.LastName, "last_name", row)
		if drawn, err = source.Produce(neither); err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if inflectsMasculine(drawn) || strings.HasSuffix(drawn, "y") {
			t.Errorf("row %d gave a person of no stated gender the masculine form %q", row, drawn)
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
	if feminine == 0 {
		t.Error("no woman received a surname ending in ska in two thousand draws, so the " +
			"feminine form of an inflecting surname is never reached")
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

// Only the generators another one can read play a role, and the role comes from
// the generator rather than from the name of the column. The city joined the
// date of birth and the gender on 2026-11-12, read by a region in en-US
// (DEC-278), and the region the same day, read by a postal code (DEC-282).
func TestTheRoleComesFromTheGeneratorAndOnlyFourPlayOne(t *testing.T) {
	source := values.New("pl-PL", "roles")
	for generator, wanted := range map[string]string{
		values.BirthDate: provider.RoleBirthDate,
		values.Gender:    provider.RoleGender,
		values.City:      provider.RoleCity,
		values.Region:    provider.RoleRegion,
	} {
		role, plays := source.RoleOf(generator)
		if !plays || role != wanted {
			t.Errorf("%s plays %q and should play %q", generator, role, wanted)
		}
	}
	if _, plays := source.RoleOf(values.ISBN); plays {
		t.Error("an ISBN plays a role nothing reads")
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

// Every generator stands on exactly one shelf: a scheme or a topic.
//
// # Why this is a guard and not a remark
//
// A surface listing every kind puts each under a heading, and the heading is
// read off the scheme for eleven generators and off a declared topic for the
// other six. A generator with neither would land under no heading at all, and
// nothing about building the window would say so - the list would simply be
// one entry short in a place nobody counts. A generator with both would be a
// second answer to a question the scheme already answered.
func TestEveryGeneratorStandsOnExactlyOneShelf(t *testing.T) {
	source := values.New("en-US", "shelves")
	for _, generator := range source.Names() {
		_, behindScheme := source.ProviderOf(generator)
		topic, declared := source.TopicOf(generator)
		switch {
		case behindScheme && declared:
			t.Errorf("%s is behind a scheme AND declares the topic %q, which is two answers", generator, topic)
		case !behindScheme && !declared:
			t.Errorf("%s is behind no scheme and declares no topic, so no list can shelve it", generator)
		case declared && topic != values.TopicPeople && topic != values.TopicPlaces &&
			topic != values.TopicAccounts:
			t.Errorf("%s declares the topic %q, which is not a shelf this build names", generator, topic)
		}
	}
	if topic, declared := source.TopicOf(values.Pesel); declared {
		t.Errorf("a PESEL is behind a scheme and still answered with the topic %q", topic)
	}
}
