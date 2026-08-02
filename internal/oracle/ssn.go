package oracle

import "strings"

// The Social Security number, checked from the Administration's own sentences.
//
// The provider takes those sentences apart into numbers and compares numbers.
// This holds the sentences themselves and compares the digits of a value
// against the digits printed in them. The two meeting is evidence that the
// wording was read correctly, which is the one thing a second implementation
// by the same hand can still establish.
const (
	// ssnAreaSentence is about assignment. It names the area numbers that are
	// never handed out.
	ssnAreaSentence = "Randomization also introduced previously unassigned area " +
		"numbers for assignment excluding area numbers 000, 666 and 900-999."
	// ssnVoidSentence is a stronger claim than the one above, and the
	// difference is the whole reason this identifier is corruptible at all.
	// One says a number was never given out, the other says a number is not a
	// number.
	ssnVoidSentence = "SSN randomization will not assign group number 00 or " +
		"serial number 0000. SSNs containing group number 00 or serial " +
		"number 0000 will continue to be invalid."
)

// SSNFinding is one reason a value is not a well formed SSN. An empty list
// means the value is well formed.
type SSNFinding string

// CheckSSN reports every rule of shape a value breaks.
//
// It answers the question an ordinary application's validator answers, and
// deliberately not the question of whether the number could ever have been
// issued. Area 666 passes here. It has to: a validator refusing it would
// refuse every correct value this tool produces, and then the corruption
// aimed at never issued ranges would have nothing to distinguish itself from.
// SSNNeverAssigned is the other half.
func CheckSSN(value string) []SSNFinding {
	digits, ok := ssnDigits(value)
	if !ok {
		return []SSNFinding{"the value is not nine digits, written 123456789 or 123-45-6789"}
	}

	var findings []SSNFinding
	// The two parts the source names outright, compared against the digits
	// printed in its own sentence rather than against a rule worked out here.
	if digits[3:5] == ssnVoidPart("group") {
		findings = append(findings, "the group number is the one the scheme calls invalid")
	}
	if digits[5:9] == ssnVoidPart("serial") {
		findings = append(findings, "the serial number is the one the scheme calls invalid")
	}
	// The area written entirely in noughts. The source puts it among the
	// numbers never handed out rather than among the two it calls invalid, so
	// this is the weaker of the two claims and is worth knowing as such. It is
	// also the number the Administration tells the public to print when a
	// number must not be anybody's.
	if digits[0:3] == ssnZeroArea() {
		findings = append(findings, "the area number is the one written entirely in noughts")
	}
	return findings
}

// SSNNeverAssigned reports whether the Administration would never have issued
// this number.
//
// Everything CheckSSN refuses is in here too, plus the areas that look like
// ordinary numbers and are handed out to nobody. That gap between the two
// answers is exactly what makes a generated value safe and still plausible.
func SSNNeverAssigned(value string) bool {
	digits, ok := ssnDigits(value)
	if !ok {
		return false
	}
	for _, listed := range ssnNeverAssignedAreas() {
		if ssnAreaMatches(digits[0:3], listed) {
			return true
		}
	}
	return digits[3:5] == ssnVoidPart("group") || digits[5:9] == ssnVoidPart("serial")
}

// ssnNeverAssignedAreas pulls the list out of the sentence instead of
// restating it, so that the list in this file cannot differ from the quotation
// above it.
func ssnNeverAssignedAreas() []string {
	const marker = "excluding area numbers "
	start := strings.Index(ssnAreaSentence, marker)
	if start < 0 {
		return nil
	}
	list, _, _ := strings.Cut(ssnAreaSentence[start+len(marker):], ".")

	var areas []string
	for _, token := range strings.Split(strings.ReplaceAll(list, " and ", ", "), ",") {
		if trimmed := strings.TrimSpace(token); trimmed != "" {
			areas = append(areas, trimmed)
		}
	}
	return areas
}

// ssnAreaMatches compares an area against one entry of the quoted list, which
// is either a single number or a span written with a dash.
//
// The comparison is on the text. Every area in the list is three digits and so
// is every area in a value, so comparing the strings orders them the same way
// comparing the numbers would, without turning either into a number first.
func ssnAreaMatches(area, listed string) bool {
	low, high, isSpan := strings.Cut(listed, "-")
	if !isSpan {
		return area == listed
	}
	return area >= low && area <= high
}

// ssnZeroArea picks the never assigned area written entirely in noughts out of
// the quoted list, by that shape rather than by its position in the list.
func ssnZeroArea() string {
	for _, area := range ssnNeverAssignedAreas() {
		if strings.Trim(area, "0") == "" {
			return area
		}
	}
	return ""
}

// ssnVoidPart pulls the digits the void sentence gives for a named part.
//
// It reads them out of the sentence rather than holding 00 and 0000 as
// literals, which is the difference between checking what the source says and
// checking what somebody remembered it saying.
func ssnVoidPart(part string) string {
	marker := part + " number "
	start := strings.Index(ssnVoidSentence, marker)
	if start < 0 {
		return ""
	}
	rest := ssnVoidSentence[start+len(marker):]

	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	return rest[:end]
}

// ssnDigits returns the nine digits, accepting either way of writing them.
//
// Both spellings are the same number. Refusing the one without separators
// would report a defect in a value that has none, which is the opposite of
// what a validator is for.
func ssnDigits(value string) (string, bool) {
	stripped := value
	if len(value) == 11 && value[3] == '-' && value[6] == '-' {
		stripped = value[0:3] + value[4:6] + value[7:11]
	}
	if len(stripped) != 9 {
		return "", false
	}
	for position := 0; position < len(stripped); position++ {
		if stripped[position] < '0' || stripped[position] > '9' {
			return "", false
		}
	}
	return stripped, true
}
