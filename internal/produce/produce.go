// Package produce answers what a recipe would produce, by producing a little of
// it and keeping none of it, and produces the whole of it when asked to.
//
// # Why this is a package of its own rather than a corner of the window
//
// It sits where the take and corrupt mode sits: above the use cases, below both
// surfaces, knowing neither. The window is its first caller and will not be its
// last, because the command line has a dry run that counts rows and cannot show
// one, and the day somebody wants `--sample` this is what answers it. Written
// inside the window instead, it would be a second implementation of generating
// waiting to happen, and this project has paid for one of those.
//
// # What makes a preview trustworthy
//
// Nothing here decides what a value is, what a column is called, or which
// column carries the relation. The real plan is made, the real engine runs, and
// the real writer composes the bytes into memory rather than onto a disk. What
// this package adds is exactly one thing: it stops after a few rows.
//
// # Why the PLAN is cut rather than the recipe
//
// Measured on 2026-09-02 and recorded as DEC-198. Lowering the count in the
// recipe is cheaper and gives a different dataset: two runs of one recipe under
// one seed, differing only in the child count, agreed on every value of every
// field and disagreed on every single key naming the parent. The parent of each
// child is decided by sharing the children out among the parents, which needs
// to know how many children there are. So the plan is made from the real counts
// and then cut, which keeps row i as row i, its parent included.
package produce

