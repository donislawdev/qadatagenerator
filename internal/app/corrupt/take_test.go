package corrupt_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// readFile is a dataset that already exists, handed over row by row.
type readFile struct {
	table   dataset.Table
	records []dataset.Record
	at      int
	failAt  int
}

func (r *readFile) Table() dataset.Table { return r.table }

func (r *readFile) Next() (dataset.Record, bool, error) {
	if r.failAt > 0 && r.at == r.failAt {
		return dataset.Record{}, false, fault.New(fault.Data, "the file stopped making sense")
	}
	if r.at >= len(r.records) {
		return dataset.Record{}, false, nil
	}
	record := r.records[r.at]
	r.at++
	return record, true, nil
}

// wrote records what a writer was told, so that a test can ask what reached it
// rather than what was meant to.
type wrote struct {
	begun    []string
	records  [][]string
	ended    []string
	finished int
	failOn   string
}

func (w *wrote) Begin(table dataset.Table) error {
	w.begun = append(w.begun, table.Entity)
	if w.failOn == "begin" {
		return fault.New(fault.IO, "the file could not be opened")
	}
	return nil
}

func (w *wrote) Write(record dataset.Record) error {
	w.records = append(w.records, record.Values)
	if w.failOn == "write" {
		return fault.New(fault.IO, "the disk went away")
	}
	return nil
}

func (w *wrote) End(table dataset.Table) error {
	w.ended = append(w.ended, table.Entity)
	if w.failOn == "end" {
		return fault.New(fault.IO, "the file could not be closed")
	}
	return nil
}

func (w *wrote) Finish() error {
	w.finished++
	if w.failOn == "finish" {
		return fault.New(fault.IO, "the last write failed")
	}
	return nil
}

// stopsAfter answers that the run may carry on until it has been asked a given
// number of times, and refuses from then on.
type stopsAfter struct {
	asks  int
	after int
}

func (s *stopsAfter) Stopped() error {
	s.asks++
	if s.asks > s.after {
		return fault.New(fault.Interrupted, "somebody pressed the key")
	}
	return nil
}

func fileOf(rows ...[]string) *readFile {
	records := make([]dataset.Record, 0, len(rows))
	for _, values := range rows {
		records = append(records, dataset.Record{Values: values, Parent: dataset.NoParent})
	}
	return &readFile{
		table:   dataset.Table{Entity: "records", Columns: []string{"a", "b"}, Rows: len(rows)},
		records: records,
	}
}

func TestEveryRecordOfTheFileReachesTheWriter(t *testing.T) {
	writer := &wrote{}
	err := corrupt.Run(fileOf([]string{"1", "x"}, []string{"2", "y"}), writer, corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("an ordinary walk failed: %v", err)
	}
	if len(writer.records) != 2 || writer.records[1][1] != "y" {
		t.Errorf("what reached the writer was %v", writer.records)
	}
	if len(writer.begun) != 1 || len(writer.ended) != 1 || writer.finished != 1 {
		t.Errorf("the table was begun %d times, ended %d and finished %d",
			len(writer.begun), len(writer.ended), writer.finished)
	}
}

// A file of one row is a legal thing to hand over, and the walk has to survive
// it. A defect about the integrity of a set has nobody to span there, which is
// answered elsewhere and must not become a failure here.
func TestAFileOfOneRecordIsWalkedLikeAnyOther(t *testing.T) {
	writer := &wrote{}
	if err := corrupt.Run(fileOf([]string{"1", "x"}), writer, corrupt.NeverStops{}); err != nil {
		t.Fatalf("a file of one record failed: %v", err)
	}
	if len(writer.records) != 1 {
		t.Errorf("1 record went in and %d came out", len(writer.records))
	}
}

// 🔴 A run stopped between two rows stops there, and the table is left without
// its End. That is deliberate: what a stopped run wrote never gets its real
// name, so an unfinished file is thrown away rather than shipped as a complete
// one. A run that finished the table on the way out would be shipping one.
func TestARunStoppedBetweenRowsLeavesTheTableUnfinished(t *testing.T) {
	writer := &wrote{}
	err := corrupt.Run(
		fileOf([]string{"1"}, []string{"2"}, []string{"3"}, []string{"4"}),
		writer,
		&stopsAfter{after: 2},
	)
	if err == nil {
		t.Fatal("a run that was told to stop carried on to the end")
	}
	if fault.KindOf(err) != fault.Interrupted {
		t.Errorf("the failure has to carry the kind that decides the exit code, and carries %v",
			fault.KindOf(err))
	}
	if len(writer.records) != 2 {
		t.Errorf("the run was stopped after 2 rows and wrote %d", len(writer.records))
	}
	if len(writer.ended) != 0 || writer.finished != 0 {
		t.Error("a stopped run ended or finished the table, so what it left looks complete")
	}
}

