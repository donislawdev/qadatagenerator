package format

import (
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
func NewJSON(files Files, rec recipe.Recipe) (*JSON, error) {
	if err := usableEncoding(rec.Output.Encoding); err != nil {
		return nil, err
	}
	return &JSON{files: files, rec: rec, keys: newKeys()}, nil
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
	var built strings.Builder
	w.writeTable(&built, table, "")
	return write(file, built.String())
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
	var built strings.Builder
	built.WriteString("{\n")
	roots := 0
	for _, table := range w.tables {
		if table.planned.Parent != "" {
			continue
		}
		if roots > 0 {
			built.WriteString(",\n")
		}
		built.WriteString("  " + quoteJSON(table.planned.Entity) + ": ")
		w.writeNested(&built, table, wholeOf(table), "  ")
		roots++
	}
	built.WriteString("\n}\n")
	return write(file, built.String())
}

// writeTable writes one table as a flat array of objects.
func (w *JSON) writeTable(into *strings.Builder, table *jsonTable, indent string) {
	into.WriteString("[\n")
	for index, record := range table.rows {
		if index > 0 {
			into.WriteString(",\n")
		}
		into.WriteString(indent + "  ")
		w.writeRecord(into, table, record, indent+"  ", nil)
	}
	if len(table.rows) > 0 {
		into.WriteString("\n")
	}
	into.WriteString(indent + "]")
	if indent == "" {
		into.WriteString("\n")
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
func (w *JSON) writeNested(into *strings.Builder, table *jsonTable, rows []placed, indent string) {
	children := w.childrenOf(table.planned.Entity)
	into.WriteString("[\n")
	for index, row := range rows {
		if index > 0 {
			into.WriteString(",\n")
		}
		into.WriteString(indent + "  ")
		w.writeRecord(into, table, row.record, indent+"  ", w.gather(children, row.at))
	}
	if len(rows) > 0 {
		into.WriteString("\n")
	}
	into.WriteString(indent + "]")
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
func (w *JSON) writeRecord(into *strings.Builder, table *jsonTable, record dataset.Record,
	indent string, children []nested) {
	into.WriteString("{")
	for index, column := range table.planned.Columns {
		if index > 0 {
			into.WriteString(",")
		}
		into.WriteString("\n" + indent + "  " + quoteJSON(column) + ": " +
			quoteJSON(valueAt(record, index)))
	}
	if table.column != "" {
		into.WriteString(",\n" + indent + "  " + quoteJSON(table.column) + ": " +
			quoteJSON(w.keys.of(table.planned.Parent, record.Parent)))
	}
	for _, child := range children {
		into.WriteString(",\n" + indent + "  " + quoteJSON(child.entity) + ": ")
		w.writeNested(into, child.table, child.rows, indent+"  ")
	}
	into.WriteString("\n" + indent + "}")
}

func valueAt(record dataset.Record, index int) string {
	if index >= len(record.Values) {
		return ""
	}
	return record.Values[index]
}

func write(file WriteCloser, text string) error {
	if _, err := file.Write([]byte(text)); err != nil {
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
