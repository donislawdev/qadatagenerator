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
	// Profile is optional and comes from the team's own file.
	Profile *Profile
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
	// RowsTotal and RowsMutated describe the dataset.
	RowsTotal   int
	RowsMutated int
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
	problems := run.problems()
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
