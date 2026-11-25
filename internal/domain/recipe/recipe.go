// Package recipe holds what a user asked to be generated.
//
// It is a model and nothing else. It does not know that recipes arrive as
// YAML, it does not read files, and it does not decide how a problem is
// printed. Those belong to the layers above, which is what lets this one be
// exercised without a disk and reused by the window later.
package recipe

import (
	"sort"
	"unicode/utf8"
)

// What a recipe that leaves the seed or the locale out is read with.
//
// # Why fixed, and why here
//
// Decided by the owner on 2026-10-26 (DEC-266). Until then a recipe with no
// seed drew its data from the empty text and its manifest was refused at the
// end of the run, after the whole dataset had been produced, because an
// envelope without a seed says the run cannot be repeated. A seed that is the
// same every time keeps the promise that one recipe gives one dataset, which a
// drawn seed would not in a pipeline nobody reads the log of. "0" is what
// corrupt has always used when nobody said. The locale is the one the source of
// values already fell back to. Both live in the model so that the reader of
// recipes, the run and the record beside the data cannot each pick their own.
const (
	DefaultSeed   = "0"
	DefaultLocale = "en-US"
)

// Recipe is one description of a dataset.
//
// The order of entities and of fields inside them is the order the user wrote,
// not alphabetical. Column order in the output follows it, and a diff in a
// pull request has to stay small, which is the whole point of keeping recipes
// beside the tests.
type Recipe struct {
	Version string
	Seed    string
	// SeedGiven says whether the recipe wrote the seed, since a recipe that
	// did not carries DefaultSeed and the two read the same. The record of an
	// override needs the difference: a seed from the command line replaces
	// nothing the recipe said when the recipe said nothing.
	SeedGiven bool
	Locale    string
	Name      string
	Tag       string
	Entities  []Entity
	Output    Output
	// Policy is what to damage, empty when the recipe asks for correct data
	// only. The first pillar is useful on its own, so a recipe with no policy
	// is an ordinary recipe rather than an incomplete one.
	Policy Policy
	// Unchecked is what the reader of the recipe carried without reading,
	// such as a setting of a generator this build does not understand.
	//
	// 🔴 It travels with the recipe since 2026-10-26. It used to stay with the
	// reader, and generate takes the recipe alone, so validate warned that a
	// date of birth would ignore its range while generate wrote dates outside
	// it without a word and the record beside the data said nothing was left
	// undone. Found by an outside review on 2026-10-22.
	Unchecked []Unchecked
}

// Policy is what a run should break.
//
// # Why it names layers rather than mutators
//
// A layer is a line of defence, and a person deciding what to test thinks in
// those terms: the bytes, the container, the shape of a field, the meaning of a
// value, the rules of the business, the integrity of the set. Naming individual
// mutators would tie a recipe living in somebody's repository to the catalog
// this build happens to carry, so a recipe would stop working when the catalog
// grew. Naming layers lets the catalog grow underneath it.
type Policy struct {
	// Layers are the lines of defence to attack. Empty means no damage was
	// asked for.
	Layers []string
	// Rows is the share of rows to damage, as written, for example 5%.
	//
	// It is kept as text rather than a number because the recipe format says
	// a share can also be given as particular rows, and turning one of those
	// two into the other here would decide something this model does not own.
	Rows string
	// PerRow is how many defects one row may carry.
	PerRow string
	// ExcludeFields are the columns to leave alone, whatever the layers say.
	ExcludeFields []string
	// OnlyWhen is the condition a row has to meet before it is damaged.
	//
	// Expressions are not implemented, so a recipe carrying one is refused
	// rather than run (DEC-266, 2026-10-26). It used to be run with a warning,
	// and the run damaged every row, including the ones the condition left
	// out. A pipeline reads the exit code and not the warning, so a warning
	// was a run doing more than it was asked with success at the end.
	OnlyWhen string
}

// Asks reports whether this policy orders any damage at all.
func (p Policy) Asks() bool { return len(p.Layers) > 0 }

