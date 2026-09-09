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
//
// 🔴 It got BIGGER once, on 2026-09-09, and the exception is written down so
// that it stays an exception. The owner put the accepted drawing beside a
// render of the window and the two did not look like the same program. The
// cause was measured rather than argued: the mock built its controls out of
// paint.ButtonFace and the window built them out of the toolkit's widgets, so
// there were two drawings of one thing and nothing holding them together. The
// owner was shown both roads - dressing the toolkit's controls, which cannot
// reach the drawing, and controls that draw themselves, which costs these
// entries - and chose the second knowing the price.
var HandmadeAllowed = map[string]Handmade{
	"fixedWidth": {
		Instead: "layout.NewGridWrapLayout, which does force a size on what it holds",
		Why: "it wants a whole cell size, height included, and the column of screen names has no " +
			"height of its own until the names in it are laid out. Border, the other candidate, " +
			"gives a side whatever width its content asks for, which is the thing being avoided: " +
			"the column would change width when a screen is added or a name is translated",
	},
	"PrimaryButton": {
		Instead: "widget.Button at widget.HighImportance, which is the toolkit's own filled button",
		Why: "a focused one is blend(primary, focus) and the blend at widget/button.go:453 collapses " +
			"an opaque overlay to the overlay itself, so the button goes from its resting fill to the " +
			"flat focus colour: a focus mark of 1.23 where WCAG 1.4.11 asks for 3.0. It would repair " +
			"one button per screen by breaking the mark on that same one",
	},
	"primaryRenderer": {
		Instead: "the renderer widget.Button builds for itself",
		Why: "it is the renderer that performs the blend above. Deciding what focus looks like is the " +
			"whole reason this control exists, and a renderer is the only place the toolkit lets " +
			"anybody decide it",
	},
	"Button": {
		Instead: "widget.Button, which this was an extension of until 2026-09-09",
		Why: "three things it cannot do, each measured. The gap between an icon and its label comes " +
			"from SizeNamePadding, which this window answers with nought on purpose, so an icon on a " +
			"stock button touches its label. A focused one is flooded with the focus colour, measured " +
			"at 1.56 for the label on it, which is why parts.Flooded had to exist. And its whole look " +
			"is the toolkit's, so the accepted drawing and the window were two drawings of one control " +
			"and drifted until the owner saw them side by side",
	},
	"buttonRenderer": {
		Instead: "the renderer widget.Button builds for itself",
		Why: "it draws the toolkit's look, which is the thing being replaced. This one draws " +
			"paint.ButtonFace, so the mock and the window share ONE drawing rather than two that agree " +
			"until somebody changes either",
	},
}

// HandmadeCeiling is how many pieces of the toolkit this project may carry
// copies of. Lower it when one goes back to the library.
//
// 🔴 Raised from 1 to 5 on 2026-09-09 by the owner's decision, and the number
// is the count of the entries above rather than room to grow. Two controls draw
// themselves and each costs a type and a renderer, which is four, beside the
// one layout that was here before.
//
// What lowers it again: a control going back to the library, which will happen
// if the toolkit ever lets a caller supply the content of a button. Nothing
// else does.
const HandmadeCeiling = 5

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
