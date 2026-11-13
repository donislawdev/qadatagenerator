package corrupt

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

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
// from the catalog in a walk that visits every entry once before any repeats.
// Which field it lands on comes from what the entry says it applies to. They
// are separate because they answer to different things, and collapsing them
// would tie the amount of damage to the size of the catalog.
//
// # Why the mutator is taken in order rather than drawn
//
// Drawing from fifty two entries across fifty two orders leaves about a third
// of them untouched, which is the coupon collector rather than bad luck.
// Somebody writing layers: [L2, L3] is asking for the line of defence to be
// covered, not sampled, so every described kind of damage gets its entry in the
// manifest before any of them repeats.
//
// # Why the walk spreads the layers and starts at a drawn place
//
// Walked in identifier order from the first entry, a file of three rows with
// every layer named gets the first three entries of L0 and nothing else, for
// every seed. So the walk spreads each layer's entries evenly over its length
// (walkOf says how), which lets a few rows touch as many layers as there are
// rows, and the place it starts is drawn from the seed, so another seed reaches
// other entries.
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
	// start is where the walk over the entries begins, drawn from the seed.
	start int
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
// The entries are those of the layers the policy named, each layer in
// identifier order, so that the walk is the same on every machine. A layer with
// no entries contributes nothing here and is refused earlier, by validate.
func NewSelector(policy recipe.Policy, described catalog.Catalog, view recipe.SchemaView,
	source seed.Source) (Selector, error) {
	portion, err := shareOf(policy.Rows)
	if err != nil {
		return Selector{}, err
	}
	layers := make([][]catalog.Entry, 0, len(policy.Layers))
	for _, layer := range policy.Layers {
		layers = append(layers, described.InLayer(catalog.Layer(layer)))
	}
	entries := walkOf(layers)
	selector := Selector{policy: policy, entries: entries, view: view, source: source, share: portion}
	if len(entries) > 0 {
		selector.start = int(source.IntN(seed.Coordinate{Purpose: "corrupt:walk"}, int64(len(entries))))
	}
	if err := selector.somethingToDamage(); err != nil {
		return Selector{}, err
	}
	return selector, nil
}

// walkOf lays the entries of the named layers out in the order the selector
// walks them, spreading each layer evenly over the walk.
//
// Entry j of a layer holding n entries is placed at (j + phase) / n of the way
// along, and the walk follows those places. A layer's entries then stand about
// one n-th of the walk apart wherever it starts, so a few consecutive steps
// meet as many layers as there are steps, until the largest layer crowds them.
// The phase is the layer's position among those named over one more than their
// number, so that no two layers start at the same place. With one phase for
// all, the largest layer held both ends of the walk and the two steps across
// the seam landed in it twice.
//
// Measured 2026-11-13 on the catalog's eight layers of 5, 5, 9, 7, 7, 6, 6 and
// 7 entries: from every one of the 52 starting places, three and four
// consecutive steps meet as many layers, and five steps at least four. Taking
// the layers in turn, round after round, gave three layers from 50 of the 52,
// because only the largest layers are left at the end of the last rounds.
//
// Places are compared as fractions of whole numbers, so the order is exact and
// the same on every machine. A tie goes to the layer named first.
func walkOf(layers [][]catalog.Entry) []catalog.Entry {
	type placed struct {
		entry catalog.Entry
		// The place is over / under, kept as a fraction.
		over, under int64
		layer       int
	}
	slots := int64(len(layers) + 1)
	var all []placed
	for layer, entries := range layers {
		size := int64(len(entries))
		for j, entry := range entries {
			all = append(all, placed{
				entry: entry, over: int64(j)*slots + int64(layer) + 1, under: size * slots, layer: layer,
			})
		}
	}
	slices.SortStableFunc(all, func(a, b placed) int {
		if order := cmp.Compare(a.over*b.under, b.over*a.under); order != 0 {
			return order
		}
		return cmp.Compare(a.layer, b.layer)
	})
	walk := make([]catalog.Entry, 0, len(all))
	for _, each := range all {
		walk = append(walk, each.entry)
	}
	return walk
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
//
// # Why every entity is walked rather than the first one that survives
//
// 🔴 It used to answer as soon as one entity kept a field, which made the check
// about the first entity in the recipe rather than about the recipe. An
// exclusion emptying the second entity passed here, the run generated every row
// and wrote every temporary file, and the failure arrived at the end as an
// entry the manifest model refused. That reached the reader as exit code 1 with
// a sentence telling them to report a defect in the tool, for a fault that was
// in their own file and was visible before a single row existed.
func (s *Selector) somethingToDamage() error {
	var emptied []string
	for _, entity := range s.view.EntityNames() {
		fields := s.view.FieldsOf(entity)
		if len(fields) == 0 {
			continue
		}
		if everyFieldExcluded(s, fields) {
			emptied = append(emptied, entity)
		}
	}
	if len(emptied) == 0 {
		return nil
	}
	held := "the entity has"
	if len(emptied) > 1 {
		held = "those entities have"
	}
	return fault.New(fault.Config, fmt.Sprintf(
		"policy.exclude_fields leaves nothing to damage in %s, because it names every "+
			"field %s",
		strings.Join(emptied, ", "), held)).
		DoInstead("remove a name from the list, or drop the policy")
}

// everyFieldExcluded answers whether a policy has taken away everything an
// entity could offer.
func everyFieldExcluded(s *Selector, fields []recipe.FieldInfo) bool {
	for _, field := range fields {
		if !s.excluded(field.Name) {
			return false
		}
	}
	return true
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
	position := s.start + s.taken
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
		field, instead := s.fieldFor(entry, entity, row)
		orders = append(orders, Order{
			Mutator: entry.ID,
			Entity:  entity,
			// The manifest counts rows from one, because a person reading it
			// counts from one.
			Row:     row + 1,
			Field:   field,
			Instead: instead,
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

// fieldFor picks the column an entry is aimed at in one row, and the other
// columns it covers, which the engine tries when the damage declines the value
// in the first. It answers empty when the entry damages something other than a
// field.
//
// Eight of the fifty two entries declare that they apply to no field type at
// all: four at L0, three at L1 and one at L5. Those damage the bytes, the
// container or the set, so naming a column for them would put a location in the
// manifest that the specification says such a layer does not carry.
func (s *Selector) fieldFor(entry catalog.Entry, entity string, row int) (string, []string) {
	if !aimsAtAField(entry) {
		return "", nil
	}
	covered, typed, available := s.candidates(entry, entity)
	if len(covered) > 0 {
		// Drawn rather than taken in order, so that two entries aimed at the
		// same kind of field do not always land on the same column.
		//
		// 🔴 The row is part of the address, so one entry ordered on many rows
		// spreads over every column it covers. Without it the draw is one per
		// entry for the whole run, and damage made of letters landed on a column
		// of national identifiers read as text in every row it was ordered, and
		// refused every time, while the columns of names beside it never got it
		// (measured 2026-11-13 on a file of 200 rows).
		at := seed.Coordinate{
			Entity: entity, Row: int64(row), Field: string(entry.ID), Purpose: "corrupt:field",
		}
		drawn := int(s.source.IntN(at, int64(len(covered))))
		// Read before the delete, which shifts the slice in place.
		field := covered[drawn]
		return field, slices.Delete(covered, drawn, drawn+1)
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
			return nearest[0], nil
		}
	}
	return "", nil
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
		if entry.CoversProvider(field.Scheme) && len(columnCannot(entry.Requires, field)) == 0 {
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
