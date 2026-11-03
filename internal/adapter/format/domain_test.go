package format_test

import (
	"encoding/csv"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// oneTable is a recipe of one table and the fields handed in.
func oneTable(count int, fields ...recipe.Field) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1", Seed: "domain", Locale: "pl-PL",
		Output:   recipe.Output{Format: "csv", FilePerEntity: true},
		Entities: []recipe.Entity{{Name: "account", Count: count, HasCount: true, Fields: fields}},
	}
}

// rowsOf runs a recipe and returns the records of its one table, without the
// header, together with the bytes written.
func rowsOf(t *testing.T, rec recipe.Recipe) ([][]string, string) {
	t.Helper()
	written := run(t, rec, asCSV).written["account.csv"].String()
	records, err := csv.NewReader(strings.NewReader(written)).ReadAll()
	if err != nil {
		t.Fatalf("the table does not read back: %v", err)
	}
	return records[1:], written
}

// A unique list is used up whole when the table has as many rows as the list
// has entries, and the same seed uses it up the same way every time.
//
// 🔴 Until 2026-10-26 the twentieth draw that landed on a taken entry ended the
// run. Fifty entries are what makes the walk certain to be needed here: with
// one entry left of fifty, twenty draws miss it two times in three, and a list
// of five is used up by drawing alone in about ninety nine runs of a hundred.
func TestAUniqueListIsUsedUpWholeAndTheSameEveryTime(t *testing.T) {
	entries := listOf("s", 50)
	rec := oneTable(len(entries), recipe.Field{Name: "status", Values: entries, Unique: true})

	records, first := rowsOf(t, rec)
	_, second := rowsOf(t, rec)

	seen := map[string]bool{}
	for _, record := range records {
		seen[record[0]] = true
	}
	if len(records) != len(entries) || len(seen) != len(entries) {
		t.Errorf("fifty rows of a list of fifty gave %d rows and %d different values", len(records), len(seen))
	}
	if first != second {
		t.Error("the same seed used the list up in two different ways")
	}
}

// A unique number of two digits fills all hundred values, which is the case
// where drawing alone fails nearly every row at the end.
func TestAUniqueNumberFillsItsWholeDomain(t *testing.T) {
	rec := oneTable(100, recipe.Field{Name: "code", Type: "integer", MaxLength: 2, HasMaxLength: true, Unique: true})

	records, _ := rowsOf(t, rec)

	seen := map[string]bool{}
	for _, record := range records {
		seen[record[0]] = true
	}
	if len(seen) != 100 {
		t.Errorf("a hundred rows of a hundred values gave %d different values", len(seen))
	}
}

// A key is held whole: its parts repeat on their own and no two rows share the
// pair.
func TestAKeyHeldWholeNeverRepeatsWhileItsPartsDo(t *testing.T) {
	rec := oneTable(6, recipe.Field{Name: "branch", Values: []string{"x", "y"}},
		recipe.Field{Name: "line", Values: []string{"1", "2", "3"}})
	rec.Entities[0].PrimaryKey = []string{"branch", "line"}

	records, _ := rowsOf(t, rec)

	pairs := map[string]bool{}
	for _, record := range records {
		pairs[record[0]+"/"+record[1]] = true
	}
	if len(pairs) != 6 {
		t.Errorf("six rows of a key of two by three gave %d different keys: %v", len(pairs), records)
	}
}

// A row built again for its key gives back what its unique columns took, so a
// unique column of ten values beside a key that collides still fills all ten.
// Kept, each collision would spend a value nobody wrote, and the column would
// run out before the table did.
func TestARowBuiltAgainForItsKeyGivesBackWhatItTook(t *testing.T) {
	letters := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	rec := oneTable(10, recipe.Field{Name: "code", Values: letters},
		recipe.Field{Name: "n", Type: "integer", MaxLength: 1, HasMaxLength: true, Unique: true})
	rec.Entities[0].PrimaryKey = []string{"code"}

	records, first := rowsOf(t, rec)
	_, second := rowsOf(t, rec)

	codes, numbers := map[string]bool{}, map[string]bool{}
	for _, record := range records {
		codes[record[0]], numbers[record[1]] = true, true
	}
	if len(codes) != 10 || len(numbers) != 10 {
		t.Errorf("ten rows gave %d different keys and %d different numbers", len(codes), len(numbers))
	}
	if first != second {
		t.Error("the same seed built the rows again in two different ways")
	}
}