// Entity is a set of records of one kind.
type Entity struct {
	Name  string
	Count int
	// HasCount separates a count of nought from no count at all.
	//
	// The two mean different things and only one of them is legal for every
	// entity. A root entity with no count has nothing to produce, while a
	// child with no count takes its number from its parents and the
	// cardinality of the relation. Deriving one from the other would make a
	// recipe asking for zero orders indistinguishable from one asking for as
	// many as the relation gives, and the first is a perfectly ordinary thing
	// to ask for while a test is being written.
	HasCount bool
	// PrimaryKey names the fields that identify a record.
	//
	// It sits at the entity rather than at the field because a key is often
	// made of several fields, and a flag on each one could not say in which
	// order they combine. Empty means the recipe declared none, which is
	// legal and leaves the mutators that break keys inapplicable.
	PrimaryKey  []string
	Fields      []Field
	Consistency []Rule
	Parents     []Parent
}

// Field is one value in an entity, together with what it is made from and what
// it has to satisfy.
type Field struct {
	Name string
	// Type, Generator and Expression are three ways of saying where the
	// value comes from. Exactly one of them is set.
	Type       string
	Generator  string
	Expression string
	Unique     bool
	// Required says the field must carry a value.
	//
	// It is a separate statement from EmptyShare on purpose. A share of zero
	// says no row happens to be empty in this run, while Required says an
	// empty value is not allowed at all. Deriving one from the other would
	// make the two indistinguishable exactly when a mutator wants to leave a
	// required field blank and find out whether the application notices.
	Required bool
	// MaxLength is the longest value the field accepts, counted in
	// LengthUnit. HasMaxLength separates an absent limit from a limit of
	// zero.
	MaxLength    int
	HasMaxLength bool
	// LengthUnit is how MaxLength is counted. Empty means Characters.
	//
	// The unit is carried rather than assumed because it is the whole
	// difference between a limit an application enforces and one it does
	// not. A column of thirty characters holds thirty Polish names and ten
	// emoji, and a tool that counted one as the other would produce a value
	// that fits by its own arithmetic and is rejected on arrival.
	LengthUnit LengthUnit
	// EmptyShare is the share of rows left empty, from zero to one hundred.
	EmptyShare    float64
	HasEmptyShare bool
	// Values is a dictionary the field draws from.
	Values []string
	// MinValue and MaxValue are the smallest and the largest number a field
	// described by type alone may take, both included, and empty when the
	// recipe sets no such bound.
	//
	// Each is the decimal text of the number the recipe wrote, such as -5 or
	// 99.99, rather than a float. A float would round a large whole number,
	// and how many places a bound has is part of what the type of the field
	// checks, which only whoever makes the value knows.
	MinValue string
	MaxValue string
	// Parameters are the settings a generator understands and this build
	// does not, such as the age range of a date of birth. They are carried
	// rather than rejected, because rejecting them would reject the shape
	// the format documents, and they are reported as unchecked rather than
	// silently accepted.
	Parameters []Parameter
}

// Parameter is one generator setting, kept as written.
type Parameter struct {
	Name  string
	Value string
}

// Rule says which fields have to agree with each other.
//
// The same declaration serves both pillars. It says what must agree, so it
// also says what can be pulled apart, which is what the layer of business
// rules corrupts.
type Rule struct {
	// Field is the value that has to follow the others.
	Field string
	// Kind is how it follows them.
	Kind RuleKind
	// With is what it has to agree with.
	With []string
}

// RuleKind is the closed set of ways two fields can be tied together.
type RuleKind string

const (
	// Matches means the value has to be consistent with the others, the way
	// a national identifier encodes a date of birth.
	Matches RuleKind = "matches"
	// DerivedFrom means the value is built out of the others.
	DerivedFrom RuleKind = "derived_from"
)

// LengthUnit is the closed set of ways a maximum length can be counted.
//
// Three units and not one, because the three disagree on exactly the values
// this tool exists to produce. The Polish letter s with an acute accent is one
// character, two bytes in UTF-8 and one grapheme. A family emoji is one
// grapheme, several characters and more than a dozen bytes.
type LengthUnit string

const (
	// Characters counts Unicode code points. It is the default, because it
	// is what a person means by length and what most schema languages
	// document.
	Characters LengthUnit = "characters"
	// Bytes counts the encoded form. Databases that declare a column in
	// bytes reject values that fit by any other count.
	Bytes LengthUnit = "bytes"
	// Graphemes counts what a reader sees as one symbol.
	Graphemes LengthUnit = "graphemes"
)

