package oracle

import "strings"

// nipWeightsAsPublished is the weight table exactly as the source prints it,
// position labels included.
//
// The provider stores nine bare numbers in an array. This keeps the sentence
// and takes it apart, so a weight typed into the wrong slot there does not get
// typed into the same wrong slot here. Reading the labels also means a table
// that lost a position is caught rather than silently shortened.
const nipWeightsAsPublished = "C1 6 C2 5 C3 7 C4 2 C5 3 C6 4 C7 5 C8 6 C9 7"

// The rule this file implements, quoted from the country sheet the Ministry of
// Finance of Poland publishes on the European TIN Portal:
//
//	1. Multiply the values of each position by the corresponding weight
//	2. Add up the results of the above multiplications
//	3. Get modulo 11 of the result of the previous addition
//	4. Check digit = remainder (if remainder = 10, the TIN is not valid)
//
// And the worked example it gives, which is the thing the PESEL description
// never provided:
//
//	2234567895
//	2*6=12, 2*5=10, 3*7=21, 4*2=8, 5*3=15, 6*4=24, 7*5=35, 8*6=48, 9*7=63
//	12+10+21+8+15+24+35+48+63=236
//	236 % 11 = 5
//	Check digit = 5

// nipHyphenPositions are the offsets separators may sit at in the two ways
// Poland writes the number.
//
// The source knows no separators at all: it says ten numeric characters. That
// this project accepts two hyphenated forms is its own decision, so it is
// restated here rather than derived, and stated as the set of offsets rather
// than as groups to walk. The provider walks the groups. Two shapes of the same
// rule disagree when one of them is wrong.
var nipHyphenPositions = [][]int{
	{3, 7, 10}, // 123-456-32-18
	{3, 6, 9},  // 123-45-63-218
}

// NIPFinding is one reason a value is not a valid NIP. An empty list means the
// value is valid.
type NIPFinding string

// CheckNIP reports every rule a value breaks.
func CheckNIP(value string) []NIPFinding {
	digits, ok := nipDigits(value)
	if !ok {
		return []NIPFinding{"the value is not ten digits, bare or in one of the two hyphenated forms"}
	}

	var findings []NIPFinding

	weights := nipPublishedWeights()
	if len(weights) != 9 {
		// Loud rather than truncated. A weight table that lost a position would
		// otherwise agree with a provider that lost the same one.
		return []NIPFinding{"the published weight table did not come apart into nine weights"}
	}

	total := 0
	for position, weight := range weights {
		total += digits[position] * weight
	}

	// The source states step four as an equality between the check digit and
	// the remainder. Stated here as a divisibility instead: the weighted total
	// less the check digit has to be a multiple of eleven. It is the same claim
	// and it folds in the case the source calls out separately, because a
	// remainder of ten would need a check digit of ten and ten is not a digit.
	if (total-digits[9])%11 != 0 {
		findings = append(findings, "the check digit does not agree with the first nine digits")
	}

	// Said separately as well, because it is a different fact about the value:
	// no digit whatsoever would have made this one a NIP.
	if total%11 == 10 {
		findings = append(findings, "the first nine digits leave a remainder of ten, which no check digit can match")
	}

	return findings
}

// nipPublishedWeights takes the source sentence apart into the nine weights,
// reading the number after each position label.
func nipPublishedWeights() []int {
	fields := strings.Fields(nipWeightsAsPublished)
	weights := make([]int, 0, len(fields)/2)
	for index := 1; index < len(fields); index += 2 {
		if len(fields[index]) != 1 || fields[index][0] < '0' || fields[index][0] > '9' {
			return nil
		}
		weights = append(weights, int(fields[index][0]-'0'))
	}
	return weights
}

// nipDigits reads the ten digits out of any accepted form.
func nipDigits(value string) ([]int, bool) {
	if len(value) == 10 {
		return nipOnlyDigits(value)
	}
	for _, positions := range nipHyphenPositions {
		if bare, ok := nipStripAt(value, positions); ok {
			return nipOnlyDigits(bare)
		}
	}
	return nil, false
}

// nipStripAt removes separators sitting at exactly the named offsets, and
// refuses a value whose separators sit anywhere else.
func nipStripAt(value string, positions []int) (string, bool) {
	if len(value) != 10+len(positions) {
		return "", false
	}
	var bare strings.Builder
	at := 0
	for index := 0; index < len(value); index++ {
		if at < len(positions) && index == positions[at] {
			if value[index] != '-' {
				return "", false
			}
			at++
			continue
		}
		bare.WriteByte(value[index])
	}
	return bare.String(), true
}

func nipOnlyDigits(value string) ([]int, bool) {
	if len(value) != 10 {
		return nil, false
	}
	digits := make([]int, 0, 10)
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return nil, false
		}
		digits = append(digits, int(value[index]-'0'))
	}
	return digits, true
}
