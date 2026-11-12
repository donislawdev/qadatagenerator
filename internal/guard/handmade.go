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
	// The two text controls answer CreateRenderer with the toolkit's OWN
	// renderer, one rectangle of it hidden. Nothing is drawn again, but the
	// method is what this guard reads, and rightly: it is the seam through which
	// a control is taken over.
	"FieldEntry": {
		Instead: "widget.Entry drawn as it is, with SizeNameInputBorder set to nought by a theme " +
			"handed to the field alone, which is how its edge was put away until 2026-10-21",
		Why: "the toolkit draws its caret as wide as that same number, widget/entry.go:1918 of v2.8.2, " +
			"so the edge went and the caret with it - the owner could not see where they typed. With " +
			"the width given back, the edge returns in the colour of the caret under the keyboard and in " +
			"the grey of disabled words out of reach, and neither colour can be taken away alone. A clip " +
			"cutting the edge off was tried the same day and the software painter of the toolkit moves a " +
			"clipped rounded rectangle INTO the clip, so every render lied. Hiding the one stroked " +
			"rectangle among the renderer's objects is the smallest change that keeps both",
	},
	"Editor": {
		Instead: "widget.Entry drawn as it is, the same as FieldEntry",
		Why: "the same caret and the same edge, in the recipe editor, which the owner was typing into " +
			"when they reported it. One helper, parts.withoutEdge, does it for both",
	},
	"TapRow": {
		Instead: "widget.List, which holds a highlight, moves it with the arrow keys and only builds " +
			"the rows that are visible",
		Why: "it fixes the height of every row, and the card on the quick screen spends spare height on " +
			"its rows so that four values in a tall window are not four lines with a field of black " +
			"under them. Taking the list would have traded a fault the owner reported for one he " +
			"reported two days earlier. It is the right answer for the keyboard and is written down " +
			"as such in B-268",
	},
	"tapRowRenderer": {
		Instead: "the renderer of any toolkit widget that holds one child",
		Why: "the picture of a row differs between states rather than only changing colour, since the " +
			"word saying what a click does is absent until the pointer arrives. Swapping a subtree is " +
			"what a renderer is for, and no toolkit widget exposes that",
	},
	"frame": {
		Instead: "layout.NewBorderLayout, which held the column beside the screen until 2026-11-12, " +
			"and layout.NewGridWrapLayout, which does force a size on what it holds",
		Why: "Border gives a side whatever width its content asks for, so the column would change width " +
			"when a screen is added or a name is translated. GridWrap wants a whole cell size, height " +
			"included, and the column has no height until its names are laid out. And no layout of the " +
			"library puts an object anywhere but where its place in the list puts it, while the foot of " +
			"the column is handed over after the screen so that Tab reaches the two pages outside this " +
			"program last (DEC-276), and is drawn under the column. It took over fixedWidth, which kept " +
			"the width alone, so the register did not grow",
	},
	"evenRows": {
		Instead: "layout.NewGridLayoutWithRows, read on the source of v2.8.1 (layout/gridlayout.go), " +
			"which also divides its height into equal cells",
		Why: "it puts theme.Padding() between the cells, a value of the GLOBAL theme that this window " +
			"answers with nought in one theme and with a step of the scale in the other, so the gap " +
			"between the values on the quick screen would follow whichever theme happened to be set. " +
			"And its minimum is the tallest child times the rows, where a list of values needs a floor " +
			"per row that the window can fall back to when it has nothing to spare",
	},
	"fitHeight": {
		Instead: "container.NewVBox and layout.NewSpacer, and the scrolling container around them",
		Why: "a box gives each child its minimum and not one pixel more, and a spacer stretches the " +
			"gaps between children rather than the children, so nothing in the library grows a card " +
			"with the window up to a ceiling per row and leaves the surplus undrawn. The surplus has " +
			"to stay off the card because an empty framed box read as a failure on the render of " +
			"2026-09-08 (450 pixels of empty interior at 1600 by 1180)",
	},
	"Button": {
		Instead: "widget.Button, which this was an extension of until 2026-09-09",
		Why: "three things it cannot do, each measured. The gap between an icon and its label comes " +
			"from SizeNamePadding, which this window answers with nought on purpose, so an icon on a " +
			"stock button touches its label. A focused one is flooded with the focus colour, measured " +
			"at 1.56 for the label on it, which is why parts.Flooded had to exist until DEC-253 deleted it. And its whole look " +
			"is the toolkit's, so the accepted drawing and the window were two drawings of one control " +
			"and drifted until the owner saw them side by side",
	},
	"buttonRenderer": {
		Instead: "the renderer widget.Button builds for itself",
		Why: "it draws the toolkit's look, which is the thing being replaced. This one draws " +
			"paint.ButtonFace, so the mock and the window share ONE drawing rather than two that agree " +
			"until somebody changes either",
	},
	"Select": {
		Instead: "widget.Select dressed by the theme, which is what this was until 2026-09-09",
		Why: "two of the four differences from the accepted drawing are beyond any theme, read on the " +
			"source of v2.8.1. The toolkit draws a chooser no edge at all, so this package laid a " +
			"rectangle beside the control and showed it by hand. And selectRenderer.bgColor at " +
			"widget/select.go:432 floods a focused one with the focus colour, where the value on it " +
			"measures 1.56 against the 4.5 of WCAG 1.4.3, so a scope overriding the theme had to sit " +
			"around every chooser to change the ink. Only the DRAWING is taken over: the pop up menu, " +
			"the keys and the selection stay the toolkit's, since none of that was what differed",
	},
	"chooserRenderer": {
		Instead: "the renderer widget.Select builds for itself",
		Why: "it is the renderer that performs the flood above and that leaves out the edge, so it is " +
			"the only place either can be decided. It draws paint.ChooserFace, which is the drawing " +
			"the mock uses, so the two cannot differ. Its MinSize also stops being the width of the " +
			"words (Select one), which is what the toolkit's measures no matter what the chooser holds",
	},
	"picker": {
		Instead: "widget.PopUpMenu, which widget.Select opens, and widget.PopUp around a list of our own",
		Why: "the menu is a column of items and nothing else, read on the source of v2.8.1: it cannot " +
			"take a field to filter by, cannot put a heading between two items and shows one column " +
			"where the owner asked for two, with grouping and searching, because there will be " +
			"hundreds of kinds. It is also private to widget.Select, so it could not be extended. " +
			"PopUp draws a ground, a shadow and a radius of the toolkit's under whatever it is given, " +
			"which under a card drawing its own would be a second ground at every corner and a second " +
			"shadow. The overlay stack of the canvas takes any object, which is all PopUp does with " +
			"it. The rows are TapRow and the field is FieldEntry, so what this type holds is the " +
			"keyboard row, the filter and the keys - and it is focusable because a key goes to the " +
			"top overlay and to nothing under it (DEC-244)",
	},
	"pickerRenderer": {
		Instead: "the renderer widget.PopUp builds for itself",
		Why: "it draws the ground, the shadow and the radius named above, and a renderer is the only " +
			"place that drawing can be left out. This one holds the tree the panel was built as and " +
			"swaps nothing, since every state of the panel lives on a row, the foot or the " +
			"visibility of the scrolling area",
	},
	"Check": {
		Instead: "widget.Check, and widget.CheckGroup around eight of them, which the take screen " +
			"used until 2026-09-15",
		Why: "read on the source of v2.8.1: it marks focus with a filled circle behind the box " +
			"(focusIndicator in ColorNameFocus) and draws the box out of two icons of the theme's, " +
			"and neither can be reached from outside the renderer. Eight of them stand in one column " +
			"beside fields and choosers that mark focus with an edge, so the column would carry two " +
			"languages of states, which DEC-215 settled is a fault even when each control passes on " +
			"its own. Nothing of the toolkit's control is needed underneath: a tap toggles a flag, " +
			"and the keys are Space and Enter (DEC-245)",
	},
	"checkRenderer": {
		Instead: "the renderer widget.Check builds for itself",
		Why: "it is the renderer that draws the circle and the icons above, so it is the only place " +
			"either can be decided. This one draws paint.CheckFace, the drawing the mock uses, so " +
			"the accepted look and the window cannot differ",
	},
}

