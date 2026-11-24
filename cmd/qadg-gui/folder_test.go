package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// Only a folder that is there reaches the system, and a path that names no
// folder is answered in words the window has its own sentence for (DEC-336).
//
// A path that is gone, and one that is a file, would each be handed on by a
// file manager that asked nothing, and what the system does with either is not
// something a person can act on.
func TestOnlyAFolderThatIsThereIsHandedToTheSystem(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "customer.csv")
	if err := os.WriteFile(file, []byte("id\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, path string
		shown      bool
		answer     error
	}{
		{"a folder", dir, true, nil},
		{"a folder that is gone", filepath.Join(dir, "gone"), false, outward.ErrNoFolder},
		{"a file", file, false, outward.ErrNoFolder},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			var handed []string
			manager := fileManager{show: func(path string) error { handed = append(handed, path); return nil }}
			err := manager.ShowFolder(each.path)
			if !errors.Is(err, each.answer) || (each.answer == nil && err != nil) {
				t.Errorf("answered %v, want %v", err, each.answer)
			}
			if each.shown != (len(handed) == 1 && handed[0] == each.path) || (!each.shown && len(handed) > 0) {
				t.Errorf("the system was handed %q, want it handed the path: %v", handed, each.shown)
			}
		})
	}

	// What the system answers is passed on as it is: a folder that is there
	// and would not open is a reason for the dialog, not ErrNoFolder.
	refusal := errors.New("the shell did not open the folder, ShellExecuteW answered 5")
	manager := fileManager{show: func(string) error { return refusal }}
	if err := manager.ShowFolder(dir); !errors.Is(err, refusal) || errors.Is(err, outward.ErrNoFolder) {
		t.Errorf("a refusal of the system came back as %v", err)
	}
}
