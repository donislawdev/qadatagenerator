package generate

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Implied is an agreement a run keeps although no rule in the recipe wrote it,
// together with the entity it belongs to. Its rule names only the columns that
// were added, so that a reader is told what the recipe did not say rather than
// what it did.
type Implied struct {
	Entity string
	Rule   recipe.Rule
}

// Rules are the consistency rules a run of an entity keeps: the ones the
// recipe declares, each completed with what it leaves out, followed by the
// ones the entity implies.
//
// # Why a rule can be implied at all
//
// The recipe format promises that consistency is on by default, and that a
// record which disagrees with itself is ordered through the corruption policy
// rather than reached by leaving a rule out. Until 2026-10-21 a run kept only
// the rules written down, so the example recipe the window offers gave the name
// Piotr to a woman in about half of its rows, while the PESEL beside her agreed
// with her gender because that one rule was written. Measured on 2000 rows of
// that recipe: 999 names disagreed with the gender column.
//
// # What is implied, and what is left alone
//
// A field follows the column playing each role its generator reads, so a first
// name follows the gender and a PESEL follows the date of birth and the gender.
// A declared rule is completed the same way, because a PESEL told to match the
// gender alone used to carry a date of birth of its own beside a birth_date
// column, and nobody writes a rule meaning that. Three things keep this from
// being a guess:
//
//   - a role the rule already reads through a column it names is left alone,
//     so a recipe that writes its rules out produces what it produced before,
//   - a role played by two columns binds neither, because which of them a name
//     follows is the recipe's to say, and the plan says so in a note,
//   - a column a declared rule binds plays no role here, because a run resolves
//     one level of agreement and a rule resting on a bound column is refused.
func Rules(entity recipe.Entity, values Values) []recipe.Rule {
	rules, _ := keep(entity, values)
	return rules
}

// implied returns what a run of an entity keeps beyond what its rules say.
func implied(entity recipe.Entity, values Values) []recipe.Rule {
	_, added := keep(entity, values)
	return added
}

// keep works out the rules a run keeps and, beside them, what was added to the
// recipe: for each field, only the columns it follows that no rule names.
func keep(entity recipe.Entity, values Values) (rules, added []recipe.Rule) {
	bound := boundFields(entity.Consistency)
	players := playersOf(entity, values, bound)
	for _, rule := range entity.Consistency {
		if rule.Kind == recipe.Matches {
			if extra := missing(entity, values, players, rule.Field, rule.With); len(extra) > 0 {
				added = append(added, recipe.Rule{Field: rule.Field, Kind: recipe.Matches, With: extra})
				rule.With = append(append([]string(nil), rule.With...), extra...)
			}
		}
		rules = append(rules, rule)
	}
	for _, field := range entity.Fields {
		if field.Generator == "" || bound[field.Name] {
			continue
		}
		if extra := missing(entity, values, players, field.Name, nil); len(extra) > 0 {
			rule := recipe.Rule{Field: field.Name, Kind: recipe.Matches, With: extra}
			rules = append(rules, rule)
			added = append(added, rule)
		}
	}
	return rules, added
}

// missing names the columns a field ought to follow and is not told to: one
// for each role its generator reads, where exactly one free column plays it and
// none of the columns already named plays it.
func missing(entity recipe.Entity, values Values, players map[string][]string,
	field string, named []string) []string {
	subject, known := entity.Field(field)
	if !known || subject.Generator == "" {
		return nil
	}
	covered := rolesOf(entity, values, named)
	var with []string
	for _, role := range values.RolesRead(subject.Generator) {
		if playing := players[role]; !covered[role] && len(playing) == 1 && playing[0] != field {
			with = append(with, playing[0])
		}
	}
	return with
}

// unsure notes every field that reads a role two columns play, and therefore
// follows neither of them.
//
// A note rather than a refusal, because the recipe is not wrong: two gender
// columns are an ordinary thing to have, and a name that follows neither is
// what the recipe produced before rules were implied. What would be wrong is
// saying nothing, since the reader expects the name to follow one of them.
func unsure(entity recipe.Entity, values Values) []Note {
	bound := boundFields(entity.Consistency)
	players := playersOf(entity, values, bound)
	named := map[string][]string{}
	for _, rule := range entity.Consistency {
		named[rule.Field] = append(named[rule.Field], rule.With...)
	}
	var notes []Note
	for _, field := range entity.Fields {
		if field.Generator == "" {
			continue
		}
		covered := rolesOf(entity, values, named[field.Name])
		for _, role := range values.RolesRead(field.Generator) {
			playing := players[role]
			if covered[role] || len(playing) < 2 {
				continue
			}
			notes = append(notes, Note{"entities." + entity.Name + ".fields." + field.Name,
				field.Name + " could follow " + strings.Join(playing, " or ") +
					", so it follows neither. Write which one under consistency, for example " +
					field.Name + " matches: [" + playing[0] + "]"})
		}
	}
	return notes
}

// rolesOf returns the roles the named columns play.
func rolesOf(entity recipe.Entity, values Values, names []string) map[string]bool {
	roles := map[string]bool{}
	for _, name := range names {
		if field, known := entity.Field(name); known {
			if role, plays := values.RoleOf(field.Generator); plays {
				roles[role] = true
			}
		}
	}
	return roles
}

// playersOf names the columns playing each role, leaving out the ones a
// declared rule binds.
func playersOf(entity recipe.Entity, values Values, bound map[string]bool) map[string][]string {
	players := map[string][]string{}
	for _, field := range entity.Fields {
		role, plays := values.RoleOf(field.Generator)
		if plays && !bound[field.Name] {
			players[role] = append(players[role], field.Name)
		}
	}
	return players
}

// boundFields names the fields a set of rules produces.
func boundFields(rules []recipe.Rule) map[string]bool {
	bound := make(map[string]bool, len(rules))
	for _, rule := range rules {
		bound[rule.Field] = true
	}
	return bound
}

// chained refuses a rule that has to agree with a field another rule binds.
//
// It is asked twice, by the plan and by the run, and the sentence is written
// once. Until 2026-10-21 only the run asked, so validate called such a recipe
// usable and generate then refused it on the first row.
//
// Only a bound field playing a role is refused, because only its value is
// read. A rule naming a first name as well as a gender reads the gender alone,
// and refusing it because the first name follows the gender would turn a recipe
// that ran into one that does not, for the sake of a value nobody reads.
func chained(entity recipe.Entity, rule recipe.Rule, bound map[string]bool, values Values) error {
	for _, with := range rule.With {
		field, known := entity.Field(with)
		if !known || !bound[with] {
			continue
		}
		if _, plays := values.RoleOf(field.Generator); plays {
			return fault.New(fault.Config, "entities."+entity.Name+
				".consistency: "+rule.Field+" has to agree with "+with+
				", which is itself bound by a rule, and this build resolves one level")
		}
	}
	return nil
}
