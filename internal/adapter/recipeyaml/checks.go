package recipeyaml

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// The questions that need more than the key being read at the time.
//
// They live apart from the walk for two reasons. The walk answers "what does
// this line say", and these answer "does the recipe hold together", which is a
// different job at a different moment: each one runs after its part of the tree
// is complete, because a rule naming a field cannot be judged while half the
// fields are still unread.
//
// The split was forced by size on 2026-08-03, when the reader crossed the file
// limit, and the seam it fell along is the one that was already there.

// checkField asks the questions that need the whole field rather than one key.
func (r *reader) checkField(built recipe.Field, where string) {
	// A field that must always carry a value and is also asked to be empty
	// part of the time. Both halves are legal on their own, so nothing else
	// would catch this, and whichever half won at generation time would be
	// the tool deciding which one the user meant.
	if built.Required && built.HasEmptyShare && built.EmptyShare > 0 {
		r.report(where, "the field is required and is also asked to be empty in some rows",
			"either required: true or a share under empty, not both",
			"drop required if some rows should be empty, or drop empty if none should be")
	}

	// 🔴 A field asked to be unique and also to be empty in some rows. An empty
	// value is still a value, so the second row given one repeats the first and
	// the column does not hold what it declares. Nothing downstream notices:
	// emptiness is decided before uniqueness is ever asked, so the set that
	// remembers what has been taken never sees an empty value at all. Measured
	// on 2026-08-26 with unique: true and empty: 100%, which produced a column
	// of nothing but empty values, ended with nought and said not a word. A
	// dataset put behind a UNIQUE constraint is then refused by the database
	// for something the recipe itself declared.
	if built.Unique && built.HasEmptyShare && built.EmptyShare > 0 {
		r.report(where, "the field is asked to be unique and is also asked to be empty in some rows",
			"either unique: true or a share under empty, not both",
			"drop unique if some rows should be empty, or drop empty if every value has to differ")
	}

	// A unit with nothing to count. It reads as though a limit was set, which
	// is exactly why it is worth a line rather than being ignored.
	if built.LengthUnit != "" && !built.HasMaxLength {
		r.report(where+".counted_in", "the field says how to count a length it does not declare",
			"", "add max_length, or remove counted_in")
	}

	sources := 0
	for _, source := range []string{built.Type, built.Generator, built.Expression} {
		if source != "" {
			sources++
		}
	}
	switch sources {
	case 0:
		r.report(where, "the field does not say where its value comes from",
			"type, generator, expression", "give the field one of them")
	case 1:
	default:
		r.report(where, "the field says twice where its value comes from",
			"exactly one of type, generator, expression", "keep the one you meant")
	}
}

// checkPrimaryKey asks whether the key names fields that exist and names each
// of them once.
//
// A key naming a field twice is worth reporting rather than tidying away. It
// usually means somebody meant two different fields, and silently collapsing
// it would leave a one column key where the recipe asked for two.
func (r *reader) checkPrimaryKey(entity recipe.Entity, where string) {
	if len(entity.PrimaryKey) == 0 {
		return
	}
	fields := map[string]bool{}
	for _, field := range entity.Fields {
		fields[field.Name] = true
	}
	seen := map[string]bool{}
	for _, name := range entity.PrimaryKey {
		at := where + ".primary_key"
		if !fields[name] {
			r.report(at, "the entity has no field "+name,
				strings.Join(entity.FieldNames(), ", "), "check the spelling")
			continue
		}
		if seen[name] {
			r.report(at, "the key names "+name+" twice", "",
				"name each field once, or name the second field you meant")
		}
		seen[name] = true
	}
}

// checkRules asks whether every field a rule names exists in the entity.
func (r *reader) checkRules(entity recipe.Entity, wheres []string) {
	fields := map[string]bool{}
	for _, field := range entity.Fields {
		fields[field.Name] = true
	}
	for index, rule := range entity.Consistency {
		at := wheres[index]
		for _, name := range append([]string{rule.Field}, rule.With...) {
			if !fields[name] {
				r.report(at, "the entity has no field "+name,
					strings.Join(entity.FieldNames(), ", "), "check the spelling")
			}
		}
	}
}

// ruleSaysSomething refuses the two rules that parse and mean nothing.
//
// Both used to be accepted in silence, and a consistency rule is the one
// declaration serving both pillars: it says what has to agree when the data is
// generated and therefore what can be pulled apart when it is corrupted. A rule
// that means nothing gives the first pillar nothing to satisfy and the second
// nothing to break, so it would surface much later as a corruption that was
// never applied, with a reason nobody could act on.
//
// The count of problems already reported is passed in so that an empty list
// which was empty because its entries were unreadable is not reported twice.
// The entries were named individually where they were read, which is the more
// useful of the two messages.
func (r *reader) ruleSaysSomething(at, field string, with []string, reportedBefore int) bool {
	if len(with) == 0 {
		if len(r.problems) == reportedBefore {
			r.report(at, "the rule does not say what "+field+" has to agree with", "at least one field name",
				"write it as "+field+" matches: [birth_date, gender], or remove the rule")
		}
		return false
	}
	for _, name := range with {
		if name == field {
			r.report(at, "the rule says "+field+" has to agree with itself", "the names of other fields",
				"name the field it really follows, or remove the rule")
			return false
		}
	}
	return true
}

// checkReferences answers the questions that need the whole recipe: whether
// every name a rule or an expression mentions is a field that exists.
func (r *reader) checkReferences(built recipe.Recipe) {
	declared := map[string]bool{}
	for _, entity := range built.Entities {
		declared[entity.Name] = true
	}

	for _, entity := range built.Entities {
		where := "entities." + entity.Name
		fields := map[string]bool{}
		for _, field := range entity.Fields {
			fields[field.Name] = true
		}
		parents := map[string]bool{}
		for _, parent := range entity.Parents {
			parents[parent.Entity] = true
			if !declared[parent.Entity] {
				r.report(where+".belongs_to."+parent.Entity, "the recipe has no such entity",
					strings.Join(built.EntityNames(), ", "), "check the spelling or declare it")
			}
		}
		r.checkExpressions(built, entity, where, fields, parents)
	}
}

// checkExpressions asks whether every name an expression mentions can be
// resolved, for one entity.
func (r *reader) checkExpressions(built recipe.Recipe, entity recipe.Entity, where string, fields, parents map[string]bool) {
	for _, field := range entity.Fields {
		if field.Expression == "" {
			continue
		}
		at := where + ".fields." + field.Name + ".expression"
		for _, reference := range referencesIn(field.Expression) {
			owner, name, isParent := strings.Cut(reference, ".")
			switch {
			case isParent && !parents[owner]:
				r.report(at, entity.Name+" does not belong to "+owner,
					"", "declare the relation under belongs_to or use a field of this entity")
			case isParent:
				r.checkParentField(built, at, owner, name)
			case fields[reference]:
			default:
				// It may be a value the tool supplies, such as a random
				// domain. Those arrive with the generators, so a name that
				// is not a field cannot be told from a typo yet.
				r.note(at, "the reference to "+reference,
					"it is not a field of this entity, so it is either a value the tool supplies or a misspelling, "+
						"and telling those apart needs the generators")
			}
		}
	}
}

func (r *reader) checkParentField(built recipe.Recipe, at, parent, field string) {
	owner, ok := built.Entity(parent)
	if !ok {
		return
	}
	if _, ok := owner.Field(field); !ok {
		r.report(at, parent+" has no field "+field,
			strings.Join(owner.FieldNames(), ", "), "check the spelling")
	}
}
