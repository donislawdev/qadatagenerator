package recipeyaml_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// The shape the format document describes has to be readable, or the document
// is describing something this build does not accept.
func TestTheDocumentedShapeIsAccepted(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "documented.yaml"))
	if err != nil {
		t.Fatalf("cannot read the fixture: %v", err)
	}
	result := recipeyaml.Decode(data)
	if !result.Valid() {
		for _, found := range result.Problems {
			t.Errorf("%s: %s", found.Where, found.What)
		}
		t.FailNow()
	}

	if got := len(result.Recipe.Entities); got != 3 {
		t.Fatalf("expected three entities, got %d", got)
	}
	if result.Recipe.Seed != "release-2026-08" {
		t.Errorf("the seed did not survive: %q", result.Recipe.Seed)
	}
}

// Column order follows the recipe, so a field list that comes back sorted
// would quietly reorder somebody's output.
func TestFieldOrderIsTheOrderTheUserWrote(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  customer:
    fields:
      zeta:  { generator: a }
      alpha: { generator: b }
      mid:   { generator: c }
`)
	entity := result.Recipe.Entities[0]
	want := []string{"zeta", "alpha", "mid"}
	got := entity.FieldNames()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

// A person fixing a recipe wants the list, not one line at a time.
func TestEveryProblemIsReportedAtOnce(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
nonsense: 1
entities:
  customer:
    count: -1
    fields:
      a: {}
      b: { generator: x, expression: "y" }
`))
	if len(result.Problems) < 4 {
		for _, found := range result.Problems {
			t.Logf("%s: %s", found.Where, found.What)
		}
		t.Fatalf("expected at least four problems in one pass, got %d", len(result.Problems))
	}
}

