package format_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// held is a set of files that never touches a disk, so that a writer can be
// tested for its bytes rather than for its plumbing.
type held struct {
	written map[string]*strings.Builder
	refuse  string
}

func newHeld() *held { return &held{written: make(map[string]*strings.Builder)} }

func (h *held) Create(name string) (format.WriteCloser, error) {
	if name == h.refuse {
		return nil, errRefused{}
	}
	built := &strings.Builder{}
	h.written[name] = built
	return nopCloser{built}, nil
}

type errRefused struct{}

func (errRefused) Error() string { return "refused" }

type nopCloser struct{ *strings.Builder }

func (nopCloser) Close() error { return nil }

// sample is the recipe most of these tests run: two entities, a relation, a
// consistency rule and a field the identifier reads.
func sample(t *testing.T, perEntity bool, count int) recipe.Recipe {
	t.Helper()
	return recipe.Recipe{
		Version: "0.1", Seed: "test-seed", Locale: "pl-PL",
		Output: recipe.Output{Format: "csv", FilePerEntity: perEntity},
		Entities: []recipe.Entity{
			{
				Name: "customer", Count: 4, HasCount: true, PrimaryKey: []string{"pesel"},
				Fields: []recipe.Field{
					{Name: "first_name", Generator: "first_name"},
					{Name: "gender", Generator: "gender"},
					{Name: "pesel", Generator: "pesel"},
					{Name: "birth_date", Generator: "birth_date"},
				},
				Consistency: []recipe.Rule{
					{Field: "pesel", Kind: recipe.Matches, With: []string{"birth_date", "gender"}},
					{Field: "first_name", Kind: recipe.Matches, With: []string{"gender"}},
				},
			},
			{
				Name: "order", Count: count, HasCount: true,
				Parents: []recipe.Parent{{Entity: "customer", Min: 0, Max: 5}},
				Fields:  []recipe.Field{{Name: "placed_at", Generator: "birth_date"}},
			},
		},
	}
}

func run(t *testing.T, rec recipe.Recipe, make func(*held, recipe.Recipe) (generate.Writer, error)) *held {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source)
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	files := newHeld()
	writer, err := make(files, rec)
	if err != nil {
		t.Fatalf("the writer was refused: %v", err)
	}
	if err := generate.Run(plan, rec, source, writer, generate.NeverStops{}); err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	return files
}

func asCSV(files *held, rec recipe.Recipe) (generate.Writer, error) {
	return format.NewCSV(files, rec)
}

func asJSON(files *held, rec recipe.Recipe) (generate.Writer, error) {
	return format.NewJSON(files, rec)
}

// The whole of the first pillar in one test: a recipe goes in, files come out,
// and the child names the parent it belongs to.
func TestARecipeBecomesFilesWithTheRelationFollowable(t *testing.T) {
	rec := sample(t, true, 7)
	files := run(t, rec, asCSV)

	customers, made := files.written["customer.csv"]
	if !made {
		t.Fatalf("no file for the parent, only %v", names(files))
	}
	lines := strings.Split(strings.TrimSuffix(customers.String(), "\n"), "\n")
	if lines[0] != "first_name,gender,pesel,birth_date" {
		t.Errorf("the header is not the fields in the order the recipe wrote them: %q", lines[0])
	}
	if len(lines) != 5 {
		t.Errorf("four rows and a header make five lines, got %d", len(lines))
	}

	orders := files.written["order.csv"]
	orderLines := strings.Split(strings.TrimSuffix(orders.String(), "\n"), "\n")
	if orderLines[0] != "placed_at,customer" {
		t.Errorf("the child does not carry a column naming its parent: %q", orderLines[0])
	}
	if len(orderLines) != 8 {
		t.Errorf("seven rows and a header make eight lines, got %d", len(orderLines))
	}

	keys := map[string]bool{}
	for _, line := range lines[1:] {
		keys[strings.Split(line, ",")[2]] = true
	}
	for _, line := range orderLines[1:] {
		reference := strings.Split(line, ",")[1]
		if !keys[reference] {
			t.Errorf("an order points at %q, which is not the key of any customer", reference)
		}
	}
}

// 🔴 The promise the whole project rests on, asked of the first thing that
// produces a whole dataset. Two runs of one seed have to be the same bytes.
func TestTheSameSeedGivesTheSameBytes(t *testing.T) {
	rec := sample(t, true, 7)
	first := run(t, rec, asCSV).written["customer.csv"].String()
	second := run(t, rec, asCSV).written["customer.csv"].String()
	if first != second {
		t.Fatal("two runs of one seed produced different bytes, which is the one thing this tool " +
			"may never do")
	}
	if strings.Contains(first, "\r") {
		t.Error("a line ending came from the platform rather than from this writer")
	}
}

