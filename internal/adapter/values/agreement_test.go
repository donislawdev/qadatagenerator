package values_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// kept is a writer that holds the rows of one table in memory, keyed by column.
type kept struct {
	columns []string
	rows    []map[string]string
}

func (k *kept) Begin(table dataset.Table) error { k.columns = table.Columns; return nil }
func (k *kept) End(dataset.Table) error         { return nil }
func (k *kept) Finish() error                   { return nil }

func (k *kept) Write(record dataset.Record) error {
	row := make(map[string]string, len(k.columns))
	for i, column := range k.columns {
		row[column] = record.Values[i]
	}
	k.rows = append(k.rows, row)
	return nil
}

// person is an entity of the given fields and rules, in a Polish recipe.
func person(rows int, fields []recipe.Field, rules ...recipe.Rule) recipe.Recipe {
	return recipe.Recipe{
		Version: "0.1", Seed: "agreement", Locale: "pl-PL",
		Output: recipe.Output{Format: "csv"},
		Entities: []recipe.Entity{{
			Name: "person", Count: rows, HasCount: true, Fields: fields, Consistency: rules,
		}},
	}
}

func field(name, generator string) recipe.Field {
	return recipe.Field{Name: name, Generator: generator}
}

// produced plans and runs a recipe, failing the test on either refusal.
func produced(t *testing.T, rec recipe.Recipe) (generate.Plan, []map[string]string) {
	t.Helper()
	source := values.New(rec.Locale, rec.Seed)
	plan, err := generate.Prepare(rec, source, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}
	table := &kept{}
	if err := generate.Run(plan, rec, source, table, generate.NeverStops{}); err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	return plan, table.rows
}

func listed(t *testing.T, list string) map[string]bool {
	t.Helper()
	entries, found := corpus.Load("pl-PL", list)
	if !found {
		t.Fatalf("this build carries no %s for pl-PL", list)
	}
	set := make(map[string]bool, len(entries))
	for _, entry := range entries {
		set[entry] = true
	}
	return set
}

// A first name and a surname beside a gender column follow it although no rule
// says so, because the format promises consistency by default.
//
// 🔴 The case that found it, from the owner on 2026-10-21: the example recipe
// of the window gave Piotr to a woman. Measured on 2000 rows of that recipe
// before the change, 999 first names disagreed with the gender column.
func TestANameFollowsTheGenderColumnThoughNoRuleSaysSo(t *testing.T) {
	plan, rows := produced(t, person(400, []recipe.Field{
		field("first_name", "first_name"), field("last_name", "last_name"), field("gender", "gender"),
	}))
	female, male := listed(t, "first-names-female"), listed(t, "first-names-male")
	women := 0
	for i, row := range rows {
		switch row["gender"] {
		case "female":
			women++
			if !female[row["first_name"]] {
				t.Errorf("row %d gives the woman the name %q", i, row["first_name"])
			}
		case "male":
			if !male[row["first_name"]] {
				t.Errorf("row %d gives the man the name %q", i, row["first_name"])
			}
		}
	}
	if women == 0 || women == len(rows) {
		t.Fatalf("%d of %d rows are women, so this test compared one gender only", women, len(rows))
	}

	var said []string
	for _, kept := range plan.Implied {
		said = append(said, kept.Rule.Field+" follows "+strings.Join(kept.Rule.With, ","))
	}
	want := []string{"first_name follows gender", "last_name follows gender"}
	if !reflect.DeepEqual(said, want) {
		t.Errorf("the plan says %q about what follows what, and should say %q", said, want)
	}
}

// A recipe that writes its rules out produces the same bytes as one that leaves
// them to be implied, which is what lets a rule be implied without moving
// anybody's dataset that already declared it.
func TestWritingTheRulesOutChangesNothing(t *testing.T) {
	fields := []recipe.Field{
		field("first_name", "first_name"), field("last_name", "last_name"), field("gender", "gender"),
		field("birth_date", "birth_date"), field("pesel", "pesel"),
	}
	_, implied := produced(t, person(200, fields))
	_, written := produced(t, person(200, fields,
		recipe.Rule{Field: "pesel", Kind: recipe.Matches, With: []string{"birth_date", "gender"}},
		recipe.Rule{Field: "first_name", Kind: recipe.Matches, With: []string{"gender"}},
		recipe.Rule{Field: "last_name", Kind: recipe.Matches, With: []string{"gender"}},
	))
	if !reflect.DeepEqual(implied, written) {
		t.Error("the same recipe with its rules written out produced different rows")
	}
}

