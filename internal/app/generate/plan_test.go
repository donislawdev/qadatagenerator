package generate_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
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
	// missing names the generators this stub cannot fill, by locale, which is
	// the pair a build can carry one half of.
	missing map[string]string
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

func (v knownValues) CanProduce(generator, locale string) (bool, string) {
	if v.missing[generator] == locale {
		return false, "this build has no " + generator + " for " + locale
	}
	return true, ""
}

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

// Two tables that fit one at a time do not have to fit together.
//
// 🔴 Measured on 2026-08-17 with the real binary: two entities of 9223372036854775807
// rows each ended with exit code nought and a summary line reading minus two
// rows written. The reader refuses a single count larger than this build can
// hold, so each of them was accepted on its own and the sum wrapped one entity
// later. The guard protected a field where the rule was about the arithmetic.
func TestRowsThatDoNotAddUpToACountableTotalAreRefused(t *testing.T) {
	const most = int(^uint(0) >> 1)
	values := knownValues{generators: []string{"first_name"}}
	field := recipe.Field{Name: "first_name", Generator: "first_name"}

	twice := func(count int) recipe.Recipe {
		return recipe.Recipe{
			Seed: "plan", Locale: "pl-PL",
			Entities: []recipe.Entity{
				{Name: "a", Count: count, HasCount: true, Fields: []recipe.Field{field}},
				{Name: "b", Count: count, HasCount: true, Fields: []recipe.Field{field}},
			},
		}
	}

	if _, err := generate.Prepare(twice(most), values); err == nil {
		t.Error("two tables of the largest countable size were accepted together")
	} else if !strings.Contains(err.Error(), "add up to more than") {
		t.Errorf("the refusal does not say the total is what went wrong: %v", err)
	}

	// Half of the range each is the case that wrapped all the way round to the
	// most negative number rather than to a small one, so it is the case where
	// a check on the sign alone would have looked right.
	if _, err := generate.Prepare(twice(most/2+1), values); err == nil {
		t.Error("two tables of half the countable range each were accepted together")
	}

	// The refusal is about the total and must not start refusing ordinary
	// recipes, which is the failure mode of a ceiling put in the wrong place.
	plan, err := generate.Prepare(twice(3), values)
	if err != nil {
		t.Fatalf("an ordinary recipe was refused: %v", err)
	}
	if plan.TotalRows() != 6 {
		t.Errorf("the plan counts %d rows for two tables of three", plan.TotalRows())
	}
}

