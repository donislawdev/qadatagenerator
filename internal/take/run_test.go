package take

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
)

// sourceRows is how many records the file every test here damages holds.
//
// Enough that a run can be stopped in the middle of it rather than before the
// first record or after the last, which is the whole subject of the test below.
const sourceRows = 20

// stopAfter is a halt that lets a run take a few records and then ends it.
//
// # Why counting the asks is the way to stop in the middle
//
// A run asks this once between every two records and nowhere else, which the
// use case states as its own rule. So a halt counting the asks stops after a
// known number of records, in the same place a person pressing a button would
// have stopped it, and it does that the same way every time. A stop timed by a
// clock would land somewhere different on every machine, which is the same as
// not knowing where it landed.
type stopAfter struct {
	records int
	asked   int
}

func (s *stopAfter) Stopped() error {
	s.asked++
	if s.asked <= s.records {
		return nil
	}
	return errStoppedForTheTest
}

// errStoppedForTheTest stands in for whatever a surface ends a run with.
var errStoppedForTheTest = os.ErrDeadlineExceeded

// sourceFile writes a file this tool did not make.
func sourceFile(t *testing.T, dir string) string {
	t.Helper()
	lines := []string{"id,town"}
	for row := 1; row <= sourceRows; row++ {
		lines = append(lines, strconv.Itoa(row)+",town "+strconv.Itoa(row))
	}
	path := filepath.Join(dir, "contacts.csv")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("cannot write the file this test damages: %v", err)
	}
	return path
}

// 🔴 A run made after one that was stopped damages the whole file rather than
// the part of it that was left.
//
// # What goes wrong without this, and why nothing looks wrong
//
// The reader goes over the file twice and the second pass is what a run
// consumes. A run stopped between two records leaves that pass part way
// through. If the run does not end the walk, the next run of the same opened
// file carries on from there: it writes a damaged file holding the records that
// were left, and a manifest whose envelope counts the whole file. There is no
// failure, no refusal and no count that disagrees with itself.
//
// # Why this could not happen before there was a window
//
// A command line run opens a file, damages it once and ends, so nothing walked
// twice. A screen keeps the file open while somebody reads what was worked out
// about it, and stopping a run and pressing the button again is the ordinary
// thing to do rather than an exotic one.
func TestARunMadeAfterAStoppedOneDamagesTheWholeFile(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir)
	settings := usable()
	settings.Out = filepath.Join(dir, "stopped")

	opened, err := Open(source, settings)
	if err != nil {
		t.Fatalf("the file could not be read: %v", err)
	}
	defer func() { _ = opened.Close() }()

	halfway := sourceRows / 2
	if _, err := opened.Corrupt(settings, loaded(t), &stopAfter{records: halfway},
		time.Now()); err == nil {
		t.Fatal("a run that was stopped ended as though nothing had stopped it")
	}

	settings.Out = filepath.Join(dir, "written")
	result, err := opened.Corrupt(settings, loaded(t), corrupt.NeverStops{}, time.Now())
	if err != nil {
		t.Fatalf("the run after the stopped one did not finish: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(settings.Out, result.Written[0]))
	if err != nil {
		t.Fatalf("the run left no damaged file: %v", err)
	}
	rows := len(strings.Split(strings.TrimSpace(string(written)), "\n")) - 1
	if rows != sourceRows {
		t.Errorf("the file written after a stopped run holds %d records and the file it came "+
			"from holds %d. A run that stopped left the reader part way through it",
			rows, sourceRows)
	}
}
