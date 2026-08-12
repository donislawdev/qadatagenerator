package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Nothing we wrote asks the toolkit to leave this machine.
//
// This is the unbreakable rule about outgoing connections, stated the way the
// owner settled it on 2026-08-12: it is a rule about what the product does,
// not about what its dependency closure contains. The window links a toolkit
// that carries net/http inside it, and that is accepted with the price
// written down. What is not accepted is a line of ours that uses it.
func TestNothingInTheWindowAsksTheToolkitToLeaveThisMachine(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := ToolkitNetwork(root)
	if err != nil {
		t.Fatalf("cannot read the window: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("%d call(s) in the window would leave this machine:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
}

func TestTheToolkitGuardSeesACallThatLeavesTheMachine(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, filepath.FromSlash("internal/gui/screen"))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	written := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	written("guilty.go", "package screen\n\nimport \"fyne.io/fyne/v2\"\n\n"+
		"func load() { _, _ = fyne.LoadResourceFromURLString(\"http://example.invalid/x.png\") }\n")
	written("guilty_too.go", "package screen\n\nimport \"fyne.io/fyne/v2\"\n\n"+
		"func open(a fyne.App, u fyne.URI) { _ = a.OpenURL(u) }\n")
	// Prose is not a finding, or the guard fires on a comment and gets
	// switched off by whoever hits it first.
	written("prose.go", "package screen\n\n// We never call OpenURL from here. See the guard.\nfunc p() {}\n")
	written("clean.go", "package screen\n\nfunc c() string { return \"local only\" }\n")

	findings, err := ToolkitNetwork(root)
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected two findings, got %d:\n%s", len(findings), strings.Join(findings, "\n"))
	}
	for _, want := range []string{"guilty.go:", "guilty_too.go:"} {
		if !containsPrefix(findings, "internal/gui/screen/"+want) {
			t.Errorf("no finding names %s:\n%s", want, strings.Join(findings, "\n"))
		}
	}
}

// A guard that answers quietly when its subject is gone is the failure this
// project has paid for before.
func TestTheToolkitGuardIsLoudWhenTheWindowIsGone(t *testing.T) {
	if _, err := ToolkitNetwork(t.TempDir()); err == nil {
		t.Fatal("a missing window reported no findings, which reads exactly like a clean one")
	}
}

// The register has to carry a reason beside every name, because a name with no
// reason is one somebody removes the day it becomes inconvenient.
func TestEveryBannedToolkitCallSaysWhyItIsBanned(t *testing.T) {
	if len(ToolkitNetworkNames) == 0 {
		t.Fatal("the register is empty, so the guard checks nothing")
	}
	for name, why := range ToolkitNetworkNames {
		if strings.TrimSpace(why) == "" {
			t.Errorf("%s is banned with no reason recorded", name)
		}
	}
}
