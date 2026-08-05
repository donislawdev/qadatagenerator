package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
)

// knownLayers is every layer and cross cutting module an entry may declare.
//
// SEC is here although the starter set carries no entry in it. The module
// exists in the layer model and its entries are simply outside the first
// release, so a model that refused the name would refuse a legitimate
// contribution rather than a defect.
var knownLayers = map[Layer]bool{
	Bytes: true, Container: true, FieldShape: true, Domain: true,
	Business: true, SetLevel: true,
	Security: true, Unicode: true, Reverse: true,
}

var knownExpectations = map[Expectation]bool{
	Rejection: true, Acceptance: true, Warning: true,
}

var knownConfidences = map[Confidence]bool{Hard: true, Soft: true}

// New builds a catalog out of entries and reports everything wrong with them.
//
// It returns every problem rather than the first one. Whoever is fixing a
// catalog wants the list, and handing them out one at a time turns one round of
// work into several. This is the same habit the recipe reader has, for the same
// reason.
//
// A catalog with any problem at all is not returned. There is no partial
// catalog, because an entry silently missing from one is a mutator that a
// policy can order and that the manifest will describe neither way.
func New(version, status string, entries []Entry) (Catalog, []string) {
	problems := headerProblems(version, status)
	problems = append(problems, entryProblems(entries)...)
	if len(problems) > 0 {
		sort.Strings(problems)
		return Catalog{}, problems
	}

	built := Catalog{
		version: version,
		status:  status,
		entries: make(map[ID]Entry, len(entries)),
		ordered: make([]ID, 0, len(entries)),
	}
	for _, entry := range entries {
		built.entries[entry.ID] = entry
		built.ordered = append(built.ordered, entry.ID)
	}
	sortIDs(built.ordered)
	return built, nil
}

func headerProblems(version, status string) []string {
	var problems []string
	if strings.TrimSpace(version) == "" {
		problems = append(problems, "the catalog carries no version, and the manifest header quotes it")
	}
	if strings.TrimSpace(status) == "" {
		problems = append(problems, "the catalog carries no status")
	}
	return problems
}

// entryProblems collects what is wrong across all entries, including the one
// thing no single entry can see: two entries sharing an identifier.
func entryProblems(entries []Entry) []string {
	var problems []string
	if len(entries) == 0 {
		return []string{"the catalog carries no entries, so this build knows no damage it can do"}
	}
	seen := make(map[ID]bool, len(entries))
	for _, entry := range entries {
		if seen[entry.ID] {
			problems = append(problems, fmt.Sprintf("%s: two entries carry this identifier", entry.ID))
		}
		seen[entry.ID] = true
		problems = append(problems, entry.problems()...)
	}
	return problems
}

// problems reports everything wrong with one entry.
func (e Entry) problems() []string {
	var problems []string
	if strings.TrimSpace(string(e.ID)) == "" {
		return []string{"an entry carries no identifier, so nothing can refer to it"}
	}
	problems = append(problems, e.identityProblems()...)
	problems = append(problems, e.verdictProblems()...)
	problems = append(problems, e.shapeProblems()...)
	problems = append(problems, e.exampleProblems()...)
	return problems
}

// identityProblems covers the identifier, the layer and the agreement between
// them.
//
// The agreement is checked because the layer is the first thing a reader takes
// from an identifier. An entry called L3.something sitting in layer L4 would
// read one way in a manifest and count another way in a report, and both would
// look right on their own.
func (e Entry) identityProblems() []string {
	var problems []string
	if !knownLayers[e.Layer] {
		problems = append(problems, fmt.Sprintf("%s: layer %q is not one this build knows", e.ID, e.Layer))
		return problems
	}
	if !strings.HasPrefix(string(e.ID), string(e.Layer)+".") {
		problems = append(problems, fmt.Sprintf(
			"%s: the identifier does not begin with its own layer %q", e.ID, e.Layer))
	}
	if strings.TrimSpace(e.Summary) == "" {
		problems = append(problems, fmt.Sprintf("%s: no summary, so no surface can say what it does", e.ID))
	}
	return problems
}

// verdictProblems covers what a test is supposed to conclude from the damage.
//
// An entry with no expectation does not exist. Damage whose expected outcome is
// unknown is useless in an automated test, which is the whole problem this
// format solves.
func (e Entry) verdictProblems() []string {
	var problems []string
	if !knownExpectations[e.Expectation] {
		problems = append(problems, fmt.Sprintf(
			"%s: expectation %q is not one of rejection, acceptance or warning", e.ID, e.Expectation))
	}
	if !knownConfidences[e.Confidence] {
		problems = append(problems, fmt.Sprintf(
			"%s: confidence %q is neither hard nor soft", e.ID, e.Confidence))
		return problems
	}
	if e.Confidence == Soft && strings.TrimSpace(e.Example.Note) == "" &&
		strings.TrimSpace(e.ApplicabilityCondition) == "" {
		problems = append(problems, fmt.Sprintf(
			"%s: soft confidence with nothing saying what the reaction depends on", e.ID))
	}
	return problems
}

// shapeProblems covers where the mutator applies, what it can be tuned with and
// what the run has to be able to do before it works.
func (e Entry) shapeProblems() []string {
	var problems []string
	problems = append(problems, e.appliesToProblems()...)
	for _, parameter := range e.Parameters {
		if strings.TrimSpace(parameter.Name) == "" {
			problems = append(problems, fmt.Sprintf("%s: a parameter carries no name", e.ID))
			continue
		}
		if strings.TrimSpace(parameter.Description) == "" {
			problems = append(problems, fmt.Sprintf(
				"%s: parameter %q carries no description", e.ID, parameter.Name))
		}
	}
	for _, required := range e.Requires {
		if !capability.Known(required) {
			problems = append(problems, fmt.Sprintf(
				"%s: requires %q, which no supplier in this build brings", e.ID, required))
		}
	}
	return problems
}

// appliesToProblems checks the three dimensions.
//
// An empty dimension is refused rather than read as "everywhere". The catalog
// has two reserved words for exactly this, any and none, and the difference
// between them is the difference between a mutator that damages a field and one
// that damages the file. Guessing which was meant would decide that silently.
func (e Entry) appliesToProblems() []string {
	dimensions := []struct {
		name   string
		values []string
	}{
		{"field_types", e.AppliesTo.FieldTypes},
		{"formats", e.AppliesTo.Formats},
		{"providers", e.AppliesTo.Providers},
	}
	var problems []string
	for _, dimension := range dimensions {
		if len(dimension.values) == 0 {
			problems = append(problems, fmt.Sprintf(
				"%s: applies_to.%s is empty, and any and none mean different things", e.ID, dimension.name))
		}
	}
	return problems
}

// exampleProblems checks the pair that becomes a test case.
//
// Only the before value has to carry something. A mutator takes a correct value
// and damages it, so an example starting from nothing has nothing to show, but
// an empty result is a legitimate outcome and one entry exists to produce it:
// L2.string.empty replaces a text value with an empty string, and requiring an
// after value here would refuse the entry for doing its job. Whether the key
// was written at all is a question about the file, which is the reader's to
// ask and not the model's.
func (e Entry) exampleProblems() []string {
	if strings.TrimSpace(e.Example.Before) == "" {
		return []string{fmt.Sprintf("%s: the example has no before value to damage", e.ID)}
	}
	return nil
}
