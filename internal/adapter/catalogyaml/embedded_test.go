package catalogyaml_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/guard"
)

// canonical is the published catalog, which is the one source. The copy beside
// the reader exists only because an embed pattern cannot walk up out of its own
// package.
const canonical = "spec/" + catalogyaml.FileName

// This is the whole reason the copy is allowed to exist.
//
// Two files holding the same thing drift, and the drift is silent: the tool
// would keep shipping yesterday's catalog while the published specification
// said something else, and every entry either of them gained would be invisible
// to the other. Comparing the bytes turns a silent divergence into a red test
// in the same minute it happens.
//
// The comparison is exact rather than normalised. Line endings are held to LF
// across the tree by .gitattributes and by a separate check, so a difference in
// them is a real difference in the file rather than noise to be forgiven here.
func TestTheEmbeddedCatalogIsThePublishedOneByteForByte(t *testing.T) {
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root, so the copy cannot be checked: %v", err)
	}
	path := filepath.Join(root, filepath.FromSlash(canonical))

	published, err := os.ReadFile(path)
	if err != nil {
		// Loud rather than skipped. A missing canonical file is the one
		// case where this check has nothing to say, and saying nothing
		// would read as agreement.
		t.Fatalf("cannot read %s, so the copy is unchecked rather than correct: %v", canonical, err)
	}

	embedded := catalogyaml.Embedded()
	if bytes.Equal(published, embedded) {
		return
	}
	t.Errorf("the catalog compiled into this build is not %s any more.\n"+
		"  published: %d bytes, embedded: %d bytes, first difference at byte %d\n"+
		"  %s is the one source. Copy it over internal/adapter/catalogyaml/%s and run the tests again.",
		canonical, len(published), len(embedded), firstDifference(published, embedded),
		canonical, catalogyaml.FileName)
}

// firstDifference returns the offset where two files stop agreeing, so that a
// failure points at a place in the file rather than at the whole of it.
func firstDifference(left, right []byte) int {
	shorter := len(left)
	if len(right) < shorter {
		shorter = len(right)
	}
	for i := 0; i < shorter; i++ {
		if left[i] != right[i] {
			return i
		}
	}
	return shorter
}
