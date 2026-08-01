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
	Name        string
	Count       int
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
		fields = append(fields, FieldInfo{Name: field.Name, Type: field.Type, Unique: field.Unique})
	}
	return fields
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
