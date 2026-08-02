// Package oracle holds validators written from the published description of
// an identifier rather than from this project's own generator.
//
// # Why a second implementation of the same rules is not waste
//
// The generator and its test are usually written by the same hand in the same
// hour. When the hand is wrong, both are wrong in the same direction and agree
// with each other, and the test reports success. A golden file made from that
// output then preserves the mistake and defends it for years.
//
// So the validators here are deliberately built a second way. They are shaped
// around the wording of the source rather than around the arithmetic that was
// convenient in the provider, and where a number could be either restated or
// quoted, it is quoted. The two implementations meeting in the middle is the
// evidence. Two implementations sharing an ancestor would be a ceremony.
//
// A third layer sits outside this repository, in a script that computes the
// same answers in another language. That one catches what a shared idiom of
// one language would hide.
//
// Nothing here is used by the product. It exists for tests, and a guard makes
// sure it never ships.
package oracle

import "strings"

// peselWeights is the sequence of weights exactly as the government
// description writes it.
//
// It is kept as the source string and taken apart at start up rather than
// typed out as a list of numbers. A list would be a second telling of the same
// fact, and the point of this package is that it does not retell anything it
// can quote.
const peselWeights = "1-3-7-9-1-3-7-9-1-3"

// peselCenturies are the month ranges the description gives, each with the
// century it means.
//
// The provider stores an offset to add to a month. This stores the range a
// month falls into. The two are the same statement seen from opposite ends,
// which is the point: an offset written down wrongly does not become a range
// written down wrongly.
var peselCenturies = []struct {
	lowMonth  int
	highMonth int
	century   int
}{
	{1, 12, 1900},
	{21, 32, 2000},
	{41, 52, 2100},
	{61, 72, 2200},
	{81, 92, 1800},
}

// PESELFinding is one reason a value is not a valid PESEL. An empty list means
// the value is valid.
type PESELFinding string

// CheckPESEL reports every rule a value breaks.
//
// It returns all of them rather than the first, because a test that fixes one
// rule at a time hides how far off a value really was.
func CheckPESEL(value string) []PESELFinding {
	var findings []PESELFinding

	if len(value) != 11 {
		return []PESELFinding{PESELFinding("length is " + itoa(len(value)) + ", the scheme says eleven")}
	}
	for _, symbol := range value {
		if symbol < '0' || symbol > '9' {
			return []PESELFinding{"the value contains something that is not a digit"}
		}
	}

	digit := func(position int) int { return int(value[position] - '0') }
	number := func(from, to int) int {
		total := 0
		for position := from; position < to; position++ {
			total = total*10 + digit(position)
		}
		return total
	}

	// The date. The century comes from the range the month falls into.
	month := number(2, 4)
	matched := -1
	for index, span := range peselCenturies {
		if month >= span.lowMonth && month <= span.highMonth {
			matched = index
			break
		}
	}
	if matched < 0 {
		findings = append(findings, PESELFinding("month field "+itoa(month)+" falls in no century range the scheme defines"))
	} else {
		span := peselCenturies[matched]
		// The real month is the offset into the range that was quoted, not a
		// number subtracted by arithmetic worked out here.
		year := span.century + number(0, 2)
		realMonth := month - span.lowMonth + 1
		day := number(4, 6)
		if !peselDateExists(year, realMonth, day) {
			findings = append(findings, PESELFinding("the encoded date does not exist"))
		}
	}

	// The check digit. Stated as a sum that has to come out to a round number,
	// which is the same rule as subtracting from ten and avoids repeating the
	// step where a result of ten has to become nought.
	total := 0
	for position, weight := range strings.Split(peselWeights, "-") {
		total += digit(position) * int(weight[0]-'0')
	}
	if (total+digit(10))%10 != 0 {
		findings = append(findings, "the check digit does not agree with the first ten digits")
	}

	return findings
}

// peselDateExists is written from the calendar rather than from the standard
// library, which would normalise the thirtieth of February into March instead
// of refusing it.
func peselDateExists(year, month, day int) bool {
	if month < 1 || month > 12 || day < 1 {
		return false
	}
	longMonths := map[int]bool{1: true, 3: true, 5: true, 7: true, 8: true, 10: true, 12: true}
	switch {
	case longMonths[month]:
		return day <= 31
	case month != 2:
		return day <= 30
	case year%400 == 0 || (year%4 == 0 && year%100 != 0):
		return day <= 29
	default:
		return day <= 28
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits []byte
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}
