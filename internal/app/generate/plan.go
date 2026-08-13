package generate

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Note is something this run did not do, together with why.
//
// It exists because a run that did less than it promised has to say so, and
// because the difference between a failure and a note is whether the dataset is
// still what the recipe asked for. A consistency rule nothing can honour does
// not stop the run, and hiding it would produce exactly the silent disagreement
// this tool sells as a product.
type Note struct {
	// Where names the place in the recipe, such as entities.customer.pesel.
	Where string
	// What was not done.
	What string
}

// Plan is what a run will produce, worked out without touching a disk.
type Plan struct {
	// Entities are the tables in the order they have to be produced, parents
	// before children.
	Entities []dataset.Table
	// Notes are the things this run will not do.
	Notes []Note
	// Seed, Locale and Tag are carried so that the run note can state what the
	// dataset was made from without reading the recipe again.
	Seed   string
	Locale string
	Tag    string
}

// TotalRows is how many records the whole plan holds.
//
// It can add without checking because Prepare refused any recipe whose rows do
// not fit, so a plan that exists is one whose total can be counted. That is the
// invariant this method rests on, and the refusal below is what holds it up.
func (p Plan) TotalRows() int {
	total := 0
	for _, entity := range p.Entities {
		total += entity.Rows
	}
	return total
}

// fitsWithTheRest reports whether one more table can be added to a total.
//
// # Why a sum, when the ceiling is on a single count
//
// The recipe reader already refuses a count larger than this build can hold, so
// each entity fits on its own. Two of them did not have to. Measured on
// 2026-08-13: two entities of 9223372036854775807 rows each ended the run with
// exit code nought and a summary reading minus two rows written. The guard
// protected a field rather than the arithmetic, so the arithmetic wrapped one
// entity later.
func fitsWithTheRest(total, rows int) bool {
	return rows <= 0 || total <= maxInt-rows
}

// maxInt is the largest value this build can count to. Written from the type
// rather than typed out, because the number differs between builds and a
// literal here would be right on one machine and wrong on another.
const maxInt = int(^uint(0) >> 1)

// Prepare works out what a run will produce and refuses what it cannot.
//
// It touches no disk and reads no clock. Everything it decides is a function of
// the recipe and the seed, which is what makes a dry run the same answer as a
// real one.
func Prepare(rec recipe.Recipe, values Values) (Plan, error) {
	ordered, err := inDependencyOrder(rec)
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{Seed: rec.Seed, Locale: rec.Locale, Tag: rec.Tag}
	source := seed.New(rec.Seed)
	produced := make(map[string]dataset.Table, len(ordered))

	total := 0
	for _, entity := range ordered {
		if err := checkGenerators(entity, values, &plan); err != nil {
			return Plan{}, err
		}
		planned, err := planEntity(entity, produced, source)
		if err != nil {
			return Plan{}, err
		}
		if !fitsWithTheRest(total, planned.Rows) {
			return Plan{}, tooManyRows(planned.Entity)
		}
		total += planned.Rows
		plan.Notes = append(plan.Notes, notesFor(entity, values)...)
		produced[entity.Name] = planned
		plan.Entities = append(plan.Entities, planned)
	}
	return plan, nil
}

// tooManyRows refuses a recipe whose tables add up to more rows than this build
// can count, in the same words the reader uses for a single count that does not
// fit.
func tooManyRows(entity string) error {
	return fault.New(fault.Config, fmt.Sprintf(
		"the rows of every table in this recipe add up to more than the %d this build can "+
			"count, and %s is the table that goes past it. A count is not refused on its own "+
			"here, so this is about the total. Lower the counts, or split the recipe into "+
			"runs that each ask for fewer rows", maxInt, entity))
}

