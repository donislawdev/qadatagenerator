package mutators

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// exceedsMaxLengthID is the catalog entry this implements.
const exceedsMaxLengthID catalog.ID = "L2.string.exceeds_maxlength"

// The knobs the entry declares. Their defaults live in the catalog.
const (
	overshootParameter = "overshoot"
	unitParameter      = "unit"
)

// exceedsMaxLength makes a text value longer than the limit its column declares.
//
// # Why the limit comes from the column and not from the knob
//
// The entry says how far past the limit to go, and the schema says where the
// limit is. A mutator that added characters to whatever it was given would
// produce a value that is longer than it was and says nothing about the
// boundary, because the value it started from is usually well short of the
// limit. So this builds a value of exactly the declared maximum plus the
// overshoot, which is the boundary the entry is about and the place an off by
// one in somebody's length check lives.
//
// # Why the unit is asked about rather than assumed
//
// A column of thirty characters holds thirty Polish names and ten emoji, and
// the schema says which of the two it means. This build counts characters and
// bytes, which is what the generator counts, and refuses a limit declared in
// graphemes rather than counting something else and calling it that. Graphemes
// are never more numerous than characters, so a value built to overrun a
// character count can sit comfortably inside a grapheme one, and the entry
// promises rejection with hard confidence.
type exceedsMaxLength struct{}

// ID is the catalog identifier.
func (exceedsMaxLength) ID() catalog.ID { return exceedsMaxLengthID }

// Damage lengthens the value past the declared maximum.
func (exceedsMaxLength) Damage(target mutator.Target) mutator.Outcome {
	overshoot, refusal := knobsOf(target.Params)
	if refusal != "" {
		return mutator.Skipped(refusal)
	}
	if refusal := unusable(target); refusal != "" {
		return mutator.Skipped(refusal)
	}

	declared := target.Field.Unit()
	wanted := target.Field.MaxLength + overshoot
	have := lengthIn(target.Value, declared)
	if have >= wanted {
		return mutator.Skipped(fmt.Sprintf(
			"the value is already %d %s long, which is not short of %d", have, declared, wanted))
	}
	// 🔴 The overshoot has to land ON the mark, not past it, and that is the
	// whole worth of the default of one.
	//
	// A length check written with an off by one error accepts exactly one more
	// than it should, so a value one past the limit finds it and a value two
	// past does not: the buggy check rejects that one as surely as a correct
	// check would, and the defect stays hidden. Repeating a character wider
	// than one unit steps over the mark, which is how this was found: a name
	// ending in a two byte letter, three bytes short of the target, lands four
	// bytes on and looks like it worked.
	last, _ := utf8.DecodeLastRuneInString(target.Value)
	step := lengthIn(string(last), declared)
	if (wanted-have)%step != 0 {
		return mutator.Skipped(fmt.Sprintf(
			"the value ends in a character of %d %s and is %d short of %d, so repeating it "+
				"steps over the mark rather than onto it", step, declared, wanted-have, wanted))
	}
	return mutator.Done(padTo(target.Value, wanted, declared))
}

// knobsOf reads the two knobs and returns the overshoot, or the sentence
// explaining why there is nothing to work with.
//
// The unit is read here and checked against the column below, because one is a
// question about the order and the other about where it landed.
func knobsOf(params mutator.Parameters) (int, string) {
	overshoot, given := params.Whole(overshootParameter)
	if !given {
		return 0, fmt.Sprintf(
			"no whole number of %s was given, so nothing says how far past the limit to go",
			overshootParameter)
	}
	if overshoot < 1 {
		return 0, fmt.Sprintf(
			"an %s of %d would not take the value past the limit", overshootParameter, overshoot)
	}
	if _, named := params.Text(unitParameter); !named {
		return 0, fmt.Sprintf(
			"no %s was given, so nothing says what the overshoot counts", unitParameter)
	}
	return overshoot, ""
}

// unusable returns why this column and value cannot carry the damage, or
// nothing when they can.
func unusable(target mutator.Target) string {
	// Whether the column declares a limit at all is not asked here. The entry
	// requires schema.max-length, that capability is a property of a column
	// rather than of a run, and the engine checks it against the column this
	// landed on before calling. Asking again would put a second copy of a
	// published field in this file, which is the rule the type check already
	// follows.
	declared := target.Field.Unit()
	if unit, _ := target.Params.Text(unitParameter); unit != string(declared) {
		// Counting the overshoot in one unit against a limit declared in another
		// is arithmetic about two different measures. It can leave the value
		// inside the limit, and the entry promises rejection outright.
		return fmt.Sprintf(
			"the overshoot counts %s and the limit is declared in %s, and this build only "+
				"goes past a limit in the unit it was written in", unit, declared)
	}
	if declared == recipe.Graphemes {
		return "the limit is declared in graphemes, which this build cannot count, so it cannot " +
			"promise a value that goes past it"
	}
	if target.Value == "" {
		return "the value is empty, so there is no character to repeat to make it longer"
	}
	return ""
}

// lengthIn measures a value the way the schema says its column is measured.
//
// It counts the same two ways the generator counts, so a value this build
// refuses to produce for being too long is a value this mutator agrees is too
// long. Two different counts either side of one limit would be a tool arguing
// with itself.
func lengthIn(value string, unit recipe.LengthUnit) int {
	if unit == recipe.Bytes {
		return len(value)
	}
	return utf8.RuneCountInString(value)
}

// padTo repeats the last character of a value until it reaches the wanted
// length.
//
// The last character rather than a filler of our own, so the result still looks
// like the data it came from. A name that overruns its column by turning into
// Nowakkkkkk is recognisable in a report, and one that turns into Nowak plus a
// run of x characters reads as though the tool wrote it.
//
// # Why the count is worked out before the loop rather than inside it
//
// 🔴 Asking how long the value is now, on every pass, means counting characters
// over the whole of it on every pass, and that is quadratic in a length the
// recipe chooses. Measured before this was changed: doubling the limit
// multiplied the time by four, so eighty thousand characters took 939 ms and a
// limit of eight million would have taken about two and a half hours with
// nothing written and nothing said. A recipe is untrusted input, and the unit
// that pays this cost is the default one.
//
// The count is exact rather than approximate. The caller has already refused a
// gap the last character does not divide, so the division below has no
// remainder to lose.
func padTo(value string, wanted int, unit recipe.LengthUnit) string {
	last, _ := utf8.DecodeLastRuneInString(value)
	filler := string(last)
	times := (wanted - lengthIn(value, unit)) / lengthIn(filler, unit)
	var built strings.Builder
	built.Grow(len(value) + times*len(filler))
	built.WriteString(value)
	for i := 0; i < times; i++ {
		built.WriteString(filler)
	}
	return built.String()
}
