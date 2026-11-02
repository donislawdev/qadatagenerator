package values

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// The payment card is the one scheme the registry draws as a list, and the
// provider draws on its own as well. Both are reached: a recipe through the
// registry, the corruption engine and its tests through the provider. So the
// two have to give the same number for the same row, or a dataset and the
// tests about it would be about different numbers.
func TestTheRegistryAndTheCardDrawTheSameNumber(t *testing.T) {
	const from = "card-registry-2026-11"
	source := New("en-US", from)
	for row := int64(0); row < 500; row++ {
		at := seed.Coordinate{Entity: "payment", Row: row, Field: "card", Purpose: "value"}
		registry, err := source.Produce(generate.Request{Generator: Card, At: at})
		if err != nil {
			t.Fatalf("row %d: the registry refused: %v", row, err)
		}
		own, err := (intl.Card{}).Generate(provider.Request{Source: seed.New(from), At: at})
		if err != nil {
			t.Fatalf("row %d: the provider refused: %v", row, err)
		}
		if registry != own {
			t.Fatalf("row %d: the registry drew %s and the provider %s", row, registry, own)
		}
	}
}

// A unique column of cards can hold eight rows and no more, and the plan says
// so before a row exists rather than running out of luck on the ninth.
func TestACardColumnKnowsHowManyNumbersItHas(t *testing.T) {
	domain, known := New("pl-PL", "x").DomainOf(generate.Request{Generator: Card})
	if !known {
		t.Fatal("the numbers of the payment card are a list this build holds, and their domain is unknown")
	}
	if want := int64(len((intl.Card{}).Published())); domain.Size != want {
		t.Errorf("the domain holds %d numbers where the list holds %d", domain.Size, want)
	}
}

// A field limited to fifteen characters takes the American Express numbers
// alone, by choosing among the list rather than cutting a number short, and a
// field too short for any of them is refused with a reason.
func TestALimitOnLengthChoosesAmongTheCards(t *testing.T) {
	source := New("en-US", "card-limit")
	for row := int64(0); row < 200; row++ {
		value, err := source.Produce(generate.Request{
			Generator: Card, MaxLength: 15, HasMaxLength: true, Unit: string(recipe.Characters),
			At: seed.Coordinate{Entity: "payment", Row: row, Field: "card", Purpose: "value"},
		})
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		if len(value) != 15 {
			t.Fatalf("row %d drew %s into a field of fifteen", row, value)
		}
	}
	_, err := source.Produce(generate.Request{
		Generator: Card, MaxLength: 14, HasMaxLength: true, Unit: string(recipe.Characters),
		At: seed.Coordinate{Entity: "payment", Row: 0, Field: "card", Purpose: "value"},
	})
	if err == nil {
		t.Error("a field of fourteen characters was given a card number, and none of them is that short")
	}
}
