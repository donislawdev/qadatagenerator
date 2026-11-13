package mutators_test

import (
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"golang.org/x/text/unicode/norm"

	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// The two libraries imported above are oracles and nothing else. They hold
// the Unicode tables this build deliberately does not carry, so they can say
// whether a character normalises to the letter the code claims and whether a
// place inside a value is between two characters, which are the two things
// these mutators would otherwise be trusted on.

const (
	emojiBasic     catalog.ID = "UNI.emoji.basic"
	emojiZWJ       catalog.ID = "UNI.emoji.zwj_sequence"
	zeroWidthSpace catalog.ID = "UNI.zero_width.space"
	nonIdempotent  catalog.ID = "UNI.normalization.non_idempotent"
)

// The characters the tests look for, written as escapes because two of them
// cannot be seen.
const (
	zwsp = "\U0000200B"
	zwj  = '\U0000200D'
)

var unicodeEntries = []catalog.ID{emojiBasic, emojiZWJ, zeroWidthSpace, nonIdempotent}

func textColumn() recipe.FieldInfo {
	return recipe.FieldInfo{Name: "first_name", Type: "string"}
}

// withDefaults damages a value the way a run would, with the knobs the catalog
// publishes.
func withDefaults(t *testing.T, id catalog.ID, value string) mutator.Outcome {
	t.Helper()
	return damage(t, id, mutator.Target{
		Value: value, Field: textColumn(), Params: defaultsOf(entryOf(t, id)),
	})
}

// at damages a value with the position knob set by hand.
func at(t *testing.T, id catalog.ID, value, position string) mutator.Outcome {
	t.Helper()
	return damage(t, id, mutator.Target{
		Value: value, Field: textColumn(), Params: mutator.Parameters{"position": position},
	})
}

// The four published examples are exact values rather than shapes, so this
// build has to produce them. The catalog writes an invisible character as an
// escape inside the text, and the escape is read the way Go reads one.
func TestTheUnicodeEntriesProduceTheirPublishedExamples(t *testing.T) {
	for _, id := range unicodeEntries {
		t.Run(string(id), func(t *testing.T) {
			entry := entryOf(t, id)
			before := unescaped(t, entry.Example.Before)
			after := unescaped(t, entry.Example.After)
			outcome := withDefaults(t, id, before)
			if !outcome.Applied() {
				t.Fatalf("the published example was left alone: %s", outcome.Reason())
			}
			if outcome.Value() != after {
				t.Errorf("the specification promises %+q -> %+q and this build produces %+q",
					before, after, outcome.Value())
			}
		})
	}
}

func unescaped(t *testing.T, written string) string {
	t.Helper()
	read, err := strconv.Unquote(`"` + written + `"`)
	if err != nil {
		t.Fatalf("the example %q is not text an escape reader accepts: %v", written, err)
	}
	return read
}

// Every one of the four runs on what the damage carries with it, so neither
// mode can refuse them for want of a capability. A corpus dropped from the list
// would leave every one of them refused in the manifest and every test of the
// mutators themselves still green.
func TestTheDamageCarriesWhatTheUnicodeEntriesRequire(t *testing.T) {
	carried := capability.NewSet(mutators.Brings()...)
	for _, id := range unicodeEntries {
		if missing := carried.Missing(entryOf(t, id).Requires); len(missing) > 0 {
			t.Errorf("%s requires %v, which the damage does not carry", id, missing)
		}
	}
}

func TestTheEmojiIsOneSymbolOutsideTheBasicPlane(t *testing.T) {
	outcome := at(t, emojiBasic, "x", "end")
	added, kept := strings.CutPrefix(outcome.Value(), "x ")
	if !outcome.Applied() || !kept {
		t.Fatalf("the emoji did not follow the value after a space: %+q (%s)", outcome.Value(), outcome.Reason())
	}
	r, size := utf8.DecodeRuneInString(added)
	if size != len(added) {
		t.Fatalf("%+q is more than one code point", added)
	}
	if utf8.RuneLen(r) != 4 {
		t.Errorf("%U takes %d bytes in UTF-8, and the point of it is the fourth", r, utf8.RuneLen(r))
	}
	if !unicode.Is(unicode.So, r) {
		t.Errorf("%U is not a symbol", r)
	}
}

// The comment beside the family promises seven code points, three joiners,
// twenty five bytes and one character to a reader, and each of those is what
// a length check gets wrong.
func TestTheFamilyIsOneCharacterOfSevenCodePoints(t *testing.T) {
	outcome := damage(t, emojiZWJ, mutator.Target{Value: "x", Field: textColumn()})
	added, kept := strings.CutPrefix(outcome.Value(), "x ")
	if !outcome.Applied() || !kept {
		t.Fatalf("the family did not follow the value after a space: %+q (%s)", outcome.Value(), outcome.Reason())
	}
	runes := []rune(added)
	if len(runes) != 7 || len(added) != 25 {
		t.Fatalf("%+q is %d code points and %d bytes", added, len(runes), len(added))
	}
	for i, r := range runes {
		if i%2 == 1 && r != zwj {
			t.Errorf("code point %d is %U, and the people are joined by %U", i, r, zwj)
		}
		if i%2 == 0 && (utf8.RuneLen(r) != 4 || !unicode.Is(unicode.So, r)) {
			t.Errorf("code point %d is %U, not an emoji outside the basic plane", i, r)
		}
	}
	if clusters := graphemesOf(added); len(clusters) != 1 {
		t.Errorf("a reader sees %d characters in %+q rather than one", len(clusters), added)
	}
}

// Invisible, and not whitespace either, so trimming does not remove it.
func TestTheZeroWidthSpaceCannotBeSeenAndIsNotWhitespace(t *testing.T) {
	r, _ := utf8.DecodeRuneInString(zwsp)
	if unicode.IsPrint(r) || unicode.IsGraphic(r) {
		t.Errorf("%U is printable, so it would be seen", r)
	}
	if unicode.IsSpace(r) {
		t.Errorf("%U is whitespace, so trimming would remove it", r)
	}
	outcome := at(t, zeroWidthSpace, "ab", "middle")
	if outcome.Value() != "a"+zwsp+"b" {
		t.Errorf("the character did not land between the two letters: %+q", outcome.Value())
	}
	if strings.TrimSpace(outcome.Value()) != outcome.Value() {
		t.Error("trimming changed the damaged value")
	}
}

func TestTheThreePositionsPutTheInsertionWhereTheyName(t *testing.T) {
	cases := []struct {
		id       catalog.ID
		position string
		want     string
	}{
		{emojiBasic, "start", "\U0001F642 Maria"},
		{emojiBasic, "middle", "Ma\U0001F642ria"},
		{emojiBasic, "end", "Maria \U0001F642"},
		{zeroWidthSpace, "start", zwsp + "Maria"},
		{zeroWidthSpace, "middle", "Ma" + zwsp + "ria"},
		{zeroWidthSpace, "end", "Maria" + zwsp},
	}
	for _, one := range cases {
		outcome := at(t, one.id, "Maria", one.position)
		if !outcome.Applied() || outcome.Value() != one.want {
			t.Errorf("%s at %s gave %+q (%s), want %+q", one.id, one.position,
				outcome.Value(), outcome.Reason(), one.want)
		}
	}
}

// A middle chosen by counting code points would put the invisible character
// inside something a reader sees as one, and what comes out would no longer be
// the same text with one character added. The oracle is the published
// segmentation, which this build does not carry: a place taken has to be one it
// would break at, and the characters a reader sees afterwards have to be the
// ones before plus the one added.
//
// The first list is what a file of names, places and addresses holds, and each
// has to get its middle. The second holds the ways one character is built from
// several code points, and each may be refused, but where one is placed the
// place has to be right.
func TestTheMiddleIsAlwaysBetweenTwoCharactersAReaderSees(t *testing.T) {
	ordinary := []string{
		"maria@example.test",
		"Jo\U00000301zef",         // a decomposed accent, with ordinary letters beyond it
		"Zo\U00000301\U00000142w", // decomposed, beside a letter with no decomposition
		"\U0000017B\U000000F3\U00000142\U00000107 Nowak-Kowalska",      // Polish, a space and a hyphen
		"\U00000418\U00000432\U00000430\U0000043D\U0000043E\U00000432", // Cyrillic
		"\U0000674E\U0000660E\U00005C0F\U00006797",                     // Han
		"+48 600 100 200",
		"12,50 PLN",
	}
	awkward := []string{
		"e\U00000301e\U00000301e\U00000301e", // accents back to back
		family() + "x",
		family() + family(),
		"\U0001F1F5\U0001F1F1x\U0001F1FA\U0001F1F8",                                // flags around a letter
		"\U0001F44D\U0001F3FD\U0001F44D\U0001F3FD",                                 // skin tones
		"\U00002764\U0000FE0F\U00002764\U0000FE0F",                                 // variation selectors
		"1\U0000FE0F\U000020E32\U0000FE0F\U000020E3",                               // keycaps
		"\U0001F3F4\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007Fab", // a tag flag
		"\U00001100\U00001161\U00001100\U0000AC00",                                 // Hangul jamo before a syllable
		"\U00000915\U0000094D\U00000937\U00000924\U0000094D\U00000930",             // Devanagari conjuncts
		"\U00000E01\U00000E33\U00000E01\U00000E33",                                 // Thai, a vowel that is a letter
		"\U00000600\U00000661\U00000662",                                           // an Arabic sign that goes before
		"a\U0000200Cb\U0000200Cc",                                                  // a non joiner
		"a\r\nb",
		"\U00000645\U0000064E\U00000631\U0000064E\U00000628", // Arabic with vowel marks
	}
	for _, value := range ordinary {
		outcome := at(t, zeroWidthSpace, value, "middle")
		if !outcome.Applied() {
			t.Errorf("%+q was refused a middle: %s", value, outcome.Reason())
			continue
		}
		requireOneCharacterAdded(t, value, outcome.Value())
	}
	for _, value := range awkward {
		if outcome := at(t, zeroWidthSpace, value, "middle"); outcome.Applied() {
			requireOneCharacterAdded(t, value, outcome.Value())
		}
	}
}

// The claim the middle stands on, held for every character Unicode defines: a
// character this build calls ordinary is never joined to its neighbour. Two of
// the same side by side is the pair every rule that joins would join, apart
// from the Korean syllables, which this build never calls ordinary.
func TestNoCharacterCalledOrdinaryIsJoinedToItsNeighbour(t *testing.T) {
	placedAny := 0
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !utf8.ValidRune(r) {
			continue
		}
		pair := string(r) + string(r)
		outcome := at(t, zeroWidthSpace, pair, "middle")
		if !outcome.Applied() {
			continue
		}
		placedAny++
		if !isBoundary(pair, len(string(r))) {
			t.Errorf("%U is called ordinary and the segmentation joins two of it", r)
		}
	}
	if placedAny == 0 {
		t.Fatal("no character was called ordinary, so this checked nothing")
	}
}

