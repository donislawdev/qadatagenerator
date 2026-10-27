package format_test

import (
	"encoding/csv"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
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
	entries := make([]string, 50)
	for index := range entries {
		entries[index] = "s" + strconv.Itoa(index)
	}
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
