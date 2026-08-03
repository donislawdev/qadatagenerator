package recipeyaml

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/goccy/go-yaml"
)

// The hint below is the whole reason text is demanded rather than accepted in
// whatever shape YAML felt like reading it. Measured in a sister project:
// 010 becomes 8 because YAML reads it as octal, and 1.0 becomes a floating
// point number that comes back out as 1. Both change the value silently, and a
// seed or a version that changes silently is the worst possible place for it.
const quoteHint = "put the value in quotes, because YAML reads 010 as 8 and 1.0 as 1"

func (r *reader) key(where string, item yaml.MapItem) (string, bool) {
	key, ok := item.Key.(string)
	if !ok {
		r.report(where, fmt.Sprintf("the key %v is not text", item.Key), "", quoteHint)
		return "", false
	}
	return key, true
}

// text demands a YAML string. A number where text belongs is refused rather
// than converted, because the conversion is where the value changes.
func (r *reader) text(where string, node any) (string, bool) {
	switch value := node.(type) {
	case string:
		return value, true
	case nil:
		r.report(where, "the value is missing", "", "write the value or remove the line")
		return "", false
	case bool:
		r.report(where, "the value has to be text and YAML read it as true or false", "", quoteHint)
		return "", false
	case yaml.MapSlice, []any:
		r.report(where, "the value has to be a single piece of text", "", "")
		return "", false
	default:
		r.report(where, fmt.Sprintf("the value has to be text and YAML read %v as a number", value), "", quoteHint)
		return "", false
	}
}

// oneOf demands text from a closed list, and names the list when it is not.
func (r *reader) oneOf(where string, node any, allowed []string) string {
	value, ok := r.text(where, node)
	if !ok {
		return ""
	}
	for _, candidate := range allowed {
		if value == candidate {
			return value
		}
	}
	r.report(where, "this build does not know "+value, strings.Join(allowed, ", "), "")
	return ""
}

func (r *reader) mapping(where string, node any) (yaml.MapSlice, bool) {
	switch value := node.(type) {
	case yaml.MapSlice:
		return value, true
	case nil:
		r.report(where, "the section is empty", "", "fill it in or remove it")
		return nil, false
	default:
		r.report(where, "this has to be a section with keys under it", "", "")
		return nil, false
	}
}

func (r *reader) sequence(where string, node any) ([]any, bool) {
	switch value := node.(type) {
	case []any:
		return value, true
	case nil:
		r.report(where, "the list is empty", "", "fill it in or remove it")
		return nil, false
	default:
		r.report(where, "this has to be a list", "", "write each entry on its own line starting with a dash")
		return nil, false
	}
}

// count demands a whole number that is not negative. Zero is allowed on
// purpose: a recipe that asks for no rows of one entity is odd but legal, and
// refusing it would be this tool deciding what somebody meant to test.
// There is deliberately no upper limit here beyond what this build can hold.
// Whether a count can actually be produced is a question for the engine, which
// has its own exit code for a recipe that is valid and cannot be satisfied. A
// number chosen today would become a public contract without a measurement
// behind it.
func (r *reader) count(where string, node any) int {
	number, fault := wholeNumber(node)
	switch fault {
	case numberNotWhole:
		r.report(where, "the number of rows has to be a whole number", "", "write it without quotes, such as count: 100")
		return 0
	case numberOutOfRange:
		r.report(where, "the number of rows is larger than this build can hold", "a whole number this build can count to",
			"write a number of rows that a machine could produce")
		return 0
	}
	if number < 0 {
		r.report(where, "the number of rows is below zero", "zero or more", "")
		return 0
	}
	return number
}

// maxLength demands a whole number above zero.
//
// Zero is refused here although a count of zero is accepted a few lines up,
// and the difference is not an inconsistency. No rows of an entity is a
// smaller dataset. A maximum length of nothing is a field no value can ever
// satisfy, so every row would fail a limit the recipe set on itself.
func (r *reader) maxLength(where string, node any) (int, bool) {
	number, fault := wholeNumber(node)
	switch fault {
	case numberNotWhole:
		r.report(where, "the maximum length has to be a whole number", "",
			"write it without quotes, such as max_length: 30")
		return 0, false
	case numberOutOfRange:
		r.report(where, "the maximum length is larger than this build can hold",
			"a whole number this build can count to", "write a limit a value could actually reach")
		return 0, false
	}
	if number <= 0 {
		r.report(where, "the maximum length is not above zero", "one or more",
			"remove the limit instead, because no value can be shorter than nothing")
		return 0, false
	}
	return number, true
}

func (r *reader) flag(where string, node any) bool {
	value, ok := node.(bool)
	if !ok {
		r.report(where, "this has to be true or false", "true, false", "write it without quotes")
		return false
	}
	return value
}