// planEntity works out the row count of one entity and, when it has a parent,
// which parent each of its rows belongs to.
func planEntity(entity recipe.Entity, done map[string]dataset.Table, source seed.Source) (dataset.Table, error) {
	planned := dataset.Table{
		Entity:  entity.Name,
		Columns: entity.FieldNames(),
		Rows:    entity.Count,
	}
	if len(entity.Parents) == 0 {
		return planned, nil
	}

	parent := entity.Parents[0]
	owner, known := done[parent.Entity]
	if !known {
		return dataset.Table{}, fault.New(fault.Config, entity.Name+" belongs to "+parent.Entity+
			", which this run does not produce")
	}

	shares, err := shareOut(entity, parent, owner.Rows, source)
	if err != nil {
		return dataset.Table{}, err
	}
	planned.Parent = parent.Entity
	planned.ParentOf = expand(shares)
	planned.Rows = len(planned.ParentOf)
	return planned, nil
}

// shareOut decides how many children each parent gets.
//
// # Which number wins, and it is the recipe's own count
//
// A child that states a count gets exactly that many rows and the cardinality
// becomes a constraint on how they are spread. A child that states none gets
// whatever the draws add up to. A count that cannot be reached inside the
// bounds is refused with both bounds and the wanted number in the message,
// rather than quietly trimmed.
func shareOut(entity recipe.Entity, parent recipe.Parent, parents int, source seed.Source) ([]int, error) {
	if parents == 0 {
		return nil, nil
	}
	shares := make([]int, parents)
	span := int64(parent.Max - parent.Min + 1)
	for index := range shares {
		shares[index] = parent.Min + int(source.IntN(seed.Coordinate{
			Entity: entity.Name, Row: int64(index), Purpose: ":children",
		}, span))
	}
	if !entity.HasCount {
		return shares, nil
	}
	if err := reachable(entity, parent, parents); err != nil {
		return nil, err
	}
	return repair(shares, entity, parent, source), nil
}

// reachable refuses a count the bounds cannot produce, showing the arithmetic.
func reachable(entity recipe.Entity, parent recipe.Parent, parents int) error {
	lowest, highest := parents*parent.Min, parents*parent.Max
	if entity.Count >= lowest && entity.Count <= highest {
		return nil
	}
	return fault.New(fault.Data, fmt.Sprintf(
		"%s asks for %d rows and belongs to %s, which has %d rows at %d to %d each, "+
			"so the count has to be between %d and %d. Change the count or the range",
		entity.Name, entity.Count, parent.Entity, parents, parent.Min, parent.Max, lowest, highest))
}

// repair moves the drawn shares until they add up to the wanted count.
//
// The order it walks is drawn rather than the order of the rows, so that the
// adjustment does not always land on the first parents. It is still a function
// of the seed alone, which is what the whole tool rests on.
func repair(shares []int, entity recipe.Entity, parent recipe.Parent, source seed.Source) []int {
	order := drawnOrder(entity.Name, len(shares), source)
	total := 0
	for _, share := range shares {
		total += share
	}
	for step := 0; total != entity.Count; step++ {
		index := order[step%len(order)]
		switch {
		case total < entity.Count && shares[index] < parent.Max:
			shares[index]++
			total++
		case total > entity.Count && shares[index] > parent.Min:
			shares[index]--
			total--
		}
	}
	return shares
}

// drawnOrder returns the parent indices in an order derived from the seed.
func drawnOrder(entity string, parents int, source seed.Source) []int {
	keys := make([]int64, parents)
	order := make([]int, parents)
	for index := range order {
		order[index] = index
		keys[index] = source.IntN(seed.Coordinate{
			Entity: entity, Row: int64(index), Purpose: ":share-order",
		}, 1<<31)
	}
	sort.SliceStable(order, func(left, right int) bool {
		if keys[order[left]] != keys[order[right]] {
			return keys[order[left]] < keys[order[right]]
		}
		return order[left] < order[right]
	})
	return order
}

// expand turns a count per parent into a parent index per child row.
func expand(shares []int) []int {
	total := 0
	for _, share := range shares {
		total += share
	}
	parentOf := make([]int, 0, total)
	for parent, share := range shares {
		for taken := 0; taken < share; taken++ {
			parentOf = append(parentOf, parent)
		}
	}
	return parentOf
}

