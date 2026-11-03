package values

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// valuesANestingIsCheckedOn is how many values of each scheme are drawn to hold
// a declared nesting against. The same count the masks are checked on, for the
// same reason: enough rows that every branch of a generator is reached.
const valuesANestingIsCheckedOn = 48

// 🔴 A scheme saying it lies inside another is a claim the inference acts on
// (DEC-269), so it is held against values rather than taken on trust.
//
// Three things, in both directions. Every scheme named has to be one this build
// carries and not the scheme itself. Every value the narrower scheme produces
// has to be accepted by every scheme it names, or the choice would put a column
// under a name whose values the wider reading refuses. And the wider scheme has
// to produce at least one value the narrower refuses, or the two are one set
// under two names and choosing between them is the arbitrary choice the
// refusal was written against.
//
// The registry is walked rather than a list written here, so a nesting
// declared tomorrow is held to this without anybody adding it.
func TestEveryNarrowerSchemeLiesStrictlyInsideTheSchemesItNames(t *testing.T) {
	source := New("pl-PL", "nesting")
	declared := 0
	for _, scheme := range source.Schemes() {
		narrower, ok := scheme.(provider.Narrower)
		if !ok {
			continue
		}
		declared++
		within := narrower.Within()
		if len(within) == 0 {
			t.Errorf("%s says it lies inside other schemes and names none", scheme.ID())
		}
		for _, wider := range within {
			if wider.ID() == scheme.ID() {
				t.Errorf("%s names itself as a scheme it lies inside", scheme.ID())
				continue
			}
			if _, carried := source.ProviderNamed(wider.ID()); !carried {
				t.Errorf("%s names %s, which this build does not carry", scheme.ID(), wider.ID())
				continue
			}
			for _, value := range drawnFrom(t, scheme) {
				if verdict := wider.Validate(value); !verdict.Valid {
					t.Errorf("%s produced %s, which %s refuses, so it does not lie inside it: %s",
						scheme.ID(), value, wider.ID(), verdict.Reason)
				}
			}
			refused := 0
			for _, value := range drawnFrom(t, wider) {
				if !scheme.Validate(value).Valid {
					refused++
				}
			}
			if refused == 0 {
				t.Errorf("every value %s produced was accepted by %s, so nothing here shows the "+
					"two are different sets", wider.ID(), scheme.ID())
			}
		}
	}
	if declared == 0 {
		t.Fatal("no scheme this build carries declares a nesting, so nothing above ran")
	}
}

// drawnFrom returns values of one scheme, built by its own generator.
func drawnFrom(t *testing.T, scheme provider.Provider) []string {
	t.Helper()
	drawn := make([]string, 0, valuesANestingIsCheckedOn)
	for row := int64(0); row < valuesANestingIsCheckedOn; row++ {
		value, err := scheme.Generate(provider.Request{
			Source: seed.New("nesting"),
			At:     seed.Coordinate{Entity: "records", Row: row, Field: scheme.ID(), Purpose: "value"},
		})
		if err != nil {
			t.Fatalf("%s refused to produce row %d: %v", scheme.ID(), row, err)
		}
		drawn = append(drawn, value)
	}
	return drawn
}
