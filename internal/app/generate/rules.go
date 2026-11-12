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
// column, and nobody writes a rule meaning that. Two things keep this from
// being a guess:
//
//   - a role the rule already reads through a column it names is left alone,
//     so a recipe that writes its rules out produces what it produced before,
//   - a role played by two columns binds neither, because which of them a name
//     follows is the recipe's to say, and the plan says so in a note.
//
// A column bound by a rule still plays its role, because a run draws the rules
// of a row in the order inAgreementOrder gives: a state that follows its city
// is drawn before the postal code that follows both (DEC-283).
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
	players := playersOf(entity, values)
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
	players := playersOf(entity, values)
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

// playersOf names the columns playing each role, bound by a rule or not.
func playersOf(entity recipe.Entity, values Values) map[string][]string {
	players := map[string][]string{}
	for _, field := range entity.Fields {
		if role, plays := values.RoleOf(field.Generator); plays {
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

// inAgreementOrder returns the rules of an entity in an order a row can be
// drawn in, or refuses rules that wait on each other in a circle (DEC-283).
//
// # What one rule waits for
//
// A rule waits for every rule binding a column whose value it reads, and it
// reads a column that plays a role its generator takes account of. A column
// named in a rule and read by nobody is no reason to wait: a city told to match
// its state reads nothing, so it cannot close a circle with the state that
// follows it. A second rule for the same field waits for the first, so the
// rule written last is still the one whose value stays, as it was.
//
// # Why the order is otherwise left alone
//
// Rules that wait for nothing keep the order they came in, and each value is
// drawn at an address of its own field and row, so a recipe without a chain
// produces the bytes it produced before chains were resolved.
//
// It is asked twice, by the plan and by the run, and the sentence is written
// once, so that validate refuses what generate refuses.
func inAgreementOrder(entity recipe.Entity, rules []recipe.Rule, values Values) ([]recipe.Rule, error) {
	waits := make([][]int, len(rules))
	for at, rule := range rules {
		waits[at] = waitsFor(entity, rules, at, rule, values)
	}
	placed := make([]bool, len(rules))
	ordered := make([]recipe.Rule, 0, len(rules))
	for len(ordered) < len(rules) {
		moved := false
		for at, rule := range rules {
			if placed[at] || !allPlaced(waits[at], placed) {
				continue
			}
			placed[at], moved = true, true
			ordered = append(ordered, rule)
		}
		if !moved {
			return nil, inACircle(entity, rules, placed)
		}
	}
	return ordered, nil
}

// waitsFor names the rules the one at that place has to be drawn after.
func waitsFor(entity recipe.Entity, rules []recipe.Rule, at int, rule recipe.Rule, values Values) []int {
	var waits []int
	for earlier := 0; earlier < at; earlier++ {
		if rules[earlier].Field == rule.Field {
			waits = append(waits, earlier)
		}
	}
	subject, known := entity.Field(rule.Field)
	if !known || subject.Generator == "" {
		return waits
	}
	for _, with := range rule.With {
		if !readsValueOf(entity, values, subject.Generator, with) {
			continue
		}
		for other, binding := range rules {
			if other != at && binding.Field == with {
				waits = append(waits, other)
			}
		}
	}
	return waits
}

// readsValueOf reports whether a generator takes account of the value of that
// column, which it does when the column plays a role the generator reads.
func readsValueOf(entity recipe.Entity, values Values, generator, column string) bool {
	field, known := entity.Field(column)
	if !known {
		return false
	}
	role, plays := values.RoleOf(field.Generator)
	return plays && reads(values, generator, role)
}

func allPlaced(waits []int, placed []bool) bool {
	for _, at := range waits {
		if !placed[at] {
			return false
		}
	}
	return true
}

// inACircle refuses the rules left over, naming their fields once each in the
// order the recipe gave them. Some of them may only wait on the circle rather
// than close it, and the sentence says so rather than pretending to know which.
func inACircle(entity recipe.Entity, rules []recipe.Rule, placed []bool) error {
	var left []string
	seen := map[string]bool{}
	for at, rule := range rules {
		if !placed[at] && !seen[rule.Field] {
			seen[rule.Field] = true
			left = append(left, rule.Field)
		}
	}
	return fault.New(fault.Config, "entities."+entity.Name+".consistency: the rules of "+
		strings.Join(left, ", ")+" wait on each other in a circle, so no row can draw "+
		"them in any order").DoInstead("remove one of the rules that closes the circle")
}
