package quick

import (
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

	for _, generator := range Generators(DefaultLocale) {
		asked := Request{Generator: generator, Locale: DefaultLocale, Seed: seed}
		fromMode, modeErr := Run(asked)

		sample, err := produce.Preview(recipeFor(generator, seed), values.New(DefaultLocale, seed), 1)
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

		fromRecipe, found := firstCell(sample)
		if !found {
			t.Errorf("a recipe of %s produced no cell to compare against", generator)
			continue
		}
		if fromMode != fromRecipe {
			t.Errorf("%s: this mode answers %q where a recipe of the same seed produces %q. "+
				"The address this mode invents has drifted from the one the engine uses",
				generator, fromMode, fromRecipe)
		}
	}
}

// recipeFor is a recipe of one table, one row and one column, named so that the
// address of that cell is the address this mode uses.
func recipeFor(generator, seed string) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1",
		Seed:    seed,
		Locale:  DefaultLocale,
		Entities: []recipe.Entity{{
			Name:     Entity,
			Count:    1,
			HasCount: true,
			Fields:   []recipe.Field{{Name: generator, Generator: generator}},
		}},
	}
}

// firstCell is the one value that recipe produced.
func firstCell(sample produce.Sample) (string, bool) {
	for _, table := range sample.Tables {
		for _, row := range table.Rows {
			for _, cell := range row {
				return cell, true
			}
		}
	}
	return "", false
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
		{"a gender spelled a way this mode does not take",
			Request{Generator: "pesel", Agrees: map[string]string{RoleGender: "f"}}, UnreadableGender},
		{"a date of birth nobody can read",
			Request{Generator: "pesel", Agrees: map[string]string{RoleBirthDate: "05-05-1980"}}, UnreadableDate},
		{"a request with nothing wrong with it",
			Request{Generator: "pesel", Agrees: map[string]string{
				RoleGender: "female", RoleBirthDate: "1980-05-05"}}, NotRefused},
	}

	for _, one := range cases {
		got := Check(one.asked).Kind
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

	asked := Request{Generator: "pesel", Locale: DefaultLocale, Seed: "a-named-seed"}
	once, err := Run(asked)
	if err != nil {
		t.Fatalf("the named seed produced nothing: %v", err)
	}
	twice, err := Run(asked)
	if err != nil {
		t.Fatalf("the named seed produced nothing the second time: %v", err)
	}
	if once != twice {
		t.Errorf("a named seed produced %q and then %q, so naming one does not bring "+
			"repeatability back and the seventh rule is broken rather than excepted",
			once, twice)
	}
}
