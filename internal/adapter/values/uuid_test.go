package values_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// A field described as type uuid is the uuid generator under another name: the
// same value at the same place, a valid UUID, and a type the editor offers and
// the plan fills rather than leaving the column empty.
func TestTheTypeUUIDIsTheGeneratorUUID(t *testing.T) {
	source := values.New("en-US", "uuid")
	if !source.ProducesType(values.TypeUUID) || !slices.Contains(source.FilledTypes(), values.TypeUUID) {
		t.Fatalf("type uuid is not among the types this build fills: %v", source.FilledTypes())
	}
	if filler, filled := source.FillerOf(values.TypeUUID); !filled || filler != values.UUID {
		t.Fatalf("type uuid is filled by %q (%v) rather than by the uuid generator", filler, filled)
	}
	if _, filled := source.FillerOf(values.TypeInteger); filled {
		t.Error("an integer claims a generator, and it is filled by arithmetic of its own")
	}
	for row := range int64(50) {
		typed, err := source.Produce(generate.Request{Type: values.TypeUUID, At: at("id", row)})
		if err != nil {
			t.Fatalf("row %d: type uuid refused: %v", row, err)
		}
		named, err := source.Produce(generate.Request{Generator: values.UUID, At: at("id", row)})
		if err != nil {
			t.Fatalf("row %d: generator uuid refused: %v", row, err)
		}
		if typed != named {
			t.Fatalf("row %d: type uuid gave %s and generator uuid %s at the same place", row, typed, named)
		}
		if verdict := (intl.UUID{}).Validate(typed); !verdict.Valid {
			t.Fatalf("row %d: %s is not a UUID (%s)", row, typed, verdict.Reason)
		}
	}
}

// A UUID is 36 characters, so a limit below that is refused before a row is
// written, and so are bounds, which belong to numbers.
func TestATypeUUIDRefusesALimitItCannotFitAndBounds(t *testing.T) {
	source := values.New("en-US", "uuid")
	short := generate.Request{Type: values.TypeUUID, At: at("id", 0), HasMaxLength: true, MaxLength: 35, Unit: "characters"}
	err := source.TypeFits(short)
	if err == nil || !strings.Contains(err.Error(), "36") {
		t.Errorf("a limit of 35 was answered with %v, which does not name the 36 a UUID needs", err)
	}
	enough := short
	enough.MaxLength = 36
	if err := source.TypeFits(enough); err != nil {
		t.Errorf("a limit of 36 was refused: %v", err)
	}
	bounded := generate.Request{Type: values.TypeUUID, At: at("id", 0), MinValue: "1"}
	if err := source.TypeFits(bounded); err == nil {
		t.Error("min_value on a UUID was accepted, and it bounds numbers only")
	}
}

// The corruption engine reads a field declared as type uuid as the column of
// the uuid generator: an identifier following the scheme uuid. The type the
// recipe wrote is not a word of the catalog, so left as it stands it would make
// every entry inapplicable to the column.
func TestAFieldDeclaredAsUUIDIsAnIdentifierOfTheSchemeUUID(t *testing.T) {
	source := values.New("en-US", "schema")
	view := values.SchemaOf(shaped(
		recipe.Field{Name: "id", Type: values.TypeUUID},
		recipe.Field{Name: "key", Generator: values.UUID},
	), source)
	for _, field := range []string{"id", "key"} {
		if got := typeOfField(t, view, "customer", field); got != values.TypeIdentifier {
			t.Errorf("%s came out as type %q, want %q", field, got, values.TypeIdentifier)
		}
		if got := schemeOfField(t, view, "customer", field); got != values.UUID {
			t.Errorf("%s follows %q, want the scheme uuid", field, got)
		}
	}
}
