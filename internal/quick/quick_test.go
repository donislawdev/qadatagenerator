package quick

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// TestTheValueIsTheOneARecipeWouldProduce holds the claim that makes this mode
// worth having rather than dangerous: it is a shorter way to ask the same
// engine, not a second engine.
//
// # Why a whole run of a recipe rather than a call to the same function
//
// Calling what this package calls would prove that a line of code is the line
// of code it is. What is worth holding is that the ADDRESS this mode invents
// for its one value is the address a run of a recipe would use for the same
// value, because that address is made up here and nothing else would notice if
// it drifted. So a recipe is built with an entity of this mode's name and one
// column named after the generator, run through the ordinary engine, and the
// cell it produces is compared with what this mode answers.
//
// A drift would not fail anything else. The value would still be a valid PESEL,
// the seed would still be honoured, and the two surfaces would still agree with
// each other, because both of them ask this package. Only this notices.
func TestTheValueIsTheOneARecipeWouldProduce(t *testing.T) {
	const seed = "the-same-seed"

	// 🔴 More than one row, and that is the whole strength of this guard since
	// the count arrived. One row proves the address of the FIRST value, and
	// the address is what a count moves: a mode counting its rows from
	// somewhere else, or not counting them at all, agrees with a recipe on the
	// first value and parts company on the second.
	const rows = 4

	for _, generator := range Generators(DefaultLocale) {
		asked := Request{Generator: generator, Locale: DefaultLocale, Seed: seed, Count: rows}
		fromMode, modeErr := Run(asked)

		sample, err := produce.Preview(recipeFor(generator, seed), values.New(DefaultLocale, seed), rows,
			generate.NeverStops{})
		if err != nil {
			// A generator this build has no data for in this locale is refused
			// by both, which is agreement of the kind this test is about. What
			// would be a finding is one of them refusing alone.
			if modeErr == nil {
				t.Errorf("a recipe of %s cannot be run here and this mode produced a value anyway",
					generator)
			}
			continue
		}
		if modeErr != nil {
			t.Errorf("a recipe of %s runs and this mode refused it: %v", generator, modeErr)
			continue
		}

		fromRecipe := everyCell(sample)
		if len(fromRecipe) == 0 {
			t.Errorf("a recipe of %s produced no cell to compare against", generator)
			continue
		}
		if len(fromMode) != len(fromRecipe) {
			t.Errorf("%s: this mode produced %d values where a recipe of the same seed "+
				"produces %d rows", generator, len(fromMode), len(fromRecipe))
			continue
		}
		for row := range fromMode {
			if fromMode[row] != fromRecipe[row] {
				t.Errorf("%s: at row %d this mode answers %q where a recipe of the same "+
					"seed produces %q. The address this mode invents has drifted from "+
					"the one the engine uses",
					generator, row, fromMode[row], fromRecipe[row])
			}
		}
	}
}

// recipeFor is a recipe of one table, one column and as many rows as the guard
// asks for, named so that the address of each cell is the address this mode
// uses for the value at that position.
func recipeFor(generator, seed string) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1",
		Seed:    seed,
		Locale:  DefaultLocale,
		Entities: []recipe.Entity{{
			Name:     Entity,
			Count:    4,
			HasCount: true,
			Fields:   []recipe.Field{{Name: generator, Generator: generator}},
		}},
	}
}

// everyCell is the values that recipe produced, in the order it produced them.
func everyCell(sample produce.Sample) []string {
	var cells []string
	for _, table := range sample.Tables {
		for _, row := range table.Rows {
			cells = append(cells, row...)
		}
	}
	return cells
}

