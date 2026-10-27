package format

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// PlainCSV writes back one table that was read rather than planned.
//
// # Why this is not the writer above with a switch in it
//
// The writer built from a recipe refuses a table whose entity the recipe does
// not describe, works out the column a child carries to point at its parent,
// and remembers what identifies every row so that it can be pointed at. All
// three are answers to questions a recipe asked. A file somebody hands over has
// one table, no relations and no declared key, so all three would have to be
// turned off, and turning a refusal off with a flag turns it off for every
// caller rather than for the one that has no recipe.
//
// What the two share is the part that is a promise: the quoting, the single
// newline and the encoding. That lives in one type and both use it.
//
// # What this changes about a file, whatever the policy says
//
// The bytes it writes are UTF-8 with newline endings and our quoting, so a
// source that used carriage returns, a byte order mark or different quoting
// comes back out differently in those respects. That is a consequence of
// writing rather than damage that was ordered, so it belongs in what the run
// says about itself and never in the manifest, where every entry is something
// somebody asked for.
type PlainCSV struct {
	files     Files
	name      string
	delimiter string
	lines     lines
	begun     bool
}

// Contract check against the port of the use case that reads a dataset. It is
// written against the narrow shape rather than the package, so this file stays
// free of a dependency on either use case.
var _ interface {
	Begin(dataset.Table) error
	Write(dataset.Record) error
	End(dataset.Table) error
	Finish() error
} = (*PlainCSV)(nil)

// NewPlainCSV returns a writer that puts one table into one named file.
//
// The name is given rather than derived from the entity, because the entity is
// what the records are called and the file is where they go. In this mode those
// two differ on purpose: the records keep the name of the source so that a
// person recognises them in the manifest, while the file carries a suffix so
// that reading and writing can never meet on one path.
func NewPlainCSV(files Files, name, delimiter string) (*PlainCSV, error) {
	separator, err := usableDelimiter(delimiter, "this run was given ")
	if err != nil {
		return nil, err
	}
	return &PlainCSV{files: files, name: name, delimiter: separator}, nil
}

// Begin opens the file and writes the header.
func (w *PlainCSV) Begin(table dataset.Table) error {
	if w.begun {
		// One file, one table. A second table would be written over the first,
		// and this mode has no name to give the second one.
		return fault.New(fault.Runtime, "this writer takes one table and was given a second, "+
			"called "+table.Entity)
	}
	file, err := w.files.Create(w.name)
	if err != nil {
		return err
	}
	w.begun = true
	w.lines = newLines(file, w.delimiter)
	return w.lines.write(table.Columns)
}

// Write puts one record on its line.
func (w *PlainCSV) Write(record dataset.Record) error {
	return w.lines.write(record.Values)
}

// End empties the buffer into the file and closes it.
func (w *PlainCSV) End(dataset.Table) error { return w.lines.close() }

// Finish has nothing to do: this writer streams, so the table was already
// complete when it ended.
func (w *PlainCSV) Finish() error { return nil }
