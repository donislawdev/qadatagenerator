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
	files Files
	rec   recipe.Recipe
	keys  keys
	// tables are the tables held for the tree, which cannot be written before
	// the last child has arrived. A file per entity holds none of them.
	tables []*jsonTable
	// current is the table Begin last announced.
	current *jsonTable
	// stop is what this writer asks whether the run may carry on while it
	// writes the tree.
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

// jsonTable is one table, either held for the tree or being written to its own
// file.
type jsonTable struct {
	entity  recipe.Entity
	planned dataset.Table
	column  string
	// rows are the records of a table held for the tree.
	rows []dataset.Record
	// children are the tables that belong to this one, and family says which
	// of this table's rows belong to which row of its parent. Both are worked
	// out once, when the tree is written.
	children []*jsonTable
	family   family
	// into is the file of a table written on its own, and written is how many
	// records it holds so far.
	into    document
	written int
}

// NewJSON returns a writer for the object form.
func NewJSON(files Files, rec recipe.Recipe, stop generate.Halt) (*JSON, error) {
	if err := usableEncoding(rec.Output.Encoding); err != nil {
		return nil, err
	}
	if err := NestingCollision(rec); err != nil {
		return nil, err
	}
	return &JSON{files: files, rec: rec, keys: newKeys(rec), stop: stop}, nil
}

// NestingCollision refuses a recipe whose tree would give one object two
// members of the same name.
//
// # Why only the tree, and why this direction
//
// In one file the children of a record are written inside it, under the name
// of their entity, after its own fields. A parent with a field named after a
// child entity therefore got the field and then the array under one name, and
// the run ended with success. Found by an outside review on 2026-10-22. What a
// reader does with two members of one name is up to the reader, which is why
// it is refused rather than renamed. The other direction, a child with a field
// named after its parent, is ParentColumnCollision and holds in every form. A
// file per entity writes no tree, so nothing collides there.
//
// It is exported so that validate can ask the question the writer asks, and the
// two cannot answer differently about one recipe.
func NestingCollision(rec recipe.Recipe) error {
	if rec.Output.Format != NameJSON || rec.Output.FilePerEntity {
		return nil
	}
	for _, parent := range rec.Entities {
		for _, child := range rec.Entities {
			if len(child.Parents) == 0 || child.Parents[0].Entity != parent.Name {
				continue
			}
			for _, field := range parent.Fields {
				if field.Name != child.Name {
					continue
				}
				return fault.New(fault.Config, "entities."+parent.Name+".fields."+field.Name+
					": the field has the name of the entity "+child.Name+", whose records are "+
					"written inside each "+parent.Name+" under that name, so one object would "+
					"hold two members called "+field.Name).
					DoInstead("rename the field, or set output.file_per_entity to write each table to its own file")
			}
		}
	}
	return nil
}

// Begin starts a table, and opens its file when each table goes to its own.
func (w *JSON) Begin(planned dataset.Table) error {
	entity, known := w.rec.Entity(planned.Entity)
	if !known {
		return fault.New(fault.Runtime, "no entity called "+planned.Entity)
	}
	column, err := parentColumn(entity, planned)
	if err != nil {
		return err
	}
	table := &jsonTable{entity: entity, planned: planned, column: column}
	w.current = table
	if !w.rec.Output.FilePerEntity {
		w.tables = append(w.tables, table)
		return nil
	}
	file, err := w.files.Create(planned.Entity + ".json")
	if err != nil {
		return err
	}
	table.into = newDocument(file)
	table.into.write("[\n")
	return nil
}

// Write puts a record in its file, or keeps it for the tree.
//
// # Why one shape streams and the other holds
//
// 🔴 Until 2026-10-27 every record was held here whatever the shape, on the
// grounds that two code paths for one format is how the two drift apart, and a
// file per entity of two million rows took 797 MiB where the comma separated
// form took 199 (outside review, 2026-10-22, DEC-266). The argument did not
// hold: what the two shapes share is the writing of one record, and that is one
// function either way, so they differ in when it is called and not in what it
// writes. The tree still holds its records, because it cannot be written before
// the last child has arrived.
func (w *JSON) Write(record dataset.Record) error {
	table := w.current
	w.keys.record(table.entity, table.planned, record.Values)
	if !w.rec.Output.FilePerEntity {
		table.rows = append(table.rows, record)
		return nil
	}
	if table.written > 0 {
		table.into.write(",\n")
	}
	table.into.write("  ")
	w.writeRecord(table.into, table, record, "  ", nil)
	table.written++
	return nil
}