import (
	"encoding/csv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// DefaultRows is how many rows of each table a sample carries.
//
// Ten, decided by the owner on 2026-09-02. It is enough that a value repeating
// down a column reads as a pattern rather than as a coincidence, and few enough
// that the list of what was not checked still fits under it. It is a number
// somebody chose rather than a number anybody measured, and it is written down
// here once so that it is not chosen again at each caller.
const DefaultRows = 10

// Table is one table of the sample, as the file would carry it.
type Table struct {
	// Entity is the name the recipe gave it.
	Entity string
	// Columns are the columns of the file, which is not the same as the fields
	// of the recipe: a child carries one more, naming the parent it belongs to.
	Columns []string
	// Rows are the first rows, at most as many as were asked for.
	Rows [][]string
	// Total is how many rows a real run would write, which is what makes the
	// difference between a short table and a cut one visible.
	Total int
	// ParentColumn is where the key naming the parent sits, or minus one when
	// this table belongs to nobody.
	//
	// 🔴 It travels so that a surface can tell one empty cell from another. The
	// writer looks a parent up among the rows it has already written, and a
	// sample holds ten of them, so a child whose parent is the four hundredth
	// row gets an empty key back and no word about why. Everywhere else an
	// empty value is an ordinary thing for generated data to be. Without this
	// the two are the same cell.
	ParentColumn int
}

// ParentUnknown reports whether the key naming the parent could not be read
// back for this row, because the parent it names is outside the sample.
func (t Table) ParentUnknown(row int) bool {
	if t.ParentColumn < 0 || row >= len(t.Rows) || t.ParentColumn >= len(t.Rows[row]) {
		return false
	}
	return t.Rows[row][t.ParentColumn] == ""
}

// Cut reports whether the real run would write more rows than this carries.
func (t Table) Cut() bool { return t.Total > len(t.Rows) }

// File is one file of the sample, by the name it would have on a disk.
//
// It is carried beside the tables rather than instead of them, because the
// object form of the output is not a table and there is nothing honest to show
// for it but the bytes themselves.
type File struct {
	Name string
	Text string
}

// Sample is what a recipe would produce.
type Sample struct {
	// Tables is the readable view, filled for the comma separated form.
	Tables []Table
	// Files is what was written, by name, sorted so that two runs of one recipe
	// answer in one order.
	Files []File
	// Notes are what the plan says the run will not do, such as a column this
	// build has no values for. They travel with the sample because a preview
	// that shows an empty column without saying why is the silence this product
	// exists to end.
	Notes []generate.Note
}

// Preview produces the first rows of every table a recipe describes.
//
// The failure it returns is whatever the plan, the writer or the run said, with
// its kind intact, so that a caller can tell a recipe that cannot be used from
// a build that cannot do what it asks.
func Preview(rec recipe.Recipe, values generate.Values, rows int) (Sample, error) {
	if rows < 1 {
		rows = DefaultRows
	}
	// 🔴 Nothing can ask this to stop, the plan included, and the plan is the
	// part that is not short: it shares the children of every parent out before
	// a row is cut. What stops a check that a newer key press has made stale is
	// a question of the screen asking, and it is not answered yet (B-341).
	plan, err := planOf(rec, values, generate.NeverStops{})
	if err != nil {
		return Sample{}, err
	}

	held := newHeld()
	// A run of ten rows a table is over before anybody could stop it, and a port
	// that answers `carry on` says that plainly. The writer is told the same
	// thing as the run, because the one that writes at the end asks the question
	// itself.
	writer, err := format.For(held, rec, generate.NeverStops{})
	if err != nil {
		return Sample{}, err
	}
	if err := generate.Run(cut(plan, rows), rec, values, writer, generate.NeverStops{}); err != nil {
		return Sample{}, err
	}

	sample := Sample{Files: held.files(), Notes: plan.Notes}
	sample.Tables, err = tablesOf(plan, held, rec)
	if err != nil {
		return Sample{}, err
	}
	return sample, nil
}

// cut lowers every table of a plan to at most as many rows as were asked for.
//
// 🔴 The ownership is cut to the same count as well, and the honest reason is
// smaller than it looks: the engine walks rows up to the count and never past
// it, so a longer mapping would be ignored rather than harmful. It is cut because
// the planner sets the count FROM the mapping, so a plan carrying a count of ten
// beside a mapping of forty rows is a shape nothing else in this build produces,
// and shapes nothing produces are where the next reader loses an afternoon.
//
// What matters for correctness is the other half: the parents were shared out
// knowing how many children there are, and cutting a prefix keeps them. Row i
// of the sample is row i of the run because it is literally the same row.
func cut(plan generate.Plan, rows int) generate.Plan {
	cut := generate.Plan{
		Entities: make([]dataset.Table, len(plan.Entities)),
		Notes:    plan.Notes,
		Seed:     plan.Seed, Locale: plan.Locale, Tag: plan.Tag,
	}
	for at, table := range plan.Entities {
		if table.Rows > rows {
			table.Rows = rows
			table.Ownership = table.Ownership.Cut(rows)
		}
		cut.Entities[at] = table
	}
	return cut
}

// tablesOf reads back what the writer wrote.
//
// # Why the bytes are read rather than the records remembered
//
// The column naming the parent, the quoting, the separator and the order of the
// columns are all decisions of the writer, and a sample that assembled its own
// would be a second answer to what the file contains. Reading the bytes back
// with the standard library's own reader makes the sample a statement about the
// file rather than about our idea of it, which is the same reason the tooling
// checks artefacts with somebody else's parser.
func tablesOf(plan generate.Plan, held *held, rec recipe.Recipe) ([]Table, error) {
	if rec.Output.Format != "" && rec.Output.Format != format.NameCSV {
		return nil, nil
	}
	delimiter, err := format.Delimiter(rec)
	if err != nil {
		return nil, err
	}

	var tables []Table
	for _, planned := range plan.Entities {
		written, wrote := held.written[planned.Entity+".csv"]
		if !wrote {
			continue
		}
		reader := csv.NewReader(strings.NewReader(written.String()))
		reader.Comma = delimiter
		// A table of one column and a table of twenty are both ordinary, and a
		// reader holding the first row against the rest would refuse the recipe
		// this screen exists to be written on.
		reader.FieldsPerRecord = -1
		lines, err := reader.ReadAll()
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			continue
		}
		// The writer puts the key naming the parent after the fields, so the
		// last column is that key whenever the table belongs to somebody. It is
		// worked out from the plan rather than by matching the column name,
		// because a recipe may name a field after another entity and the writer
		// refuses exactly that case rather than resolving it.
		parent := -1
		if planned.Parent != "" {
			parent = len(lines[0]) - 1
		}
		tables = append(tables, Table{
			Entity: planned.Entity, Columns: lines[0], Rows: lines[1:], Total: planned.Rows,
			ParentColumn: parent,
		})
	}
	return tables, nil
}
