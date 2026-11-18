package assets_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/gui/assets"
	"github.com/donislawdev/qadatagenerator/internal/license"
)

// The pair of tests that close the licence notices against the register that
// describes them.
//
// # Why a pair rather than one test walking both lists
//
// A register closed in one direction answers half the question, which this
// project has paid for before. One direction catches an asset described and
// not shipped, the other catches a file shipped and described by nobody. They
// fail with different sentences because they are different faults: the first
// is a claim about terms nobody can read, the second is data travelling with
// no terms at all.
//
// # Why this lives here and not beside the register
//
// The register is a list of facts and knows nothing about a binary. What is
// actually embedded is known only to this package, because the embed directive
// resolves against the directory holding it. So the test that compares the two
// has to stand on this side of that line, the same way the corpus tests stand
// on theirs.

func TestEveryDescribedAssetShipsItsNotice(t *testing.T) {
	embedded := map[string]bool{}
	for _, name := range assets.NoticeNames() {
		embedded[name] = true
	}
	for _, asset := range license.Embedded {
		if asset.Notice == "" {
			t.Errorf("%s is in the register with no notice file named, "+
				"and its terms ask for one to travel with it", asset.Asset)
			continue
		}
		if !embedded[asset.Notice] {
			t.Errorf("%s names the notice %q, which is not in this package: "+
				"the register names terms nobody can read beside the asset",
				asset.Asset, asset.Notice)
		}
	}
}

func TestEveryShippedNoticeIsDescribed(t *testing.T) {
	described := map[string]bool{}
	for _, asset := range license.Embedded {
		described[asset.Notice] = true
	}
	for _, name := range assets.NoticeNames() {
		if !described[name] {
			t.Errorf("the notice %q is in this package and nothing in the register "+
				"names it, so --license and the About screen do not report what the window carries", name)
		}
	}
}

// TestTheNoticesAreNotEmpty reads each one rather than only listing it.
//
// # Why the bytes are read at all
//
// A name in a directory listing proves the file is in the tree. It does not
// prove the embed pattern matched it, and that distinction is the whole defect
// this file was written for: the two older patterns were each correct about
// their own files and silently left these out. Reading the contents is what
// makes the test fail rather than pass when a pattern stops matching.
func TestTheNoticesAreNotEmpty(t *testing.T) {
	names := assets.NoticeNames()
	if len(names) == 0 {
		t.Fatal("no licence notice is embedded at all, and this package " +
			"carries a typeface and an icon set that both require one")
	}
	for _, name := range names {
		text, carried := assets.Notice(name)
		if !carried {
			t.Errorf("the notice %q is listed and cannot be read back", name)
			continue
		}
		if text == "" {
			t.Errorf("the notice %q is embedded but empty", name)
			continue
		}
		// A licence text says the word. This is not a check that the terms
		// are correct, which no test can make: it is a check that the file
		// embedded under this name is a notice rather than something that
		// happened to be called one.
		if !strings.Contains(strings.ToLower(text), "permission") {
			t.Errorf("the notice %q does not read like a licence text", name)
		}
	}
}
