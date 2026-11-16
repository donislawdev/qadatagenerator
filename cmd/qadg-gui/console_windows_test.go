package main

import (
	"debug/pe"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// What this defends: a double click on qadg-gui.exe opens the window and
// nothing else, while qadg.exe stays a console program, which a terminal waits
// for and shows (B-392).
//
// It reads the header of binaries it builds rather than the line in
// console_windows.go, because that line works through the order in which the C
// linker sees its flags, and the line can stay while the order changes. The
// command line binary is the canary. Built the same way, it has to come out as
// a console program, so a reader that answered GUI for everything fails here.
func TestTheWindowStartsWithoutAConsole(t *testing.T) {
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("this test builds both binaries and needs the go command, which go test puts on PATH: %v", err)
	}
	dir := t.TempDir()
	binaries := []struct {
		pkg, exe string
		want     uint16
		why      string
	}{
		{".", "qadg-gui.exe", pe.IMAGE_SUBSYSTEM_WINDOWS_GUI,
			"a console program gets a console window beside the window"},
		{"../qadg", "qadg.exe", pe.IMAGE_SUBSYSTEM_WINDOWS_CUI,
			"a GUI program is not waited for by a terminal and shows it nothing"},
	}
	for _, each := range binaries {
		built := filepath.Join(dir, each.exe)
		if output, err := exec.Command(goTool, "build", "-o", built, each.pkg).CombinedOutput(); err != nil {
			t.Fatalf("go build %s: %v\n%s", each.pkg, err, output)
		}
		if got := subsystemOf(t, built); got != each.want {
			t.Errorf("%s has subsystem %d and should have %d, because %s", each.exe, got, each.want, each.why)
		}
	}
}

// subsystemOf is the subsystem written in the header of an executable.
func subsystemOf(t *testing.T, path string) uint16 {
	t.Helper()
	file, err := pe.Open(path)
	if err != nil {
		t.Fatalf("%s is not a readable executable: %v", path, err)
	}
	defer func() { _ = file.Close() }()
	switch header := file.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		return header.Subsystem
	case *pe.OptionalHeader32:
		return header.Subsystem
	}
	t.Fatalf("%s has no optional header to read a subsystem from", path)
	return 0
}

// A stream with no handle behind it is missing, and a pipe somebody handed the
// program is not. The second half is what keeps a redirected answer where it
// was sent instead of in the console.
func TestAStreamIsMissingOnlyWhenItLeadsNowhere(t *testing.T) {
	if !missing(nil) {
		t.Error("no stream at all was taken for a stream")
	}
	if !missing(os.NewFile(0, "no handle")) {
		t.Error("a stream with no handle was taken for a stream")
	}
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	defer func() { _ = writer.Close() }()
	if missing(writer) {
		t.Error("a pipe was taken for a missing stream, so a redirected answer would go to the console")
	}
}
