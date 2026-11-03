package generate

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// uniqueTries is how many times a unique field is drawn again, and how many
// times a row is built again for its key, before the run stops drawing.
//
// # Why a limit rather than drawing until it works
//
// A field asked to be unique across more rows than it has possible values can
// never be satisfied, and a loop looking for the impossible looks exactly like
// a slow run. The limit turns that into a named outcome. Where the values are
// known the run then walks to a free one, and where they are not it says what
// happened, which is the difference between a tool a person can act on and one
// they have to kill.
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
	table := newTableRun(planned, entity, values, source)
	for row := 0; row < planned.Rows; row++ {
		// A table left without its End is not a problem here: what a stopped run
		// wrote never gets its real name, so an unfinished table is thrown away
		// rather than shipped as a complete one.
		if ended := stop.Stopped(); ended != nil {
			return ended
		}
		built, err := table.row(row)
		if err != nil {
			return err
		}
		record := dataset.Record{Values: built, Parent: dataset.NoParent}
		if planned.Parent != "" {
			record.Parent = planned.Ownership.ParentOf(row)
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return writer.End(planned)
}

// tableRun is what the rows of one table share while they are produced.
type tableRun struct {
	entity recipe.Entity
	rows   int
	// rules are the ones the run keeps, declared and implied together, worked
	// out once per table rather than per row, because the answer depends on
	// the recipe and the build and on nothing a row holds.
	rules  []recipe.Rule
	bound  map[string]bool
	values Values
	source seed.Source
	taken  *uniqueSet
	// key is nil when the run does not hold the primary key, see keyParts.
	key *keySet
	// domains are those of the unique fields whose domain is the same in
	// every row, asked the first time one of their rows walks, see domainOf.
	domains map[string]askedDomain
}

func newTableRun(planned dataset.Table, entity recipe.Entity, values Values, source seed.Source) *tableRun {
	rules := Rules(entity, values)
	table := &tableRun{entity: entity, rows: planned.Rows, rules: rules, bound: boundFields(rules),
		values: values, source: source, taken: newUniqueSet(entity), domains: make(map[string]askedDomain)}
	if parts, held := keyParts(entity, values); held {
		table.key = newKeySet(parts)
	}
	return table
}

// row produces one record, holding the primary key when the run holds it.
//
// # Why the whole row is built again for its key
//
// A key that repeats is a property of the row, not of one value, and the parts
// of a key can be bound to other columns: a PESEL follows the date of birth and
// the gender. Building the row again with the parts of the key drawn at a new
// address, and everything else at its own, gives the same values everywhere
// else and lets every agreement be worked out once more from what changed. What
// unique columns claimed in the attempt is given back first, so the second
// build sees the table as the first one did.
func (t *tableRun) row(row int) ([]string, error) {
	for attempt := 0; attempt < uniqueTries; attempt++ {
		built, err := t.build(row, attempt, nil)
		if err != nil {
			return nil, err
		}
		if t.key == nil || t.key.claim(built) {
			t.taken.keep()
			return t.ordered(built), nil
		}
		t.taken.release()
	}
	forced, err := t.key.walk(t, row)
	if err != nil {
		return nil, err
	}
	built, err := t.build(row, 0, forced)
	if err != nil {
		return nil, err
	}
	t.key.claim(built)
	t.taken.keep()
	return t.ordered(built), nil
}

// build produces the values of one row.
//
// Fields that agree with nothing come first and fields bound by a consistency
// rule come second, because the second kind is generated from the first. A rule
// pointing at a field that is itself bound is refused rather than resolved in
// whatever order the recipe happened to list them.
//
// Forced values are the parts of a key the run chose by walking, and they take
// the place of a draw.
func (t *tableRun) build(row, attempt int, forced map[string]string) (map[string]string, error) {
	built := make(map[string]string, len(t.entity.Fields))
	for _, field := range t.entity.Fields {
		if t.bound[field.Name] {
			continue
		}
		if value, chosen := forced[field.Name]; chosen {
			built[field.Name] = value
			continue
		}
		value, err := t.oneValue(field, row, attempt, provider.Agreement{})
		if err != nil {
			return nil, err
		}
		built[field.Name] = value
	}

	for _, rule := range t.rules {
		field, known := t.entity.Field(rule.Field)
		if !known {
			continue
		}
		agreement, err := agreementFor(rule, t.entity, built, t.bound, t.values)
		if err != nil {
			return nil, err
		}
		value, err := t.oneValue(field, row, attempt, agreement)
		if err != nil {
			return nil, err
		}
		built[field.Name] = value
	}
	return built, nil
}

// ordered lays the values of a row out in the order of the fields.
func (t *tableRun) ordered(built map[string]string) []string {
	ordered := make([]string, 0, len(t.entity.Fields))
	for _, field := range t.entity.Fields {
		ordered = append(ordered, built[field.Name])
	}
	return ordered
}

// agreementFor gathers what a bound field has to agree with, keyed by role.
//
// # Why an empty value binds nothing
//
// A column with a share of empty values is blank in some rows, and a blank
// says nothing about the person. Until 2026-10-21 it was bound all the same,
// so a recipe giving the gender ten per cent of empty values and a PESEL that
// matches it stopped on the first blank row, because no scheme can encode a
// gender of nothing. The value is now left unbound in that row, the way it is
// in a recipe with no gender column at all: the PESEL draws a gender and a
// first name is drawn from both lists.
func agreementFor(rule recipe.Rule, entity recipe.Entity, built map[string]string,
	bound map[string]bool, values Values) (provider.Agreement, error) {
	if err := chained(entity, rule, bound, values); err != nil {
		return provider.Agreement{}, err
	}
	roles := make(map[string]string, len(rule.With))
	for _, with := range rule.With {
		field, known := entity.Field(with)
		if !known || built[with] == "" {
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
//
// A part of the key built again for its key is drawn at an address of its own,
// value:key#1 and on, and every other field at the address it always has.
func (t *tableRun) oneValue(field recipe.Field, row, attempt int, agrees provider.Agreement) (string, error) {
	purpose := "value"
	if attempt > 0 && t.key.holds(field.Name) {
		purpose = fmt.Sprintf("value:key#%d", attempt)
	}
	request := requestFor(t.entity.Name, field, row, purpose)
	request.Agrees = agrees
	if leftEmpty(field, request.At, t.source) {
		return "", nil
	}
	if leftBlank(field, t.values) {
		// An expression, or a type this build cannot fill. Both are named in
		// the plan, so the empty column is stated rather than silent.
		return "", nil
	}
	if !field.Unique {
		return t.values.Produce(request)
	}
	return t.uniqueValue(field, request)
}

// requestFor is what the source is asked for one field, the same question from
// the plan and from the run.
func requestFor(entity string, field recipe.Field, row int, purpose string) Request {
	return Request{
		Generator: field.Generator, Type: field.Type, Values: field.Values,
		At:        seed.Coordinate{Entity: entity, Row: int64(row), Field: field.Name, Purpose: purpose},
		MaxLength: field.MaxLength, HasMaxLength: field.HasMaxLength,
		Unit: string(field.Unit()),
	}
}

// leftBlank reports whether this build leaves the field empty in every row,
// because it is an expression or a type with nothing to fill it.
func leftBlank(field recipe.Field, values Values) bool {
	return field.Expression != "" ||
		(field.Generator == "" && len(field.Values) == 0 && !values.ProducesType(field.Type))
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
func (t *tableRun) uniqueValue(field recipe.Field, request Request) (string, error) {
	base := request.At.Purpose
	for try := 0; try < uniqueTries; try++ {
		attempt := request
		if try > 0 {
			attempt.At.Purpose = fmt.Sprintf("%s:unique#%d", base, try)
		}
		value, err := t.values.Produce(attempt)
		if err != nil {
			return "", err
		}
		if t.taken.claim(field.Name, value) {
			return value, nil
		}
	}
	return t.walkUnique(field, request)
}

// walkUnique takes the next free value of a known domain, from a place the
// seed draws, once drawing has run out of luck.
func (t *tableRun) walkUnique(field recipe.Field, request Request) (string, error) {
	where := "entities." + t.entity.Name + ".fields." + field.Name
	domain, known := t.domainOf(field, request)
	if !known {
		return "", fault.New(fault.Data, fmt.Sprintf(
			"%s is unique and %d draws in a row landed on a value already used, at row %d of %d. "+
				"This build does not count how many values the field can take, so it cannot tell "+
				"whether any are left", where, uniqueTries, request.At.Row+1, t.rows)).
			DoInstead("lower the count, drop unique, or draw the field from a list under values")
	}
	walker := t.taken.walkerFor(field.Name, domain)
	start := t.drawnPlace(request.At, request.At.Purpose+":unique#walk", domain.Size)
	place, found := walker.next(start, func(at int64) bool {
		return t.taken.free(field.Name, domain.At(at))
	})
	if !found {
		return "", fault.New(fault.Data, fmt.Sprintf(
			"%s is unique and every one of the %d values it can take in this row is already used, "+
				"at row %d of %d", where, domain.Size, request.At.Row+1, t.rows)).
			DoInstead("lower the count, drop unique, or widen what the field draws from")
	}
	value := domain.At(place)
	t.taken.claim(field.Name, value)
	return value, nil
}

// drawnPlace is where a walk starts, drawn at an address of its own.
func (t *tableRun) drawnPlace(at seed.Coordinate, purpose string, size int64) int64 {
	if size <= 0 {
		return 0
	}
	return t.source.IntN(seed.Coordinate{Entity: at.Entity, Row: at.Row, Field: at.Field,
		Purpose: purpose}, size)
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
