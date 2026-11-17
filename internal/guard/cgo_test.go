package guard

import (
	"path/filepath"
	"strings"
	"testing"
)

// The rule applied to the repository as it stands: one file may call C, and
// only the Windows names its entry lists.
func TestTheProductCallsCOnlyWhereItIsAllowed(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := CgoUse(root, CgoAllowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("C is called where it may not be:\n%s\n\n"+
			"guard.CgoAllowed says which file may call C and which names it may take. Widening it\n"+
			"is a decision about the promise that nothing here reaches outside this machine, not a\n"+
			"formality.",
			strings.Join(findings, "\n"))
	}
}

// Without this the rule above would pass by scanning a tree that is not there.
func TestTheCScanReachesTheProduct(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := CgoUse(root, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(strings.Join(findings, "\n"), "console_windows.go imports C") {
		t.Fatalf("scanning the repository with no allowance did not report the file that calls C, "+
			"so the walk is not reaching it: %v", findings)
	}
}

// A file nobody allowed is a finding, the same file with its names allowed is
// not, and a name beyond the entry is one again.
func TestCScannerSeesAFileNobodyAllowedAndANameBeyondItsEntry(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "reaching.go"), `package sample

/*
#include <winsock2.h>
*/
import "C"

func open() { C.FreeConsole() }

func dial() { C.socket(2, 1, 0) }
`)

	findings := scanC(t, dir, nil)
	if len(findings) != 1 || !strings.Contains(findings[0], "only the files in guard.CgoAllowed") {
		t.Fatalf("a file calling C with no allowance was not reported: %v", findings)
	}

	findings = scanC(t, dir, map[string][]string{"reaching.go": {"FreeConsole", "socket"}})
	if len(findings) != 0 {
		t.Fatalf("the allowed file was still reported: %v", findings)
	}

	findings = scanC(t, dir, map[string][]string{"reaching.go": {"FreeConsole"}})
	if len(findings) != 1 || !strings.Contains(findings[0], "C.socket") {
		t.Fatalf("a name outside the allowance was not reported: %v", findings)
	}
}

// C written above the import is a function with any name doing anything, and a
// flag can change what an allowed name means, so only system headers and the
// linker flags in CgoLinkerFlags may stand there. The allowed preamble is asked
// too, written both ways cgo reads, or the check could refuse every one.
func TestCScannerSeesCWrittenAboveTheImport(t *testing.T) {
	const single, grouped = "import \"C\"", "import (\n\t\"C\"\n)"
	cases := []struct {
		name     string
		preamble string
		imports  string
		refused  string
	}{
		{"build line and system header", "/*\n#cgo LDFLAGS: -Wl,--subsystem,windows\n#include <windows.h>\n*/", single, ""},
		{"the same in line comments", "// #cgo LDFLAGS: -Wl,--subsystem,windows\n// #include <windows.h>", single, ""},
		{"a function of its own", "/*\n#include <windows.h>\nstatic void hello(void) { }\n*/", single, "static void hello(void) { }"},
		{"a function above an import of one in parentheses", "/*\nstatic void hello(void) { }\n*/", grouped, "static void hello(void) { }"},
		{"a header of our own", "/*\n#include \"ours.h\"\n*/", single, `#include \"ours.h\"`},
		{"a macro renaming an allowed call", "// #define FreeConsole socket", single, "#define FreeConsole socket"},
		{"a directive the comment text leaves out", "//line hidden.go:1\n// #include <windows.h>", single, "line hidden.go:1"},
		{"the allowed flag for one system", "// #cgo windows LDFLAGS: -Wl,--subsystem,windows", single, ""},
		{"a compiler flag renaming an allowed call", "// #cgo CFLAGS: -DFreeConsole=socket", single, "#cgo CFLAGS: -DFreeConsole=socket"},
		{"the same flag after a tab", "/*\n#cgo\tCPPFLAGS: -DFreeConsole=socket\n*/", single, `#cgo\tCPPFLAGS`},
		{"a linker flag nobody named", "// #cgo LDFLAGS: -Wl,--subsystem,windows -lws2_32", single, "passes -lws2_32 to the linker"},
		{"a library found by pkg-config", "// #cgo pkg-config: ws2", single, "#cgo pkg-config: ws2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			writeGo(t, filepath.Join(dir, "calling.go"), "package sample\n\n"+c.preamble+"\n"+c.imports+"\n\nfunc stop() { C.FreeConsole() }\n")
			findings := scanC(t, dir, map[string][]string{"calling.go": {"FreeConsole"}})
			if c.refused == "" {
				if len(findings) != 0 {
					t.Fatalf("a preamble of build lines and system headers was refused: %v", findings)
				}
				return
			}
			if len(findings) != 1 || !strings.Contains(findings[0], c.refused) {
				t.Fatalf("expected one finding naming %s, got %v", c.refused, findings)
			}
		})
	}
}

// 🔴 The exception is one package wide in the network rule and no wider. The
// window's package may import C, the store may not.
func TestTheNetworkRuleStillRefusesCOutsideTheAllowedPackage(t *testing.T) {
	allowed := Package{Path: "cmd/qadg-gui", Imports: []string{"C"}}
	other := Package{Path: "internal/adapter/store", Imports: []string{"C"}}

	findings := Network([]Package{allowed, other})
	if len(findings) != 1 || !strings.Contains(findings[0], "internal/adapter/store") {
		t.Fatalf("expected the package with no allowance to be the only finding, got %v", findings)
	}
}

func scanC(t *testing.T, dir string, allowed map[string][]string) []string {
	t.Helper()
	findings, err := CgoUse(dir, allowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return findings
}
