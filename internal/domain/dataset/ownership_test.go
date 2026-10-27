package dataset_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
)

// expanded is the mapping the way the plan held it until 2026-10-27, a parent
// for every child, which is the oracle the compact form has to agree with.
func expanded(shares []int) []int {
	var parents []int
	for parent, share := range shares {
		for range share {
			parents = append(parents, parent)
		}
	}
	return parents
}

// Every row belongs to the parent a parent per child gave it, parents without
// children included.
//
// 🔴 The parents with no children are the point. Their end equals the end of the
// parent before them, so a search that looks for the row itself rather than for
// the row after it lands on such a parent and puts the first child of the next
// one under a parent that has none, with every count still adding up.
func TestEveryRowBelongsToTheParentAParentPerChildGaveIt(t *testing.T) {
	shares := []int{0, 2, 0, 3, 1, 0}
	want := expanded(shares)
	owned := dataset.OwnershipOf(append([]int(nil), shares...))

	if owned.Rows() != len(want) {
		t.Fatalf("the mapping covers %d rows and the children add up to %d", owned.Rows(), len(want))
	}
	for row, parent := range want {
		if got := owned.ParentOf(row); got != parent {
			t.Errorf("row %d belongs to parent %d and the mapping says %d", row, parent, got)
		}
	}
	for _, outside := range []int{-1, len(want), len(want) + 7} {
		if got := owned.ParentOf(outside); got != dataset.NoParent {
			t.Errorf("row %d is not in the table and the mapping gives it parent %d", outside, got)
		}
	}
}

// A cut keeps the parent of every row it keeps and holds nothing past them.
//
// Row i of a preview is row i of the run because it is the same lookup, and a
// row past the cut is not in the table, which the preview has to be told rather
// than shown.
func TestACutKeepsTheParentOfEveryRowItKeeps(t *testing.T) {
	shares := []int{0, 2, 0, 3, 1}
	want := expanded(shares)
	owned := dataset.OwnershipOf(append([]int(nil), shares...))

	cut := owned.Cut(4)
	if cut.Rows() != 4 {
		t.Fatalf("a cut to four rows covers %d", cut.Rows())
	}
	for row := range 4 {
		if got := cut.ParentOf(row); got != want[row] {
			t.Errorf("row %d belonged to parent %d and after the cut to %d", row, want[row], got)
		}
	}
	if got := cut.ParentOf(4); got != dataset.NoParent {
		t.Errorf("row 4 is past the cut and still belongs to parent %d", got)
	}
	if longer := owned.Cut(100); longer.Rows() != len(want) {
		t.Errorf("a cut longer than the table made it %d rows long rather than %d",
			longer.Rows(), len(want))
	}
	if none := owned.Cut(-3); none.Rows() != 0 {
		t.Errorf("a cut below nought left %d rows", none.Rows())
	}
}

// No parents at all is a mapping of no rows, which is what a child of an empty
// table is.
func TestNoParentsIsAMappingOfNoRows(t *testing.T) {
	owned := dataset.OwnershipOf(nil)
	if owned.Rows() != 0 || owned.ParentOf(0) != dataset.NoParent {
		t.Errorf("a mapping of no parents covers %d rows and gives row 0 parent %d",
			owned.Rows(), owned.ParentOf(0))
	}
}
