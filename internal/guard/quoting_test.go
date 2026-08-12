package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// No numbering scheme puts a raw value into a message.
//
// 🔴 This is the real tree rather than a fixture, because the point of this
// guard is the scheme nobody has written yet. A test on a fixture proves the
// scanner works and proves nothing about the repository, which is the mistake
// the audit script already paid for once: a probe passed because the artificial
// tree had no repository in it and the check read no files at all.
func TestNoSchemeWritesAValueIntoAMessageUnbounded(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	schemes := filepath.Join(root, "internal", "adapter", "provider")
	if _, statErr := os.Stat(schemes); statErr != nil {
		t.Fatalf("the schemes are not where this guard looks: %v", statErr)
	}

	findings, err := RawQuotedValues(schemes)
	if err != nil {
		t.Fatalf("cannot read the schemes: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("%d message(s) carry an unbounded value:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
}

// The guard has to be able to fail, and it has to stay quiet about prose.
//
// Both halves are here because they are the two ways this kind of guard is
// worthless. One that never fires guards nothing. One that fires on a comment
// gets switched off, and a switched off guard is worse than none because
// nobody notices the day it stopped looking.
func TestTheQuotingGuardFiresOnCodeAndNotOnProse(t *testing.T) {
	dir := t.TempDir()
	written := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	written("guilty.go", "package p\n\nimport \"fmt\"\n\n"+
		"func f(v string) error { return fmt.Errorf(\"x: %q is wrong\", v) }\n")
	written("prose.go", "package p\n\n// This one talks about %q in a sentence and uses none.\nfunc g() {}\n")
	written("innocent.go", "package p\n\nimport \"fmt\"\n\n"+
		"func h(v string) error { return fmt.Errorf(\"x: %s is wrong\", show(v)) }\n\n"+
		"func show(v string) string { return v }\n")
	// A test file is out of scope: proving something about the verb needs the
	// verb, so a guard that read tests would fire on its own evidence.
	written("guilty_test.go", "package p\n\nimport \"fmt\"\n\n"+
		"func i(v string) error { return fmt.Errorf(\"x: %q is wrong\", v) }\n")

	findings, err := RawQuotedValues(dir)
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding, got %d:\n%s", len(findings), strings.Join(findings, "\n"))
	}
	if !strings.HasPrefix(findings[0], "guilty.go:") {
		t.Errorf("the finding does not name the file that is guilty: %s", findings[0])
	}
}
