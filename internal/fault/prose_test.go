package fault_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// 🔴 Fragments joined into prose begin every later sentence with a capital,
// and nothing taken from a file is changed on the way (B-352).
//
// The case that went wrong is the first row: two problems of a manifest read
// back from disk, joined by a full stop alone. The cases below it are why the
// letter is raised only when the first word is lower case letters alone. The
// identifier of a manifest entry leads its own problems, followed by a colon,
// and a capital there would be a different identifier from the one in the
// file the reader has open.
func TestEverySentenceAfterTheFirstBeginsWithACapitalAndADataValueIsLeftAlone(t *testing.T) {
	for _, one := range []struct {
		name  string
		parts []string
		want  string
	}{
		{"two problems of a manifest",
			[]string{"the inference about pesel carries no basis", "the relation between a and b carries no basis"},
			"the inference about pesel carries no basis. The relation between a and b carries no basis"},
		{"a row and a sentence led by an article",
			[]string{"first", "row 3 is past the end", "an inferred column has no name"},
			"first. Row 3 is past the end. An inferred column has no name"},
		{"the identifier of an entry, as a file wrote it",
			[]string{"first", "books-1-L3.checksum.off_by_one: no rationale"},
			"first. books-1-L3.checksum.off_by_one: no rationale"},
		{"an identifier of letters alone, still with its colon",
			[]string{"first", "pesel: layer \"L9\" is not one this build knows"},
			"first. pesel: layer \"L9\" is not one this build knows"},
		{"a quoted value", []string{"first", "\"maybe\" is not a verdict"}, "first. \"maybe\" is not a verdict"},
		{"a number", []string{"first", "2 rows are numbered 0"}, "first. 2 rows are numbered 0"},
		{"the first keeps its case", []string{"the only one"}, "the only one"},
		{"nothing at all", nil, ""},
		{"an empty fragment", []string{"first", ""}, "first. "},
	} {
		if got := fault.Sentences(one.parts); got != one.want {
			t.Errorf("%s: got %q, want %q", one.name, got, one.want)
		}
	}
}

// Rows refused for one reason are said once, with three of their numbers and
// a count of the rest, and the verb agrees with how many rows lead it. A row
// of nought stands for the record as a whole and is said without a number.
func TestRowsRefusedForOneReasonAreSaidOnce(t *testing.T) {
	late := fault.Reason{One: "is past the end", Many: "are past the end"}
	whole := fault.Reason{One: "the record names two entities", Many: "the record names two entities"}
	for _, one := range []struct {
		name string
		rows []int
		why  fault.Reason
		want string
	}{
		{"one row", []int{7}, late, "row 7 is past the end"},
		{"three rows", []int{7, 8, 9}, late, "rows 7, 8 and 9 are past the end"},
		{"more than three", []int{7, 8, 9, 10, 11}, late, "rows 7, 8, 9 and 2 more are past the end"},
		{"the record as a whole", []int{0}, whole, "the record names two entities"},
	} {
		grouped := fault.Grouped{}
		for _, row := range one.rows {
			grouped.Add(one.why, row)
		}
		said := grouped.Said()
		if len(said) != 1 || said[0] != one.want {
			t.Errorf("%s: got %q, want %q", one.name, said, one.want)
		}
	}
}

// Several reasons come out in one order whatever order they were met in, so
// one record always gives one message.
func TestReasonsComeOutInOneOrder(t *testing.T) {
	first, second := fault.Grouped{}, fault.Grouped{}
	a := fault.Reason{One: "a reason", Many: "a reason"}
	b := fault.Reason{One: "b reason", Many: "b reason"}
	first.Add(a, 1)
	first.Add(b, 2)
	second.Add(b, 2)
	second.Add(a, 1)
	if x, y := fault.Sentences(first.Said()), fault.Sentences(second.Said()); x != y {
		t.Errorf("the same reasons met in two orders gave %q and %q", x, y)
	}
}
