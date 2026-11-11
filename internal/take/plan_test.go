package take

import (
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
)

// A directory left unnamed becomes the folder of the source, one named without
// its root is taken inside that folder, and one named from its root stays as
// it was typed (B-302, B-207).
func TestADirectoryIsCountedFromTheFolderOfTheSource(t *testing.T) {
	folder := filepath.Join(t.TempDir(), "in")
	source := filepath.Join(folder, "contacts.csv")
	elsewhere := filepath.Join(t.TempDir(), "elsewhere")
	// A separator and a name, which is rooted everywhere and whole only where
	// paths have no drive, so on Windows it is the one rooted path that is not
	// whole.
	fromTheRoot := string(filepath.Separator) + "elsewhere"
	for _, each := range []struct {
		name, out, source, want string
	}{
		{"nothing named", "", source, folder},
		{"a file named without a folder", "", "contacts.csv", "."},
		{"a directory named without its root", "written", source, filepath.Join(folder, "written")},
		{"the dot the command line defaults to", DefaultOut, source, folder},
		{"a directory above the folder", "..", source, filepath.Dir(folder)},
		{"a whole path", elsewhere, source, elsewhere},
		{"a path from the root of the drive", fromTheRoot, source, fromTheRoot},
	} {
		settings := Defaults()
		settings.Out = each.out
		if got := settings.Beside(each.source).Out; got != each.want {
			t.Errorf("%s: writes into %q, want %q", each.name, got, each.want)
		}
	}
}

// The plan of a run says, for each layer that was named, how much of it this
// build carries out, and the two sums are the sums of those rows.
//
// # Why a row per layer and not only the sums
//
// Two sums say that something is missing and not what. Somebody who ordered
// two layers and reads six of sixteen cannot tell which of the two this build
// has barely begun on, and that is the question the manifest exists to answer
// about a run before anybody runs it (DEC-245).
//
// The order is the order the layers were named, because the screen draws the
// rows in the order a person ticked the boxes and the command line prints them
// in the order --layers listed them, and both are that order.
func TestThePlanSplitsWhatItWouldDoByLayerInTheOrderTheLayersWereNamed(t *testing.T) {
	dir := t.TempDir()
	settings := usable()
	settings.Layers = []string{"L3", "L2"}
	opened, err := Open(sourceFile(t, dir), settings, corrupt.NeverStops{})
	if err != nil {
		t.Fatalf("the file could not be read: %v", err)
	}
	defer func() { _ = opened.Close() }()

	planned := opened.Plan(settings, loaded(t))
	if len(planned.ByLayer) != 2 || planned.ByLayer[0].Layer != "L3" || planned.ByLayer[1].Layer != "L2" {
		t.Fatalf("the plan splits into %v, want the two layers in the order they were named", planned.ByLayer)
	}
	ordered, implemented := 0, 0
	for _, layer := range planned.ByLayer {
		if layer.Ordered <= 0 {
			t.Errorf("layer %s: the catalog describes no damage in it, so it is not a layer anybody can name", layer.Layer)
		}
		if layer.Implemented < 0 || layer.Implemented > layer.Ordered {
			t.Errorf("layer %s: %d carried out of %d ordered", layer.Layer, layer.Implemented, layer.Ordered)
		}
		ordered += layer.Ordered
		implemented += layer.Implemented
	}
	if ordered != planned.Ordered || implemented != planned.Implemented {
		t.Errorf("the rows sum to %d of %d, and the plan says %d of %d",
			implemented, ordered, planned.Implemented, planned.Ordered)
	}
}
