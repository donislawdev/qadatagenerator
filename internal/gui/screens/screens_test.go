package screens

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
)

var update = flag.Bool("update", false, "rewrite the stored widget trees")

// The size every stored tree was taken at.
//
// It is pinned rather than taken from the palette's starting size, because a
// change to how big the window opens is a change somebody makes on purpose and
// should not silently rewrite what every screen is compared against.
var size = fyne.NewSize(1342, 900)

func TestMain(m *testing.M) {
	application := test.NewApp()
	// 🔴 The test driver ships a deliberately garish theme so colour changes
	// stand out. Storing trees drawn with it would make every colour in them a
	// statement about the toolkit's test theme rather than about this product.
	application.Settings().SetTheme(fynetheme.Dark{})
	os.Exit(m.Run())
}

// Every state of every screen still draws what it drew before.
//
// 🔴 The stored file is the widget tree rather than a picture, and that is the
// whole design. A picture says that something changed and leaves a person with
// two images to compare. A tree says WHAT changed, in words, in an ordinary
// diff: a size, a position, a colour name, a piece of text. It also separates
// two failures that look identical in a log, because a different tree means the
// screen was BUILT differently and the same tree with different pixels means the
// same screen was DRAWN differently.
//
// 🔴 The price, written down before anybody discovers it the hard way: this
// guard freezes the appearance that exists, including the parts nobody likes.
// Every deliberate change to a screen turns it red, and that is it working.
// Regenerating with -update is part of making the change, not a way around the
// guard - but the diff has to be read, because the diff is the review.
//
// What it cannot see, all three measured here on 2026-08-18 rather than assumed:
//
//   - Whether the screen looks good. Nothing answers that but a person.
//   - Whether two colour names in the tree stand for one colour on the screen.
//     The tree names a colour rather than carrying its value, so that question
//     goes to the theme, and there is a guard next door that asks it.
//   - 🔴 Whether text is underlined or struck through. The tree records bold,
//     italic and monospace and nothing else, so the mark this window puts on the
//     cell the keyboard stands on is HALF invisible to this guard. The other
//     half is held by the guard that renders and counts pixels, and that is why
//     that one exists rather than being replaced by this file.
//   - 🔴 On macOS, whether a panel has grown past what it can show. Measured
//     there on 2026-09-05, the first time these trees ran on that system:
//     twenty three states out of thirty nine differed and every one of them
//     differed in the same thing, a scroll bar the toolkit draws on Windows and
//     Linux and does not draw there. The bar is dropped from both sides of the
//     comparison on that system alone, so the other two keep the whole tree and
//     with it the size of the bar, which is the only thing here that says a
//     panel started overflowing.
func TestEveryScreenStillDrawsWhatItDrewBefore(t *testing.T) {
	data := sampleWithSource(t)
	for _, state := range All() {
		t.Run(state.Name, func(t *testing.T) {
			drawn := render(t, data, state)
			stored := filepath.Join("testdata", state.Name+".xml")
			if *update {
				// 🔴 Refused rather than allowed to write a weaker tree. A file
				// written where no scroll bar is drawn has none in it, and it
				// would then match everywhere, quietly taking the overflow out
				// of every state for the two systems that can still see it.
				if runtime.GOOS == "darwin" {
					t.Fatal("stored trees cannot be written on macOS, because the toolkit draws no " +
						"scroll bar there and the file would lose one for every system. Write them on " +
						"Windows or Linux")
				}
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(stored, []byte(drawn), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(stored)
			if err != nil {
				t.Fatalf("no stored tree for this state: %v.\nRun the test with -update to write one, "+
					"then read what it wrote before committing it", err)
			}
			here, before := asComparedHere(drawn), asComparedHere(string(want))
			if here != before {
				t.Errorf("%s draws differently than it did.\n%s\nIf the change was meant, run this test "+
					"with -update and READ the diff, because the diff is the review.\nIf it was not, the "+
					"cause is in the screen rather than in this file",
					state.Name, firstDifference(here, before))
			}
		})
	}
}

// firstDifference names the line the two trees stop agreeing on.
//
// 🔴 Written on 2026-09-05, when this guard failed on twenty three states on
// the macOS runner and said nothing except that they differed. The instruction
// it gave, to run with -update and read the diff, is an instruction nobody here
// can carry out: the only machine that disagrees is the one this project does
// not have. A guard that fails where its own advice cannot be followed reports
// that something is wrong and withholds what.
//
// One line rather than a whole diff on purpose. These trees run to hundreds of
// lines and a failing matrix would print all of them for every state, which is
// the shape of output people stop reading.
func firstDifference(drawn, want string) string {
	got, stored := strings.Split(drawn, "\n"), strings.Split(want, "\n")
	for i := 0; i < len(got) || i < len(stored); i++ {
		if lineAt(got, i) == lineAt(stored, i) {
			continue
		}
		return fmt.Sprintf("first difference at line %d of %d stored lines:\n  stored: %s\n  drawn:  %s",
			i+1, len(stored), lineAt(stored, i), lineAt(got, i))
	}
	return "the trees differ in something the lines do not show, such as a final newline"
}

// lineAt says so when one tree has run out rather than reading past its end.
func lineAt(lines []string, i int) string {
	if i >= len(lines) {
		return "(this tree ends here)"
	}
	return lines[i]
}

// asComparedHere is the tree in the form this system is entitled to be judged on.
//
// 🔴 It is the identity everywhere except macOS, and that is the point. The
// toolkit draws no scroll bar there, so a stored tree written on Windows carries
// one that can never appear, and twenty three of thirty nine states were red for
// that single reason on 2026-09-05. Dropping the bar on every system would have
// been simpler and would have cost the only thing in these trees that says a
// panel has more in it than it can show, on the two systems that can still say
// it. So the loss is taken once, where it is forced, and written into the list
// above of what this guard cannot see.
//
// The area the bar sits in is kept. It is drawn on all three systems and its
// size is the panel, so removing it would take a real measurement along with a
// toolkit's decision.
func asComparedHere(tree string) string {
	if runtime.GOOS != "darwin" {
		return tree
	}
	return withoutScrollBars(tree)
}

// withoutScrollBars drops each scroll bar element together with what is inside it.
func withoutScrollBars(tree string) string {
	kept := make([]string, 0, strings.Count(tree, "\n")+1)
	closing := ""
	for _, line := range strings.Split(tree, "\n") {
		switch {
		case closing != "":
			if line == closing {
				closing = ""
			}
		case strings.Contains(line, `type="*widget.scrollBar">`):
			closing = indentationOf(line) + "</widget>"
		default:
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

// indentationOf is the tabs a line opens with, which is how its closing tag is
// recognised. The markup is written with one tab per level and nothing else.
func indentationOf(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, "\t"))]
}

const (
	barOpens     = `type="*widget.scrollBar">`
	barPaints    = `fillColor="scrollbar"`
	barSitsIn    = `type="*widget.scrollBarArea">`
	linesPerBar  = 3 // it opens, it paints one rectangle, it closes
	whatWentWith = "something other than a scroll bar came out of %s: %d lines for %d bars"
)

// Nothing but the scroll bars comes out of a tree, and every stored tree agrees.
//
// 🔴 This runs on every system although the removal only happens on one. A
// helper exercised on macOS alone is a helper nobody here has ever run, and that
// arrangement is exactly what hid the difference this was written for until the
// runners came back. It reads the real stored trees rather than a sample,
// because the shape being relied on is the toolkit's and not ours.
func TestNothingButTheScrollBarsComesOutOfAStoredTree(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.xml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no stored trees to read: %v", err)
	}
	found := 0
	for _, file := range files {
		before := readTree(t, file)
		bars := strings.Count(before, barOpens)
		found += bars
		after := withoutScrollBars(before)
		if bars == 0 {
			if after != before {
				t.Errorf("%s has no scroll bar and was changed anyway", file)
			}
			continue
		}
		checkBarsRemoved(t, file, before, after, bars)
	}
	if found == 0 {
		t.Error("not one stored tree carries a scroll bar, so this proves nothing about removing them")
	}
}

func checkBarsRemoved(t *testing.T, file, before, after string, bars int) {
	t.Helper()
	if strings.Contains(after, barOpens) || strings.Contains(after, barPaints) {
		t.Errorf("a scroll bar survived in %s", file)
	}
	if strings.Count(after, barSitsIn) != strings.Count(before, barSitsIn) {
		t.Errorf("the area a bar sits in went with the bar in %s, and it is drawn on every system", file)
	}
	removed := strings.Count(before, "\n") - strings.Count(after, "\n")
	if removed != bars*linesPerBar {
		t.Errorf(whatWentWith, file, removed, bars)
	}
}

func readTree(t *testing.T, file string) string {
	t.Helper()
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("cannot read %s: %v", file, err)
	}
	return string(content)
}