// Adding a field must not move the columns that were already there. It is the
// property that makes a diff of a regenerated dataset readable.
func TestAddingAFieldLeavesTheOtherColumnsWhereTheyWere(t *testing.T) {
	rec := sample(t, true, 7)
	before := column(t, run(t, rec, asCSV).written["customer.csv"].String(), 2)

	wider := sample(t, true, 7)
	wider.Entities[0].Fields = append(wider.Entities[0].Fields,
		recipe.Field{Name: "city", Generator: "city"})
	after := column(t, run(t, wider, asCSV).written["customer.csv"].String(), 2)

	if strings.Join(before, "|") != strings.Join(after, "|") {
		t.Fatalf("a new column moved an old one: %v became %v", before, after)
	}
}

// The count of a child wins and the cardinality bounds how it is spread.
func TestTheChildCountIsExactAndTheParentsShareIt(t *testing.T) {
	for _, count := range []int{0, 1, 7, 20} {
		rec := sample(t, true, count)
		source := values.New(rec.Locale, rec.Seed)
		plan, err := generate.Prepare(rec, source)
		if err != nil {
			t.Fatalf("count %d was refused: %v", count, err)
		}
		child := plan.Entities[1]
		if child.Rows != count {
			t.Errorf("asked for %d children and the plan makes %d", count, child.Rows)
		}
		perParent := map[int]int{}
		for _, parent := range child.ParentOf {
			perParent[parent]++
		}
		for parent, held := range perParent {
			if held > 5 {
				t.Errorf("parent %d holds %d children and the recipe allows five", parent, held)
			}
		}
	}
}

// A count the bounds cannot reach is refused with the arithmetic in it, rather
// than trimmed to what fits.
func TestACountTheBoundsCannotReachIsRefusedWithTheArithmetic(t *testing.T) {
	rec := sample(t, true, 21)
	_, err := generate.Prepare(rec, values.New(rec.Locale, rec.Seed))
	if err == nil {
		t.Fatal("twenty one children of four parents at five each was accepted")
	}
	for _, part := range []string{"21", "0 to 5", "between 0 and 20"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("the refusal does not show %q: %v", part, err)
		}
	}
}

// The identifier has to agree with the columns it was bound to, which is the
// difference between a dataset and a pile of correct values.
func TestTheIdentifierAgreesWithTheColumnsItWasBoundTo(t *testing.T) {
	rec := sample(t, true, 0)
	lines := strings.Split(strings.TrimSuffix(run(t, rec, asCSV).written["customer.csv"].String(), "\n"), "\n")

	for _, line := range lines[1:] {
		parts := strings.Split(line, ",")
		gender, national, born := parts[1], parts[2], parts[3]
		if len(national) != 11 {
			t.Fatalf("%q is not a personal number", national)
		}
		if want := born[2:4] + born[5:7] + born[8:10]; !bornAgrees(national, want) {
			t.Errorf("the number %s does not carry the date %s", national, born)
		}
		even := (national[9]-'0')%2 == 0
		if even != (gender == "female") {
			t.Errorf("the number %s disagrees with the gender column %q", national, gender)
		}
	}
}

// bornAgrees checks the six date digits, allowing for the month carrying the
// century.
func bornAgrees(national, yymmdd string) bool {
	if national[0:2] != yymmdd[0:2] || national[4:6] != yymmdd[4:6] {
		return false
	}
	month := int(national[2]-'0')*10 + int(national[3]-'0')
	return month%20 == int(yymmdd[2]-'0')*10+int(yymmdd[3]-'0')
}

// The nested form puts children inside the parent they belong to, which is the
// shape the format document promises.
func TestTheNestedFormPutsChildrenInsideTheirParent(t *testing.T) {
	rec := sample(t, false, 7)
	files := run(t, rec, asJSON)

	whole, made := files.written["dataset.json"]
	if !made {
		t.Fatalf("no single file was written, only %v", names(files))
	}
	text := whole.String()
	if !strings.Contains(text, `"customer": [`) {
		t.Error("the tree does not start at the entity that belongs to nobody")
	}
	if !strings.Contains(text, `"order": [`) {
		t.Error("the children are not inside the parent")
	}
	if strings.Contains(text, `"order": [`) && strings.Index(text, `"customer": [`) >
		strings.Index(text, `"order": [`) {
		t.Error("the child appears before the parent it belongs to")
	}
	if count := strings.Count(text, `"placed_at"`); count != 7 {
		t.Errorf("seven children were made and %d are in the tree", count)
	}
}

