// Package response is what the application under test did with a dataset.
//
// # The one file this project reads rather than writes
//
// Everything else here is ours: we write it, and a model refuses a shape that
// would lie. This one arrives from outside, produced by somebody's import loop
// or by a query run after their test. So the rules here are not about keeping
// ourselves honest, they are about refusing an input that cannot support the
// comparison it is offered for, and saying which part of it was missing.
//
// # Why the unit is a row
//
// A row can carry several defects. An application does not react to a defect,
// it reacts to a record, and asking which of three defects caused a rejection
// is a question nobody can answer, including the application. Recorded as
// DEC-166 with what was rejected.
package response

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Verdict is what the application did with one row.
//
// Two values and no third. "Did not react" is not a verdict, because an
// application that never saw a row has nothing to say about it: that case is
// carried by the row being absent, which is what a receiver silently dropping
// everything past a limit actually produces.
type Verdict string

const (
	Accepted Verdict = "accepted"
	Rejected Verdict = "rejected"
)

// Row is what happened to one record.
type Row struct {
	// Entity is the table the record belongs to, as the producer wrote it, and
	// empty when they did not say. Empty means the only entity of the dataset,
	// and whether the dataset has only one is a question for the manifest,
	// which this model does not hold.
	Entity string
	// Row is the record, counting the first one after the header as one.
	Number  int
	Verdict Verdict
	// Code and Message are the application's own, kept as it wrote them.
	Code    string
	Message string
	// Stored is what the application holds now, by column, and HasStored says
	// whether the producer answered that question at all.
	//
	// 🔴 The pair, rather than an empty map meaning no, because the two are
	// different answers and the difference is the whole reason this member
	// exists. A producer that read nothing back has not told us the row is
	// unchanged, and treating silence as agreement is what this file is for.
	Stored    map[string]string
	HasStored bool
	// Warnings and HasWarnings are the same pair for anything the application
	// said without refusing the row.
	Warnings    []string
	HasWarnings bool
}

// Record is one run of a dataset through an application.
type Record struct {
	SpecVersion string
	// RowsTotal is what the producer believes it sent, and HasRowsTotal says
	// whether it claimed anything. It is a cross check rather than a source:
	// the manifest knows how many rows there were.
	RowsTotal    int
	HasRowsTotal bool
	Rows         []Row
}

// Problems is everything wrong with a record, all of it at once.
//
// All of it at once rather than the first, because whoever produced this file
// is going to run the tool again after fixing what it named, and a reader that
// stops at the first fault turns one mistake into as many runs as there are
// mistakes.
func (r Record) Problems() []string {
	var problems []string
	if strings.TrimSpace(r.SpecVersion) == "" {
		problems = append(problems, "the record does not say which version of the format it is "+
			"written to, so nothing can tell it from a later one")
	}
	if len(r.Rows) == 0 {
		problems = append(problems, "the record names no row at all. A run where the application "+
			"reacted to nothing is a run where every row went missing, and saying that needs the "+
			"rows it did react to")
	}
	seen := map[Place]bool{}
	hinted := false
	for _, row := range r.Rows {
		problems = append(problems, row.problems(seen, &hinted)...)
	}
	sort.Strings(problems)
	return problems
}

// Place is a record as the producer named it: an entity, or none, and a row.
type Place struct {
	Entity string
	Number int
}

// Place is where the producer said this row is.
func (row Row) Place() Place { return Place{Entity: row.Entity, Number: row.Number} }

// Name is the row as a reader finds it in a sentence: its number, and the
// entity when the producer named one.
func (p Place) Name() string {
	if p.Entity == "" {
		return "row " + strconv.Itoa(p.Number)
	}
	return "row " + strconv.Itoa(p.Number) + " of " + p.Entity
}

// problems is everything wrong with one row. hinted carries across the rows
// whether the sentence about naming entities was said already.
func (row Row) problems(seen map[Place]bool, hinted *bool) []string {
	var problems []string
	switch {
	case row.Number < 1:
		problems = append(problems, "a row is numbered "+strconv.Itoa(row.Number)+
			", and rows count from one because the header is not a record")
	case seen[row.Place()]:
		// 🔴 Two verdicts about one row is the one fault a reader cannot work
		// around by choosing, because choosing is what would make the answer
		// depend on the order of somebody else's file.
		//
		// The same number under two entities is two rows, which is what a
		// dataset of two tables has. A row named once with its entity and once
		// without it is a question for the manifest, which knows whether the
		// dataset has one entity or several, so it is asked where the two are
		// put side by side.
		said := fmt.Sprintf("%s is answered twice, and the two answers cannot both be what happened",
			row.Place().Name())
		// Once per record, beside the first such row. A number repeated without
		// an entity is most likely a producer of two tables numbering each from
		// one, which is right, and naming neither, which is not. Measured on
		// 2026-10-27: that record met this sentence before anything could say
		// the entity was missing, because the manifest that knows there are two
		// tables is not read here.
		if row.Entity == "" && !*hinted {
			said += ". Rows of different entities each name their entity"
			*hinted = true
		}
		problems = append(problems, said)
	default:
		seen[row.Place()] = true
	}
	if row.Verdict != Accepted && row.Verdict != Rejected {
		problems = append(problems, fmt.Sprintf(
			"%s has the verdict %q, and the two a row can have are %q and %q",
			row.Place().Name(), row.Verdict, Accepted, Rejected))
	}
	return problems
}

// Marked reports whether the application said anything about a row without
// refusing it, and whether the producer answered that question.
func (row Row) Marked() (marked bool, known bool) {
	if !row.HasWarnings {
		return false, false
	}
	return len(row.Warnings) > 0, true
}

// Changed reports whether a field holds something other than what was sent,
// and whether the producer answered that question.
//
// A field the producer did not mention is not a field that survived. It is a
// field nobody looked at, and the difference is the measured one: a database
// taking a row, turning text into zero and cutting a string to the column
// width reports no error at all.
func (row Row) Changed(field, sent string) (changed bool, known bool) {
	if !row.HasStored {
		return false, false
	}
	held, mentioned := row.Stored[field]
	if !mentioned {
		return false, false
	}
	return held != sent, true
}
