package generate

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// uniqueTries is how many times a unique field is drawn again before the run
// gives up.
//
// # Why a limit rather than drawing until it works
//
// A field asked to be unique across more rows than it has possible values can
// never be satisfied, and a loop looking for the impossible looks exactly like
// a slow run. The limit turns that into a named failure with the arithmetic in
// it, which is the difference between a tool a person can act on and one they
// have to kill.
//
// Twenty is high enough that an ordinary collision costs nothing and low enough
// that a hopeless field is reported in the same second.
const uniqueTries = 20

// Run produces the dataset described by a plan and hands it to a writer.
//
// # Why the plan comes in rather than being worked out here
//
// The two phases answer different questions and only the second one touches the
// world. Keeping them apart is what makes a dry run the same arithmetic as a
// real run rather than an estimate that can drift away from it.
//
// # Where it can be stopped
//
// Between tables and between rows, and nowhere else. Those are the two places
// where the work is divisible, and a run that stops at one of them has produced
// nothing that anybody has seen: the bytes are still under a temporary name and
// the caller throws them away. Asking more often would cost a check inside the
// arithmetic of a single value and buy nothing, because there is no half a row.
func Run(plan Plan, rec recipe.Recipe, values Values, writer Writer, stop Halt) error {
	source := seed.New(rec.Seed)
	for _, planned := range plan.Entities {
		if ended := stop.Stopped(); ended != nil {
			return ended
		}
		entity, known := rec.Entity(planned.Entity)
		if !known {
			return fault.New(fault.Runtime, "the plan names an entity the recipe does not have: "+
				planned.Entity)
		}
		if err := writeTable(planned, entity, values, writer, source, stop); err != nil {
			return err
		}
	}
	return writer.Finish()
}

