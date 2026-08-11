package corrupt_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// aCatalog builds a catalog of entries with the given identifiers, all of them
// aimed at text fields.
func aCatalog(t *testing.T, ids ...catalog.ID) catalog.Catalog {
	t.Helper()
	entries := make([]catalog.Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, catalog.Entry{
			ID: id, Layer: catalog.Layer(strings.SplitN(string(id), ".", 2)[0]),
			Summary:     "damage",
			AppliesTo:   catalog.AppliesTo{FieldTypes: []string{"string"}, Formats: []string{catalog.Any}, Providers: []string{catalog.Any}},
			Expectation: catalog.Rejection, Confidence: catalog.Hard,
			Example: catalog.Example{Before: "before", After: "after"},
		})
	}
	built, problems := catalog.New("0.1", "draft", entries)
	if len(problems) > 0 {
		t.Fatalf("the catalog for this test is not usable: %v", problems)
	}
	return built
}

// aView is a schema of one entity with text fields.
func aView(fields ...string) recipe.SchemaView {
	entity := recipe.Entity{Name: "customer", Count: 10, HasCount: true}
	for _, name := range fields {
		entity.Fields = append(entity.Fields, recipe.Field{Name: name, Type: "string"})
	}
	return recipe.Recipe{Seed: "s", Locale: "pl-PL", Entities: []recipe.Entity{entity}}
}

func selectorFor(t *testing.T, policy recipe.Policy, described catalog.Catalog,
	view recipe.SchemaView) *corrupt.Selector {
	t.Helper()
	made, err := corrupt.NewSelector(policy, described, view, seed.New("orders"))
	if err != nil {
		t.Fatalf("the selector refused this policy: %v", err)
	}
	return &made
}

// Every described kind of damage gets an order before any of them repeats.
//
// 🔴 This is the reason the mutator is taken in order rather than drawn.
// Drawing across as many orders as there are entries leaves about a third of
// them untouched, which is the coupon collector rather than bad luck, and
// somebody naming a layer is asking for it to be covered rather than sampled.
func TestEveryKindOfDamageIsOrderedBeforeAnyRepeats(t *testing.T) {
	described := aCatalog(t, "L2.a.one", "L2.b.two", "L2.c.three", "L2.d.four")
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L2"}, Rows: "100%"},
		described, aView("name", "city"))

	seen := map[catalog.ID]int{}
	for row := 0; row < 4; row++ {
		for _, order := range selector.Orders("customer", row, 4) {
			seen[order.Mutator]++
		}
	}
	if len(seen) != 4 {
		t.Fatalf("four rows covered %d of the four kinds of damage: %v", len(seen), seen)
	}
	for id, times := range seen {
		if times != 1 {
			t.Errorf("%s was ordered %d times before the others had their turn", id, times)
		}
	}
}

// The same seed chooses the same rows. Without this the second pillar cannot be
// used in a pipeline at all: determinism covers which rows were damaged and how,
// not only which values were generated.
func TestTheSameSeedChoosesTheSameRows(t *testing.T) {
	described := aCatalog(t, "L2.a.one", "L2.b.two")
	view := aView("name")
	policy := recipe.Policy{Layers: []string{"L2"}, Rows: "30%"}

	first := chosenRows(selectorFor(t, policy, described, view), 200)
	second := chosenRows(selectorFor(t, policy, described, view), 200)
	if len(first) == 0 {
		t.Fatal("a share of thirty percent chose no row at all out of two hundred")
	}
	if len(first) != len(second) {
		t.Fatalf("two runs of one seed chose %d and %d rows", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("the runs part company at position %d: %d and %d", i, first[i], second[i])
		}
	}
}

func chosenRows(selector *corrupt.Selector, rows int) []int {
	var chosen []int
	for row := 0; row < rows; row++ {
		if len(selector.Orders("customer", row, rows)) > 0 {
			chosen = append(chosen, row)
		}
	}
	return chosen
}

// A share of rows damages about that share, so that the number in the recipe
// means something. It is a draw rather than a count, and the test says so by
// allowing a band rather than an exact figure.
func TestAShareOfRowsDamagesAboutThatShare(t *testing.T) {
	described := aCatalog(t, "L2.a.one")
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L2"}, Rows: "25%"},
		described, aView("name"))
	const rows = 2000
	chosen := len(chosenRows(selector, rows))
	if chosen < rows*20/100 || chosen > rows*30/100 {
		t.Errorf("a share of 25%% chose %d of %d rows, which is outside 20%% to 30%%", chosen, rows)
	}
}

// A column the policy excluded is never damaged.
func TestAnExcludedColumnIsNeverAimedAt(t *testing.T) {
	described := aCatalog(t, "L2.a.one", "L2.b.two")
	selector := selectorFor(t, recipe.Policy{
		Layers: []string{"L2"}, Rows: "100%", ExcludeFields: []string{"pesel"},
	}, described, aView("name", "pesel", "city"))

	for row := 0; row < 20; row++ {
		for _, order := range selector.Orders("customer", row, 20) {
			if order.Field == "pesel" {
				t.Fatalf("row %d aimed damage at an excluded column", row)
			}
		}
	}
}

