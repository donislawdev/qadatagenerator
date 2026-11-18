package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type outcome struct {
	code           int
	stdout, stderr string
}

func invoke(args []string, stdin io.Reader, stdout io.Writer, from tree) outcome {
	var out, errs bytes.Buffer
	if stdout == nil {
		stdout = &out
	}
	code := run(args, stdin, stdout, &errs, from)
	return outcome{code: code, stdout: out.String(), stderr: errs.String()}
}

// rendererFolder writes fixtureRenderer to a folder, for the command to read
// the way a release run has it read .github/opengl.
func rendererFolder(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for name, file := range fixtureRenderer() {
		if err := os.WriteFile(filepath.Join(dir, name), file.Data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestTheDocumentGoesToStandardOutputAndTheSummaryToStandardError(t *testing.T) {
	o := invoke(nil, strings.NewReader(fixtureListing()), nil, fixtureTree)
	if o.code != 0 {
		t.Fatalf("exit %d: %s", o.code, o.stderr)
	}
	var doc spdxDocument
	if err := json.Unmarshal([]byte(o.stdout), &doc); err != nil {
		t.Fatalf("standard output is not the document: %v", err)
	}
	if len(doc.Packages) != 6 {
		t.Errorf("%d packages, want the program, the standard library, two modules and two works of data", len(doc.Packages))
	}
	want := "sbom: qadg 0.1.0 at 89c6fcaaf91f, 2 build(s) (linux/amd64, windows/arm64), 5 components beside the program\n"
	if o.stderr != want {
		t.Errorf("standard error is %q, want %q", o.stderr, want)
	}
}

func TestOneDocumentDescribesBothPrograms(t *testing.T) {
	o := invoke([]string{"-renderer", rendererFolder(t)}, strings.NewReader(fixtureListing()+fixtureWindowListing()), nil, fixtureTree)
	if o.code != 0 {
		t.Fatalf("exit %d: %s", o.code, o.stderr)
	}
	want := "sbom: qadg 0.1.0 at 89c6fcaaf91f, 5 build(s) (linux/amd64, qadg-gui darwin/arm64, qadg-gui linux/amd64, " +
		"qadg-gui windows/amd64, windows/arm64), 14 components beside the programs\n"
	if o.stderr != want {
		t.Errorf("standard error is %q, want %q", o.stderr, want)
	}
}

// The notices this tree ships, the renderer's notices it keeps, and a listing
// of builds of both programs that link what they name: the inputs a release
// run gives this command, short of building.
//
// It asks the command for both answers: the same builds with the toolkit at
// a version the notices do not name have to be refused, and the builds as the
// notices name them have to be described. The modules of the builds come from
// the notices themselves, so a work dropped from the table in programs.go is
// what TestTheNoticesInThisTreeDescribeABuild sees, not this.
func TestTheShippedNoticesDescribeABuildThatLinksWhatTheyName(t *testing.T) {
	notes, err := parseTree(shippedTree)
	if err != nil {
		t.Fatal(err)
	}
	depsOf := func(n notices) map[string]string {
		deps := map[string]string{}
		for path, c := range n.modules {
			deps[path] = c.version
		}
		return deps
	}
	n, w := notes[programPath], notes[windowPath]
	var listing string
	for _, target := range []string{"linux/amd64", "linux/arm64", "darwin/arm64", "windows/amd64", "windows/arm64"} {
		listing += listingOf(programPath, "dist/qadg-"+target, target, n.std.version, depsOf(n))
	}
	for _, target := range []string{"windows/amd64", "linux/amd64", "darwin/arm64"} {
		listing += listingOf(windowPath, "dist/qadg-gui-"+target, target, w.std.version, depsOf(w))
	}
	toolkit := w.modules["fyne.io/fyne/v2"].version
	if toolkit == "" {
		t.Fatal("the window's notices name no fyne.io/fyne/v2, so the planted case would plant nothing")
	}
	newer := strings.Replace(listing, "\tdep\tfyne.io/fyne/v2\t"+toolkit+"\t", "\tdep\tfyne.io/fyne/v2\tv9.9.9\t", 1)
	if r := invoke([]string{"-renderer", rendererDir}, strings.NewReader(newer), nil, shippedTree); r.code != 1 ||
		!strings.Contains(r.stderr, "links fyne.io/fyne/v2 v9.9.9, and the window's notices name "+toolkit) {
		t.Errorf("a build linking the toolkit at a version the notices do not name: exit %d with %q", r.code, r.stderr)
	}
	o := invoke([]string{"-renderer", rendererDir}, strings.NewReader(listing), nil, shippedTree)
	if o.code != 0 {
		t.Fatalf("exit %d: %s", o.code, o.stderr)
	}
	var doc spdxDocument
	if err := json.Unmarshal([]byte(o.stdout), &doc); err != nil {
		t.Fatal(err)
	}
	named := map[string]bool{}
	for _, p := range doc.Packages {
		named[p.Name] = true
	}
	for _, want := range []string{"qadg", "qadg-gui", "stdlib", "fyne.io/fyne/v2", "github.com/go-gl/gl", "IBM/plex",
		"lucide-icons/lucide", "MinGW-w64 runtime", "GNU C Library", "Mesa 3D", "joke2k/faker", "U.S. Census Bureau"} {
		if !named[want] {
			t.Errorf("the document has no package %q", want)
		}
	}
	if len(doc.Extracted) != 7 {
		t.Errorf("the document carries %d texts without an SPDX identifier, and the notices quote seven", len(doc.Extracted))
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("the pipe broke") }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("the disk is full") }

func TestEveryWayTheCommandRefusesHasItsExitCode(t *testing.T) {
	mismatched := listingFor("dist/a", "linux/amd64", "go1.27.4", map[string]string{"example.com/alpha": "v1.2.3"})
	windows := windowListingFor("dist/w.exe", "windows/amd64", windowDeps(fixturePortal))
	prerelease := fixtureTree
	prerelease.notices = strings.ReplaceAll(fixtureNotices, "go1.27.4", "go1.28rc1")
	nowhere := filepath.Join(t.TempDir(), "nowhere")
	for _, c := range []struct {
		name   string
		args   []string
		stdin  io.Reader
		stdout io.Writer
		from   tree
		code   int
		want   string
	}{
		{"arguments", []string{"x.json"}, strings.NewReader(""), nil, fixtureTree, 2, "usage:"},
		{"a flag it does not know", []string{"-o", "x.json"}, strings.NewReader(""), nil, fixtureTree, 2, "usage:"},
		{"the renderer flag without a folder", []string{"-renderer"}, strings.NewReader(""), nil, fixtureTree, 2, "usage:"},
		{"broken pipe", nil, failingReader{}, nil, fixtureTree, 2, "the pipe broke"},
		{"too much input", nil, strings.NewReader(strings.Repeat("x", inputCeiling+1)), nil, fixtureTree, 2, "is over"},
		{"not a listing", nil, strings.NewReader("hello\n"), nil, fixtureTree, 2, "names no binary"},
		{"notices unreadable", nil, strings.NewReader(fixtureListing()), nil, tree{notices: "no sections", version: "0.1.0"}, 1,
			"cannot describe a build"},
		{"no renderer folder", []string{"-renderer", nowhere}, strings.NewReader(windows), nil, fixtureTree, 1,
			"the software renderer's notices cannot be read from " + nowhere},
		{"the renderer not given", nil, strings.NewReader(windows), nil, fixtureTree, 1,
			"  qadg-gui windows/amd64 travels with Mesa 3D"},
		{"disagreement", nil, strings.NewReader(mismatched), nil, fixtureTree, 1, "  the notices name example.com/beta/v2"},
		{"not a release toolchain", nil, strings.NewReader(strings.ReplaceAll(fixtureListing(), "go1.27.4", "go1.28rc1")), nil,
			prerelease, 1, "not a Go release"},
		{"output fails", nil, strings.NewReader(fixtureListing()), failingWriter{}, fixtureTree, 1, "the disk is full"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := invoke(c.args, c.stdin, c.stdout, c.from)
			if o.code != c.code || !strings.Contains(o.stderr, c.want) {
				t.Errorf("exit %d with %q, want %d with %q", o.code, o.stderr, c.code, c.want)
			}
			if o.stdout != "" {
				t.Errorf("a refusal wrote %d bytes of document", len(o.stdout))
			}
		})
	}
}
