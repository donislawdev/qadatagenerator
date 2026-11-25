package format

import (
	"bufio"
	"strconv"
	"strings"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// SQL writes one INSERT statement per row, for the database the recipe names
// (DEC-341).
//
// # Why one statement per row, and no transaction around them
//
// A database that refuses a value refuses the statement holding it. With one
// row per statement a refused row stops only itself: psql and sqlite3 go on to
// the next statement, the mysql client does when given --force, and sqlcmd does
// because every statement is its own batch here. A statement of many rows would
// lose all of them for one bad value. A transaction around the file measured
// worse than none on SQL Server: a value it could not convert rolled the
// transaction back, and the rows after it were then written one by one, so the
// database held the end of the file and not its start. Loading inside one
// transaction is many times faster, and is left to the command that loads the
// file, where a person chooses it.
//
// # Why every value is text in quotes
//
// The database turns text into the type of its column the way an import of a
// comma separated file does, and whether it accepts or refuses is what a test
// wants to see. Text is also the one way to hand a value over exactly as the
// manifest describes it, damage included. A cell given no value is NULL, and
// a value that is the empty text is the empty text (dataset.Record.Absent).
//
// # Why the column list is always written
//
// The statement then names every column it fills, so it does not depend on the
// order the person's own table declares them in, and a table with more columns
// than the recipe takes the rest from its defaults.
type SQL struct {
	files   Files
	rec     recipe.Recipe
	dialect dialect
	// values says what the values of each field are, which the table a recipe
	// asks the file to create is typed from (output.create_tables).
	values generate.Values
	keys   keys
	// shared is the file every table goes into when the recipe does not ask
	// for one per table. The first table opens it and Finish closes it.
	shared *statements
	// current is the table Begin last announced.
	current *sqlTable
}

// Contract check.
var _ generate.Writer = (*SQL)(nil)

// sqlTable is the table being written.
type sqlTable struct {
	entity  recipe.Entity
	planned dataset.Table
	column  string
	into    *statements
	// insert is the start every statement of this table shares, worked out
	// once rather than for every row.
	insert string
}

// statements is one file of statements on its way to a disk.
//
// The failure of a write is asked for once, when the file closes, for the
// reason given at document: the buffer keeps the first one.
type statements struct {
	to   *bufio.Writer
	file WriteCloser
	// written is how many statements of rows the file holds so far.
	written int
}

func (s *statements) write(text string) { _, _ = s.to.WriteString(text) }

func (s *statements) close() error { return finish(s.to, s.file) }

// NewSQL returns a writer of statements for the database the recipe names.
//
// values is where the run takes its values from, asked what the values of each
// field are when the recipe asks the file to create its tables.
func NewSQL(files Files, rec recipe.Recipe, values generate.Values) (*SQL, error) {
	if err := usableEncoding(rec.Output.Encoding); err != nil {
		return nil, err
	}
	if rec.Output.CreateTables && values == nil {
		return nil, fault.New(fault.Runtime, "the sql form was asked to create tables and given "+
			"nothing to type their columns from")
	}
	if err := SQLFits(rec, values); err != nil {
		return nil, err
	}
	written, _ := dialectOf(rec.Output.Dialect)
	return &SQL{files: files, rec: rec, dialect: written, values: values, keys: newKeys(rec)}, nil
}

// Begin starts a table, opening its file when each table goes to its own and
// the one file when it is the first table.
func (w *SQL) Begin(planned dataset.Table) error {
	entity, known := w.rec.Entity(planned.Entity)
	if !known {
		return fault.New(fault.Runtime, "no entity called "+planned.Entity)
	}
	column, err := parentColumn(entity, planned)
	if err != nil {
		return err
	}
	into, err := w.fileFor(planned)
	if err != nil {
		return err
	}
	columns := planned.Columns
	if column != "" {
		columns = append(append([]string{}, columns...), column)
	}
	quoted := make([]string, 0, len(columns))
	for _, name := range columns {
		quoted = append(quoted, w.dialect.quote(name))
	}
	w.current = &sqlTable{entity: entity, planned: planned, column: column, into: into,
		insert: "INSERT INTO " + w.dialect.quote(planned.Entity) + " (" +
			strings.Join(quoted, ", ") + ") VALUES ("}
	if w.rec.Output.CreateTables {
		// The table comes before its rows, and the plan puts a parent before
		// its children, so a link to a parent always names a table that is
		// there.
		into.write(shapeOfTable(w.rec, entity, planned.Columns, w.values).create(w.dialect))
		into.written++
	}
	return nil
}

// fileFor returns the file a table goes into.
//
// In the one file a table that follows another one with rows starts after an
// empty line, so that a person reading the file sees where each table begins.
// Nothing names the table there: a comment would carry the name of an entity,
// and a name with a line break in it would end the comment and become a
// statement.
func (w *SQL) fileFor(planned dataset.Table) (*statements, error) {
	if w.rec.Output.FilePerEntity {
		return w.open(planned.Entity + ".sql")
	}
	if w.shared == nil {
		opened, err := w.open("dataset.sql")
		if err != nil {
			return nil, err
		}
		w.shared = opened
		return opened, nil
	}
	if w.shared.written > 0 {
		w.shared.write("\n")
	}
	w.shared.written = 0
	return w.shared, nil
}

// open creates a file and writes the statements every file starts with.
func (w *SQL) open(name string) (*statements, error) {
	file, err := w.files.Create(name)
	if err != nil {
		return nil, err
	}
	opened := &statements{to: bufio.NewWriter(file), file: file}
	for _, line := range w.dialect.opening {
		opened.write(line + "\n" + w.dialect.after)
	}
	return opened, nil
}

// Write puts one row down as its statement.
func (w *SQL) Write(record dataset.Record) error {
	table := w.current
	var statement strings.Builder
	statement.WriteString(table.insert)
	for at, value := range record.Values {
		if at > 0 {
			statement.WriteString(", ")
		}
		if record.AbsentAt(at) {
			statement.WriteString("NULL")
			continue
		}
		statement.WriteString(w.dialect.literal(value))
	}
	if table.column != "" {
		if len(record.Values) > 0 {
			statement.WriteString(", ")
		}
		statement.WriteString(w.parentOf(table, record))
	}
	statement.WriteString(")" + statementEnd + "\n")
	statement.WriteString(w.dialect.after)
	w.keys.record(table.entity, table.planned, record.Values)
	table.into.write(statement.String())
	table.into.written++
	return nil
}

// parentOf writes what names the parent of a row: the key of the parent row,
// or NULL for a row that belongs to none of the rows the parent table holds.
func (w *SQL) parentOf(table *sqlTable, record dataset.Record) string {
	if !w.keys.holds(table.planned.Parent, record.Parent) {
		return "NULL"
	}
	return w.dialect.literal(w.keys.of(table.planned.Parent, record.Parent))
}

// End closes the file of a table that has one of its own.
func (w *SQL) End(dataset.Table) error {
	table := w.current
	w.current = nil
	if !w.rec.Output.FilePerEntity {
		return nil
	}
	return table.into.close()
}

// Finish closes the one file, opening it first in the case of a run with no
// tables at all, so that the file the run names is always there.
func (w *SQL) Finish() error {
	if w.rec.Output.FilePerEntity {
		return nil
	}
	if w.shared == nil {
		opened, err := w.open("dataset.sql")
		if err != nil {
			return err
		}
		w.shared = opened
	}
	return w.shared.close()
}

// SQLFits refuses a recipe the SQL form cannot write for the database it
// names, and says nothing about a recipe asking for another form.
//
// It is exported so that validate asks the question the writer asks, and the
// two cannot answer differently about one recipe, the arrangement
// NestingCollision has for the tree. values answers what the values of a field
// are, for the tables a recipe asks the file to create.
func SQLFits(rec recipe.Recipe, values generate.Values) error {
	if rec.Output.Format != NameSQL {
		return nil
	}
	written, known := dialectOf(rec.Output.Dialect)
	if !known {
		return fault.New(fault.Config, "the sql form writes for one of "+
			strings.Join(dataset.Dialects(), ", ")+", and the recipe asks for "+
			strconv.Quote(rec.Output.Dialect)).
			DoInstead("set output.dialect to the database the file will be loaded into")
	}
	if err := parentsHaveOneKey(rec); err != nil {
		return err
	}
	if err := namesFit(rec, written); err != nil {
		return err
	}
	return tablesFit(rec, values, written)
}

// parentsHaveOneKey refuses a relation the statements could not express.
//
// A child points at its parent by a value in one column. The other forms write
// the row number of a parent without a primary key, which a reader of the file
// can count to, and a table in a database has no row numbers at all: a parent
// with an id column counting from one and children counting rows from nought
// would join every child to the wrong parent without a word. A key of several
// fields is written joined by a hyphen elsewhere, which matches no column of
// the parent.
func parentsHaveOneKey(rec recipe.Recipe) error {
	for _, child := range rec.Entities {
		if len(child.Parents) == 0 {
			continue
		}
		name := child.Parents[0].Entity
		parent, known := rec.Entity(name)
		if !known {
			continue
		}
		switch len(parent.PrimaryKey) {
		case 1:
			continue
		case 0:
			return fault.New(fault.Config, "entities."+child.Name+" belongs to "+name+
				", which declares no primary key, and a database has no row numbers for the "+
				child.Name+" rows to point at instead").
				DoInstead("give " + name + " a primary_key of the one field that identifies its rows")
		default:
			return fault.New(fault.Config, "entities."+child.Name+" belongs to "+name+
				", whose primary key has "+strconv.Itoa(len(parent.PrimaryKey))+
				" fields, and the column of "+child.Name+" that names its parent holds one value").
				DoInstead("give " + name + " a primary_key of one field")
		}
	}
	return nil
}

// namesFit refuses a name the database would refuse or change.
//
// A control character is refused in every dialect, because a name with a line
// break would break a statement over two lines, the thing the values are
// escaped to avoid. Two names the database cuts to the same one are refused as
// the one name they become.
func namesFit(rec recipe.Recipe, written dialect) error {
	tables := map[string]string{}
	for _, entity := range rec.Entities {
		if err := nameFits(written, entity.Name, "entities."+entity.Name); err != nil {
			return err
		}
		if err := oneName(tables, written, entity.Name, "entities"); err != nil {
			return err
		}
		columns := map[string]string{}
		for _, field := range entity.Fields {
			where := "entities." + entity.Name + ".fields." + field.Name
			if err := nameFits(written, field.Name, where); err != nil {
				return err
			}
			if err := oneName(columns, written, field.Name, "entities."+entity.Name+".fields"); err != nil {
				return err
			}
		}
		if len(entity.Parents) > 0 {
			if err := oneName(columns, written, entity.Parents[0].Entity,
				"entities."+entity.Name+".fields"); err != nil {
				return err
			}
		}
	}
	return nil
}

// nameFits refuses one name.
func nameFits(written dialect, name, where string) error {
	for _, symbol := range name {
		if unicode.IsControl(symbol) {
			return fault.New(fault.Config, where+": the name holds a control character, and a "+
				"statement could not carry it on one line").
				DoInstead("rename it without the character")
		}
	}
	if said := written.nameFits(name); said != "" {
		return fault.New(fault.Config, where+": "+said).
			DoInstead("shorten or rename it, or write the file for another database")
	}
	return nil
}

// oneName refuses a name the database would take for one already given.
func oneName(seen map[string]string, written dialect, name, where string) error {
	key := clip(name, written.clipped)
	if earlier, taken := seen[key]; taken && earlier != name {
		return fault.New(fault.Config, where+": "+strconv.Quote(earlier)+" and "+
			strconv.Quote(name)+" begin with the same "+strconv.Itoa(written.clipped)+
			" bytes, and PostgreSQL keeps only those, so it would take both for one name").
			DoInstead("make them differ in their first " + strconv.Itoa(written.clipped) + " bytes")
	}
	seen[key] = name
	return nil
}
