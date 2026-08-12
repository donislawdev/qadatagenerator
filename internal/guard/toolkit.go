package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ToolkitNetworkNames are the calls of the graphical toolkit that leave this
// machine.
//
// # Why this guard exists at all, and what it is not
//
// The import scan answers about our own packages, so it sees a network package
// we import and nothing else. The toolkit is a single import that carries
// net/http inside it: measured on 2026-08-12, the window closes over 328
// packages against 152 for the command line binary, and the root package of
// the toolkit imports net/http directly to serve resources over it.
//
// The owner settled what the unbreakable rule means here: it is a rule about
// the behaviour of the product, not about the dependency closure. A promise
// about behaviour needs a guard about behaviour, which is what this is. It
// does not claim the binary cannot reach the network. It claims that nothing
// we wrote asks it to, and that a line which starts asking has to be a
// deliberate act rather than an autocompletion.
//
// This is the same shape as the one allowance for reaching the operating
// system: a list of names rather than a blanket rule about a file, because a
// blanket allowance would let everything through and nobody would remember a
// year later why the file was exempt.
var ToolkitNetworkNames = map[string]string{
	"LoadResourceFromURLString": "fetches a resource over the network",
	"LoadResourceFromURL":       "fetches a resource over the network",
	"OpenURL":                   "hands a URL to the desktop, which opens it outside this process",
}

// ToolkitNetwork reports every call in the window that would leave this
// machine.
//
// It reads the syntax tree, so the name inside a comment or a message is not a
// finding. It matches on the selected name rather than on the package, because
// the toolkit is imported under several names across its own packages and a
// list of import aliases is a register somebody has to keep in step.
//
// # What it does not cover
//
// A call reached through a variable of an interface type, or through
// reflection. Both are possible and neither is something a screen does by
// accident, which is the failure this guard is aimed at.
func ToolkitNetwork(root string) ([]string, error) {
	view := filepath.Join(root, filepath.FromSlash(ViewRoot))
	if _, err := os.Stat(view); err != nil {
		return nil, fmt.Errorf("the window is not at %s, so this guard would check nothing: %w", ViewRoot, err)
	}

	var findings []string
	walkErr := filepath.WalkDir(view, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != view && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found, err := toolkitNetworkIn(path, filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		findings = append(findings, found...)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sort.Strings(findings)
	return findings, nil
}

func toolkitNetworkIn(path, name string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var findings []string
	ast.Inspect(parsed, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if why, banned := ToolkitNetworkNames[selector.Sel.Name]; banned {
			findings = append(findings, fmt.Sprintf("%s:%d calls %s, which %s",
				name, fileSet.Position(node.Pos()).Line, selector.Sel.Name, why))
		}
		return true
	})
	return findings, nil
}