func TestProblems(t *testing.T) {
	cases := []struct {
		name  string
		yaml  string
		where string
		what  string
	}{
		{
			name:  "a number where the schema version belongs",
			yaml:  "version: 1.0\nentities:\n  a:\n    fields:\n      x: { generator: y }\n",
			where: "version",
			what:  "has to be text",
		},
		{
			name:  "a seed YAML read as octal",
			yaml:  "version: \"0.1\"\nseed: 010\nentities:\n  a:\n    fields:\n      x: { generator: y }\n",
			where: "seed",
			// The message shows the 8 the user never wrote, which is the
			// whole reason a number is refused here.
			what: "read 8 as a number",
		},
		{
			name:  "a schema version this build does not speak",
			yaml:  "version: \"9.9\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\n",
			where: "version",
			what:  "does not speak recipe schema 9.9",
		},
		{
			name:  "no schema version at all",
			yaml:  "entities:\n  a:\n    fields:\n      x: { generator: y }\n",
			where: "version",
			what:  "does not say which schema version",
		},
		{
			name:  "an unknown key at the top",
			yaml:  "version: \"0.1\"\nentitites:\n  a: {}\n",
			where: "entitites",
			what:  "no such key",
		},
		{
			name:  "an unknown key in an entity",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    countt: 5\n    fields:\n      x: { generator: y }\n",
			where: "entities.a.countt",
			what:  "no such key",
		},
		{
			name:  "an unknown key in the output",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\noutput:\n  compression: none\n",
			where: "output.compression",
			what:  "no such key",
		},
		{
			name:  "a locale with no providers behind it",
			yaml:  "version: \"0.1\"\nlocale: de-DE\nentities:\n  a:\n    fields:\n      x: { generator: y }\n",
			where: "locale",
			what:  "does not know de-DE",
		},
		{
			name:  "a format this build cannot write",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\noutput:\n  format: xlsx\n",
			where: "output.format",
			what:  "does not know xlsx",
		},
		{
			name:  "a field with nowhere for its value to come from",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { unique: true }\n",
			where: "entities.a.fields.x",
			what:  "does not say where its value comes from",
		},
		{
			name:  "a field with two sources",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: g, expression: \"e\" }\n",
			where: "entities.a.fields.x",
			what:  "says twice where its value comes from",
		},
		{
			name:  "a share without a percent sign",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: g, empty: \"15\" }\n",
			where: "entities.a.fields.x.empty",
			what:  "has to end with a percent sign",
		},
		{
			name:  "a share outside the range",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: g, empty: 150% }\n",
			where: "entities.a.fields.x.empty",
			what:  "outside nought to a hundred",
		},
		{
			name:  "a count below zero",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    count: -1\n    fields:\n      x: { generator: y }\n",
			where: "entities.a.count",
			what:  "below zero",
		},
		{
			name:  "a rule tying a field that does not exist",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\n    consistency:\n      - x matches: ghost\n",
			where: "entities.a.consistency[0]",
			what:  "no field ghost",
		},
		{
			name:  "a way of following a field that does not exist",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\n    consistency:\n      - x wibbles: x\n",
			where: "entities.a.consistency[0]",
			what:  "no such way of following",
		},
		{
			name:  "a relation to an entity that was never declared",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    belongs_to: { ghost: 0-5 }\n    fields:\n      x: { generator: y }\n",
			where: "entities.a.belongs_to.ghost",
			what:  "no such entity",
		},
		{
			name:  "a range that runs backwards",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\n  b:\n    belongs_to: { a: 5-1 }\n    fields:\n      y: { generator: y }\n",
			where: "entities.b.belongs_to.a",
			what:  "runs backwards",
		},
		{
			name:  "an expression reaching a parent that is not declared",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { expression: \"${ghost.name}\" }\n",
			where: "entities.a.fields.x.expression",
			what:  "does not belong to ghost",
		},
		{
			name:  "an expression reaching a field the parent does not have",
			yaml:  "version: \"0.1\"\nentities:\n  p:\n    fields:\n      real: { generator: y }\n  c:\n    belongs_to: { p: 0-1 }\n    fields:\n      x: { expression: \"${p.ghost}\" }\n",
			where: "entities.c.fields.x.expression",
			what:  "p has no field ghost",
		},
		{
			name:  "a file that is not YAML at all",
			yaml:  "version: \"0.1\"\n  bad indentation\n",
			where: "",
			what:  "not valid YAML",
		},
		{
			name:  "an empty file",
			yaml:  "",
			where: "",
			what:  "the recipe is empty",
		},
		{
			name:  "a recipe with no entities",
			yaml:  "version: \"0.1\"\n",
			where: "entities",
			what:  "declares no entities",
		},
		{
			name:  "a primary key naming a field that does not exist",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    primary_key: ghost\n    fields:\n      x: { generator: y }\n",
			where: "entities.a.primary_key",
			what:  "has no field ghost",
		},
		{
			name:  "a primary key naming the same field twice",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    primary_key: [x, x]\n    fields:\n      x: { generator: y }\n",
			where: "entities.a.primary_key",
			what:  "names x twice",
		},
		{
			name:  "a field that is required and also asked to be empty",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y, required: true, empty: 15% }\n",
			where: "entities.a.fields.x",
			what:  "required and is also asked to be empty",
		},
		{
			name:  "a maximum length of nothing",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y, max_length: 0 }\n",
			where: "entities.a.fields.x.max_length",
			what:  "not above zero",
		},
		{
			name:  "a unit for a length that was never declared",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y, counted_in: bytes }\n",
			where: "entities.a.fields.x.counted_in",
			what:  "does not declare",
		},
		{
			name:  "a unit this build does not know",
			yaml:  "version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y, max_length: 10, counted_in: furlongs }\n",
			where: "entities.a.fields.x.counted_in",
			what:  "does not know furlongs",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			result := recipeyaml.Decode([]byte(test.yaml))
			for _, found := range result.Problems {
				if found.Where == test.where && strings.Contains(found.What, test.what) {
					return
				}
			}
			for _, found := range result.Problems {
				t.Logf("got %q: %s", found.Where, found.What)
			}
			t.Fatalf("no problem at %q saying %q", test.where, test.what)
		})
	}
}

