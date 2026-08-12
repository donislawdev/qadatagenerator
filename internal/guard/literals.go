package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ViewRoot is the half of the tree this guard watches.
const ViewRoot = "internal/gui"

// LiteralsAllowed lists the packages inside the window that may hold a value
// of their own.
//
// # Why there is exactly one
//
// A colour woven into thirty controls costs a rewrite to change, and a gap
// typed as a number beside each control stops being comparable to the gap next
// to it. Both are cheap to prevent and expensive to undo, which is why this
// guard was written before the first screen rather than after it. Written
// afterwards it would have had to tolerate whatever the first screen happened
// to do, and the tolerance would have looked like a rule.
//
// A second entry here is a claim that some other package needs values of its
// own. That claim deserves the paragraph this one has.
var LiteralsAllowed = map[string]bool{
	"internal/gui/theme": true,
}

// A colour written as text, in either case, in any string the window carries.
var hexColour = regexp.MustCompile(`#[0-9A-Fa-f]{6}\b`)

// colourPackages are the packages whose composite literals are a colour.
//
// The check is on the package rather than on the type name, because a colour
// arrives under several names from one place and listing the names would be a
// register somebody has to keep.
var colourPackages = map[string]bool{
	"color": true,
}

// Literals reports every value inside the window that should have come from
// the theme instead.
//
// It reads the syntax tree rather than the text, so a colour named in a
// comment and the number in a sentence are not findings. A guard that fires on
// prose gets switched off, and one switched off is worse than absent.
//
// # What counts as a finding
//
// A composite literal of a colour, a string carrying a hex colour, and a
// number other than nought or one. The last is the strict one and it is
// deliberate: nought and one are how a program says empty and single, and
// every other number in a view is a distance, a size or a count that has a
// name in the theme or deserves one.
//
// # What it does not cover
//
// Whether the value taken from the theme is the right one for that control.
// Whether the screen is legible. It also says nothing at all while the window
// holds no package but the theme, which is the state on the day it was
// written, and the probe beside it is what shows it can fail meanwhile.
func Literals(root string, allowed map[string]bool) ([]string, error) {
	view := filepath.Join(root, filepath.FromSlash(ViewRoot))
	if _, err := os.Stat(view); err != nil {
		// Loud rather than a quiet nought findings. A missing view is
		// either a rename this guard has not been told about or a
		// deletion, and both are worse news than any finding it reports.
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
		name := filepath.ToSlash(relative)
		pkg := filepath.ToSlash(filepath.Dir(filepath.ToSlash(relative)))
		// A test may write a colour, because checking that a control shows
		// the right one needs the value written down to compare against.
		if allowed[pkg] || strings.HasSuffix(name, "_test.go") {
			return nil
		}

		found, err := literalsIn(path, name)
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

func literalsIn(path, name string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	var findings []string
	at := func(node ast.Node, what string) {
		findings = append(findings, fmt.Sprintf("%s:%d %s", name, fileSet.Position(node.Pos()).Line, what))
	}

	// An import path is a string and never a colour, and walking it would
	// only invite a rule about which strings are exempt.
	imports := map[ast.Node]bool{}
	for _, spec := range parsed.Imports {
		imports[spec.Path] = true
	}

	ast.Inspect(parsed, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.CompositeLit:
			if selector, ok := typed.Type.(*ast.SelectorExpr); ok {
				if pkg, ok := selector.X.(*ast.Ident); ok && colourPackages[pkg.Name] {
					at(node, fmt.Sprintf("builds a colour of its own (%s.%s), and the theme is where a colour lives", pkg.Name, selector.Sel.Name))
				}
			}
		case *ast.BasicLit:
			if imports[typed] {
				return true
			}
			switch typed.Kind {
			case token.STRING:
				if hexColour.MatchString(typed.Value) {
					at(node, "carries a colour written as text, and the theme is where a colour lives")
				}
			case token.INT, token.FLOAT:
				if typed.Value != "0" && typed.Value != "1" {
					at(node, fmt.Sprintf("uses the bare number %s, and a distance or a size comes from the scale in the theme", typed.Value))
				}
			}
		}
		return true
	})
	return findings, nil
}
