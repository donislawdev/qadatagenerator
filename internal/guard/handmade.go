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

// The register of things the window builds for itself that the toolkit already
// has.
//
// # Why this guard exists
//
// Written on 2026-08-18, out of a measured cost rather than a principle. The
// manifest browser was built three times in one evening. The first attempt made
// rows out of containers, so nothing lined up between one entry and the next.
// The second made a layout of equal columns by hand, and it ran over itself in a
// narrow window because a piece of canvas text draws its whole string wherever
// it is put. The third used widget.Table, which sets column widths, carries its
// own heading row and builds only the cells that can be seen, and it had been
// there the whole time. So had the trimming: rich text takes a truncation mode,
// beside which the hand written shortener was a worse copy.
//
// The reading of the toolkit's documentation that settled it took one round.
// Both wrong attempts took longer than that, and neither was visible in any
// test: they compiled, they passed, and the cost only appeared on a screenshot.
//
// # What this guard actually asks
//
// Not "never write a layout or a widget". It asks that writing one be a
// decision somebody wrote down, naming what in the toolkit was looked at and
// why it did not fit. That question is cheap to answer when the answer is
// known, and it is exactly the question nobody asks when a hand written version
// is the first thing that comes to mind.
//
// # What it cannot answer
//
// Whether the reason given is true. A register entry claiming the toolkit has
// nothing suitable, written by somebody who did not look, passes here. This
// makes the claim visible and dated, which is the most a reading of the tree
// can do.

// Handmade is one thing this project built rather than took.
type Handmade struct {
	// Instead names what in the toolkit was considered.
	Instead string
	// Why says what made it unsuitable.
	Why string
}

// HandmadeAllowed is every layout and widget the window implements itself.
//
// 🔴 This map may only get smaller. Each entry is a piece of the toolkit this
// project decided to write again, and the toolkit gains features while this
// register cannot gain them back on its own.
var HandmadeAllowed = map[string]Handmade{
	"fixedWidth": {
		Instead: "layout.NewGridWrapLayout, which does force a size on what it holds",
		Why: "it wants a whole cell size, height included, and the column of screen names has no " +
			"height of its own until the names in it are laid out. Border, the other candidate, " +
			"gives a side whatever width its content asks for, which is the thing being avoided: " +
			"the column would change width when a screen is added or a name is translated",
	},
}

// HandmadeCeiling is how many pieces of the toolkit this project may carry
// copies of. Lower it when one goes back to the library.
const HandmadeCeiling = 1

// Handmades reports every layout and widget written inside the window, and
// every register entry naming one that is no longer there.
//
// 🔴 Closed in both directions, for the reason this project has paid for once
// already: a register that only answers one of those two questions answers half
// of the question asked. An entry left behind after its type is gone reads as a
// decision somebody made about code that does not exist.
func Handmades(root string) ([]string, error) {
	view := filepath.Join(root, filepath.FromSlash(ViewRoot))
	if _, err := os.Stat(view); err != nil {
		return nil, fmt.Errorf("the window is not at %s, so this guard would check nothing: %w", ViewRoot, err)
	}

	found := map[string]string{}
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
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return handmadeIn(path, filepath.ToSlash(relative), found)
	})
	if walkErr != nil {
		return nil, walkErr
	}

	var findings []string
	for name, where := range found {
		if _, allowed := HandmadeAllowed[name]; !allowed {
			findings = append(findings, fmt.Sprintf(
				"%s builds %s, which the toolkit may already have. Say in HandmadeAllowed what was "+
					"looked at and why it does not fit, or use the one from the library", where, name))
		}
	}
	for name, entry := range HandmadeAllowed {
		if _, there := found[name]; !there {
			findings = append(findings, fmt.Sprintf(
				"HandmadeAllowed names %q, which the window no longer builds. Remove it rather than "+
					"leaving a decision recorded about code that is gone", name))
			continue
		}
		if strings.TrimSpace(entry.Instead) == "" || strings.TrimSpace(entry.Why) == "" {
			findings = append(findings, fmt.Sprintf(
				"HandmadeAllowed allows %q without naming both what in the toolkit was considered "+
					"and why it does not fit, so it excuses rather than decides", name))
		}
	}

	sort.Strings(findings)
	return findings, nil
}

// handmadeIn collects every type in one file that implements a layout or a
// widget of its own.
//
// The two are recognised by the methods that make them what they are rather
// than by a name: a layout is asked to place objects and to size them, and a
// widget is asked for its renderer. A type can be called anything and still be
// one of those, and a name is what somebody chooses while a method set is what
// the toolkit calls.
func handmadeIn(path, where string, found map[string]string) error {
	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	methods := map[string]map[string]bool{}
	for _, declaration := range parsed.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 {
			continue
		}
		owner := ownerOfMethod(function.Recv.List[0].Type)
		if owner == "" {
			continue
		}
		if methods[owner] == nil {
			methods[owner] = map[string]bool{}
		}
		methods[owner][function.Name.Name] = true
	}

	for owner, has := range methods {
		if (has["Layout"] && has["MinSize"]) || has["CreateRenderer"] {
			found[owner] = fmt.Sprintf("%s:%s", where, owner)
		}
	}
	return nil
}

// ownerOfMethod is the type a method hangs on, whether or not it is a pointer.
func ownerOfMethod(expression ast.Expr) string {
	switch typed := expression.(type) {
	case *ast.StarExpr:
		return ownerOfMethod(typed.X)
	case *ast.Ident:
		return typed.Name
	}
	return ""
}