// requireOneCharacterAdded holds a damaged value to the original: the same text
// with the invisible character at a place the segmentation breaks at, and one
// more character to a reader than before.
func requireOneCharacterAdded(t *testing.T, value, damaged string) {
	t.Helper()
	place := strings.Index(damaged, zwsp)
	if place <= 0 || place >= len(value) || damaged[:place]+damaged[place+len(zwsp):] != value {
		t.Errorf("%+q became %+q, which is not the value with one character inside it", value, damaged)
		return
	}
	if !isBoundary(value, place) {
		t.Errorf("%+q: byte %d is inside a character a reader sees as one", value, place)
	}
	if before, after := graphemesOf(value), graphemesOf(damaged); len(after) != len(before)+1 {
		t.Errorf("%+q holds %d characters and %d after one was added", value, len(before), len(after))
	}
}

func family() string {
	return "\U0001F469\U0000200D\U0001F469\U0000200D\U0001F467\U0000200D\U0001F466"
}

func graphemesOf(value string) []string {
	var found []string
	for seg := graphemes.FromString(value); seg.Next(); {
		found = append(found, seg.Value())
	}
	return found
}

func isBoundary(value string, place int) bool {
	for seg := graphemes.FromString(value); seg.Next(); {
		if seg.End() == place {
			return true
		}
	}
	return false
}

