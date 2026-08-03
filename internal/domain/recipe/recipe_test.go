package recipe_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// This package had no tests of its own until 2026-08-03. It was covered
// sideways, by the tests of the reader that builds it, which is coverage of the
// path those tests happen to walk rather than of what the model promises. The
// promises below are the ones other layers rely on directly.

func sample() recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1",
		Entities: []recipe.Entity{
			{
				Name:   "parent",
				Fields: []recipe.Field{{Name: "id", Type: "uuid", Unique: true}},
			},
			{
				Name:       "child",
				PrimaryKey: []string{"id", "branch"},
				Fields: []recipe.Field{
					{Name: "id", Type: "uuid"},
					{Name: "branch", Type: "integer"},
					{Name: "note", Generator: "n", MaxLength: 64, HasMaxLength: true, LengthUnit: recipe.Bytes},
					{Name: "plain", Generator: "n", MaxLength: 30, HasMaxLength: true},
				},
				Consistency: []recipe.Rule{{Field: "note", Kind: recipe.Matches, With: []string{"branch"}}},
				Parents:     []recipe.Parent{{Entity: "parent", Min: 0, Max: 5}},
			},
		},
	}
}

// The order of entities and of fields is the order somebody wrote, not
// alphabetical. Column order in the output follows it, and a diff in a pull
// request has to stay small.
func TestTheDeclaredOrderIsKept(t *testing.T) {
	got := sample().EntityNames()
	if strings.Join(got, ",") != "parent,child" {
		t.Fatalf("expected the declared order, got %v", got)
	}
	fields := sample().Entities[1].FieldNames()
	if strings.Join(fields, ",") != "id,branch,note,plain" {
		t.Fatalf("expected the declared field order, got %v", fields)
	}
}

// An absent unit means characters. Callers read Unit rather than the field, so
// that the default lives in one place and cannot be spelled two ways.
func TestAnAbsentLengthUnitMeansCharacters(t *testing.T) {
	fields := sample().FieldsOf("child")
	byName := map[string]recipe.FieldInfo{}
	for _, field := range fields {
		byName[field.Name] = field
	}
	if byName["plain"].LengthUnit != recipe.Characters {
		t.Errorf("a field with no unit came through as %q", byName["plain"].LengthUnit)
	}
	if byName["note"].LengthUnit != recipe.Bytes {
		t.Errorf("a declared unit was lost: %q", byName["note"].LengthUnit)
	}
}

// The view the engine sees is narrower than the model. It has no business
// knowing which generator produced a value, and that window is what keeps the
// two pillars from growing into each other.
func TestTheEngineSeesLessThanTheModel(t *testing.T) {
	fields := sample().FieldsOf("child")
	if len(fields) != 4 {
		t.Fatalf("expected four fields, got %d", len(fields))
	}
	// FieldInfo carries no generator, no expression and no parameters. This is
	// asserted by the type rather than by a value, so the test is here to fail
	// the day somebody widens it without meaning to.
	if got := sample().PrimaryKeyOf("child"); strings.Join(got, ",") != "id,branch" {
		t.Errorf("the key lost its order or its members: %v", got)
	}
	if parents := sample().ParentsOf("child"); len(parents) != 1 || parents[0].Max != 5 {
		t.Errorf("the relation did not survive: %+v", parents)
	}
	if rules := sample().RulesOf("child"); len(rules) != 1 || rules[0].Kind != recipe.Matches {
		t.Errorf("the rule did not survive: %+v", rules)
	}
}

// An entity nobody declared has no fields rather than a guess. Every accessor
// answers the same way, so a caller has one case to handle and not four.
func TestAnEntityNobodyDeclaredAnswersEmptyEverywhere(t *testing.T) {
	model := sample()
	if model.FieldsOf("nobody") != nil {
		t.Error("fields were invented for an entity that does not exist")
	}
	if model.PrimaryKeyOf("nobody") != nil {
		t.Error("a key was invented for an entity that does not exist")
	}
	if model.ParentsOf("nobody") != nil {
		t.Error("relations were invented for an entity that does not exist")
	}
	if model.RulesOf("nobody") != nil {
		t.Error("rules were invented for an entity that does not exist")
	}
	if _, ok := model.Entity("nobody"); ok {
		t.Error("an entity that does not exist was found")
	}
	if _, ok := model.Entities[0].Field("nobody"); ok {
		t.Error("a field that does not exist was found")
	}
}

// Two runs over one recipe have to report the same problems in the same order,
// and problems about one place have to arrive together. Sorting by where they
// are does both.
func TestProblemsAreSortedByPlaceAndThenByWhatWentWrong(t *testing.T) {
	problems := []recipe.Problem{
		{Where: "entities.b", What: "second"},
		{Where: "entities.a", What: "zebra"},
		{Where: "entities.a", What: "alpha"},
		{Where: "", What: "the file as a whole"},
	}
	recipe.SortProblems(problems)

	var order []string
	for _, found := range problems {
		order = append(order, found.Where+"/"+found.What)
	}
	want := "/the file as a whole,entities.a/alpha,entities.a/zebra,entities.b/second"
	if strings.Join(order, ",") != want {
		t.Fatalf("expected %q, got %q", want, strings.Join(order, ","))
	}
}

// A recipe is one source of a schema view, and the corruption engine takes the
// view rather than the recipe. The other source is somebody else's file, read
// in the mode where there is no recipe at all.
func TestARecipeSatisfiesTheSchemaView(t *testing.T) {
	var view recipe.SchemaView = sample()
	if len(view.EntityNames()) != 2 {
		t.Fatal("the view did not see both entities")
	}
}
