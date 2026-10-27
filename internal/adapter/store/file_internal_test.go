package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// A file its writer closed before the commit is committed like any other.
//
// 🔴 Writers close the file of a table when the table ends since 2026-10-27,
// and the set closes every file again when it commits. A second close of a file
// the system has already closed is an error, so a set that asked the system
// twice would refuse every run whose writer did the right thing.
func TestAFileClosedByItsWriterIsStillCommitted(t *testing.T) {
	dir := t.TempDir()
	set := New(dir, false)
	file, err := set.Create("a.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("id\n1\n")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("the first close failed: %v", err)
	}

	if err := set.Commit(); err != nil {
		t.Fatalf("a file its writer had closed stopped the commit: %v", err)
	}
	held, err := os.ReadFile(filepath.Join(dir, "a.csv"))
	if err != nil || string(held) != "id\n1\n" {
		t.Errorf("a.csv holds %q (%v)", held, err)
	}
}

// A close that failed when the writer asked still stops the commit, even when
// the writer said nothing about it.
//
// The failure is made by closing the file underneath, which is the one way to
// have the system refuse a close on every supported system: the close the
// writer then asks for is refused, and the set has to remember that rather
// than ask again and give the run its real names.
func TestACloseThatFailedStopsTheCommit(t *testing.T) {
	dir := t.TempDir()
	set := New(dir, false)
	file, err := set.Create("a.csv")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.held.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err == nil {
		t.Fatal("a close the system refused was answered as a success")
	}

	err = set.Commit()
	if err == nil {
		t.Fatal("a file that failed to close was given its real name")
	}
	if fault.KindOf(err) != fault.IO {
		t.Errorf("the commit refused with %v rather than %v: %v", fault.KindOf(err), fault.IO, err)
	}
	set.Discard()
	if _, err := os.Stat(filepath.Join(dir, "a.csv")); !os.IsNotExist(err) {
		t.Error("the file that failed to close is under its real name")
	}
}
