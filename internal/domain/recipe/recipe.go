// Package recipe holds what a user asked to be generated.
//
// It is a model and nothing else. It does not know that recipes arrive as
// YAML, it does not read files, and it does not decide how a problem is
// printed. Those belong to the layers above, which is what lets this one be
// exercised without a disk and reused by the window later.
package recipe

import "sort"

// Recipe is one description of a dataset.
//
// The order of entities and of fields inside them is the order the user wrote,
// not alphabetical. Column order in the output follows it, and a diff in a
// pull request has to stay small, which is the whole point of keeping recipes
// beside the tests.
type Recipe struct {
	Version  string
	Seed     string
	Locale   string
	Name     string
	Tag      string
	Entities []Entity
	Output   Output
}

// Entity is a set of records of one kind.
type Entity struct {
	Name  string
	Count int
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

// Unit returns how the maximum length of this field is counted, defaulting to
// characters. Callers use it rather than reading LengthUnit, so the default
// lives in one place.
func (f Field) Unit() LengthUnit {
	if f.LengthUnit == "" {
		return Characters
	}
	return f.LengthUnit
}

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
	Encoding      string
	Delimiter     string
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
}