// TestEachHandsOutAHugeCountOneValueAtATime holds R12: a count with no ceiling
// is drawn one value at a time, and the first error take returns ends it.
//
// 🔴 The count is the largest an int holds, and that is what makes this a guard
// rather than an example. Holding the values first fails at once on a count
// like this one, and drawing on after take said stop asks for a fourth value,
// which ends the test from inside take rather than leaving it running.
func TestEachHandsOutAHugeCountOneValueAtATime(t *testing.T) {
	gone := errors.New("the reader has gone")
	asked := Request{Generator: "pesel", Locale: DefaultLocale, Seed: "for-the-test", Count: math.MaxInt}
	var taken []string
	err := Each(asked, func(value string) error {
		if len(taken) == 3 {
			t.Fatalf("a fourth value was drawn after the third was answered with an error")
		}
		taken = append(taken, value)
		if len(taken) == 3 {
			return gone
		}
		return nil
	})
	if !errors.Is(err, gone) {
		t.Errorf("the error take returned came back as %v, so a caller whose output failed "+
			"cannot tell that from anything else", err)
	}
	asked.Count = 3
	bounded, runErr := Run(asked)
	if runErr != nil {
		t.Fatalf("three values could not be produced: %v", runErr)
	}
	if !slices.Equal(taken, bounded) {
		t.Errorf("one value at a time gave %q and the same three held together gave %q", taken, bounded)
	}
}

// An error of the engine ends the drawing before take hears of a value, and
// comes back as the engine said it.
func TestEachEndsWhereTheEngineRefuses(t *testing.T) {
	asked := Request{Generator: "telephone", Locale: DefaultLocale, Seed: "for-the-test", Count: 5}
	handed := 0
	err := Each(asked, func(string) error {
		handed++
		return nil
	})
	if err == nil {
		t.Fatal("a generator this build does not have was drawn from without an error")
	}
	if handed != 0 {
		t.Errorf("take was handed %d values from a generator the engine refused", handed)
	}
}

// TestQuickRefusesWhatItCannotProduce walks every refusal this mode has and
// holds that each one is reached by the request it is about.
//
// # Why the kinds are checked rather than the sentences
//
// The sentences live on the surfaces, one set per surface, and a guard beside
// each register holds it against Kinds in both directions. What has to be true here
// is that the kind a request produces is the kind that request deserves, since
// a refusal of the wrong kind is a confident four part message about something
// that did not happen.
func TestQuickRefusesWhatItCannotProduce(t *testing.T) {
	cases := []struct {
		named string
		asked Request
		want  RefusalKind
	}{
		{"a value this build has no generator for",
			Request{Generator: "telephone"}, UnknownGenerator},
		{"a gender beside a value that takes no account of one",
			Request{Generator: "ssn", Agrees: map[string]string{RoleGender: "female"}}, RoleNotRead},
		{"a date of birth beside a value that takes no account of one",
			Request{Generator: "ssn", Agrees: map[string]string{RoleBirthDate: "1980-05-05"}}, RoleNotRead},
		// 🔴 A word with no meaning, not a short spelling. Until DEC-214 the
		// short ones were refused here, because a first name ignored them while
		// a PESEL took them. They are taken now, everywhere, so the case that
		// belongs in this list is the one that is not a gender at all.
		{"a word that is not a gender in any spelling",
			Request{Generator: "pesel", Agrees: map[string]string{RoleGender: "yes"}}, UnreadableGender},
		{"a gender written the short way, which every reader takes",
			Request{Generator: "pesel", Agrees: map[string]string{RoleGender: "f"}}, NotRefused},
		{"a gender written as the digit a database export gives",
			Request{Generator: "pesel", Agrees: map[string]string{RoleGender: "2"}}, NotRefused},
		{"a date of birth nobody can read",
			Request{Generator: "pesel", Agrees: map[string]string{RoleBirthDate: "05-05-1980"}}, UnreadableDate},
		{"a request with nothing wrong with it",
			Request{Generator: "pesel", Agrees: map[string]string{
				RoleGender: "female", RoleBirthDate: "1980-05-05"}}, NotRefused},
		{"more than one value, which is what a count is for",
			Request{Generator: "pesel", Count: 25}, NotRefused},
		// 🔴 The count of these two is written out rather than left to the
		// zero value, because the zero value is what the case below is about
		// and a table where one row means the absence of a field and another
		// means a chosen nought cannot say which it tested.
		{"no values at all, which this verb has nothing to answer with",
			Request{Generator: "pesel", Count: 0}, CountNotPositive},
		{"fewer values than none",
			Request{Generator: "pesel", Count: -3}, CountNotPositive},
		// The postal code of pl-PL is the one kind lacking data in a locale
		// this build knows (DEC-92 holds it). The day it arrives this row has
		// to find another, and the one below it keeps the refusal reachable.
		{"a value this build carries no data for in that locale",
			Request{Generator: "postal_code", Locale: "pl-PL"}, Unavailable},
		{"a value drawn from a list, in a locale this build has never heard of",
			Request{Generator: "first_name", Locale: "xx-XX"}, Unavailable},
		{"the same value in a locale that has its data",
			Request{Generator: "postal_code", Locale: "en-US"}, NotRefused},
		{"a value no locale changes, in a locale this build has never heard of",
			Request{Generator: "pesel", Locale: "xx-XX"}, NotRefused},
		// Unavailable comes before the roles: a gender beside a first name
		// that cannot be drawn at all is not the thing to fix first.
		{"a gender beside a value that cannot be produced in that locale",
			Request{Generator: "first_name", Locale: "xx-XX",
				Agrees: map[string]string{RoleGender: "yes"}}, Unavailable},
	}

	for _, one := range cases {
		// A count is filled in for every case that did not name one, so that
		// the table goes on being about the thing each row is named after. The
		// two rows about the count say so themselves.
		asked := one.asked
		if asked.Count == 0 && one.want != CountNotPositive {
			asked.Count = 1
		}
		got := Check(asked).Kind
		if got != one.want {
			t.Errorf("%s was answered with kind %d where %d is the one about it",
				one.named, got, one.want)
		}
		// The registers on the surfaces are held against Kinds, so a kind
		// Check can reach and Kinds leaves out would be one nobody has words
		// for, with every guard green.
		if got != NotRefused && !slices.Contains(Kinds(), got) {
			t.Errorf("%s was refused as kind %d, which Kinds does not name", one.named, got)
		}
	}
}