// schemeCatalog builds one entry aimed at identifiers of named schemes, which
// is how the eight entries naming schemes are written in the catalog.
func schemeCatalog(t *testing.T, id catalog.ID, schemes ...string) catalog.Catalog {
	t.Helper()
	built, problems := catalog.New("0.1", "draft", []catalog.Entry{{
		ID: id, Layer: catalog.Layer(strings.SplitN(string(id), ".", 2)[0]), Summary: "damage",
		AppliesTo: catalog.AppliesTo{
			FieldTypes: []string{"identifier"}, Formats: []string{catalog.Any}, Providers: schemes,
		},
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Example: catalog.Example{Before: "before", After: "after"},
	}})
	if len(problems) > 0 {
		t.Fatalf("the catalog for this test is not usable: %v", problems)
	}
	return built
}

// identifiersView is a schema with a text column and two identifier columns of
// different schemes, which is the shape that tells the three tiers apart.
func identifiersView() recipe.SchemaView {
	return schemeView{}
}

// The rest of the view is written out rather than embedded, for the same reason
// the real one is: adding a method to the interface should fail to compile here
// instead of silently inheriting an answer nobody thought about.
type schemeView struct{}

func (schemeView) EntityNames() []string { return []string{"customer"} }

func (schemeView) FieldsOf(string) []recipe.FieldInfo {
	return []recipe.FieldInfo{
		{Name: "name", Type: "string"},
		{Name: "ssn", Type: "identifier", Provider: "US.ssn"},
		{Name: "pesel", Type: "identifier", Provider: "PL.pesel"},
	}
}

func (schemeView) ParentsOf(string) []recipe.ParentInfo { return nil }
func (schemeView) RulesOf(string) []recipe.Rule         { return nil }
func (schemeView) PrimaryKeyOf(string) []string         { return nil }

// Damage meant for one scheme lands on the column that follows it, when the
// dataset carries columns of several.
func TestDamageMeantForOneSchemeLandsOnThatScheme(t *testing.T) {
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L3"}, Rows: "100%"},
		schemeCatalog(t, "L3.checksum.off_by_one", "PL.pesel"), identifiersView())

	for row := 0; row < 20; row++ {
		for _, order := range selector.Orders("customer", row, 20) {
			if order.Field != "pesel" {
				t.Fatalf("row %d aimed check digit damage at %q", row, order.Field)
			}
		}
	}
}

// 🔴 When no column follows the scheme, the order points at one of the right
// type anyway, so that the refusal names the difference that is real.
//
// Pointing at the text column instead would have the manifest say the type is
// wrong, in a dataset that carries a column of exactly the right type. That
// sentence would send whoever reads it looking for the wrong thing.
func TestWithNoColumnOfThatSchemeTheOrderStillNamesAnIdentifier(t *testing.T) {
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L3"}, Rows: "100%"},
		schemeCatalog(t, "L3.checksum.off_by_one", "iban"), identifiersView())

	orders := selector.Orders("customer", 0, 20)
	if len(orders) == 0 {
		t.Fatal("no order was produced at all, which would be damage going missing")
	}
	if orders[0].Field != "ssn" && orders[0].Field != "pesel" {
		t.Errorf("the order names %q, which is not a column of the type this damage covers",
			orders[0].Field)
	}
}

// Excluding every column is refused at the start rather than producing orders
// that can never land. A name matching everything and one matching nothing look
// alike in a recipe, and only the arithmetic tells them apart.
func TestAPolicyThatExcludesEverythingIsRefused(t *testing.T) {
	described := aCatalog(t, "L2.a.one")
	_, err := corrupt.NewSelector(recipe.Policy{
		Layers: []string{"L2"}, Rows: "100%", ExcludeFields: []string{"name", "city"},
	}, described, aView("name", "city"), seed.New("s"))
	if err == nil {
		t.Fatal("a policy excluding every column was accepted")
	}
	if !strings.Contains(err.Error(), "exclude_fields") {
		t.Errorf("the refusal does not name the key at fault: %v", err)
	}
}

// exactly_one means one defect per damaged row, and any means more than one.
// The two are the whole point of the setting: one isolates a cause, several
// test what happens when failures cascade.
func TestPerRowDecidesHowMuchOneRowCarries(t *testing.T) {
	described := aCatalog(t, "L2.a.one", "L2.b.two", "L2.c.three", "L2.d.four")
	view := aView("name")
	for _, each := range []struct {
		perRow string
		want   int
	}{{"exactly_one", 1}, {"any", 3}} {
		selector := selectorFor(t, recipe.Policy{
			Layers: []string{"L2"}, Rows: "100%", PerRow: each.perRow,
		}, described, view)
		if got := len(selector.Orders("customer", 0, 10)); got != each.want {
			t.Errorf("per_row %s put %d defects on a row, want %d", each.perRow, got, each.want)
		}
	}
}
