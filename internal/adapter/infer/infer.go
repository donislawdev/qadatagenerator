// Package infer works out the shape of a dataset nobody described.
//
// # The one rule everything here follows
//
// A column is only said to hold something when the values themselves prove it.
// The single mechanism that can prove anything about a value without being told
// is a check digit: a scheme that carries one refuses almost everything that is
// not of that scheme, so a column whose every value is accepted is a column of
// that scheme and the arithmetic says by how much.
//
// Everything else a recipe would declare is left unanswered on purpose, and the
// list of what those are travels with the manifest. The two temptations both
// end in the same place, which is why they are named here rather than argued
// again at each call site:
//
//   - the longest value seen is not a declared maximum length, and a column with
//     no blanks in it is not a required column. Those are properties of a
//     contract, and a file carries data rather than its contract.
//   - a column of digits is not a column of numbers. Invoice numbers, product
//     codes and telephone numbers live in text fields in other people's systems,
//     where writing "n/a" over one is accepted. Damage aimed at a number
//     publishes a hard expectation of rejection, and a hard expectation resting
//     on the observation that the values looked numeric is the kind of promise
//     this tool exists to argue against.
//
// So this answers two questions, identifier with its scheme, or text, and says
// out loud that it answered no others.
package infer

import (
	"sort"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// MinimumEvidence is how many values a column has to hold before a numbering
// scheme may be read off them.
//
// # Why eight, and it is a calculation rather than a round number
//
// The weakest check digit among the schemes this build carries is the one on a
// US routing number: nine digits, modulo ten, no other structure, so an
// unrelated value is accepted about one time in ten. Eight values accepted in a
// row therefore happen by chance about once in a hundred million. Every other
// scheme is stronger by orders of magnitude, because a national identifier also
// demands a fixed width and a real calendar date.
//
// A threshold that scaled with the size of the file was rejected: a share of a
// three row file is two values, and two values are an anecdote.
const MinimumEvidence = 8

// The two types this mode is willing to state. They match the vocabulary the
// catalog aims damage at, and they are constants so that a typo fails to
// compile rather than quietly making every entry inapplicable.
const (
	typeIdentifier = "identifier"
	typeString     = "string"
)

// Finding is what was decided about one column and what it was decided from.
//
// The basis is a sentence rather than a pair of numbers because its reader is a
// person deciding whether this tool guessed right about their data. A reader
// handed "tested 40, passed 40" has to already know how much forty is worth.
type Finding struct {
	Field  string
	Type   string
	Scheme string
	Basis  string
}

// Schema is a shape worked out by looking, and the evidence for it.
type Schema struct {
	entity   string
	fields   []recipe.FieldInfo
	findings []Finding
}

// Contract check, so that a mismatch fails to compile rather than at run time.
var _ recipe.SchemaView = Schema{}

// From works out what the columns of a table hold.
//
// The schemes given are every one this build carries. Which of them may be read
// off values is decided here rather than by the caller, because that is the
// whole content of the rule and a caller filtering the list first would be a
// second place for the rule to live.
func From(table dataset.Table, samples []dataset.Sample, schemes []provider.Provider) Schema {
	selfChecking := selfCheckingOnly(schemes)
	built := Schema{entity: table.Entity}
	byName := make(map[string]dataset.Sample, len(samples))
	for _, sample := range samples {
		byName[sample.Field] = sample
	}
	for _, column := range table.Columns {
		finding := look(byName[column], selfChecking)
		finding.Field = column
		built.findings = append(built.findings, finding)
		built.fields = append(built.fields, recipe.FieldInfo{
			Name: column, Type: finding.Type, Provider: finding.Scheme,
		})
	}
	return built
}

// look decides what one column holds.
func look(sample dataset.Sample, schemes []provider.Provider) Finding {
	tested := len(sample.Values)
	switch {
	case sample.NonEmpty == 0:
		return Finding{Type: typeString, Basis: "the column holds no value at all, so nothing " +
			"in it could confirm a numbering scheme"}
	case tested < MinimumEvidence:
		return Finding{Type: typeString, Basis: counted(tested, "value") + " to look at, and " +
			strconv.Itoa(MinimumEvidence) + " are needed before a check digit tells a numbering " +
			"scheme from a coincidence"}
	}

	accepted := acceptedBy(sample.Values, schemes)
	switch len(accepted) {
	case 0:
		return Finding{Type: typeString, Basis: "no numbering scheme this build carries accepted " +
			howMany(sample) + " of this column"}
	case 1:
		return Finding{Type: typeIdentifier, Scheme: accepted[0], Basis: "every one of " +
			howMany(sample) + " of this column was accepted by the " + accepted[0] +
			" validator, and by no other scheme carrying a check digit"}
	default:
		// 🔴 Two schemes accepting the whole sample is a refusal rather than a
		// choice. Picking the first would make the answer depend on the order of
		// a registry, and the manifest would then carry a scheme name with no
		// evidence behind it that the other was not equally good.
		return Finding{Type: typeString, Basis: howMany(sample) + " of this column were accepted " +
			"by " + strings.Join(accepted, " and by ") + " alike, so no scheme was chosen"}
	}
}

// acceptedBy returns the schemes that accept every value given, sorted.
//
// Every value has to be accepted, rather than most of them. A column of
// national identifiers carrying a few typos exists and would be missed by this,
// which is the cheaper of the two mistakes: the expensive one is a column of
// unrelated numbers accepted by a validator that only looks at shape, and no
// share threshold can tell those two apart.
func acceptedBy(values []string, schemes []provider.Provider) []string {
	var accepted []string
	for _, scheme := range schemes {
		if acceptsAll(scheme, values) {
			accepted = append(accepted, scheme.ID())
		}
	}
	sort.Strings(accepted)
	return accepted
}

func acceptsAll(scheme provider.Provider, values []string) bool {
	for _, value := range values {
		// The third verdict is not a pass. A provider that says it cannot judge
		// this value, as the international account number does outside the one
		// country whose structure it knows, has told us it has no opinion, and
		// treating no opinion as approval is the quiet half of DEC-98.
		if verdict := scheme.Validate(value); !verdict.Valid {
			return false
		}
	}
	return true
}

// selfCheckingOnly keeps the schemes whose validator can refuse on the strength
// of the value alone, sorted by identifier.
//
// A scheme without a check digit is left out however well it seems to fit, and
// the measurement behind that is worth keeping close to the code. The social
// security number validator refuses only a zero area, a zero group and a zero
// serial, because it has to accept the values this tool itself generates. Nine
// unrelated digits therefore pass it about ninety nine times in a hundred, so a
// column of telephone numbers would be announced as a column of social security
// numbers and every defect aimed at it would carry an expectation nobody can
// meet.
func selfCheckingOnly(schemes []provider.Provider) []provider.Provider {
	var kept []provider.Provider
	for _, scheme := range schemes {
		for _, brought := range scheme.Capabilities() {
			if brought == capability.IdentifierCheckDigit {
				kept = append(kept, scheme)
				break
			}
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].ID() < kept[j].ID() })
	return kept
}

