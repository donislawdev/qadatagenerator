package sourcecsv_test

import (
	"errors"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/sourcecsv"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// recordOf is a record of exactly size bytes, its line end included.
func recordOf(size int) string {
	return "x," + strings.Repeat("y", size-len("x,\n")) + "\n"
}

// 🔴 The limit is exact: a record of MaxRecordBytes is read, and one a byte
// longer is refused, naming the line it starts on.
//
// The allowance under the parser alone would leave a band of one buffer where a
// record past the limit is read or refused depending on where the buffer
// happened to stand, which is a limit nobody could state.
func TestARecordOfTheLimitIsReadAndOneByteLongerIsRefused(t *testing.T) {
	fits := written(t, "fits.csv", "a,b\n"+recordOf(sourcecsv.MaxRecordBytes)+"1,2\n")
	file, err := sourcecsv.Open(fits, ',', corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("a record of exactly the limit was refused: %v", err)
	}
	if rows := allRows(t, file); len(rows) != 2 {
		t.Errorf("a record of exactly the limit came out of the second pass as %d records", len(rows))
	}

	over := written(t, "over.csv", "a,b\n1,2\n"+recordOf(sourcecsv.MaxRecordBytes+1))
	_, err = sourcecsv.Open(over, ',', corrupt.NeverStops{})
	if err == nil {
		t.Fatal("a record a byte longer than the limit was read")
	}
	if fault.KindOf(err) != fault.Data || !strings.Contains(err.Error(), "starting on line 3") {
		t.Errorf("the refusal has to be about the data and name line 3, and is %v: %q",
			fault.KindOf(err), err)
	}
}

// 🔴 A quote opened and never closed is refused at the limit, and the memory
// it costs is the memory of the limit rather than of the file.
//
// Until 2026-10-27 the parser read the rest of the file into one record before
// it said a word about the quote: measured at six times the size of what was
// read, so a file of 16 MiB cost about a hundred, and one of a few gigabytes
// cost the machine. Through the allowance it reads one limit and a buffer.
func TestAnUnclosedQuoteIsRefusedAtTheLimitRatherThanAtTheEnd(t *testing.T) {
	const size = 16 << 20
	path := written(t, "quote.csv", "a,b\n1,\""+strings.Repeat("xxxxxxxxx\n", size/10))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := sourcecsv.Open(path, ',', corrupt.NeverStops{})
	runtime.ReadMemStats(&after)

	if err == nil || !strings.Contains(err.Error(), "starting on line 2") {
		t.Fatalf("the refusal has to name the line the quote is on, and is %q", err)
	}
	// Measured on go1.27.3: six to seven times what the parser was handed, so
	// a little over 6 MiB with the allowance and about 100 MiB without it. The
	// bound sits between the two with room on both sides.
	if spent := after.TotalAlloc - before.TotalAlloc; spent > 16*sourcecsv.MaxRecordBytes {
		t.Errorf("refusing a quote left open in a file of %d MiB allocated %d bytes, which is "+
			"the file being read rather than one record", size>>20, spent)
	}
}

// The line of a record too long counts the lines of the records before it,
// including a record whose quoted field spans lines.
func TestTheLineOfARecordTooLongCountsTheLinesBeforeIt(t *testing.T) {
	for _, given := range []struct {
		what    string
		content string
		line    string
	}{
		{"the header itself", strings.Repeat("h", sourcecsv.MaxRecordBytes) + "\n1\n", "line 1"},
		{"after a record spanning lines", "a,b\n1,\"one\ntwo\nthree\"\n" +
			recordOf(sourcecsv.MaxRecordBytes+1), "line 5"},
		{"nothing but separators", "a,b\n" + strings.Repeat(",", 2*sourcecsv.MaxRecordBytes) + "\n",
			"line 2"},
	} {
		_, err := sourcecsv.Open(written(t, "long.csv", given.content), ',', corrupt.NeverStops{})
		if err == nil || !strings.Contains(err.Error(), "starting on "+given.line) {
			t.Errorf("%s: the refusal has to name %s, and is %q", given.what, given.line, err)
		}
	}
}

// headerOf names columns c0, c1 and so on.
func headerOf(columns int) string {
	names := make([]string, columns)
	for at := range names {
		names[at] = "c" + strconv.Itoa(at)
	}
	return strings.Join(names, ",") + "\n"
}

// A file as wide as a spreadsheet is read, and one column wider is refused
// before anything is kept for its columns.
func TestAFileWiderThanASpreadsheetIsRefused(t *testing.T) {
	widest := sourcecsv.MaxColumns
	row := strings.Repeat(",", widest-1) + "\n"
	if _, err := sourcecsv.Open(written(t, "widest.csv", headerOf(widest)+row), ',',
		corrupt.NeverStops{}); err != nil {
		t.Fatalf("a file of exactly %d columns was refused: %v", widest, err)
	}

	_, err := sourcecsv.Open(written(t, "wider.csv", headerOf(widest+1)+","+row), ',',
		corrupt.NeverStops{})
	if err == nil {
		t.Fatalf("a file of %d columns was read", widest+1)
	}
	if fault.KindOf(err) != fault.Data || !strings.Contains(err.Error(), strconv.Itoa(widest+1)+" columns") {
		t.Errorf("the refusal has to be about the data and count the columns, and is %v: %q",
			fault.KindOf(err), err)
	}
}

// 🔴 A directory, a device or a pipe is refused by name.
//
// This mode reads its source twice. A device hands its bytes over for as long
// as somebody reads, and reading the console waits for somebody to type, so
// until 2026-10-27 a window handed one stopped answering every later read.
func TestASourceThatIsNotAFileIsRefused(t *testing.T) {
	device := "/dev/null"
	if runtime.GOOS == "windows" {
		device = "NUL"
	}
	for _, given := range []struct {
		path string
		says string
	}{
		{t.TempDir(), "is a directory"},
		{device, "is a device or a pipe"},
	} {
		_, err := sourcecsv.Open(given.path, ',', corrupt.NeverStops{})
		if err == nil || fault.KindOf(err) != fault.Data || !strings.Contains(err.Error(), given.says) {
			t.Errorf("%s: the refusal has to say it %s, and is %q", given.path, given.says, err)
		}
	}
}

// 🔴 A column keeps the values that fit its share of SampleBytes and nothing
// after the first that does not, so what is kept is the first values of it.
//
// infer says "the first N of M values" about a sample, and a column that went
// on taking shorter values after a long one would make that sentence false.
func TestAColumnKeepsItsFirstValuesUpToItsShare(t *testing.T) {
	const columns = 2048
	share := sourcecsv.SampleBytes / columns
	long := strings.Repeat("v", share/3)
	empties := strings.Repeat(",", columns-1)
	var file strings.Builder
	file.WriteString(headerOf(columns))
	for _, value := range []string{long, long, long, "short"} {
		file.WriteString(value + empties + "\n")
	}

	opened, err := sourcecsv.Open(written(t, "shares.csv", file.String()), ',', corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	sample := opened.Samples()[0]
	if len(sample.Values) != 2 || sample.NonEmpty != 4 {
		t.Fatalf("a column whose share holds two of its values kept %d of %d", len(sample.Values),
			sample.NonEmpty)
	}
	if sample.Rows[0] != 1 || sample.Rows[1] != 2 {
		t.Errorf("the values kept came from records %v rather than the first two", sample.Rows)
	}
}

// 🔴 A kept value does not keep its record.
//
// The parser cuts every field of a record out of one string, so until
// 2026-10-27 a short value kept in the sample kept the whole record it came
// from, and a thousand records of a megabyte were a gigabyte of sample.
func TestAKeptValueDoesNotKeepItsRecord(t *testing.T) {
	const columns, rows = 64, 16
	body := strings.Repeat("b", sourcecsv.MaxRecordBytes/2)
	tail := strings.Repeat(",", columns-2)
	var file strings.Builder
	file.WriteString(headerOf(columns))
	for row := range rows {
		file.WriteString(strconv.Itoa(row) + "," + body + tail + "\n")
	}
	path := written(t, "bodies.csv", file.String())
	file.Reset()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	opened, err := sourcecsv.Open(path, ',', corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("cannot open: %v", err)
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if kept := len(opened.Samples()[0].Values); kept != rows {
		t.Fatalf("the column of numbers kept %d values of %d", kept, rows)
	}
	// The bodies are each longer than the share of their column, so none of
	// them is kept, and the records only stay if the numbers keep them.
	if held := int64(after.HeapAlloc) - int64(before.HeapAlloc); held > sourcecsv.MaxRecordBytes {
		t.Errorf("sixteen numbers kept from records of half a megabyte hold %d bytes", held)
	}
	runtime.KeepAlive(opened)
}

// stopsAfter lets a survey read so many records and then answers that it has
// to stop.
type stopsAfter struct {
	left  int
	asked int
}

func (s *stopsAfter) Stopped() error {
	s.asked++
	if s.left == 0 {
		return errStopped
	}
	s.left--
	return nil
}

var errStopped = errors.New("stopped by the test")

// 🔴 A stop between records ends the survey, and the file is let go.
//
// The survey walks all of a file whose size somebody else chose, and until
// 2026-10-27 nothing could end it, so a window that had asked a newer question
// waited for the whole of the old one (B-341).
func TestAStopBetweenRecordsEndsTheSurveyAndLetsTheFileGo(t *testing.T) {
	path := written(t, "many.csv", "a\n"+strings.Repeat("1\n", 1000))
	stop := &stopsAfter{left: 10}
	if _, err := sourcecsv.Open(path, ',', stop); !errors.Is(err, errStopped) {
		t.Fatalf("a survey told to stop ended with %v", err)
	}
	if stop.asked != 11 {
		t.Errorf("the survey asked whether to stop %d times for ten records, and asks once a record",
			stop.asked)
	}
	// On Windows a file still open cannot be removed.
	if err := os.Remove(path); err != nil {
		t.Errorf("the file of a stopped survey is still held: %v", err)
	}
}
