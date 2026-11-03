package take

import (
	"errors"
	"os"
	"testing"
)

// 🔴 A read stopped after its records, while its columns are worked out, ends
// there and lets the file go.
//
// The reading asks before every record and once more to find the end, so a halt
// letting sourceRows + 1 asks through stops at the first ask of the working out.
// Until 2026-11-03 there was none, and a window that had asked a newer question
// waited for every column of the old file to be paired (B-356).
func TestAReadStoppedWhileItsColumnsAreWorkedOutEndsAndLetsTheFileGo(t *testing.T) {
	path := sourceFile(t, t.TempDir())
	stop := &stopAfter{records: sourceRows + 1}
	opened, err := Open(path, Defaults(), stop)
	if !errors.Is(err, errStoppedForTheTest) {
		t.Fatalf("a read told to stop while its columns were worked out ended with %v", err)
	}
	if opened != nil {
		t.Error("a stopped read handed back a file, so a screen would show a shape nobody " +
			"finished working out")
	}
	if stop.asked != sourceRows+2 {
		t.Errorf("the read asked whether to stop %d times, and the first ask after the "+
			"reading should have ended it", stop.asked)
	}
	// On Windows a file still open cannot be removed.
	if err := os.Remove(path); err != nil {
		t.Errorf("the file of a read stopped while its columns were worked out is still held: %v", err)
	}
}