// TestARefusedLocaleNamesTheLocalesTheValueIsProducedIn holds the next move
// both surfaces point at, which is why this refusal is made before the run
// rather than by it.
//
// Allowed is held against what Check itself answers in each locale, so the
// two cannot disagree: a locale named here is one where the same request is
// carried out, and every such locale is named.
func TestARefusedLocaleNamesTheLocalesTheValueIsProducedIn(t *testing.T) {
	refusal := Check(Request{Generator: "postal_code", Locale: "pl-PL", Count: 1})
	if refusal.Kind != Unavailable {
		t.Fatalf("a postal code in pl-PL was answered with kind %d. If its data has "+
			"arrived (DEC-92), choose another kind this build lacks data for", refusal.Kind)
	}
	if refusal.Value != "postal_code" || refusal.Locale != "pl-PL" {
		t.Errorf("the refusal names %q in %q rather than what was asked for",
			refusal.Value, refusal.Locale)
	}
	if len(refusal.Allowed) == 0 {
		t.Fatal("the refusal names no locale at all, so the next move points nowhere")
	}
	for _, locale := range Locales() {
		carried := !Check(Request{Generator: "postal_code", Locale: locale, Count: 1}).Refused()
		named := slices.Contains(refusal.Allowed, locale)
		if carried != named {
			t.Errorf("postal_code is carried out in %s: %v, and the refusal names it: %v",
				locale, carried, named)
		}
	}
}

// TestKindsIsEveryRefusalAndNothingElse holds the count Kinds is made by.
func TestKindsIsEveryRefusalAndNothingElse(t *testing.T) {
	kinds := Kinds()
	if slices.Contains(kinds, NotRefused) {
		t.Error("Kinds names the absence of a refusal as one of them")
	}
	if slices.Contains(kinds, kindsEnd) {
		t.Error("Kinds names the marker after the last refusal as one of them")
	}
	for _, kind := range []RefusalKind{UnknownGenerator, CountNotWhole, Unavailable} {
		if !slices.Contains(kinds, kind) {
			t.Errorf("Kinds leaves out kind %d", kind)
		}
	}
}

