package values_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// shaped builds a recipe of one entity with the given fields.
func shaped(fields ...recipe.Field) recipe.Recipe {
	return recipe.Recipe{
		Seed:     "schema",
		Locale:   "pl-PL",
		Entities: []recipe.Entity{{Name: "customer", Count: 3, HasCount: true, Fields: fields}},
	}
}

func typeOfField(t *testing.T, view recipe.SchemaView, entity, field string) string {
	t.Helper()
	for _, info := range view.FieldsOf(entity) {
		if info.Name == field {
			return info.Type
		}
	}
	t.Fatalf("no field called %s", field)
	return ""
}

// A field described by a generator gets the type that generator produces.
//
// Without this the corruption engine sees a column of no type, and an entry
// only covers such a column when it uses the reserved word any. Two of the
// fifty two entries do, so an ordinary recipe would leave fifty of them
// inapplicable for a reason nobody chose.
func TestAFieldWithAGeneratorGetsTheTypeThatGeneratorProduces(t *testing.T) {
	source := values.New("pl-PL", "schema")
	view := values.SchemaOf(shaped(
		recipe.Field{Name: "first_name", Generator: "first_name"},
		recipe.Field{Name: "pesel", Generator: "pesel"},
		recipe.Field{Name: "born", Generator: "birth_date"},
		recipe.Field{Name: "zip", Generator: "postal_code"},
	), source)

	for field, want := range map[string]string{
		"first_name": values.TypeString,
		"pesel":      values.TypeIdentifier,
		"born":       values.TypeDate,
		"zip":        values.TypePostalCode,
	} {
		if got := typeOfField(t, view, "customer", field); got != want {
			t.Errorf("%s came out as %q, want %q", field, got, want)
		}
	}
}

// 🔴 A field drawn from a numbering scheme says which one, and a field drawn
// from a word list says none.
//
// The type cannot answer this. Every national identifier is of type identifier
// and they disagree about what they carry, so damage aimed at a check digit
// would otherwise land on a scheme that has none and promise a rejection nothing
// performs. A column of names claiming a scheme would be the same mistake from
// the other side.
func TestAFieldDrawnFromASchemeSaysWhichOne(t *testing.T) {
	source := values.New("pl-PL", "schema")
	view := values.SchemaOf(shaped(
		recipe.Field{Name: "pesel", Generator: "pesel"},
		recipe.Field{Name: "tax", Generator: "nip"},
		recipe.Field{Name: "account", Generator: "iban"},
		recipe.Field{Name: "first_name", Generator: "first_name"},
		recipe.Field{Name: "amount", Type: values.TypeInteger},
	), source)

	for field, want := range map[string]string{
		"pesel":      "PL.pesel",
		"tax":        "PL.nip",
		"account":    "iban",
		"first_name": "",
		"amount":     "",
	} {
		if got := schemeOfField(t, view, "customer", field); got != want {
			t.Errorf("%s follows %q, want %q", field, got, want)
		}
	}
}

func schemeOfField(t *testing.T, view recipe.SchemaView, entity, field string) string {
	t.Helper()
	for _, info := range view.FieldsOf(entity) {
		if info.Name == field {
			return info.Scheme
		}
	}
	t.Fatalf("no field called %s", field)
	return ""
}

// A type the recipe stated itself outranks anything worked out from a
// generator. A person who wrote the type meant it.
func TestATypeTheRecipeStatedWins(t *testing.T) {
	source := values.New("pl-PL", "schema")
	view := values.SchemaOf(shaped(
		recipe.Field{Name: "amount", Type: values.TypeInteger},
	), source)
	if got := typeOfField(t, view, "customer", "amount"); got != values.TypeInteger {
		t.Errorf("the declared type became %q", got)
	}
}

// A generator this build does not know leaves the type empty rather than
// guessing one. Guessing would aim damage at a column on the strength of a
// name, and the manifest would then promise a rejection nobody rejects.
func TestAnUnknownGeneratorLeavesTheTypeEmpty(t *testing.T) {
	source := values.New("pl-PL", "schema")
	view := values.SchemaOf(shaped(
		recipe.Field{Name: "mystery", Generator: "no_such_generator"},
	), source)
	if got := typeOfField(t, view, "customer", "mystery"); got != "" {
		t.Errorf("an unknown generator produced the type %q", got)
	}
}

// 🔴 A field drawn from a list of people's names says which part of a name it
// holds, and a field drawn from a list of towns says none.
//
// Both are text, so the type cannot tell them apart. Damage that supplies a
// real surname expects it accepted, and in a column of towns, or in a column a
// recipe calls surname without a generator behind it, that expectation would
// rest on nothing.
func TestAFieldDrawnFromNamesSaysWhichPartOfAName(t *testing.T) {
	source := values.New("pl-PL", "schema")
	view := values.SchemaOf(shaped(
		recipe.Field{Name: "given", Generator: "first_name"},
		recipe.Field{Name: "family", Generator: "last_name"},
		recipe.Field{Name: "town", Generator: "city"},
		recipe.Field{Name: "province", Generator: "region"},
		recipe.Field{Name: "sex", Generator: "gender"},
		recipe.Field{Name: "surname", Type: values.TypeString},
	), source)

	want := map[string]recipe.NamePart{
		"given": recipe.GivenName, "family": recipe.Surname,
		"town": "", "province": "", "sex": "", "surname": "",
	}
	for _, info := range view.FieldsOf("customer") {
		if info.NamePart != want[info.Name] {
			t.Errorf("%s came out holding %q, want %q", info.Name, info.NamePart, want[info.Name])
		}
	}
}

// The rest of the view is the recipe's own answer and has to survive the
// wrapping. A decorator that quietly drops a relation would leave the mutators
// that damage keys with nothing to work on.
func TestWrappingTheRecipeKeepsEverythingElseItSays(t *testing.T) {
	source := values.New("pl-PL", "schema")
	rec := shaped(recipe.Field{Name: "pesel", Generator: "pesel"})
	rec.Entities[0].PrimaryKey = []string{"pesel"}
	view := values.SchemaOf(rec, source)

	if names := view.EntityNames(); len(names) != 1 || names[0] != "customer" {
		t.Errorf("entity names came out as %v", names)
	}
	if key := view.PrimaryKeyOf("customer"); len(key) != 1 || key[0] != "pesel" {
		t.Errorf("the primary key came out as %v", key)
	}
}

// Asking about an entity that is not there answers empty rather than crashing,
// because the take and corrupt mode will ask about names read from somebody
// else's file.
func TestAskingAboutAnEntityThatIsNotThereIsSafe(t *testing.T) {
	source := values.New("pl-PL", "schema")
	view := values.SchemaOf(shaped(recipe.Field{Name: "a", Generator: "city"}), source)
	if fields := view.FieldsOf("nobody"); len(fields) != 0 {
		t.Errorf("an unknown entity answered with %d fields", len(fields))
	}
}
