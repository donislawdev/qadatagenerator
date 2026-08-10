package values_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// byType builds the request for a field a recipe described by type alone.
func byType(kind, field string, row int64) generate.Request {
	return generate.Request{Type: kind, At: at(field, row)}
}

// A field declared as a number produces a number. Until 2026-08-10 it produced
// nothing at all and the plan said nothing either, so a declared column arrived
// empty with no part of the run mentioning it.
func TestAFieldDeclaredAsANumberIsFilled(t *testing.T) {
	source := values.New("pl-PL", "seed")
	for _, kind := range []string{values.TypeInteger, values.TypeDecimal} {
		got, err := source.Produce(byType(kind, "amount", 1))
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if got == "" {
			t.Fatalf("%s produced nothing, which is the silence this closes", kind)
		}
		if _, err := strconv.ParseFloat(got, 64); err != nil {
			t.Errorf("%s produced %q, which is not a number", kind, got)
		}
	}
}

// A decimal carries two places on every row. A varying number of places is a
// different value on every row for a reason the reader cannot see.
func TestADecimalAlwaysCarriesTwoPlaces(t *testing.T) {
	source := values.New("en-US", "seed")
	for row := int64(0); row < 50; row++ {
		got, err := source.Produce(byType(values.TypeDecimal, "rate", row))
		if err != nil {
			t.Fatalf("row %d: %v", row, err)
		}
		point := strings.IndexByte(got, '.')
		if point < 0 {
			t.Fatalf("row %d produced %q with no decimal point", row, got)
		}
		if places := len(got) - point - 1; places != 2 {
			t.Errorf("row %d produced %q with %d places, want 2", row, got, places)
		}
	}
}

// The same address gives the same number every time, because the whole product
// stands on that.
func TestANumberIsTheSameForTheSameAddress(t *testing.T) {
	first := values.New("pl-PL", "repeat")
	second := values.New("pl-PL", "repeat")
	for row := int64(0); row < 20; row++ {
		left, err := first.Produce(byType(values.TypeInteger, "amount", row))
		if err != nil {
			t.Fatal(err)
		}
		right, err := second.Produce(byType(values.TypeInteger, "amount", row))
		if err != nil {
			t.Fatal(err)
		}
		if left != right {
			t.Fatalf("row %d gave %q and then %q", row, left, right)
		}
	}
}

// A different seed gives different numbers, or the address is not being used
// and the test above would pass against a constant.
func TestADifferentSeedMovesTheNumbers(t *testing.T) {
	left := values.New("pl-PL", "one")
	right := values.New("pl-PL", "two")
	same := 0
	const rows = 30
	for row := int64(0); row < rows; row++ {
		a, err := left.Produce(byType(values.TypeInteger, "amount", row))
		if err != nil {
			t.Fatal(err)
		}
		b, err := right.Produce(byType(values.TypeInteger, "amount", row))
		if err != nil {
			t.Fatal(err)
		}
		if a == b {
			same++
		}
	}
	if same == rows {
		t.Fatal("two seeds produced identical numbers on every row, so the seed is not reaching the value")
	}
}

// A number never overruns the column it sits in. A value that fits by our own
// arithmetic and is rejected on arrival is the failure the unit on max_length
// exists to prevent.
func TestANumberHonoursTheLengthOfItsColumn(t *testing.T) {
	source := values.New("en-US", "limits")
	for _, limit := range []int{1, 2, 3, 4, 8} {
		for row := int64(0); row < 25; row++ {
			request := byType(values.TypeInteger, "amount", row)
			request.MaxLength, request.HasMaxLength = limit, true
			request.Unit = string(recipe.Characters)
			got, err := source.Produce(request)
			if err != nil {
				t.Fatalf("limit %d row %d: %v", limit, row, err)
			}
			if len(got) > limit {
				t.Errorf("limit %d produced %q, which is %d long", limit, got, len(got))
			}
		}
	}
}

// A decimal with no room for a point and two places is refused with the
// arithmetic in it, rather than producing something that does not fit.
func TestADecimalWithNoRoomForItsPlacesIsRefused(t *testing.T) {
	source := values.New("en-US", "narrow")
	request := byType(values.TypeDecimal, "rate", 0)
	request.MaxLength, request.HasMaxLength = 2, true
	request.Unit = string(recipe.Characters)
	if _, err := source.Produce(request); err == nil {
		t.Fatal("a decimal was produced for a column two characters wide")
	}
}

// A type this build does not fill is refused by name rather than answered with
// an empty value, so that the caller cannot mistake one for the other.
func TestATypeThisBuildDoesNotFillIsRefusedByName(t *testing.T) {
	source := values.New("en-US", "seed")
	_, err := source.Produce(byType("money", "price", 0))
	if err == nil {
		t.Fatal("a value was produced for a type this build does not fill")
	}
	if !strings.Contains(err.Error(), "money") {
		t.Errorf("the refusal does not name the type: %v", err)
	}
}

// ProducesType and Produce agree. Two answers about the same question are a
// place where a run says one thing to the plan and does another.
func TestTheAnswerAboutATypeMatchesWhatProducingItDoes(t *testing.T) {
	source := values.New("pl-PL", "agree")
	for _, kind := range []string{values.TypeInteger, values.TypeDecimal, values.TypeString, "money", ""} {
		_, err := source.Produce(byType(kind, "f", 0))
		produced := err == nil
		if says := source.ProducesType(kind); says != produced {
			t.Errorf("type %q: ProducesType says %v and producing it %v", kind, says, produced)
		}
	}
}
