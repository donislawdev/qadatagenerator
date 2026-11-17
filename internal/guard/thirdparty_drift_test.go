package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The canary of TestThirdPartyHoldsOnlyTheCarriedBinding: a module root with
// the carried binding and the one replacement passes, and a second directory
// in third_party, a second replacement or a replacement in a block do not.
func TestTheThirdPartyCheckSeesWhatElseIsThere(t *testing.T) {
	clean := "module example\n\n// a comment saying replace x => y\nreplace " + carriedModule + " => ./" + carriedDir + "\n"
	cases := []struct {
		name, goMod, extraDir, want string
	}{
		{"the binding alone", clean, "", ""},
		{"a second directory", clean, "third_party/other", "third_party/other is there"},
		{"a second replacement", clean + "replace example.com/a => ./a\n", "", "go.mod replaces"},
		{"a block of replacements", clean + "replace (\n\texample.com/a => ./a\n)\n", "", "go.mod replaces"},
		{"no replacement", "module example\n", "", "go.mod replaces"},
	}
	for _, each := range cases {
		root := t.TempDir()
		for _, dir := range []string{carriedDir, each.extraDir} {
			if dir != "" {
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o700); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(each.goMod), 0o600); err != nil {
			t.Fatal(err)
		}
		findings := strings.Join(thirdPartyDrift(root), "\n")
		switch {
		case each.want == "" && findings != "":
			t.Errorf("%s was refused: %s", each.name, findings)
		case each.want != "" && !strings.Contains(findings, each.want):
			t.Errorf("%s was not named as %q, the check said: %q", each.name, each.want, findings)
		}
	}
}

// The canary of TestTheOpenGLBindingIsThePublishedOnePlusOneChange: every way
// a copy can drift from the published module, planted on a module of a few
// lines, has to be named, and the copy that matches has to pass.

// plantedPublished is a published module in miniature, with one file the copy
// does not keep.
func plantedPublished() map[string][]byte {
	return map[string][]byte{
		"go.mod":                []byte("module github.com/go-gl/gl\n"),
		"LICENSE":               []byte("MIT\n"),
		"README.md":             []byte("not kept\n"),
		"v2.1/gl/package.go":    []byte("a\nb\nc\n"),
		"v2.1/gl/procaddr.go":   []byte("x\ny\nz\n"),
		"v3.1/gles2/package.go": []byte("p\n"),
	}
}

// plantedDiff removes one line from one file and changes one in the other.
const plantedDiff = `--- a/v2.1/gl/package.go
+++ b/v2.1/gl/package.go
@@ -1,3 +1,2 @@
 a
-b
 c
--- a/v2.1/gl/procaddr.go
+++ b/v2.1/gl/procaddr.go
@@ -1,3 +1,3 @@
 x
-y
+Y
 z
`

// plantedCopy is the copy that matches plantedPublished with plantedDiff.
func plantedCopy() map[string][]byte {
	return map[string][]byte{
		"go.mod":                []byte("module github.com/go-gl/gl\n"),
		"LICENSE":               []byte("MIT\n"),
		"v2.1/gl/package.go":    []byte("a\nc\n"),
		"v2.1/gl/procaddr.go":   []byte("x\nY\nz\n"),
		"v3.1/gles2/package.go": []byte("p\n"),
		"PATCH.md":              []byte("notes\n"),
		"PATCH.diff":            []byte(plantedDiff),
	}
}

func TestTheCopyCheckNamesEveryWayACopyDrifts(t *testing.T) {
	if findings := copyDrift(plantedPublished(), plantedCopy()); len(findings) != 0 {
		t.Fatalf("the copy that matches was refused, so the check fails everything: %q", findings)
	}
	cases := []struct {
		name, want string
		plant      func(published, carried map[string][]byte)
	}{
		{"an unpatched file changed", "v3.1/gles2/package.go differs",
			func(_, carried map[string][]byte) { carried["v3.1/gles2/package.go"] = []byte("q\n") }},
		{"a patched file changed beyond the patch", "v2.1/gl/package.go differs",
			func(_, carried map[string][]byte) { carried["v2.1/gl/package.go"] = []byte("a\nc\nd\n") }},
		{"a kept file missing", "LICENSE is published and missing",
			func(_, carried map[string][]byte) { delete(carried, "LICENSE") }},
		{"a file of nobody's added", "extra.go is in the copy",
			func(_, carried map[string][]byte) { carried["v2.1/gl/extra.go"] = []byte("e\n") }},
		{"a published file the copy does not keep added", "README.md is in the copy",
			func(_, carried map[string][]byte) { carried["README.md"] = []byte("not kept\n") }},
		{"the patch reaching a third file", "PATCH.diff touches",
			func(_, carried map[string][]byte) {
				carried["PATCH.diff"] = []byte(plantedDiff + "--- a/LICENSE\n+++ b/LICENSE\n@@ -1,1 +1,1 @@\n-MIT\n+MIT\n")
			}},
		{"the published file not the one the patch was made on", "does not apply to v2.1/gl/package.go",
			func(published, _ map[string][]byte) { published["v2.1/gl/package.go"] = []byte("a\nB\nc\n") }},
		{"a hunk whose header miscounts its lines", "counts 3 and 1 lines",
			func(_, carried map[string][]byte) {
				carried["PATCH.diff"] = []byte(strings.Replace(plantedDiff, "+1,2 @@", "+1,1 @@", 1))
			}},
		{"a line the diff format does not have", "is not a line of a unified diff",
			func(_, carried map[string][]byte) {
				carried["PATCH.diff"] = []byte(plantedDiff + "\\ No newline at end of file\n")
			}},
	}
	for _, each := range cases {
		published, carried := plantedPublished(), plantedCopy()
		each.plant(published, carried)
		findings := strings.Join(copyDrift(published, carried), "\n")
		if !strings.Contains(findings, each.want) {
			t.Errorf("%s was not named as %q, the check said:\n%s", each.name, each.want, findings)
		}
	}
}
