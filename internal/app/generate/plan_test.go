package generate_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// knownValues is a source of values that answers about names rather than
// producing anything.
//
// The plan never produces a value, so a stub is the honest shape here: handing
// in the real registry would make these tests depend on which generators this
// build happens to carry, and they are about what the plan SAYS rather than
// about what any generator does.
type knownValues struct {
	generators []string
	types      map[string]string
	produces   []string
}

func (v knownValues) Known(generator string) bool {
	for _, name := range v.generators {
		if name == generator {
			return true
		}
	}
	return false
}

func (v knownValues) Names() []string { return v.generators }

func (v knownValues) RoleOf(string) (string, bool) { return "", false }

func (v knownValues) TypeOf(generator string) (string, bool) {
	kind, ok := v.types[generator]
	return kind, ok
}

func (v knownValues) ProducesType(kind string) bool {
	for _, name := range v.produces {
		if name == kind {
			return true
		}
	}
	return false
}

func (v knownValues) Produce(generate.Request) (string, error) { return "", nil }

func oneEntity(fields ...recipe.Field) recipe.Recipe {
	return recipe.Recipe{
		Seed: "plan", Locale: "pl-PL",
		Entities: []recipe.Entity{{Name: "order", Count: 2, HasCount: true, Fields: fields}},
	}
}

func noteAbout(plan generate.Plan, field string) (generate.Note, bool) {
	for _, note := range plan.Notes {
		if strings.HasSuffix(note.Where, "."+field) {
			return note, true
		}
	}
	return generate.Note{}, false
}

// A column this build will leave empty is named in the plan.
//
// 🔴 This is the silence found on 2026-08-10 while planning the second pillar.
// A field described only by a type produced nothing, and the plan reported an
// expression in the same position while saying nothing at all about this one.
// The two look alike in a recipe and only one of them spoke, which is why the
// gap survived from the first day of the verb until somebody asked what the
// corruption engine would aim at.
func TestAColumnThisBuildWillLeaveEmptyIsNamed(t *testing.T) {
	values := knownValues{produces: []string{"integer"}}
	plan, err := generate.Prepare(oneEntity(
		recipe.Field{Name: "price", Type: "money"},
	), values)
	if err != nil {
		t.Fatal(err)
	}
	note, found := noteAbout(plan, "price")
	if !found {
		t.Fatal("a column that will arrive empty is not mentioned anywhere in the plan")
	}
	if !strings.Contains(note.What, "money") {
		t.Errorf("the note does not name the type that cannot be filled: %q", note.What)
	}
	if !strings.Contains(note.What, "empty") {
		t.Errorf("the note does not say the column will be empty: %q", note.What)
	}
}

// A type this build does fill is not reported as a gap, or the note would cry
// wolf on every recipe that uses a number.
func TestATypeThisBuildFillsIsNotReportedAsAGap(t *testing.T) {
	values := knownValues{produces: []string{"integer", "decimal"}}
	plan, err := generate.Prepare(oneEntity(
		recipe.Field{Name: "quantity", Type: "integer"},
	), values)
	if err != nil {
		t.Fatal(err)
	}
	if note, found := noteAbout(plan, "quantity"); found {
		t.Errorf("a column that will be filled was reported as empty: %q", note.What)
	}
}

// An expression is still reported, because this change added a case beside it
// rather than replacing it.
func TestAnExpressionIsStillReported(t *testing.T) {
	values := knownValues{produces: []string{"integer"}}
	plan, err := generate.Prepare(oneEntity(
		recipe.Field{Name: "label", Expression: "${a} ${b}"},
	), values)
	if err != nil {
		t.Fatal(err)
	}
	if _, found := noteAbout(plan, "label"); !found {
		t.Fatal("an expression is no longer reported as unevaluated")
	}
}

// A generator this build lacks stops the run rather than producing a note. The
// two answers mean different things: a recipe naming a generator that does not
// exist cannot be run at all, while a column left empty still produces a
// dataset.
func TestAGeneratorThisBuildLacksStopsTheRun(t *testing.T) {
	values := knownValues{generators: []string{"city"}, produces: []string{"integer"}}
	_, err := generate.Prepare(oneEntity(
		recipe.Field{Name: "name", Generator: "no_such_generator"},
	), values)
	if err == nil {
		t.Fatal("a recipe naming an unknown generator was planned rather than refused")
	}
	if !strings.Contains(err.Error(), "no_such_generator") {
		t.Errorf("the refusal does not name the generator: %v", err)
	}
}
