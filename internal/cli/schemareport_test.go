package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/take"
)

// wideFile writes a file of as many columns as asked for, with nothing in them
// that any scheme recognises, and returns the path.
//
// The values are deliberately plain. What is under test is the length of the
// report rather than what it concluded, and a column holding an identifier
// would make the number of lines depend on how well this build guesses.
func wideFile(t *testing.T, columns int) string {
	t.Helper()
	header := make([]string, 0, columns)
	row := make([]string, 0, columns)
	for at := 0; at < columns; at++ {
		header = append(header, "c"+strconv.Itoa(at))
		row = append(row, "v"+strconv.Itoa(at))
	}
	path := filepath.Join(t.TempDir(), "wide.csv")
	content := strings.Join(header, ",") + "\n" + strings.Join(row, ",") + "\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("cannot write the wide file: %v", err)
	}
	return path
}

// 🔴 The report of what a run worked out used to grow with the file it
// describes, one line per column, and the reason written above it was that a
// column nothing was recognised in is exactly the one somebody needs to know
// about. That reason is true of a file with twenty columns. Measured on one
// with five thousand, the log took 804 476 bytes in 5005 lines to describe
// 86 670 bytes of file, and nobody reads any of it.
//
// So this holds the threshold from both sides. The columns that are named stay
// named, the ones that are not are counted rather than dropped, and the whole
// of it stays in the manifest, which is where it lived for good all along.
func TestTheSchemaReportStopsGrowingWithTheFileItDescribes(t *testing.T) {
	const columns = 120
	got := run(commandCorrupt, "--layers", "L3", "--out", t.TempDir(), wideFile(t, columns))
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	lines := strings.Count(got.log, "\n")
	if lines > take.ColumnsListed+10 {
		t.Errorf("a file of %d columns put %d lines on the log, and the point of the "+
			"threshold is that this number stops following that one", columns, lines)
	}
	named := strings.Count(got.log, prefixDetail+"c")
	if named != take.ColumnsListed {
		t.Errorf("%d columns were named and the threshold says %d", named, take.ColumnsListed)
	}
	rest := strconv.Itoa(columns-take.ColumnsListed) + " more columns"
	if !strings.Contains(got.log, rest) {
		t.Errorf("the log does not count the ones it left out as %q: %q", rest, got.log)
	}
	if !strings.Contains(got.log, noteRestInManifest) {
		t.Errorf("the log leaves out %d columns without saying where they are: %q",
			columns-take.ColumnsListed, got.log)
	}
}

// A file nobody would call wide keeps every one of its columns named, which is
// the half of the rule that stops a threshold from turning into a cut.
func TestAFileOfOrdinaryWidthHasEveryColumnNamed(t *testing.T) {
	const columns = 12
	got := run(commandCorrupt, "--layers", "L3", "--out", t.TempDir(), wideFile(t, columns))
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	for at := 0; at < columns; at++ {
		if !strings.Contains(got.log, prefixDetail+"c"+strconv.Itoa(at)+" taken for") {
			t.Errorf("column c%d was not named: %q", at, got.log)
		}
	}
	if strings.Contains(got.log, "more columns") {
		t.Errorf("a file of %d columns had something counted instead of named: %q",
			columns, got.log)
	}
}

// 🔴 A dry run writes no manifest, so the line telling somebody where the rest
// of the columns are cannot be the line a real run uses. A message whose last
// sentence sends the reader to a file the run did not write is the family of
// failure this project has already paid for once, and this is the same shape.
func TestADryRunDoesNotSendTheReaderToAManifestItWillNotWrite(t *testing.T) {
	into := t.TempDir()
	got := run(commandCorrupt, "--layers", "L3", "--dry-run", "--out", into, wideFile(t, 120))
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	if strings.Contains(got.log, noteRestInManifest) {
		t.Errorf("a dry run points at a manifest as though it had written one: %q", got.log)
	}
	if !strings.Contains(got.log, noteRestInManifestOfARealRun) {
		t.Errorf("a dry run leaves out columns without saying how to see them: %q", got.log)
	}
	left, err := os.ReadDir(into)
	if err != nil {
		t.Fatalf("cannot read the output directory: %v", err)
	}
	if len(left) != 0 {
		t.Errorf("the dry run left %d files behind, so the sentence about a manifest "+
			"is not the only thing that would be wrong", len(left))
	}
}
