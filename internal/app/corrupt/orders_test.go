package corrupt_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
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

// realLayers builds a catalog shaped like the real one, eight layers of 5, 5,
// 9, 7, 7, 6, 6 and 7 entries, all aimed at text so that every one of them can
// be ordered here. It returns the catalog, the layers and how many entries.
func realLayers(t *testing.T) (catalog.Catalog, []string, int) {
	t.Helper()
	sizes := []struct {
		layer string
		count int
	}{{"L0", 5}, {"L1", 5}, {"L2", 9}, {"L3", 7}, {"L4", 7}, {"L5", 6}, {"UNI", 6}, {"REV", 7}}
	var ids []catalog.ID
	var layers []string
	for _, each := range sizes {
		layers = append(layers, each.layer)
		for i := 0; i < each.count; i++ {
			ids = append(ids, catalog.ID(fmt.Sprintf("%s.area.v%c", each.layer, 'a'+i)))
		}
	}
	return aCatalog(t, ids...), layers, len(ids)
}

// 🔴 A few rows touch as many layers as there are rows, wherever the walk
// starts.
//
// Walked in identifier order, a file of three rows with every layer named got
// the first three entries of L0 for every seed. One run over the whole walk and
// three rows more reads every window of three and four consecutive rows, which
// is every starting place a seed could draw.
func TestAFewRowsTouchAsManyLayersAsThereAreRows(t *testing.T) {
	described, layers, count := realLayers(t)
	rows := count + 3
	selector := selectorFor(t, recipe.Policy{Layers: layers, Rows: "100%"}, described, aView("name"))

	var walked []string
	seen := map[catalog.ID]bool{}
	for row := 0; row < rows; row++ {
		orders := selector.Orders("customer", row, rows)
		if len(orders) != 1 {
			t.Fatalf("row %d carries %d orders, want one", row, len(orders))
		}
		walked = append(walked, strings.SplitN(string(orders[0].Mutator), ".", 2)[0])
		if row < count {
			seen[orders[0].Mutator] = true
		}
	}
	if len(seen) != count {
		t.Errorf("the first %d rows reached %d of the %d entries, so some repeated", count, len(seen), count)
	}
	for _, width := range []int{3, 4} {
		if problem := crowdedWindow(walked, width); problem != "" {
			t.Error(problem)
		}
	}
	// The canary is the shape of the defect: the walk in identifier order.
	if crowdedWindow([]string{"L0", "L0", "L0", "L0", "L1"}, 3) == "" {
		t.Fatal("the check does not notice rows that all land in one layer")
	}
}

// crowdedWindow names the first run of consecutive rows touching fewer layers
// than there are rows, and nothing when there is none.
func crowdedWindow(walked []string, width int) string {
	for start := 0; start+width <= len(walked); start++ {
		distinct := map[string]bool{}
		for _, layer := range walked[start : start+width] {
			distinct[layer] = true
		}
		if len(distinct) < width {
			return fmt.Sprintf("rows %d to %d touched %d layers: %v",
				start+1, start+width, len(distinct), walked[start:start+width])
		}
	}
	return ""
}

