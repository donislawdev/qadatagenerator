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
	// Values is the list the recipe wrote under the field, empty when it wrote
	// none. A field with a list draws from it and from nothing else.
	Values []string
	// MinValue and MaxValue bound the number a field described by type alone
	// draws, as the recipe wrote them, and are empty when it wrote none.
	MinValue string
	MaxValue string
}

// Domain is every value a request can produce, when the source knows them all.
//
// # Why a size and a function rather than a list
//
// A whole number of six digits has a million values, and a list of them would
// cost a run more memory than the dataset it writes. The run needs two things
// from a domain, how large it is and the value at one place in it, and both are
// arithmetic for a number and a lookup for a word list.
type Domain struct {
	// Key names the domain, so that two rows drawing from the same one share
	// what was learned about it. A first name has two of them, one for each
	// gender, and a row of a woman has nothing to learn from a row of a man.
	Key string
	// Size is how many values there are. Nought means none fits the field.
	Size int64
	// At returns the value at a place from nought to Size-1. The order is the
	// domain's own and stays the same for the same request.
	At func(index int64) string
}

// Shape is what the values a request produces are, as far as a table made to
// hold them needs to know (DEC-341).
//
// # Why records stay text and this is a separate answer
//
// A record carries every value as text, damage included, and the writers write
// text. A shape describes the values a run puts in a column when nothing damages
// them, which is what a table declared for them has to hold. Damage is free to
// put anything into the column, and the database the table lives in decides
// whether to take it, which is what a test of broken data wants to see.
type Shape struct {
	Kind ShapeKind
	// Digits is how many digits the whole part of the longest number has, for
	// ShapeInteger and ShapeDecimal.
	Digits int
	// Places is how many digits follow the point, for ShapeDecimal.
	Places int
}

// ShapeKind is the closed set of kinds of values a column can hold.
type ShapeKind string

// The kinds of values a column can hold.
const (
	// ShapeText is any text, and what a column is when nothing says otherwise.
	ShapeText ShapeKind = "text"
	// ShapeInteger is a whole number written in digits.
	ShapeInteger ShapeKind = "integer"
	// ShapeDecimal is a number written in digits with a point.
	ShapeDecimal ShapeKind = "decimal"
	// ShapeDate is a day written as year-month-day.
	ShapeDate ShapeKind = "date"
	// ShapeUUID is a UUID written in its 36 characters.
	ShapeUUID ShapeKind = "uuid"
)

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
	// TypeFits refuses a field described by type alone whose length and bounds
	// do not hold together, with what to do instead, and returns nil for one
	// that can be filled.
	//
	// 🔴 Asked by the plan, because a decimal declared three characters long
	// passed every check and the run wrote 0.44 into every row, four
	// characters each (measured 2026-10-26). The same holds for a range: a
	// bound with three places on a decimal, a bound longer than the field, a
	// smallest number above the largest. Only whoever builds the value knows
	// how short it can be and what a range of its type looks like, so the
	// answer is here rather than in the plan.
	//
	// The refusal carries its own next step because the reasons differ in
	// what fixes them, and one sentence for all of them would be true of none.
	TypeFits(req Request) error
	// ShapeOf says what the values of a field are, for a table made to hold
	// them.
	//
	// Only whoever builds a value knows the type a generator gives and the
	// longest number a range allows, the reason TypeFits is here as well. The
	// request needs no address, since the answer is the same for every row:
	// FieldRequest makes it.
	ShapeOf(req Request) Shape
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
	// DomainOf returns every value the request can produce in the field it
	// describes, after its limit on length and its agreement with the row, and
	// whether the source knows them all.
	//
	// 🔴 Asked since 2026-10-26. A unique field used to give up after twenty
	// draws in a row landed on taken values and said the field had fewer values
	// than the rows asked for, which the run did not know. Where the answer is
	// known the plan refuses with the arithmetic before a row is written and the
	// run walks to a free value instead of giving up. Only the source knows how
	// many values a word list holds after the filter, so the question is here.
	//
	// The answer depends on everything in the request but its address. The
	// plan, the walk of a key and the walk of a unique field each ask once for
	// a field whose requests differ only there, and keep the answer for every
	// row of the table.
	DomainOf(req Request) (Domain, bool)
	// Accepts reports whether a value written in the recipe under values can
	// stand in the field the request describes, and says why not when it cannot.
	Accepts(req Request, value string) (bool, string)
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
