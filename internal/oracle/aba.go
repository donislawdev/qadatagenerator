package oracle

import "strings"

// The ABA routing number, checked against the registrar's own sentences for
// its shape and against a measured congruence for its check digit.
//
// # Why this file is split in two halves and says so
//
// The two halves of this identifier do not have the same kind of source, and
// pretending otherwise would be the defect this whole package exists to catch.
//
// The shape is a quotation. The registrar publishes the structure and the
// allocation of the leading series, and two short sentences are held below
// verbatim, attributed, for the purpose of checking a value against them. The
// table of series beside them is the facts of that allocation written in our
// own words, which is a deliberate difference and is explained where it sits.
//
// The arithmetic is not. The registrar sends the reader to a paid publication
// for it, so the weights rest on a measurement over real numbers published in
// federal law rather than on anything anybody wrote down as a rule. That is a
// weaker kind of source and it is named as such here, in the register of
// anchors, and in the provider.
//
// The provider computes the check digit and this checks the whole number
// against the congruence. Two different statements of one rule, which is what
// makes their meeting worth anything.
const (
	// abaStructureSentence is the shape, from ABA Routing Number Policy &
	// Procedures version 9/20, section IV, published by LexisNexis Risk
	// Solutions as the official routing number registrar.
	abaStructureSentence = "The routing number consists of nine digits: a four digit " +
		"Federal Reserve Routing Symbol, a four digit ABA Institution Identifier, " +
		"and a check digit."

	// abaBoardNote is the sentence that decides the ethics of this provider,
	// and it is the reason nothing is generated inside a spare series.
	//
	// A series marked reserved is not a harbour. It is stock the registrar has
	// not handed out yet and says out loud it may hand out. Generating there
	// would be picking a range because it looks free, which is the mistake this
	// project already made once with a taxpayer identifier.
	abaBoardNote = "NOTE: The use of numbers in series designated \"Reserved for future use\" " +
		"will be determined by the Board."

	// abaReservedPhrase is the wording the policy uses for a spare series. It
	// is pulled out so that the table below is matched against the source's own
	// phrase rather than against a paraphrase.
	abaReservedPhrase = "Reserved for future use"
)

// abaAllocation is which leading two digit series is used for what.
//
// It is data rather than a set of conditions because the question asked of it
// changes: pillar one needs the series that are issued, a corruption needs the
// series that are not, and a report needs to name what a series is for.
//
// # The numbers are read from the source and the words are ours
//
// Which range covers what is a fact, and a fact belongs to nobody. The
// descriptions are written here rather than copied, because the registrar's
// policy is published by a commercial publisher and reproducing the wording and
// arrangement of somebody's table is a different act from quoting a sentence to
// analyse it. This repository is going to be public, so the distinction is worth
// the two minutes it costs.
//
// One phrase stays verbatim and only one. The wording marking a series as spare
// carries the whole ethical argument of this provider, which is that the
// registrar reserves the right to hand those numbers out, so paraphrasing it
// would be paraphrasing the evidence. Four words, attributed, for the purpose of
// analysing them. That is a quotation in anybody's law.
// # Meaning is a value, description is prose, and they are two columns
//
// The kind is what every question here is answered from. The description is for
// a human reading a report and nothing branches on it.
//
// They were one column until 2026-08-04, when the descriptions were reworded and
// two predicates changed their answers, because they were matching fragments of
// the prose. That is the same defect this project keeps meeting: a machine
// reading text that was written for a person. Rewording a sentence must not be
// able to move a number from the issued series into the spare ones.
type abaSeriesKind int

const (
	// abaSeriesOther is any use that is neither the base for generated values
	// nor a spare block.
	abaSeriesOther abaSeriesKind = iota
	// abaSeriesIssuedToInstitutions is where correct values are generated,
	// because this scheme offers nothing safer.
	abaSeriesIssuedToInstitutions
	// abaSeriesReserved is a block the registrar has not handed out and says it
	// may still hand out. Never a place to generate, always a place to corrupt
	// into.
	abaSeriesReserved
)

var abaAllocation = []struct {
	low, high int
	kind      abaSeriesKind
	use       string
}{
	{0, 0, abaSeriesOther, "federal government payments"},
	{1, 12, abaSeriesIssuedToInstitutions, "banks, reserve banks and state entities, for cheques and electronic transactions alike"},
	{13, 20, abaSeriesReserved, abaReservedPhrase},
	{21, 32, abaSeriesOther, "savings institutions, closed to new assignment since 1985"},
	{33, 39, abaSeriesReserved, abaReservedPhrase},
	{40, 49, abaSeriesReserved, abaReservedPhrase},
	{50, 59, abaSeriesOther, "in use inside banks and reserve banks for their own processing, so not spare"},
	{60, 60, abaSeriesReserved, abaReservedPhrase},
	{61, 72, abaSeriesOther, "identifiers for electronic transactions"},
	{73, 79, abaSeriesReserved, abaReservedPhrase},
	{80, 80, abaSeriesOther, "travellers cheques"},
	{81, 92, abaSeriesReserved, abaReservedPhrase},
	{93, 99, abaSeriesReserved, abaReservedPhrase},
}

