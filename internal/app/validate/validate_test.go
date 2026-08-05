package validate_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

func TestAUsableRecipeReportsWhatWasUnderstood(t *testing.T) {
	decoder := func([]byte) validate.Decoded {
		return validate.Decoded{Recipe: recipe.Recipe{Entities: []recipe.Entity{
			{Name: "customer", Fields: []recipe.Field{{Name: "a"}, {Name: "b"}}},
			{Name: "order", Fields: []recipe.Field{{Name: "c"}}},
		}}}
	}

	report, err := validate.Run(decoder, validate.Source{Name: "recipe.yaml"}, wszystkieZnane{})
	if err != nil {
		t.Fatalf("a usable recipe is not a failure: %v", err)
	}
	if !report.Usable() {
		t.Fatal("expected the recipe to be usable")
	}
	if report.Entities != 2 || report.Fields != 3 {
		t.Fatalf("expected two entities and three fields, got %d and %d", report.Entities, report.Fields)
	}
}

// The kind travels with the failure, so the surface never has to work out
// which ending this was. A recipe that cannot be used says something about the
// recipe and nothing about the tool.
func TestAnUnusableRecipeIsAConfigurationFailure(t *testing.T) {
	decoder := func([]byte) validate.Decoded {
		return validate.Decoded{Problems: []recipe.Problem{
			{Where: "version", What: "missing"},
			{Where: "entities", What: "missing"},
		}}
	}

	report, err := validate.Run(decoder, validate.Source{Name: "recipe.yaml"}, wszystkieZnane{})
	if err == nil {
		t.Fatal("expected a failure")
	}
	if got := fault.KindOf(err); got != fault.Config {
		t.Fatalf("expected a configuration failure, got %v", got)
	}
	if len(report.Problems) != 2 {
		t.Fatalf("the report has to carry the detail, got %+v", report.Problems)
	}
	const want = "2 problems in recipe.yaml"
	if err.Error() != want {
		t.Fatalf("expected %q, got %q", want, err.Error())
	}
}

func TestOneProblemIsCountedInTheSingular(t *testing.T) {
	decoder := func([]byte) validate.Decoded {
		return validate.Decoded{Problems: []recipe.Problem{{Where: "version", What: "missing"}}}
	}
	_, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, wszystkieZnane{})
	if err == nil || err.Error() != "1 problem in r.yaml" {
		t.Fatalf("unexpected message: %v", err)
	}
}

// A clean report that hides its own blind spots invites trust it has not
// earned, so what was not checked survives a successful run.
func TestWhatWasNotCheckedSurvivesASuccessfulRun(t *testing.T) {
	decoder := func([]byte) validate.Decoded {
		return validate.Decoded{
			Recipe:    recipe.Recipe{Entities: []recipe.Entity{{Name: "a"}}},
			Unchecked: []recipe.Unchecked{{Where: "a.b", What: "a setting", Reason: "no generators yet"}},
		}
	}
	report, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, wszystkieZnane{})
	if err != nil {
		t.Fatalf("unexpected failure: %v", err)
	}
	if len(report.Unchecked) != 1 {
		t.Fatal("the report dropped what it did not check")
	}
}

// wszystkieZnane is a registry that has every name. The point of these tests is
// the shape of the report, and a registry refusing names would make each of them
// fail for a second reason.
type wszystkieZnane struct{}

func (wszystkieZnane) Known(string) bool { return true }
func (wszystkieZnane) Names() []string   { return []string{"first_name"} }