// Count measures a value in this unit.
//
// It is the one count in the build. The generator refusing a value for being
// too long, the damage that goes past a limit on purpose and the engine keeping
// every other damage inside it all ask here, so a value one of them calls too
// long is a value the others agree is too long. Two counts either side of one
// limit would be a tool arguing with itself.
//
// Graphemes are counted as characters, and this build says so rather than
// pretending otherwise: the standard library has no grapheme segmentation, and
// the package that does would be a second dependency. A value never has more
// graphemes than characters, so a value this count lets through fits a limit
// in graphemes too, and what is lost is the other direction: a family emoji
// that a reader sees as one symbol counts as seven.
func (u LengthUnit) Count(value string) int {
	if u == Bytes {
		return len(value)
	}
	return utf8.RuneCountInString(value)
}

// Unit returns how the maximum length of this field is counted, defaulting to
// characters. Callers use it rather than reading LengthUnit, so the default
// lives in one place.
func (f Field) Unit() LengthUnit {
	if f.LengthUnit == "" {
		return Characters
	}
	return f.LengthUnit
}

// Bounded reports whether the recipe gives the field a smallest or a largest
// number.
func (f Field) Bounded() bool { return f.MinValue != "" || f.MaxValue != "" }

// Parent is a relation, declared at the child and carrying how many children
// one parent has.
type Parent struct {
	Entity string
	Min    int
	Max    int
}

// Output says where the dataset goes and in what shape.
type Output struct {
	Format        string
	Dir           string
	FilePerEntity bool
	// FilePerEntityGiven says the recipe wrote file_per_entity at all.
	//
	// A setting nobody wrote and one written as false are the same value and
	// different intentions, and only the second is worth a note saying that the
	// form asked for ignores it. The command line keeps the same distinction
	// for its flags.
	FilePerEntityGiven bool
	Encoding           string
	Delimiter          string
	// Dialect is the database the SQL form writes for, one of
	// dataset.Dialects. It has no default, and the SQL form is refused without
	// one, because no file of statements means the same in every database
	// (DEC-341). The other forms do not read it.
	Dialect string
	// CreateTables asks the SQL form to start every table with a CREATE TABLE
	// statement, its columns typed from the recipe (DEC-341). Without it the
	// file fills tables that already exist. The other forms do not read it.
	CreateTables bool
}

// Problem is one reason a recipe cannot be used, described in parts rather
// than as a sentence.
//
// It carries no formatting, because the surface that shows it decides how it
// reads. The command line prints four lines, the window will put the same
// parts in a panel, and neither should have to take a sentence apart.
type Problem struct {
	// Where names the place in the recipe, such as
	// entities.customer.fields.email.
	Where string
	// What went wrong.
	What string
	// Allowed says what would have been accepted. Empty when the answer is
	// not a short list.
	Allowed string
	// Hint says what to do instead. Empty when the fix is obvious from
	// What.
	Hint string
	// Line is where in the recipe this was written, or zero when the place
	// cannot be tied to exactly one line.
	//
	// Zero is an ordinary answer rather than a failure. A problem about the
	// file as a whole belongs to no line, a problem about a section that was
	// never written has none to point at, and a name carrying a dot makes the
	// path itself ambiguous. Whoever shows this draws no line rather than the
	// wrong one.
	Line int
}

// Unchecked is something a recipe declares that this build cannot verify yet.
//
// It exists so that a report can say what it did not look at. A validation
// that stays silent about its own blind spots is worth less than one that
// finds nothing, because it invites trust it has not earned.
type Unchecked struct {
	Where  string
	What   string
	Reason string
}

