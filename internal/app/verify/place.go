package verify

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// dataset is what the manifest says about the tables a response record can
// name, which is the half of the comparison the record cannot carry itself.
//
// # Why a row is placed before anything is compared
//
// A response record names a row by its number and, from version 0.2, by its
// entity. Until 2026-10-27 the number alone was looked up, so row 1 of a
// customer table and row 1 of an order table got the same verdict, and the rows
// that never arrived were counted against the sum of every table. A record
// about one of two tables then ended a clean run with the expectation code
// (B-192). So every row is placed in one table of the dataset first, and a row
// that cannot be placed is refused rather than guessed at: a row past the end
// of its table is usually every row shifted by one, because the producer
// counted the header, and a comparison of shifted rows is a verdict about the
// wrong records (DEC-266).
type dataset struct {
	// tables is every entity, in the order the manifest lists them.
	tables []manifest.Entity
	// counted reports whether every table carries its own number of rows. A
	// manifest written to 0.1 does not, and when its entries stand in several
	// tables every bound is the dataset as a whole.
	counted bool
	total   int
	// unnamed reports that the manifest names no entity at all, which only a
	// manifest written to 0.1 with no entry standing anywhere can do.
	unnamed bool
	order   map[string]int
}

// datasetOf reads the tables out of a manifest.
//
// A manifest written to 0.2 lists them. One written to 0.1 does not, so its
// tables are the entities its entries stand in: with one, or none, the dataset
// is a single table of every row, which is what that version could describe.
// With several, the tables are known by name and not by size.
func datasetOf(promised manifest.Manifest) dataset {
	run := promised.Run()
	if len(run.Entities) > 0 {
		return newDataset(run.Entities, true, run.RowsTotal)
	}
	named := map[string]bool{}
	for _, defect := range promised.Defects() {
		defect.NameEntities(named)
	}
	names := keys(named)
	if len(names) <= 1 {
		only := manifest.Entity{Rows: run.RowsTotal}
		if len(names) == 1 {
			only.Name = names[0]
		}
		built := newDataset([]manifest.Entity{only}, true, run.RowsTotal)
		built.unnamed = len(names) == 0
		return built
	}
	tables := make([]manifest.Entity, 0, len(names))
	for _, name := range names {
		tables = append(tables, manifest.Entity{Name: name})
	}
	return newDataset(tables, false, run.RowsTotal)
}

func newDataset(tables []manifest.Entity, counted bool, total int) dataset {
	order := make(map[string]int, len(tables))
	for at, table := range tables {
		order[table.Name] = at
	}
	return dataset{tables: tables, counted: counted, total: total, order: order}
}

// names is every entity of the dataset, in the order the manifest lists them.
func (d dataset) names() []string {
	names := make([]string, 0, len(d.tables))
	for _, table := range d.tables {
		names = append(names, table.Name)
	}
	return names
}

// several reports whether the dataset has more than one table, which is when a
// place has to say which one it is in.
func (d dataset) several() bool { return len(d.tables) > 1 }

// key is where a place of the manifest is looked up. With one table every
// place is in it, whatever the entry or the row calls it, so that a record
// leaving the entity out and a manifest naming it meet at one key.
func (d dataset) key(entity string, row int) response.Place {
	if !d.several() {
		entity = d.tables[0].Name
	}
	return response.Place{Entity: entity, Number: row}
}

