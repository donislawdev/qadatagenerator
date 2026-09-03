package format

import (
	"bufio"
	"strings"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// JSON writes the dataset as objects, flat or nested.
//
// # Two shapes, chosen by the setting that already exists
//
// A file per entity gives an array of objects per table, with a child naming
// its parent in a field. One file gives the tree the recipe describes, children
// inside the parent they belong to. The recipe already carries that choice for
// the other formats, so this writer reads it rather than adding a second way of
// saying the same thing.
//
// # Why the bytes are written here rather than marshalled
//
// The order of keys has to be the order of the recipe, and the standard library
// orders by field name or by map iteration, neither of which is that. Escaping
// has to be ours as well: the standard encoder turns three characters into
// escapes for the benefit of browsers, which is a decision about somebody
// else's problem being made inside our data. Writing it here also removes the
// open question of whether the encoder is stable between releases of Go, since
// the bytes now depend on this file and a golden test stands in front of it.
type JSON struct {
	files  Files
	rec    recipe.Recipe
	keys   keys
	tables []*jsonTable
	// stop is what this writer asks whether the run may carry on.
	//
	// 🔴 It is held here rather than passed to Finish, which would read better,
	// because the two pillars declare the same writer port under two names and
	// Go matches them by shape. A parameter of a named Halt type would make the
	// two shapes different, and the one writer that satisfies both without a
	// line of adapting code would stop satisfying either.
	stop generate.Halt
	// ended is the failure that stopped the run, kept because the walk that
	// meets it writes nothing back.
	ended error
}

// Contract check.
var _ generate.Writer = (*JSON)(nil)

// jsonTable is one table held until the shape is known.
type jsonTable struct {
	entity  recipe.Entity
	planned dataset.Table
	column  string
	rows    []dataset.Record
}

// NewJSON returns a writer for the object form.
func NewJSON(files Files, rec recipe.Recipe, stop generate.Halt) (*JSON, error) {
	if err := usableEncoding(rec.Output.Encoding); err != nil {
		return nil, err
	}
	return &JSON{files: files, rec: rec, keys: newKeys(), stop: stop}, nil
}

// Begin starts a table.
func (w *JSON) Begin(planned dataset.Table) error {
	entity, known := w.rec.Entity(planned.Entity)
	if !known {
		return fault.New(fault.Runtime, "no entity called "+planned.Entity)
	}
	column, err := parentColumn(entity, planned)
	if err != nil {
		return err
	}
	w.tables = append(w.tables, &jsonTable{entity: entity, planned: planned, column: column})
	return nil
}

// Write keeps a record.
//
// Everything is held rather than streamed, and that is a real cost stated
// rather than hidden: the nested shape cannot be written before the last child
// has arrived, so this writer is the one place in the first pillar that does
// not stream. The flat shape could, and does not, because two code paths for
// one format is how the two drift apart.
func (w *JSON) Write(record dataset.Record) error {
	table := w.tables[len(w.tables)-1]
	w.keys.record(table.planned.Entity,
		identify(table.entity, table.planned, record.Values, len(table.rows)))
	table.rows = append(table.rows, record)
	return nil
}

// End finishes a table and writes it when each one goes to its own file.
func (w *JSON) End(planned dataset.Table) error {
	if !w.rec.Output.FilePerEntity {
		return nil
	}
	table := w.tables[len(w.tables)-1]
	file, err := w.files.Create(planned.Entity + ".json")
	if err != nil {
		return err
	}
	into := newDocument(file)
	w.writeTable(into, table, "")
	return w.endOf(into)
}

// Finish writes the tree when the whole dataset goes to one file.
func (w *JSON) Finish() error {
	if w.rec.Output.FilePerEntity {
		return nil
	}
	file, err := w.files.Create("dataset.json")
	if err != nil {
		return err
	}
	into := newDocument(file)
	into.write("{\n")
	roots := 0
	for _, table := range w.tables {
		if table.planned.Parent != "" {
			continue
		}
		if w.halted() {
			break
		}
		if roots > 0 {
			into.write(",\n")
		}
		into.write("  " + quoteJSON(table.planned.Entity) + ": ")
		w.writeNested(into, table, wholeOf(table), "  ")
		roots++
	}
	into.write("\n}\n")
	return w.endOf(into)
}

// halted asks whether something outside the run wants it to stop, and remembers
// the answer.
//
// # Why once per record rather than once per table
//
// Once per table is what the run above asks between tables, and it would be
// enough here for a recipe with several of them. For a recipe with one entity
// it would be one question asked before a single byte is written, which is a
// writer that cannot be stopped at all while looking like one that can. The
// question costs a call through an interface, which is what the run pays per
// row already.
//
// A walk that stops leaves the document unfinished, and that is the arrangement
// working rather than failing: the bytes are still under a temporary name and
// the caller throws them away.
func (w *JSON) halted() bool {
	if w.ended != nil {
		return true
	}
	w.ended = w.stop.Stopped()
	return w.ended != nil
}

// endOf finishes one file: the stop that interrupted the walk comes first, and
// the bytes are only asked for when nothing interrupted it.
func (w *JSON) endOf(into document) error {
	if w.ended != nil {
		return w.ended
	}
	return into.flush()
}

// writeTable writes one table as a flat array of objects.
func (w *JSON) writeTable(into document, table *jsonTable, indent string) {
	into.write("[\n")
	for index, record := range table.rows {
		if w.halted() {
			return
		}
		if index > 0 {
			into.write(",\n")
		}
		into.write(indent + "  ")
		w.writeRecord(into, table, record, indent+"  ", nil)
	}
	if len(table.rows) > 0 {
		into.write("\n")
	}
	into.write(indent + "]")
	if indent == "" {
		into.write("\n")
	}
}

// placed is a record together with the position it holds in its own table.
//
// 🔴 The position travels with the record because a child names its parent by
// the row number in the whole table, and the tree walks subsets of that table.
// Writing the grandchildren of a subset by the position inside the subset put
// them under the wrong parent and left the real ones out of the file
// altogether, with the record count still adding up.
type placed struct {
	record dataset.Record
	at     int
}

// wholeOf numbers every row of a table, which is what the roots of the tree are
// written from.
func wholeOf(table *jsonTable) []placed {
	rows := make([]placed, 0, len(table.rows))
	for at, record := range table.rows {
		rows = append(rows, placed{record: record, at: at})
	}
	return rows
}

// writeNested writes the given rows of one table with their children inside
// each record.
//
// The rows are passed in rather than read from the table, because a table
// appears once as a whole and once per parent as a subset of itself, and the
// writer has no business changing what the table holds in order to say which.
func (w *JSON) writeNested(into document, table *jsonTable, rows []placed, indent string) {
	children := w.childrenOf(table.planned.Entity)
	into.write("[\n")
	for index, row := range rows {
		if w.halted() {
			return
		}
		if index > 0 {
			into.write(",\n")
		}
		into.write(indent + "  ")
		w.writeRecord(into, table, row.record, indent+"  ", w.gather(children, row.at))
	}
	if len(rows) > 0 {
		into.write("\n")
	}
	into.write(indent + "]")
}

// nested is the children of one record, grouped by the entity they belong to.
type nested struct {
	entity string
	table  *jsonTable
	rows   []placed
}

// gather collects the children of one parent row, keeping the position each of
// them holds in its own table so that its own children can be found.
func (w *JSON) gather(children []*jsonTable, parent int) []nested {
	gathered := make([]nested, 0, len(children))
	for _, child := range children {
		rows := make([]placed, 0)
		for at, record := range child.rows {
			if record.Parent == parent {
				rows = append(rows, placed{record: record, at: at})
			}
		}
		gathered = append(gathered, nested{entity: child.planned.Entity, table: child, rows: rows})
	}
	return gathered
}

// childrenOf returns the tables that belong to an entity, in plan order.
func (w *JSON) childrenOf(entity string) []*jsonTable {
	var children []*jsonTable
	for _, table := range w.tables {
		if table.planned.Parent == entity {
			children = append(children, table)
		}
	}
	return children
}

// writeRecord writes one object.
func (w *JSON) writeRecord(into document, table *jsonTable, record dataset.Record,
	indent string, children []nested) {
	into.write("{")
	for index, column := range table.planned.Columns {
		if index > 0 {
			into.write(",")
		}
		into.write("\n" + indent + "  " + quoteJSON(column) + ": " +
			quoteJSON(valueAt(record, index)))
	}
	if table.column != "" {
		into.write(",\n" + indent + "  " + quoteJSON(table.column) + ": " +
			quoteJSON(w.keys.of(table.planned.Parent, record.Parent)))
	}
	for _, child := range children {
		into.write(",\n" + indent + "  " + quoteJSON(child.entity) + ": ")
		w.writeNested(into, child.table, child.rows, indent+"  ")
	}
	into.write("\n" + indent + "}")
}

func valueAt(record dataset.Record, index int) string {
	if index >= len(record.Values) {
		return ""
	}
	return record.Values[index]
}

// document is where the bytes of one file go on their way to a disk.
//
// # Why a buffer rather than the string this used to build
//
// The whole document was assembled in memory and handed over in one call, which
// held the dataset three times at the moment of writing: the records, the text
// of them, and the copy that call made of the text.
//
// Measured by tools/kanal_i_pamiec.py on 200 000 rows of two fields, as the peak
// working set of the process: 106 MB before, in one run on 2026-09-03, and 65 MB
// after, in four runs on the same day that gave 65 MB every time. The reading
// from 2026-09-02, one run of a binary two commits older, was 95 MB. The spread
// of the before readings is 11 MB and the difference is 30, so the difference is
// the wider of the two. The file is 11 543 991 bytes and identical to the byte,
// which the golden tests say rather than this paragraph. The comma separated
// form, which streams, takes 34 MB for the same rows.
//
// What is left in memory is the records, and that one is not an oversight: the
// nested shape cannot be written before the last child has arrived, which is
// said at Write above.
//
// # Why the failure of a write is not checked at each write
//
// A buffered writer keeps the first failure and turns every later write into
// nothing, so asking once at the end asks exactly as well as asking twenty
// times. What matters is that it is asked at all: a file that could not be
// written must never be reported as written, and the guard beside this holds
// that rather than the reading of this paragraph.
type document struct{ to *bufio.Writer }

// newDocument wraps a file in the same buffer the other writer of this package
// uses, of the same size, because two buffer sizes would be two answers to a
// question neither form is asking.
func newDocument(file WriteCloser) document { return document{to: bufio.NewWriter(file)} }

func (d document) write(text string) { _, _ = d.to.WriteString(text) }

func (d document) flush() error {
	if err := d.to.Flush(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	return nil
}

// quoteJSON writes a string the way this project writes strings.
//
// Only what the grammar requires is escaped. The characters a browser cares
// about are left alone, because a dataset is not a web page and a mutator that
// puts a control character in a value has to survive being written out.
func quoteJSON(value string) string {
	var built strings.Builder
	built.Grow(len(value) + 2)
	built.WriteByte('"')
	for _, symbol := range value {
		switch symbol {
		case '"':
			built.WriteString(`\"`)
		case '\\':
			built.WriteString(`\\`)
		case '\n':
			built.WriteString(`\n`)
		case '\r':
			built.WriteString(`\r`)
		case '\t':
			built.WriteString(`\t`)
		default:
			writeSymbol(&built, symbol)
		}
	}
	built.WriteByte('"')
	return built.String()
}

// writeSymbol writes one character, escaping the ones the grammar forbids raw
// and the one that is not a character at all.
func writeSymbol(into *strings.Builder, symbol rune) {
	const hex = "0123456789abcdef"
	switch {
	case symbol < 0x20:
		into.WriteString(`\u00`)
		into.WriteByte(hex[symbol>>4])
		into.WriteByte(hex[symbol&0xF])
	case symbol == utf8.RuneError:
		// A byte that is not valid UTF-8 arrives here as the replacement
		// character. Writing it as an escape keeps the file readable rather
		// than carrying a byte no reader can interpret.
		into.WriteString(`�`)
	default:
		into.WriteRune(symbol)
	}
}
