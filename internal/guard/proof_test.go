package guard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// testFunctionsIn returns the test and fuzz functions declared in one package
// directory.
func testFunctionsIn(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot read %s: %v", dir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			t.Fatalf("cannot read %s: %v", entry.Name(), err)
		}
		names = append(names, declaredTests(parsed)...)
	}
	sort.Strings(names)
	return names
}

func declaredTests(file *ast.File) []string {
	var names []string
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		if name := function.Name.Name; strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Fuzz") {
			names = append(names, name)
		}
	}
	return names
}

// The mechanical half of the register, and the reason it is worth having.
//
// A hand kept list is a list somebody eventually forgets to extend, and this
// project wrote that rule down before it had anywhere to apply it. Here the
// list cannot fall behind: every guard in this package is enumerated from the
// source, so adding one without saying how it is proven fails the build.
//
// Note what this does not do. Guards in other packages are registered by hand
// and a new one added without an entry is still invisible. That is the honest
// limit of the mechanism.
func TestEveryGuardInThisPackageIsRegistered(t *testing.T) {
	registered := make(map[string]bool)
	for _, guard := range Guards {
		if guard.Package == "internal/guard" {
			registered[guard.Test] = true
		}
	}

	found := testFunctionsIn(t, ".")
	if len(found) == 0 {
		t.Fatal("no test functions were found here, so this check is reading nothing")
	}

	for _, name := range found {
		if !registered[name] {
			t.Errorf("%s is a guard in this package and is not in the register.\n"+
				"Add it with the state that is true, including unproven. A guard nobody "+
				"has proven is meant to look different from one that is proven.", name)
		}
	}
}

// A register naming a test that is gone would be a promise of protection that
// nothing keeps, and a rename is the quietest way to get there.
func TestEveryRegisteredTestExistsAndIsNamedOnce(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}

	declared := make(map[string][]string)
	seen := make(map[string]string)
	for _, guard := range Guards {
		if where, repeated := seen[guard.Test]; repeated {
			t.Errorf("%s is registered twice, here and for %q", guard.Test, where)
			continue
		}
		seen[guard.Test] = guard.Behaviour

		if _, read := declared[guard.Package]; !read {
			declared[guard.Package] = testFunctionsIn(t, filepath.Join(root, guard.Package))
		}
		if !contains(declared[guard.Package], guard.Test) {
			t.Errorf("the register names %s in %s and no such test is declared there",
				guard.Test, guard.Package)
		}
	}
}

func contains(names []string, wanted string) bool {
	for _, name := range names {
		if name == wanted {
			return true
		}
	}
	return false
}

// The ratchet. Raising the ceiling to make a run green is the same act as
// editing a golden value for the same reason.
func TestTheUnprovenListOnlyShrinks(t *testing.T) {
	counts := CountByProof()

	if counts[Unproven] > UnprovenCeiling {
		t.Fatalf("%d guards are unproven and the ceiling is %d.\n"+
			"Prove one instead of raising the ceiling, which is what the ceiling is for.",
			counts[Unproven], UnprovenCeiling)
	}
	if total := counts[ByMutation] + counts[ByProbe] + counts[Unproven]; total != len(Guards) {
		t.Fatalf("%d guards carry a state this check knows and the register holds %d, "+
			"so some entry has a state nobody counts", total, len(Guards))
	}
	t.Logf("proven by mutation %d, proven by probe %d, unproven %d",
		counts[ByMutation], counts[ByProbe], counts[Unproven])
}