// A generator this build carries and cannot fill in the recipe's locale is
// refused by the plan, which is what a dry run answers from.
//
// 🔴 The name and the locale are two halves of one question and only the first
// used to be asked. A recipe naming a generator whose word list this build does
// not carry for that locale passed the plan, so validate called it usable and a
// dry run reported how many rows it would write, while the real run failed on
// the first row. The plan is meant to be the first phase of the run rather than
// a separate estimator, and this is the line where it stopped being one.
func TestAGeneratorTheLocaleCannotFillIsRefusedByThePlan(t *testing.T) {
	values := knownValues{
		generators: []string{"city"},
		missing:    map[string]string{"city": "en-US"},
	}
	rec := recipe.Recipe{
		Seed: "plan", Locale: "en-US",
		Entities: []recipe.Entity{{Name: "place", Count: 2, HasCount: true,
			Fields: []recipe.Field{{Name: "town", Generator: "city"}}}},
	}

	_, err := generate.Prepare(rec, values)
	if err == nil {
		t.Fatal("the plan accepted a generator this build cannot fill in that locale")
	}
	if kind := fault.KindOf(err); kind != fault.Config {
		t.Errorf("a recipe asking for the impossible is %v rather than a configuration failure",
			kind)
	}
	if !strings.Contains(err.Error(), "entities.place.fields.town") {
		t.Errorf("the refusal does not name the field: %v", err)
	}
	if !strings.Contains(err.Error(), "en-US") {
		t.Errorf("the refusal does not name the locale, which is half the answer: %v", err)
	}

	// The same recipe in a locale this stub can fill has to go through, or the
	// refusal above would be about the generator rather than about the pair.
	rec.Locale = "pl-PL"
	if _, err := generate.Prepare(rec, values); err != nil {
		t.Errorf("the same generator was refused in a locale that carries it: %v", err)
	}
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

// A relation asking for more rows than this build can plan for is refused with
// the arithmetic, rather than reported as a defect of this tool.
//
// 🔴 Both of these ended with the four line message that begins internal
// defect and asks for a bug report, measured with the real binary on
// 2026-09-02. One reached the seed with a bound of minus nine million million
// million, because the width of the range was worked out in the arithmetic
// that had just overflowed. The other asked the runtime for a slice it refuses
// to make. In each case a recipe somebody wrote was answered by this tool
// blaming itself, which sends a person to report a fault that is not there.
//
// The limit itself is at MaxChildRows and carries the measurement behind it.
// What matters here is that a refusal arrives at all, that it carries the kind
// meaning the recipe reads correctly and cannot be carried out, and that it
// quotes the numbers it was given.
func TestARelationTooWideToPlanForIsRefusedWithTheArithmetic(t *testing.T) {
	const most = int(^uint(0) >> 1)
	values := knownValues{generators: []string{"first_name"}}

	for _, width := range []int{most, 4_000_000_000_000_000_000, 2_000_000_000} {
		_, err := generate.Prepare(related(3, 0, width), values)
		if err == nil {
			t.Fatalf("a range of nought to %d children each was accepted, so this build "+
				"was about to plan for more rows than it can hold", width)
		}
		if fault.KindOf(err) != fault.Data {
			t.Errorf("the refusal of nought to %d carries kind %v rather than the one "+
				"meaning the recipe reads correctly and cannot be carried out",
				width, fault.KindOf(err))
		}
		if !strings.Contains(err.Error(), strconv.Itoa(width)) {
			t.Errorf("the refusal does not quote the range it was given, so nobody can "+
				"tell which number to lower: %v", err)
		}
		if !strings.Contains(err.Error(), strconv.Itoa(generate.MaxChildRows)) {
			t.Errorf("the refusal does not say what this build can do, which is the third "+
				"of the four parts a refusal owes: %v", err)
		}
	}
}

// A parent table too large to hold a share for is refused even when the range
// asks for no children at all.
//
// 🔴 This is the case the check on the range cannot see. A share is kept for
// every parent while the draw goes on, so nought to nought children of nine
// million million parents allocates nothing for the children and one word for
// each parent. Multiplying by a range of nought gives nought, so a single
// check written as a product would have let this through.
func TestAParentTableTooLargeToShareOutBetweenIsRefused(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}

	_, err := generate.Prepare(related(generate.MaxChildRows+1, 0, 0), values)

	if err == nil {
		t.Fatal("a parent table larger than this build can share children out between " +
			"was accepted")
	}
	if fault.KindOf(err) != fault.Data {
		t.Errorf("the refusal carries kind %v rather than the one meaning the recipe "+
			"reads correctly and cannot be carried out", fault.KindOf(err))
	}
	if !strings.Contains(err.Error(), "parents") {
		t.Errorf("the refusal does not say that it is the number of parents that is too "+
			"large, so somebody would go looking at the range: %v", err)
	}
}

// The refusal is about what cannot be planned for and must leave ordinary
// relations alone, which is the failure mode of a ceiling put in the wrong
// place.
//
// 🔴 The boundary is pinned from above only, and that is deliberate rather
// than lazy. A recipe just under it plans for four thousand million rows and
// takes 32 GB to do it, so a test standing on the other side of the line would
// be a test that has to be able to allocate it.
func TestARelationThisBuildCanPlanForIsNotRefused(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}

	plan, err := generate.Prepare(related(3, 0, 4), values)

	if err != nil {
		t.Fatalf("an ordinary relation was refused: %v", err)
	}
	if plan.TotalRows() < 3 {
		t.Errorf("the plan counts %d rows for three parents and their children",
			plan.TotalRows())
	}
}

// related is a recipe of one parent table and one child table that belongs to
// it, with the cardinality handed in.
func related(parents, low, high int) recipe.Recipe {
	field := recipe.Field{Name: "first_name", Generator: "first_name"}
	return recipe.Recipe{
		Seed: "plan", Locale: "pl-PL",
		Entities: []recipe.Entity{
			{Name: "customer", Count: parents, HasCount: true,
				Fields: []recipe.Field{field}},
			{Name: "order", Fields: []recipe.Field{field},
				Parents: []recipe.Parent{{Entity: "customer", Min: low, Max: high}}},
		},
	}
}
