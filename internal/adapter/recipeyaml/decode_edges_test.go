package recipeyaml_test

import (
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
)

// The edges of the reader, each one found by review or by measurement rather
// than by a test failing. They are kept together because they share a shape: a
// recipe that parses, means something nobody intended, and was accepted in
// silence.

// # The value that switched off the check written to catch it
//
// Every comparison against a NaN is false, so a share of NaN passed the range
// test, and then passed the rule refusing a field that is both required and
// sometimes empty. One value made a recipe carrying a contradiction read as
// examined and approved, which is worse than being accepted quietly.
func TestAShareThatIsNotANumberIsRefused(t *testing.T) {
	for _, written := range []string{"NaN%", "nan%", "Inf%", "+Inf%"} {
		result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  c:
    fields:
      e: { type: string, required: true, empty: "` + written + `" }
`))
		if result.Valid() {
			t.Errorf("a share written %s was accepted", written)
			continue
		}
		named := false
		for _, found := range result.Problems {
			if strings.Contains(found.What, "not a number") {
				named = true
			}
		}
		if !named {
			t.Errorf("a share written %s was refused without naming why: %+v", written, result.Problems)
		}
	}
}

// A share that is an ordinary number still has to work, or the guard above is
// refusing the format rather than an attack.
func TestAnOrdinaryShareStillWorks(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  c:
    fields:
      e: { type: string, empty: "15%" }
`)
	field := result.Recipe.Entities[0].Fields[0]
	if !field.HasEmptyShare || field.EmptyShare != 15 {
		t.Fatalf("expected a share of fifteen, got %v %v", field.HasEmptyShare, field.EmptyShare)
	}
}

// # Why braces are counted as well as square brackets
//
// The nesting limit read square brackets alone until 2026-08-03, and the note
// beside it cleared braces on one measurement taken at six percent of the
// largest recipe this build accepts. Measured across three sizes instead, the
// same shape written with braces cost 27 MB at 20 kB, 194 MB at 40 kB and
// 597 MB at 80 kB, which is the same machine killed by the operating system
// that the limit exists to prevent.
//
// The payload here is deliberately modest, for the reason the sibling test
// gives: a test may not be a weapon.
func TestARecipeNestedInBracesIsRefusedBeforeParsing(t *testing.T) {
	nested := []byte(strings.Repeat("{", 5000))

	start := time.Now()
	result := recipeyaml.Decode(nested)
	elapsed := time.Since(start)

	if result.Valid() {
		t.Fatal("a recipe nesting braces past the limit was accepted")
	}
	if len(result.Problems) != 1 || !strings.Contains(result.Problems[0].What, "nests brackets") {
		t.Fatalf("expected one problem naming the nesting, got %+v", result.Problems)
	}
	if elapsed > time.Second {
		t.Errorf("the refusal took %v, so the parser was handed the input after all", elapsed)
	}
}

// The two kinds nest inside each other, so counting them apart would let a
// document reach twice the limit with each counter innocent. That arithmetic is
// exactly what a hostile recipe is written to find.
func TestNestingIsCountedAcrossBothKindsOfBracket(t *testing.T) {
	result := recipeyaml.Decode([]byte(strings.Repeat("[{", 40)))

	if result.Valid() {
		t.Fatal("eighty levels split over two kinds of bracket were accepted")
	}
	if !strings.Contains(result.Problems[0].What, "nests brackets") {
		t.Fatalf("expected the nesting to be named, got %+v", result.Problems)
	}
}

// A closing bracket that is part of a string or of a comment does not close
// anything, so it cannot buy room for more nesting.
//
// Found by an outside review on 2026-10-22: the scan over bytes counted the
// bracket in "]" as a closing one, so x: ["]", ["]", ... nested 8192 levels in
// 57 350 bytes while reading as one level, and the parser took 445 MiB. Two
// hundred levels here, for the reason the siblings give.
func TestABracketInsideAStringOrACommentBuysNoRoom(t *testing.T) {
	const levels = 200
	for _, given := range []struct{ name, recipe string }{
		{"a double quoted bracket",
			"x: " + strings.Repeat(`["]", `, levels) + "0" + strings.Repeat("]", levels)},
		{"a single quoted bracket",
			"x: " + strings.Repeat(`[']', `, levels) + "0" + strings.Repeat("]", levels)},
		{"a bracket after a comment mark",
			"x: " + strings.Repeat("[ # ]\n  ", levels) + "0" + strings.Repeat("]", levels) + "\n"},
	} {
		result := recipeyaml.Decode([]byte(given.recipe))
		if result.Valid() || len(result.Problems) != 1 ||
			!strings.Contains(result.Problems[0].What, "nests brackets 200 deep") {
			t.Errorf("%s: %d levels were not refused for their depth: %+v", given.name, levels,
				result.Problems)
		}
	}
}

