// Package manifest is the model of what a corruption run promises about the
// data it produced.
//
// The manifest is the product rather than a description of the product. A
// dataset with damage in it and no manifest is worth less than no dataset at
// all, because a test written against it asserts whatever the author guessed.
//
// # Silence is the thing this package exists to prevent
//
// Every damage the policy ordered gets exactly one entry: applied, with the
// values, or not applied, with the reason. A dataset where twenty seven of
// forty ordered defects were applied looks exactly like a complete one, and it
// will be trusted as a complete one. That is why the envelope carries both
// counts and why an entry that failed still has to say what it wanted to do.
//
// The rules here are the published specification made checkable. The
// specification says an entry that was not applied carries a reason, and a
// reader of prose can forget that. A build that refuses to record such an entry
// cannot.
//
// # What is deliberately not here
//
// Nothing about JSON, CSV, Markdown or Gherkin. The canonical form is JSON and
// the others are derived from it, and all of them are somebody else's job:
// this package answers what a manifest is, not how it is written down.
//
// Nothing about what a value should have become either. The three levels this
// model carries describe how an application should react to a record, and not
// what it should compute from it. That is a decision rather than an oversight,
// and reversing it needs a shape for recording the application's answer, which
// does not exist yet.
package manifest

import (
	"fmt"
	"sort"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

// Location is where a defect sits. Which members are filled depends on the
// layer, because the layers damage different things.
type Location struct {
	// Entity is the table, used from L2 upwards when the dataset has named
	// entities.
	Entity string
	// Row is one based, used from L1 upwards. Zero means no row.
	Row int
	// Field is the column, used from L2 upwards.
	Field string
	// Path addresses a place inside a nested structure, for the cases where
	// naming a field is not enough.
	Path string
	// ByteOffset is where the damage is in the bytes, used at L0.
	// HasByteOffset separates an offset of zero from no offset at all, which
	// matters because the first byte of a file is a real place.
	ByteOffset    int
	HasByteOffset bool
	// Related are the other places a defect spans, required at L5.
	//
	// Set integrity is never about one record. A duplicated primary key
	// involves two rows, and naming one of them is useless to whoever has to
	// go and look.
	Related []Location
}

// Empty reports whether this location names no place at all, which is a legal
// answer only for damage to a whole file.
func (l Location) Empty() bool {
	return l.Entity == "" && l.Row == 0 && l.Field == "" && l.Path == "" &&
		!l.HasByteOffset && len(l.Related) == 0
}

// Profile is what one specific application is expected to do about a defect.
//
// It does not come from this tool. A team owns the file it is read from,
// because whether a discount over one hundred percent is an error or a
// correction is a fact about their system and not about data.
type Profile struct {
	Code       string
	HTTPStatus int
	Field      string
}

// Defect is one damage in one place.
type Defect struct {
	// ID is stable within one manifest, so that a test can refer to a case.
	ID string
	// Location is where it is.
	Location Location
	// Mutator and Layer are copied from the catalog entry, so that the
	// manifest can be read on its own without the catalog beside it.
	Mutator catalog.ID
	Layer   catalog.Layer
	// Applied says whether the mutator actually ran.
	Applied bool
	// ValueBefore is the original, carried when the mutator is reversible.
	// ValueAfter is what took its place.
	//
	// Both are text and both have a presence flag, because an empty string is
	// a value a mutator produces on purpose. One entry of the catalog exists
	// to replace a field with nothing.
	ValueBefore    string
	HasValueBefore bool
	ValueAfter     string
	HasValueAfter  bool
	// Expectation and Confidence are what a test should conclude, copied
	// from the catalog entry.
	Expectation catalog.Expectation
	Confidence  catalog.Confidence
	// Rationale is one sentence in plain language, written to be pasted into
	// a report by somebody who was not here when it ran.
	Rationale string
	// FailureReason is required when Applied is false, and says why the
	// damage could not be done.
	FailureReason string
	// FailureKind is the same answer in a form a machine can act on, and is
	// required exactly when FailureReason is.
	//
	// # Why both, when one of them is already a sentence
	//
	// The sentence is written for a person reading a report and cannot be
	// anything else: it names columns, schemes and values, and it will be
	// reworded whenever a reword makes it clearer. Anything downstream that
	// had to tell "this build cannot do it yet" from "your recipe does not
	// carry what it needs" would therefore be matching on English prose that
	// is free to change under it. This project has already paid for that once,
	// where a validator recognised a series of routing numbers by searching
	// inside a description written for a human, and rewording the description
	// quietly moved the number into another series.
	//
	// So the two travel together and neither replaces the other. The kind is
	// closed, small and stable. The reason is open and says what the kind
	// cannot: which column, which capability, which value.
	FailureKind FailureKind
	// Profile is optional and comes from the team's own file.
	Profile *Profile
}

// FailureKind is the closed set of ways ordered damage can fail to happen.
//
// # Why these six and not three
//
// The exit codes of this tool are split by the repair they force, because a
// pipeline can only act on the answer as a whole. A manifest is read by
// something that can act on any part of it, and a reader can always collapse
// six answers into three while nobody can recover six from three. So the
// vocabulary follows what the engine genuinely distinguishes rather than what a
// caller is expected to do about it.
//
// The order below is the order the engine asks the questions in, which is also
// roughly the order from "our fault" to "nothing is wrong here".
type FailureKind string

const (
	// Undescribed means no catalog entry carries the identifier that was
	// ordered. It is the only kind that indicates a defect in this tool
	// rather than a property of the run: whatever ordered it should have
	// been refused earlier.
	Undescribed FailureKind = "undescribed"
	// Unimplemented means the catalog describes this damage and this build
	// carries no implementation of it. It is the ordinary state of a project
	// whose catalog grows faster than its engine, and it is nobody's fault.
	Unimplemented FailureKind = "unimplemented"
	// RunLacksCapability means the entry requires something this run cannot
	// do. It is the kind a person can usually act on, because what a run can
	// do follows from what the recipe declares.
	RunLacksCapability FailureKind = "run_lacks_capability"
	// FieldTypeNotCovered means the dataset has no column of a kind this
	// damage applies to.
	FieldTypeNotCovered FailureKind = "field_type_not_covered"
	// SchemeNotCovered means the column follows a numbering scheme this
	// damage is not meaningful for. Being an identifier is not enough: the
	// schemes disagree about what they carry.
	SchemeNotCovered FailureKind = "scheme_not_covered"
	// DeclinedByMutator means the damage itself refused, for a reason only it
	// could know. It is often the correct outcome rather than a shortfall: a
	// value already sitting in the block a corruption aims at is the example
	// the catalog itself gives.
	DeclinedByMutator FailureKind = "declined_by_mutator"
)

// failureKinds is every kind, which is what lets a reader be held to the closed
// set rather than to whatever a caller happened to write.
var failureKinds = map[FailureKind]bool{
	Undescribed: true, Unimplemented: true, RunLacksCapability: true,
	FieldTypeNotCovered: true, SchemeNotCovered: true, DeclinedByMutator: true,
}

// Known reports whether this is one of the kinds the format defines.
func (k FailureKind) Known() bool { return failureKinds[k] }

// FailureKinds is the closed set written out, sorted.
//
// # Why the whole set and not only the membership test above
//
// A surface that shows a person why ordered damage did not happen needs one
// phrase per kind, and a register of phrases can only be closed against a list.
// Asked one kind at a time, nothing can put the question "is any kind
// unanswered", so a seventh kind would arrive on a screen as a blank space
// instead of as a failing test. The ban on silence is the whole reason an
// unapplied entry carries a kind at all, and it would be undone at the last
// step by the one surface a person actually reads.
func FailureKinds() []FailureKind {
	all := make([]FailureKind, 0, len(failureKinds))
	for kind := range failureKinds {
		all = append(all, kind)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	return all
}

// Run is the envelope: what this run was, and how much of it happened.
type Run struct {
	// GeneratedAt is when the run happened, as text in the form the manifest
	// is written with. It is text rather than a clock reading because
	// whoever runs the tool owns the format and this model owns none of it.
	GeneratedAt string
	// Seed, Locale and CatalogVersion are what the dataset was made from.
	//
	// Locale looks redundant and is not. Without it nobody can reconstruct
	// why certain mutators are absent, because availability at L3 and L4
	// depends on which provider the locale selected.
	Seed           string
	Locale         string
	CatalogVersion string
	// Recipe, Policy and Tag are optional references back to what asked for
	// this run.
	Recipe string
	Policy string
	Tag    string
	// Source is the file this run took and damaged, empty when the data was
	// produced from a recipe rather than read.
	//
	// It is not the recipe under another name. A recipe says what the data was
	// meant to be, and this says whose data it was, which is the question
	// somebody holding two similar files an hour later actually has.
	Source string
	// Inference is what the run worked out about a dataset nobody described,
	// and what it worked it out from. It is absent for a run from a recipe,
	// where nothing was guessed.
	Inference *Inference
	// RowsTotal and RowsMutated describe the dataset.
	RowsTotal   int
	RowsMutated int
}

// ColumnsDescribed is how many columns of a file this build accounts for one by
// one.
//
// # Where the number comes from
//
// An entry costs about 203 bytes, measured on 2026-09-03 from the slope between
// two real runs: 50 000 columns produced 9.6 MB of manifest and 200 000
// produced 38.6 MB, so 29 MiB bought 150 000 entries. At this limit the section
// costs about 2 MB, a twentieth of what this build reads back, so the account
// of the columns can never on its own be why a manifest is refused. Defects
// still can, and that is a different question.
//
// # Why it is not the fifty the schema report uses
//
// That number bounds lines a person reads going past in a terminal, and this
// bounds entries a machine archives and a person searches. Carrying the fifty
// across because it stood next to this would have turned the limit into a cut
// on files people really have: a few thousand columns is a feature matrix or a
// run of sensor readings, and every column of one is described here.
//
// Above this width the file is machine made and nobody argues with it column by
// column, which is what the entries are for. That sentence is a judgement and is
// written down as one, because the arithmetic above does not reach it.
const ColumnsDescribed = 10000

// Inference is what a run decided about the shape of somebody else's data.
//
// # Why the evidence is part of the promise rather than a note beside it
//
// Everything else in a manifest is something this tool did. This is something
// it guessed, about data it did not produce, and the person who can tell
// whether the guess was right is the one reading it. A conclusion published
// without its evidence is a conclusion they cannot argue with, and damage aimed
// at a column wrongly taken for a national identifier would have them chasing a
// rejection that was never going to happen.
type Inference struct {
	// Method names how the shape was worked out, so that a later run using a
	// different way of looking is telling apart from this one.
	Method string
	// Columns is what was decided about each column, in column order.
	Columns []InferredColumn
	// Relations is what was proved to hold between two columns, empty when
	// nothing was.
	//
	// Separate from the columns rather than folded into them, because a
	// relation is a statement about a pair and belongs to neither half of it.
	// Damage that contradicts a neighbouring column rests on exactly these, so
	// a reader doubting such an entry reads the basis here.
	Relations []InferredRelation
	// NotInferred names the properties this way of looking never reads.
	//
	// It is required rather than optional, and the model refuses an inference
	// without it. A report saying only what it recognised invites the reader to
	// conclude that the rest was not there, and the rest is precisely where the
	// damage they asked for did not happen.
	NotInferred []string
	// ColumnsTotal is how many columns the file has, which is not always how
	// many Columns describes.
	//
	// # Why a count sits beside a list that used to be the whole of it
	//
	// An entry runs about two hundred bytes, so a file wide enough turns this
	// section into a manifest larger than this build reads back: measured on
	// 2026-09-03, 260 000 columns produced 50.2 MB and verify refused it, with
	// not one defect entry in it. Beyond a width no person reads column by
	// column, the account stops and this says how much of the file it covered.
	//
	// It is required rather than written only when something was left out. A
	// truncated account with nothing beside it reads as a complete account of a
	// narrower file, and there is no way to tell the two apart at the boundary.
	// That is the same argument NotInferred rests on, applied to the count
	// rather than to the kind of thing being reported.
	ColumnsTotal int
}

// InferredColumn is one column and the evidence behind what it was taken for.
type InferredColumn struct {
	// Field is the column.
	Field string
	// Type is what it was taken to hold.
	Type string
	// Scheme is the numbering scheme it was taken to follow, empty when none
	// was confirmed.
	Scheme string
	// Basis is the evidence, as a sentence a person reads.
	//
	// Required for every column, including the ones nothing was concluded
	// about. "Nothing recognised these values" is an answer, and a column with
	// no entry at all would leave the reader unable to tell a column that was
	// examined from one that was overlooked.
	Basis string
}

// InferredRelation is one agreement between two columns and the evidence for it.
type InferredRelation struct {
	// Field is the column carrying the numbering scheme and With is the column
	// it agrees with. The direction is the one a mutator works in: it rebuilds
	// the identifier and leaves it contradicting the neighbour.
	Field string
	With  string
	// Kind is what agrees, such as the date or the gender the scheme encodes.
	Kind string
	// Basis is the evidence, as a sentence a person reads. Required for the
	// same reason a column basis is: a relation stated without one is this
	// tool asking to be trusted about somebody else's data.
	Basis string
}

// Manifest is one run's promise about one dataset.
//
// The counts of ordered and applied damage are not members. They are derived
// from the entries, so that the summary cannot disagree with the list under
// it, which is the one disagreement nobody would check.
type Manifest struct {
	specVersion string
	run         Run
	defects     []Defect
}

// SpecVersion is the version of the manifest format this one is written in.
func (m Manifest) SpecVersion() string { return m.specVersion }

// Run returns the envelope.
func (m Manifest) Run() Run { return m.run }

// Defects returns the entries, in the order they were recorded.
//
// The order is the order damage was attempted, which is deterministic for one
// seed. It is a copy, so that a caller cannot reach in and change what the
// manifest says after it has said it.
func (m Manifest) Defects() []Defect {
	entries := make([]Defect, len(m.defects))
	copy(entries, m.defects)
	return entries
}

// Requested is how many damages the policy ordered, which is every entry.
func (m Manifest) Requested() int { return len(m.defects) }

// Applied is how many actually happened.
func (m Manifest) Applied() int {
	applied := 0
	for _, defect := range m.defects {
		if defect.Applied {
			applied++
		}
	}
	return applied
}

// Complete reports whether everything the policy ordered was done.
//
// A caller that wants to treat an incomplete run as a failure has to ask.
// Damage that could not be applied is sometimes the correct outcome rather
// than a fault, so this answers the question and does not decide it.
func (m Manifest) Complete() bool { return m.Applied() == m.Requested() }

// Unapplied returns the entries that did not happen, in recorded order.
func (m Manifest) Unapplied() []Defect {
	var missing []Defect
	for _, defect := range m.defects {
		if !defect.Applied {
			missing = append(missing, defect)
		}
	}
	return missing
}

// New builds a manifest and reports everything wrong with it.
//
// It returns every problem rather than the first, and it returns no manifest
// at all when there is one. A manifest that is partly wrong is worse than none,
// because it will be read as a promise.
func New(specVersion string, run Run, defects []Defect) (Manifest, []string) {
	problems := run.Problems()
	problems = append(problems, defectProblems(defects)...)
	if strings.TrimSpace(specVersion) == "" {
		problems = append(problems, "the manifest carries no format version, so nobody can tell how to read it")
	}
	if mutated := countRowsMutated(defects); run.RowsMutated < mutated {
		problems = append(problems, fmt.Sprintf(
			"the envelope says %d rows were mutated and the entries name %d",
			run.RowsMutated, mutated))
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return Manifest{}, problems
	}
	kept := make([]Defect, len(defects))
	copy(kept, defects)
	return Manifest{specVersion: specVersion, run: run, defects: kept}, nil
}

// countRowsMutated counts the distinct rows carrying applied damage, so that
// the envelope can be held to what the entries actually say.
func countRowsMutated(defects []Defect) int {
	type place struct {
		entity string
		row    int
	}
	seen := map[place]bool{}
	for _, defect := range defects {
		if defect.Applied && defect.Location.Row > 0 {
			seen[place{defect.Location.Entity, defect.Location.Row}] = true
		}
	}
	return len(seen)
}