// End finishes the file of a table written on its own and closes it.
//
// The halt is not asked here, because in this shape the record is written at
// the moment it is produced, and the run asks before every one of them.
func (w *JSON) End(dataset.Table) error {
	if !w.rec.Output.FilePerEntity {
		return nil
	}
	table := w.current
	if table.written > 0 {
		table.into.write("\n")
	}
	table.into.write("]\n")
	return table.into.close()
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
	w.relate()
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
	// The stop that interrupted the walk comes first, and the bytes are only
	// asked for when nothing interrupted it.
	if w.ended != nil {
		return w.ended
	}
	return into.close()
}

// relate works out, once for the whole tree, which tables belong to which and
// which rows of each belong to which row of its parent.
//
// # Why once
//
// 🔴 Both used to be worked out again for every record the tree held: the
// tables by walking all of them, and the children of a row by walking every row
// of every child table, so a relation of one child to each of P parents cost P
// times P comparisons to write 2P records. Found by an outside review on
// 2026-10-22.
func (w *JSON) relate() {
	byName := make(map[string]*jsonTable, len(w.tables))
	for _, table := range w.tables {
		byName[table.planned.Entity] = table
	}
	for _, table := range w.tables {
		parent, held := byName[table.planned.Parent]
		if table.planned.Parent == "" || !held {
			continue
		}
		parent.children = append(parent.children, table)
		table.family = familyOf(table.rows, len(parent.rows))
	}
}

// family says which rows of a table belong to which row of its parent.
type family struct {
	// The rows of parent p are at[starts[p]:starts[p+1]], as positions in the
	// table.
	starts []int
	at     []int
}

// familyOf sorts the rows of a table by the parent they belong to, keeping the
// order of the table among the rows of one parent, which is the order the tree
// has always written them in.
//
// It counts rather than assuming that the children of one parent stand
// together, although a plan always puts them so, because what a writer is
// handed is records and not a promise about them. A row naming no parent the
// table above holds is left out, as it always was: the tree is walked from its
// roots, and such a row hangs under none of them.
func familyOf(rows []dataset.Record, parents int) family {
	starts := make([]int, parents+1)
	for _, record := range rows {
		if record.Parent >= 0 && record.Parent < parents {
			starts[record.Parent+1]++
		}
	}
	for parent := range parents {
		starts[parent+1] += starts[parent]
	}
	at := make([]int, starts[parents])
	next := append([]int(nil), starts[:parents]...)
	for position, record := range rows {
		if record.Parent >= 0 && record.Parent < parents {
			at[next[record.Parent]] = position
			next[record.Parent]++
		}
	}
	return family{starts: starts, at: at}
}

// halted asks whether something outside the run wants it to stop, and remembers
// the answer.
//
// # Why once per record rather than once per table
//
// The tree is written after every record has been produced, so the questions
// the run asked on the way are all behind it. Once per table would be one
// question for a recipe with one entity, asked before a single byte is written,
// which is a writer that cannot be stopped at all while looking like one that
// can. The question costs a call through an interface, which is what the run
// pays per row already.
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
	into.write("[\n")
	for index, row := range rows {
		if w.halted() {
			return
		}
		if index > 0 {
			into.write(",\n")
		}
		into.write(indent + "  ")
		w.writeRecord(into, table, row.record, indent+"  ", gather(table.children, row.at))
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
func gather(children []*jsonTable, parent int) []nested {
	gathered := make([]nested, 0, len(children))
	for _, child := range children {
		positions := child.family.at[child.family.starts[parent]:child.family.starts[parent+1]]
		rows := make([]placed, 0, len(positions))
		for _, at := range positions {
			rows = append(rows, placed{record: child.rows[at], at: at})
		}
		gathered = append(gathered, nested{entity: child.planned.Entity, table: child, rows: rows})
	}
	return gathered
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
// What is left in memory for the tree is the records, and that one is not an
// oversight: the nested shape cannot be written before the last child has
// arrived, which is said at Write above. A file per entity holds none.
//
// # Why the failure of a write is not checked at each write
//
// A buffered writer keeps the first failure and turns every later write into
// nothing, so asking once at the end asks exactly as well as asking twenty
// times. What matters is that it is asked at all: a file that could not be
// written must never be reported as written, and the guard beside this holds
// that rather than the reading of this paragraph.
type document struct {
	to   *bufio.Writer
	file WriteCloser
}

// newDocument wraps a file in the same buffer the other writer of this package
// uses, of the same size, because two buffer sizes would be two answers to a
// question neither form is asking.
func newDocument(file WriteCloser) document {
	return document{to: bufio.NewWriter(file), file: file}
}

func (d document) write(text string) { _, _ = d.to.WriteString(text) }

func (d document) close() error { return finish(d.to, d.file) }

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