// place puts every row of a record in one table of the dataset, or refuses the
// record with every reason at once.
//
// All at once, for the reason the record's own problems are: whoever produced
// the file runs the tool again after fixing what it named. Grouped by reason,
// because the likely mistake repeats on every row, and a record of four hundred
// thousand rows without an entity would otherwise be four hundred thousand
// sentences in one message.
func (d dataset) place(actual response.Record) (map[response.Place]response.Row, error) {
	placed := make(map[response.Place]response.Row, len(actual.Rows))
	refused := fault.Grouped{}
	written := map[string]bool{}
	for _, row := range actual.Rows {
		at, ok := d.locate(row, refused)
		if !ok {
			continue
		}
		if row.Entity != "" {
			written[row.Entity] = true
		}
		first, twice := placed[at]
		if !twice {
			placed[at] = row
			continue
		}
		// Two different names in one place is a manifest naming no entity and
		// a record naming two, which the sentence below says once for the
		// whole record rather than once for every row it shows up on.
		if first.Entity != "" && row.Entity != "" && first.Entity != row.Entity {
			continue
		}
		how := ""
		if first.Entity != row.Entity {
			how = ", once with the entity named and once without it"
		}
		refused.Add(fault.Reason{
			One:  "is answered twice in " + d.called(at.Entity) + how,
			Many: "are answered twice in " + d.called(at.Entity) + how,
		}, at.Number)
	}
	if d.unnamed && len(written) > 1 {
		said := fmt.Sprintf("the record names %d entities, %s, and the dataset is one table",
			len(written), fault.Listed(keys(written)))
		refused.Add(fault.Reason{One: said, Many: said}, 0)
	}
	if len(refused) > 0 {
		return nil, fault.New(fault.Config, "the response record cannot be put beside this "+
			"manifest: "+fault.Sentences(refused.Said())).
			DoInstead("number the rows of each entity from one, the first record after the " +
				"header being one, and name the entity of every row when the dataset has more " +
				"than one")
	}
	return placed, nil
}

// locate finds the one table a row is in, and records why when there is none.
func (d dataset) locate(row response.Row, refused fault.Grouped) (response.Place, bool) {
	at := d.key(row.Entity, row.Number)
	switch {
	case !d.several():
		if !d.unnamed && row.Entity != "" && row.Entity != d.tables[0].Name {
			refused.Add(unknown(row.Entity, d.tables[0].Name), row.Number)
			return at, false
		}
	case row.Entity == "":
		also := fmt.Sprintf("and the dataset has %d, %s", len(d.tables), fault.Listed(d.names()))
		refused.Add(fault.Reason{One: "names no entity, " + also, Many: "name no entity, " + also},
			row.Number)
		return at, false
	default:
		if _, known := d.order[row.Entity]; !known {
			refused.Add(unknown(row.Entity, fault.Listed(d.names())), row.Number)
			return at, false
		}
	}
	if bound, which := d.bound(at.Entity); row.Number > bound {
		refused.Add(fault.Reason{One: "is past the end of " + which, Many: "are past the end of " + which},
			row.Number)
		return at, false
	}
	return at, true
}

func unknown(entity, known string) fault.Reason {
	also := fmt.Sprintf("the entity %q, which the manifest does not list. It lists %s", entity, known)
	return fault.Reason{One: "names " + also, Many: "name " + also}
}

// bound is how many rows a table can have, and the words naming where that
// number comes from.
func (d dataset) bound(entity string) (int, string) {
	switch {
	case !d.counted:
		return d.total, "the dataset, which has " + rowsCalled(d.total) + " across its entities, " +
			"and the manifest does not say how many each one had"
	case !d.several():
		return d.total, "the dataset, which has " + rowsCalled(d.total)
	}
	rows := d.tables[d.order[entity]].Rows
	return rows, entity + ", which has " + rowsCalled(rows)
}

// rowsCalled is a number of rows with the noun agreeing, since a table of one
// row is ordinary for a child entity and "1 rows" is what it read otherwise.
func rowsCalled(rows int) string {
	if rows == 1 {
		return "1 row"
	}
	return strconv.Itoa(rows) + " rows"
}

// called names a table in a sentence, or the dataset when it is the only one.
func (d dataset) called(entity string) string {
	if !d.several() {
		return "the dataset"
	}
	return entity
}

// keys is a set written out, sorted.
func keys(set map[string]bool) []string {
	all := make([]string, 0, len(set))
	for key := range set {
		all = append(all, key)
	}
	sort.Strings(all)
	return all
}