// share reads a percentage written the way the format documents it, as 15%.
// A bare number is refused, because 15 and 15% differ by a factor of a hundred
// and nothing downstream could tell which one was meant.
func (r *reader) share(where string, node any) (float64, bool) {
	value, ok := r.text(where, node)
	if !ok {
		return 0, false
	}
	if !strings.HasSuffix(value, "%") {
		r.report(where, "the share has to end with a percent sign", "a share such as 15%", "")
		return 0, false
	}
	share, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
	if err != nil {
		r.report(where, "the share is not a number", "a share such as 15%", "")
		return 0, false
	}
	// Not a number and infinity are refused before the range is compared, and
	// the order is the whole point. Every comparison against NaN is false, so
	// the range test below passes it, and so does the rule a few files away
	// that refuses a field both required and sometimes empty. One value would
	// therefore switch off a check written to catch it, which is worse than
	// being accepted: the recipe reads as though it had been examined.
	if math.IsNaN(share) || math.IsInf(share, 0) {
		r.report(where, "the share is not a number that can be counted with", "a share such as 15%",
			"write a plain number between 0 and 100 followed by a percent sign")
		return 0, false
	}
	if share < 0 || share > 100 {
		r.report(where, "the share is outside nought to a hundred percent", "a share such as 15%", "")
		return 0, false
	}
	return share, true
}

func (r *reader) list(where string, node any) []string {
	entries, ok := r.sequence(where, node)
	if !ok {
		return nil
	}
	values := make([]string, 0, len(entries))
	for index, entry := range entries {
		value, ok := r.text(fmt.Sprintf("%s[%d]", where, index), entry)
		if ok {
			values = append(values, value)
		}
	}
	return values
}

// listOrOne accepts both a list and a single value, because a rule tying one
// field to one other field reads better as postal_code matches: city than as a
// list of one.
func (r *reader) listOrOne(where string, node any) []string {
	if _, isList := node.([]any); isList {
		return r.list(where, node)
	}
	if value, ok := r.text(where, node); ok {
		return []string{value}
	}
	return nil
}

// numberFault says why a number could not be used.
//
// It is three states rather than a boolean so that a value too large to hold
// is not reported as though it were not a number. The two need different words:
// one is a mistake in what was written, the other is a limit of this build.
type numberFault int

const (
	numberOK numberFault = iota
	// numberNotWhole means the value is not a whole number at all.
	numberNotWhole
	// numberOutOfRange means it is one this build cannot hold.
	numberOutOfRange
)

// exactWhole is the largest magnitude at which a float64 still represents every
// whole number.
//
// Above it the gaps between representable values exceed one, so the number that
// arrives is not the number somebody typed, and accepting it would mean
// generating a different count than the page asks for.
const exactWhole = 1 << 53

// wholeNumber accepts the shapes a YAML integer arrives in.
//
// Every conversion to int is guarded, and that is not defensive habit. Go
// leaves the result undefined when the source does not fit, so the unguarded
// form turned a number above the range into a negative one, which was then
// reported as a count below zero: a message about something the user never
// wrote.
func wholeNumber(node any) (int, numberFault) {
	switch value := node.(type) {
	case int:
		return value, numberOK
	case int64:
		if value > math.MaxInt || value < math.MinInt {
			return 0, numberOutOfRange
		}
		return int(value), numberOK
	case uint64:
		if value > uint64(math.MaxInt) {
			return 0, numberOutOfRange
		}
		return int(value), numberOK
	case float64:
		// A number written with a decimal point is not a count. Saying so
		// beats rounding it and generating a different number of rows.
		if math.IsNaN(value) || math.IsInf(value, 0) || value != math.Trunc(value) {
			return 0, numberNotWhole
		}
		// Both bounds are compared as floats, because converting first is the
		// undefined step this guard exists to avoid.
		if value > exactWhole || value < -exactWhole {
			return 0, numberOutOfRange
		}
		if value > float64(math.MaxInt) || value < float64(math.MinInt) {
			return 0, numberOutOfRange
		}
		return int(value), numberOK
	default:
		return 0, numberNotWhole
	}
}

// parseSpan reads how many children one parent has, written as 0-5.
func parseSpan(text string) (int, int, bool) {
	low, high, found := strings.Cut(text, "-")
	if !found {
		return 0, 0, false
	}
	first, err := strconv.Atoi(strings.TrimSpace(low))
	if err != nil {
		return 0, 0, false
	}
	second, err := strconv.Atoi(strings.TrimSpace(high))
	if err != nil {
		return 0, 0, false
	}
	return first, second, true
}

var reference = regexp.MustCompile(`\$\{([^}]*)\}`)

