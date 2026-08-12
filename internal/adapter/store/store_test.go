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

// Two runs writing into one directory cannot both take the same name.
//
// # What this is about, since it is not about an attacker
//
// The package promises at the top that a file which already exists stops the
// run. That promise used to hold only for a file sitting on the disk when the
// run started: asking whether a name is free and renaming onto it are two
// moments with a whole run in between, so two runs both found the name free,
// both worked, and the second replaced the first result without a word.
//
// Whoever can write to the output directory is outside the threat model. Two
// runs of this tool are not, and the promise was ours to keep.
func TestASecondRunCannotTakeANameTheFirstIsWriting(t *testing.T) {
	dir := t.TempDir()
	first := store.New(dir, false)
	second := store.New(dir, false)

	if _, err := first.Create("customer.csv"); err != nil {
		t.Fatalf("the first run could not start: %v", err)
	}

	_, err := second.Create("customer.csv")
	if err == nil {
		t.Fatal("the second run took a name the first one is writing, so one result would replace the other")
	}
	if kind := fault.KindOf(err); kind != fault.IO {
		t.Errorf("the refusal is of kind %v rather than an input output failure", kind)
	}
	// The message has to name the file to delete, or somebody meeting this
	// after a killed run has no way forward.
	if !strings.Contains(err.Error(), ".claim") {
		t.Errorf("the refusal does not say which file is holding the name: %s", err)
	}

	// A different name is still free, because the claim is per file rather
	// than per directory.
	if _, err := second.Create("order.csv"); err != nil {
		t.Errorf("the second run cannot write a different file: %v", err)
	}
	first.Discard()
	second.Discard()
}

// The marker exists only while a run needs it. Left behind it would stop every
// later run against a directory that now legitimately holds a result.
func TestTheMarkerIsGoneOnceTheResultIsInPlace(t *testing.T) {
	dir := t.TempDir()
	set := store.New(dir, false)

	file, err := set.Create("customer.csv")
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, ".customer.csv.claim")
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("no marker was taken while the run was going: %v", err)
	}
	if _, err := file.WriteString("id\n1\n"); err != nil {
		t.Fatal(err)
	}
	if err := set.Commit(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Error("the marker outlived the run and would block the next one")
	}
	// And a later run refuses for the ordinary reason instead.
	if _, err := store.New(dir, false).Create("customer.csv"); err == nil {
		t.Error("a run after a finished one overwrote the result")
	}
}

func TestADiscardedRunLeavesNoMarkerBehind(t *testing.T) {
	dir := t.TempDir()
	set := store.New(dir, false)
	if _, err := set.Create("customer.csv"); err != nil {
		t.Fatal(err)
	}
	set.Discard()

	if _, err := os.Stat(filepath.Join(dir, ".customer.csv.claim")); !os.IsNotExist(err) {
		t.Fatal("a discarded run left a marker that stops every later run")
	}
	// The whole point: the next run can proceed.
	next := store.New(dir, false)
	if _, err := next.Create("customer.csv"); err != nil {
		t.Errorf("the next run is blocked by leftovers of a discarded one: %v", err)
	}
	// And it has to be discarded rather than left open, or the temporary file
	// stays locked and the directory cannot be removed. Windows reports that
	// as a cleanup failure with no connection to what this test is about.
	next.Discard()
}

// Overwriting is a deliberate act, and it takes no marker because it is not
// promising anybody that the name was free.
func TestOverwritingTakesNoMarker(t *testing.T) {
	dir := t.TempDir()
	set := store.New(dir, true)
	if _, err := set.Create("customer.csv"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".customer.csv.claim")); !os.IsNotExist(err) {
		t.Error("a run allowed to overwrite still took a marker")
	}
	set.Discard()
}
