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

// The register of screens in the sidebar exists twice, and this is what holds
// the two copies to the same promise.
//
// # What happened, measured on 2026-09-09
//
// One copy is in cmd/qadg-gui, which is what the program builds from. The other
// is in internal/gui/screens, which is what the probe that renders every screen
// state builds from, and what every test reads. They have to be two, because
// the first opens real files and real dialogs and the second must not.
//
// The second gained an icon per screen on the day the icons were drawn. The
// first never did. So every render anybody looked at showed a column of five
// names with an icon each, and the program showed five names with a gap where
// the icon would be, since the same layout reserves that space either way.
//
// It survived because the fault is an ABSENCE on the one surface nothing
// photographs. The tests read the probe's copy. The renders come from the
// probe's copy. The only way to see the program's copy is to run the binary and
// look, and the owner did, and asked why the program looked worse than the
// drawing. This was most of the answer.
//
// # Why this asks about a field rather than comparing the two lists
//
// Comparing them would need this guard to know which entries correspond, and
// they are built differently on purpose: one indexes a slice by a constant, the
// other writes a literal per line. Pairing them would be a third copy of the
// order, which is the thing that went wrong here in the first place.
//
// Asking each literal whether it names an icon needs no pairing and catches the
// same fault from either side. A screen added to one list without an icon fails
// here whichever list it was added to.
//
// # What it cannot answer
//
// Whether the icon named is the RIGHT one, or the same one in both copies. A
// screen given the wrong icon passes here and is caught by looking, which is
// the correct division: this holds the thing that is invisible, and a person
// holds the thing that is obvious the moment it is seen.

// screenLiteral is the type whose every use has to name an icon.
const screenLiteral = "Screen"

// screenIconField is the field it has to set.
const screenIconField = "Icon"

// SidebarScreens reports every place that builds a sidebar entry without an
// icon, and every file that was expected to build one and no longer does.
//
// 🔴 The second half is what keeps this from passing by finding nothing. A
// guard that only objects to what it finds is green when it finds nothing at
// all, which is exactly the state a moved or renamed package leaves it in.
func SidebarScreens(root string) ([]string, error) {
	// Both roots are named rather than discovered, because the fault this
	// guard exists for is a list living in two places, and a walk of the whole
	// tree would go quiet the day one of them moved.
	roots := []string{ViewRoot, "cmd/qadg-gui"}

	var findings []string
	built := 0
	for _, where := range roots {
		dir := filepath.Join(root, filepath.FromSlash(where))
		if _, err := os.Stat(dir); err != nil {
			return nil, fmt.Errorf("%s is not there, so this guard would check half of what it names: %w", where, err)
		}
		found, walked, err := screensBuiltUnder(dir, root)
		if err != nil {
			return nil, err
		}
		built += walked
		findings = append(findings, found...)
	}

	if built == 0 {
		findings = append(findings, fmt.Sprintf(
			"nothing in %s builds a %s at all, so this guard passed by walking nothing. Either the "+
				"sidebar is built somewhere else now, or the type was renamed",
			strings.Join(roots, " or "), screenLiteral))
	}
	sort.Strings(findings)
	return findings, nil
}

// screensBuiltUnder collects the sidebar entries built in one tree, and says
// how many it saw whether or not they were faulty.
func screensBuiltUnder(dir, root string) ([]string, int, error) {
	var findings []string
	built := 0
	walkErr := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != dir && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found, seen, err := screensIn(path, filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		findings = append(findings, found...)
		built += seen
		return nil
	})
	return findings, built, walkErr
}

// screensIn reads one file for sidebar entries built without an icon.
//
// The literal is recognised by the type name it composes rather than by the
// package qualifying it, so that one copy writing shell.Screen and the other
// writing Screen from inside the package are both seen.
func screensIn(path, where string) ([]string, int, error) {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return nil, 0, err
	}

	var findings []string
	built := 0
	read := func(literal *ast.CompositeLit) {
		built++
		for _, element := range literal.Elts {
			field, named := element.(*ast.KeyValueExpr)
			if !named {
				continue
			}
			if key, is := field.Key.(*ast.Ident); is && key.Name == screenIconField {
				return
			}
		}
		findings = append(findings, fmt.Sprintf(
			"%s:%d builds the sidebar entry %s without an %s, so the program draws a name with a gap "+
				"beside it where every render shows a picture. The other copy of this list is the one "+
				"the probe reads, and it is not the one anybody runs",
			where, fileSet.Position(literal.Pos()).Line, screenNamedBy(literal), screenIconField))
	}

	ast.Inspect(parsed, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		switch {
		case buildsAScreen(literal.Type):
			read(literal)
		case buildsAListOfScreens(literal.Type):
			// 🔴 The entries of a slice literal carry NO type of their own,
			// which is how the program's copy is written and is why the first
			// version of this guard found nothing there. An entry inside
			// []shell.Screen{...} is a screen as much as one written out in
			// full, and it was the only place the fault actually was.
			for _, element := range literal.Elts {
				if entry, is := element.(*ast.CompositeLit); is {
					read(entry)
				}
			}
		}
		return true
	})
	return findings, built, nil
}

// buildsAScreen says whether this literal composes the sidebar entry type.
//
// 🔴 The qualified form only, and that is a correction rather than an
// oversight. Matching a bare Screen as well made this object to two literals in
// internal/gui/manifestview, which has a type of its own by that name and
// nothing to do with the sidebar. A name is not a type, and the package
// qualifier is the only thing in the syntax that tells these apart.
//
// What it gives up: a list built inside package shell itself, where the name
// would be bare. Neither copy is there, and a list moved there would take the
// count of screens seen to nought, which this guard reports.
func buildsAScreen(expression ast.Expr) bool {
	qualified, ok := expression.(*ast.SelectorExpr)
	return ok && qualified.Sel.Name == screenLiteral
}

// buildsAListOfScreens says whether this literal is a slice of them.
func buildsAListOfScreens(expression ast.Expr) bool {
	list, ok := expression.(*ast.ArrayType)
	return ok && buildsAScreen(list.Elt)
}

// screenNamedBy is the translation key a sidebar entry is named by, for a
// message somebody can act on without opening the file.
func screenNamedBy(literal *ast.CompositeLit) string {
	for _, element := range literal.Elts {
		field, named := element.(*ast.KeyValueExpr)
		if !named {
			continue
		}
		key, is := field.Key.(*ast.Ident)
		if !is || key.Name != "Name" {
			continue
		}
		call, isCall := field.Value.(*ast.CallExpr)
		if !isCall || len(call.Args) != 1 {
			continue
		}
		if argument, isSelector := call.Args[0].(*ast.SelectorExpr); isSelector {
			return argument.Sel.Name
		}
	}
	return "(unnamed)"
}