// Nesting a recipe actually uses has to survive, or the guard is refusing the
// format rather than an attack. A field written inline is one level and a list
// of values inside it is two.
func TestTheNestingARecipeActuallyUsesIsAccepted(t *testing.T) {
	decode(t, `
version: "0.1"
entities:
  c:
    fields:
      s: { type: enum, values: [new, active, closed] }
`)
}

// # Why the count grew a third spelling on 2026-09-02
//
// A sequence written with dashes nests without a bracket, so it walked past a
// limit that counted brackets. Along one line it costs two bytes a level,
// which put a document at under a third of the size limit at nearly five
// gigabytes. The measurements are at MaxNestingDepth.
//
// The payload here is deliberately modest, for the reason its two siblings
// give: a test may not be a weapon. That leaves the assertion with a problem
// worth naming, because a modest payload is also a cheap one. Ten kilobytes of
// dashes cost about a tenth of a second without the guard, and a time bound
// loose enough to be stable on a busy machine cannot tell that from nothing.
// So this measures what the guard actually saves, which is memory.
func TestARecipeNestedInDashesAlongOneLineIsRefusedBeforeParsing(t *testing.T) {
	nested := []byte(strings.Repeat("- ", 5000) + "x")

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	result := recipeyaml.Decode(nested)
	runtime.ReadMemStats(&after)

	if result.Valid() {
		t.Fatal("a recipe nesting dashes past the limit was accepted")
	}
	if len(result.Problems) != 1 || !strings.Contains(result.Problems[0].What, "dashes") {
		t.Fatalf("expected one problem naming the dashes, got %+v", result.Problems)
	}
	if taken := after.TotalAlloc - before.TotalAlloc; taken > 8<<20 {
		t.Errorf("the refusal allocated %d bytes, so the parser was handed the input after all", taken)
	}
}

// The dashes are held to the number the brackets are held to, so the boundary
// is worth pinning from both sides. Sixty four is already a shape nobody
// writes, and it is still not what this refuses.
func TestTheDashesAreCountedUpToTheSameLimitAsTheBrackets(t *testing.T) {
	atTheLimit := recipeyaml.Decode([]byte(strings.Repeat("- ", 64) + "x"))
	for _, problem := range atTheLimit.Problems {
		if strings.Contains(problem.What, "dashes") {
			t.Fatalf("sixty four dashes were refused for nesting, got %+v", atTheLimit.Problems)
		}
	}

	past := recipeyaml.Decode([]byte(strings.Repeat("- ", 65) + "x"))
	if past.Valid() || len(past.Problems) == 0 ||
		!strings.Contains(past.Problems[0].What, "dashes") {
		t.Fatalf("sixty five dashes were not refused for nesting, got %+v", past.Problems)
	}
}

// # Why a dash and two spaces has to count the same as a dash and one
//
// Measured on 2026-09-02: the same depth written with two spaces after every
// dash cost 1 245 MB against 1 243 MB for one space, so the spelling that is
// cheaper to refuse is the same price to read. A count looking for a two byte
// token would have turned one away and waved the other through, which is a
// guard shaped like a guard.
func TestADashCountsWhateverBlanksFollowIt(t *testing.T) {
	result := recipeyaml.Decode([]byte(strings.Repeat("-  ", 65) + "x"))

	if result.Valid() || len(result.Problems) == 0 ||
		!strings.Contains(result.Problems[0].What, "dashes") {
		t.Fatalf("dashes separated by two spaces were not counted, got %+v", result.Problems)
	}
}

// # Why the count resets at every newline
//
// Depth carried across lines is bought with indentation, and indentation costs
// the square of the depth, so the size limit caps that shape near seven
// hundred levels. Measured on 2026-09-02: seven hundred levels in 247 kB cost
// 24 MB, and a file spending its whole size budget on dashes spread over sixty
// five lines reached 33 MB. Counting across the file would therefore refuse a
// long list somebody wrote while refusing nothing a bomb needs.
func TestTheDashesAreCountedWithinALineAndNotAcrossTheFile(t *testing.T) {
	line := strings.Repeat("- ", 33) + "\n"

	result := recipeyaml.Decode([]byte(strings.Repeat(line, 3)))

	for _, problem := range result.Problems {
		if strings.Contains(problem.What, "dashes") {
			t.Fatalf("three lines of thirty three dashes were refused for nesting, got %+v",
				result.Problems)
		}
	}
}

