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

// MaxChildRows is how many rows this build can plan for an entity that belongs
// to another one, and how many parents it can share them out between.
//
// # Why there is a limit here when there is deliberately none on a count
//
// DEC-116 refused an upper limit on the number of rows a recipe may ask for,
// and the reason it gave is the reason this exists rather than an argument
// against it: validate answers whether a recipe is correct, and whether a run
// can be carried out is the engine's question, answered with the exit code for
// data. This is the engine answering it.
//
// It is also not the same quantity. An entity with no parent is streamed, so a
// count of a billion costs a run its time and nothing else. An entity that
// belongs to another is planned first: the plan says which parent each child
// row belongs to, one machine word per row, before a byte is written.
//
// # Why four thousand million
//
// Measured on 2026-09-02, on a dry run of this build, which plans and writes
// nothing: 8.4 bytes a row at a million rows and 7.8 at five million, which is
// the one word this holds and nothing else. At this limit the index alone is
// 32 GB before a single value is drawn, which is more memory than the machines
// this is built for have. A run that reached it was never going to finish, so
// what this changes is that it is told so with the arithmetic instead of being
// stopped by the allocator, or by the operating system.
//
// The full run costs about seventy bytes a row on top, measured the same day
// at five million rows, so the ceiling is generous by roughly a factor of ten
// against what a machine could actually write.
//
// # What it prevents besides an unfinishable run
//
// Two panics, both reachable from a recipe and both reported as defects of
// this tool until this existed. A range of nine million million million
// children overflowed the arithmetic behind the draw and reached the seed as a
// negative bound. A range of four thousand million million asked the runtime
// for a slice it refuses to make. Neither is a mistake in the tool, and both
// said it was.
const MaxChildRows = 4_000_000_000

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
	// Implied are the agreements this run keeps that no rule in the recipe
	// wrote, so that whoever reads the plan is told which columns follow which
	// rather than finding out from the data.
	Implied []Implied
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
// 2026-08-17: two entities of 9223372036854775807 rows each ended the run with
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
		if err := checkGenerators(entity, values, rec.Locale, &plan); err != nil {
			return Plan{}, err
		}
		if err := refuseChains(entity, values); err != nil {
			return Plan{}, err
		}
		planned, err := planEntity(entity, produced, source)
		if err != nil {
			return Plan{}, err
		}
		if !fitsWithTheRest(total, planned.Rows) {
			return Plan{}, tooManyRows(planned.Entity)
		}
		if err := checkDomains(entity, planned.Rows, values); err != nil {
			return Plan{}, err
		}
		total += planned.Rows
		plan.Notes = append(plan.Notes, notesFor(entity, values)...)
		plan.Notes = append(plan.Notes, unsure(entity, values)...)
		for _, rule := range implied(entity, values) {
			plan.Implied = append(plan.Implied, Implied{Entity: entity.Name, Rule: rule})
		}
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
	// 🔴 Refused in the first phase rather than by the writer that trips on it,
	// because a plan says what will be produced and the column carrying the
	// relation is part of that. Left to the writer, a dry run counted the rows
	// of a dataset the real run refuses to write, which is the same disagreement
	// between the two phases that B-205 is about between the two verbs.
	if name, collides := entity.ParentColumnCollision(); collides {
		return dataset.Table{}, fault.New(fault.Config, "entities."+entity.Name+" has a field "+
			"called "+name+" and also belongs to an entity of that name, so the relation and "+
			"the field would need the same column").
			DoInstead("rename the field")
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
	if err := withinReach(entity, parent, parents); err != nil {
		return nil, err
	}
	shares := make([]int, parents)
	// Both sides are widened before the subtraction rather than after it, which
	// is the whole of the difference between this line and the one it replaced.
	// It is defence in depth now that withinReach has bounded the range, in the
	// same spirit as take.CheckOutput, which stays although the suffix makes it
	// unreachable: the guard above answers a question about the recipe and this
	// answers one about the arithmetic, and the day either moves the other is
	// still true.
	span := int64(parent.Max) - int64(parent.Min) + 1
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

// withinReach refuses a relation this build cannot hold a plan for, showing
// the arithmetic that says so.
//
// # Why the two questions are asked apart
//
// The parents are counted on their own because a share is held for each of
// them while the draw goes on, so a parent table too large to index makes the
// relation impossible whatever its range says. The product is then compared by
// dividing rather than by multiplying, because multiplying is the overflow
// this is here to refuse.
func withinReach(entity recipe.Entity, parent recipe.Parent, parents int) error {
	if int64(parents) > MaxChildRows {
		return fault.New(fault.Data, fmt.Sprintf(
			"%s belongs to %s, which has %d rows, and this build shares children out "+
				"between at most %d parents. Lower the number of rows in %s",
			entity.Name, parent.Entity, parents, MaxChildRows, parent.Entity))
	}
	most := int64(parent.Max)
	if most > 0 && int64(parents) > MaxChildRows/most {
		return fault.New(fault.Data, fmt.Sprintf(
			"%s belongs to %s at up to %d children each, and %s has %d rows, so this asks "+
				"for more than the %d rows this build can plan for. It holds one machine "+
				"word for every child row before it writes anything. Lower the range or "+
				"the number of parents",
			entity.Name, parent.Entity, parent.Max, parent.Entity, parents,
			MaxChildRows))
	}
	return nil
}

// reachable refuses a count the bounds cannot produce, showing the arithmetic.
func reachable(entity recipe.Entity, parent recipe.Parent, parents int) error {
	lowest, highest := parents*parent.Min, parents*parent.Max
	if entity.Count >= lowest && entity.Count <= highest {
		return nil
	}
	return fault.New(fault.Data, fmt.Sprintf(
		"%s asks for %d rows and belongs to %s, which has %d rows at %d to %d each, "+
			"so the count has to be between %d and %d",
		entity.Name, entity.Count, parent.Entity, parents, parent.Min, parent.Max,
		lowest, highest)).
		DoInstead("change the count, or the range of the relation")
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
func checkGenerators(entity recipe.Entity, values Values, locale string, plan *Plan) error {
	for _, field := range entity.Fields {
		where := "entities." + entity.Name + ".fields." + field.Name
		switch {
		case len(field.Values) > 0:
			if err := checkListed(entity.Name, field, where, values); err != nil {
				return err
			}
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
		case field.Generator == "" && field.HasMaxLength:
			if fine, why := values.TypeFits(field.Type, field.MaxLength); !fine {
				return fault.New(fault.Config, where+": "+why).
					DoInstead("raise max_length, or describe the field with a generator")
			}
		case field.Generator == "":
			continue
		case !values.Known(field.Generator):
			return fault.New(fault.Config, where+": this build has no generator called "+
				strconv.Quote(field.Generator)+". It has "+join(values.Names()))
		default:
			// 🔴 The name exists and the pair may still not. Asking here is
			// what stops a dry run from counting rows that cannot be produced
			// and validate from calling such a recipe usable, both of which it
			// did until 2026-08-26.
			if fine, why := values.CanProduce(field.Generator, locale); !fine {
				return fault.New(fault.Config, where+": "+why)
			}
		}
	}
	return nil
}

// checkListed refuses an entry of a list under values that the field cannot
// hold, naming the entry by its place in the list.
func checkListed(entity string, field recipe.Field, where string, values Values) error {
	request := requestFor(entity, field, 0, "value")
	for index, entry := range field.Values {
		if fine, why := values.Accepts(request, entry); !fine {
			return fault.New(fault.Config, fmt.Sprintf("%s.values[%d]: %s", where, index, why)).
				DoInstead("change the entry, or the type or max_length it has to fit")
		}
	}
	return nil
}

// checkDomains refuses a field whose values are known and are too few for the
// rows planned, with the arithmetic, before a row is written.
//
// # Which fields are asked
//
// A field no rule binds, so that its domain is the same in every row. A field
// a rule binds draws from a domain that depends on the row, a name for a woman
// or for a man, and is asked by the run when it gets there. A column this build
// leaves blank is not asked either, because the plan already says it will be
// empty and refusing it would refuse the example the format documents.
//
// A domain with nothing in it is refused in the sentence the run would give on
// the first row, by asking the source for that row the way the run asks. Two
// sentences for one failure would be two places for the plan and the run to
// part company.
func checkDomains(entity recipe.Entity, rows int, values Values) error {
	if rows == 0 {
		return nil
	}
	bound := boundFields(Rules(entity, values))
	for _, field := range entity.Fields {
		if bound[field.Name] || leftBlank(field, values) || alwaysEmpty(field) {
			continue
		}
		request := requestFor(entity.Name, field, 0, "value")
		domain, known := values.DomainOf(request)
		switch {
		case !known:
		case domain.Size == 0:
			if _, err := values.Produce(request); err != nil {
				return err
			}
		case field.Unique && int64(rows) > domain.Size:
			return fault.New(fault.Data, fmt.Sprintf(
				"entities.%s.fields.%s is unique and can take %d different values, and the table has %d rows",
				entity.Name, field.Name, domain.Size, rows)).
				DoInstead("lower the count, drop unique, or widen what the field draws from")
		}
	}
	return checkKey(entity, rows, bound, values)
}

// alwaysEmpty reports whether every row leaves the field blank by its share.
func alwaysEmpty(field recipe.Field) bool {
	return field.HasEmptyShare && field.EmptyShare >= 100 && !field.Required
}

// checkKey refuses a primary key whose parts cannot make as many different
// keys as the table has rows, when every part is free of any rule and its
// values are known.
func checkKey(entity recipe.Entity, rows int, bound map[string]bool, values Values) error {
	parts, held := keyParts(entity, values)
	if !held {
		return nil
	}
	keys := int64(1)
	for _, part := range parts {
		if bound[part.Name] {
			return nil
		}
		domain, known := values.DomainOf(requestFor(entity.Name, part, 0, "value"))
		if !known {
			return nil
		}
		if keys > int64(rows)/max(domain.Size, 1) {
			// Already more keys than rows, and multiplying further could only
			// overflow on the way to the same answer.
			return nil
		}
		keys *= domain.Size
	}
	if keys >= int64(rows) {
		return nil
	}
	return fault.New(fault.Data, fmt.Sprintf(
		"entities.%s.primary_key: the fields of the key can make %d different keys, and the table has %d rows",
		entity.Name, keys, rows)).
		DoInstead("lower the count, add a field to the key, or widen what its fields draw from")
}

// notesFor reports the consistency rules this run cannot honour.
//
// A rule binding a field to one the identifier cannot carry is a declaration
// with nothing in it. The provider sees only its own side and would ignore it,
// which is the one place in this code where something would happen quietly.
//
// 🔴 The same holds from the other side, and until 2026-10-21 only the first
// side was asked. A rule such as nip matches: [gender] binds a column that does
// play a role to a generator that takes no account of it, so the number was
// drawn as if the rule were not there and nothing said so. Measured on the real
// binary that day: validate called it usable and generate wrote it without a
// word, and the same went for a city asked to match a gender.
func notesFor(entity recipe.Entity, values Values) []Note {
	var notes []Note
	for _, rule := range entity.Consistency {
		where := "entities." + entity.Name + ".consistency." + rule.Field
		subject, _ := entity.Field(rule.Field)
		for _, with := range rule.With {
			field, known := entity.Field(with)
			if !known {
				continue
			}
			role, plays := values.RoleOf(field.Generator)
			switch {
			case !plays:
				notes = append(notes, Note{where, rule.Field + " is asked to agree with " + with +
					", and " + with + " plays no role an identifier can read, so the two are " +
					"generated independently"})
			case subject.Generator != "" && !reads(values, subject.Generator, role):
				notes = append(notes, Note{where, rule.Field + " is asked to agree with " + with +
					", and " + subject.Generator + " takes no account of " + role +
					", so the two are generated independently"})
			}
		}
	}
	return notes
}

// reads reports whether a generator takes account of a role.
func reads(values Values, generator, role string) bool {
	for _, read := range values.RolesRead(generator) {
		if read == role {
			return true
		}
	}
	return false
}

// refuseChains refuses a declared rule resting on a column another rule binds,
// in the words the run uses, so that validate refuses what generate would.
func refuseChains(entity recipe.Entity, values Values) error {
	rules := Rules(entity, values)
	bound := boundFields(rules)
	for _, rule := range rules {
		if err := chained(entity, rule, bound, values); err != nil {
			return err
		}
	}
	return nil
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
