// Package sourcecsv reads somebody else's comma separated file.
//
// # Why reading is the standard library and writing is not
//
// The writer in the format package is deliberately hand written, because the
// bytes it produces are a promise: line endings, quoting and encoding are
// decided by us and never by the machine the tool happens to run on. None of
// that applies here. This side has to accept whatever a file already contains,
// including quoted fields carrying the delimiter and records spanning several
// lines, and the encoder in the standard library is a careful implementation of
// exactly that.
//
// Lazy quotes stay off. A file whose quoting is broken is refused by name
// rather than read into whatever the salvage would produce, because a value
// silently mangled on the way in becomes damage this tool did not order and
// does not describe.
//
// # Two passes, and the reason is in DEC-155
//
// Three things have to be known before the first row can be damaged: how many
// rows there are, because the manifest envelope requires the count and a set
// level defect needs a second row to span, and what the columns hold, because
// nothing declared it. All three need rows that come later than the first one.
//
// So the first pass counts and samples, and the second pass damages. Between
// them this holds a count and a bounded sample per column, and not one record.
// The cost is stated rather than hidden: the file is read twice, and a stream
// that cannot be rewound therefore cannot be a source.
package sourcecsv

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// SampleCap is how many values of one column the first pass keeps.
//
// # Why there is a cap at all, and why the count beside it has none
//
// Working out what a column holds means running validators over its values, and
// the file is one somebody else chose the size of. Ten million rows across ten
// columns is five hundred million validations for an answer that a thousand
// values already settle beyond argument: the weakest check digit in this build
// leaves a one in ten chance per value, so eight values put a wrong guess below
// one in a hundred million.
//
// The row count is not capped and cannot be, because the manifest envelope
// carries a number rather than an estimate.
const SampleCap = 1000

// byteOrderMark is what a file written by a spreadsheet often starts with.
//
// Written as an escape rather than as the character, because Go refuses one in
// the middle of a source file and would refuse this one.
const byteOrderMark = "\ufeff"

// File is somebody else's dataset, opened for reading twice.
type File struct {
	path      string
	delimiter rune
	table     dataset.Table
	samples   []dataset.Sample
	// hadByteOrderMark records that the file started with one. It is reported
	// rather than reproduced: this tool writes UTF-8 without one, so a run that
	// dropped it in silence would have changed the file in a way the manifest
	// does not describe.
	hadByteOrderMark bool

	reading *os.File
	rows    *csv.Reader
	row     int
}

// Open reads the whole file once and returns what it found.
//
// It leaves nothing open. The second pass opens the file again, so a caller
// that surveys a file and then decides not to damage it holds no handle.
func Open(path string, delimiter rune) (*File, error) {
	file := &File{path: path, delimiter: delimiter}
	if err := file.survey(); err != nil {
		return nil, err
	}
	return file, nil
}

// Table is what was read: the entity, its columns and how many records it
// holds.
func (f *File) Table() dataset.Table { return f.table }

// Samples is what each column held, in the order of the columns.
func (f *File) Samples() []dataset.Sample { return f.samples }

// HadByteOrderMark reports whether the file began with one.
func (f *File) HadByteOrderMark() bool { return f.hadByteOrderMark }

// Next returns the next record, and false when the file is exhausted.
//
// The second pass is opened here rather than in Open, so that the handle exists
// only while records are being taken.
func (f *File) Next() (dataset.Record, bool, error) {
	if f.rows == nil {
		if err := f.startReading(); err != nil {
			return dataset.Record{}, false, err
		}
	}
	fields, err := f.rows.Read()
	if errors.Is(err, io.EOF) {
		if f.row < f.table.Rows {
			return dataset.Record{}, false, f.changedUnderneath(
				"it held " + strconv.Itoa(f.table.Rows) + " records when it was surveyed and " +
					strconv.Itoa(f.row) + " when it was read")
		}
		return dataset.Record{}, false, f.Close()
	}
	if err != nil {
		// The handle goes before the refusal travels. A walk that ends in a
		// failure ends here, and leaving it open holds the file against whoever
		// tries to clean up after the run.
		_ = f.Close()
		return dataset.Record{}, false, f.readFailure(err)
	}
	f.row++
	// Caught here rather than at the end, because a record past the count is a
	// record the writer would already have written.
	if f.row > f.table.Rows {
		return dataset.Record{}, false, f.changedUnderneath(
			"it held " + strconv.Itoa(f.table.Rows) + " records when it was surveyed and has " +
				"more than that now")
	}
	return dataset.Record{Values: fields, Parent: dataset.NoParent}, true, nil
}