// # A recipe in the wrong encoding names its encoding
//
// Notepad offers UTF-16 under the name Unicode, and a recipe saved that way
// used to be refused for not being a mapping of keys, with advice to add a
// version line that was already the first line of the file. That is the trap
// the byte order mark of UTF-8 was fixed for, met on the other half of the same
// save dialog.
func TestARecipeInTheWrongEncodingIsNamedRatherThanPuzzledOver(t *testing.T) {
	cases := map[string][]byte{
		"UTF-16 little endian": {0xFF, 0xFE, 'v', 0x00, 'e', 0x00},
		"UTF-16 big endian":    {0xFE, 0xFF, 0x00, 'v', 0x00, 'e'},
		"UTF-32 little endian": {0xFF, 0xFE, 0x00, 0x00, 'v', 0x00, 0x00, 0x00},
		"UTF-32 big endian":    {0x00, 0x00, 0xFE, 0xFF, 0x00, 0x00, 0x00, 'v'},
	}
	for name, data := range cases {
		result := recipeyaml.Decode(data)
		if result.Valid() {
			t.Errorf("a file in %s was accepted", name)
			continue
		}
		if len(result.Problems) != 1 || !strings.Contains(result.Problems[0].What, name) {
			t.Errorf("a file in %s was refused without naming the encoding: %+v", name, result.Problems)
		}
	}
}

// # The encoding that announces nothing
//
// A single byte encoding carries no mark, so it is caught by the bytes failing
// to be UTF-8. This is the one the people this tool is for are most likely to
// produce: on a Polish Windows the ANSI entry of a save dialog means
// Windows-1250, and older editors and spreadsheet exports still default to it.
//
// Measured 2026-08-03, before this check: the recipe below was accepted as
// usable, and the entity named zażółć reached the model as za followed by four
// replacement characters. Nobody was told, and that damaged name is what a
// generated column would have been headed with.
func TestARecipeInASingleByteEncodingIsRefused(t *testing.T) {
	// The same recipe as below, written in Windows-1250. The Polish letters are
	// single bytes above 0x7f, which is what makes the whole file invalid UTF-8.
	polish1250 := []byte("version: \"0.1\"\nentities:\n  za\xbf\xf3\xb3\xe6:\n    fields:\n      g\xea\x9cl\xb9: { type: string }\n")

	if utf8.Valid(polish1250) {
		t.Fatal("the fixture is meant to be invalid UTF-8, so this test would prove nothing")
	}

	result := recipeyaml.Decode(polish1250)
	if result.Valid() {
		t.Fatal("a recipe in a single byte encoding was accepted, so its names are already damaged")
	}
	if len(result.Problems) != 1 || !strings.Contains(result.Problems[0].What, "not valid UTF-8") {
		t.Fatalf("expected one problem naming the encoding, got %+v", result.Problems)
	}
	if !strings.Contains(result.Problems[0].Hint, "ANSI") {
		t.Fatalf("the hint has to name what a Polish Windows calls it, got %q", result.Problems[0].Hint)
	}
}

// The same recipe in UTF-8 is read, with its Polish names intact. Without this
// the check above could be refusing every accented letter.
func TestTheSameRecipeInUTF8KeepsItsAccentedNames(t *testing.T) {
	result := decode(t, "version: \"0.1\"\nentities:\n  zażółć:\n    fields:\n      gęślą: { type: string }\n")

	if name := result.Recipe.Entities[0].Name; name != "zażółć" {
		t.Fatalf("the entity name did not survive: %q", name)
	}
	if field := result.Recipe.Entities[0].Fields[0].Name; field != "gęślą" {
		t.Fatalf("the field name did not survive: %q", field)
	}
}

// A recipe in UTF-8 stays readable with the mark and without it, or the check
// above is refusing the encoding this tool actually uses.
func TestUTF8StaysReadableWithAndWithoutTheMark(t *testing.T) {
	decode(t, oneRecipe)
	if result := recipeyaml.Decode(append([]byte{0xEF, 0xBB, 0xBF}, oneRecipe...)); !result.Valid() {
		t.Fatalf("a recipe with a UTF-8 mark was refused: %+v", result.Problems)
	}
}