// HandmadeCeiling is how many pieces of the toolkit this project may carry
// copies of. Lower it when one goes back to the library.
//
// 🔴 Raised from 1 to 5, then to 7, then to 9 on 2026-09-09, and the number is
// the count of the entries above rather than room to grow. Three controls draw
// themselves and each costs a type and a renderer, which is six, beside the one
// layout that was here before and the row that answers the pointer.
//
// The last two are the one raise NOT covered by the owner's pace below, and it
// is stated rather than folded in: they came from a different request, "I
// cannot copy a single value from values", and a row that answers nobody cannot
// answer that. widget.List was the library answer and was rejected for a
// measured reason written beside the entry, not for want of looking.
//
// The owner set both the decision and its pace: one control at a time, with a
// render of the window to look at before the next one starts. So this number
// moves by two, on purpose, and a jump of more than two is somebody having
// converted several controls without anybody having seen any of them.
//
// 🔴 Raised from 9 to 11 on 2026-09-14 for the panel a kind is picked from,
// which is one control and its renderer, at the owner's pace: it was the one
// control of that session, and the window was rendered with it open before
// anything else started. It is the seventh fault the owner named, and the
// last one open, so this raise closes B-258 rather than opening anything.
//
// 🔴 Raised from 11 to 13 on 2026-09-15 for the check box, one control and
// its renderer, at the same pace: it is the one control of the take screen's
// move onto paint, the owner chose it over widget.Check on a render of the
// screen (DEC-245), and the screen is rendered with it before anything else
// on that screen changes.
//
// 🔴 Held at 13 on 2026-09-24 while two entries went and two came, and the
// swap is stated rather than folded in. The filled button stopped being a
// control of its own and became a flag on Button, so PrimaryButton and its
// renderer left. evenRows and fitHeight had been built in paint with their
// case written beside them and never entered here, so this guard was red on
// both and the red was carried from session to session as known noise. They
// are entered now with what in the library was read and why it does not fit.
// The ceiling did not have to move to admit them, which is the only reason it
// did not: two layouts the toolkit lacks are not a control drawn over again.
//
// 🔴 Raised from 13 to 15 on 2026-10-21 for FieldEntry and Editor, and the
// raise is not a control drawn over again: both hand back the toolkit's own
// renderer with its edge hidden, so that the caret has a width (the entries say
// why). It is still two entries, so it is still stated rather than folded in.
//
// What lowers it again: a control going back to the library, which will happen
// if the toolkit ever lets a caller supply the content of a button, or draws
// its caret at a width of its own rather than the width of its edge. Nothing
// else does.
const HandmadeCeiling = 15

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
