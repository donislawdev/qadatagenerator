package sourcecsv

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// MaxRecordBytes is the longest record this reads, counted from the end of the
// record before it to the end of its own line.
//
// # Why there is a limit at all
//
// The parser builds a whole record before it hands it over, and the file is one
// somebody else chose the contents of. A quote opened and never closed turns
// the rest of the file into one record, so until 2026-10-27 a file of any size
// with a stray quote on its second line was read into memory whole before the
// parser said a word about the quote.
//
// # Why it is this one
//
// Measured on go1.27.3 with nothing but the parser, four runs with the first
// dropped. A record costs about seven times its size in allocations when it is
// one long field and 145 to 164 times when it is nothing but separators,
// because the parser keeps an index, a position and a string header for every
// field and counts them against the header only once the record is built. That
// is a peak working set of about 100 MiB at 1 MiB of separators and 320 MiB at
// 4 MiB, on a machine that has been measured with 1.7 GB free.
//
// From below: a cell of a spreadsheet holds at most 32,767 UTF-16 characters,
// which is at most 98,301 bytes of UTF-8 and two quotes, so a record of this
// size holds ten full cells.
//
// Empty lines before a record count towards it, because the parser steps over
// them on the way to the record. A file would need half a million of them in a
// row to be refused for that.
//
// It stays a whole number of mebibytes, because the refusal says it in those.
const MaxRecordBytes = 1 << 20

// MaxColumns is the widest file this reads. It is the widest sheet a
// spreadsheet holds, so a file exported from one always fits.
//
// It bounds what is kept for every column and the fields of every record. It
// does not bound the time it takes to work out what the columns hold. Pairing
// a column of identifiers with the others still grows with the square of the
// width for the columns with a blank among their first records, and only for
// those (B-344).
const MaxColumns = 16384

// SampleBytes is how much of the file the first pass keeps for all the columns
// together, shared out equally between them.
//
// # Why bytes as well as SampleCap
//
// SampleCap bounds how many values are kept and says nothing about how long
// they are, and until 2026-10-27 every value kept was a slice of the record it
// came from, which keeps the whole record alive. A thousand records of a
// megabyte each were a gigabyte of sample for a column of check digits.
//
// # Why this much
//
// A PESEL number costs 35 bytes kept, with valueCost, so every column keeps the
// full SampleCap of them up to 479 columns, and the widest file this reads still
// keeps 29 a column, over three times the eight that infer.MinimumEvidence asks
// for. The slices holding them may grow to twice what they hold, so the sample
// stays under twice this.
const SampleBytes = 16 << 20

// valueCost is what one kept value costs besides its own bytes: the header of
// the string and the number of its record, 24 bytes on a 64-bit build.
const valueCost = 24

// readAhead is the buffer between the file and the parser. It is the size
// bufio.NewReader chooses, which is what the parser read through before the
// allowance below existed, so the reading itself is unchanged.
const readAhead = 4096

// errTooLong is what the allowance answers once a record has used all of its.
var errTooLong = errors.New("the record is longer than the limit")

// allowance is what the parser reads the file through.
//
// It hands out no byte past the start of the record being read plus
// MaxRecordBytes and one buffer, so a record too long to keep is refused while
// the parser is still building it rather than after. The buffer is in the
// allowance because what the parser has buffered may already reach past the
// start of the record, and a record inside the limit must never be refused
// for it.
type allowance struct {
	file   io.Reader
	handed int64
	until  int64
}

// Read is io.Reader, cut short at the end of the allowance.
func (a *allowance) Read(into []byte) (int, error) {
	room := a.until - a.handed
	if room <= 0 {
		return 0, errTooLong
	}
	if int64(len(into)) > room {
		into = into[:room]
	}
	read, err := a.file.Read(into)
	a.handed += int64(read)
	return read, err
}

// from moves the allowance to the record that starts at offset.
func (a *allowance) from(offset int64) { a.until = offset + MaxRecordBytes + readAhead }

