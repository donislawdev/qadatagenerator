package validate_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
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

	report, err := validate.Run(decoder, validate.Source{Name: "recipe.yaml"}, everyNameKnown{}, nil)
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

// A generator this build carries and cannot fill in the recipe's locale is a
// problem rather than a silence, because generate refuses such a recipe.
//
// 🔴 This verb is the gate people put in front of a pipeline, and it used to
// call such a recipe usable. The run then failed on the first row, which is the
// two verbs disagreeing about one file. The second half of this test matters as
// much as the first: the same recipe in a locale this build carries has to stay
// usable, or the gate would start refusing what works.
func TestAGeneratorTheLocaleCannotFillIsAProblem(t *testing.T) {
	forLocale := func(locale string) validate.Decoded {
		return validate.Decoded{Recipe: recipe.Recipe{Locale: locale,
			Entities: []recipe.Entity{{Name: "place",
				Fields: []recipe.Field{{Name: "town", Generator: "city"}}}}}}
	}

	report, err := validate.Run(func([]byte) validate.Decoded { return forLocale("en-US") },
		validate.Source{Name: "recipe.yaml"}, noCityHere{}, nil)
	if err == nil {
		t.Fatal("a recipe generate refuses was called usable")
	}
	if kind := fault.KindOf(err); kind != fault.Config {
		t.Errorf("an unusable recipe is %v rather than a configuration failure", kind)
	}
	if len(report.Problems) != 1 {
		t.Fatalf("the report has to name the field, it carries %+v", report.Problems)
	}
	if report.Problems[0].Where != "entities.place.fields.town" {
		t.Errorf("the problem points at %q", report.Problems[0].Where)
	}
	if !strings.Contains(report.Problems[0].What, "en-US") {
		t.Errorf("the problem does not name the locale: %q", report.Problems[0].What)
	}

	usable, err := validate.Run(func([]byte) validate.Decoded { return forLocale("pl-PL") },
		validate.Source{Name: "recipe.yaml"}, noCityHere{}, nil)
	if err != nil {
		t.Fatalf("the same recipe was refused in a locale that carries the list: %v", err)
	}
	if !usable.Usable() {
		t.Errorf("a recipe that runs was reported unusable: %+v", usable.Problems)
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

	report, err := validate.Run(decoder, validate.Source{Name: "recipe.yaml"}, everyNameKnown{}, nil)
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
	_, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, everyNameKnown{}, nil)
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
	report, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, everyNameKnown{}, nil)
	if err != nil {
		t.Fatalf("unexpected failure: %v", err)
	}
	if len(report.Unchecked) != 1 {
		t.Fatal("the report dropped what it did not check")
	}
}

// everyNameKnown is a registry that has every name. The point of these tests is
// the shape of the report, and a registry refusing names would make each of them
// fail for a second reason.
type everyNameKnown struct{}

func (everyNameKnown) Known(string) bool { return true }
func (everyNameKnown) Names() []string   { return []string{"first_name"} }

// CanProduce answers yes for every pair, which keeps the tests using this stub
// about the names they were written for. The locale is answered for by
// noCityHere below.
func (everyNameKnown) CanProduce(string, string) (bool, string) { return true, "" }

// noCityHere carries the generator and not the locale, which is the pair that
// used to pass every check and fail on the first row.
type noCityHere struct{ everyNameKnown }

func (noCityHere) CanProduce(generator, locale string) (bool, string) {
	if generator == "city" && locale == "en-US" {
		return false, "this build has no cities for en-US"
	}
	return true, ""
}

// someLayers is a catalog carrying damage in two layers, one of which has
// nothing implemented behind it yet.
type someLayers struct{}

func (someLayers) Layers() []string { return []string{"L2", "L3"} }

func (someLayers) Unimplemented(layer string) int {
	if layer == "L3" {
		return 7
	}
	return 0
}

// A policy naming a layer this build has no damage in orders a run that
// produces nothing while looking as though it produced everything. Validate has
// to say so, because corrupt would.
func TestAPolicyNamingALayerWithNoDamageIsRefused(t *testing.T) {
	decoder := func([]byte) validate.Decoded {
		return validate.Decoded{Recipe: recipe.Recipe{
			Policy: recipe.Policy{Layers: []string{"L2", "SEC"}},
		}}
	}
	report, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, everyNameKnown{}, someLayers{})
	if err == nil {
		t.Fatal("a policy naming a layer with no damage was accepted")
	}
	joined := problemText(report)
	if !strings.Contains(joined, `no damage in "SEC"`) {
		t.Errorf("the refusal does not name the layer: %s", joined)
	}
	if !strings.Contains(joined, "L2, L3") {
		t.Errorf("the refusal does not list what this build does carry: %s", joined)
	}
	if strings.Contains(joined, `"L2"`) {
		t.Errorf("a layer this build carries was refused as well: %s", joined)
	}
}