// Every state on the list can actually be built from the sample.
//
// 🔴 A state nobody can reach is a finding rather than a reason to skip it. The
// neighbouring project measured the cost of the other habit: two of their lists
// were never opened by any probe, and the screen they belonged to sat at two
// photographed states against nine on the screen beside it, with nothing saying
// so. Here the sample carries a manifest and a response record precisely so that
// no state has an excuse.
func TestEveryStateCanBeBuiltFromTheSample(t *testing.T) {
	data := sampleWithSource(t)
	for _, state := range All() {
		if buildable, missing := state.Buildable(data); !buildable {
			t.Errorf("state %q cannot be built from the sample: %s, so nothing ever renders it",
				state.Name, missing)
		}
	}
}

// sampleWithSource is the sample together with a file this tool did not make.
//
// The file is written into a directory of the test rather than kept in the
// repository, because the screen it feeds reads whatever it is pointed at and a
// committed file would be a second thing to keep in step with the readers.
func sampleWithSource(t *testing.T) Data {
	t.Helper()
	data := Sample()
	path, err := SampleFile(t.TempDir())
	if err != nil {
		t.Fatalf("cannot write the file the take and corrupt screen reads: %v", err)
	}
	data.SourcePath = path
	return data
}

// A state that needs a manifest says so rather than drawing an empty screen.
func TestAStateThatNeedsAManifestIsRefusedWithoutOne(t *testing.T) {
	empty := Data{}
	for _, state := range All() {
		buildable, missing := state.Buildable(empty)
		if state.NeedsManifest && buildable {
			t.Errorf("state %q claims it can be built with no manifest at all", state.Name)
		}
		if state.NeedsManifest && missing == "" {
			t.Errorf("state %q is refused without saying what is missing", state.Name)
		}
	}
}

func render(t *testing.T, data Data, state State) string {
	t.Helper()
	content := Assemble(i18n.English, data, state)
	window := test.NewWindow(content)
	defer window.Close()
	window.Resize(size)
	// 🔴 The keyboard is applied after the window exists, because focus without
	// a canvas has nothing to land on, and a state built before the window is a
	// state photographed before it happened.
	Walk(window.Canvas(), state)
	// 🔴 Laid out twice on purpose. A wrapping label reports a height for the
	// width it knows about, and on the first pass that is not the width it ends
	// up with, so it reserves a line it then does not use. Measured in the
	// neighbouring project as gaps of 24, 24 and then 43 pixels.
	window.Resize(size)
	return test.RenderToMarkup(window.Canvas())
}
