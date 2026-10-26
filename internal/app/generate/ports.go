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
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Request is everything the source of values needs to produce one value.
type Request struct {
	// Generator is the name the recipe used, such as first_name.
	Generator string
	// Type is what the recipe declared instead of a generator, such as
	// integer. Exactly one of the two is set, because a recipe field names one
	// source of its value and the parser refuses a field that names two.
	Type string
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
	// TypeOf returns the kind of value a generator produces, and whether the
	// generator is known at all.
	//
	// It is here rather than beside the recipe because the answer belongs to
	// whoever implements the generator. The schema view the corruption engine
	// reads is built from it, so damage aimed at a column of dates reaches a
	// column of dates rather than nothing.
	TypeOf(generator string) (string, bool)
	// ProducesType reports whether this build can fill a field described by
	// that type alone.
	//
	// The plan asks before the run does, so a column this build will leave
	// empty is named while there is still time to change the recipe. Answering
	// this from the run instead would mean either a failure late or a quiet
	// blank column, and the blank column is what this project calls silence.
	ProducesType(kind string) bool
	// TypeFits reports whether a field described by that type alone can hold
	// a value in the length it declares, and says why not when it cannot.
	//
	// 🔴 Asked by the plan since 2026-10-26. A decimal declared three
	// characters long passed every check, and the run wrote 0.44 into every
	// row, four characters each, because the shortest decimal this build makes
	// is a digit, a point and two places. Only whoever builds the value knows
	// how short it can be, so the answer is here rather than in the plan.
	TypeFits(kind string, maxLength int) (bool, string)
	// CanProduce reports whether this build can fill a field with that
	// generator in a locale, and says why not when it cannot.
	//
	// 🔴 Known answers about the name and this answers about the pair. A build
	// carrying a generator and not the word list it draws on in the locale
	// asked for used to pass every check and fail on the first row, so validate
	// called such a recipe usable and a dry run counted the rows of a dataset
	// that could not be made. The locale travels as an argument because the
	// question is about the recipe rather than about whoever is asking.
	CanProduce(generator, locale string) (bool, string)
	// RoleOf returns the role a generator plays in a consistency rule, and
	// whether it plays one at all.
	//
	// The role comes from the generator rather than from the name of the
	// column, because a recipe may call a column anything and half of them
	// will not be in English.
	RoleOf(generator string) (string, bool)
	// RolesRead returns the roles a generator takes account of in the locale
	// this source draws in, sorted.
	//
	// It is what lets a run keep a person consistent without a rule saying
	// so. A first name beside a gender column follows it because the first
	// name reads the gender, and only the source knows which generator reads
	// what, and in Polish a surname reads it while in English it does not.
	RolesRead(generator string) []string
	// Produce returns one value.
	Produce(req Request) (string, error)
}

// Halt is how a run learns that something outside it wants it to stop.
//
// # Why a port rather than a context
//
// The answer carries the kind of the failure, so the run ends through the same
// path as every other failure and the exit code comes from the kind rather than
// from a second decision made at the top. A context would carry cancellation but
// not which of the two signals arrived, and the two have different codes that
// somebody else's pipeline reads.
//
// # Why it answers instead of blocking
//
// A use case that waits on something is a use case with a schedule in it. This
// one is asked between rows, answers immediately, and the whole question of what
// a signal is stays in the adapter that owns the process.
type Halt interface {
	// Stopped returns the failure that ends the run, or nil to carry on.
	//
	// It gives the same answer every time it is asked after the first stop. A
	// port that consumed the stop would let a second caller carry on, which is
	// the quiet half of a run that was told to end.
	Stopped() error
}

// NeverStops is a Halt for a caller with nothing that could interrupt it.
//
// It exists so that such a caller passes something rather than nil. A nil
// interface here would be a panic at the first row instead of a run that
// finishes, and the difference would show up in production rather than in a
// test.
type NeverStops struct{}

// Stopped always answers that the run may carry on.
func (NeverStops) Stopped() error { return nil }

// Writer is where the bytes go.
//
// It takes one record at a time rather than a slice, because the streaming mode
// this project promises for large datasets cannot hold them all. A writer that
// needs the whole table, as the nested form does, is free to gather them, and
// then the cost is its own rather than everybody's.
type Writer interface {
	// Begin announces a table and its columns.
	//
	// 🔴 It comes before every Write and End of that table, and the writers rely
	// on it rather than checking. They keep the table Begin announced and reach
	// for it by position, so a Write arriving first panics instead of failing.
	// Run is the only caller and holds the order, which is why this is written
	// down rather than guarded: a check in each writer would be four copies of
	// one rule, and the rule belongs to whoever drives them.
	Begin(entity dataset.Table) error
	// Write takes one record. It belongs to the table Begin last announced.
	Write(record dataset.Record) error
	// End closes a table.
	End(entity dataset.Table) error
	// Finish is called once, after every table.
	//
	// It exists for the shape that cannot be written until the last record has
	// arrived. A writer that streams has nothing to do here and says so by
	// answering nil, which is better than every caller having to know which
	// kind of writer it holds.
	Finish() error
}
