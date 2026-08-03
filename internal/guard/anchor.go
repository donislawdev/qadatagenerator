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
	// Unanchored means nothing outside our own reading supports this rule.
	// Said out loud so that it does not look like the three above. This list
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
		Source:   "country sheet for Poland on the European TIN Portal, worked example 1234563218",
		State:    ByWorkedExample,
		Detail:   "TestTheWorkedExampleFromTheSourceIsReproduced",
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
	}
	return faults
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