// changedUnderneath is the refusal for a file that is not the one that was
// surveyed.
//
// # Why this exists and what it does not buy
//
// 🔴 Reading twice is stated in DEC-155 as a cost, and until this refusal
// existed it was also a hole. Nothing compared the second reading with the
// first, so a file replaced in between produced a manifest describing the rows
// of one file with the columns and the count of another, an output file whose
// rows did not match its own header, and an exit code of zero. Measured on
// 2026-08-11: a survey of two columns and three records, then a replacement,
// then one record of three values handed over without a word.
//
// The window is not narrow. It runs from the survey to the first record, which
// is the whole of working out what the columns hold and planning what to damage.
//
// 🔴 What this does NOT do, said here so that nobody reads it as more: it does
// not close the race. A file whose header and record count are unchanged and
// whose values are different passes every check here, because nothing short of
// holding the file open across both passes could see that, and holding it open
// is a different design with its own costs. This closes the loud half.
func (f *File) changedUnderneath(what string) error {
	// The handle goes first. This refusal ends the walk, and a file left open
	// cannot be removed on Windows, which turns one failure into two.
	_ = f.Close()
	return fault.New(fault.Data, f.path+" changed while it was being read: "+what+
		". This mode reads the file twice, because the number of records and what the columns "+
		"hold have to be known before the first record can be damaged, so a file that is still "+
		"being written cannot be a source. Nothing was written").
		DoInstead("run this again once whatever produces that file has finished")
}

// Close releases the handle of the second pass. It is safe to call more than
// once, because the ordinary end of a walk closes and a caller that stopped
// early closes as well.
func (f *File) Close() error {
	if f.reading == nil {
		return nil
	}
	handle := f.reading
	f.reading, f.rows = nil, nil
	if err := handle.Close(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot close "+f.path)
	}
	return nil
}

// survey is the first pass: it counts the rows and keeps a bounded sample of
// every column.
func (f *File) survey() error {
	handle, reader, header, err := f.openAndReadHeader()
	if err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()

	f.table = dataset.Table{Entity: EntityOf(f.path), Columns: header}
	f.samples = make([]dataset.Sample, len(header))
	for at, name := range header {
		f.samples[at] = dataset.Sample{Field: name}
	}

	rows := 0
	for {
		fields, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return f.readFailure(err)
		}
		rows++
		f.gather(fields, rows)
	}
	f.table.Rows = rows
	if rows == 0 {
		return fault.New(fault.Data, f.path+" has a header and no records. There is nothing to "+
			"damage, so nothing was written").
			DoInstead("give a file with at least one row of data")
	}
	return nil
}

// gather records one row into the samples.
//
// The record number travels with the value, because a column with a gap in it
// keeps fewer values than the rows above it and anything reading two columns
// side by side would otherwise compare one record against another.
func (f *File) gather(fields []string, row int) {
	for at := range f.samples {
		if at >= len(fields) {
			continue
		}
		value := fields[at]
		if strings.TrimSpace(value) == "" {
			f.samples[at].Empty++
			continue
		}
		f.samples[at].NonEmpty++
		if len(f.samples[at].Values) < SampleCap {
			f.samples[at].Values = append(f.samples[at].Values, value)
			f.samples[at].Rows = append(f.samples[at].Rows, row)
		}
	}
}

// startReading opens the file for the second pass and steps over the header.
//
// The header of the second reading is compared with the first rather than
// discarded. Discarding it was the hole described at changedUnderneath: the
// columns a manifest names come from the first reading and the values come from
// the second, so two different headers mean the two describe different files.
func (f *File) startReading() error {
	handle, reader, header, err := f.openAndReadHeader()
	if err != nil {
		return err
	}
	if !sameColumns(header, f.table.Columns) {
		_ = handle.Close()
		return f.changedUnderneath("its columns were " + strings.Join(f.table.Columns, ", ") +
			" when it was surveyed and are " + fault.Shorten(strings.Join(header, ", ")) + " now")
	}
	f.reading, f.rows, f.row = handle, reader, 0
	return nil
}

