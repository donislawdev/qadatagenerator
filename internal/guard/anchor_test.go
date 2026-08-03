package guard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// providerRoot is where the implementations live. The contract lives in the
// domain and declares no identifiers of its own.
const providerRoot = "internal/adapter/provider"

// providerIDsInTree returns the identifier every provider answers with, read
// from the source rather than from a list somebody maintains.
//
// A hand kept list is the failure this register was built to avoid, so reading
// one here would put it straight back.
func providerIDsInTree(t *testing.T, root string) []string {
	t.Helper()

	var ids []string
	dir := filepath.Join(root, filepath.FromSlash(providerRoot))
	walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ids = append(ids, declaredProviderIDs(parsed)...)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("cannot read %s: %v", providerRoot, walkErr)
	}
	sort.Strings(ids)
	return ids
}

// declaredProviderIDs finds methods named ID that return one string literal.
func declaredProviderIDs(file *ast.File) []string {
	var ids []string
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != "ID" {
			continue
		}
		if id, found := returnedLiteral(function); found {
			ids = append(ids, id)
		}
	}
	return ids
}

func returnedLiteral(function *ast.FuncDecl) (string, bool) {
	if function.Body == nil || len(function.Body.List) != 1 {
		return "", false
	}
	returned, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return "", false
	}
	literal, ok := returned.Results[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	unquoted, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	return unquoted, true
}

// The direction that matters. A provider nobody anchored is a set of rules
// resting on one reading of one document, and every test around it would be
// green because both halves of the reading agree with each other.
func TestEveryProviderInTheTreeCarriesAnAnchor(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}

	ids := providerIDsInTree(t, root)
	if len(ids) == 0 {
		t.Fatal("no provider identifiers were found, so this check is reading nothing")
	}

	for _, missing := range ProvidersWithoutAnchor(ids, Anchors) {
		t.Errorf("%s is a provider in the tree and no entry in the register covers it.\n"+
			"Add what supports its rules, including unanchored if nothing outside this "+
			"project does. A rule nobody anchored is meant to look different from one "+
			"that is anchored.", missing)
	}
}

// The scanner has to see a provider that is not covered, or the check above is
// a sentence rather than a guard. The fault goes in a fixture because the real
// tree is meant to stay clean.
func TestAnchorScannerSeesAProviderNobodyAnchored(t *testing.T) {
	anchored := []Anchor{{Rule: "r", Provider: "PL.pesel", Source: "s", State: ByWorkedExample, Detail: "d"}}

	missing := ProvidersWithoutAnchor([]string{"PL.pesel", "XX.newone"}, anchored)
	if len(missing) != 1 || missing[0] != "XX.newone" {
		t.Fatalf("a provider outside the register went unnoticed, got %v", missing)
	}
	if none := ProvidersWithoutAnchor([]string{"PL.pesel"}, anchored); len(none) != 0 {
		t.Fatalf("a covered provider was reported as missing, got %v", none)
	}
}

// The other direction. An entry naming a provider that is gone promises
// coverage nothing keeps, and a rename is the quietest way to get there.
func TestEveryAnchorNamesAProviderThatExists(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}

	ids := providerIDsInTree(t, root)
	for _, dangling := range AnchorsNamingNoProvider(ids, Anchors) {
		t.Errorf("the register carries %q and no provider in the tree answers with that "+
			"identifier", dangling)
	}
}

func TestAnchorScannerSeesAnEntryPointingNowhere(t *testing.T) {
	anchors := []Anchor{
		{Rule: "still here", Provider: "PL.pesel", Source: "s", State: ByWorkedExample, Detail: "d"},
		{Rule: "gone", Provider: "PL.removed", Source: "s", State: ByWorkedExample, Detail: "d"},
	}

	dangling := AnchorsNamingNoProvider([]string{"PL.pesel"}, anchors)
	if len(dangling) != 1 || !strings.HasPrefix(dangling[0], "PL.removed") {
		t.Fatalf("an entry pointing at nothing went unnoticed, got %v", dangling)
	}
}

// An entry claiming the source did the arithmetic has to name the test that
// reproduces it, and that test has to exist. Otherwise the strongest state in
// the register is the one easiest to claim.
func TestEveryWorkedExampleAnchorNamesATestThatExists(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}

	declared := testFunctionsUnder(t, filepath.Join(root, filepath.FromSlash(providerRoot)))
	if len(declared) == 0 {
		t.Fatal("no tests were found under the providers, so this check is reading nothing")
	}

	for _, anchor := range Anchors {
		if anchor.State != ByWorkedExample {
			continue
		}
		if !contains(declared, anchor.Detail) {
			t.Errorf("%s says the source works the example and points at %q, "+
				"which is not a test under %s", anchor.Provider, anchor.Detail, providerRoot)
		}
	}
}

// testFunctionsUnder returns every test declared anywhere below dir.
func testFunctionsUnder(t *testing.T, dir string) []string {
	t.Helper()

	var names []string
	walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		names = append(names, declaredTests(parsed)...)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("cannot read %s: %v", dir, walkErr)
	}
	sort.Strings(names)
	return names
}

// A measured rule that does not say how many values or when asks to be
// believed. The whole point of naming measurement as the weaker state is that
// the next session repeats it instead.
func TestEveryMeasurementAnchorCanBeRepeated(t *testing.T) {
	for _, fault := range IncompleteAnchors(Anchors) {
		t.Errorf("%s", fault)
	}
}

func TestAnchorScannerSeesAMeasurementNobodyCouldRepeat(t *testing.T) {
	cases := []struct {
		name   string
		detail string
		faulty bool
	}{
		{"count and date", "174 numbers checked on 2026-08-02", false},
		{"date but no count", "checked on 2026-08-02", true},
		{"count but no date", "174 numbers checked", true},
		{"neither", "checked against the register", true},
		{"empty", "", true},
	}

	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			anchors := []Anchor{{
				Rule: "weights", Provider: "PL.regon", Source: "s",
				State: ByMeasurement, Detail: one.detail,
			}}
			faults := IncompleteAnchors(anchors)
			if faulty := len(faults) > 0; faulty != one.faulty {
				t.Fatalf("detail %q reported %d faults, wanted faulty=%v",
					one.detail, len(faults), one.faulty)
			}
		})
	}
}

// The ratchet, and the count nobody has to trust because the run prints it.
func TestTheUnanchoredListOnlyShrinks(t *testing.T) {
	counts := CountByAnchor(Anchors)

	if counts[Unanchored] > UnanchoredCeiling {
		t.Fatalf("%d rules rest on nothing but our own reading and the ceiling is %d.\n"+
			"Anchor one instead of raising the ceiling, which is what the ceiling is for.",
			counts[Unanchored], UnanchoredCeiling)
	}
	total := counts[ByWorkedExample] + counts[ByQuotedSentence] + counts[ByMeasurement] + counts[Unanchored]
	if total != len(Anchors) {
		t.Fatalf("%d entries carry a state this check knows and the register holds %d, "+
			"so some entry has a state nobody counts", total, len(Anchors))
	}
	t.Logf("worked example %d, quoted sentence %d, measured once %d, unanchored %d",
		counts[ByWorkedExample], counts[ByQuotedSentence], counts[ByMeasurement], counts[Unanchored])
}