// FieldInfo is what the corruption engine is allowed to know about a field.
//
// It is deliberately narrower than Field. The engine has no business knowing
// which generator produced a value or what settings it took, and a window that
// small is what keeps the two pillars from growing into each other.
type FieldInfo struct {
	Name   string
	Type   string
	Unique bool
	// Required, MaxLength, HasMaxLength and LengthUnit are here because the
	// engine genuinely needs them. Leaving a field blank and overrunning a
	// declared limit are both corruptions, and neither can be produced by a
	// mutator that cannot see the constraint it is breaking.
	Required     bool
	MaxLength    int
	HasMaxLength bool
	LengthUnit   LengthUnit
	// Values is the declared value set of an enumerated field, empty when
	// the field is not one. A mutator that puts a value outside the set has
	// to know the set.
	Values []string
	// Scheme is the numbering scheme the column follows, such as PL.pesel,
	// empty when it follows none.
	//
	// It is not the generator under another name, and the difference is what
	// makes it belong here. A generator is how a value came to exist, which
	// is the first pillar's business and none of the engine's. A scheme is
	// what the value has to satisfy, which is a property of the data itself:
	// in the mode that reads somebody else's file there is no generator at
	// all, and the column still holds tax numbers or it does not.
	//
	// The engine needs it because the field type cannot answer for it. Every
	// identifier is of type identifier, and they differ on whether they
	// carry a check digit, encode a date or have a range nobody is issued
	// from. A catalog entry names the schemes it covers precisely because
	// that difference exists.
	//
	// 🔴 It was called Provider until 2026-08-11 and the rename is the point
	// of this field's name rather than tidying. A provider is the Go type
	// that implements a scheme, and this column has no provider: in the mode
	// that reads somebody else's file nothing implements anything, and the
	// column still either holds tax numbers or does not. The binding name in
	// the glossary is numbering scheme, and this field carried the name of
	// the implementation into the one mode where no implementation exists.
	//
	// The catalog key applies_to.providers keeps its name on purpose. That
	// one is a published contract, and changing it belongs to the pass over
	// identifiers before the repository becomes public.
	Scheme string
	// NamePart is the part of a person's name the column holds, empty when it
	// holds no names of people.
	//
	// It is here for the same reason as Scheme: the type cannot answer for
	// it. A column of surnames and a column of towns are both text, and damage
	// that supplies a real surname expects it accepted, which holds in the
	// first and says nothing about the second. What sets it differs by mode,
	// the generator in a recipe and the header in somebody else's file, and
	// the engine never learns which of the two it was.
	NamePart NamePart
}

// NamePart is the part of a person's name a column holds, or the whole of it.
type NamePart string

// The parts of a name a column can hold.
//
// A whole name is one of them rather than a pair of the other two, because
// nothing in its value says where the given name ends. Jan Kowalski and
// Kowalski Jan are both written, and the damage that needs a column of names
// replaces the whole value with a whole name, so it never has to know.
const (
	// GivenName is a first name, or a second one.
	GivenName NamePart = "given"
	// Surname is a family name.
	Surname NamePart = "surname"
	// FullName is a given name and a surname in one value.
	FullName NamePart = "full"
)

// Unit returns how the maximum length of this field is counted, defaulting to
// characters.
//
// The same answer as the one a recipe field gives, because it is the same
// question and the default has to live in one place. A view built by looking at
// somebody else's file may leave the unit unset, and reading it raw there would
// make a mutator measure in nothing at all.
func (f FieldInfo) Unit() LengthUnit {
	if f.LengthUnit == "" {
		return Characters
	}
	return f.LengthUnit
}

// Fits reports whether a value is within the maximum length the field
// declares, counted in its unit. A field that declares none fits anything.
func (f FieldInfo) Fits(value string) bool {
	return !f.HasMaxLength || f.Unit().Count(value) <= f.MaxLength
}

// ParentInfo is what the engine is allowed to know about a relation.
type ParentInfo struct {
	Entity string
	Min    int
	Max    int
}

// SchemaView is the shape of data, without saying where the shape came from.
//
// The corruption engine takes this and never the recipe itself, because in the
// take and corrupt mode there is no recipe at all and the view is worked out
// from somebody else's file. One interface, two sources, and the engine cannot
// tell them apart.
type SchemaView interface {
	EntityNames() []string
	FieldsOf(entity string) []FieldInfo
	ParentsOf(entity string) []ParentInfo
	RulesOf(entity string) []Rule
	// PrimaryKeyOf returns the fields identifying a record, empty when none
	// was declared. Duplicating a key and producing a second record for the
	// same person are different corruptions, and both need to know it.
	PrimaryKeyOf(entity string) []string
}