// sameColumns reports whether two headers name the same columns in the same
// order.
func sameColumns(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for at := range left {
		if left[at] != right[at] {
			return false
		}
	}
	return true
}

// openAndReadHeader opens the file and takes its first record as the column
// names, which is the one thing both passes do the same way.
func (f *File) openAndReadHeader() (*os.File, *csv.Reader, []string, error) {
	handle, err := os.Open(f.path)
	if err != nil {
		return nil, nil, nil, fault.Wrap(fault.IO, err, "cannot read "+f.path)
	}
	reader := csv.NewReader(bufio.NewReader(handle))
	reader.Comma = f.delimiter
	// Every record has to have as many fields as the header. The reader
	// enforces it once FieldsPerRecord is left at zero and the header has been
	// read, which turns a ragged file into a refusal naming the line rather
	// than into rows whose columns have quietly shifted.
	reader.FieldsPerRecord = 0
	reader.LazyQuotes = false
	// Reusing the slice would hand the same backing array to every record, and
	// the corruption engine writes into the record it is given.
	reader.ReuseRecord = false

	header, err := reader.Read()
	if errors.Is(err, io.EOF) {
		_ = handle.Close()
		return nil, nil, nil, fault.New(fault.Data, f.path+" is empty. This mode reads a header "+
			"row and then the records under it, so give a file that has both")
	}
	if err != nil {
		_ = handle.Close()
		return nil, nil, nil, f.readFailure(err)
	}
	if problem := usableHeader(header, f.path); problem != nil {
		_ = handle.Close()
		return nil, nil, nil, problem
	}
	if strings.HasPrefix(header[0], byteOrderMark) {
		f.hadByteOrderMark = true
		header[0] = strings.TrimPrefix(header[0], byteOrderMark)
	}
	return handle, reader, header, nil
}

// usableHeader refuses a header this mode cannot name columns from.
//
// A column with no name cannot be reported in a manifest location, and two
// columns with one name would make every location ambiguous. Both are refused
// rather than repaired, because repairing them invents a name that is not in
// the file the reader is looking at.
//
// 🔴 The name goes into the sentence through Shorten, because it is untrusted
// text of unbounded length. A file has no size limit here and cannot have one,
// since this mode exists for large ones, so a column name may be as long as the
// file. Measured on 2026-08-11 before this: a name of sixty four kilobytes came
// back as a refusal of sixty five thousand nine hundred and twenty seven bytes,
// straight into somebody's build log. Same class as the report that grew with a
// hostile recipe, reached by the third road.
func usableHeader(header []string, path string) error {
	seen := make(map[string]bool, len(header))
	for at, name := range header {
		clean := strings.TrimPrefix(strings.TrimSpace(name), byteOrderMark)
		if clean == "" {
			return fault.New(fault.Data, path+" has a column with no name, at position "+
				strconv.Itoa(at+1)+". Every column needs a name, because the manifest says which "+
				"column each defect landed in").
				DoInstead("add a header name and run this again")
		}
		if seen[clean] {
			return fault.New(fault.Data, path+" has two columns called "+fault.Shorten(clean)+
				". Names have to differ, because a manifest naming that column would point at "+
				"either of them").
				DoInstead("rename one of them and run this again")
		}
		seen[clean] = true
	}
	return nil
}

// readFailure turns a refusal from the parser into ours, keeping what it said
// about the line.
//
// The wording of the library names the line and the column, which is the part
// worth keeping, and it is the only part somebody can act on.
func (f *File) readFailure(err error) error {
	var badField *csv.ParseError
	if errors.As(err, &badField) {
		// 🔴 The move says what to do and never which control does it, because
		// it travels to both surfaces and the window has no flags. It named one
		// until 2026-09-03.
		return fault.New(fault.Data, f.path+" cannot be read as comma separated data: "+
			err.Error()).
			DoInstead("check the quoting on that line, or say which delimiter this file uses")
	}
	return fault.Wrap(fault.IO, err, "cannot read "+f.path)
}

// EntityOf is what the records of a file are called.
//
// The base name without its extension, because that is the word the person
// running this already uses for the data and the one they will look for in the
// manifest. It is not the name of the file that gets written, which carries a
// suffix so that reading and writing cannot meet on one path.
func EntityOf(path string) string {
	base := filepath.Base(path)
	if dot := strings.LastIndex(base, "."); dot > 0 {
		base = base[:dot]
	}
	return base
}