// A unique field bound to a rule draws from a domain that depends on the row,
// so the plan leaves it to the run, and the run says when every value is used
// rather than claiming a number it guessed.
func TestAUniqueFieldBoundToARuleSaysWhenEveryValueIsUsed(t *testing.T) {
	rec := oneTable(3, recipe.Field{Name: "sex", Generator: "gender"},
		recipe.Field{Name: "code", Values: []string{"p", "q"}, Unique: true})
	rec.Entities[0].Consistency = []recipe.Rule{{Field: "code", Kind: recipe.Matches, With: []string{"sex"}}}

	err := runFailing(t, rec, values.New(rec.Locale, rec.Seed))

	for _, part := range []string{"fields.code is unique", "every one of the 2 values", "row 3 of 3"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("the refusal does not carry %q: %v", part, err)
		}
	}
}

// askedFor is the real source counting, field by field, how often it is asked
// for a domain.
type askedFor struct {
	values.Source
	asked map[string]int
}

func (a askedFor) DomainOf(req generate.Request) (generate.Domain, bool) {
	a.asked[req.At.Field]++
	return a.Source.DomainOf(req)
}

// inBlocks is the real source with the gender column drawn as a woman in the
// first rows and as a man in the rest.
type inBlocks struct {
	values.Source
	women int64
}

func (b inBlocks) Produce(req generate.Request) (string, error) {
	switch {
	case req.At.Field != "sex":
		return b.Source.Produce(req)
	case req.At.Row < b.women:
		return "female", nil
	}
	return "male", nil
}

// rowsFrom plans and runs a recipe of one table on the source handed in, calls
// between before the run starts, and returns the records without the header.
func rowsFrom(t *testing.T, rec recipe.Recipe, source generate.Values, between func()) [][]string {
	t.Helper()
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	between()
	files := newHeld()
	writer, err := format.NewCSV(files, rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := generate.Run(plan, rec, source, writer, generate.NeverStops{}); err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(files.written["account.csv"].String())).ReadAll()
	if err != nil {
		t.Fatalf("the table does not read back: %v", err)
	}
	return records[1:]
}

// listOf is a list of entries that all begin with the same prefix.
func listOf(prefix string, size int) []string {
	entries := make([]string, size)
	for index := range entries {
		entries[index] = prefix + strconv.Itoa(index)
	}
	return entries
}

// The domain of a unique field that is the same in every row is asked for once
// by the run, however many rows walk it: a field no rule binds, and a field a
// rule binds that draws from a list under values all the same.
//
// 🔴 Until 2026-11-03 every row that walked asked again, and the source counts
// a list anew each time, so a list of 28000 entries took 2.8 s where the same
// table without unique took 0.077 s (B-348). Five hundred entries make 21 or
// 22 rows of each field walk, which is how often the run asked before, and
// fields with lists of their own check that each keeps its own answer.
func TestADomainTheSameInEveryRowIsAskedForOnce(t *testing.T) {
	lists := map[string][]string{"code": listOf("c", 500), "tag": listOf("t", 500), "mark": listOf("m", 500)}
	rec := oneTable(500, recipe.Field{Name: "sex", Generator: "gender"},
		recipe.Field{Name: "code", Values: lists["code"], Unique: true},
		recipe.Field{Name: "tag", Values: lists["tag"], Unique: true},
		recipe.Field{Name: "mark", Values: lists["mark"], Unique: true})
	rec.Entities[0].Consistency = []recipe.Rule{{Field: "mark", Kind: recipe.Matches, With: []string{"sex"}}}
	source := askedFor{values.New(rec.Locale, rec.Seed), map[string]int{}}

	records := rowsFrom(t, rec, source, func() { clear(source.asked) })

	for at, name := range []string{"code", "tag", "mark"} {
		column := at + 1
		listed, seen := map[string]bool{}, map[string]bool{}
		for _, entry := range lists[name] {
			listed[entry] = true
		}
		for _, record := range records {
			if !listed[record[column]] {
				t.Fatalf("%s holds %q, which its list does not", name, record[column])
			}
			seen[record[column]] = true
		}
		if len(seen) != len(lists[name]) {
			t.Errorf("%s holds %d different values of %d", name, len(seen), len(lists[name]))
		}
		if source.asked[name] != 1 {
			t.Errorf("the run asked for the domain of %s %d times", name, source.asked[name])
		}
	}
}

