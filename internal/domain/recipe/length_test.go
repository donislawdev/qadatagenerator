package recipe_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// zolc is the Polish word for bile, four letters each wider than one byte in
// UTF-8: four characters and eight bytes.
const zolc = "\U0000017B\U000000F3\U00000142\U00000107"

// family is one symbol to a reader and seven code points: four people joined
// by three zero width joiners.
const family = "\U0001F469\U0000200D\U0001F469\U0000200D\U0001F467\U0000200D\U0001F466"

// Each unit counts what its name says, and graphemes are counted as characters,
// which is the gap the count names rather than hides.
func TestEveryUnitCountsWhatItsNameSays(t *testing.T) {
	cases := []struct {
		unit  recipe.LengthUnit
		value string
		want  int
	}{
		{recipe.Characters, zolc, 4},
		{recipe.Bytes, zolc, 8},
		{recipe.Graphemes, zolc, 4},
		{recipe.Characters, family, 7},
		{recipe.Bytes, family, 25},
		// One symbol to a reader, counted as the seven characters it is made of.
		{recipe.Graphemes, family, 7},
		{recipe.Characters, "", 0},
		{recipe.Bytes, "", 0},
	}
	for _, c := range cases {
		if got := c.unit.Count(c.value); got != c.want {
			t.Errorf("%s counts %q as %d, want %d", c.unit, c.value, got, c.want)
		}
	}
}

// 🔴 A value exactly at the limit fits and one past it does not, in the unit
// the field declares. The edge is where an off by one would make the engine
// refuse damage that fits, or let through a value the column refuses.
func TestAValueFitsUpToTheLimitAndNotPastIt(t *testing.T) {
	inCharacters := recipe.FieldInfo{Name: "name", MaxLength: 4, HasMaxLength: true}
	if !inCharacters.Fits(zolc) {
		t.Errorf("four characters do not fit a limit of four characters")
	}
	if inCharacters.Fits(zolc + "a") {
		t.Errorf("five characters fit a limit of four characters")
	}

	inBytes := recipe.FieldInfo{Name: "name", MaxLength: 7, HasMaxLength: true, LengthUnit: recipe.Bytes}
	if inBytes.Fits(zolc) {
		t.Errorf("eight bytes fit a limit of seven bytes, so the unit was not read")
	}
	inBytes.MaxLength = 8
	if !inBytes.Fits(zolc) {
		t.Errorf("eight bytes do not fit a limit of eight bytes")
	}

	// A limit of nought is a limit, and only the empty value is within it.
	nought := recipe.FieldInfo{Name: "name", MaxLength: 0, HasMaxLength: true}
	if !nought.Fits("") || nought.Fits("a") {
		t.Errorf("a limit of nought holds the empty value %v and one letter %v",
			nought.Fits(""), nought.Fits("a"))
	}

	unlimited := recipe.FieldInfo{Name: "name"}
	if !unlimited.Fits(zolc + zolc + zolc + family) {
		t.Errorf("a field that declares no limit refused a value")
	}
}
