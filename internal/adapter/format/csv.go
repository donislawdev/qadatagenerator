package format

import (
	"bufio"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// CSV writes one file per table.
//
// # Why this writer does not use the standard library
//
// The encoder there writes the line ending of whichever system it runs on and
// takes its own view of when a field needs quoting. Both are decisions this
// project has to make itself, because the bytes of the output are a promise and
// not an implementation detail. Writing them here is a dozen lines and removes
// a whole class of platform difference.
type CSV struct {
	files     Files
	rec       recipe.Recipe
	delimiter string
	keys      keys
	current   *csvTable
}

// Contract check.
var _ generate.Writer = (*CSV)(nil)

// csvTable is the file being written.
type csvTable struct {
	entity   recipe.Entity
	planned  dataset.Table
	file     WriteCloser
	buffered *bufio.Writer
	rows     int
	column   string
}

// NewCSV returns a writer for the comma separated form.
//
// The delimiter is what the recipe asked for, defaulting to a comma. It is an
// explicit setting rather than a detail because this is exactly where an import
// falls over, and the same difference on the other side of the product is a
// mutator.
func NewCSV(files Files, rec recipe.Recipe) (*CSV, error) {
	delimiter := rec.Output.Delimiter
	if delimiter == "" {
		delimiter = ","
	}
	if len([]rune(delimiter)) != 1 {
		return nil, fault.New(fault.Config, "the delimiter has to be one character and "+
			"the recipe gives "+delimiter)
	}
	if err := usableEncoding(rec.Output.Encoding); err != nil {
		return nil, err
	}
	return &CSV{files: files, rec: rec, delimiter: delimiter, keys: newKeys()}, nil
}

// Begin opens the file of one table and writes its header.
func (w *CSV) Begin(planned dataset.Table) error {
	entity, known := w.rec.Entity(planned.Entity)
	if !known {
		return fault.New(fault.Runtime, "no entity called "+planned.Entity)
	}
	column, err := parentColumn(entity, planned)
	if err != nil {
		return err
	}
	file, err := w.files.Create(planned.Entity + ".csv")
	if err != nil {
		return err
	}

	w.current = &csvTable{entity: entity, planned: planned, file: file,
		buffered: bufio.NewWriter(file), column: column}
	header := planned.Columns
	if column != "" {
		header = append(append([]string{}, header...), column)
	}
	return w.writeLine(header)
}

// Write puts one record on its line.
func (w *CSV) Write(record dataset.Record) error {
	line := record.Values
	if w.current.column != "" {
		line = append(append([]string{}, line...),
			w.keys.of(w.current.planned.Parent, record.Parent))
	}
	w.keys.record(w.current.planned.Entity,
		identify(w.current.entity, w.current.planned, record.Values, w.current.rows))
	w.current.rows++
	return w.writeLine(line)
}

// End flushes and closes the file of one table.
func (w *CSV) End(dataset.Table) error {
	if err := w.current.buffered.Flush(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	w.current = nil
	return nil
}

// Finish has nothing to do: this writer streams, so every table was already
// complete when it ended.
func (w *CSV) Finish() error { return nil }

// writeLine writes one row, quoting the fields that need it.
func (w *CSV) writeLine(fields []string) error {
	var line strings.Builder
	for index, field := range fields {
		if index > 0 {
			line.WriteString(w.delimiter)
		}
		line.WriteString(w.quote(field))
	}
	// A single newline, always, whatever the system. The mixed endings this
	// tool produces on purpose belong to a mutator and arrive as data rather
	// than from the platform.
	line.WriteByte('\n')
	if _, err := w.current.buffered.WriteString(line.String()); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	return nil
}

// quote wraps a field when it carries something that would end it early.
func (w *CSV) quote(field string) string {
	if !strings.ContainsAny(field, w.delimiter+"\"\r\n") {
		return field
	}
	return `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
}

// usableEncoding refuses an encoding this build does not write.
//
// It refuses by name rather than writing the bytes of one encoding under the
// label of another, which is the failure the recipe reader already avoids on
// the way in.
func usableEncoding(encoding string) error {
	switch strings.ToLower(encoding) {
	case "", "utf-8", "utf8":
		return nil
	default:
		return fault.New(fault.Config, "this build writes UTF-8 and the recipe asks for "+
			encoding+". Remove the setting or ask for utf-8")
	}
}
