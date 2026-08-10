// Package corrupted joins the two pillars without letting either import the
// other.
//
// # Why a decorator over the writer
//
// The corruption engine takes a value and gives back a value and an entry. The
// generator hands records to a writer. Putting the engine between them means
// the generator does not know damage exists and the writer does not know a
// value was changed, which is what keeps the second pillar from reaching into
// the first.
//
// It lives in an adapter rather than in either use case because a use case may
// not import another one's port. The rule that the corruption package never
// imports the generating package is the architectural form of each pillar being
// useful on its own.
//
// # Why the damage happens here rather than after the file is written
//
// A second pass over the finished file would have to parse what this tool just
// produced, which the architecture refuses, and would have lost the value that
// was there before. Doing it in flight also means the same arrangement works
// when the records come from somebody else's file instead of from a recipe.
package corrupted

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Writer damages records on their way to the real writer and records every
// order in the manifest.
type Writer struct {
	next     generate.Writer
	engine   corrupt.Engine
	selector *corrupt.Selector
	entries  *manifestjson.Writer
	view     recipe.SchemaView

	table   dataset.Table
	row     int
	columns map[string]int
	fields  map[string]recipe.FieldInfo

	requested   int
	applied     int
	rowsMutated int
}

// Contract check.
var _ generate.Writer = (*Writer)(nil)

// New wraps a writer so that what passes through it is damaged as the policy
// asks.
func New(next generate.Writer, engine corrupt.Engine, selector *corrupt.Selector,
	entries *manifestjson.Writer, view recipe.SchemaView) *Writer {
	return &Writer{next: next, engine: engine, selector: selector, entries: entries, view: view}
}

// Begin announces a table and works out where its columns are.
func (w *Writer) Begin(table dataset.Table) error {
	w.table, w.row = table, 0
	w.columns = make(map[string]int, len(table.Columns))
	for at, name := range table.Columns {
		w.columns[name] = at
	}
	w.fields = make(map[string]recipe.FieldInfo, len(table.Columns))
	for _, field := range w.view.FieldsOf(table.Entity) {
		w.fields[field.Name] = field
	}
	return w.next.Begin(table)
}

// Write damages the record if this row was chosen, then passes it on.
func (w *Writer) Write(record dataset.Record) error {
	row := w.row
	w.row++
	damaged := false
	for _, order := range w.selector.Orders(w.table.Entity, row, w.table.Rows) {
		entry := w.apply(order, record, row)
		w.entries.Add(entry)
		w.requested++
		if entry.Applied {
			w.applied++
			damaged = true
		}
	}
	if damaged {
		w.rowsMutated++
	}
	return w.next.Write(record)
}

// apply carries out one order against the record and returns the entry.
//
// A record whose values are changed in place is what the writer then receives,
// which is why the engine returning the original on refusal matters: the caller
// writes back whatever it got and is correct either way.
func (w *Writer) apply(order corrupt.Order, record dataset.Record, row int) manifest.Defect {
	at := seed.Coordinate{
		Entity: order.Entity, Row: int64(row), Field: order.Field, Purpose: "corrupt",
	}
	column, addressed := w.columns[order.Field]
	if !addressed {
		// Damage aimed at no column of this table: either the entry damages the
		// bytes or the container rather than a field, or no column suited it.
		// The engine still answers, and the answer is an entry with a reason.
		_, entry := w.engine.Apply(order, w.fields[order.Field], "", at)
		return entry
	}
	value, entry := w.engine.Apply(order, w.fields[order.Field], record.Values[column], at)
	record.Values[column] = value
	return entry
}

// End closes a table.
func (w *Writer) End(table dataset.Table) error { return w.next.End(table) }

// Finish closes the underlying writer.
func (w *Writer) Finish() error { return w.next.Finish() }

// Requested, Applied and RowsMutated are what the envelope needs.
//
// They are counted here rather than gathered from the manifest afterwards,
// because the entries are already on disk by then and reading them back to
// count would be this tool parsing its own output.
func (w *Writer) Requested() int   { return w.requested }
func (w *Writer) Applied() int     { return w.applied }
func (w *Writer) RowsMutated() int { return w.rowsMutated }