// # A count larger than this build can hold
//
// The conversion to int used to be unguarded, and Go leaves it undefined when
// the value does not fit. A number above the range came back negative and was
// reported as a count below zero, which is a message about something the user
// never wrote.
// The two shapes are both here because they arrive by different routes, and
// which route a number takes was measured rather than assumed. On this parser
// 9223372036854775808 comes back as an unsigned integer, one past the signed
// range, while 1.0e19 comes back as a floating point value, where anything
// above two to the fifty third no longer represents consecutive whole numbers.
//
// Written without the decimal point, 1e19 comes back as a string and never
// reaches the arithmetic at all, which is why the case here carries the point.
func TestANumberTooLargeToHoldIsNamedRatherThanWrapped(t *testing.T) {
	for _, written := range []string{"9223372036854775808", "1.0e19"} {
		result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  c:
    count: ` + written + `
    fields:
      x: { generator: y }
`))
		if result.Valid() {
			t.Errorf("a count written %s was accepted", written)
			continue
		}
		for _, found := range result.Problems {
			if strings.Contains(found.What, "below zero") {
				t.Errorf("a count written %s was reported as one below zero: %+v", written, found)
			}
		}
		if !strings.Contains(result.Problems[0].What, "larger than this build can hold") {
			t.Errorf("a count written %s was refused without naming why: %+v", written, result.Problems)
		}
	}
}

// A count this build can hold is not the place to invent a business limit. The
// engine has its own exit code for a recipe that is valid and cannot be
// satisfied, and a number chosen here would become a public contract with no
// measurement behind it.
func TestALargeButHoldableCountIsAccepted(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  c:
    count: 1000000000
    fields:
      x: { generator: y }
`)
	if result.Recipe.Entities[0].Count != 1000000000 {
		t.Fatalf("the count did not survive: %d", result.Recipe.Entities[0].Count)
	}
}

// # Rules that parse and mean nothing
//
// A consistency rule is the one declaration serving both pillars: it says what
// has to agree when data is generated and therefore what can be pulled apart
// when it is corrupted. A rule with nothing on the right gives the first pillar
// nothing to satisfy, and a rule naming its own field is a contradiction. Both
// were accepted in silence, and one of them sat in this package's own fixture
// without anybody noticing.
func TestARuleThatSaysNothingIsRefused(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  c:
    fields:
      pesel: { type: string }
    consistency:
      - "pesel matches": []
`))
	if result.Valid() {
		t.Fatal("a rule with nothing to agree with was accepted")
	}
	if !strings.Contains(result.Problems[0].What, "does not say what pesel has to agree with") {
		t.Fatalf("expected the empty rule to be named, got %+v", result.Problems)
	}
}

func TestARuleNamingItsOwnFieldIsRefused(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  c:
    fields:
      pesel: { type: string }
      dob:   { type: date }
    consistency:
      - "pesel matches": [dob, pesel]
`))
	if result.Valid() {
		t.Fatal("a rule naming its own field was accepted")
	}
	if !strings.Contains(result.Problems[0].What, "agree with itself") {
		t.Fatalf("expected the self reference to be named, got %+v", result.Problems)
	}
}

// An unreadable list is named where its entries were read, so the rule above it
// must not add a second complaint about the same line. Two messages for one
// mistake is how a five minute fix turns into five rounds.
func TestAnUnreadableRuleListIsReportedOnce(t *testing.T) {
	result := recipeyaml.Decode([]byte(`
version: "0.1"
entities:
  c:
    fields:
      pesel: { type: string }
    consistency:
      - "pesel matches": [[nested]]
`))
	if result.Valid() {
		t.Fatal("a rule listing something that is not a field name was accepted")
	}
	for _, found := range result.Problems {
		if strings.Contains(found.What, "does not say what") {
			t.Fatalf("the entries were named and the rule complained as well: %+v", result.Problems)
		}
	}
}

// A rule that means something has to survive all of the above.
func TestAnOrdinaryRuleStillWorks(t *testing.T) {
	result := decode(t, `
version: "0.1"
entities:
  c:
    fields:
      pesel: { type: string }
      dob:   { type: date }
    consistency:
      - "pesel matches": [dob]
`)
	rules := result.Recipe.Entities[0].Consistency
	if len(rules) != 1 || rules[0].Field != "pesel" || len(rules[0].With) != 1 {
		t.Fatalf("the rule did not survive: %+v", rules)
	}
}
