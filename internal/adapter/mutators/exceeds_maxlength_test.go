package mutators_test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

const (
	exceedsMaxLength catalog.ID = "L2.string.exceeds_maxlength"
	stringEmpty      catalog.ID = "L2.string.empty"
)

// limited is a text column with a declared maximum, counted the way the schema
// says.
func limited(max int, unit recipe.LengthUnit) recipe.FieldInfo {
	return recipe.FieldInfo{
		Name: "last_name", Type: "string", MaxLength: max, HasMaxLength: true, LengthUnit: unit,
	}
}

// 🔴 The value has to end up past the DECLARED limit, not merely longer than it
// was.
//
// This is the whole content of the entry. A generated value is usually well
// short of its column's maximum, so a mutator that added a character to what it
// was handed would produce something longer and nowhere near the boundary,
// which is exactly where an off by one in somebody's length check lives.
func TestTheValueEndsUpPastTheDeclaredLimit(t *testing.T) {
	entry := entryOf(t, exceedsMaxLength)
	overshoot, whole := defaultsOf(entry).Whole("overshoot")
	if !whole || overshoot < 1 {
		t.Fatalf("the declared overshoot is %#v", defaultsOf(entry)["overshoot"])
	}

	for _, short := range []string{"Nowak", "A", "Żółć"} {
		for _, max := range []int{5, 9, 30} {
			outcome := damage(t, exceedsMaxLength, mutator.Target{
				Value: short, Field: limited(max, recipe.Characters), Params: defaultsOf(entry),
			})
			if !outcome.Applied() {
				t.Fatalf("%q against a limit of %d was left alone: %s", short, max, outcome.Reason())
			}
			if got := utf8.RuneCountInString(outcome.Value()); got != max+overshoot {
				t.Errorf("%q against a limit of %d became %d characters, want %d",
					short, max, got, max+overshoot)
			}
			if !strings.HasPrefix(outcome.Value(), short) {
				t.Errorf("%q became %q, which is not the value it started from",
					short, outcome.Value())
			}
		}
	}
}

// A limit counted in bytes is overrun in bytes. The two counts part company on
// exactly the values this tool exists to produce, so a value built to overrun
// thirty characters can sit well inside thirty bytes and the other way round.
func TestALimitCountedInBytesIsOverrunInBytes(t *testing.T) {
	// Four Polish letters, eight bytes. A limit of nine bytes leaves a gap of
	// two, which one more two byte letter fills exactly.
	outcome := damage(t, exceedsMaxLength, mutator.Target{
		Value: "Żółć", Field: limited(9, recipe.Bytes),
		Params: mutator.Parameters{"overshoot": 1, "unit": string(recipe.Bytes)},
	})
	if !outcome.Applied() {
		t.Fatalf("nothing happened: %s", outcome.Reason())
	}
	if got := len(outcome.Value()); got != 10 {
		t.Errorf("the value is %d bytes, want 10: %q", got, outcome.Value())
	}
	// 🔴 The same value is well short of the limit by the other count, which is
	// why the unit has to be carried at all rather than assumed.
	if got := utf8.RuneCountInString(outcome.Value()); got >= 10 {
		t.Errorf("the value is also %d characters, so this test proves nothing about units", got)
	}
}

// 🔴 A value that cannot land exactly on the mark is refused rather than left
// past it.
//
// The default overshoot of one is the classic boundary, and it is worth having
// only if the value lands there. A length check written with an off by one
// error accepts exactly one more than it should: a value one past finds it, and
// a value two past is rejected by the buggy check as surely as by a correct one,
// so the defect stays hidden and the run reports damage that proved nothing.
// Found by running this against a name ending in a two byte letter.
func TestAValueThatCannotLandOnTheMarkIsRefusedRatherThanOvershot(t *testing.T) {
	outcome := damage(t, exceedsMaxLength, mutator.Target{
		Value: "Żółć", Field: limited(10, recipe.Bytes),
		Params: mutator.Parameters{"overshoot": 1, "unit": string(recipe.Bytes)},
	})
	if outcome.Applied() {
		t.Fatalf("the value stepped over the mark and was reported as damaged: %q (%d bytes)",
			outcome.Value(), len(outcome.Value()))
	}
	if !strings.Contains(outcome.Reason(), "steps over the mark") {
		t.Errorf("the reason does not name the case: %s", outcome.Reason())
	}
}

