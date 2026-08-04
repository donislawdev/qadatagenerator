package oracle

import "strings"

// The United States postal code, checked from the census bureau's own
// sentences.
//
// # What there is to be independent about here, which is not what it was for
// the identifiers
//
// The other validators in this package exist because an identifier carries
// arithmetic, and arithmetic written twice by one hand can still be checked for
// agreement. A postal code carries none. Nothing is computed, so there is no
// second computation to compare.
//
// What can go wrong instead is the reading, in two places, and both are held
// here as the words of the source rather than as a paraphrase in the provider.
//
// The first is what the codes are. They are tabulation areas, not postal codes,
// and the difference decides what the provider is allowed to call invalid. A
// validator built on the belief that the registry is the postal register would
// fault a real code the registry does not carry.
//
// The second is the term the census appends to a place name. The registry ships
// bare names, which means something was removed from every one of them, and a
// removal rule that is slightly wrong truncates names quietly. The terms are
// held here so that a test can assert no shipped name still carries one, using
// the list from the source rather than the list the extraction happened to use.
const (
	// zctaSentence says what a tabulation area is. The provider's refusal to
	// call an unlisted code invalid rests on it.
	zctaSentence = "ZIP Code Tabulation Areas or ZCTAs are a geographic product of the " +
		"U.S. Census Bureau created to allow mapping, display, and geographic analyses " +
		"of the United States Postal Service (USPS) Zone Improvement Plan (ZIP) Codes dataset."
	// zctaGapSentence is the stronger of the two and the one that decides the
	// question. It says the registry is not complete over the postal scheme, so
	// absence from it proves nothing about a value.
	zctaGapSentence = "Not all valid ZIP Codes are represented by a 2020 ZCTA."
)

// zctaCodeLength is the length the source gives for the code: the gazetteer
// calls its identifier the "Five digit ZIP Code Tabulation Area Census Code".
const zctaCodeLength = 5

// PlaceTerms are the words the census appends to a place name, keyed by the
// legal or statistical area description code that selects one.
//
// Taken from the bureau's own code list, where every one of these is marked a
// suffix, meaning it follows the name. The list is not trusted on its own: the
// extraction refuses any place whose name does not actually end in the term its
// code selects, and 2026-08-04 that held for every name carrying a term.
//
// Code 00 is deliberately absent. It selects no term, and the names under it are
// the balances of consolidated cities, written like "Indianapolis city
// (balance)". There is no rule in the source that reduces those to a bare name,
// so they are dropped from the registry rather than reduced by a rule invented
// here.
var PlaceTerms = map[string]string{
	"21": "borough",
	"25": "city",
	"35": "township",
	"37": "municipality",
	"43": "town",
	"47": "village",
	"53": "city and borough",
	"55": "comunidad",
	"57": "CDP",
	"62": "zona urbana",
	"CG": "consolidated government",
	"CN": "corporation",
	"MG": "metropolitan government",
	"UC": "urban county",
	"UG": "unified government",
}

// PostalFinding is one reason a value is not a well formed postal code. An
// empty list means the value is well formed.
type PostalFinding string

// CheckPostalCode reports every rule of shape a value breaks.
//
// It answers about shape and deliberately nothing else. Whether the registry
// carries the code is a different question, and the sentence held above is the
// reason it must stay a different question: the registry approximates the postal
// scheme and a real code can be missing from it.
func CheckPostalCode(value string) []PostalFinding {
	head, plus, hyphen, ok := splitPostalCode(value)
	if !ok {
		return []PostalFinding{"the value is not five digits, written 12345 or 12345-6789"}
	}

	var findings []PostalFinding
	if len(head) != postalLengthFromSentence() {
		findings = append(findings, "the code is not as long as the source says a tabulation area code is")
	}
	if !allDigitsOracle(head) {
		findings = append(findings, "the code carries something that is not a digit")
	}
	// The hyphen is asked about rather than the part after it. A value ending in
	// a hyphen has an empty tail, and a check reading only the tail would find
	// nothing to complain about and pass it. That is how this was found: the
	// provider refused 12345- and this validator did not.
	//
	// One condition rather than two, and that is not tidying. Written as a pair,
	// the length check and the digit check both fired for a value ending in a
	// hyphen, so breaking either one alone left the other covering it and the
	// mutation ran green. A condition nothing can break on its own is a
	// condition nobody is guarding, which this project has already met once in
	// an unused weight.
	if hyphen && (len(plus) != 4 || !allDigitsOracle(plus)) {
		findings = append(findings, "the part after the hyphen is not four digits")
	}
	return findings
}

// PlaceNameCarriesATerm reports whether a name still ends in one of the terms
// the census appends, which would mean the registry shipped a name that was
// never reduced.
//
// It returns the term so that a failing test can name it instead of only
// counting.
func PlaceNameCarriesATerm(name string) (string, bool) {
	for _, term := range PlaceTerms {
		if strings.HasSuffix(name, " "+term) {
			return term, true
		}
	}
	return "", false
}

// RegistryIsNotThePostalRegister is the sentence the provider's leniency rests
// on, exposed so a test can assert the claim is still the one being made.
func RegistryIsNotThePostalRegister() string { return zctaGapSentence }

// RegistryApproximatesThePostalScheme is the definition the registry is built
// on. It is exposed for the same reason as the sentence above: a constant no
// caller can reach is a quotation nobody can check, and this project has
// already found one dead constant that way.
func RegistryApproximatesThePostalScheme() string { return zctaSentence }

// postalLengthFromSentence takes the length from the source's own wording
// rather than from a constant chosen here, so that the two cannot drift.
func postalLengthFromSentence() int { return zctaCodeLength }

func splitPostalCode(value string) (head, plus string, hyphen, ok bool) {
	if value == "" {
		return "", "", false, false
	}
	head, plus, found := strings.Cut(value, "-")
	if !found {
		return value, "", false, len(value) == zctaCodeLength
	}
	return head, plus, true, len(head) == zctaCodeLength
}

func allDigitsOracle(value string) bool {
	for _, letter := range value {
		if letter < '0' || letter > '9' {
			return false
		}
	}
	return value != ""
}
