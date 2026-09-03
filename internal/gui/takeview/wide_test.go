package takeview

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/take"
)

// wideSource writes a file of the given width, which is what the panel
// describing a file grows with.
func wideSource(t testing.TB, dir string, columns int) string {
	t.Helper()
	var text strings.Builder
	for column := range columns {
		if column > 0 {
			text.WriteString(separator)
		}
		text.WriteString("c" + strconv.Itoa(column))
	}
	text.WriteString("\n")
	for row := 1; row <= 3; row++ {
		for column := range columns {
			if column > 0 {
				text.WriteString(separator)
			}
			text.WriteString("v" + strconv.Itoa(row) + "_" + strconv.Itoa(column))
		}
		text.WriteString("\n")
	}
	path := filepath.Join(dir, "wide.csv")
	if err := os.WriteFile(path, []byte(text.String()), 0o600); err != nil {
		t.Fatalf("cannot write the wide file this measures: %v", err)
	}
	return path
}

// TestHowMuchThePanelAboutAFileCostsAtWidth is a probe rather than a guard.
//
// It reports what the panel describing somebody else's file costs as the file
// gets wider, because the entry recorded for it said the shape was known and
// the cost was not measured. There is no threshold here: the number depends on
// the machine, and what a threshold would be worth is a separate question from
// what the cost is.
//
// Run it with -v to see the numbers.
func TestHowMuchThePanelAboutAFileCostsAtWidth(t *testing.T) {
	if testing.Short() {
		t.Skip("this walks files wide enough to be slow, and it measures rather than guards")
	}
	for _, columns := range []int{10, 100, 1000, 5000} {
		t.Run(strconv.Itoa(columns)+" columns", func(t *testing.T) {
			dir := t.TempDir()
			window := test.NewWindow(nil)
			defer window.Close()
			view, _ := opened(t, window, wideSource(t, dir, columns), dir)

			var before, after runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&before)
			drawn := view.conclusions()
			runtime.ReadMemStats(&after)

			t.Logf("%d columns: %d objects at the top level, %d bytes allocated building them",
				columns, len(drawn), after.TotalAlloc-before.TotalAlloc)
		})
	}
}

// The panel about a file accounts for as many columns as this build lists, and
// counts the rest rather than dropping them.
//
// 🔴 Both sides are the test, as they are for the log describing the same file
// and for the manifest. One column here is six objects and the panel lays out
// every one of them whether it is on screen or not, so a wide file is a cost
// nobody asked for: measured on 2026-09-03 at 1088 bytes per column, which is
// 5.4 MB at five thousand columns. A bound that bit on ordinary files would be
// a cut instead, so a file narrower than the limit has every column described.
func TestThePanelAccountsForAsManyColumnsAsThisBuildListsAndCountsTheRest(t *testing.T) {
	for _, given := range []struct {
		what    string
		columns int
		left    int
	}{
		{"a file narrower than the limit has every column described", 6, 0},
		{"a file exactly at the limit has every column described", take.ColumnsListed, 0},
		{"a file one column wider counts the one it does not describe",
			take.ColumnsListed + 1, 1},
		{"a much wider file counts all of them",
			take.ColumnsListed + 950, 950},
	} {
		t.Run(given.what, func(t *testing.T) {
			dir := t.TempDir()
			window := test.NewWindow(nil)
			defer window.Close()
			view, _ := opened(t, window, wideSource(t, dir, given.columns), dir)

			drawn := view.conclusions()
			read := texts(container.NewVBox(drawn...))
			// Counted by what a person reads rather than by the shape of the
			// tree. The first version counted containers holding two things and
			// was one out on every case, because the notice about nothing being
			// recognised is one of those too.
			described := 0
			for _, line := range read {
				if strings.Contains(line, takenFor) {
					described++
				}
			}
			said := strings.Join(read, " ")

			want := min(given.columns, take.ColumnsListed)
			if described != want {
				t.Errorf("the panel describes %d columns of %d, want %d",
					described, given.columns, want)
			}
			if given.left == 0 {
				if strings.Contains(said, moreColumns) {
					t.Errorf("a file of %d columns is described in full and still counts a "+
						"remainder: %q", given.columns, said)
				}
				return
			}
			if !strings.Contains(said, moreColumns+": "+strconv.Itoa(given.left)) {
				t.Errorf("the panel does not count the %d columns it did not describe: %q",
					given.left, said)
			}
		})
	}
}

// The English of the two lines this guard reads, so that it reads what a person
// reads rather than the shape of the tree behind them.
const (
	moreColumns = "More columns than are accounted for here"
	takenFor    = "taken for"
)