// referencesIn pulls the names an expression mentions.
func referencesIn(expression string) []string {
	var names []string
	for _, match := range reference.FindAllStringSubmatch(expression, -1) {
		name := strings.TrimSpace(match[1])
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// firstLine keeps a parser message to one line. The library prints the
// offending source underneath, which is useful in a terminal and unreadable
// inside a four part failure.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(line)
}

// refusedBeforeParsing runs the checks that have to answer before the parser
// is handed anything, and reports whether one of them refused.
//
// All of them guard the same thing for the same reason: the cost they exist to
// avoid belongs to the parser, and there is no undoing it once it has begun.
// They are kept together so that the next one lands beside them rather than
// somewhere a reader has to go looking.
//
// # The rule this block has to keep
//
// Everything here reads bytes and nothing here parses. A third check used to
// live below that counted the documents in the file by running the parser over
// all of it, which spent exactly the cost the other two exist to refuse. It now
// reads the syntax tree that Decode builds once. Anything added here has to
// answer from the bytes alone, or it belongs on the other side of the parse.
func refusedBeforeParsing(r *reader, data []byte) bool {
	// The size is not quoted back, and that is deliberate rather than terse.
	// Knowing it would mean holding the whole file, which is the cost this
	// limit exists to refuse, so whoever reads the file stops one byte past
	// the limit and this can only say which side of it the recipe fell on.
	if len(data) > MaxSize {
		r.report("", fmt.Sprintf("the recipe is larger than the %d kB this build reads at most", MaxSize/1024),
			fmt.Sprintf("a recipe up to %d kB", MaxSize/1024),
			"split it into several recipes, or check that the file is a recipe and not something else")
		return true
	}

	// The encoding is answered before the shape, because a file in the wrong
	// one produces a complaint about its shape that names the wrong culprit.
	if encoding, wrong := notUTF8(data); wrong {
		r.report("", "the recipe is saved as "+encoding+" and this build reads UTF-8",
			"UTF-8, with or without a byte order mark",
			"save the file again as UTF-8. In the Notepad save dialog that is the entry named UTF-8, not the ones named Unicode")
		return true
	}

	// A single byte encoding announces nothing, so it is caught by the bytes
	// failing to be UTF-8 rather than by a mark.
	//
	// # Why this is not the same check as the one above
	//
	// The marks above are a declaration. This is a measurement, and it is the
	// one that catches the encoding the people this tool is for are most likely
	// to produce: on a Polish Windows the ANSI entry of a save dialog means
	// Windows-1250, and older editors and spreadsheet exports still default to
	// it. Nothing in such a file says so.
	//
	// Measured 2026-08-03, before this: a recipe written in Windows-1250 was
	// accepted as usable, and the entity named zażółć reached the model as
	// za followed by four replacement characters. The report said the recipe
	// could be used, so nobody was told, and the damaged name is what a
	// generated column would have been headed with.
	//
	// YAML itself requires UTF-8, UTF-16 or UTF-32, so refusing anything else
	// follows the format rather than adding a rule of ours.
	if !utf8.Valid(data) {
		r.report("", "the recipe is not valid UTF-8, so it was saved in some other encoding",
			"UTF-8, with or without a byte order mark",
			"save the file again as UTF-8. A Polish Windows writes Windows-1250 when the save dialog says ANSI")
		return true
	}

	// One pass over bytes already in hand, comparing four characters, which is
	// free next to what it refuses.
	if depth := deepestNesting(data); depth > MaxNestingDepth {
		r.report("", fmt.Sprintf("the recipe nests brackets %d deep and this build reads at most %d", depth, MaxNestingDepth),
			fmt.Sprintf("nesting up to %d deep, counting square brackets and braces together", MaxNestingDepth),
			"nothing written by hand nests anywhere near this deep, so check that the file is a recipe and not something else")
		return true
	}

	return false
}

// mark is the byte order mark Notepad writes at the head of a file by default.
var mark = []byte{0xEF, 0xBB, 0xBF}

// The byte order marks of the encodings this build does not read.
//
// # Why these are named rather than left to the parser
//
// The same reason stripBOM exists, met on the other half of the same dialog.
// Notepad offers UTF-8, UTF-16 LE under the name Unicode, and UTF-16 BE. A
// recipe saved as either of the last two reaches the parser as text with a nul
// byte between every letter, and the complaint that comes back is that the
// recipe has to be a mapping of keys, followed by advice to add a version line
// which is already the first line of the file.
//
// Measured on 2026-08-03, before this: a UTF-16 file beginning with version:
// was refused with exactly that message. The reader is told to fix something
// that is not wrong, and nothing anywhere names the encoding.
//
// Refused rather than transcoded, on purpose. A recipe is untrusted input and
// converting encodings is a new surface for no gain: naming the cause removes
// the whole of the reader's loss, and the fix is one entry of the dialog they
// already used.
var (
	mark16LE = []byte{0xFF, 0xFE}
	mark16BE = []byte{0xFE, 0xFF}
	mark32LE = []byte{0xFF, 0xFE, 0x00, 0x00}
	mark32BE = []byte{0x00, 0x00, 0xFE, 0xFF}
)

// notUTF8 names the encoding when a file announces one this build cannot read.
//
// The four byte marks are tested first, because UTF-32 LE begins with the two
// bytes of UTF-16 LE and would otherwise be named as the wrong one.
func notUTF8(data []byte) (string, bool) {
	switch {
	case bytes.HasPrefix(data, mark32LE):
		return "UTF-32 little endian", true
	case bytes.HasPrefix(data, mark32BE):
		return "UTF-32 big endian", true
	case bytes.HasPrefix(data, mark16LE):
		return "UTF-16 little endian", true
	case bytes.HasPrefix(data, mark16BE):
		return "UTF-16 big endian", true
	}
	return "", false
}

// stripBOM removes a leading byte order mark.
//
// # Why this is ordinary rather than exotic
//
// Notepad on Windows writes one by default, and the people this tool is for
// are testers on Windows. A recipe arriving with one is a recipe somebody
// wrote normally.
//
// Measured before this existed: the mark became part of the first key, so the
// version was read as an unknown field and the recipe was refused for not
// declaring a schema version. The hint told the reader to add a line that was
// already the first line of their file, and the second message, which named
// the real culprit, was unreadable because the mark does not render.
//
// Only a leading mark and only one. Further into the file it is ordinary text
// that a comment or a value has every right to carry.
func stripBOM(data []byte) []byte {
	return bytes.TrimPrefix(data, mark)
}

// deepestNesting returns the deepest run of unclosed flow brackets, counting
// square brackets and braces together.
//
// It reads bytes rather than a parsed document, and that is the whole point:
// the cost being guarded against belongs to the parser, so the answer has to
// arrive before the parser does. One pass, four characters compared, no
// allocation.
//
// # Why both kinds, and why one counter
//
// This counted square brackets alone until 2026-08-03. The note beside the
// limit cleared braces on the strength of a single measurement at sixteen
// kilobytes, which is six percent of the largest recipe this build accepts, and
// a conclusion drawn at six percent of a range does not hold across it.
// Measured on the same shape written with braces:
//
//	4 000 levels    20 kB    27 MB
//	8 000 levels    40 kB   194 MB
//	16 000 levels   80 kB   597 MB
//
// The growth is faster than linear, so the size limit alone leaves the same
// machine killed by the operating system that the square bracket limit exists
// to prevent. It is the same failure through the other spelling of the same
// idea.
//
// One counter rather than two, because the two nest inside each other. Separate
// counters would each stay under the limit while the document reached twice it,
// which is the arithmetic a hostile recipe is written to find.
//
// A bracket inside a quoted string counts towards the depth as well. That is
// deliberate and it errs in the safe direction, because a quoted bracket costs
// the parser nothing: the scan can therefore refuse a recipe nobody would
// write, and can never accept one that is a bomb. Teaching it about quoting
// would buy a case that does not occur and cost a second parser to keep
// correct.
func deepestNesting(data []byte) int {
	depth, deepest := 0, 0
	for _, symbol := range data {
		switch symbol {
		case '[', '{':
			depth++
			if depth > deepest {
				deepest = depth
			}
		case ']', '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return deepest
}

// shortest is how much of any one part of a message survives, in characters.
//
// Characters rather than bytes, and the unit is named because for the text
// this tool handles the two are different numbers. It is set well above every
// message this package writes, so it only ever cuts something a recipe put
// there.
const shortest = 200

// shorten trims one part of a message to something a report can carry.
//
// # Why this exists, found by review rather than by a test
//
// A recipe is untrusted input and it has a size limit. The report about it had
// none. Measured on 2026-08-02: a recipe carrying one two hundred kilobyte
// value produced a two hundred kilobyte error message, so hostile input came
// back amplified into somebody's build log, and with the machine readable flag
// onto the data channel where a pipeline parses it.
//
// It is applied where a problem is recorded rather than at the dozen places
// that build one. Names of entities, fields and keys all come from the recipe
// and all end up inside a sentence, so bounding the sentence bounds every one
// of them, including the ones somebody writes next year.
//
// The cut lands on a character boundary. Cutting on a byte would split a
// multi byte character and put invalid UTF-8 into that same machine readable
// report.
func shorten(text string) string {
	count := 0
	for index := range text {
		count++
		if count > shortest {
			return text[:index] + "... (shortened)"
		}
	}
	return text
}
