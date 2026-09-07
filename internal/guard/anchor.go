package guard

import (
	"regexp"
	"strings"
)

// The register of rules taken from outside this project, and of what proves we
// read each one correctly.
//
// # Why a second implementation is not enough
//
// Every identifier here is written twice on purpose: once as the provider and
// once as an independent validator built from the wording of the source. That
// catches a slip made in one place and not the other. It cannot catch a slip
// made in the reading, because both halves come from the same reading by the
// same hand in the same hour. When the reading is wrong the two agree with each
// other, the golden file preserves the agreement, and every test is green.
//
// The oracle package says this about itself in as many words: two independent
// implementations meeting is evidence the wording was read correctly, which is
// the one thing a second implementation by the same hand can still establish.
// It is not evidence the wording said what we think it said.
//
// So the only thing that settles the question is an anchor outside our own
// head: a source that performs the arithmetic itself, or real published values
// that reproduce it. Those anchors exist for some rules here and not others,
// and until this register there was nowhere that said which.
//
// # Rules, not providers
//
// One provider usually has more than one entry, because the shape of an
// identifier and the arithmetic over it are two separate questions to a source
// and one can be answered without the other. A register with a single row per
// provider would lose exactly that distinction, which is the most expensive
// thing this project has learned about reading sources.
//
// # What this closes and what it does not
//
// Provider identifiers are enumerated from the tree rather than kept by hand,
// so a provider added without an entry fails the build and an entry naming a
// provider that is gone fails too. What the register does not do is recompute
// anything. It makes the absence of an anchor visible. It does not make
// correctness automatic, and a state saying a rule was measured once is a
// statement about the past that the next session is expected to repeat rather
// than trust.

// AnchorState is how we know a rule taken from outside was read correctly.
//
// The four are ordered by the strength of the evidence, strongest first.
type AnchorState string

const (
	// ByWorkedExample means the source performs the calculation itself and a
	// test in this repository reproduces it. Strongest, because it runs on
	// every build and it did not come from our reading of the prose.
	ByWorkedExample AnchorState = "worked-example"
	// ByQuotedSentence means the source states the rule in words and the
	// independent validator holds those words rather than a paraphrase. This
	// is the right state where there is no arithmetic to anchor, and the
	// wrong state where there is one.
	ByQuotedSentence AnchorState = "quoted-sentence"
	// ByMeasurement means the rule was reproduced against real published
	// values, once, outside this repository. It is deliberately kept apart
	// from the two above and it is deliberately not called proven.
	//
	// The values cannot be stored here, because a correct identifier may
	// belong to a real subject, so the anchor is a description of a
	// measurement rather than a test. That makes it the weakest state that
	// still counts, and Detail has to carry enough for somebody to run the
	// measurement again instead of believing this line.
	ByMeasurement AnchorState = "measurement"
	// ByConstruction means the rule was made here and no source could ever
	// support it, because it answers a question no source is asking.
	//
	// # Why this is not a softer word for unanchored
	//
	// Unanchored means we looked and found nothing, and it carries a ratchet
	// because looking harder is always possible. This means there is nothing to
	// look for. A registry pairing a postal code with one of the several places
	// it overlaps has to pick one, and the source that lists the overlaps ranks
	// none of them and was never trying to.
	//
	// Leaving such a rule out of the register entirely was the alternative, and
	// it is worse. The register enumerates providers from the tree, not rules,
	// so a rule nobody entered is invisible to both halves of the check. A
	// provider would show two anchored rules while a third decided its output.
	//
	// # The thing that stops this becoming a way out
	//
	// Detail has to name which source was consulted and what it did not answer.
	// Without that sentence this state would be available to anybody who did not
	// feel like searching, which is exactly what Unanchored is for. It is checked
	// rather than trusted, the same way a measurement has to say how many and
	// when.
	ByConstruction AnchorState = "construction"
	// Unanchored means nothing outside our own reading supports this rule.
	// Said out loud so that it does not look like the four above. This list
	// may only shrink.
	Unanchored AnchorState = "unanchored"
)

