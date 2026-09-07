package oracle

import "strings"

// The rule this file implements.
//
// The ISBN Users' Manual of the International ISBN Agency, seventh edition,
// London 2017, appendix A1.1, downloaded from the agency's own site on
// 2026-09-07. The manual states the calculation, works an example of it, prints
// every intermediate product of that example, and then states the same rule a
// second time as a condition on the finished number.
//
// That second statement is what this file implements, and it is the reason this
// validator is worth having. The provider computes a check digit and compares
// it. This asks the manual's other question instead: whether the weighted sum
// of all thirteen digits divides by ten. The two are the same rule and they are
// not the same operation, so a fold written the wrong way round in one of them
// is not written the same wrong way round in the other.
//
// Nothing here rests on a measurement.

// isbnWeightingRuleAsPublished is the manual's sentence for how the twelve
// digits are weighted, in the words it uses.
//
// Held rather than paraphrased because the provider counts the alternation from
// the left and the article number in this build counts an identical alternation
// from the right. Which end the count starts from is exactly the thing a
// paraphrase loses.
const isbnWeightingRuleAsPublished = "Each of the first 12 digits of the ISBN is alternately multiplied by 1 and 3"

// isbnValidityRuleAsPublished is the condition the manual puts on a finished
// number, in the words it uses.
//
// This is the sentence the check below is written from. It is quoted rather
// than restated because a restatement would have been written by whoever wrote
// the check, and would then agree with the check for that reason rather than
// because the manual says so.
const isbnValidityRuleAsPublished = "The sum of the weighted products of the first 12 digits plus the check digit must be divisible by 10 without a remainder for the ISBN to be valid."

// isbnLengthRuleAsPublished is the agency's own statement of how long an ISBN
// is, from the page on its website describing the number.
//
// It is held because it carries the whole reason the ten character form is
// absent from this build. A later reader who wants to add it will find the
// sentence that argued against it standing next to the arithmetic, rather than
// only in a document nobody reads at the same time as the code.
const isbnLengthRuleAsPublished = "ISBNs were 10 digits in length up to the end of December 2006, but since 1 January 2007 they now always consist of 13 digits."

// ismnPrefixRuleAsPublished is the International ISMN Agency's statement of
// what a music number looks like.
//
// A different standard's document, quoted here because this validator refuses
// values under that prefix and a refusal should be able to point at the source
// which says the space belongs to somebody else.
const ismnPrefixRuleAsPublished = "The ISMN consists of 13 digits starting with 979-0"

// ISBNFinding is one reason a value is not a valid thirteen digit book number.
// An empty list means the value is valid.
type ISBNFinding string

// CheckISBN reports every rule a value breaks.
//
// It answers the same question as the provider's Validate and deliberately not
// a wider one. Whether a value sits in the window the generator uses is not
// asked here, because the provider does not refuse values outside it either and
// an oracle asking more would disagree with a value neither of them calls
// wrong.
func CheckISBN(value string) []ISBNFinding {
	if !isbnPublishedRuleStillNamesTheAlternation() {
		// The weights below are read from this sentence. If the sentence stops
		// saying what they do, they are supported by nothing and should say so
		// rather than carry on being right by accident.
		return []ISBNFinding{"the published weighting rule no longer names an alternation of one and three"}
	}
	if !isbnPublishedRuleStillNamesDivisibilityByTen() {
		return []ISBNFinding{"the published validity rule no longer names divisibility by ten"}
	}
	if len(value) != isbnPublishedLength() {
		return []ISBNFinding{"a book number is thirteen digits"}
	}
	for at := 0; at < len(value); at++ {
		if value[at] < '0' || value[at] > '9' {
			return []ISBNFinding{"the value carries a character that is not a digit"}
		}
	}
	if strings.HasPrefix(value, ismnPublishedPrefix()) {
		return []ISBNFinding{"the value begins with the prefix the music number occupies"}
	}
	if !strings.HasPrefix(value, "978") && !strings.HasPrefix(value, "979") {
		return []ISBNFinding{"a book number begins 978 or 979"}
	}

	// The manual's second statement taken literally: weight the twelve, add the
	// check digit as printed, and ask whether the total divides by ten. The
	// check digit takes the light weight, because it stands in the thirteenth
	// place and the alternation begins light.
	total := 0
	for at := 0; at < len(value); at++ {
		weight := 1
		if at%2 == 1 {
			weight = 3
		}
		total += int(value[at]-'0') * weight
	}

	var findings []ISBNFinding
	if total%10 != 0 {
		findings = append(findings, "the weighted sum of all thirteen digits does not divide by ten")
	}
	return findings
}

// ISBNPublishedLengthSentence hands back the agency's sentence about length, so
// a test can hold the code's thirteen against the source's rather than against
// a second copy of itself.
func ISBNPublishedLengthSentence() string { return isbnLengthRuleAsPublished }

// ISMNPublishedPrefix returns the prefix read out of the music agency's
// sentence.
//
// Read rather than typed, so the refusal in this file is compared against the
// sentence the other agency prints instead of against a constant somebody
// entered next to the check.
func ISMNPublishedPrefix() string { return ismnPublishedPrefix() }

// ismnPublishedPrefix takes the digits out of the quoted sentence, dropping the
// hyphen the agency writes inside them.
func ismnPublishedPrefix() string {
	for _, field := range strings.Fields(ismnPrefixRuleAsPublished) {
		trimmed := strings.TrimSuffix(field, ".")
		if !strings.Contains(trimmed, "-") {
			continue
		}
		joined := strings.ReplaceAll(trimmed, "-", "")
		if joined == "" {
			continue
		}
		allDigits := true
		for at := 0; at < len(joined); at++ {
			if joined[at] < '0' || joined[at] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return joined
		}
	}
	return ""
}

// isbnPublishedLength reads the length out of the agency's sentence rather than
// holding a second copy of the number thirteen.
//
// The sentence names two lengths and this takes the one it says numbers now
// have, which is the second. Reading the wrong one of the two would make this
// validator reject every current book number, which is a loud failure rather
// than a quiet one.
func isbnPublishedLength() int {
	after := isbnLengthRuleAsPublished
	at := strings.Index(after, "now always consist of ")
	if at < 0 {
		return 0
	}
	rest := after[at+len("now always consist of "):]
	length := 0
	for index := 0; index < len(rest); index++ {
		symbol := rest[index]
		if symbol < '0' || symbol > '9' {
			break
		}
		length = length*10 + int(symbol-'0')
	}
	return length
}

// isbnPublishedRuleStillNamesTheAlternation reports whether the held sentence
// still carries the parts the weighting depends on.
//
// It reads the sentence rather than merely storing it, which is the difference
// between a constant that documents and a constant that holds. A quotation
// nothing consults is a comment with quotation marks around it, and the
// compiler says so.
func isbnPublishedRuleStillNamesTheAlternation() bool {
	sentence := strings.ToLower(isbnWeightingRuleAsPublished)
	for _, part := range []string{"first 12 digits", "alternately multiplied", "1 and 3"} {
		if !strings.Contains(sentence, part) {
			return false
		}
	}
	return true
}

// isbnPublishedRuleStillNamesDivisibilityByTen reports whether the held
// condition still says what the arithmetic below does with it.
func isbnPublishedRuleStillNamesDivisibilityByTen() bool {
	sentence := strings.ToLower(isbnValidityRuleAsPublished)
	for _, part := range []string{"plus the check digit", "divisible by 10", "without a remainder"} {
		if !strings.Contains(sentence, part) {
			return false
		}
	}
	return true
}
