package recipeyaml

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