// A value carrying something that would end a field early has to survive the
// round trip, because the second pillar exists to put such values in on purpose.
func TestAValueCarryingTheDelimiterOrAQuoteSurvives(t *testing.T) {
	files := newHeld()
	rec := recipe.Recipe{Seed: "s", Locale: "en-US",
		Output: recipe.Output{FilePerEntity: true},
		Entities: []recipe.Entity{{Name: "t", Count: 1, HasCount: true,
			Fields: []recipe.Field{{Name: "a"}}}}}
	writer, err := format.NewCSV(files, rec)
	if err != nil {
		t.Fatalf("the writer was refused: %v", err)
	}
	planned := dataset.Table{Entity: "t", Columns: []string{"a"}, Rows: 1}
	if err := writer.Begin(planned); err != nil {
		t.Fatal(err)
	}
	if err := writer.Write(dataset.Record{Values: []string{`a,b "c"
d`}, Parent: dataset.NoParent}); err != nil {
		t.Fatal(err)
	}
	if err := writer.End(planned); err != nil {
		t.Fatal(err)
	}
	written := files.written["t.csv"].String()
	quoted := "\"a,b \"\"c\"\"\nd\""
	if !strings.Contains(written, quoted) {
		t.Errorf("the value was not quoted the way the format requires: %q", written)
	}
}

func column(t *testing.T, text string, index int) []string {
	t.Helper()
	var taken []string
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n")[1:] {
		parts := strings.Split(line, ",")
		if index < len(parts) {
			taken = append(taken, parts[index])
		}
	}
	return taken
}

func names(files *held) []string {
	var made []string
	for name := range files.written {
		made = append(made, name)
	}
	return made
}

// A field asked to be unique gets values no earlier row took. It is the first
// promise a recipe can make that costs the engine something to keep.
func TestAUniqueFieldNeverRepeatsAValue(t *testing.T) {
	rec := sample(t, true, 0)
	rec.Entities[0].Count = 40
	// 🔴 The field was an account number until 2026-08-04, and that made this
	// test pass against a build with no uniqueness at all: sixteen digits drawn
	// forty times never collide by accident, so nothing here needed the
	// mechanism to work. A name drawn from a word list of a few hundred entries
	// collides almost every time, which is what makes the guard a guard. Found
	// by mutation, which reported the removal of the check as unnoticed.
	rec.Entities[0].Fields = append(rec.Entities[0].Fields,
		recipe.Field{Name: "nickname", Generator: "first_name", Unique: true})

	lines := strings.Split(strings.TrimSuffix(
		run(t, rec, asCSV).written["customer.csv"].String(), "\n"), "\n")
	seen := map[string]bool{}
	for _, line := range lines[1:] {
		value := strings.Split(line, ",")[4]
		if seen[value] {
			t.Fatalf("%q came up twice in a column asked to be unique", value)
		}
		seen[value] = true
	}
	if len(seen) != 40 {
		t.Errorf("forty rows produced %d distinct values", len(seen))
	}
}

// A field with fewer possible values than rows cannot be unique, and the run
// says so with the arithmetic rather than looping for the impossible.
func TestAUniqueFieldWithTooFewValuesIsRefusedWithTheArithmetic(t *testing.T) {
	rec := sample(t, true, 0)
	rec.Entities[0].Count = 500
	rec.Entities[0].Fields = append(rec.Entities[0].Fields,
		recipe.Field{Name: "sex", Generator: "gender", Unique: true})

	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source)
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	writer, err := format.NewCSV(newHeld(), rec)
	if err != nil {
		t.Fatal(err)
	}
	err = generate.Run(plan, rec, source, writer, generate.NeverStops{})
	if err == nil {
		t.Fatal("five hundred rows of a field with two possible values were accepted")
	}
	for _, part := range []string{"sex", "unique", "500"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("the refusal does not carry %q: %v", part, err)
		}
	}
}

// The flat form of the object output puts one file beside each table, which is
// what the setting in the recipe asks for.
func TestTheFlatObjectFormWritesOneFilePerTable(t *testing.T) {
	rec := sample(t, true, 3)
	files := run(t, rec, asJSON)

	for _, name := range []string{"customer.json", "order.json"} {
		written, made := files.written[name]
		if !made {
			t.Fatalf("no %s, only %v", name, names(files))
		}
		text := written.String()
		if !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]\n") {
			t.Errorf("%s is not an array of records: %q", name, text)
		}
	}
	orders := files.written["order.json"].String()
	if !strings.Contains(orders, `"customer": "`) {
		t.Error("a child in the flat form does not name the parent it belongs to")
	}
	if strings.Contains(files.written["customer.json"].String(), `"order"`) {
		t.Error("the flat form nested a child inside its parent")
	}
}
