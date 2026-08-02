package recipeyaml

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
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
func (r *reader) count(where string, node any) int {
	number, ok := wholeNumber(node)
	if !ok {
		r.report(where, "the number of rows has to be a whole number", "", "write it without quotes, such as count: 100")
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
	number, ok := wholeNumber(node)
	if !ok {
		r.report(where, "the maximum length has to be a whole number", "",
			"write it without quotes, such as max_length: 30")
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

// wholeNumber accepts the shapes a YAML integer arrives in.
func wholeNumber(node any) (int, bool) {
	switch value := node.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case uint64:
		return int(value), true
	case float64:
		// A number written with a decimal point is not a count. Saying so
		// beats rounding it and generating a different number of rows.
		if value != float64(int(value)) {
			return 0, false
		}
		return int(value), true
	default:
		return 0, false
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

	// One pass over bytes already in hand, comparing two characters, which is
	// free next to what it refuses.
	if depth := deepestNesting(data); depth > MaxNestingDepth {
		r.report("", fmt.Sprintf("the recipe nests square brackets %d deep and this build reads at most %d", depth, MaxNestingDepth),
			fmt.Sprintf("nesting up to %d deep", MaxNestingDepth),
			"nothing written by hand nests anywhere near this deep, so check that the file is a recipe and not something else")
		return true
	}

	// A file holding a second recipe is refused rather than half read. The
	// decoder takes the first document and drops the rest without a word,
	// which is the shape of silence this project argues against: the caller
	// gets some of the entities they asked for and an exit code saying
	// everything went well.
	if count, err := recipesIn(data); err == nil && count > 1 {
		r.report("", fmt.Sprintf("the file holds %d recipes and this build reads one", count),
			"one recipe per file",
			"split them into separate files, or remove the --- separator between them")
		return true
	}

	return false
}

// mark is the byte order mark Notepad writes at the head of a file by default.
var mark = []byte{0xEF, 0xBB, 0xBF}

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

// recipesIn counts the documents of a file that carry something.
//
// 🔴 Documents carrying content, not raw documents, and the difference was
// measured here rather than taken from a neighbouring project that hit the
// same question. On this version of the parser a recipe followed by a trailing
// separator reads as two raw documents and one carrying one, so counting the
// raw ones would refuse a file somebody had every right to write. The
// neighbouring project measured a different shape catching them out, which is
// the reason for measuring rather than copying.
//
// Verified on six layouts: a bare recipe, a leading separator, a leading
// comment before a separator, a trailing separator, a trailing comment, and
// two recipes. Only the last counts as two.
func recipesIn(data []byte) (int, error) {
	file, err := parser.ParseBytes(data, 0)
	if err != nil {
		return 0, err
	}
	carrying := 0
	for _, document := range file.Docs {
		if document.Body != nil {
			carrying++
		}
	}
	return carrying, nil
}

// deepestNesting returns the deepest run of unclosed square brackets.
//
// It reads bytes rather than a parsed document, and that is the whole point:
// the cost being guarded against belongs to the parser, so the answer has to
// arrive before the parser does. One pass, two characters compared, no
// allocation.
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
		case '[':
			depth++
			if depth > deepest {
				deepest = depth
			}
		case ']':
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
