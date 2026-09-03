package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// No refusal carries its own next move inside its message.
//
// 🔴 This ran on the real tree on 2026-09-03 and found five places after the
// family had been closed on two the day before. That is the whole argument for
// a guard rather than a sixth repair: the family was closed on the members
// somebody had looked at, and nobody had counted them.
func TestNoRefusalAdvisesInsideItsOwnMessage(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	tree := filepath.Join(root, "internal")
	if _, statErr := os.Stat(tree); statErr != nil {
		t.Fatalf("the tree is not where this guard looks: %v", statErr)
	}

	found, err := AdviceInsideMessages(tree)
	if err != nil {
		t.Fatalf("the tree could not be read: %v", err)
	}
	for _, finding := range found {
		t.Errorf("%s", finding)
	}
}

// The guard fires on a message that advises, proved on a tree written for it
// rather than on the real one, which passes.
func TestTheAdviceGuardSeesAMessageThatAdvises(t *testing.T) {
	root := t.TempDir()
	written := filepath.Join(root, "sample.go")
	source := "package sample\n\n" +
		"func refuse() error {\n" +
		"\treturn fault.New(fault.Data, \"the file is too wide. Split it into smaller ones\")\n" +
		"}\n"
	if err := os.WriteFile(written, []byte(source), 0o600); err != nil {
		t.Fatalf("cannot write the tree this test reads: %v", err)
	}

	found, err := AdviceInsideMessages(root)
	if err != nil {
		t.Fatalf("the tree could not be read: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("a message advising with %q gave %d findings, want 1: %v", "Split", len(found), found)
	}
	if !strings.Contains(found[0], "Split") {
		t.Errorf("the finding does not name the word it fired on: %q", found[0])
	}
}

// A message describing a failure with the same word inside a statement is not a
// finding, because that word is not a move there.
func TestTheAdviceGuardLeavesAStatementAlone(t *testing.T) {
	root := t.TempDir()
	written := filepath.Join(root, "sample.go")
	source := "package sample\n\n" +
		"func refuse() error {\n" +
		"\treturn fault.New(fault.Data, \"the file changed while it was being read\")\n" +
		"}\n"
	if err := os.WriteFile(written, []byte(source), 0o600); err != nil {
		t.Fatalf("cannot write the tree this test reads: %v", err)
	}

	found, err := AdviceInsideMessages(root)
	if err != nil {
		t.Fatalf("the tree could not be read: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("a statement was read as advice: %v", found)
	}
}

// A move handed over properly is not a finding, which is the difference the
// whole guard is about.
func TestTheAdviceGuardLeavesAMoveHandedOverAlone(t *testing.T) {
	root := t.TempDir()
	written := filepath.Join(root, "sample.go")
	source := "package sample\n\n" +
		"func refuse() error {\n" +
		"\treturn fault.New(fault.Data, \"the file is too wide\").\n" +
		"\t\tDoInstead(\"split it into smaller ones\")\n" +
		"}\n"
	if err := os.WriteFile(written, []byte(source), 0o600); err != nil {
		t.Fatalf("cannot write the tree this test reads: %v", err)
	}

	found, err := AdviceInsideMessages(root)
	if err != nil {
		t.Fatalf("the tree could not be read: %v", err)
	}
	if len(found) != 0 {
		t.Errorf("a move handed over with DoInstead was read as advice in the message: %v", found)
	}
}
