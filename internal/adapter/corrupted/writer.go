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
	"fmt"
	"sort"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Entries is where the record of every ordered damage goes.
//
// It is a port rather than the JSON writer itself, and the reason is the
// exports. A run may write the same entries in more than one form, and the
// forms are peers: each is fed as the entries happen, so none of them is
// another's input. A writer holding one concrete sink could only be given a
// second by growing a second field, and the third form would grow a third.
//
// Only Add is here. Closing a document belongs to whoever opened it, which is
// the surface that created the files, and a port that could also finish would
// invite this writer to finish something it did not begin.
type Entries interface {
	Add(defect manifest.Defect)
}

// Tee sends every entry to each sink in turn, in the order given.
//
// The order is kept rather than left to a map, because two documents describing
// one run have to be comparable line by line, and that is the same rule the
// rest of this project follows about iteration order.
type Tee []Entries

// Add hands the entry to every sink.
func (t Tee) Add(defect manifest.Defect) {
	for _, sink := range t {
		sink.Add(defect)
	}
}

// Writer damages records on their way to the real writer and records every
// order in the manifest.
type Writer struct {
	next     generate.Writer
	engine   corrupt.Engine
	selector *corrupt.Selector
	entries  Entries
	view     recipe.SchemaView

	table   dataset.Table
	row     int
	columns map[string]int
	fields  map[string]recipe.FieldInfo

	requested   int
	applied     int
	rowsMutated int
	// refusals counts why ordered damage did not happen, by kind. It is
	// counted here for the same reason the totals are: the entries are on a
	// disk by the time anybody could read them back, and reading them back to
	// count would be this tool parsing its own output.
	refusals map[manifest.FailureKind]int
}

// Contract check.
var _ generate.Writer = (*Writer)(nil)

// New wraps a writer so that what passes through it is damaged as the policy
// asks.
func New(next generate.Writer, engine corrupt.Engine, selector *corrupt.Selector,
	entries Entries, view recipe.SchemaView) *Writer {
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
		} else {
			if w.refusals == nil {
				w.refusals = map[manifest.FailureKind]int{}
			}
			w.refusals[entry.FailureKind]++
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
// ByKind says how much of the ordered damage did not happen, one line per kind,
// in a fixed order.
//
// Sorted rather than left to map iteration, because these lines reach a person
// and two runs of one seed have to read the same way. The sentences are written
// here rather than taken from the manifest, because the manifest says what
// happened to one defect and this says what happened to a run.
func (w *Writer) ByKind() []string {
	said := map[manifest.FailureKind]string{
		manifest.Unimplemented: "this build has no implementation of them yet",
		// 🔴 This said "which follows from the recipe" until 2026-08-11, and it
		// was true of every run that existed when it was written. The verb that
		// damages somebody else's file has no recipe, so the clause became a
		// sentence pointing at a thing the reader does not have. What replaced
		// it is true in both modes and points at where the answer is either way.
		manifest.RunLacksCapability:  "this run cannot do what they need, and the manifest names the missing capability for each one",
		manifest.FieldTypeNotCovered: "the dataset has no column of the kind they apply to",
		manifest.SchemeNotCovered:    "no column follows a numbering scheme they apply to",
		manifest.DeclinedByMutator:   "the values in those rows did not suit them",
		manifest.Undescribed:         "nothing in the catalog describes them, which is a fault in this tool",
	}
	kinds := make([]manifest.FailureKind, 0, len(w.refusals))
	for kind := range w.refusals {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i] < kinds[j] })

	lines := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		explained, known := said[kind]
		if !known {
			// A kind nobody wrote a sentence for is still counted rather than
			// dropped. Saying less than the truth here would be the silence
			// this whole package exists to prevent, one level up.
			explained = string(kind)
		}
		lines = append(lines, fmt.Sprintf("  %d because %s", w.refusals[kind], explained))
	}
	return lines
}

func (w *Writer) Requested() int   { return w.requested }
func (w *Writer) Applied() int     { return w.applied }
func (w *Writer) RowsMutated() int { return w.rowsMutated }
