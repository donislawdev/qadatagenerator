package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The rule this guard exists for, applied to the repository as it stands.
//
// One file may write syscall and only for the termination signal. Everything
// else naming that package is somebody reaching for the operating system, which
// is the one way around every other name the network rule lists.
func TestTheProductNamesSyscallOnlyWhereItIsAllowed(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := SyscallUse(root, SyscallAllowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("syscall is named where it may not be:\n%s\n\n"+
			"guard.SyscallAllowed says which file may write syscall and which identifiers it may\n"+
			"take. Widening it is a decision about the promise that nothing here reaches outside\n"+
			"this machine, not a formality.",
			strings.Join(findings, "\n"))
	}
}

// Without this the rule above would pass by scanning a tree that is not there.
func TestTheSyscallScanReachesTheProduct(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	// With no allowance at all, the file that does name syscall has to be
	// reported. That proves the walk reaches the product rather than nothing.
	findings, err := SyscallUse(root, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) == 0 {
		t.Fatal("scanning the repository with no allowance reported nothing, so the walk " +
			"is not reaching the file that names syscall")
	}
}

// A file nobody allowed is a finding, and the same file with an allowance is
// not. Both halves, because a guard that never passes gets switched off.
func TestSyscallScannerSeesAFileNobodyAllowed(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "reaching.go"), `package sample

import "syscall"

var stop = syscall.SIGTERM
`)

	findings := scanSyscalls(t, dir, nil)
	if len(findings) != 1 || !strings.Contains(findings[0], "only the files") {
		t.Fatalf("a file naming syscall with no allowance was not reported: %v", findings)
	}

	findings = scanSyscalls(t, dir, map[string][]string{"reaching.go": {"SIGTERM"}})
	if len(findings) != 0 {
		t.Fatalf("the allowed file was still reported: %v", findings)
	}
}

// 🔴 The half that makes this an allowance for a name rather than for a file.
// Without it the exception would cover the whole of syscall inside that file,
// which is the door this guard exists to keep shut.
func TestSyscallScannerSeesANameOutsideTheAllowance(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "reaching.go"), `package sample

import "syscall"

var stop = syscall.SIGTERM

func open() (int, error) { return syscall.Socket(1, 2, 3) }
`)

	findings := scanSyscalls(t, dir, map[string][]string{"reaching.go": {"SIGTERM"}})
	if len(findings) != 1 || !strings.Contains(findings[0], "syscall.Socket") {
		t.Fatalf("a name outside the allowance was not reported: %v", findings)
	}
}

// The scanner follows the name the file gives the package, or an allowed file
// could import syscall under another name and take anything from it unseen.
func TestSyscallScannerFollowsThePackageUnderAnotherName(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "reaching.go"), `package sample

import system "syscall"

var stop = system.SIGTERM

func open() (int, error) { return system.Socket(1, 2, 3) }
`)

	findings := scanSyscalls(t, dir, map[string][]string{"reaching.go": {"SIGTERM"}})
	if len(findings) != 1 || !strings.Contains(findings[0], "system.Socket") {
		t.Fatalf("a name taken under another name of the package was not reported: %v", findings)
	}
}

// An allowance for a file that stopped naming syscall is a door left open with
// nobody looking at it, so the list has to close from that side too.
func TestSyscallAllowanceNeverNamesAFileThatDoesNotUseIt(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "quiet.go"), `package sample

var stop = 15
`)

	findings := scanSyscalls(t, dir, map[string][]string{"quiet.go": {"SIGTERM"}})
	if len(findings) != 1 || !strings.Contains(findings[0], "stale") {
		t.Fatalf("a stale allowance was not reported: %v", findings)
	}
}

// A test may name syscall freely, and the scanner has to leave it alone. Ours
// does, in the one place that sends a real signal to its own process.
func TestSyscallScannerIgnoresTests(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "sample_test.go"), `package sample

import "syscall"

var stop = syscall.Kill
`)

	if findings := scanSyscalls(t, dir, nil); len(findings) != 0 {
		t.Fatalf("a test file was reported: %v", findings)
	}
}

// 🔴 The exception is one package wide in the network rule and no wider. A
// second package importing syscall stays a finding, which is what stops the
// allowance from quietly becoming a general permission.
func TestTheNetworkRuleStillRefusesSyscallOutsideTheAllowedPackage(t *testing.T) {
	allowed := Package{Path: "internal/cli", Imports: []string{"syscall"}}
	other := Package{Path: "internal/adapter/store", Imports: []string{"syscall"}}

	findings := Network([]Package{allowed, other})
	if len(findings) != 1 || !strings.Contains(findings[0], "internal/adapter/store") {
		t.Fatalf("expected the package with no allowance to be the only finding, got %v", findings)
	}
}

func scanSyscalls(t *testing.T, dir string, allowed map[string][]string) []string {
	t.Helper()
	findings, err := SyscallUse(dir, allowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return findings
}

func writeGo(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the sample: %v", err)
	}
}