// Findings is what was decided about each column, in column order.
func (s Schema) Findings() []Finding { return s.findings }

// NotInferred names what this mode never reads out of a file.
//
// # Why this travels with the answer instead of living in a document
//
// A report that says only what it recognised invites the reader to conclude
// that the rest was not there. This is the same rule that makes the audit script
// print what it does not check and every oracle print its own scope, and it is
// the difference between a report and an advertisement.
//
// The wording is what a person reads, so it names the properties rather than the
// keys of the recipe format. Whoever needs the keys has the recipe format.
func NotInferred() []string {
	return []string{
		"maximum length, because the longest value in a file is not a declared limit",
		"required columns, because a column with no blanks in it was not declared to need one",
		"uniqueness, because values that happen not to repeat were not declared unique",
		"the set of allowed values of an enumerated column, which a file never carries",
		"the primary key, and with it every defect about the identity of a record",
		"relations between columns, so damage promising to contradict a neighbouring column " +
			"has nothing to contradict",
		"relations between tables, because one file is one table",
		"numeric, date and postal column types, because a column of digits is not a column " +
			"of numbers",
	}
}

// EntityNames returns the one entity a read file holds.
func (s Schema) EntityNames() []string { return []string{s.entity} }

// FieldsOf returns what was worked out about the columns.
func (s Schema) FieldsOf(entity string) []recipe.FieldInfo {
	if entity != s.entity {
		return nil
	}
	return s.fields
}

// The rest of the view is empty, and emptily on purpose. They are written out
// one by one rather than left to a zero value, so that a method added to the
// interface fails to compile here instead of inheriting an answer nobody
// thought about. What each of them would have to claim, and why a file cannot
// support the claim, is in NotInferred.

// ParentsOf returns nothing: one file is one table, so no record here belongs
// to a record elsewhere.
func (s Schema) ParentsOf(string) []recipe.ParentInfo { return nil }

// RulesOf returns nothing: a rule binding two columns is a statement about what
// the data is supposed to satisfy, and a file carries what it holds rather than
// what it promised.
func (s Schema) RulesOf(string) []recipe.Rule { return nil }

// PrimaryKeyOf returns nothing: a column whose values happen not to repeat in
// this file was not declared to identify a record.
func (s Schema) PrimaryKeyOf(string) []string { return nil }

// howMany describes the part of a column that was looked at, saying so when it
// was only a part.
func howMany(sample dataset.Sample) string {
	tested := len(sample.Values)
	if tested < sample.NonEmpty {
		return "the first " + strconv.Itoa(tested) + " of " +
			counted(sample.NonEmpty, "value")
	}
	return counted(tested, "value")
}

// counted writes a number with its word, and the word agrees with the number.
func counted(number int, word string) string {
	if number == 1 {
		return "1 " + word
	}
	return strconv.Itoa(number) + " " + word + "s"
}
