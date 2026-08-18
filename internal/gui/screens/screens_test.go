package screens

import (
	"flag"
	"os"
	"path/filepath"
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
func TestEveryScreenStillDrawsWhatItDrewBefore(t *testing.T) {
	data := Sample()
	for _, state := range All() {
		t.Run(state.Name, func(t *testing.T) {
			drawn := render(t, data, state)
			stored := filepath.Join("testdata", state.Name+".xml")
			if *update {
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
			if drawn != string(want) {
				t.Errorf("%s draws differently than it did.\nIf the change was meant, run this test with "+
					"-update and READ the diff, because the diff is the review.\nIf it was not, the cause "+
					"is in the screen rather than in this file", state.Name)
			}
		})
	}
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
	data := Sample()
	for _, state := range All() {
		if buildable, missing := state.Buildable(data); !buildable {
			t.Errorf("state %q cannot be built from the sample: %s, so nothing ever renders it",
				state.Name, missing)
		}
	}
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