// A unique field a rule binds to the gender walks the domain of the gender of
// its row, so a table that uses up the first names of women and then those of
// men gives every row a name of its own gender.
//
// One answer kept for the whole column would hand the men the names of women,
// which are all taken by then, and the run would say every value is used.
func TestAUniqueFieldBoundToTheGenderWalksTheDomainOfItsRow(t *testing.T) {
	plain := values.New("pl-PL", "domain")
	of := func(gender string) map[string]bool {
		req := generate.Request{Generator: values.FirstName,
			Agrees: provider.NewAgreement(map[string]string{provider.RoleGender: gender})}
		domain, _ := plain.DomainOf(req)
		names := map[string]bool{}
		for place := int64(0); place < domain.Size; place++ {
			names[domain.At(place)] = true
		}
		return names
	}
	women, men := of("female"), of("male")
	shared := 0
	for name := range women {
		if men[name] {
			shared++
		}
	}
	rec := oneTable(len(women)+len(men)-shared, recipe.Field{Name: "sex", Generator: "gender"},
		recipe.Field{Name: "name", Generator: "first_name", Unique: true})
	rec.Entities[0].Consistency = []recipe.Rule{{Field: "name", Kind: recipe.Matches, With: []string{"sex"}}}

	records := rowsFrom(t, rec, inBlocks{values.New(rec.Locale, rec.Seed), int64(len(women))}, func() {})

	own := map[string]map[string]bool{"female": women, "male": men}
	for row, record := range records {
		if !own[record[0]][record[1]] {
			t.Fatalf("row %d is %s and holds the name %q", row+1, record[0], record[1])
		}
	}
}

// sameValue is the real source with its draw replaced by one value and its
// domains made unknown, which is what an identifier with a narrow scheme looks
// like to the run.
type sameValue struct{ values.Source }

func (sameValue) Produce(generate.Request) (string, error) { return "same", nil }

func (sameValue) DomainOf(generate.Request) (generate.Domain, bool) { return generate.Domain{}, false }

// Where the values cannot be counted, running out of luck is said as what
// happened, and nothing is claimed about how many values there are.
func TestAColumnThatCannotBeCountedSaysWhatHappenedAndNoMore(t *testing.T) {
	unique := oneTable(2, recipe.Field{Name: "code", Generator: "pesel", Unique: true})
	keyed := oneTable(2, recipe.Field{Name: "code", Generator: "pesel"})
	keyed.Entities[0].PrimaryKey = []string{"code"}

	for name, test := range map[string]struct {
		rec  recipe.Recipe
		says string
	}{
		"a unique column": {unique, "does not count how many values"},
		"a key":           {keyed, "does not count how many keys"},
	} {
		err := runFailing(t, test.rec, sameValue{values.New(test.rec.Locale, test.rec.Seed)})
		if !strings.Contains(err.Error(), test.says) || strings.Contains(err.Error(), "fewer") {
			t.Errorf("%s: %v", name, err)
		}
		if fault.KindOf(err) != fault.Data {
			t.Errorf("%s: the refusal is %v rather than data", name, fault.KindOf(err))
		}
	}
}

// runFailing plans and runs a recipe that the plan accepts and the run has to
// refuse, and returns what the run said.
func runFailing(t *testing.T, rec recipe.Recipe, source generate.Values) error {
	t.Helper()
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	writer, err := format.NewCSV(newHeld(), rec)
	if err != nil {
		t.Fatal(err)
	}
	err = generate.Run(plan, rec, source, writer, generate.NeverStops{})
	if err == nil {
		t.Fatal("the run finished")
	}
	return err
}