// TestASeedIsDrawnAfreshEveryTime holds the one place in this tool where a run
// is not repeatable, and holds it from both sides: two draws differ, and two
// runs of a named seed do not.
func TestASeedIsDrawnAfreshEveryTime(t *testing.T) {
	first, err := NewSeed()
	if err != nil {
		t.Fatalf("no seed could be drawn: %v", err)
	}
	second, err := NewSeed()
	if err != nil {
		t.Fatalf("no second seed could be drawn: %v", err)
	}
	if first == second {
		t.Error("two draws gave the same seed, so asking twice would give the same value " +
			"and the whole point of this mode is that it does not")
	}

	asked := Request{Generator: "pesel", Locale: DefaultLocale, Seed: "a-named-seed", Count: 3}
	once, err := Run(asked)
	if err != nil {
		t.Fatalf("the named seed produced nothing: %v", err)
	}
	twice, err := Run(asked)
	if err != nil {
		t.Fatalf("the named seed produced nothing the second time: %v", err)
	}
	if !slices.Equal(once, twice) {
		t.Errorf("a named seed produced %q and then %q, so naming one does not bring "+
			"repeatability back and the seventh rule is broken rather than excepted",
			once, twice)
	}
}

// 🔴 The other half of MostShownAtOnce: this mode does NOT enforce it.
//
// The constant lives here so that both surfaces can see one number, and a
// constant sitting in a package is an invitation to start applying it. The day
// somebody does, the command line stops being able to produce a million values
// and nothing in the window would notice - so the claim that it is a limit of
// DRAWING and not of producing is held by a test rather than by the paragraph
// beside it.
func TestTheModeDoesNotApplyTheCeilingTheScreenDraws(t *testing.T) {
	asked := Request{
		Generator: "pesel", Locale: DefaultLocale, Seed: "for-the-test",
		Agrees: map[string]string{}, Count: MostShownAtOnce * 50,
	}
	if refusal := Check(asked); refusal.Refused() {
		t.Errorf("asking for %d values was refused as %v, so the ceiling the SCREEN "+
			"draws has become a limit of the mode and the command line lost with it",
			asked.Count, refusal.Kind)
	}
}

// A count below one is still the mode's business, so that both surfaces refuse
// it for the same reason and neither has to remember why.
func TestFewerThanOneValueIsStillTheModesRefusal(t *testing.T) {
	asked := Request{
		Generator: "pesel", Locale: DefaultLocale, Seed: "for-the-test",
		Agrees: map[string]string{}, Count: 0,
	}
	if refusal := Check(asked); refusal.Kind != CountNotPositive {
		t.Errorf("a count of nought was answered with %v rather than CountNotPositive",
			refusal.Kind)
	}
}

// CountFrom answers the SHAPE of the text, and 010 is ten rather than eight -
// which is the trap DEC-170 cost this project once, in a recipe reader that
// read a leading nought as octal.
func TestCountFromReadsBaseTenAndRefusesWhatIsNotANumber(t *testing.T) {
	for text, wanted := range map[string]int{"1": 1, "010": 10, "20000": 20000} {
		got, refused := CountFrom(text)
		if refused.Refused() {
			t.Errorf("%q was refused and it is a number", text)
			continue
		}
		if got != wanted {
			t.Errorf("%q was read as %d rather than %d", text, got, wanted)
		}
	}
	// 🔴 Padding is refused HERE and trimmed by the window before it asks.
	// Loosening this would have widened a flag that is already published.
	for _, text := range []string{"", "ten", "1.5", " 10 ", "10x"} {
		if _, refused := CountFrom(text); !refused.Refused() {
			t.Errorf("%q was taken as a number of values", text)
		}
	}
}