// Anchor is one rule taken from outside, together with what supports it.
type Anchor struct {
	// Rule is what stops being trustworthy if this anchor is wrong. One
	// phrase, about the scheme rather than about our code.
	Rule string
	// Provider is the identifier of the provider that implements it, such as
	// PL.pesel. It has to match a provider that exists in the tree.
	Provider string
	// Source is the document the rule was taken from, named well enough to
	// be found again by somebody who has never seen it.
	Source string
	// State is the strength of the evidence.
	State AnchorState
	// Detail is what makes the state checkable by a human.
	//
	// For ByWorkedExample it is the name of the test that reproduces the
	// calculation. For ByMeasurement it is what was measured, how many
	// values, from where and when, so the measurement can be repeated. For
	// ByQuotedSentence it is where the quoted wording is held.
	Detail string
}

// UnanchoredCeiling is a ratchet. It may only be lowered.
//
// It stands at nought and the nought is real rather than aspirational: every
// rule in the register today has some anchor outside this project. Raising it
// to admit a new provider would be the same act as widening a test until the
// code passes.
const UnanchoredCeiling = 0

// Anchors is the register.
//
// It is not every fact this project took from the world. It is the rules whose
// being wrong would produce values that look correct and are not, which is the
// failure this product exists to avoid committing itself.
var Anchors = []Anchor{
	// --- PL.pesel ----------------------------------------------------------
	{
		Rule:     "the eleven digit shape, the century table and the digit carrying gender",
		Provider: "PL.pesel",
		Source:   "country sheet for Poland on the European TIN Portal, carrying a copyright note of the Ministry of Finance",
		State:    ByWorkedExample,
		Detail:   "TestTheWorkedExampleFromTheMinistryIsReproduced",
	},
	{
		Rule:     "the check digit weights",
		Provider: "PL.pesel",
		Source:   "country sheet for Poland on the European TIN Portal, which takes one number apart and prints every intermediate result",
		State:    ByWorkedExample,
		Detail:   "TestTheWorkedExampleFromTheMinistryIsReproduced",
	},

	// --- PL.nip ------------------------------------------------------------
	{
		Rule:     "the ten digit shape and a remainder of ten meaning the number is not valid",
		Provider: "PL.nip",
		Source:   "country sheet for Poland on the European TIN Portal",
		State:    ByQuotedSentence,
		Detail:   "the wording is held in internal/oracle/nip.go, weights labelled by position",
	},
	{
		Rule:     "the check digit weights",
		Provider: "PL.nip",
		// The number itself is deliberately not repeated here. It lives in the
		// test this entry names, and an identifier written down twice is an
		// identifier published twice. The earlier wording also named the wrong
		// one: it quoted an ascending run this repository uses to check the
		// three accepted spellings, not the number the source works out.
		Source: "country sheet for Poland on the European TIN Portal, which takes one number apart and prints every intermediate result",
		State:  ByWorkedExample,
		Detail: "TestTheWorkedExampleFromTheSourceIsReproduced",
	},

	// --- US.ssn ------------------------------------------------------------
	{
		Rule:     "the areas never assigned, and the stronger claim that group 00 and serial 0000 are invalid",
		Provider: "US.ssn",
		Source:   "ssa.gov randomization pages, read at the source rather than through a mirror",
		State:    ByQuotedSentence,
		Detail:   "both sentences are held verbatim in internal/oracle/ssn.go and compared digit by digit",
	},

	// --- US.aba ------------------------------------------------------------
	//
	// The clearest split in the register between a shape that was quoted and an
	// arithmetic that was measured. Both rows name the same provider and they
	// are not the same kind of evidence.
	{
		Rule:     "the nine digit shape and the allocation of the leading series, including that spare series are stock the Board may hand out",
		Provider: "US.aba",
		Source:   "ABA Routing Number Policy & Procedures version 9/20, section IV, published by LexisNexis Risk Solutions as the official routing number registrar",
		State:    ByQuotedSentence,
		Detail:   "the structure sentence, the allocation table and the note about the Board are held in internal/oracle/aba.go",
	},
	{
		Rule:     "the check digit weights",
		Provider: "US.aba",
		Source:   "no document we can read states the arithmetic. The registrar sends the reader to a publication it sells, the Federal Reserve directory FAQ is silent on it, and the rule is most likely inside the paid ANSI X9 standards",
		State:    ByMeasurement,
		Detail: "67 of 67 routing numbers printed in 12 CFR part 229 appendix A close under the weights 3 7 1, " +
			"measured 2026-08-03. A second sample of 3 from unrelated federal publishers agreed. Walking all 1000 " +
			"weight vectors over the first sample leaves exactly 10 solutions, the zero vector and the 9 multiples " +
			"of this one modulo 10, which are the same congruence, so no other solution exists. The numbers were " +
			"not kept, by decision DEC-109 point 4, so this cannot be recomputed here",
	},

	// --- US.postal ------------------------------------------------------------
	//
	// The only entry here for a scheme with no arithmetic in it, which changes
	// what an anchor can even be. Nothing is computed, so there is no worked
	// example to reproduce and no weights to force by measurement. What can be
	// read wrongly is what the data is and what was removed from it, so those
	// are the two rules, and both are quoted.
	//
	// The third rule is the one that made this register grow a fifth state. It
	// decides the output and no source will ever support it.
	{
		Rule:     "the five digit shape, and that a tabulation area is an approximation of the postal scheme rather than a register of it",
		Provider: "US.postal",
		Source:   "U.S. Census Bureau guidance on ZIP Code Tabulation Areas, and the gazetteer record layout calling the identifier a five digit code",
		State:    ByQuotedSentence,
		Detail:   "both sentences are held verbatim in internal/oracle/postal.go, and the one saying not every valid code is represented is what stops Validate from faulting a code the registry lacks",
	},
	{
		Rule:     "the term the census appends to a place name, which the registry removes to leave a bare name",
		Provider: "US.postal",
		Source:   "U.S. Census Bureau list of legal and statistical area description codes, where each of these terms is marked a suffix",
		State:    ByQuotedSentence,
		Detail:   "the fifteen terms are held in internal/oracle.PlaceTerms, a test asserts no shipped name still ends in one, and the extraction refuses any place whose name does not end in the term its code selects",
	},
	{
		Rule:     "which of the several places a code overlaps is the place of that code",
		Provider: "US.postal",
		Source:   "consulted the 2020 ZCTA to Place relationship file of the U.S. Census Bureau, which lists every intersection, up to seventy seven for one code, and does not rank them or name a principal place. No census document answers this, because the question is not one the census asks",
		State:    ByConstruction,
		Detail:   "the place sharing the largest land area wins, and a tie goes to the lower place identifier so the answer cannot depend on the order of rows. Stated in the header of the shipped file and in the provider comment, because a reader of either has to know it was computed here",
	},

	// --- iban ---------------------------------------------------------------
	//
	// The only provider here whose arithmetic is anchored better than its
	// generalisation, which is the reverse of every other row in this register.
	// The rule itself was found stated in words in a normative act after the
	// standards holding it turned out to be sold rather than published. What
	// rests on a measurement is not the rule but its reach: the act names the
	// values of two letters and this scheme uses twenty six.
	{
		Rule:     "the twenty eight character shape, the split into check number, settlement number and account, and the settlement number's own check digit",
		Provider: "iban",
		Source:   "regulation 7/2017 of the President of the National Bank of Poland, in force, Dz. Urz. NBP 2017 item 3, consolidated at 2019 item 9, sections 13 and 14 and annex 2",
		State:    ByQuotedSentence,
		Detail:   "the wording is held in internal/oracle/iban.go in the original Polish, with an English rendering beside it, and the length is pulled out of the quoted sentence rather than typed twice",
	},
	{
		Rule:     "the modulus, the rotation of four characters, the accepted remainder of one and the check number being ninety eight less that remainder",
		Provider: "iban",
		Source:   "regulation 5/2002 of the President of the National Bank of Poland, repealed, Dz. Urz. NBP 2002 no. 8 item 26, annexes 1 and 2, which state the whole calculation in words and print a table of thirty weights they worked out",
		State:    ByWorkedExample,
		Detail:   "TestTheWeightTableThePolishActPublishesIsReproduced",
	},
	{
		Rule:     "that the same arithmetic holds for every country, with letter values running from ten for A to thirty five for Z",
		Provider: "iban",
		Source:   "no document we can use states this. The act names the values of P and L only, and the standards that state the general rule, ISO 7064 and ISO 13616, are sold rather than published. The register of national formats is published under terms that forbid onward licensing of the kind this project's licence requires",
		State:    ByMeasurement,
		Detail: "89 example account numbers from 89 countries, taken from the ISO 13616 register and " +
			"not kept, measured 2026-08-04. Searching three axes at once leaves exactly 1 triple: " +
			"modulus 97, rotation 4, letter offset 10. The modulus is forced beyond the search range " +
			"as well, because the greatest common divisor of every value less one is 97 itself, whose " +
			"only divisor above one is 97",
	},

	// --- vin ----------------------------------------------------------------
	//
	// The only scheme here whose source answers all three questions a source
	// can be asked. The shape is stated, the arithmetic is stated, and the
	// document works an example of it. Four official documents describe the
	// shape of a REGON and none of them states its arithmetic, which is why
	// that one rests on a measurement and this one does not.
	{
		Rule:     "the seventeen character shape, the set of characters a value may hold with I, O and Q left out, and the check digit sitting in position nine",
		Provider: "vin",
		Source:   "49 CFR 565.13, read at ecfr.gov on 2026-09-06, title 49 up to date as of 2026-09-03. A work of the federal government of the United States and therefore free of copyright under 17 U.S.C. 105",
		State:    ByQuotedSentence,
		Detail:   "the alphabet is held as the regulation brackets it in internal/oracle/vin.go, and the provider holds the same set as a plain string, so a character dropped from one is not dropped from the other",
	},
	{
		Rule:     "the transliteration of letters to values, the weight of each position, the divisor of eleven, and the letter X for the residue that does not fit in a digit",
		Provider: "vin",
		Source:   "49 CFR 565.15(c) with Tables III, IV and V, and the calculation worked in Table VI, read at ecfr.gov on 2026-09-06",
		State:    ByWorkedExample,
		Detail:   "TestTheCalculationWorkedByTheRegulationComesOutTheSameHere",
	},

	// --- ean ----------------------------------------------------------------
	//
	// The only scheme here anchored on two first party documents that were
	// written independently of each other. GS1 publishes the standard itself,
	// and the International ISBN Agency states the same arithmetic for the
	// prefixes 978 and 979, which are GTIN-13s carrying books. Each works its
	// own example, so the two can be held against one another rather than
	// merely believed in turn.
	//
	// The third row is the rare one. Every other scheme in this register either
	// has no safe harbour or, in the single case of the social security number,
	// has one named by the body that issues the numbers. This is the second.
	{
		Rule:     "the thirteen digit shape, numeric throughout, with the check digit in the last position",
		Provider: "ean",
		Source:   "GS1 General Specifications, Release 26.0 ratified January 2026, Table 7-8, downloaded from ref.gs1.org on 2026-09-06",
		State:    ByQuotedSentence,
		Detail:   "the published weight row and the field lengths are held as the standard prints them in internal/oracle/ean.go, while the provider holds the alternation as a rule, so a weight misread in one is not misread the same way in the other",
	},
	{
		Rule:     "the two multipliers anchored at the right hand end of the body, the divisor of ten, and the computed ten written as nought",
		Provider: "ean",
		Source:   "GS1 General Specifications Release 26.0 Table 7-8 with the calculation worked in Table 7-9, and the ISBN Users' Manual of the International ISBN Agency, sixth edition 2012, Appendix A1.1, which works a second example of the same arithmetic",
		State:    ByWorkedExample,
		Detail:   "TestTheExamplesTheStandardsWorkComeOutTheSameHere",
	},
	{
		Rule:     "the prefix 952 is set aside for demonstrations and examples, so nothing generated there can belong to a real article",
		Provider: "ean",
		Source:   "GS1 General Specifications Release 26.0, Table 1-4 and Table 1-5, downloaded from ref.gs1.org on 2026-09-06. The same carve out is present in releases 23.0.0 and 25.0.0 and absent from 21.0.1",
		State:    ByQuotedSentence,
		Detail:   "the row is held as the standard writes it in internal/oracle/ean.go. What it does not establish is whether a company prefix was issued in 952 before the carve out, and docs/05 carries that limit in the same words rather than leaving it to this field",
	},

	// --- imei ---------------------------------------------------------------
	//
	// One document, and it is worth saying why it is not counted as two. ETSI
	// republishes this specification and its republication carries the annex
	// word for word, so reading both is one text read twice. The article
	// number above really does stand on two independently written documents
	// and this one does not.
	//
	// There is no third row here. This scheme has no safe harbour, and the
	// reason is set out where it can be acted on, in the provider and in
	// docs/05, rather than as an anchor for a rule that does not exist.
	{
		Rule:     "the fifteen digit shape, an eight digit allocation code and a six digit serial number, with the check digit last and numeric throughout",
		Provider: "imei",
		Source:   "ETSI TS 123 003 version 19.7.0 clause 6.2.1, the transposition of 3GPP TS 23.003 published by ETSI, downloaded on 2026-09-07. The newer 3GPP text of version 20.0.0 was read as well and says the same, and it is not the one cited because it states that it shall not be implemented and points at the publications offices, of which ETSI is one",
		State:    ByQuotedSentence,
		Detail:   "the element lengths are held as the specification writes them in internal/oracle/imei.go and are read back out of that sentence, so the fourteen the check digit covers is taken from the source rather than typed as a number in two places",
	},
	{
		Rule:     "the digits labelled D14 down to D1, the odd labelled ones doubled, the individual digits of those results added, and the total taken up to the next number ending in nought",
		Provider: "imei",
		Source:   "ETSI TS 123 003 version 19.7.0 annex B, which is normative and works its own example in B.3, printing the seven products of step 1 and the total 53 of step 2 before arriving at the check digit 7. The 3GPP text of version 20.0.0 carries the annex word for word",
		State:    ByWorkedExample,
		Detail:   "TestTheExampleTheSpecificationWorksComesOutTheSameHere",
	},

	// --- PL.regon ----------------------------------------------------------
	{
		Rule:     "the nine digit shape and the ban on any digit carrying meaning",
		Provider: "PL.regon",
		Source:   "Council of Ministers regulation of 30 November 2015, and the repealed one of 27 July 1999 saying the same thing sixteen years apart",
		State:    ByQuotedSentence,
		Detail:   "the wording is held in internal/oracle/regon.go, weights labelled by position",
	},
	{
		Rule:     "the check digit weights and a remainder of ten being written as nought",
		Provider: "PL.regon",
		Source:   "no official document states the arithmetic. Four were read and none carries it",
		State:    ByMeasurement,
		Detail: "174 real numbers from two unrelated public registers reproduced the check digit, " +
			"174 of 174, fifteen of them landing on a remainder of ten, measured 2026-08-02. " +
			"The numbers were not kept. Uniqueness of the weights was not established, so a " +
			"different vector closing the same sample has not been ruled out",
	},
}

