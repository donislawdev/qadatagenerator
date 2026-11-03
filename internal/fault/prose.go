package fault

import (
	"sort"
	"strconv"
	"strings"
)

// Sentences joins fragments written to follow other words into prose, one
// sentence each.
//
// A refusal that names several problems at once is a message, a colon and the
// problems, and every problem is written in lower case because the first one
// follows that colon. Joined with a full stop alone, the second read "...nobody
// can argue with. the inference about pesel carries no basis" (B-352). So the
// first fragment keeps its case and every later one has its first letter
// raised.
//
// 🔴 Raised only when the first word of the fragment is made of lower case
// letters and nothing else, which is how every sentence this tool writes
// begins ("the", "row", "an"). A fragment that begins with something taken
// from a file, the identifier of a manifest entry followed by its colon or a
// quoted value, comes through exactly as it was written, because a capital
// there would change what somebody sent.
func Sentences(parts []string) string {
	var said strings.Builder
	for at, part := range parts {
		if at > 0 {
			said.WriteString(". ")
			part = raisedWhenOurs(part)
		}
		said.WriteString(part)
	}
	return said.String()
}

// raisedWhenOurs raises the first letter of a fragment whose first word is
// lower case letters alone, and returns any other fragment as it is.
func raisedWhenOurs(part string) string {
	word, _, _ := strings.Cut(part, " ")
	if word == "" {
		return part
	}
	for _, letter := range []byte(word) {
		if letter < 'a' || letter > 'z' {
			return part
		}
	}
	return string(part[0]-'a'+'A') + part[1:]
}

// Reason is why some rows were refused, said of one row and of several, since
// an English verb has to agree with the rows in front of it. Both halves are
// written to follow the row numbers, so "is past the end of the dataset" and
// "are past the end of the dataset".
type Reason struct {
	One, Many string
}

// Grouped is rows refused for the same reason, kept together with their
// numbers.
//
// # Why grouped rather than one sentence a row
//
// Because the likely mistake repeats on every row. A record of four hundred
// thousand rows answering each one twice was four hundred thousand sentences in
// one message (B-339), and the reader needs the reason once, the first few rows
// it applies to and how many more there are.
type Grouped map[Reason][]int

// Add records that a row was refused for this reason. A row of nought stands
// for the record as a whole, and the reason is then said without a number.
func (g Grouped) Add(why Reason, row int) { g[why] = append(g[why], row) }

// Said is every reason led by the rows it applies to, sorted so that one
// record always produces one message.
func (g Grouped) Said() []string {
	reasons := make([]Reason, 0, len(g))
	for why := range g {
		reasons = append(reasons, why)
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i].One < reasons[j].One })
	parts := make([]string, 0, len(reasons))
	for _, why := range reasons {
		parts = append(parts, why.of(g[why]))
	}
	return parts
}

// shownRows is how many row numbers a reason names before it counts the rest.
//
// Three because the reader needs enough to see the pattern, a header counted as
// a record shows up as one row past the end and nothing else, and few enough
// that the sentence stays a sentence.
const shownRows = 3

// of says a reason about the rows it applies to, or on its own when it is about
// the record as a whole and names no row.
func (why Reason) of(rows []int) string {
	numbers := make([]string, 0, shownRows)
	for _, row := range rows {
		if row >= 1 && len(numbers) < shownRows {
			numbers = append(numbers, strconv.Itoa(row))
		}
	}
	switch {
	case len(numbers) == 0:
		return why.One
	case len(rows) == 1:
		return "row " + numbers[0] + " " + why.One
	case len(rows) > shownRows:
		return "rows " + strings.Join(numbers, ", ") + " and " + strconv.Itoa(len(rows)-shownRows) +
			" more " + why.Many
	}
	return "rows " + Listed(numbers) + " " + why.Many
}

// Listed joins names the way a sentence lists them.
func Listed(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
