package cli

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// TestEveryRefusalOfQuickHasWordsOfItsOwn holds the register of this surface
// against the refusals the mode can produce, in both directions.
//
// A kind with no words reaches the loud branch of quickRefusalOf, which says
// only that words are missing. Words for a kind the mode no longer has are a
// message about something nobody can be refused for.
func TestEveryRefusalOfQuickHasWordsOfItsOwn(t *testing.T) {
	for _, gap := range quickRegisterGaps(quickRefusalSays) {
		t.Error(gap)
	}
}

// TestTheGuardOfTheQuickRegisterSeesAGapInEitherDirection is the canary of the
// guard above: a register with one kind taken out and one the mode does not
// have put in, each of which the guard has to name.
func TestTheGuardOfTheQuickRegisterSeesAGapInEitherDirection(t *testing.T) {
	planted := maps.Clone(quickRefusalSays)
	delete(planted, quick.Unavailable)
	planted[quick.RefusalKind(9999)] = refusalWords{what: "x", why: "x", instead: "x"}
	if gaps := quickRegisterGaps(planted); len(gaps) != 2 {
		t.Errorf("the guard found %d gaps in a register with two planted: %q", len(gaps), gaps)
	}
}

// quickRegisterGaps is every way the register fails to answer for the mode.
func quickRegisterGaps(register map[quick.RefusalKind]refusalWords) []string {
	kinds := quick.Kinds()
	if len(kinds) == 0 {
		return []string{"the mode reports no refusals at all, so this guard would check nothing"}
	}
	var gaps []string
	for _, kind := range kinds {
		words, known := register[kind]
		if !known {
			gaps = append(gaps, fmt.Sprintf("the mode refuses as kind %d and this surface "+
				"has no words for it", kind))
			continue
		}
		if words.what == "" || words.why == "" || words.instead == "" {
			gaps = append(gaps, fmt.Sprintf("the refusal of kind %d lacks one of what, why "+
				"and instead, which every refusal here owes its reader", kind))
		}
	}
	for kind := range register {
		if !slices.Contains(kinds, kind) {
			gaps = append(gaps, fmt.Sprintf("this surface carries words for kind %d, which "+
				"the mode does not refuse", kind))
		}
	}
	return gaps
}

// TestAValueWithNoDataInTheLocaleNamesTheLocalesThatHaveIt holds the next move
// of that refusal to this command (B-378). Advice written for a recipe names a
// dataset and a field, and somebody who typed one flag has neither.
func TestAValueWithNoDataInTheLocaleNamesTheLocalesThatHaveIt(t *testing.T) {
	got := run("quick", "postal_code", "--locale", "pl-PL")

	if got.code != int(codeUsage) {
		t.Fatalf("the run ended with %d rather than %d: %s", got.code, codeUsage, got.log)
	}
	if got.data != "" {
		t.Errorf("a refused run wrote data: %q", got.data)
	}
	for _, said := range []string{"postal_code in pl-PL", labelAllowed + "en-US", "--locale"} {
		if !strings.Contains(got.log, said) {
			t.Errorf("the refusal does not say %q: %q", said, got.log)
		}
	}
	for _, unsaid := range []string{"dataset", "recipe", "field"} {
		if strings.Contains(got.log, unsaid) {
			t.Errorf("the refusal speaks of a %s, which this command does not have: %q",
				unsaid, got.log)
		}
	}
}
