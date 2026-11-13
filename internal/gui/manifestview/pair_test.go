package manifestview

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

// samePair is a value, the same text in another Unicode form, and both as the
// card should write them. The expected forms are written out code point by
// code point rather than asked of golang.org/x/text, which decides them in the
// code under test and would only agree with itself.
type samePair struct {
	why                     string
	before, after           string
	shownBefore, shownAfter string
}

var samePairs = []samePair{
	{"the decomposed form the damage writes, with the l with a stroke that has no decomposition",
		"\U0000017B\U000000F3\U00000142\U00000107", "Z\U00000307o\U00000301\U00000142c\U00000301",
		"\U0000017B\U000000F3\U00000142\U00000107", "Z<U+0307>o<U+0301>\U00000142c<U+0301>"},
	{"a character replaced by another of the same look has nothing but its code point to show",
		"\U00002126", "\U000003A9",
		"\U00002126", "<U+03A9>"},
	{"a sign that decomposes to a letter and a ring",
		"\U0000212B", "A\U0000030A",
		"\U0000212B", "A<U+030A>"},
	{"a Hangul syllable is one segment on one side and three on the other",
		"\U0000D55C", "\U00001112\U00001161\U000011AB",
		"\U0000D55C", "\U00001112<U+1161><U+11AB>"},
	{"Devanagari keeps its vowel signs drawn and writes out only the nukta the damage separated",
		"\U00000958\U0000093F\U00000932\U0000093E", "\U00000915\U0000093C\U0000093F\U00000932\U0000093E",
		"\U00000958\U0000093F\U00000932\U0000093E", "\U00000915<U+093C>\U0000093F\U00000932\U0000093E"},
	{"an accent stored apart in both values stays drawn in both",
		"\U0000017Be\U00000301", "Z\U00000307e\U00000301",
		"\U0000017Be\U00000301", "Z<U+0307>e\U00000301"},
	{"the side with more code points is written out, here the value before",
		"Z\U00000307", "\U0000017B",
		"Z<U+0307>", "\U0000017B"},
	{"two accents in another order are as many code points, so the after is written out",
		"a\U00000301\U00000323", "a\U00000323\U00000301",
		"a\U00000301\U00000323", "a<U+0323><U+0301>"},
	{"a character that draws nothing is written out beside the accent",
		"\U0000017B\U0000200B", "Z\U00000307\U0000200B",
		"\U0000017B<U+200B>", "Z<U+0307><U+200B>"},
}

func valuesLine(shownBefore, shownAfter string) string {
	return i18n.English.T(i18n.ManifestEntryBefore) + " \"" + shownBefore + "\"" + separator +
		i18n.English.T(i18n.ManifestEntryAfter) + " \"" + shownAfter + "\""
}

// Damage that writes a value in another Unicode form leaves a before and an
// after the font draws the same. The card writes out the code points where the
// two differ and draws everything else as it is.
func TestTheSameTextInAnotherFormIsWrittenOutWhereTheTwoDiffer(t *testing.T) {
	for _, one := range samePairs {
		defect := applied("one", 1, catalog.Acceptance, catalog.Hard)
		defect.ValueBefore, defect.ValueAfter = one.before, one.after
		got := screenOf(t, nil, defect).Rows[0].Values.Text
		if want := valuesLine(one.shownBefore, one.shownAfter); got != want {
			t.Errorf("%s: %+q and %+q are shown as %+q, want %+q", one.why, one.before, one.after, got, want)
		}
	}
}

// A pair that is not the same text looks different already or differs by a
// character visible writes out on its own, so its accents stay drawn. Writing
// them out would make an ordinary value in a script that stores its accents
// apart unreadable, whatever damage it carries.
func TestAPairThatIsNotTheSameTextKeepsItsAccentsDrawn(t *testing.T) {
	cases := []struct{ why, before, after string }{
		{"the Turkish letters are other text", "Irina", "\U00000130r\U00000131na"},
		{"a Devanagari value emptied by other damage", "\U00000958\U0000093F\U00000932\U0000093E", ""},
		{"a value lengthened after an accent stored apart", "e\U00000301", "e\U00000301x"},
		{"a value in neither form, so decomposed to be compared, then lengthened",
			"\U0000017Be\U00000301", "\U0000017Be\U00000301!"},
		{"a value the damage did not change", "\U0000017Be\U00000301", "\U0000017Be\U00000301"},
	}
	for _, one := range cases {
		defect := applied("one", 1, catalog.Rejection, catalog.Soft)
		defect.ValueBefore, defect.ValueAfter = one.before, one.after
		got := screenOf(t, nil, defect).Rows[0].Values.Text
		if want := valuesLine(one.before, one.after); got != want {
			t.Errorf("%s: %+q and %+q are shown as %+q, want them as they are", one.why, one.before, one.after, got)
		}
	}
}

// Only a pair can tell an accent the damage separated from one a script stores
// apart, so a value carried without the other half is drawn as it is.
func TestAValueWithoutItsOtherHalfKeepsItsAccentsDrawn(t *testing.T) {
	defect := applied("one", 1, catalog.Acceptance, catalog.Hard)
	defect.ValueBefore, defect.HasValueBefore = "", false
	defect.ValueAfter = "Z\U00000307o\U00000301"
	got := screenOf(t, nil, defect).Rows[0].Values.Text
	if want := i18n.English.T(i18n.ManifestEntryAfter) + " \"Z\U00000307o\U00000301\""; got != want {
		t.Errorf("an after carried alone is shown as %+q, want %+q", got, want)
	}
}

// The pieces of a pair put back together are the two values, byte for byte,
// so no part of a value is dropped from the screen or shown twice.
func TestThePiecesOfAPairAreTheWholeOfBothValues(t *testing.T) {
	for _, one := range samePairs {
		var before, after strings.Builder
		for _, part := range piecesOf(one.before, one.after) {
			before.WriteString(part.before)
			after.WriteString(part.after)
		}
		if before.String() != one.before || after.String() != one.after {
			t.Errorf("%s: the pieces of %+q and %+q put back together are %+q and %+q",
				one.why, one.before, one.after, before.String(), after.String())
		}
	}
}
