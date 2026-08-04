// Package generate is the use case behind the first pillar: correct data,
// useful on its own.
//
// # Two phases, and the separation is what makes a dry run honest
//
// Planning answers what will be produced and touches no disk. Running answers
// what the bytes are. A dry run is the first phase without the second rather
// than a separate estimator, so it cannot drift from the real path. That is the
// cross cutting assumption of the work order rather than a step in it.
//
// # Why the ports are parameters
//
// A use case may not import an adapter. Everything this one needs from the
// world arrives as an interface: where values come from, where bytes go. That
// is the dependency rule applied to the one place it would be easiest to break,
// because a generator needs a word list and a word list is a file.
package generate

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Request is everything the source of values needs to produce one value.
type Request struct {
	// Generator is the name the recipe used, such as first_name.
	Generator string
	// At is the address of the value. Randomness is derived from it, so the
	// same address always gives the same value.
	At seed.Coordinate
	// Agrees carries the values this one has to be consistent with, keyed by
	// the role each plays. It is empty when the recipe declared no rule.
	Agrees provider.Agreement
	// MaxLength is the longest value the field accepts, and HasMaxLength
	// separates an absent limit from a limit of zero.
	//
	// It is passed in rather than checked afterwards because a word list can
	// honour it by choosing a shorter entry, while a check afterwards could
	// only fail. A scheme with a fixed width cannot honour it either way and
	// says so.
	MaxLength    int
	HasMaxLength bool
	// Unit is how MaxLength is counted. The three units disagree on exactly the
	// values this tool exists to produce.
	Unit string
}

// Values is where a generated value comes from.
//
// One port rather than one per kind of source, because the use case has no
// business knowing whether a value came from a word list, from an identifier
// provider or from arithmetic. What it does need to know is which names exist,
// so that an unknown one is refused by name instead of quietly producing
// nothing.
type Values interface {
	// Known reports whether this build has a generator of that name.
	Known(generator string) bool
	// Names returns every generator this build has, sorted, for the message
	// that refuses an unknown one.
	Names() []string
	// RoleOf returns the role a generator plays in a consistency rule, and
	// whether it plays one at all.
	//
	// The role comes from the generator rather than from the name of the
	// column, because a recipe may call a column anything and half of them
	// will not be in English.
	RoleOf(generator string) (string, bool)
	// Produce returns one value.
	Produce(req Request) (string, error)
}

// Record is one row on its way to a writer.
type Record struct {
	// Values are the fields in the order the recipe wrote them.
	Values []string
	// Parent is the row index of the parent record, or NoParent.
	//
	// It travels with the row rather than being looked up later, because the
	// nested form of the output needs it and a writer that had to ask for it
	// would need the whole dataset in memory to answer.
	Parent int
}

// NoParent is the parent index of a record that belongs to nobody.
const NoParent = -1

// Writer is where the bytes go.
//
// It takes one record at a time rather than a slice, because the streaming mode
// this project promises for large datasets cannot hold them all. A writer that
// needs the whole table, as the nested form does, is free to gather them, and
// then the cost is its own rather than everybody's.
type Writer interface {
	// Begin announces a table and its columns.
	Begin(entity Planned) error
	// Write takes one record.
	Write(record Record) error
	// End closes a table.
	End(entity Planned) error
	// Finish is called once, after every table.
	//
	// It exists for the shape that cannot be written until the last record has
	// arrived. A writer that streams has nothing to do here and says so by
	// answering nil, which is better than every caller having to know which
	// kind of writer it holds.
	Finish() error
}
