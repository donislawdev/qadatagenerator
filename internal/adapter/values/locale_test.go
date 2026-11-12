package values

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// TestEveryGeneratorDependsOnTheLocaleExactlyWhenTheRegisterSays closes the
// answer about the locale from both sides, by measurement rather than by trust.
//
// # What depending on the locale means here
//
// That choosing another locale can change what a person gets from the
// generator: another value at the same seed and address, a value where there
// was a refusal, or another set of roles the value takes account of. A screen
// leaves its choice of locale out beside a generator this calls independent
// (B-367), so a wrong "independent" hides a choice that matters and a wrong
// "dependent" offers one that does nothing.
//
// # Why nothing here can be green by measuring nothing
//
// The answers of every locale are compared whole, so an answer that ignored the
// locale would call every generator independent, and one that carried the
// locale in it would call every generator dependent. The register holds both
// kinds, and the test requires the measurement to find both, so either fault in
// the measurement fails here rather than blessing the register.
func TestEveryGeneratorDependsOnTheLocaleExactlyWhenTheRegisterSays(t *testing.T) {
	locales := Locales()
	if len(locales) < 2 {
		t.Fatalf("this build carries %d locales, and a dependence on the locale cannot be "+
			"measured with fewer than two", len(locales))
	}
	found := map[bool]bool{}
	source := New("", "")
	for _, generator := range source.Names() {
		moved := false
		first := answerIn(generator, locales[0])
		for _, locale := range locales[1:] {
			if answerIn(generator, locale) != first {
				moved = true
			}
		}
		found[moved] = true

		claimed := source.DependsOnLocale(generator)
		if moved && !claimed {
			t.Errorf("%s gives something different in another locale and the register calls it "+
				"independent, so a screen hides a choice that changes the value", generator)
		}
		if claimed && !moved {
			t.Errorf("%s gives the same in every locale and the register says it depends on the "+
				"locale, so a screen offers a choice that does nothing", generator)
		}
	}
	if !found[true] || !found[false] {
		t.Errorf("the measurement found only generators that %s, and this build has both kinds, "+
			"so the measurement is what is wrong", map[bool]string{
			true: "move with the locale", false: "stay the same in every locale",
		}[found[true]])
	}
}

// localeRows is how many values of each generator are compared between locales.
// One value is an anecdote: a list of two words shared by two locales would
// agree on it half the time.
const localeRows = 16

// answerIn is everything a person can get from a generator in one locale,
// written as one string so that the answers of two locales can be compared.
//
// A refusal is written as the word alone rather than as its message, because
// the message names the locale and two refusals would differ for that reason
// only.
func answerIn(generator, locale string) string {
	source := New(locale, "locale-register")
	can, _ := source.CanProduce(generator, locale)
	var answer strings.Builder
	fmt.Fprintf(&answer, "produces %v, reads %v\n", can, source.RolesRead(generator))
	for row := int64(0); row < localeRows; row++ {
		at := seed.Coordinate{Entity: "quick", Row: row, Field: generator, Purpose: "value"}
		value, err := source.Produce(generate.Request{Generator: generator, At: at})
		if err != nil {
			answer.WriteString("refused\n")
			continue
		}
		answer.WriteString(value + "\n")
	}
	return answer.String()
}
