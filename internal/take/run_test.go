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

// A run says which of the files it wrote this build will not read back.
//
// 🔴 This is worked out here rather than on either surface because it used to
// be worked out on the command line alone, and the window therefore said
// nothing: somebody damaging a wide file in the window got their files, correct
// counts, and a refusal from another screen days later with nothing connecting
// the two. Both surfaces now read the same answer from the same run.
//
// The files are made by length rather than by a run wide enough to produce a
// real one, because what is under test is which files are named and on what
// grounds, not the arithmetic that reaches the size.
func TestARunSaysWhichFilesWillNotBeReadBack(t *testing.T) {
	for _, given := range []struct {
		what    string
		sizes   map[string]int64
		written []string
		want    []string
	}{
		{"nothing oversized",
			map[string]int64{"a.corrupted.csv": 10, "expectation-manifest.json": 10},
			[]string{"a.corrupted.csv", "expectation-manifest.json"}, nil},
		{"a manifest one byte past the limit",
			map[string]int64{"a.corrupted.csv": 10, "expectation-manifest.json": ReadBackLimit + 1},
			[]string{"a.corrupted.csv", "expectation-manifest.json"},
			[]string{"expectation-manifest.json"}},
		{"a manifest exactly at the limit is not named",
			map[string]int64{"expectation-manifest.json": ReadBackLimit},
			[]string{"expectation-manifest.json"}, nil},
		// The damaged file is the one somebody asked for and nothing reads it
		// back as a manifest, so its size is not this warning's business however
		// large it gets.
		{"a damaged file larger than the limit",
			map[string]int64{"a.corrupted.csv": ReadBackLimit + 1},
			[]string{"a.corrupted.csv"}, nil},
		{"a file that is not there at all",
			map[string]int64{}, []string{"expectation-manifest.json"}, nil},
	} {
		t.Run(given.what, func(t *testing.T) {
			dir := t.TempDir()
			for name, size := range given.sizes {
				file, err := os.Create(filepath.Join(dir, name))
				if err != nil {
					t.Fatalf("cannot make %s: %v", name, err)
				}
				if err := file.Truncate(size); err != nil {
					t.Fatalf("cannot give %s its length: %v", name, err)
				}
				if err := file.Close(); err != nil {
					t.Fatalf("cannot close %s: %v", name, err)
				}
			}

			found := oversized(dir, given.written)

			if len(found) != len(given.want) {
				t.Fatalf("%s named %v, want %v", given.what, found, given.want)
			}
			for i, one := range found {
				if one.Name != given.want[i] {
					t.Errorf("named %q at %d, want %q", one.Name, i, given.want[i])
				}
				if one.Bytes != given.sizes[one.Name] {
					t.Errorf("said %s is %d bytes, and it is %d",
						one.Name, one.Bytes, given.sizes[one.Name])
				}
			}
		})
	}
}
