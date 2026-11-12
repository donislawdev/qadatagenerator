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
	findings, err := ToolkitNetwork(root, ToolkitNetworkAllowed)
	if err != nil {
		t.Fatalf("cannot read the window: %v", err)
	}
	if len(findings) > 0 {
		t.Errorf("%d call(s) in the window would leave this machine:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
}

// writtenUnder writes Go files into a tree for the guard to read, one
// directory at a time.
func writtenUnder(t *testing.T, root, where string) func(name, body string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(where))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	return func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTheToolkitGuardSeesACallThatLeavesTheMachine(t *testing.T) {
	root := t.TempDir()
	written := writtenUnder(t, root, "internal/gui/screen")
	written("guilty.go", "package screen\n\nimport \"fyne.io/fyne/v2\"\n\n"+
		"func load() { _, _ = fyne.LoadResourceFromURLString(\"http://example.invalid/x.png\") }\n")
	written("guilty_too.go", "package screen\n\nimport \"fyne.io/fyne/v2\"\n\n"+
		"func open(a fyne.App, u fyne.URI) { _ = a.OpenURL(u) }\n")
	// Prose is not a finding, or the guard fires on a comment and gets
	// switched off by whoever hits it first.
	written("prose.go", "package screen\n\n// We never call OpenURL from here. See the guard.\nfunc p() {}\n")
	written("clean.go", "package screen\n\nfunc c() string { return \"local only\" }\n")
	// The program that opens the window is read as well, since 2026-11-12.
	// Before that this call would have been nobody's finding.
	writtenUnder(t, root, WindowProgram)("main.go", "package main\n\nimport \"fyne.io/fyne/v2/app\"\n\n"+
		"func main() {\n\ta := app.New()\n\t_ = a.OpenURL(nil)\n}\n")

	findings, err := ToolkitNetwork(root, nil)
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("expected three findings, got %d:\n%s", len(findings), strings.Join(findings, "\n"))
	}
	for _, want := range []string{"internal/gui/screen/guilty.go:", "internal/gui/screen/guilty_too.go:",
		WindowProgram + "/main.go:"} {
		if !containsPrefix(findings, want) {
			t.Errorf("no finding names %s:\n%s", want, strings.Join(findings, "\n"))
		}
	}
}

// The one file allowed to hand an address to the desktop may do that and no
// more: the call it is allowed, the addresses listed for it, and no way for a
// caller to hand in an address of its own (DEC-276).
//
// Each case is a fault the allowance must not let through, written into a copy
// of the allowed file, and the clean copy is asked first so that a guard
// objecting to everything cannot pass by objecting.
func TestTheOneFileAllowedToOpenAPageMayOpenOnlyItsOwnPages(t *testing.T) {
	const file = "internal/gui/outward/outward.go"
	allowed := map[string]ToolkitAllowance{file: {
		Call:      "OpenURL",
		Addresses: []string{"https://example.invalid/one", "https://example.invalid/two"},
		Why:       "the case under test",
	}}
	const head = "package outward\n\nimport \"net/url\"\n\n" +
		"const (\n\tone = \"https://example.invalid/one\"\n\ttwo = \"https://example.invalid/two\"\n)\n\n" +
		"type Desktop interface{ OpenURL(*url.URL) error }\n\n" +
		"func opening(d Desktop, page string) error {\n\tu, _ := url.Parse(page)\n\treturn d.OpenURL(u)\n}\n\n" +
		"func Links(d Desktop) []error { return []error{opening(d, one), opening(d, two)} }\n"

	cases := []struct {
		name, body, finding string
	}{
		{"clean", head, ""},
		{"a third address", head + "\nconst three = \"https://example.invalid/three\"\n", "names https://example.invalid/three"},
		{"an address lost", strings.Replace(head, "\"https://example.invalid/two\"", "one", 1),
			"no longer names https://example.invalid/two"},
		{"an address taken from a caller", head + "\nfunc Open(d Desktop, page string) error { return opening(d, page) }\n",
			"Open takes a string or a URL"},
		{"a URL taken from a caller", head + "\nfunc OpenThis(d Desktop, u *url.URL) error { return d.OpenURL(u) }\n",
			"OpenThis takes a string or a URL"},
		{"a call it is not allowed", head + "\nfunc load(f interface{ LoadResourceFromURL(string) }) " +
			"{ f.LoadResourceFromURL(one) }\n", "calls LoadResourceFromURL"},
		{"the call gone", strings.Replace(head, "return d.OpenURL(u)", "_ = u\n\treturn nil", 1),
			"does not call it"},
	}
	for _, each := range cases {
		t.Run(each.name, func(t *testing.T) {
			root := t.TempDir()
			writtenUnder(t, root, "internal/gui/outward")("outward.go", each.body)
			writtenUnder(t, root, WindowProgram)("main.go", "package main\n\nfunc main() {}\n")
			findings, err := ToolkitNetwork(root, allowed)
			if err != nil {
				t.Fatalf("cannot read the tree: %v", err)
			}
			if each.finding == "" {
				if len(findings) > 0 {
					t.Fatalf("the allowed file as it should be drew findings:\n%s", strings.Join(findings, "\n"))
				}
				return
			}
			if len(findings) != 1 || !strings.Contains(findings[0], each.finding) {
				t.Errorf("want one finding saying %q, got:\n%s", each.finding, strings.Join(findings, "\n"))
			}
		})
	}

	// An allowance for a file that is not there describes nothing, and says so.
	root := t.TempDir()
	writtenUnder(t, root, ViewRoot)("clean.go", "package gui\n")
	writtenUnder(t, root, WindowProgram)("main.go", "package main\n\nfunc main() {}\n")
	findings, err := ToolkitNetwork(root, allowed)
	if err != nil {
		t.Fatalf("cannot read the tree: %v", err)
	}
	if len(findings) != 1 || !strings.Contains(findings[0], "is not there") {
		t.Errorf("an allowance for a missing file was not reported:\n%s", strings.Join(findings, "\n"))
	}
}

// A guard that answers quietly when its subject is gone is the failure this
// project has paid for before. Either half missing is that silence.
func TestTheToolkitGuardIsLoudWhenTheWindowIsGone(t *testing.T) {
	if _, err := ToolkitNetwork(t.TempDir(), nil); err == nil {
		t.Fatal("a missing window reported no findings, which reads exactly like a clean one")
	}
	root := t.TempDir()
	writtenUnder(t, root, ViewRoot)("clean.go", "package gui\n")
	if _, err := ToolkitNetwork(root, nil); err == nil {
		t.Fatalf("a missing %s reported no findings, which reads exactly like a clean one", WindowProgram)
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
	// The same for an exception, which is a name somebody will want to widen
	// the day it is inconvenient, and an allowance for a call nobody bans is an
	// allowance for nothing.
	for file, allowance := range ToolkitNetworkAllowed {
		if strings.TrimSpace(allowance.Why) == "" {
			t.Errorf("%s is allowed %s with no reason recorded", file, allowance.Call)
		}
		if _, banned := ToolkitNetworkNames[allowance.Call]; !banned {
			t.Errorf("%s is allowed %s, which this guard does not ban", file, allowance.Call)
		}
		if len(allowance.Addresses) == 0 {
			t.Errorf("%s is allowed %s with no address, so it may name none and the allowance is empty",
				file, allowance.Call)
		}
	}
}
