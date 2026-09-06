package oracle

import "strings"

// The rule this file implements.
//
// The GS1 General Specifications, release 26.0 ratified January 2026, which GS1
// publishes itself and keeps current. Table 7-8 states the check digit
// algorithm and Table 7-9 works an example of it. Table 1-4 states what each
// prefix range is for.
//
// A second document states the same arithmetic independently: the ISBN Users'
// Manual of the International ISBN Agency, whose Appendix A1.1 works an example
// of its own. An ISBN-13 is a GTIN-13 carrying the prefix 978 or 979, so the two
// documents are describing one calculation. Nothing here rests on a
// measurement.

// eanWeightsAsPublished is the multiplier row of Table 7-8, written the way the
// standard prints it.
//
// # Why the whole row and not the twelve this scheme needs
//
// The standard prints one row of multipliers and lays every field against its
// right hand edge, so the twelve a GTIN-13 uses are the last twelve of these.
// The provider expresses that as an alternation walked backwards from the check
// digit. This holds the printed row and takes its tail, which is a different
// operation on a different shape.
//
// The two disagree the moment the alternation is started on the wrong foot, and
// that is the whole reason for the contrast. Starting it on the wrong foot
// would still produce the correct answer for this scheme, because twelve is
// even, and would produce the wrong one for the seventeen digit field. A second
// copy of the alternation would have agreed with the first and hidden it.
const eanWeightsAsPublished = "x3 x1 x3 x1 x3 x1 x3 x1 x3 x1 x3 x1 x3 x1 x3 x1 x3"

// eanCheckDigitRuleAsPublished is the third step of Table 7-8, in the words the
// standard uses for it.
//
// Held because the provider says the same thing as a remainder folded twice,
// which is compact and is not what the standard says. This file follows the
// sentence literally instead, so a fold written the wrong way round in one is
// not written the same wrong way round in the other.
const eanCheckDigitRuleAsPublished = "Subtract sum from nearest equal or higher multiple of ten = check digit"

// eanDemonstrationRowAsPublished is the row of Table 1-4 that gives this scheme
// its safe harbour, written as the standard writes it.
//
// # Why a prefix row is worth holding at all
//
// It is not arithmetic and no calculation reads it. It is held because the
// claim it supports is the strongest one this provider makes: that nothing it
// writes can belong to a real article. A claim of that weight should not rest
// on a three character string typed into a provider and never compared with
// anything.
const eanDemonstrationRowAsPublished = "952 Used for demonstrations and examples of the GS1 system"

// EANFinding is one reason a value is not a valid thirteen digit article
// number. An empty list means the value is valid.
type EANFinding string

// CheckEAN reports every rule a value breaks.
//
// It answers the same question as the provider's Validate and deliberately not
// a wider one. Whether a value sits in the demonstration prefix is not asked
// here, because the provider does not refuse values outside it either and an
// oracle asking more would disagree with a value neither of them calls wrong.
func CheckEAN(value string) []EANFinding {
	weights := eanPublishedWeights()
	if len(weights) != 17 {
		// Loud rather than truncated. A row that lost a multiplier would
		// otherwise quietly weigh the digits of every value wrongly.
		return []EANFinding{"the published weight row did not come apart into seventeen multipliers"}
	}
	if !eanPublishedRuleStillSaysMultipleOfTen() {
		// The fold below is written from this sentence. If the sentence stops
		// saying what the fold does, the fold is no longer supported by
		// anything and should say so rather than carry on being right by
		// accident.
		return []EANFinding{"the published check digit rule no longer names the nearest equal or higher multiple of ten"}
	}
	if len(value) != 13 {
		return []EANFinding{"an article number is thirteen digits"}
	}
	for at := 0; at < len(value); at++ {
		if value[at] < '0' || value[at] > '9' {
			return []EANFinding{"the value carries a character that is not a digit"}
		}
	}

	// The twelve multipliers a thirteen digit field uses are the last twelve of
	// the printed row, because the standard aligns the fields to its right edge.
	body := len(value) - 1
	tail := weights[len(weights)-body:]

	total := 0
	for at := 0; at < body; at++ {
		total += int(value[at]-'0') * tail[at]
	}

	var findings []EANFinding
	if want := eanNextMultipleOfTen(total) - total; value[12] != byte('0'+want) {
		findings = append(findings, "the check digit does not agree with the twelve digits before it")
	}
	return findings
}

// EANDemonstrationPrefix returns the prefix read out of the published row.
//
// It is read rather than typed, so that the provider's copy is compared against
// the sentence the standard prints instead of against a second copy of itself.
func EANDemonstrationPrefix() string {
	fields := strings.Fields(eanDemonstrationRowAsPublished)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// eanPublishedRuleStillSaysMultipleOfTen reports whether the held sentence
// still carries the three parts the fold below depends on.
//
// It reads the sentence rather than merely storing it, which is the difference
// between a constant that documents and a constant that holds. A quotation
// nothing consults is a comment with quotation marks around it, and the
// compiler says so.
func eanPublishedRuleStillSaysMultipleOfTen() bool {
	sentence := strings.ToLower(eanCheckDigitRuleAsPublished)
	for _, part := range []string{"subtract sum", "equal or higher multiple of ten", "check digit"} {
		if !strings.Contains(sentence, part) {
			return false
		}
	}
	return true
}

// eanNextMultipleOfTen is the published sentence taken literally: the nearest
// equal or higher multiple of ten.
//
// A total already on a multiple stays where it is, which is how a check digit
// of nought arises. The provider reaches the same number by folding a remainder
// twice, and the two shapes disagree if either is wrong.
func eanNextMultipleOfTen(total int) int {
	return ((total + 9) / 10) * 10
}

// eanPublishedWeights takes the printed row apart into its multipliers, reading
// the number that follows each multiplication sign.
func eanPublishedWeights() []int {
	fields := strings.Fields(eanWeightsAsPublished)
	weights := make([]int, 0, len(fields))
	for _, field := range fields {
		written := strings.TrimPrefix(strings.ToLower(field), "x")
		if written == "" {
			return nil
		}
		weight := 0
		for at := 0; at < len(written); at++ {
			symbol := written[at]
			if symbol < '0' || symbol > '9' {
				return nil
			}
			weight = weight*10 + int(symbol-'0')
		}
		weights = append(weights, weight)
	}
	return weights
}
