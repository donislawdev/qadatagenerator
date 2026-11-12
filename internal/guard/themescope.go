package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// ThemeScopeNames are what the toolkit offers for giving part of a window a
// theme of its own, and why the window may not use them.
//
// # Why this is a guard about memory, measured
//
// The toolkit gives such a theme a new scope when it is built, when it is
// refreshed, and every time it is drawn again after a minute out of sight,
// because the drawing of anything unseen for a minute is thrown away and the
// new one asks for a new scope (fyne v2.8.2, container/theme.go and
// internal/cache/theme.go). Every scope parses every font it draws text in
// anew, and the fonts stay until the program ends. Measured on 2026-11-12
// inside the real window (B-370): a set of fonts weighs about 6.1 MB of live
// heap, a return to the screens after a minute and a half added five of them,
// and four such returns held 78.6, 111.3, 144.0 and 176.7 MB. With no theme of
// a control's own anywhere the same walk held 45.8 to 46.6 MB and three sets
// of fonts throughout.
//
// So the cost is not paid once, where the theme is written, but every time a
// person comes back to the screen holding it, which no test of a screen sees.
// What the four themes of that day hid is hidden another way now: the ground
// and edge of a field by parts.withoutEdge, the shadow by the window's own
// theme for the whole window.
//
// # What it does not see
//
// A theme scope taken through a package of the toolkit this does not read,
// which is none today. Test files, since a test cannot put a theme into the
// window.
var ThemeScopeNames = map[string]string{
	"NewThemeOverride": "builds a theme of its own for what it holds",
	"ThemeOverride":    "is a theme of its own for what it holds",
}

// themeScopePackage is the package of the toolkit that offers them.
const themeScopePackage = "fyne.io/fyne/v2/container"

// ThemeScopes reports every place in the window, and in the program that opens
// it, that names one of ThemeScopeNames.
func ThemeScopes(root string) ([]string, error) {
	var findings []string
	for _, where := range []string{ViewRoot, WindowProgram} {
		err := eachProductFile(root, where, func(path, name string) error {
			found, err := themeScopesIn(path, name)
			findings = append(findings, found...)
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// themeScopesIn reads one file. The package is recognised by the name the
// file imports it under, so an alias does not hide a theme, and a name of the
// same spelling on a package of our own is not one.
func themeScopesIn(path, name string) ([]string, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	local, err := importedAs(parsed, themeScopePackage)
	if err != nil || local == "" || local == "_" {
		return nil, err
	}

	var findings []string
	report := func(node ast.Node, named string) {
		findings = append(findings, fmt.Sprintf("%s:%d names %s.%s, which %s, and costs the window a set "+
			"of fonts every time it is drawn again after a minute out of sight",
			name, fileSet.Position(node.Pos()).Line, themeScopePackage, named, ThemeScopeNames[named]))
	}
	// The name after a dot is visited after its selector, and is not a bare
	// name whatever it spells, so it is noted on the way down.
	afterADot := map[*ast.Ident]bool{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			afterADot[typed.Sel] = true
			if owner, ok := typed.X.(*ast.Ident); ok && owner.Name == local {
				if _, banned := ThemeScopeNames[typed.Sel.Name]; banned {
					report(node, typed.Sel.Name)
				}
			}
		case *ast.Ident:
			if _, banned := ThemeScopeNames[typed.Name]; banned && local == "." && !afterADot[typed] {
				report(node, typed.Name)
			}
		}
		return true
	})
	return findings, nil
}