// A layer waiting for its implementations is the ordinary state of a catalog
// that grows faster than the engine. It must not be a refusal, and it must not
// be a surprise either.
func TestAPolicyIsToldWhatThisBuildWillNotDoYet(t *testing.T) {
	decoder := func([]byte) validate.Decoded {
		return validate.Decoded{Recipe: recipe.Recipe{
			Policy: recipe.Policy{Layers: []string{"L2", "L3"}},
		}}
	}
	report, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, everyNameKnown{}, someLayers{})
	if err != nil {
		t.Fatalf("a usable policy was refused: %v", err)
	}

	var said []string
	for _, note := range report.Unchecked {
		said = append(said, note.Where+" "+note.What+" "+note.Reason)
	}
	joined := strings.Join(said, "\n")
	if !strings.Contains(joined, "7 kinds of damage in L3") {
		t.Errorf("nothing says what this build cannot do yet: %s", joined)
	}
	if strings.Contains(joined, "in L2") {
		t.Errorf("a layer this build implements was reported as waiting: %s", joined)
	}
}

// A recipe with no policy asks for correct data only, which is the whole of the
// first pillar and a perfectly ordinary thing to want.
func TestARecipeWithNoPolicyIsNotToldAnythingAboutDamage(t *testing.T) {
	decoder := func([]byte) validate.Decoded { return validate.Decoded{} }
	report, err := validate.Run(decoder, validate.Source{Name: "r.yaml"}, everyNameKnown{}, someLayers{})
	if err != nil {
		t.Fatalf("a recipe with no policy was refused: %v", err)
	}
	for _, note := range report.Unchecked {
		if strings.Contains(note.Where, "policy") {
			t.Errorf("a recipe asking for no damage was told about damage: %+v", note)
		}
	}
}

func problemText(report validate.Report) string {
	var said []string
	for _, problem := range report.Problems {
		said = append(said, problem.Where+" "+problem.What+" "+problem.Allowed)
	}
	return strings.Join(said, "\n")
}

// A problem raised here, above the decoder, is pointed at the same line the
// decoder would have pointed at.
//
// 🔴 This closes a spelling written down three times. The decoder names a place
// while it walks the file, the walk that finds lines names it while it walks the
// syntax tree, and the checks in this file build the same name from the model.
// The first two are held together by a guard in the decoder's own package. This
// is the third, and without it a recipe whose only fault is an unknown generator
// would show a problem with no line beside problems that have one, for a reason
// nobody looking at the screen could work out.
//
// It is a real join rather than a spelling comparison: the line asserted here is
// the line the field is written on in the source below.
func TestAProblemRaisedAboveTheDecoderCarriesTheLineOfItsPlace(t *testing.T) {
	// version on line 1, entities on 2, customer on 3, count on 4,
	// fields on 5, and the field with the unknown generator on line 6.
	source := "version: \"0.1\"\nentities:\n  customer:\n    count: 1\n    fields:\n" +
		"      email: { generator: nosuchgenerator }\n"

	// The real decoder, reached the same way the command line reaches it: by
	// converting its result rather than by rewriting the fields. That
	// conversion is what fails the day either side gains a field, which is how
	// the line arrived on this surface in the first place.
	decode := func(data []byte) validate.Decoded { return validate.Decoded(recipeyaml.Decode(data)) }
	report, err := validate.Run(decode,
		validate.Source{Name: "recipe.yaml", Data: []byte(source)}, onlyKnown{}, nil)
	if err == nil {
		t.Fatal("a recipe asking for a generator this build has no name for was called usable")
	}

	found := false
	for _, problem := range report.Problems {
		if problem.Where != "entities.customer.fields.email" {
			continue
		}
		found = true
		if problem.Line != 6 {
			t.Errorf("the unknown generator was reported at line %d and the field is written "+
				"on line 6. The three places that spell this path have come apart, and the "+
				"screen showing it points somewhere else", problem.Line)
		}
	}
	if !found {
		t.Fatal("nothing was reported about the field with the unknown generator, so this " +
			"guard is checking something other than what it says")
	}
}

// onlyKnown is a registry that knows one name, which is not the one the recipe
// above asks for.
type onlyKnown struct{}

func (onlyKnown) Known(name string) bool { return name == "first_name" }
func (onlyKnown) Names() []string        { return []string{"first_name"} }
func (onlyKnown) CanProduce(string, string) (bool, string) {
	return true, ""
}
