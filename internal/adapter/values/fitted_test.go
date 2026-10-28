package values_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// limited is one field drawn from a word list under a limit on length.
type limited struct {
	generator string
	gender    string
	limit     int
	unit      recipe.LengthUnit
}

// rowsOf draws forty values of a field from a source, one per row.
func rowsOf(t *testing.T, source values.Source, ask limited) []string {
	t.Helper()
	bound := map[string]string{}
	if ask.gender != "" {
		bound[provider.RoleGender] = ask.gender
	}
	drawn := make([]string, 0, 40)
	for row := int64(0); row < 40; row++ {
		value, err := source.Produce(generate.Request{
			Generator: ask.generator,
			At:        seed.Coordinate{Entity: "person", Row: row, Field: "f", Purpose: "value"},
			Agrees:    provider.NewAgreement(bound),
			MaxLength: ask.limit, HasMaxLength: true, Unit: string(ask.unit),
		})
		if err != nil {
			t.Fatalf("%+v was refused at row %d: %v", ask, row, err)
		}
		drawn = append(drawn, value)
	}
	return drawn
}

// 🔴 A list under a limit is the same list whichever limit, unit or list the
// source was asked about before it.
//
// A source remembers the entries of a list that fit a limit, so that a field
// with a limit no longer measures the whole list for every value (found by an
// outside review on 2026-10-22). A memory keyed by too little hands one field
// the list of another, and the values stay plausible names: nothing else would
// notice a surname drawn from the list of another limit. So every ask on one
// source is held against the same ask on a source asked nothing before.
func TestAListUnderALimitIsTheSameWhicheverLimitWasAskedFirst(t *testing.T) {
	asks := []limited{
		{values.LastName, "female", 8, recipe.Characters},
		{values.LastName, "female", 8, recipe.Bytes},
		{values.LastName, "male", 8, recipe.Characters},
		{values.LastName, "female", 12, recipe.Characters},
		{values.FirstName, "", 5, recipe.Characters},
		{values.City, "", 9, recipe.Characters},
		{values.LastName, "female", 8, recipe.Characters},
	}
	fresh := func(ask limited) []string { return rowsOf(t, values.New("pl-PL", "fitted"), ask) }

	// The fixture has to be able to tell the asks apart, or the comparison
	// below would pass for a memory keyed by nothing at all.
	first := strings.Join(fresh(asks[0]), ",")
	for _, other := range asks[1:4] {
		if strings.Join(fresh(other), ",") == first {
			t.Fatalf("%+v draws the same values as %+v, so this cannot tell them apart", other, asks[0])
		}
	}

	shared := values.New("pl-PL", "fitted")
	for _, ask := range asks {
		got, want := rowsOf(t, shared, ask), fresh(ask)
		for row := range want {
			if got[row] != want[row] {
				t.Errorf("%+v drew %q at row %d after other asks, and %q on a fresh source",
					ask, got[row], row, want[row])
				break
			}
		}
		for _, value := range got {
			if measured := measuredIn(value, ask.unit); measured > ask.limit {
				t.Errorf("%+v drew %q, which is %d long", ask, value, measured)
				break
			}
		}
	}
}

func measuredIn(value string, unit recipe.LengthUnit) int {
	if unit == recipe.Bytes {
		return len(value)
	}
	return utf8.RuneCountInString(value)
}

// A value under a limit is drawn without measuring the list again. Before
// 2026-10-28 every value built the list of entries that fit anew, about a dozen
// allocations a value for the Polish surnames, so a value under a limit is held
// to what a value without one costs.
func TestAValueUnderALimitIsDrawnWithoutMeasuringTheListAgain(t *testing.T) {
	source := values.New("pl-PL", "fitted")
	cost := func(hasLimit bool) float64 {
		req := generate.Request{
			Generator: values.LastName,
			At:        seed.Coordinate{Entity: "person", Row: 3, Field: "f", Purpose: "value"},
			Agrees:    provider.NewAgreement(map[string]string{provider.RoleGender: "female"}),
		}
		if hasLimit {
			req.HasMaxLength, req.MaxLength, req.Unit = true, 8, string(recipe.Characters)
		}
		if _, err := source.Produce(req); err != nil {
			t.Fatalf("the first draw was refused: %v", err)
		}
		return testing.AllocsPerRun(50, func() { _, _ = source.Produce(req) })
	}
	free, underLimit := cost(false), cost(true)
	if underLimit > free+1 {
		t.Errorf("a value under a limit costs %.0f allocations and one without a limit %.0f",
			underLimit, free)
	}
}
