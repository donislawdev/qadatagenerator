package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
)

// namedGate is a package the product may import only in the files a list
// names, and from which each of those files may take only the names its entry
// lists. syscall and C are both gated this way, because each can do whatever the
// operating system can, and a whole file allowed either would be a door nobody
// remembers is open a year from now.
type namedGate struct {
	// importPath is the package being controlled.
	importPath string
	// list is the name of the variable holding the allowances, printed with
	// every finding so that whoever meets one knows where the decision lives.
	list string
	// beyondNames holds an allowed file to what its names alone cannot say, and
	// may be nil.
	beyondNames func(parsed *ast.File, name string) []string
}

// use reports every place that imports the gate's package outside what allowed
// lets it, in both directions.
//
// The three findings it can make are different failures. A file nobody allowed
// is somebody reaching for the operating system. A name outside the allowance
// is the exception widening quietly inside a file that already had one. An
// entry naming a file that no longer imports the package is the list rotting,
// which matters because a stale allowance is an open door nobody is looking at.
//
// A test may import either package freely. The promise being kept is about the
// product, and the job in CI that walks the build graph answers the other half.
func (g namedGate) use(root string, allowed map[string][]string) ([]string, error) {
	var findings []string
	seen := map[string]bool{}
	err := eachProductFile(root, ".", func(path, name string) error {
		found, imports, err := g.in(path, name, allowed[name])
		if imports {
			seen[name] = true
		}
		findings = append(findings, found...)
		return err
	})
	if err != nil {
		return nil, err
	}
	for file := range allowed {
		if !seen[file] {
			findings = append(findings, file+" is allowed to import "+g.importPath+
				" and does not, so the allowance is stale")
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// in reports what one file does with the gate's package, and whether it
// imports it.
func (g namedGate) in(path, name string, permitted []string) ([]string, bool, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, false, err
	}
	local, err := importedAs(parsed, g.importPath)
	if err != nil || local == "" {
		return nil, false, err
	}
	switch {
	case permitted == nil:
		return []string{name + " imports " + g.importPath + ", and only the files in guard." + g.list + " may"}, true, nil
	case local == ".":
		return []string{name + " imports " + g.importPath + " with a dot, which hides what it takes"}, true, nil
	}
	findings := g.namesTaken(parsed, name, local, permitted)
	if g.beyondNames != nil {
		findings = append(findings, g.beyondNames(parsed, name)...)
	}
	return findings, true, nil
}

// namesTaken reports every name a file takes from the gate's package that its
// entry does not list.
//
// It follows the name the file imports the package under. Matching the
// package's own name would let an allowed file import syscall under another
// name and take anything from it unseen.
func (g namedGate) namesTaken(parsed *ast.File, name, local string, permitted []string) []string {
	allowed := make(map[string]bool, len(permitted))
	for _, one := range permitted {
		allowed[one] = true
	}
	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, isSelector := node.(*ast.SelectorExpr)
		if isSelector && isIdent(selector.X, local) && !allowed[selector.Sel.Name] {
			findings = append(findings, fmt.Sprintf("%s uses %s.%s, which its entry in guard.%s does not name",
				name, local, selector.Sel.Name, g.list))
		}
		return true
	})
	return findings
}

// holdsAnAllowedFile reports whether a package holds a file that an allowance
// list names, which is what lets the network rule pass that package's import.
//
// It is derived from the list rather than being a second list of its own. Two
// lists agreeing today is two lists disagreeing the first time somebody edits
// one of them, which is the failure this project has already had between a
// document and the code beside it.
func holdsAnAllowedFile[V any](allowed map[string]V, pkg string) bool {
	for file := range allowed {
		if path.Dir(file) == pkg {
			return true
		}
	}
	return false
}