// The walk starts at a place drawn from the seed: one seed starts at the same
// entry every time, and other seeds reach other entries.
func TestTheWalkStartsWhereTheSeedSays(t *testing.T) {
	described, layers, _ := realLayers(t)
	policy := recipe.Policy{Layers: layers, Rows: "100%"}
	firstOf := func(name string) catalog.ID {
		made, err := corrupt.NewSelector(policy, described, aView("name"), seed.New(name))
		if err != nil {
			t.Fatalf("the selector refused this policy: %v", err)
		}
		return made.Orders("customer", 0, 10)[0].Mutator
	}
	// Two selectors built from one seed, so the comparison is between runs.
	first, again := firstOf("one seed"), firstOf("one seed")
	if first != again {
		t.Fatalf("one seed started the walk at %s and then at %s", first, again)
	}
	starts := map[catalog.ID]bool{}
	for i := 0; i < 20; i++ {
		starts[firstOf(fmt.Sprintf("seed %d", i))] = true
	}
	// Twenty draws over 52 places reach about seventeen of them.
	if len(starts) < 5 {
		t.Errorf("twenty seeds started the walk at only %d entries", len(starts))
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

// An exclusion that empties any entity is refused, and not only one that
// empties the first.
//
// 🔴 The check used to answer as soon as one entity kept a field, so it was
// about the first entity rather than about the recipe. An exclusion emptying
// the second passed, the run made every row and every temporary file, and the
// failure arrived at the end as exit code 1 with a sentence telling the reader
// to report a defect in the tool. The recipe was theirs and the fault was
// visible before a single row existed, which is why the second half of this
// test asks for the exit kind and not only for the refusal.
func TestAnExclusionThatEmptiesAnyEntityIsRefused(t *testing.T) {
	described := aCatalog(t, "L2.a.one")
	policy := recipe.Policy{Layers: []string{"L2"}, Rows: "100%", ExcludeFields: []string{"drop"}}

	_, err := corrupt.NewSelector(policy, described, twoEntities(), seed.New("orders"))
	if err == nil {
		t.Fatal("an exclusion emptying the second entity was accepted")
	}
	if kind := fault.KindOf(err); kind != fault.Config {
		t.Errorf("a policy the reader wrote is %v rather than a configuration failure", kind)
	}
	if !strings.Contains(err.Error(), "order") {
		t.Errorf("the refusal does not name the entity it emptied: %v", err)
	}
	if strings.Contains(err.Error(), "customer") {
		t.Errorf("the refusal names an entity that still had a field: %v", err)
	}
}

// twoEntities is a schema whose second entity has one field and whose first has
// two, which is the shape in which a check that stops at the first entity looks
// exactly like one that does not.
func twoEntities() recipe.SchemaView {
	return recipe.Recipe{Seed: "s", Locale: "pl-PL", Entities: []recipe.Entity{
		{Name: "customer", Count: 10, HasCount: true, Fields: []recipe.Field{
			{Name: "city", Type: "string"}, {Name: "keep", Type: "string"},
		}},
		{Name: "order", Count: 10, HasCount: true, Fields: []recipe.Field{
			{Name: "drop", Type: "string"},
		}},
	}}
}

// schemeCatalog builds one entry aimed at identifiers of named schemes, which
// is how the eight entries naming schemes are written in the catalog.
func schemeCatalog(t *testing.T, scheme string, ids ...catalog.ID) catalog.Catalog {
	t.Helper()
	entries := make([]catalog.Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, catalog.Entry{
			ID: id, Layer: catalog.Layer(strings.SplitN(string(id), ".", 2)[0]), Summary: "damage",
			AppliesTo: catalog.AppliesTo{
				FieldTypes: []string{"identifier"}, Formats: []string{catalog.Any},
				Providers: []string{scheme},
			},
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

// 🔴 Two identifiers and five columns that are neither, and the count is not
// decoration.
//
// Which column damage lands on is drawn for each entry and each row. With few
// columns and few draws a build that ignored the scheme would still land on the
// right one often enough to look correct. With seven columns and twenty rows,
// being right by luck is not a plausible reading of a green test.
func (schemeView) FieldsOf(string) []recipe.FieldInfo {
	fields := []recipe.FieldInfo{
		{Name: "ssn", Type: "identifier", Scheme: "US.ssn"},
		{Name: "pesel", Type: "identifier", Scheme: "PL.pesel"},
	}
	for _, name := range []string{"name", "city", "region", "note", "title"} {
		fields = append(fields, recipe.FieldInfo{Name: name, Type: "string"})
	}
	return fields
}

func (schemeView) ParentsOf(string) []recipe.ParentInfo { return nil }
func (schemeView) RulesOf(string) []recipe.Rule         { return nil }
func (schemeView) PrimaryKeyOf(string) []string         { return nil }

// Damage meant for one scheme lands on the column that follows it, when the
// dataset carries columns of several.
//
// 🔴 Four entries over twenty rows, and the number of draws is what makes this
// test worth having. Which column an order lands on is drawn for each entry and
// each row, so a build that ignores the scheme is wrong on most of them. Every
// draw landing correctly by chance is not a reading anybody has to entertain.
func TestDamageMeantForOneSchemeLandsOnThatScheme(t *testing.T) {
	described := schemeCatalog(t, "PL.pesel",
		"L3.checksum.off_by_one", "L3.checksum.transposed_digits",
		"L3.national_id.impossible_date", "L3.range.unassigned")
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L3"}, Rows: "100%"},
		described, identifiersView())

	seen := 0
	for row := 0; row < 20; row++ {
		for _, order := range selector.Orders("customer", row, 20) {
			seen++
			if order.Field != "pesel" {
				t.Fatalf("%s was aimed at %q, and only pesel follows the scheme it covers",
					order.Mutator, order.Field)
			}
		}
	}
	if seen < 4 {
		t.Fatalf("only %d orders were produced, so not every entry had its turn", seen)
	}
}

// 🔴 One kind of damage ordered on many rows spreads over every column it
// covers, rather than landing on the same one for the whole run.
//
// Drawn once per entry, damage made of letters lands on a column of identifiers
// read as text in every row it is ordered and is refused every time, while the
// columns of names beside it never get it. Forty rows over four columns leave
// any one column out with a chance well under one in a thousand.
func TestOneKindOfDamageSpreadsOverEveryColumnItCovers(t *testing.T) {
	columns := []string{"name", "city", "street", "note"}
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L2"}, Rows: "100%"},
		aCatalog(t, "L2.a.one"), aView(columns...))

	var landed []string
	for row := 0; row < 40; row++ {
		for _, order := range selector.Orders("customer", row, 40) {
			landed = append(landed, order.Field)
		}
	}
	if missed := columnsMissed(columns, landed); len(missed) > 0 {
		t.Errorf("forty orders of one kind of damage never landed on %v", missed)
	}
	// The canary is the shape of the defect: one column for every row.
	if len(columnsMissed(columns, []string{"name", "name", "name"})) == 0 {
		t.Fatal("the check does not notice damage that always lands on one column")
	}
}

// The order carries every other column the damage covers, so that a decline in
// the drawn one can move on, and never the drawn one twice. A column the policy
// excluded is not among them.
func TestAnOrderCarriesEveryOtherCoveredColumnOnce(t *testing.T) {
	selector := selectorFor(t, recipe.Policy{
		Layers: []string{"L2"}, Rows: "100%", ExcludeFields: []string{"note"},
	}, aCatalog(t, "L2.a.one"), aView("name", "city", "street", "note"))

	for row := 0; row < 20; row++ {
		for _, order := range selector.Orders("customer", row, 20) {
			places := append([]string{order.Field}, order.Instead...)
			if missed := columnsMissed([]string{"name", "city", "street"}, places); len(missed) > 0 ||
				len(places) != 3 {
				t.Fatalf("row %d carries %v, want name, city and street once each", row, places)
			}
		}
	}
}

// columnsMissed lists the columns no order landed on.
func columnsMissed(columns, landed []string) []string {
	hit := map[string]bool{}
	for _, field := range landed {
		hit[field] = true
	}
	var missed []string
	for _, column := range columns {
		if !hit[column] {
			missed = append(missed, column)
		}
	}
	return missed
}

// 🔴 When no column follows the scheme, the order points at one of the right
// type anyway, so that the refusal names the difference that is real.
//
// Pointing at the text column instead would have the manifest say the type is
// wrong, in a dataset that carries a column of exactly the right type. That
// sentence would send whoever reads it looking for the wrong thing.
func TestWithNoColumnOfThatSchemeTheOrderStillNamesAnIdentifier(t *testing.T) {
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L3"}, Rows: "100%"},
		schemeCatalog(t, "iban", "L3.checksum.off_by_one"), identifiersView())

	orders := selector.Orders("customer", 0, 20)
	if len(orders) == 0 {
		t.Fatal("no order was produced at all, which would be damage going missing")
	}
	if orders[0].Field != "ssn" && orders[0].Field != "pesel" {
		t.Errorf("the order names %q, which is not a column of the type this damage covers",
			orders[0].Field)
	}
}

// limitedView is a schema with two text columns, one of which declares a
// maximum length. It is the shape that tells a capability of the run from a
// capability of a column.
type limitedView struct{}

func (limitedView) EntityNames() []string { return []string{"customer"} }

// Five columns without a limit and one with it. The count is not decoration:
// which column damage lands on is drawn for each row, and a build that ignored
// the limit would land on one of the five in almost every row. With six columns
// and twenty rows, being right by luck is not a plausible reading.
func (limitedView) FieldsOf(string) []recipe.FieldInfo {
	fields := []recipe.FieldInfo{
		{Name: "last_name", Type: "string", MaxLength: 20, HasMaxLength: true},
	}
	for _, name := range []string{"nickname", "note", "city", "region", "title"} {
		fields = append(fields, recipe.FieldInfo{Name: name, Type: "string"})
	}
	return fields
}

func (limitedView) ParentsOf(string) []recipe.ParentInfo { return nil }
func (limitedView) RulesOf(string) []recipe.Rule         { return nil }
func (limitedView) PrimaryKeyOf(string) []string         { return nil }

// 🔴 Damage needing something of the column lands on a column that has it.
//
// The run carries schema.max-length because one column declares a limit. Aimed
// by type alone this would draw between the two and refuse half the time, on a
// column that could never carry the damage, while the one that could sat beside
// it.
func TestDamageNeedingSomethingOfTheColumnLandsWhereItIs(t *testing.T) {
	needsLimit, problems := catalog.New("0.1", "draft", []catalog.Entry{{
		ID: "L2.string.exceeds_maxlength", Layer: catalog.FieldShape, Summary: "damage",
		AppliesTo: catalog.AppliesTo{
			FieldTypes: []string{"string"}, Formats: []string{catalog.Any},
			Providers: []string{catalog.Any},
		},
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Requires: []capability.Name{capability.SchemaMaxLength},
		Example:  catalog.Example{Before: "Nowak", After: "Nowakkkkkk"},
	}})
	if len(problems) > 0 {
		t.Fatalf("the catalog for this test is not usable: %v", problems)
	}
	selector := selectorFor(t, recipe.Policy{Layers: []string{"L2"}, Rows: "100%"},
		needsLimit, limitedView{})

	for row := 0; row < 20; row++ {
		for _, order := range selector.Orders("customer", row, 20) {
			if order.Field != "last_name" {
				t.Fatalf("row %d aimed damage about a maximum length at %q", row, order.Field)
			}
		}
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
