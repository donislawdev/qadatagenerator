package format_test

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
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
	return format.NewJSON(files, rec, generate.NeverStops{})
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
// produces a whole dataset. Runs of one seed have to be the same bytes.
//
// # Why several runs rather than two, which is the shape this had until
// 2026-08-19
//
// What this guards against is randomness, and one comparison is one sample. The
// mutation that breaks it walks a map instead of the declared fields, and Go
// randomises that walk, so two runs can come out alike by luck. Measured that
// day: two walks of a map of four keys come out in the same order 43.7% of the
// time, a record is one such map, and the sample writes four records, so two
// runs agree by chance in 3.71% of attempts. The full mutation run duly missed
// it once.
//
// A third comparison takes that to roughly one attempt in eight hundred, for one
// line. It does not make the proof certain, and nothing about a map walk can:
// what it does is move the guard from failing once a month to failing once a
// career.
func TestTheSameSeedGivesTheSameBytes(t *testing.T) {
	rec := sample(t, true, 7)
	first := run(t, rec, asCSV).written["customer.csv"].String()
	for again := 0; again < 2; again++ {
		if run(t, rec, asCSV).written["customer.csv"].String() != first {
			t.Fatal("two runs of one seed produced different bytes, which is the one thing this " +
				"tool may never do")
		}
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

// threeLevels is a recipe with a grandchild, which is the shallowest shape in
// which the tree can put a record under the wrong parent.
func threeLevels(perEntity bool) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1", Seed: "three-levels", Locale: "pl-PL",
		Output: recipe.Output{Format: "json", FilePerEntity: perEntity},
		Entities: []recipe.Entity{
			{Name: "customer", Count: 2, HasCount: true,
				Fields: []recipe.Field{{Name: "city", Generator: "city"}}},
			{Name: "order", Parents: []recipe.Parent{{Entity: "customer", Min: 2, Max: 2}},
				Fields: []recipe.Field{{Name: "city", Generator: "city"}}},
			{Name: "item", Parents: []recipe.Parent{{Entity: "order", Min: 2, Max: 2}},
				Fields: []recipe.Field{{Name: "city", Generator: "city"}}},
		},
	}
}

// treeItem is a grandchild as it stands in the tree, naming the parent it
// belongs to.
type treeItem struct {
	City  string `json:"city"`
	Order string `json:"order"`
}

// treeRoot is as much of the tree as this test reads.
type treeRoot struct {
	Customers []struct {
		Orders []struct {
			Items []treeItem `json:"item"`
		} `json:"order"`
	} `json:"customer"`
}

// The tree has to carry every grandchild under the parent it belongs to.
//
// # Why three levels and not two
//
// 🔴 A child names its parent by the position that parent holds in the whole
// table, and the tree walks a subset of a table once per parent. At two levels
// the subset is the whole table, so counting positions inside the subset gives
// the right answer by accident and no two level test can tell the two apart. At
// three levels it stopped being an accident: grandchildren appeared under the
// wrong parent, the real ones were absent, and the record count still added up,
// so nothing counting rows would have noticed.
//
// # Why the flat form is the oracle
//
// The same run written a file per entity names each parent by key rather than
// by position, and it cannot be affected by how the tree is walked. Comparing
// the two asks the one question that matters: does the tree say what the run
// produced.
func TestTheTreeCarriesEveryGrandchildUnderTheParentItBelongsTo(t *testing.T) {
	whole, made := run(t, threeLevels(false), asJSON).written["dataset.json"]
	if !made {
		t.Fatal("no single file was written")
	}
	flat, made := run(t, threeLevels(true), asJSON).written["item.json"]
	if !made {
		t.Fatal("no file for the grandchild was written")
	}

	var loose []treeItem
	if err := json.Unmarshal([]byte(flat.String()), &loose); err != nil {
		t.Fatalf("the flat form is not readable: %v", err)
	}
	expected := map[string][]string{}
	for _, item := range loose {
		expected[item.Order] = append(expected[item.Order], item.City)
	}
	if len(expected) != 4 {
		t.Fatalf("the run was meant to make four orders with grandchildren, it made %d",
			len(expected))
	}

	var tree treeRoot
	if err := json.Unmarshal([]byte(whole.String()), &tree); err != nil {
		t.Fatalf("the tree is not readable: %v", err)
	}
	seen := map[string]bool{}
	for _, customer := range tree.Customers {
		for _, order := range customer.Orders {
			belongsTo := oneParent(t, order.Items)
			if seen[belongsTo] {
				t.Fatalf("the grandchildren of order %s stand under two different orders",
					belongsTo)
			}
			seen[belongsTo] = true
			if got := cities(order.Items); !sameOrder(got, expected[belongsTo]) {
				t.Errorf("order %s holds %v in the tree and %v in the flat form",
					belongsTo, got, expected[belongsTo])
			}
		}
	}
	for belongsTo := range expected {
		if !seen[belongsTo] {
			t.Errorf("the grandchildren of order %s are in no parent at all", belongsTo)
		}
	}
}

