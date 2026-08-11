package corrupt

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Selector decides what damage a policy asks for and where it lands.
//
// # Three separate choices, three different mechanisms
//
// Which rows are damaged is drawn from the seed. Which damage they get is taken
// from the catalog in order. Which field it lands on comes from what the entry
// says it applies to. They are separate because they answer to different
// things, and collapsing them would tie the amount of damage to the size of the
// catalog.
//
// # Why the mutator is taken in order rather than drawn
//
// Drawing from fifty two entries across fifty two orders leaves about a third
// of them untouched, which is the coupon collector rather than bad luck.
// Somebody writing layers: [L2, L3] is asking for the line of defence to be
// covered, not sampled, so every described kind of damage gets its entry in the
// manifest before any of them repeats.
//
// # What this deliberately does not do
//
// It never skips an order because this build cannot carry it out. An order
// nobody implemented still becomes an entry saying so. Filtering here would be
// the silence this product exists to find in other people's tools, moved into
// our own.
type Selector struct {
	policy  recipe.Policy
	entries []catalog.Entry
	view    recipe.SchemaView
	source  seed.Source
	share   share
	// taken counts the rows chosen so far, which is what walks the catalog.
	//
	// It is state, and state means this has to be used by one caller in row
	// order. That holds today because a run produces rows in order, and the
	// alternative, deriving the position from the row number alone, would make
	// the walk skip most of the catalog exactly when few rows are damaged.
	taken int
	// unpartnered counts set level orders that could not be built.
	unpartnered int
}

// NewSelector prepares the choices for one run.
//
// The entries are those of the layers the policy named, in identifier order, so
// that the walk is the same on every machine. A layer with no entries
// contributes nothing here and is refused earlier, by validate.
func NewSelector(policy recipe.Policy, described catalog.Catalog, view recipe.SchemaView,
	source seed.Source) (Selector, error) {
	portion, err := shareOf(policy.Rows)
	if err != nil {
		return Selector{}, err
	}
	var entries []catalog.Entry
	for _, layer := range policy.Layers {
		entries = append(entries, described.InLayer(catalog.Layer(layer))...)
	}
	selector := Selector{policy: policy, entries: entries, view: view, source: source, share: portion}
	if err := selector.somethingToDamage(); err != nil {
		return Selector{}, err
	}
	return selector, nil
}

// somethingToDamage refuses a policy that has excluded everything it could
// touch.
//
// # Why this is a failure at the start rather than an entry per order
//
// A policy whose exclusions leave an entity with no fields orders damage that
// can never land, and saying so once is worth more than saying it fifty two
// times in a manifest. It is also the shape of a typo: a name in
// exclude_fields that matches nothing looks identical to one that matches
// everything, and only the arithmetic tells them apart.
func (s *Selector) somethingToDamage() error {
	for _, entity := range s.view.EntityNames() {
		fields := s.view.FieldsOf(entity)
		if len(fields) == 0 {
			continue
		}
		for _, field := range fields {
			if !s.excluded(field.Name) {
				return nil
			}
		}
		return fault.New(fault.Config, fmt.Sprintf(
			"policy.exclude_fields leaves nothing to damage in %s, because it names every "+
				"field the entity has. Remove a name from the list, or drop the policy", entity))
	}
	return nil
}

// Orders returns the damage asked for in one row, in the order it will be
// applied.
//
// An empty answer means this row was not chosen, which is the ordinary case for
// most rows of most runs.
func (s *Selector) Orders(entity string, row, ofRows int) []Order {
	if len(s.entries) == 0 || !s.chosen(entity, row) {
		return nil
	}
	position := s.taken
	s.taken++

	count := 1
	if s.policy.PerRow == perRowAny {
		count = perRowManyCount
	}
	orders := make([]Order, 0, count)
	for i := 0; i < count && i < len(s.entries); i++ {
		entry := s.entries[(position+i)%len(s.entries)]
		partner, spans := s.partnerOf(entry, row, ofRows)
		if !spans {
			// A set level defect with nobody to span. Counted rather than
			// dropped, and said once at the end of the run.
			s.unpartnered++
			continue
		}
		orders = append(orders, Order{
			Mutator: entry.ID,
			Entity:  entity,
			// The manifest counts rows from one, because a person reading it
			// counts from one.
			Row:     row + 1,
			Field:   s.fieldFor(entry, entity),
			ID:      identify(entity, row+1, entry.ID),
			Partner: partner,
		})
	}
	return orders
}