// Every refusal is damage that did not happen, and the manifest says why. Each
// of these would otherwise be an entry claiming a change that is not there, or
// a change of a different kind than the entry describes.
func TestTheUnicodeEntriesRefuseWhatTheyCannotDoWithAReason(t *testing.T) {
	cases := []struct {
		what   string
		id     catalog.ID
		target mutator.Target
		says   string
	}{
		{"an emoji into nothing", emojiBasic,
			mutator.Target{Value: "", Params: mutator.Parameters{"position": "end"}}, "empty"},
		{"a family into nothing", emojiZWJ, mutator.Target{Value: ""}, "empty"},
		{"an emoji into a code", emojiBasic,
			mutator.Target{Value: "666-25-8528", Params: mutator.Parameters{"position": "end"}}, "no letter"},
		{"a family into a date", emojiZWJ, mutator.Target{Value: "1990-04-11"}, "no letter"},
		{"an invisible character into nothing", zeroWidthSpace,
			mutator.Target{Value: "", Params: mutator.Parameters{"position": "middle"}}, "empty"},
		{"the middle of one character", zeroWidthSpace,
			mutator.Target{Value: "A", Params: mutator.Parameters{"position": "middle"}}, "no middle"},
		{"no position given", emojiBasic, mutator.Target{Value: "Maria"}, "no position"},
		{"a position nobody named", zeroWidthSpace,
			mutator.Target{Value: "Maria", Params: mutator.Parameters{"position": "centre"}}, "none of"},
		{"no letter with a raised capital", nonIdempotent,
			mutator.Target{Value: "Szyszy 2024"}, "no Latin letter"},
		{"nothing at all to raise", nonIdempotent, mutator.Target{Value: ""}, "no Latin letter"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			one.target.Field = textColumn()
			outcome := damage(t, one.id, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %+q", outcome.Value())
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}

// The table of raised capitals is twenty two code points written by hand, and
// a wrong one would supply a value that normalises to the wrong letter or not
// at all. Held against the normalisation tables, each one has to come back as
// the capital it stands for, and the four letters without one have to be left
// alone.
func TestEveryRaisedCapitalNormalisesToTheLetterItStandsFor(t *testing.T) {
	for letter := 'A'; letter <= 'Z'; letter++ {
		for _, written := range []rune{letter, unicode.ToLower(letter)} {
			outcome := damage(t, nonIdempotent, mutator.Target{Value: string(written), Field: textColumn()})
			if strings.ContainsRune("SXYZ", letter) {
				if outcome.Applied() {
					t.Errorf("%c has no raised capital in Unicode and became %+q", written, outcome.Value())
				}
				continue
			}
			if !outcome.Applied() {
				t.Errorf("%c was left alone: %s", written, outcome.Reason())
				continue
			}
			if got := norm.NFKC.String(outcome.Value()); got != string(letter) {
				t.Errorf("%c became %+q, which normalises to %q rather than %c", written, outcome.Value(), got, letter)
			}
			if strings.ToLower(outcome.Value()) != outcome.Value() {
				t.Errorf("%+q changes when lower cased, so the pipeline would not leave it alone", outcome.Value())
			}
		}
	}
}

// The claim of the entry, measured: lower casing and then normalising gives a
// different answer the second time, on the damaged value and not on the
// original. The second half is what the soft confidence is about: a pipeline
// that normalises first is stable on the same value.
func TestNormalisingTheDamagedValueTwiceDiffersFromOnce(t *testing.T) {
	lowerThenNormalise := func(s string) string { return norm.NFKC.String(strings.ToLower(s)) }
	normaliseThenLower := func(s string) string { return strings.ToLower(norm.NFKC.String(s)) }
	for _, value := range []string{"bigbird", "Maria", "osoba1@example.test", "Szczyrk", "\U00000141ukasz"} {
		outcome := withDefaults(t, nonIdempotent, value)
		if !outcome.Applied() {
			t.Errorf("%q was left alone: %s", value, outcome.Reason())
			continue
		}
		damaged := outcome.Value()
		if once := lowerThenNormalise(value); lowerThenNormalise(once) != once {
			t.Fatalf("%q is unstable before any damage, so the damage proves nothing", value)
		}
		once := lowerThenNormalise(damaged)
		if lowerThenNormalise(once) == once {
			t.Errorf("%+q gives %+q both times, so it supplies nothing", damaged, once)
		}
		if first := normaliseThenLower(damaged); normaliseThenLower(first) != first {
			t.Errorf("%+q is unstable the other way round too, which the comment says it is not", damaged)
		}
	}
}

// Only the letters change. Bytes that are not text in the first place are
// copied as they were, because a decoder that read them would hand back the
// replacement character and the value would change somewhere the manifest does
// not say.
func TestRaisingLeavesEverythingButTheLettersAsItWas(t *testing.T) {
	cases := []struct{ value, want string }{
		{"b\xffa", "\U00001D2E\xff\U00001D2C"},
		{"\U00000141\U000000F3d\U0000017A", "\U00000141\U000000F3\U00001D30\U0000017A"},
		{"\U00000131b", "\U00000131\U00001D2E"},
		{"a1-b", "\U00001D2C1-\U00001D2E"},
	}
	for _, one := range cases {
		outcome := withDefaults(t, nonIdempotent, one.value)
		if outcome.Value() != one.want {
			t.Errorf("%+q became %+q, want %+q", one.value, outcome.Value(), one.want)
		}
	}
}