// oneParent answers which parent a group of grandchildren names, and refuses a
// group that names more than one.
func oneParent(t *testing.T, items []treeItem) string {
	t.Helper()
	if len(items) == 0 {
		t.Fatal("an order in the tree holds no grandchildren, and each was given two")
	}
	for _, item := range items[1:] {
		if item.Order != items[0].Order {
			t.Fatalf("one order holds grandchildren of %s and of %s", items[0].Order, item.Order)
		}
	}
	return items[0].Order
}

func cities(items []treeItem) []string {
	taken := make([]string, 0, len(items))
	for _, item := range items {
		taken = append(taken, item.City)
	}
	return taken
}

func sameOrder(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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

// A table of one column with blank values reads back with every record it was
// written with.
//
// Reproduced by an outside review on 2026-10-22: three blank rows were written
// as three empty lines, a reader of the format skips an empty line, and the
// file read back as a header and no records.
func TestABlankRecordOfOneColumnIsStillARecord(t *testing.T) {
	files := newHeld()
	rec := recipe.Recipe{Seed: "s", Locale: "en-US",
		Output: recipe.Output{FilePerEntity: true},
		Entities: []recipe.Entity{{Name: "t", Count: 3, HasCount: true,
			Fields: []recipe.Field{{Name: "name"}}}}}
	writer, err := format.NewCSV(files, rec)
	if err != nil {
		t.Fatalf("the writer was refused: %v", err)
	}
	planned := dataset.Table{Entity: "t", Columns: []string{"name"}, Rows: 3}
	if err := writer.Begin(planned); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if err := writer.Write(dataset.Record{Values: []string{""}, Parent: dataset.NoParent}); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.End(planned); err != nil {
		t.Fatal(err)
	}
	written := files.written["t.csv"].String()
	records, err := csv.NewReader(strings.NewReader(written)).ReadAll()
	if err != nil {
		t.Fatalf("the file does not read back: %v\n%q", err, written)
	}
	if len(records) != 4 {
		t.Errorf("a header and three blank records read back as %d lines: %q", len(records), written)
	}
	if written != "name\n\"\"\n\"\"\n\"\"\n" {
		t.Errorf("the bytes are %q", written)
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

// A field with fewer possible values than rows cannot be unique, and the plan
// says so with the arithmetic before a row is written.
//
// 🔴 The run used to find out after twenty draws and say the field had fewer
// values than the rows, which it did not know. A gender has two, and the plan
// knows that since 2026-10-26, so the refusal moved forward and gained the
// number it was guessing at.
func TestAUniqueFieldWithTooFewValuesIsRefusedWithTheArithmetic(t *testing.T) {
	rec := sample(t, true, 0)
	rec.Entities[0].Count = 500
	rec.Entities[0].Fields = append(rec.Entities[0].Fields,
		recipe.Field{Name: "sex", Generator: "gender", Unique: true})

	_, err := generate.Prepare(rec, values.New(rec.Locale, rec.Seed))
	if err == nil {
		t.Fatal("five hundred rows of a field with two possible values were planned")
	}
	if fault.KindOf(err) != fault.Data {
		t.Errorf("the refusal is %v, and a recipe that is valid and cannot be produced is data", fault.KindOf(err))
	}
	for _, part := range []string{"sex", "unique", "can take 2 ", "500 rows"} {
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

// A parent with a field named after one of its children is refused by the
// writer of the tree, and by nobody else, because only the tree puts the two
// under one name.
//
// Reproduced by an outside review on 2026-10-22: the object of a parent held
// "child": "469479" and then "child": [...], and the run ended with success.
func TestATreeRefusesAParentFieldNamedAfterAChild(t *testing.T) {
	rec := recipe.Recipe{Seed: "s", Locale: "en-US",
		Output: recipe.Output{Format: format.NameJSON},
		Entities: []recipe.Entity{
			{Name: "parent", Count: 2, HasCount: true,
				Fields: []recipe.Field{{Name: "child", Type: "integer"}}},
			{Name: "child", Parents: []recipe.Parent{{Entity: "parent", Min: 1, Max: 1}},
				Fields: []recipe.Field{{Name: "n", Type: "integer"}}},
		}}
	_, err := format.NewJSON(newHeld(), rec, generate.NeverStops{})
	if err == nil {
		t.Fatal("a tree with two members called child in one object was accepted")
	}
	if fault.KindOf(err) != fault.Config || !strings.Contains(err.Error(), "entities.parent.fields.child") {
		t.Errorf("the refusal is %v with %q", fault.KindOf(err), err)
	}

	for _, other := range []recipe.Output{
		{Format: format.NameJSON, FilePerEntity: true},
		{Format: format.NameCSV},
	} {
		rec.Output = other
		if err := format.NestingCollision(rec); err != nil {
			t.Errorf("%+v writes no tree and was refused: %v", other, err)
		}
	}
}

// 🔴 A writer that trusts its caller is a writer that writes two things into
// one column the day it gains a second caller.
//
// The plan now refuses this recipe before any writer is built, which is where
// B-205 put the rule so that validate and a dry run could see it too. That
// makes this line unreachable through generate, and unreachable is exactly the
// state in which a guard quietly stops being one. So the writer is called here
// directly, with the table a plan would have handed it.
func TestAWriterRefusesATableWhoseRelationWantsAColumnAFieldAlreadyHas(t *testing.T) {
	rec := recipe.Recipe{
		Version: "0.1", Seed: "s", Locale: "pl-PL",
		Output: recipe.Output{Format: "csv"},
		Entities: []recipe.Entity{
			{
				Name: "customer", Count: 1, HasCount: true,
				Fields: []recipe.Field{{Name: "first_name", Generator: "first_name"}},
			},
			{
				Name: "order", Count: 1, HasCount: true,
				Fields:  []recipe.Field{{Name: "customer", Generator: "first_name"}},
				Parents: []recipe.Parent{{Entity: "customer", Min: 0, Max: 1}},
			},
		},
	}
	writer, err := format.NewCSV(newHeld(), rec)
	if err != nil {
		t.Fatalf("the writer could not be built at all: %v", err)
	}

	table := dataset.Table{
		Entity: "order", Columns: []string{"customer"}, Rows: 1,
		Parent: "customer", ParentOf: []int{0},
	}
	if err := writer.Begin(table); err == nil {
		t.Fatal("the writer began a table in which the relation and a field want one column")
	} else if !strings.Contains(err.Error(), "customer") {
		t.Errorf("the refusal does not name the column both things want: %v", err)
	}
}
