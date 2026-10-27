package version_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// The specifications are the source of two of these four numbers. The
// constants in the package are a copy that the binary can read without a
// parser, and a copy with nothing watching it drifts. These tests are what
// watches it.

func read(t *testing.T, relative string) string {
	t.Helper()
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
	if err != nil {
		t.Fatalf("cannot read %s: %v", relative, err)
	}
	return string(content)
}

func TestCatalogVersionMatchesTheCatalog(t *testing.T) {
	matcher := regexp.MustCompile(`(?m)^catalog_version:\s*"([^"]+)"`)
	found := matcher.FindStringSubmatch(read(t, "spec/corruption-catalog.yaml"))
	if found == nil {
		t.Fatal("spec/corruption-catalog.yaml carries no catalog_version, so this build cannot state one")
	}
	if found[1] != version.Catalog {
		t.Fatalf("catalog version is %q in the specification and %q in the code", found[1], version.Catalog)
	}
}

func TestManifestVersionMatchesTheManifestSpecification(t *testing.T) {
	holdsTo(t, "spec/expectation-manifest.md", "manifest", version.Manifest)
}

// The response format is read rather than written, and its constant is still a
// copy of a number a specification states. It had no test until it moved to 0.2
// on 2026-10-27, which is the day a copy drifts if it ever does.
func TestResponseVersionMatchesTheResponseSpecification(t *testing.T) {
	holdsTo(t, "spec/application-response.md", "response", version.Response)
}

// holdsTo checks every version a specification states, in its heading and in
// every example, against the constant this build carries.
func holdsTo(t *testing.T, relative, format, constant string) {
	t.Helper()
	content := read(t, relative)
	found := regexp.MustCompile(`"spec_version"\s*:\s*"([^"]+)"`).FindAllStringSubmatch(content, -1)
	if len(found) == 0 {
		t.Fatalf("%s shows no spec_version, so this build cannot state one", relative)
	}
	heading := regexp.MustCompile(`(?m)^\**Version ([0-9.]+),`).FindAllStringSubmatch(content, -1)
	if len(heading) != 1 {
		t.Fatalf("%s states its version in %d headings, and one is what a reader looks for", relative, len(heading))
	}
	for _, occurrence := range append(found, heading...) {
		if occurrence[1] != constant {
			t.Fatalf("%s version is %q in the specification and %q in the code", format, occurrence[1], constant)
		}
	}
}

// The recipe schema has no specification file yet, so the constant is its only
// source. This test fails the day such a file appears, which is the day the
// constant needs to start reading from it. Without it the two would part
// company quietly, the way the other two cannot.
func TestRecipeSchemaStillHasNoSpecificationFile(t *testing.T) {
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	found, err := filepath.Glob(filepath.Join(root, "spec", "recipe*"))
	if err != nil {
		t.Fatalf("cannot look inside spec: %v", err)
	}
	if len(found) > 0 {
		t.Fatalf("the recipe schema now has %v, so version.Recipe has to be checked against it the way the other two are", found)
	}
}

func TestCurrentReportsEveryNumber(t *testing.T) {
	current := version.Current()
	for name, value := range map[string]string{
		"tool":            current.Tool,
		"catalog":         current.Catalog,
		"manifest format": current.Manifest,
		"recipe schema":   current.Recipe,
	} {
		if value == "" {
			t.Errorf("the %s version is empty, and a version that says nothing is worse than none", name)
		}
	}
}