// checkGenerators refuses a field this build cannot fill, naming what it does
// have.
//
// A generator nobody implements is a recipe that cannot be run, so it is a
// configuration failure rather than a note. The validator already reports such
// a name as unchecked, and this is the half that can finally tell a typo from a
// setting we simply do not read.
func checkGenerators(entity recipe.Entity, values Values, plan *Plan) error {
	for _, field := range entity.Fields {
		where := "entities." + entity.Name + ".fields." + field.Name
		switch {
		case field.Expression != "":
			plan.Notes = append(plan.Notes, Note{where,
				"expressions are not evaluated by this build, so the column will be empty"})
		case field.Generator == "" && !values.ProducesType(field.Type):
			// A type this build cannot fill. Until 2026-08-10 this case said
			// nothing at all, so a declared column arrived empty and no part of
			// the run mentioned it. The expression above has been reported
			// since the first day and this one had not, which is why it was
			// invisible: the two look alike in the recipe and only one of them
			// spoke.
			plan.Notes = append(plan.Notes, Note{where,
				"this build produces no value for a field declared only as " +
					strconv.Quote(field.Type) + ", so the column will be empty"})
		case field.Generator == "":
			continue
		case !values.Known(field.Generator):
			return fault.New(fault.Config, where+": this build has no generator called "+
				strconv.Quote(field.Generator)+". It has "+join(values.Names()))
		}
	}
	return nil
}

// notesFor reports the consistency rules this run cannot honour.
//
// A rule binding a field to one the identifier cannot carry is a declaration
// with nothing in it. The provider sees only its own side and would ignore it,
// which is the one place in this code where something would happen quietly.
func notesFor(entity recipe.Entity, values Values) []Note {
	var notes []Note
	for _, rule := range entity.Consistency {
		where := "entities." + entity.Name + ".consistency." + rule.Field
		for _, with := range rule.With {
			field, known := entity.Field(with)
			if !known {
				continue
			}
			if _, plays := values.RoleOf(field.Generator); !plays {
				notes = append(notes, Note{where, rule.Field + " is asked to agree with " + with +
					", and " + with + " plays no role an identifier can read, so the two are " +
					"generated independently"})
			}
		}
	}
	return notes
}

// inDependencyOrder returns the entities with parents before children.
func inDependencyOrder(rec recipe.Recipe) ([]recipe.Entity, error) {
	if err := refuseSecondParent(rec); err != nil {
		return nil, err
	}
	placed := make(map[string]bool, len(rec.Entities))
	ordered := make([]recipe.Entity, 0, len(rec.Entities))

	for len(ordered) < len(rec.Entities) {
		moved := false
		for _, entity := range rec.Entities {
			if placed[entity.Name] || !parentsPlaced(entity, placed) {
				continue
			}
			placed[entity.Name] = true
			ordered = append(ordered, entity)
			moved = true
		}
		if !moved {
			return nil, fault.New(fault.Config,
				"the relations form a loop, so no entity can be produced before the one it belongs to")
		}
	}
	return ordered, nil
}

func parentsPlaced(entity recipe.Entity, placed map[string]bool) bool {
	for _, parent := range entity.Parents {
		if !placed[parent.Entity] {
			return false
		}
	}
	return true
}

// refuseSecondParent says out loud what this build does not do.
//
// The model carries several parents because a record can have several in a real
// database. Spreading one child across two independent cardinalities is a
// separate question and answering it by accident would freeze the wrong answer.
func refuseSecondParent(rec recipe.Recipe) error {
	for _, entity := range rec.Entities {
		if len(entity.Parents) > 1 {
			return fault.New(fault.Config, entity.Name+
				" belongs to more than one entity, and this build produces one relation per entity")
		}
	}
	return nil
}

func join(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	joined := names[0]
	for _, name := range names[1:] {
		joined += ", " + name
	}
	return joined
}