// The stop is asked before the first record as well, so that a run told to stop
// straight away writes nothing at all.
func TestARunStoppedBeforeTheFirstRecordWritesNothing(t *testing.T) {
	writer := &wrote{}
	if err := corrupt.Run(fileOf([]string{"1"}), writer, &stopsAfter{after: 0}); err == nil {
		t.Fatal("a run told to stop immediately carried on")
	}
	if len(writer.records) != 0 {
		t.Errorf("a run stopped before its first record wrote %d of them", len(writer.records))
	}
}

// A failure on the way in ends the run rather than being read as the end of the
// file. The two look the same to a loop that only asks whether there are more
// records, and this is the difference between a truncated dataset and a refusal.
func TestAFailureReadingEndsTheRun(t *testing.T) {
	source := fileOf([]string{"1"}, []string{"2"}, []string{"3"})
	source.failAt = 2
	writer := &wrote{}
	err := corrupt.Run(source, writer, corrupt.NeverStops{})
	if err == nil {
		t.Fatal("a file that stopped making sense was treated as a file that ended")
	}
	if !strings.Contains(err.Error(), "stopped making sense") {
		t.Errorf("the failure of the reader was replaced by %q", err)
	}
	if writer.finished != 0 {
		t.Error("the writer was finished after the reader failed, so the output looks complete")
	}
}

// Every failure the writer can have ends the run and comes back as it was. A
// loop that swallowed one of them would leave a file nobody knows is short.
func TestAFailureWritingEndsTheRunWhereverItHappens(t *testing.T) {
	for _, at := range []string{"begin", "write", "end", "finish"} {
		writer := &wrote{failOn: at}
		err := corrupt.Run(fileOf([]string{"1"}, []string{"2"}), writer, corrupt.NeverStops{})
		if err == nil {
			t.Errorf("a writer that failed at %s did not stop the run", at)
			continue
		}
		if fault.KindOf(err) != fault.IO {
			t.Errorf("the failure at %s came back as %v rather than as what the writer said",
				at, fault.KindOf(err))
		}
	}
}

// A writer that fails to begin is never handed a record. Writing into something
// that failed to open is how a second, confusing failure hides the first.
func TestAWriterThatCannotBeginIsNeverHandedARecord(t *testing.T) {
	writer := &wrote{failOn: "begin"}
	if err := corrupt.Run(fileOf([]string{"1"}), writer, corrupt.NeverStops{}); err == nil {
		t.Fatal("a writer that could not begin was carried on from")
	}
	if len(writer.records) != 0 {
		t.Errorf("%d records were written into a table that never opened", len(writer.records))
	}
}

// NeverStops exists so that a caller with nothing to interrupt it passes
// something rather than nil, which would be a crash at the first row.
func TestACallerWithNothingToStopItPassesSomething(t *testing.T) {
	if err := (corrupt.NeverStops{}).Stopped(); err != nil {
		t.Errorf("the halt for a caller that cannot be stopped answered %v", err)
	}
	var halt corrupt.Halt = corrupt.NeverStops{}
	if halt.Stopped() != nil {
		t.Error("it does not satisfy the port it exists for")
	}
}

// An error from the reader is passed on rather than wrapped again, so that the
// exit code comes from the kind the failure already carried.
func TestTheKindOfAFailureSurvivesTheWalk(t *testing.T) {
	source := fileOf([]string{"1"}, []string{"2"})
	source.failAt = 1
	err := corrupt.Run(source, &wrote{}, corrupt.NeverStops{})
	var carried fault.Kind
	if err != nil {
		carried = fault.KindOf(err)
	}
	if carried != fault.Data {
		t.Errorf("the kind reaching the caller is %v rather than the one the reader gave", carried)
	}
	if errors.Unwrap(err) != nil && fault.KindOf(errors.Unwrap(err)) == carried {
		t.Log("the failure is wrapped and still carries its kind, which is fine")
	}
}
