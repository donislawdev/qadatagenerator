package generate_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// roleValues is knownValues with roles, so that a test can build agreements the
// real registry does not have.
type roleValues struct {
	knownValues
	plays map[string]string
	reads map[string][]string
}

func (v roleValues) RoleOf(generator string) (string, bool) {
	role, plays := v.plays[generator]
	return role, plays
}

func (v roleValues) RolesRead(generator string) []string { return v.reads[generator] }

// circled is two generators that each read the role the other plays, which no
// generator of this build does today and the next one might.
func circled() roleValues {
	return roleValues{
		knownValues: knownValues{generators: []string{"left", "right", "free"}},
		plays:       map[string]string{"left": "east", "right": "west"},
		reads:       map[string][]string{"left": {"west"}, "right": {"east"}},
	}
}

// Rules that wait on each other in a circle are refused by the plan, before any
// row, and the refusal names the fields and says what to do (DEC-283).
//
// The circle here is implied rather than written, because two columns reading
// each other's role is enough to close one, and a recipe author would not see
// it in their own text.
func TestRulesThatWaitOnEachOtherInACircleAreRefused(t *testing.T) {
	rec := oneEntity(
		recipe.Field{Name: "a", Generator: "left"},
		recipe.Field{Name: "b", Generator: "right"},
		recipe.Field{Name: "c", Generator: "free"},
	)
	_, err := generate.Prepare(rec, circled(), generate.NeverStops{})
	if err == nil {
		t.Fatal("two columns reading each other's role were accepted")
	}
	if fault.KindOf(err) != fault.Config {
		t.Errorf("the refusal is of kind %v rather than a recipe that cannot be read as one", fault.KindOf(err))
	}
	for _, said := range []string{"a, b", "circle"} {
		if !strings.Contains(err.Error(), said) {
			t.Errorf("the refusal does not say %q: %v", said, err)
		}
	}
	if strings.Contains(err.Error(), "b, c") {
		t.Errorf("the refusal names a column outside the circle: %v", err)
	}
	if fault.InsteadOf(err) == "" {
		t.Error("the refusal says what is wrong and not what to do about it")
	}
}

// A column bound by a rule and read by another is a chain rather than a circle,
// and the plan takes it, whichever order the recipe wrote the rules in.
func TestAChainOfRulesIsTakenInEitherOrder(t *testing.T) {
	chain := roleValues{
		knownValues: knownValues{generators: []string{"root", "middle", "end"}},
		plays:       map[string]string{"root": "first", "middle": "second"},
		reads:       map[string][]string{"middle": {"first"}, "end": {"first", "second"}},
	}
	fields := []recipe.Field{
		{Name: "end", Generator: "end"},
		{Name: "middle", Generator: "middle"},
		{Name: "root", Generator: "root"},
	}
	for _, rules := range [][]recipe.Rule{
		nil,
		{{Field: "end", Kind: recipe.Matches, With: []string{"root", "middle"}},
			{Field: "middle", Kind: recipe.Matches, With: []string{"root"}}},
		{{Field: "middle", Kind: recipe.Matches, With: []string{"root"}},
			{Field: "end", Kind: recipe.Matches, With: []string{"root", "middle"}}},
	} {
		rec := oneEntity(fields...)
		rec.Entities[0].Consistency = rules
		if _, err := generate.Prepare(rec, chain, generate.NeverStops{}); err != nil {
			t.Errorf("a chain written as %v was refused: %v", rules, err)
		}
	}
}