// partnerOf returns the other row a set level defect spans, and whether the
// order can exist at all.
//
// The partner is the next row, cyclically, so it is the same on every machine
// and needs no knowledge of the data. A table of one row has nobody to span, and
// there the order cannot be built: an entry naming one place for a defect about
// the integrity of a set would be a promise nobody can check.
func (s *Selector) partnerOf(entry catalog.Entry, row, ofRows int) (int, bool) {
	if entry.Layer != catalog.SetLevel {
		return 0, true
	}
	if ofRows < 2 {
		return 0, false
	}
	return (row+1)%ofRows + 1, true
}

// Unpartnered is how many set level orders could not be built because the table
// held a single row.
//
// It is reported at the end of a run rather than left out. Damage that was asked
// for and did not happen is exactly what this product refuses to pass over in
// silence, and a table of one row is a legal thing to ask for.
func (s *Selector) Unpartnered() int { return s.unpartnered }

// chosen says whether this row is damaged.
//
// The draw has its own address, so changing what a field draws from does not
// move which rows are damaged. That is the same reason the share of empty values
// is drawn at its own address in the generator.
func (s *Selector) chosen(entity string, row int) bool {
	if s.share.all {
		return true
	}
	const parts = 10000
	drawn := s.source.IntN(seed.Coordinate{
		Entity: entity, Row: int64(row), Purpose: "corrupt:row",
	}, parts)
	return float64(drawn) < s.share.percent*parts/100
}

// fieldFor picks the column an entry is aimed at, and answers empty when the
// entry damages something other than a field.
//
// Eight of the fifty two entries declare that they apply to no field type at
// all: four at L0, three at L1 and one at L5. Those damage the bytes, the
// container or the set, so naming a column for them would put a location in the
// manifest that the specification says such a layer does not carry.
func (s *Selector) fieldFor(entry catalog.Entry, entity string) string {
	if !aimsAtAField(entry) {
		return ""
	}
	covered, typed, available := s.candidates(entry, entity)
	if len(covered) > 0 {
		// Drawn rather than taken in order, so that two entries aimed at the
		// same kind of field do not always land on the same column.
		at := seed.Coordinate{Entity: entity, Field: string(entry.ID), Purpose: "corrupt:field"}
		return covered[s.source.IntN(at, int64(len(covered)))]
	}
	// 🔴 No column this entry covers, and the order still has to name one.
	//
	// The obvious answer, leaving the location without a field, produces an
	// entry the manifest model refuses: a defect in a value carries an entity, a
	// row and a field. Found by running this, not by reading it.
	//
	// So the order points at a column it could have damaged and the engine
	// refuses it there, in a sentence naming what differs. The nearest miss goes
	// first, because that sentence is only useful when it names the real
	// difference: a dataset holding one national identifier and damage meant for
	// another has a column of exactly the right type, and pointing somewhere
	// else would have the manifest say the type is wrong when it is not.
	// Dropping the order instead would be the silence this product exists to
	// find.
	for _, nearest := range [][]string{typed, available} {
		if len(nearest) > 0 {
			return nearest[0]
		}
	}
	return ""
}

// candidates sorts the columns of an entity by how well they suit an entry:
// those it covers outright, those of the right type following the wrong scheme,
// and the rest. Columns the policy excluded are in none of them.
//
// Both halves of what the entry covers are asked, because either alone picks
// wrongly. A dataset carrying a national identifier and a social security number
// has two columns of type identifier, and damage aimed at a check digit is
// meaningful in one of them: asked about the type only, this would draw between
// them and refuse half the time while a column it could have damaged sat beside
// it.
func (s *Selector) candidates(entry catalog.Entry, entity string) (covered, typed, available []string) {
	for _, field := range s.view.FieldsOf(entity) {
		if s.excluded(field.Name) {
			continue
		}
		available = append(available, field.Name)
		if !entry.CoversFieldType(field.Type) {
			continue
		}
		typed = append(typed, field.Name)
		// The column has to suit the entry in all three ways: its type, its
		// numbering scheme, and what it declares about itself. The third is
		// what stops damage about a maximum length landing on a column that
		// declares none while another column in the same recipe does.
		if entry.CoversProvider(field.Provider) && len(columnCannot(entry.Requires, field)) == 0 {
			covered = append(covered, field.Name)
		}
	}
	return covered, typed, available
}

// excluded reports whether the policy told this column to be left alone.
func (s *Selector) excluded(field string) bool {
	for _, name := range s.policy.ExcludeFields {
		if name == field {
			return true
		}
	}
	return false
}

// aimsAtAField reports whether an entry damages a field at all.
func aimsAtAField(entry catalog.Entry) bool {
	for _, declared := range entry.AppliesTo.FieldTypes {
		if declared == catalog.None {
			return false
		}
	}
	return len(entry.AppliesTo.FieldTypes) > 0
}
