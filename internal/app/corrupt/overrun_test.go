package corrupt_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

const growingID catalog.ID = "UNI.grows.value"

// bile is the Polish word for bile written with four letters that each take two
// bytes in UTF-8: four characters and eight bytes.
const bile = "\U0000017B\U000000F3\U00000142\U00000107"

// growing is an engine whose one damage answers with the value it is told to,
// the way a mutator that adds an accent or an emoji makes the value longer.
func growing(t *testing.T, result string) corrupt.Engine {
	t.Helper()
	return corrupt.New(aCatalog(t, growingID), registry{growingID: stated{growingID, mutator.Done(result)}},
		noSchemes{}, capability.NewSet(), seed.New("overrun"))
}

func growingOrder() corrupt.Order {
	return corrupt.Order{ID: "d1", Mutator: growingID, Entity: "person", Row: 1, Field: "name"}
}

func limited(limit int, unit recipe.LengthUnit) recipe.FieldInfo {
	return recipe.FieldInfo{Name: "name", Type: "string", MaxLength: limit, HasMaxLength: true,
		LengthUnit: unit}
}

// 🔴 Damage that would leave a value longer than its column allows is refused,
// with the length and the limit in the reason, and the value stays as it was.
//
// The column refuses such a value for its length, rightly, so an entry about
// something else would promise an answer the application gives for another
// reason. Measured against the entry about the two Unicode forms: Zolc with its
// accents is four characters composed and seven decomposed, and the entry
// promised acceptance with hard confidence.
func TestDamageThatWouldOverrunTheColumnIsRefused(t *testing.T) {
	value, entry := growing(t, bile+"!").Apply(growingOrder(), limited(4, recipe.Characters), bile,
		seed.Coordinate{})
	if entry.Applied || value != bile {
		t.Fatalf("a five character result was written into a column of four: %q", value)
	}
	if entry.FailureKind != manifest.DeclinedByMutator {
		t.Errorf("the refusal is %s, want %s", entry.FailureKind, manifest.DeclinedByMutator)
	}
	for _, want := range []string{"5 characters long", "at most 4", "for its length"} {
		if !strings.Contains(entry.FailureReason, want) {
			t.Errorf("the reason does not say %q: %s", want, entry.FailureReason)
		}
	}
}

// The edge belongs to the value: a result exactly as long as the limit is
// written, and the limit is counted in the unit the column declares.
func TestDamageUpToTheLimitInTheColumnsUnitIsWritten(t *testing.T) {
	_, atTheLimit := growing(t, bile).Apply(growingOrder(), limited(4, recipe.Characters), "Zolc",
		seed.Coordinate{})
	if !atTheLimit.Applied {
		t.Errorf("a result of four characters was refused by a limit of four: %s",
			atTheLimit.FailureReason)
	}

	// The same four characters are eight bytes.
	_, inBytes := growing(t, bile).Apply(growingOrder(), limited(6, recipe.Bytes), "Zolc",
		seed.Coordinate{})
	if inBytes.Applied {
		t.Fatal("eight bytes were written into a column of six bytes")
	}
	if !strings.Contains(inBytes.FailureReason, "8 bytes long") {
		t.Errorf("the reason does not count in bytes: %s", inBytes.FailureReason)
	}

	// No limit, nothing to overrun.
	_, unlimited := growing(t, bile+bile+bile).Apply(growingOrder(),
		recipe.FieldInfo{Name: "name", Type: "string"}, "Zolc", seed.Coordinate{})
	if !unlimited.Applied {
		t.Errorf("a column that declares no limit refused a long result: %s", unlimited.FailureReason)
	}
}

// 🔴 Damage that needs the column's limit is about the limit, and going past it
// is what it does, so it is left to.
func TestDamageAboutTheLimitIsLeftToGoPastIt(t *testing.T) {
	aboutTheLimit := entry()
	aboutTheLimit.AppliesTo.FieldTypes = []string{"string"}
	aboutTheLimit.Requires = []capability.Name{capability.SchemaMaxLength}
	engine := corrupt.New(described(t, aboutTheLimit),
		registry{damageID: stated{damageID, mutator.Done("Nowakk")}}, noSchemes{},
		capability.NewSet(capability.SchemaMaxLength), seed.New("overrun"))

	value, done := engine.Apply(order(), limited(5, recipe.Characters), "Nowak", seed.Coordinate{})
	if !done.Applied || value != "Nowakk" {
		t.Errorf("damage about the limit was kept inside it: %s", done.FailureReason)
	}
}

// A limit in graphemes is counted as characters, and the reason says so rather
// than calling a count of characters a count of graphemes.
func TestALimitInGraphemesIsCountedAsCharactersAndSaysSo(t *testing.T) {
	_, entry := growing(t, bile+"!").Apply(growingOrder(), limited(4, recipe.Graphemes), bile,
		seed.Coordinate{})
	if entry.Applied {
		t.Fatal("five characters were written into a column of four graphemes")
	}
	for _, want := range []string{"5 characters long", "at most 4 graphemes", "counts a grapheme as a character"} {
		if !strings.Contains(entry.FailureReason, want) {
			t.Errorf("the reason does not say %q: %s", want, entry.FailureReason)
		}
	}
}

// overrunRow answers for a row whose columns declare their own limits.
type overrunRow struct {
	values map[string]string
	limits map[string]int
}

func (r overrunRow) cell(field string) (corrupt.Cell, bool) {
	value, held := r.values[field]
	info := recipe.FieldInfo{Name: field, Type: "string"}
	if limit, declared := r.limits[field]; declared {
		info.MaxLength, info.HasMaxLength = limit, true
	}
	return corrupt.Cell{Field: info, Value: value}, held
}

// 🔴 An overrun moves the damage on to another column of the row that can hold
// it, the way a value holding no letter does, because it follows from the
// length of the value and the limit of the column and says nothing about the
// application.
func TestDamageThatWouldOverrunMovesToAColumnThatHoldsIt(t *testing.T) {
	row := overrunRow{
		values: map[string]string{"first": "Anna", "city": "Krakow"},
		limits: map[string]int{"first": 4},
	}
	order := letteredOrder(1, "first", "city")
	entry, change, wrote := letteredEngine(t).ApplyInRow(order, row.cell)
	if !entry.Applied || !wrote {
		t.Fatalf("a column without a limit sat beside the overrun and the damage was refused: %s",
			entry.FailureReason)
	}
	if change.Field != "city" || change.Value != "Krakow!" {
		t.Errorf("the change is %+v, and only city can hold a longer value", change)
	}

	// With nowhere else to go, the refusal is the overrun in the column drawn.
	alone := overrunRow{values: map[string]string{"first": "Anna"}, limits: map[string]int{"first": 4}}
	entry, _, wrote = letteredEngine(t).ApplyInRow(letteredOrder(1, "first"), alone.cell)
	if entry.Applied || wrote || !strings.Contains(entry.FailureReason, "at most 4") {
		t.Errorf("the overrun with no other column came back as %+v", entry)
	}
}