// writeTable produces one table.
func writeTable(planned dataset.Table, entity recipe.Entity, values Values, writer Writer,
	source seed.Source, stop Halt) error {
	if err := writer.Begin(planned); err != nil {
		return err
	}
	taken := newUniqueSet(entity)
	for row := 0; row < planned.Rows; row++ {
		// A table left without its End is not a problem here: what a stopped run
		// wrote never gets its real name, so an unfinished table is thrown away
		// rather than shipped as a complete one.
		if ended := stop.Stopped(); ended != nil {
			return ended
		}
		built, err := buildRow(entity, row, values, source, taken)
		if err != nil {
			return err
		}
		record := dataset.Record{Values: built, Parent: dataset.NoParent}
		if planned.Parent != "" {
			record.Parent = planned.ParentOf[row]
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.End(planned)
}

// buildRow produces one record.
//
// Fields that agree with nothing come first and fields bound by a consistency
// rule come second, because the second kind is generated from the first. A rule
// pointing at a field that is itself bound is refused rather than resolved in
// whatever order the recipe happened to list them.
func buildRow(entity recipe.Entity, row int, values Values, source seed.Source,
	taken uniqueSet) ([]string, error) {
	built := make(map[string]string, len(entity.Fields))
	bound := boundFields(entity)

	for _, field := range entity.Fields {
		if bound[field.Name] {
			continue
		}
		value, err := oneValue(entity, field, row, values, source, taken, provider.Agreement{})
		if err != nil {
			return nil, err
		}
		built[field.Name] = value
	}

	for _, rule := range entity.Consistency {
		field, known := entity.Field(rule.Field)
		if !known {
			continue
		}
		agreement, err := agreementFor(rule, entity, built, bound, values)
		if err != nil {
			return nil, err
		}
		value, err := oneValue(entity, field, row, values, source, taken, agreement)
		if err != nil {
			return nil, err
		}
		built[field.Name] = value
	}

	ordered := make([]string, 0, len(entity.Fields))
	for _, field := range entity.Fields {
		ordered = append(ordered, built[field.Name])
	}
	return ordered, nil
}

// agreementFor gathers what a bound field has to agree with, keyed by role.
func agreementFor(rule recipe.Rule, entity recipe.Entity, built map[string]string,
	bound map[string]bool, values Values) (provider.Agreement, error) {
	roles := make(map[string]string, len(rule.With))
	for _, with := range rule.With {
		if bound[with] {
			return provider.Agreement{}, fault.New(fault.Config, "entities."+entity.Name+
				".consistency: "+rule.Field+" has to agree with "+with+
				", which is itself bound by a rule, and this build resolves one level")
		}
		field, known := entity.Field(with)
		if !known {
			continue
		}
		role, plays := values.RoleOf(field.Generator)
		if !plays {
			continue
		}
		roles[role] = built[with]
	}
	return provider.NewAgreement(roles), nil
}

// oneValue produces the value of one field, honouring emptiness and uniqueness.
func oneValue(entity recipe.Entity, field recipe.Field, row int, values Values,
	source seed.Source, taken uniqueSet, agrees provider.Agreement) (string, error) {
	at := seed.Coordinate{Entity: entity.Name, Row: int64(row), Field: field.Name, Purpose: "value"}
	if leftEmpty(field, at, source) {
		return "", nil
	}
	if field.Generator == "" && !values.ProducesType(field.Type) {
		// An expression, or a type this build cannot fill. Both are named in
		// the plan, so the empty column is stated rather than silent.
		return "", nil
	}

	request := Request{
		Generator: field.Generator, Type: field.Type, At: at, Agrees: agrees,
		MaxLength: field.MaxLength, HasMaxLength: field.HasMaxLength,
		Unit: string(field.Unit()),
	}
	if !field.Unique {
		return values.Produce(request)
	}
	return uniqueValue(entity, field, request, values, taken)
}

// uniqueValue draws again until the value is one no earlier row took.
//
// # What this costs and why it is still the right trade
//
// Drawing again makes the value of a row depend on the rows before it, so a
// unique field cannot be produced in parallel. The alternative, numbering
// instead of drawing, would make every row depend on every earlier one for the
// value itself rather than only for the retry, and that breaks the promise that
// adding a field leaves the other columns where they were.
func uniqueValue(entity recipe.Entity, field recipe.Field, request Request, values Values,
	taken uniqueSet) (string, error) {
	for try := 0; try < uniqueTries; try++ {
		attempt := request
		if try > 0 {
			attempt.At.Purpose = fmt.Sprintf("value:unique#%d", try)
		}
		value, err := values.Produce(attempt)
		if err != nil {
			return "", err
		}
		if taken.claim(field.Name, value) {
			return value, nil
		}
	}
	return "", fault.New(fault.Data, fmt.Sprintf(
		"entities.%s.fields.%s is unique and %d draws in a row landed on a value already used. "+
			"The field has fewer possible values than the %d rows asked for. Lower the count, "+
			"drop the uniqueness, or widen what the field draws from",
		entity.Name, field.Name, uniqueTries, entity.Count))
}

// leftEmpty decides whether this row leaves the field blank.
//
// The share is drawn at its own address rather than from the value itself, so
// that changing what a field draws from does not move which rows are empty.
func leftEmpty(field recipe.Field, at seed.Coordinate, source seed.Source) bool {
	if !field.HasEmptyShare || field.EmptyShare <= 0 || field.Required {
		return false
	}
	const parts = 10000
	drawn := source.IntN(seed.Coordinate{
		Entity: at.Entity, Row: at.Row, Field: at.Field, Purpose: "empty",
	}, parts)
	return float64(drawn) < field.EmptyShare*parts/100
}

// boundFields names the fields a consistency rule produces.
func boundFields(entity recipe.Entity) map[string]bool {
	bound := make(map[string]bool, len(entity.Consistency))
	for _, rule := range entity.Consistency {
		bound[rule.Field] = true
	}
	return bound
}

// uniqueSet remembers what a unique field has already produced.
type uniqueSet map[string]map[string]bool

func newUniqueSet(entity recipe.Entity) uniqueSet {
	set := make(uniqueSet)
	for _, field := range entity.Fields {
		if field.Unique {
			set[field.Name] = make(map[string]bool)
		}
	}
	return set
}

// claim records a value and reports whether it was free.
func (s uniqueSet) claim(field, value string) bool {
	seen, tracked := s[field]
	if !tracked {
		return true
	}
	if seen[value] {
		return false
	}
	seen[value] = true
	return true
}
