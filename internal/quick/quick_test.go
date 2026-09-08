package quick

import (
	"slices"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
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

		sample, err := produce.Preview(recipeFor(generator, seed), values.New(DefaultLocale, seed), rows)
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

// TestQuickRefusesWhatItCannotProduce walks every refusal this mode has and
// holds that each one is reached by the request it is about.
//
// # Why the kinds are checked rather than the sentences
//
// The sentences live on the surfaces, one set per surface, and both are held
// against this list of kinds by their own registers. What has to be true here
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