// pass is one reading of the file from its first line.
type pass struct {
	handle  *os.File
	parser  *csv.Reader
	allowed *allowance
	// line is the line the next record starts on, counting the first as one.
	// Only the refusal of a record too long reads it, because the parser names
	// the line itself in every other refusal.
	line int
}

// newPass starts reading an open file, with every record held to the limit.
func newPass(handle *os.File, delimiter rune) *pass {
	allowed := &allowance{file: handle}
	parser := csv.NewReader(bufio.NewReaderSize(allowed, readAhead))
	parser.Comma = delimiter
	// Every record has to have as many fields as the header. The reader
	// enforces it once FieldsPerRecord is left at zero and the header has been
	// read, which turns a ragged file into a refusal naming the line rather
	// than into rows whose columns have quietly shifted.
	parser.FieldsPerRecord = 0
	parser.LazyQuotes = false
	// Reusing the slice would hand the same backing array to every record, and
	// the corruption engine writes into the record it is given.
	parser.ReuseRecord = false
	return &pass{handle: handle, parser: parser, allowed: allowed, line: 1}
}

// next is the next record, or errTooLong once it runs past MaxRecordBytes.
//
// The allowance bounds the memory and the comparison after the record makes
// the limit exact: a record of MaxRecordBytes is read and one a byte longer is
// refused, wherever the buffer happened to stand.
func (p *pass) next() ([]string, error) {
	start := p.parser.InputOffset()
	p.allowed.from(start)
	fields, err := p.parser.Read()
	if errors.Is(err, errTooLong) || err == nil && p.parser.InputOffset()-start > MaxRecordBytes {
		return nil, errTooLong
	}
	if err != nil {
		return fields, err
	}
	// The last field may carry line ends of its own when it is quoted, and the
	// record ends on the line its last one is on.
	last := len(fields) - 1
	at, _ := p.parser.FieldPos(last)
	p.line = at + strings.Count(fields[last], "\n") + 1
	return fields, nil
}

// regular refuses anything but a regular file: a directory, a device or a pipe.
//
// This mode reads its source twice, so a source that hands its bytes over once
// cannot be one, and a device can hand them over for ever. Measured on Windows
// on 2026-10-27: NUL and CON are character devices and a directory is a
// directory, all three were opened without a word, and reading CON waits for
// somebody to type.
//
// A path that cannot be looked at is left to the opening and the reading that
// follow, so that a file that does not exist is refused in the words it always
// was.
func (f *File) regular(info os.FileInfo, err error) error {
	if err != nil || info.Mode().IsRegular() {
		return nil
	}
	what := "a device or a pipe"
	if info.IsDir() {
		what = "a directory"
	}
	return fault.New(fault.Data, f.path+" is "+what+" rather than a file. This mode reads its "+
		"source twice, once to count the records and once to damage them, so the source has to "+
		"be a file that is still there the second time").
		DoInstead("give the path of a comma separated file")
}

// tooWide refuses a header naming more columns than MaxColumns.
func (f *File) tooWide(columns int) error {
	return fault.New(fault.Data, f.path+" has "+strconv.Itoa(columns)+" columns, and this mode "+
		"reads at most "+strconv.Itoa(MaxColumns)+", the widest sheet a spreadsheet holds").
		DoInstead("split the file into narrower ones, or leave out the columns nobody needs")
}

// tooLong refuses a record longer than MaxRecordBytes, naming the line it
// starts on.
//
// The line is the useful part. A quote opened and never closed is what makes a
// record this long in a file somebody did not mean to break, and the quote is
// on the line the record starts on.
func (f *File) tooLong(line int) error {
	limit := strconv.Itoa(MaxRecordBytes>>20) + " MiB"
	return fault.New(fault.Data, f.path+" has a record starting on line "+strconv.Itoa(line)+
		" that is longer than "+limit+". This mode reads at most "+limit+" as one record, so "+
		"that a broken file cannot take all of the memory. A quote opened and never closed "+
		"turns the rest of the file into one record, which makes that line the first place to look").
		DoInstead("close the quote on that line, or split the record into shorter ones")
}