func TestLengthDamageThatCannotBeDoneIsRefusedWithAReason(t *testing.T) {
	entry := entryOf(t, exceedsMaxLength)
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"the column declares no limit", mutator.Target{
			Value: "Nowak", Field: recipe.FieldInfo{Name: "last_name", Type: "string"},
			Params: defaultsOf(entry)},
			"declares no maximum length"},
		{"the overshoot is not a whole number", mutator.Target{
			Value: "Nowak", Field: limited(9, recipe.Characters),
			Params: mutator.Parameters{"overshoot": 1.5, "unit": "characters"}},
			"nothing says how far past the limit to go"},
		{"the overshoot is nought", mutator.Target{
			Value: "Nowak", Field: limited(9, recipe.Characters),
			Params: mutator.Parameters{"overshoot": 0, "unit": "characters"}},
			"would not take the value past the limit"},
		{"the overshoot counts another unit than the limit", mutator.Target{
			Value: "Nowak", Field: limited(9, recipe.Bytes),
			Params: mutator.Parameters{"overshoot": 1, "unit": "characters"}},
			"only goes past a limit in the unit it was written in"},
		{"the limit is counted in graphemes", mutator.Target{
			Value: "Nowak", Field: limited(9, recipe.Graphemes),
			Params: mutator.Parameters{"overshoot": 1, "unit": string(recipe.Graphemes)}},
			"which this build cannot count"},
		{"the value is empty", mutator.Target{
			Value: "", Field: limited(9, recipe.Characters), Params: defaultsOf(entry)},
			"no character to repeat"},
		{"the value already passes the limit", mutator.Target{
			Value: "Nowakkkkkkkkkk", Field: limited(5, recipe.Characters), Params: defaultsOf(entry)},
			"which is not short of"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			outcome := damage(t, exceedsMaxLength, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %q", outcome.Value())
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}

// The fifth published example this build produces rather than only checks. The
// entry shows a five character name against a limit its own note implies, so
// the limit is what makes the pair reproducible.
func TestTheCatalogExampleIsWhatTheLengthMutatorProduces(t *testing.T) {
	entry := entryOf(t, exceedsMaxLength)
	overshoot, _ := defaultsOf(entry).Whole("overshoot")
	limit := utf8.RuneCountInString(entry.Example.After) - overshoot

	outcome := damage(t, exceedsMaxLength, mutator.Target{
		Value: entry.Example.Before, Field: limited(limit, recipe.Characters),
		Params: defaultsOf(entry),
	})
	if !outcome.Applied() {
		t.Fatalf("the published example was left alone: %s", outcome.Reason())
	}
	if outcome.Value() != entry.Example.After {
		t.Errorf("the specification promises %s -> %s and this build produces %s",
			entry.Example.Before, entry.Example.After, outcome.Value())
	}
}

// Emptying a value is the one damage whose result is a value rather than the
// absence of one, and the difference is what the entry exists to test.
func TestAnEmptiedValueIsEmptyRatherThanUntouched(t *testing.T) {
	outcome := damage(t, stringEmpty, mutator.Target{
		Value: "Nowak", Field: recipe.FieldInfo{Name: "last_name", Type: "string"},
	})
	if !outcome.Applied() {
		t.Fatalf("nothing happened: %s", outcome.Reason())
	}
	if outcome.Value() != "" {
		t.Errorf("the value became %q", outcome.Value())
	}
}

// A column that is already empty is left alone with a reason. A share of empty
// values is a thing recipes ask for, so this is an ordinary case rather than a
// fault.
func TestAValueThatIsAlreadyEmptyIsRefusedWithAReason(t *testing.T) {
	outcome := damage(t, stringEmpty, mutator.Target{
		Value: "", Field: recipe.FieldInfo{Name: "last_name", Type: "string"},
	})
	if outcome.Applied() {
		t.Fatal("an empty value was reported as emptied")
	}
	if !strings.Contains(outcome.Reason(), "already empty") {
		t.Errorf("the reason does not say why: %s", outcome.Reason())
	}
}

// The published example of the empty entry is a value going to nothing, and the
// reader has to keep that difference rather than reading it as a missing key.
func TestTheCatalogExampleIsWhatTheEmptyMutatorProduces(t *testing.T) {
	entry := entryOf(t, stringEmpty)
	outcome := damage(t, stringEmpty, mutator.Target{
		Value: entry.Example.Before, Field: recipe.FieldInfo{Name: "last_name", Type: "string"},
	})
	if !outcome.Applied() {
		t.Fatalf("the published example was left alone: %s", outcome.Reason())
	}
	if outcome.Value() != entry.Example.After {
		t.Errorf("the specification promises %q -> %q and this build produces %q",
			entry.Example.Before, entry.Example.After, outcome.Value())
	}
}
