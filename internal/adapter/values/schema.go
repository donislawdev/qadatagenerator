package values

import (
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// Schema is a recipe seen as a shape of data, with the type of every field
// filled in and the numbering scheme named wherever there is one.
//
// # Why this is not a decorator over the schema view
//
// The obvious shape would wrap recipe.SchemaView and fill in what is missing.
// It cannot: FieldInfo deliberately does not carry the generator, because the
// corruption engine has no business knowing which one produced a value, and
// that window being small is what keeps the two pillars from growing into each
// other. So the type has to be worked out where the generator is still visible,
// which is the recipe itself.
//
// # Why it lives in this package
//
// The mapping from a generator to the kind of value it produces belongs to
// whoever implements the generators, and that is here. Putting it in the domain
// would mean the model of a recipe knowing the names of things this build
// happens to carry, and those names change with every provider added.
type Schema struct {
	rec    recipe.Recipe
	source Source
}

// Contract check, so that a mismatch fails to compile rather than at run time.
var _ recipe.SchemaView = Schema{}

// SchemaOf returns the shape of a recipe, with field types resolved.
func SchemaOf(rec recipe.Recipe, source Source) Schema {
	return Schema{rec: rec, source: source}
}

// FieldsOf returns the fields of one entity, with the type and the numbering
// scheme filled in from the generator wherever the recipe declared one instead
// of a type.
//
// A recipe field names exactly one of type, generator or expression, so a field
// with a generator carries no type at all. Left that way, the engine asks each
// catalog entry whether it covers a field of no type, and only the two entries
// using the reserved word any can say yes. Fifty of the fifty two would then be
// inapplicable to an ordinary recipe for a reason nobody chose.
func (s Schema) FieldsOf(entity string) []recipe.FieldInfo {
	fields := s.rec.FieldsOf(entity)
	found, known := s.rec.Entity(entity)
	if !known {
		return fields
	}
	for i := range fields {
		field, has := found.Field(fields[i].Name)
		if !has {
			continue
		}
		// A field described by a type that a generator fills is that
		// generator under another name (fillers), so damage reads it as the
		// generator's column, with its type and its scheme.
		generator := field.Generator
		if filler, filled := s.source.FillerOf(field.Type); generator == "" && filled {
			generator = filler
		}
		if generator == "" {
			continue
		}
		field.Generator = generator
		// 🔴 There is deliberately no check here for a type the recipe stated
		// itself, and the reason is worth keeping. One was written, and a
		// mutation came back PRZEOCZONA against it: a field naming a type never
		// names a generator, because the parser refuses a field that says twice
		// where its value comes from. So the guard could never run, and a
		// condition nobody can break is a condition nobody can prove. The same
		// finding as the dead ninth weight in ABA and the overlapping conditions
		// in the postal oracle.
		//
		// What protects a stated type is therefore the parser, not this loop.
		// If the two ever become expressible together, this needs the check back
		// and a mutation that can fail.
		if kind, produces := s.source.TypeOf(field.Generator); produces {
			fields[i].Type = kind
		}
		// Which scheme the column follows, for the same reason as the type and
		// from the same place. A generator drawing on a word list or on
		// arithmetic leaves this empty, which is the truthful answer: nothing
		// has said those values obey any published numbering rules.
		if scheme, draws := s.source.ProviderOf(field.Generator); draws {
			fields[i].Scheme = scheme
		}
		// Which part of a person's name the column holds, from the same place
		// again. A recipe that writes names into a field of type string, with
		// no generator, says nothing about what is in it, so nothing is set.
		if part, names := s.source.NamePartOf(field.Generator); names {
			fields[i].NamePart = part
		}
	}
	return fields
}

// The rest of the view is the recipe's own answer, passed through. They are
// written out rather than embedded so that adding a method to the interface
// fails to compile here, instead of silently inheriting an answer that has not
// been thought about.

// EntityNames returns the entity names in the order they were written.
func (s Schema) EntityNames() []string { return s.rec.EntityNames() }

// ParentsOf returns what the engine may know about the relations of an entity.
func (s Schema) ParentsOf(entity string) []recipe.ParentInfo { return s.rec.ParentsOf(entity) }

// RulesOf returns the consistency rules a run of an entity keeps, the declared
// ones followed by the ones the entity implies.
//
// 🔴 Not the declared ones alone, and the reason is the one rule serving both
// pillars. The run keeps an implied agreement, so the data holds it, so the
// corruption engine has something real to pull apart. Answering with the
// declared rules here would leave a dataset that agrees with itself and a
// policy told there is nothing in it that agrees.
func (s Schema) RulesOf(entity string) []recipe.Rule {
	found, known := s.rec.Entity(entity)
	if !known {
		return s.rec.RulesOf(entity)
	}
	return generate.Rules(found, s.source)
}

// PrimaryKeyOf returns the fields identifying a record, empty when none was
// declared.
func (s Schema) PrimaryKeyOf(entity string) []string { return s.rec.PrimaryKeyOf(entity) }