// A declared rule that leaves out a column the identifier reads is completed,
// because nobody writes pesel matches: [gender] meaning a date of birth that
// disagrees with the birth_date column beside it.
func TestARuleIsCompletedWithTheColumnItLeavesOut(t *testing.T) {
	plan, rows := produced(t, person(100, []recipe.Field{
		field("gender", "gender"), field("birth_date", "birth_date"), field("pesel", "pesel"),
	}, recipe.Rule{Field: "pesel", Kind: recipe.Matches, With: []string{"gender"}}))
	for i, row := range rows {
		if !bornAgrees(row["pesel"], row["birth_date"]) {
			t.Errorf("row %d: the PESEL %s does not carry the date %s", i, row["pesel"], row["birth_date"])
		}
	}
	if len(plan.Implied) != 1 || !reflect.DeepEqual(plan.Implied[0].Rule.With, []string{"birth_date"}) {
		t.Errorf("the plan should name birth_date alone as what was added, and says %+v", plan.Implied)
	}
}

// bornAgrees checks the six date digits of a PESEL against a date written as
// YYYY-MM-DD, allowing for the month carrying the century.
func bornAgrees(national, born string) bool {
	if len(national) != 11 || len(born) != 10 {
		return false
	}
	if national[0:2] != born[2:4] || national[4:6] != born[8:10] {
		return false
	}
	month := int(national[2]-'0')*10 + int(national[3]-'0')
	return month%20 == int(born[5]-'0')*10+int(born[6]-'0')
}

// Two gender columns bind neither, and the plan says which field was left
// following nothing, because which of them a name follows is the recipe's to
// say.
func TestTwoGenderColumnsBindNeitherAndThePlanSaysSo(t *testing.T) {
	plan, _ := produced(t, person(10, []recipe.Field{
		field("first_name", "first_name"), field("gender", "gender"), field("partner_gender", "gender"),
	}))
	if len(plan.Implied) != 0 {
		t.Errorf("a name was bound to one of two gender columns: %+v", plan.Implied)
	}
	if !noted(plan, "first_name could follow gender or partner_gender") {
		t.Errorf("the plan does not say the name follows neither column: %+v", plan.Notes)
	}
}

// A blank in the gender column binds nothing in that row, so the identifier
// draws a gender of its own there instead of stopping the run.
//
// 🔴 Until 2026-10-21 a blank was bound like any value, and a recipe giving the
// gender a share of empty values and a PESEL that matches it stopped on the
// first blank row.
func TestABlankGenderLeavesTheIdentifierToDrawItsOwn(t *testing.T) {
	gender := field("gender", "gender")
	gender.EmptyShare, gender.HasEmptyShare = 40, true
	_, rows := produced(t, person(100, []recipe.Field{gender, field("pesel", "pesel")}))
	blank := 0
	for i, row := range rows {
		if row["gender"] == "" {
			blank++
			continue
		}
		if even := (row["pesel"][9]-'0')%2 == 0; even != (row["gender"] == "female") {
			t.Errorf("row %d: the PESEL %s disagrees with %q", i, row["pesel"], row["gender"])
		}
	}
	if blank == 0 {
		t.Fatal("no row left the gender blank, so this test never reached the case it is about")
	}
}

// A rule the generator takes no account of is named in the plan rather than
// dropped in silence.
func TestARuleTheGeneratorIgnoresIsNamedInThePlan(t *testing.T) {
	plan, _ := produced(t, person(5, []recipe.Field{field("gender", "gender"), field("nip", "nip")},
		recipe.Rule{Field: "nip", Kind: recipe.Matches, With: []string{"gender"}}))
	if !noted(plan, "nip takes no account of gender") {
		t.Errorf("the plan does not say the rule does nothing: %+v", plan.Notes)
	}
}

// A rule resting on a column another rule binds is refused by the plan, in the
// words the run uses, so that validate refuses what generate would.
func TestAChainOfRulesIsRefusedBeforeAnyRow(t *testing.T) {
	rec := person(5, []recipe.Field{
		field("first_name", "first_name"), field("gender", "gender"), field("pesel", "pesel"),
	},
		recipe.Rule{Field: "gender", Kind: recipe.Matches, With: []string{"first_name"}},
		recipe.Rule{Field: "pesel", Kind: recipe.Matches, With: []string{"gender"}},
	)
	_, err := generate.Prepare(rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
	if err == nil || !strings.Contains(err.Error(), "resolves one level") {
		t.Errorf("the plan accepted a rule resting on a bound column, answering %v", err)
	}
}

func noted(plan generate.Plan, words string) bool {
	for _, note := range plan.Notes {
		if strings.Contains(note.What, words) {
			return true
		}
	}
	return false
}
