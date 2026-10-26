package format

import (
	"bufio"
	"strconv"
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
	entity  recipe.Entity
	planned dataset.Table
	file    WriteCloser
	lines   lines
	rows    int
	column  string
}

// NewCSV returns a writer for the comma separated form.
//
// The delimiter is what the recipe asked for, defaulting to a comma. It is an
// explicit setting rather than a detail because this is exactly where an import
// falls over, and the same difference on the other side of the product is a
// mutator.
func NewCSV(files Files, rec recipe.Recipe) (*CSV, error) {
	delimiter, err := usableDelimiter(rec.Output.Delimiter, "the recipe gives ")
	if err != nil {
		return nil, err
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
		lines: newLines(file, w.delimiter), column: column}
	header := planned.Columns
	if column != "" {
		header = append(append([]string{}, header...), column)
	}
	return w.current.lines.write(header)
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
	return w.current.lines.write(line)
}

// End flushes and closes the file of one table.
func (w *CSV) End(dataset.Table) error {
	if err := w.current.lines.flush(); err != nil {
		return err
	}
	w.current = nil
	return nil
}

// Finish has nothing to do: this writer streams, so every table was already
// complete when it ended.
func (w *CSV) Finish() error { return nil }

// lines writes rows of one file with the quoting and the line ending this
// project promises.
//
// # Why this is a type of its own
//
// Two writers produce comma separated bytes: the one above, which lays out a
// dataset a recipe described, and the one beside it, which writes back a table
// that was read from somebody else's file. What they share is exactly the part
// that is a promise rather than an arrangement, and a second copy of the
// quoting rules is a second answer to a question the format has one answer to.
type lines struct {
	buffered  *bufio.Writer
	delimiter string
}

func newLines(to WriteCloser, delimiter string) lines {
	return lines{buffered: bufio.NewWriter(to), delimiter: delimiter}
}

// write puts one row down, quoting the fields that need it.
//
// # Why a row of one empty field is written as two quotes
//
// 🔴 Written as it stands, it is an empty line, and a reader of this format
// skips an empty line rather than reading a record from it. A table of one
// column with blank values came back with fewer records than the run said it
// wrote, and corrupt refused the file as having a header and no rows. Found by
// an outside review on 2026-10-22. Two quotes are an empty field to every
// reader and cannot be taken for the end of a line.
func (l lines) write(fields []string) error {
	var line strings.Builder
	if len(fields) == 1 && fields[0] == "" {
		line.WriteString(`""`)
	}
	for index, field := range fields {
		if index > 0 {
			line.WriteString(l.delimiter)
		}
		line.WriteString(l.quote(field))
	}
	// A single newline, always, whatever the system. The mixed endings this
	// tool produces on purpose belong to a mutator and arrive as data rather
	// than from the platform.
	line.WriteByte('\n')
	if _, err := l.buffered.WriteString(line.String()); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	return nil
}

// quote wraps a field when it carries something that would end it early.
func (l lines) quote(field string) string {
	if !strings.ContainsAny(field, l.delimiter+"\"\r\n") {
		return field
	}
	return `"` + strings.ReplaceAll(field, `"`, `""`) + `"`
}

func (l lines) flush() error {
	if err := l.buffered.Flush(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	return nil
}

// Delimiter is the character this build separates values with for a recipe.
//
// It is exported so that whoever reads those bytes back reads them the way they
// were written. A caller holding its own comma is a second answer to a question
// the recipe is allowed to change, and the two would agree until the first
// recipe that sets the setting.
func Delimiter(rec recipe.Recipe) (rune, error) {
	given, err := usableDelimiter(rec.Output.Delimiter, "the recipe gives ")
	if err != nil {
		return 0, err
	}
	return []rune(given)[0], nil
}

// usableDelimiter refuses a delimiter the format cannot be split on, and
// defaults to a comma.
//
// The rule is dataset.UsableSeparator, the one the recipe and the reader of
// somebody else's file ask as well. It stays here although the recipe is now
// refused earlier, because a recipe built by a screen rather than read from a
// file reaches this writer without passing the reader of recipes.
func usableDelimiter(given, from string) (string, error) {
	if given == "" {
		return ",", nil
	}
	if _, usable := dataset.UsableSeparator(given); !usable {
		return "", fault.New(fault.Config, "the delimiter has to be one character other than "+
			"a quote or a line end, and "+from+strconv.Quote(given))
	}
	return given, nil
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
			encoding).
			DoInstead("remove the setting, or ask for utf-8")
	}
}