// The position in a message has to name the rule the user wrote. A rule that
// could not be read leaves no entry behind it, so counting entries afterwards
// would send somebody to the wrong line.
func TestARuleThatCouldNotBeReadDoesNotShiftThePositions(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  a:
    fields:
      x: { generator: y }
    consistency:
      - x wibbles: x
      - x matches: ghost
`))
	var found bool
	for _, problem := range result.Problems {
		if problem.Where == "entities.a.consistency[1]" && strings.Contains(problem.What, "ghost") {
			found = true
		}
		if problem.Where == "entities.a.consistency[0]" && strings.Contains(problem.What, "ghost") {
			t.Fatal("the second rule was reported as the first")
		}
	}
	if !found {
		t.Fatal("the problem in the second rule was not reported against it")
	}
}

// One bad line must not hide a second problem on the same line, because that
// turns one fix into two rounds.
func TestABadRangeDoesNotHideAMissingParent(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  a:
    belongs_to: { ghost: 5-1 }
    fields:
      x: { generator: y }
`))
	var backwards, missing bool
	for _, problem := range result.Problems {
		backwards = backwards || strings.Contains(problem.What, "runs backwards")
		missing = missing || strings.Contains(problem.What, "no such entity")
	}
	if !backwards || !missing {
		t.Fatalf("expected both problems, got backwards=%v missing=%v", backwards, missing)
	}
}

// A generator setting cannot be checked before the generators exist. Carrying
// it in silence would let a misspelled unique turn itself off without a word.
func TestGeneratorSettingsAreCarriedAndReportedAsUnchecked(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  a:
    fields:
      x: { generator: birth_date, age: 18-75, uniqe: true }
`)
	field := result.Recipe.Entities[0].Fields[0]
	if len(field.Parameters) != 2 {
		t.Fatalf("expected both settings to be carried, got %+v", field.Parameters)
	}
	var sawMisspelling bool
	for _, skipped := range result.Unchecked {
		if strings.Contains(skipped.Where, "uniqe") {
			sawMisspelling = true
		}
	}
	if !sawMisspelling {
		t.Fatal("a misspelled generic key has to be visible, not swallowed as a setting")
	}
}

// A reference that is not a field may be a value the tool supplies. Saying so
// beats calling it an error and beats saying nothing.
func TestAnUnresolvedReferenceIsReportedAsUnchecked(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  a:
    fields:
      x: { expression: "${random_domain}" }
`)
	if len(result.Unchecked) != 1 || !strings.Contains(result.Unchecked[0].What, "random_domain") {
		t.Fatalf("expected one note about the reference, got %+v", result.Unchecked)
	}
}