// abaWeights is a measurement, not a quotation, and the distinction is the
// point of this comment.
//
// Measured against 12 CFR part 229 appendix A, which prints complete routing
// numbers of the Federal Reserve Banks and the Federal Home Loan Banks: 67 of
// 67 close under these weights. A second sample from unrelated federal
// publishers closed as well. Walking all one thousand weight vectors over the
// first sample leaves exactly ten that close it, the zero vector and the nine
// multiples of this one modulo ten, which are the same congruence. No other
// solution exists in the space, so the measurement does not merely agree with
// these weights, it forces them.
//
// What it does not show is that the registrar writes the rule this way in the
// publication it sells. The numbers are real and were promulgated in federal
// law, so the rule they satisfy is the rule. The sentence "the weights come
// from the source" would still be untrue and must not be written.
var abaWeights = [9]int{3, 7, 1, 3, 7, 1, 3, 7, 1}

// ABAFinding is one reason a value is not a well formed routing number. An
// empty list means it is well formed.
type ABAFinding string

// CheckABA reports every rule a value breaks.
//
// It states the check digit rule as a property of the whole number, that the
// weighted total is divisible by ten, while the provider states it as an
// arithmetic producing the last digit. Two statements of one rule is the only
// thing a second implementation by the same hand can still establish.
func CheckABA(value string) []ABAFinding {
	digits, ok := abaDigits(value)
	if !ok {
		return []ABAFinding{"the value is not nine digits, written 123456789 or 1234-5678-9"}
	}

	var findings []ABAFinding
	total := 0
	for position, symbol := range digits {
		total += int(symbol-'0') * abaWeights[position]
	}
	if total%10 != 0 {
		findings = append(findings, "the weighted total is not divisible by ten, so the check digit does not match")
	}
	return findings
}

// ABASeriesUse returns what the registrar says the leading series of a value is
// for, and whether the value could be read at all.
func ABASeriesUse(value string) (string, bool) {
	entry, ok := abaSeriesOf(value)
	if !ok {
		return "", false
	}
	return entry.use, true
}

// abaSeriesOf finds the row covering the leading series of a value.
//
// Every question about a series goes through here, so a value whose series no
// row covers is answered as unknown in one place rather than defaulting to
// something plausible in several.
func abaSeriesOf(value string) (struct {
	low, high int
	kind      abaSeriesKind
	use       string
}, bool) {
	digits, ok := abaDigits(value)
	if !ok {
		return abaAllocation[0], false
	}
	series := int(digits[0]-'0')*10 + int(digits[1]-'0')
	for _, entry := range abaAllocation {
		if series >= entry.low && series <= entry.high {
			return entry, true
		}
	}
	return abaAllocation[0], false
}

// ABAInReservedSeries reports whether a value sits in a series the registrar
// has marked as spare.
func ABAInReservedSeries(value string) bool {
	entry, ok := abaSeriesOf(value)
	return ok && entry.kind == abaSeriesReserved
}

// ABAIssuedToInstitutions reports whether a value sits in the series carrying
// the transactions of banks, which is where correct generated data has to sit
// because this scheme has no harbour.
func ABAIssuedToInstitutions(value string) bool {
	entry, ok := abaSeriesOf(value)
	return ok && entry.kind == abaSeriesIssuedToInstitutions
}

// ABAStructureSaysNineDigits pulls the length out of the quoted sentence rather
// than restating it, so a test comparing against nine is comparing against the
// source and not against a number somebody typed twice.
func ABAStructureSaysNineDigits() bool {
	return strings.Contains(abaStructureSentence, "nine digits")
}

// ABASpareSeriesAreNotAHarbour reports whether the source itself says the spare
// series may still be handed out.
//
// This exists so that the ethical rule of the provider is checked against a
// sentence rather than against a habit. If this ever returns false, the reason
// for generating inside an issued series has gone, and so has the obligation to
// mark the dataset synthetic.
func ABASpareSeriesAreNotAHarbour() bool {
	return strings.Contains(abaBoardNote, "will be determined by the Board")
}

// abaDigits returns the nine digits, accepting the bare form and the four four
// one grouping that federal law prints them in.
//
// Both are the same number. Refusing the grouped form would report a defect in
// a value copied out of the regulation that publishes it.
func abaDigits(value string) (string, bool) {
	stripped := value
	if len(value) == 11 && value[4] == '-' && value[9] == '-' {
		stripped = value[0:4] + value[5:9] + value[10:11]
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
