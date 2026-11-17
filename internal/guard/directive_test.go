package guard

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The rule applied to the repository: only the directives in
// CompilerDirectives stand in our files. Asked first with none allowed, the
// scan has to report the go:embed lines, or the rule would pass by reading
// nothing.
func TestTheProductWritesOnlyTheCompilerDirectivesItMay(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := DirectiveUse(root, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if !strings.Contains(strings.Join(findings, "\n"), "writes //go:embed") {
		t.Fatalf("with no directive allowed the scan did not report the go:embed lines of the product, "+
			"so it is not reaching them: %v", findings)
	}
	findings, err = DirectiveUse(root, CompilerDirectives)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("a compiler directive stands where none may:\n%s\n\n"+
			"guard.CompilerDirectives says which may and why the rest are refused. A new one is a decision "+
			"about the promise that nothing here reaches outside this machine.",
			strings.Join(findings, "\n"))
	}
}

// A directive is seen wherever it stands, inside a function too, and what
// only looks like one is not: a comment with a space, a block comment, a
// string, and a test file, which never ships.
func TestTheDirectiveScanSeesWhatTheCompilerReads(t *testing.T) {
	dir := t.TempDir()
	writeGo(t, filepath.Join(dir, "linked.go"), `package sample

import (
	_ "embed"
	_ "unsafe"
)

//go:embed linked.go
var source string

//go:linkname socket syscall.socket
func socket(domain, typ, proto int) (int, error)

//go:cgo_import_dynamic sample._socket socket "ws2_32.dll"

func inside() {
	//go:noinline
}

// go:linkname with a space is prose.
/*go:linkname in a block is not a directive either*/
var text = "//go:linkname in a string"
`)
	writeGo(t, filepath.Join(dir, "linked_test.go"), "package sample\n\n//go:linkname now time.now\nfunc now() int64\n")

	findings, err := DirectiveUse(dir, CompilerDirectives)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	var where []string
	for _, finding := range findings {
		where = append(where, strings.SplitN(finding, ", ", 2)[0])
	}
	expected := []string{
		"linked.go:11 writes //go:linkname",
		"linked.go:14 writes //go:cgo_import_dynamic",
		"linked.go:17 writes //go:noinline",
	}
	if !slices.Equal(where, expected) {
		t.Fatalf("found\n%s\nexpected\n%s", strings.Join(findings, "\n"), strings.Join(expected, "\n"))
	}
}
