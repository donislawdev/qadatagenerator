package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Nothing is visible under its real name until the whole run has succeeded, so
// a run that stops halfway leaves no file that looks complete.
func TestNothingAppearsUnderItsRealNameBeforeTheRunSucceeds(t *testing.T) {
	dir := t.TempDir()
	set := store.New(dir, false)

	file, err := set.Create("customer.csv")
	if err != nil {
		t.Fatalf("cannot create: %v", err)
	}
	if _, err := file.Write([]byte("a,b\n")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "customer.csv")); err == nil {
		t.Error("the file appeared under its real name before the run finished")
	}

	if err := set.Commit(); err != nil {
		t.Fatalf("cannot commit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "customer.csv")); err != nil {
		t.Errorf("the file never arrived: %v", err)
	}
}

// A discarded run takes its temporary files with it, which is what a failure
// and a signal both need.
func TestADiscardedRunLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	set := store.New(dir, false)
	if _, err := set.Create("customer.csv"); err != nil {
		t.Fatalf("cannot create: %v", err)
	}
	set.Discard()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("a discarded run left %d files behind", len(entries))
	}
}

// A file that already exists stops the run, unless overwriting was asked for.
func TestAnExistingFileStopsTheRunUnlessOverwritingIsAllowed(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "customer.csv"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	refusing := store.New(dir, false)
	defer refusing.Discard()
	_, err := refusing.Create("customer.csv")
	if err == nil {
		t.Fatal("an existing file was about to be overwritten without a word")
	}
	if fault.KindOf(err) != fault.IO {
		t.Errorf("the refusal carries the kind %v and a write failure is %v", fault.KindOf(err), fault.IO)
	}
	if !strings.Contains(err.Error(), "Nothing has been written") {
		t.Errorf("the refusal does not say the data is untouched: %v", err)
	}

	allowing := store.New(dir, true)
	defer allowing.Discard()
	if _, err := allowing.Create("customer.csv"); err != nil {
		t.Errorf("overwriting was asked for and still refused: %v", err)
	}
}

// A name the file system would refuse, or would take and turn into something
// else, is reported rather than repaired.
//
// Repairing it is the tempting answer and the wrong one: two entities differing
// only in a forbidden character would become one file, and the second dataset
// would overwrite the first without anybody being told.
func TestANameTheFileSystemWouldNotKeepIsRefusedRatherThanRepaired(t *testing.T) {
	refused := []string{
		"", "a/b.csv", "..", ".", "a<b.csv", "a|b.csv", "CON.csv", "con.json",
		"NUL", "trailing.csv" + ".", "trailing.csv" + " ",
		// 🔴 Two dots, which is what the verb that damages somebody else's file
		// always produces. Windows reserves the name by the part before the
		// FIRST dot, so these are the console and the printer as surely as
		// CON.csv is. Measured on 2026-08-11: both were accepted until the cut
		// moved from the last dot to the first.
		"CON.corrupted.csv", "PRN.corrupted.csv", "lpt1.corrupted.csv",
	}
	for _, name := range refused {
		set := store.New(t.TempDir(), false)
		if _, err := set.Create(name); err == nil {
			set.Discard()
			t.Errorf("%q was accepted as a file name", name)
		} else if fault.KindOf(err) != fault.Config {
			t.Errorf("%q was refused with the kind %v and a bad recipe is %v",
				name, fault.KindOf(err), fault.Config)
		}
	}

	// A space before the extension is deliberately absent from the list above.
	// Windows trims trailing dots and spaces from the end of a name, not from
	// the middle of one, so refusing it would be this build inventing a rule the
	// file system does not have.
	// The last two are the other side of moving the cut to the first dot, and
	// without them that move would be a refusal nobody measured the cost of.
	// The name the damaging verb always produces has to keep working, and a
	// device name that is not the first part is an ordinary name.
	for _, name := range []string{"customer.csv", "zamówienia.csv", "a.b.c.json", "name .csv",
		"customer.corrupted.csv", "data.CON.csv"} {
		set := store.New(t.TempDir(), false)
		if _, err := set.Create(name); err != nil {
			t.Errorf("%q is a usable name and was refused: %v", name, err)
		}
		set.Discard()
	}
}
