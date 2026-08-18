package guard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
)

// Nothing in the window rebuilds a piece of the toolkit without saying why.
//
// 🔴 This came out of a measured cost rather than a principle. The manifest
// browser was built three times in one evening: rows made of containers that
// lined up with nothing, then a hand written layout of equal columns that ran
// over itself in a narrow window, and finally widget.Table, which sets column
// widths, carries its own heading row and builds only the cells on screen. It
// had been there the whole time, and so had the trimming that a hand written
// shortener was a worse copy of. Neither wrong attempt was visible in a test.
func TestNothingInTheWindowRebuildsTheToolkitWithoutSayingWhy(t *testing.T) {
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := guard.Handmades(root)
	if err != nil {
		t.Fatalf("the window could not be read: %v", err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
	if len(guard.HandmadeAllowed) > guard.HandmadeCeiling {
		t.Errorf("%d pieces of the toolkit are written again here and the ceiling is %d. Each one is "+
			"a decision rather than housekeeping", len(guard.HandmadeAllowed), guard.HandmadeCeiling)
	}
}

// The guard finds a layout and a widget written by hand, and is loud when there
// is no window to read.
//
// Without this the guard could pass by checking nothing, which is the failure
// mode every register in this project is written against: a rename or a
// deletion retires a rule in silence.
func TestTheHandmadeGuardSeesBothKindsAndIsLoudWhenThereIsNothingToRead(t *testing.T) {
	root := t.TempDir()
	view := filepath.Join(root, "internal", "gui", "screen")
	if err := os.MkdirAll(view, 0o750); err != nil {
		t.Fatal(err)
	}

	// A layout is recognised by the pair of methods the toolkit calls, and a
	// widget by the one that hands over its renderer. Neither is named
	// anything suggestive, because a name is what somebody chooses and a method
	// set is what the toolkit calls.
	source := `package screen

type ourOwn struct{}

func (ourOwn) Layout(objects []int, size int) {}
func (ourOwn) MinSize(objects []int) int      { return 0 }

type alsoOurs struct{}

func (a *alsoOurs) CreateRenderer() int { return 0 }
`
	if err := os.WriteFile(filepath.Join(view, "screen.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	findings, err := guard.Handmades(root)
	if err != nil {
		t.Fatalf("the guard could not read a tree that has a window in it: %v", err)
	}
	if len(findings) < 2 {
		t.Errorf("a hand written layout and a hand written widget produced %d findings: %v",
			len(findings), findings)
	}

	// The register is closed the other way too: an entry about a type that is
	// not there is a decision recorded about code that is gone.
	empty := t.TempDir()
	if _, err := guard.Handmades(empty); err == nil {
		t.Error("a tree with no window at all was read without complaint, so the guard would pass " +
			"by checking nothing")
	}
}
