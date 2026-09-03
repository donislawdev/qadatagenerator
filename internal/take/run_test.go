package take

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
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

// wideFile writes a file of the given width, which is the thing the account of
// the columns is bounded against.
func wideFile(t *testing.T, dir string, columns int) string {
	t.Helper()
	var text strings.Builder
	for column := range columns {
		if column > 0 {
			text.WriteByte(',')
		}
		text.WriteString("c" + strconv.Itoa(column))
	}
	text.WriteByte('\n')
	for row := 1; row <= 3; row++ {
		for column := range columns {
			if column > 0 {
				text.WriteByte(',')
			}
			text.WriteString("v" + strconv.Itoa(row) + "_" + strconv.Itoa(column))
		}
		text.WriteByte('\n')
	}
	path := filepath.Join(dir, "wide.csv")
	if err := os.WriteFile(path, []byte(text.String()), 0o600); err != nil {
		t.Fatalf("cannot write the wide file this test damages: %v", err)
	}
	return path
}

// The account of the columns stops at a width, and the width of the file is
// recorded whole either way.
//
// 🔴 Both halves are the test. An entry costs about two hundred bytes, so a file
// wide enough turned this one section into a manifest larger than this build
// reads back: measured on 2026-09-03, 260 000 columns produced 50.2 MB and
// verify refused it with not one defect entry in it. Bounding the account fixes
// that and introduces the failure guarded on the other side: a limit that bit
// on files people really have would be a cut rather than a bound, so a file
// narrower than the limit has every one of its columns described.
//
// The count is what tells the two apart. Without it a truncated account reads
// as a complete account of a narrower file and nothing in the document says
// otherwise.
func TestTheAccountOfTheColumnsStopsAtAWidthAndSaysHowWideTheFileWas(t *testing.T) {
	for _, given := range []struct {
		what     string
		columns  int
		describe int
	}{
		{"a file narrower than the limit has every column described",
			8, 8},
		{"a file exactly at the limit has every column described",
			manifest.ColumnsDescribed, manifest.ColumnsDescribed},
		{"a file one column wider stops at the limit",
			manifest.ColumnsDescribed + 1, manifest.ColumnsDescribed},
	} {
		t.Run(given.what, func(t *testing.T) {
			dir := t.TempDir()
			settings := usable()
			settings.Out = filepath.Join(dir, "out")

			opened, err := Open(wideFile(t, dir, given.columns), settings)
			if err != nil {
				t.Fatalf("the file could not be read: %v", err)
			}
			defer func() { _ = opened.Close() }()

			worked := opened.inference()

			if worked.ColumnsTotal != given.columns {
				t.Errorf("the manifest says the file has %d columns, and it has %d",
					worked.ColumnsTotal, given.columns)
			}
			if len(worked.Columns) != given.describe {
				t.Fatalf("the manifest describes %d columns, want %d",
					len(worked.Columns), given.describe)
			}
			// A bound that dropped the count as well would leave a reader
			// unable to tell this from a complete account, which is the whole
			// reason the count is required. The envelope is asked rather than
			// the section, because that is the road every manifest takes.
			envelope := manifest.Run{
				GeneratedAt: "2026-09-03T00:00:00Z", Seed: "s", Locale: "en-US",
				CatalogVersion: "0.1", Source: "wide.csv", Inference: worked,
				RowsTotal: 3, RowsMutated: 0,
			}
			for _, problem := range envelope.Problems() {
				t.Errorf("the model refuses what this run produced: %s", problem)
			}
		})
	}
}