// CountByAnchor counts the register by state.
//
// It takes the register rather than reading the package variable so that a test
// can hand it one that breaks a rule and check that the answer changes.
func CountByAnchor(anchors []Anchor) map[AnchorState]int {
	counts := make(map[AnchorState]int)
	for _, anchor := range anchors {
		counts[anchor.State]++
	}
	return counts
}

// ProvidersWithoutAnchor returns the provider identifiers that no entry covers.
//
// This is the direction that matters most. A provider nobody anchored is a set
// of rules resting on one reading, which is the failure this register exists to
// make visible.
func ProvidersWithoutAnchor(ids []string, anchors []Anchor) []string {
	covered := make(map[string]bool, len(anchors))
	for _, anchor := range anchors {
		covered[anchor.Provider] = true
	}

	var missing []string
	for _, id := range ids {
		if !covered[id] {
			missing = append(missing, id)
		}
	}
	return missing
}

// AnchorsNamingNoProvider returns the entries pointing at a provider that is not
// in the tree.
//
// A register naming something that is gone promises coverage nothing keeps, and
// a rename is the quietest way to get there.
func AnchorsNamingNoProvider(ids []string, anchors []Anchor) []string {
	exists := make(map[string]bool, len(ids))
	for _, id := range ids {
		exists[id] = true
	}

	var dangling []string
	for _, anchor := range anchors {
		if !exists[anchor.Provider] {
			dangling = append(dangling, anchor.Provider+": "+anchor.Rule)
		}
	}
	return dangling
}

