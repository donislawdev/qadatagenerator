package guard

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The rule applied to the repository: one file may import unsafe, and only for
// the names its entry lists. Asked first with no allowance, the scan has to
// report that file, or the rule would pass by reading nothing.
func TestTheProductImportsUnsafeOnlyWhereItIsAllowed(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := UnsafeUse(root, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(strings.Join(findings, "\n"), "console_windows.go imports unsafe") {
		t.Fatalf("with no allowance the scan did not report the file that imports unsafe, so it is not "+
			"reaching it: %v", findings)
	}
	findings, err = UnsafeUse(root, UnsafeAllowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("unsafe is used where it may not be:\n%s\n\n"+
			"guard.UnsafeAllowed says which file may import it and which names it may take. Widening it is a "+
			"decision about the promise that nothing here reaches outside this machine.",
			strings.Join(findings, "\n"))
	}
}

// A file importing unsafe for nothing is a finding, because that is how a file
// asks for go:linkname, and a name beyond an entry is one too.
func TestUnsafeScannerSeesABlankImportAndANameBeyondTheEntry(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "blank.go"), "package sample\n\nimport _ \"unsafe\"\n")
	writeGo(t, filepath.Join(dir, "pointer.go"), `package sample

import "unsafe"

var size = unsafe.Sizeof(0)

func at(p *int) uintptr { return uintptr(unsafe.Pointer(p)) }
`)
	findings, err := UnsafeUse(dir, map[string][]string{"pointer.go": {"Sizeof"}})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	expected := []string{
		"blank.go imports unsafe, and only the files in guard.UnsafeAllowed may",
		"pointer.go uses unsafe.Pointer, which its entry in guard.UnsafeAllowed does not name",
	}
	if !slices.Equal(findings, expected) {
		t.Fatalf("found\n%s\nexpected\n%s", strings.Join(findings, "\n"), strings.Join(expected, "\n"))
	}
}

// The exception is one package wide in the network rule and no wider.
func TestTheNetworkRuleStillRefusesUnsafeOutsideTheAllowedPackage(t *testing.T) {
	allowed := Package{Path: "cmd/qadg-gui", Imports: []string{"unsafe"}}
	other := Package{Path: "internal/adapter/store", Imports: []string{"unsafe"}}

	findings := Network([]Package{allowed, other})
	if len(findings) != 1 || !strings.Contains(findings[0], "internal/adapter/store") {
		t.Fatalf("expected the package with no allowance to be the only finding, got %v", findings)
	}
}