// A recipe is one source of a schema view.
var _ SchemaView = Recipe{}

// EntityNames returns the entity names in the order they were written.
func (r Recipe) EntityNames() []string {
	names := make([]string, 0, len(r.Entities))
	for _, entity := range r.Entities {
		names = append(names, entity.Name)
	}
	return names
}

// FieldsOf returns the narrow view of the fields of one entity.
func (r Recipe) FieldsOf(entity string) []FieldInfo {
	found, ok := r.Entity(entity)
	if !ok {
		return nil
	}
	fields := make([]FieldInfo, 0, len(found.Fields))
	for _, field := range found.Fields {
		fields = append(fields, FieldInfo{
			Name:         field.Name,
			Type:         field.Type,
			Unique:       field.Unique,
			Required:     field.Required,
			MaxLength:    field.MaxLength,
			HasMaxLength: field.HasMaxLength,
			LengthUnit:   field.Unit(),
			Values:       field.Values,
		})
	}
	return fields
}

// PrimaryKeyOf returns the fields identifying a record of one entity.
func (r Recipe) PrimaryKeyOf(entity string) []string {
	found, ok := r.Entity(entity)
	if !ok {
		return nil
	}
	return found.PrimaryKey
}

// ParentsOf returns the relations declared at one entity.
func (r Recipe) ParentsOf(entity string) []ParentInfo {
	found, ok := r.Entity(entity)
	if !ok {
		return nil
	}
	parents := make([]ParentInfo, 0, len(found.Parents))
	for _, parent := range found.Parents {
		parents = append(parents, ParentInfo(parent))
	}
	return parents
}

// RulesOf returns the consistency rules of one entity.
func (r Recipe) RulesOf(entity string) []Rule {
	found, ok := r.Entity(entity)
	if !ok {
		return nil
	}
	return found.Consistency
}

// Entity finds an entity by name.
func (r Recipe) Entity(name string) (Entity, bool) {
	for _, entity := range r.Entities {
		if entity.Name == name {
			return entity, true
		}
	}
	return Entity{}, false
}

// Field finds a field by name.
func (e Entity) Field(name string) (Field, bool) {
	for _, field := range e.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return Field{}, false
}

// FieldNames returns the field names in the order they were written.
func (e Entity) FieldNames() []string {
	names := make([]string, 0, len(e.Fields))
	for _, field := range e.Fields {
		names = append(names, field.Name)
	}
	return names
}

// ParentColumnCollision returns the name a child would carry twice, once as a
// field of its own and once as the column pointing at the entity it belongs
// to, and whether there is one.
//
// # Why the rule lives here rather than where it is fatal
//
// It is fatal in the writer composing a file, because that is where two things
// would go into one column, and it was written there and stayed there. The
// cost of that was measured on 2026-09-02 and recorded as B-205: validate
// called such a recipe usable, a dry run counted its rows, and generate refused
// it with a message telling the reader to run validate. The refusal sent people
// to the verb that had just approved the file.
//
// A collision between a field name and a relation is a fact about a recipe, not
// about a file format, so it belongs where the recipe does. Both surfaces now
// ask this one function, and the wording each of them uses is its own, the way
// an unknown generator is already reported twice in two shapes.
//
// Every parent is looked at rather than only the first. A second parent is
// refused elsewhere, so today the loop sees one, and the day it sees two this
// answers about both instead of about whichever came first.
func (e Entity) ParentColumnCollision() (string, bool) {
	for _, parent := range e.Parents {
		for _, field := range e.Fields {
			if field.Name == parent.Entity {
				return parent.Entity, true
			}
		}
	}
	return "", false
}

// SortProblems puts problems in a stable order so that two runs over the same
// recipe report them the same way. Sorting by where they are rather than by
// the order they were found also groups everything about one field together.
func SortProblems(problems []Problem) {
	sort.SliceStable(problems, func(i, j int) bool {
		if problems[i].Where != problems[j].Where {
			return problems[i].Where < problems[j].Where
		}
		return problems[i].What < problems[j].What
	})
	// Sorted by place and not by line, which is deliberate and worth saying
	// because a list carrying line numbers invites the other order. Problems
	// about one field belong together, and a person fixing a recipe fixes a
	// field rather than a line.
}