// IncompleteAnchors returns the entries whose detail does not carry what their
// state promises.
//
// An entry saying a rule was measured, without saying what was measured or when,
// asks to be believed rather than repeated. That is the shape this register was
// built to refuse, so it is checked rather than trusted.
func IncompleteAnchors(anchors []Anchor) []string {
	var faults []string
	for _, anchor := range anchors {
		if anchor.Rule == "" || anchor.Provider == "" || anchor.Source == "" || anchor.Detail == "" {
			faults = append(faults, anchor.Provider+": an entry left a field empty")
			continue
		}
		if anchor.State == ByMeasurement && !detailIsRepeatable(anchor.Detail) {
			faults = append(faults, anchor.Provider+": "+anchor.Rule+
				" was measured and the detail does not say how many values or when")
		}
		if anchor.State == ByConstruction && !sourceSaysWhatItDoesNotAnswer(anchor.Source) {
			faults = append(faults, anchor.Provider+": "+anchor.Rule+
				" is called ours by construction and the source field does not say what was consulted and left unanswered")
		}
	}
	return faults
}

// unanswered are the ways of writing that a source was asked and did not
// answer.
//
// # This check is weak on purpose and saying so is part of it
//
// It cannot judge whether a sentence is true, only whether one was written. A
// rule taken from a source and mislabelled as ours would pass it by adding a
// denial that is simply false. What it does stop is the cheaper failure, which
// is the one that actually happens: reaching for this state because nothing was
// found yet, and leaving the field saying nothing at all. That belongs in
// Unanchored, which carries a ratchet precisely so it cannot be settled for.
var unanswered = []string{
	"does not", "do not", "did not", "no document", "no source", "is not one", "nothing",
}

// sourceSaysWhatItDoesNotAnswer reports whether the source field of a rule made
// here names something that was consulted and states that it left the question
// open.
func sourceSaysWhatItDoesNotAnswer(source string) bool {
	lowered := strings.ToLower(source)
	for _, phrase := range unanswered {
		if strings.Contains(lowered, phrase) {
			return true
		}
	}
	return false
}

// measuredOn matches the date a measurement was taken.
//
// The shape is fixed rather than lenient on purpose. A date somebody has to
// interpret is a date two sessions will read two ways.
var measuredOn = regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`)

// countedValues matches a number that is not part of the date.
//
// Checking for any run of digits would be satisfied by the date alone, which
// would make half of detailIsRepeatable dead while looking like it worked.
var countedValues = regexp.MustCompile(`\b\d+\b`)

// detailIsRepeatable says whether a measurement detail carries both a count of
// values and the date it was taken, which is the least somebody needs to run it
// again rather than believe it.
func detailIsRepeatable(detail string) bool {
	date := measuredOn.FindString(detail)
	if date == "" {
		return false
	}
	for _, number := range countedValues.FindAllString(detail, -1) {
		if !strings.Contains(date, number) {
			return true
		}
	}
	return false
}