func TestTheSchemaViewShowsLessThanTheModel(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  p:
    fields:
      k: { type: uuid, unique: true }
  c:
    belongs_to: { p: 0-5 }
    fields:
      v: { generator: g, age: 18-75 }
    consistency:
      - v matches: v
`)
	var view recipe.SchemaView = result.Recipe
	if names := view.EntityNames(); strings.Join(names, ",") != "p,c" {
		t.Fatalf("expected the declared order, got %v", names)
	}
	fields := view.FieldsOf("c")
	if len(fields) != 1 || fields[0].Name != "v" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	parents := view.ParentsOf("c")
	if len(parents) != 1 || parents[0].Entity != "p" || parents[0].Max != 5 {
		t.Fatalf("unexpected parents: %+v", parents)
	}
	if rules := view.RulesOf("c"); len(rules) != 1 || rules[0].Kind != recipe.Matches {
		t.Fatalf("unexpected rules: %+v", rules)
	}
	if view.FieldsOf("nobody") != nil {
		t.Error("an entity that does not exist has no fields rather than a guess")
	}
}

// The three constraints a mutator needs in order to break them have to reach
// the engine, and the engine reads the schema view rather than the recipe.
func TestTheConstraintsAMutatorBreaksReachTheSchemaView(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  customer:
    primary_key: [id, branch]
    fields:
      id:     { type: uuid }
      branch: { type: integer }
      name:   { generator: n, required: true, max_length: 30 }
      note:   { generator: n, max_length: 64, counted_in: bytes }
      status: { type: enum, values: [new, active, closed] }
`)
	var view recipe.SchemaView = result.Recipe

	if key := view.PrimaryKeyOf("customer"); strings.Join(key, ",") != "id,branch" {
		t.Errorf("the key has to keep the order it was written in, got %v", key)
	}
	if view.PrimaryKeyOf("nobody") != nil {
		t.Error("an entity that does not exist has no key rather than a guess")
	}

	byName := map[string]recipe.FieldInfo{}
	for _, field := range view.FieldsOf("customer") {
		byName[field.Name] = field
	}

	name := byName["name"]
	if !name.Required || name.MaxLength != 30 {
		t.Errorf("the declared constraints did not survive: %+v", name)
	}
	// Left unwritten in the recipe, so the default has to appear rather than
	// an empty unit the engine would have to interpret for itself.
	if name.LengthUnit != recipe.Characters {
		t.Errorf("expected the default unit, got %q", name.LengthUnit)
	}
	if unit := byName["note"].LengthUnit; unit != recipe.Bytes {
		t.Errorf("expected bytes, got %q", unit)
	}
	if values := byName["status"].Values; strings.Join(values, ",") != "new,active,closed" {
		t.Errorf("the declared value set did not reach the view: %v", values)
	}

	// A field with no limit has to be distinguishable from one limited to
	// zero, which is why the flag exists beside the number.
	if byName["id"].HasMaxLength {
		t.Error("a field without a limit reports one")
	}
}

// A share of zero and a required field are different statements, and the
// contradiction check must not collapse them. This is the degenerate input
// that a naive check would reject.
func TestRequiredWithAnExplicitShareOfNothingIsLegal(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  a:
    fields:
      x: { generator: y, required: true, empty: 0% }
`))
	for _, found := range result.Problems {
		t.Errorf("expected no problem, got %s: %s", found.Where, found.What)
	}
	field := result.Recipe.Entities[0].Fields[0]
	if !field.Required || !field.HasEmptyShare || field.EmptyShare != 0 {
		t.Fatalf("both statements have to survive: %+v", field)
	}
}

func decode(t *testing.T, source string) recipeyaml.Result {
	t.Helper()
	result := recipeyaml.Decode([]byte(source))
	if !result.Valid() {
		for _, found := range result.Problems {
			t.Errorf("%s: %s", found.Where, found.What)
		}
		t.FailNow()
	}
	return result
}

// A recipe arrives from somebody else's repository, so in a pipeline it can
// come through a pull request. Reading a deliberately nested document costs
// time growing with the square of its size, and that cost belongs to the YAML
// library rather than to anything here, so the only lever is the size of the
// input before the parser sees it.
func TestARecipeTooLargeToReadIsRefusedBeforeParsing(t *testing.T) {
	oversized := make([]byte, recipeyaml.MaxSize+1)
	for i := range oversized {
		oversized[i] = '['
	}

	start := time.Now()
	result := recipeyaml.Decode(oversized)
	elapsed := time.Since(start)

	if result.Valid() {
		t.Fatal("a recipe past the limit was accepted")
	}
	if len(result.Problems) != 1 || !strings.Contains(result.Problems[0].What, "at most") {
		t.Fatalf("expected one problem naming the limit, got %+v", result.Problems)
	}
	// The refusal has to be cheap. Taking the slow path and then refusing
	// would leave the cost this exists to avoid.
	if elapsed > time.Second {
		t.Errorf("the refusal took %v, so the parser was handed the input after all", elapsed)
	}

	// One byte under the limit still goes to the parser, or the guard would be
	// refusing things nobody asked it to.
	fine := recipeyaml.Decode([]byte("version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\n"))
	if !fine.Valid() {
		t.Fatalf("an ordinary recipe was refused: %+v", fine.Problems)
	}
}
