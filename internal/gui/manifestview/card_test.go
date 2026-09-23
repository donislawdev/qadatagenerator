package manifestview

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
)

// threeEntries is the screen with a manifest of three entries open, in a
// window, so that the table has a size and a pointer has somewhere to be. The
// last two land on one row, because the shared envelope says two rows changed.
func threeEntries(t *testing.T) (*View, fyne.Window) {
	t.Helper()
	view := New(i18n.English, nil, Readers{}, background.ReadHere, nil)
	view.Open(built(t,
		applied("d1", 1, catalog.Rejection, catalog.Hard),
		applied("d2", 2, catalog.Rejection, catalog.Hard),
		applied("d3", 2, catalog.Rejection, catalog.Soft),
	), "")
	window := test.NewWindow(view.Content())
	window.Resize(fyne.NewSize(1342, 900))
	return view, window
}

// rowPoints finds a point inside each row of the table by clicking down its
// first column, the way a person meets the table.
//
// The toolkit's geometry is its own and unexported - the height of its heading
// row, the gap between rows - so it is measured here rather than copied. A
// copy would go on passing the day the toolkit changed its padding, clicking
// between rows and proving nothing.
func rowPoints(t *testing.T, view *View) map[int]fyne.Position {
	t.Helper()
	points := map[int]fyne.Position{}
	height := view.entries.Size().Height
	for y := float32(1); y < height; y += 2 {
		at := fyne.NewPos(8, y)
		view.entries.Tapped(&fyne.PointEvent{Position: at})
		if _, seen := points[view.selected]; view.hasSelected && !seen {
			points[view.selected] = at
		}
	}
	if len(points) != len(view.screen.Rows) {
		t.Fatalf("clicking down the table reached %d of its %d rows, so this test cannot point at them",
			len(points), len(view.screen.Rows))
	}
	return points
}

func hover(view *View, at fyne.Position) {
	event := &desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at}}
	view.entries.MouseIn(event)
	view.entries.MouseMoved(event)
}

// Passing over an entry with the pointer lights it and chooses nothing: the
// card goes on reading out the entry that was clicked (DEC-249).
//
// 🔴 Measured on the real window on 2026-09-23 before this was true (B-287):
// the card read out whichever entry the pointer last crossed and stayed there
// after it left the table, beside the violet of the entry somebody had clicked.
// A person moving from the entry they clicked down to the card to read it
// crossed every entry in between and read the last of them. The toolkit calls
// the same function for its hover as for the keyboard, so the line keeping
// them apart is in parts.KeyTable, and this is what holds it.
func TestThePointerPassingOverAnEntryOnlyLightsIt(t *testing.T) {
	view, window := threeEntries(t)
	defer window.Close()
	points := rowPoints(t, view)

	view.entries.Tapped(&fyne.PointEvent{Position: points[0]})
	if !view.hasSelected || view.selected != 0 {
		t.Fatalf("a click on the first entry left the card on entry %d, chosen %v", view.selected, view.hasSelected)
	}
	hover(view, points[1])
	hover(view, points[2])
	view.entries.MouseOut()
	if view.selected != 0 {
		t.Errorf("the pointer passed over entry %d and the card reads it out in place of the one clicked",
			view.selected+1)
	}
	if !saying(view.detailBox, "entry 1 of 3") {
		t.Errorf("after the pointer passed over the table the card says %q", texts(view.detailBox))
	}

	// 🔴 The keyboard is heard again once the pointer has gone. The function
	// is set aside for the length of a pointer's event and has to come back.
	view.entries.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if view.selected != 1 {
		t.Errorf("the keyboard moved down after the pointer left and the card reads out entry %d",
			view.selected+1)
	}
}

// Loading what an application did keeps the entry that was chosen, and the
// card reads it out again with the answer added (B-288).
//
// 🔴 Measured on the real window on 2026-09-23: the card went back to the
// sentence asking for an entry to be chosen. The entries are the same entries,
// and rule 3 of the window says a refresh resets nothing.
func TestTheChosenEntrySurvivesLoadingWhatTheApplicationDid(t *testing.T) {
	view, window := threeEntries(t)
	defer window.Close()
	window.Canvas().Focus(view.entries)
	view.entries.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
	if view.selected != 1 {
		t.Fatalf("the keyboard moved to the second entry and the card reads out entry %d", view.selected+1)
	}
	table := view.entries

	view.Compare(response.Record{SpecVersion: "0.1", Rows: []response.Row{
		{Number: 1, Verdict: response.Rejected},
		{Number: 2, Verdict: response.Accepted},
		{Number: 3, Verdict: response.Rejected},
	}}, "")

	if view.entries != table {
		t.Error("the table was built again for the same entries, which is what takes its scroll and selection away")
	}
	if !view.hasSelected || view.selected != 1 {
		t.Fatalf("after the answer was loaded the card reads out entry %d, chosen %v",
			view.selected+1, view.hasSelected)
	}
	finding := view.screen.Rows[1].Finding.Text
	if finding == "" || !saying(view.detailBox, finding) {
		t.Errorf("the card does not read out what the application did about the chosen entry, %q", finding)
	}
}

// The envelope of the run takes the place of the entry in the card, the
// heading says which of the two the card holds, and the window asks for no
// more height with the envelope than without it (DEC-249).
//
// 🔴 Over the table the envelope asked for 172 more pixels of the real window
// beside a manifest and 269 beside a comparison, past what the owner's screen
// has (B-286). The height is asked of the whole content in both states,
// because the comparison is the state that broke.
func TestTheRunTakesThePlaceOfTheEntryAndTheWindowDoesNotGrow(t *testing.T) {
	for _, compared := range []bool{false, true} {
		view, window := threeEntries(t)
		if compared {
			view.Compare(response.Record{SpecVersion: "0.1", Rows: []response.Row{
				{Number: 1, Verdict: response.Rejected},
			}}, "")
		}
		before := view.Content().MinSize().Height
		view.ShowRun()
		after := view.Content().MinSize().Height
		if after > before {
			t.Errorf("compared %v: the envelope makes the screen ask for %.0f units of height where it asked for %.0f",
				compared, after, before)
		}
		if !saying(view.detailBox, i18n.English.T(i18n.ManifestRunGenerated)) {
			t.Errorf("compared %v: the envelope is open and the card says %q", compared, texts(view.detailBox))
		}
		if !saying(view.cardHeading, i18n.English.T(i18n.ManifestHeadingRun)) ||
			view.about.Text != i18n.English.T(i18n.ManifestShowEntry) {
			t.Errorf("compared %v: the heading says %q and its button %q while the card holds the run",
				compared, texts(view.cardHeading), view.about.Text)
		}
		window.Close()
	}
}

// Choosing an entry puts it back in the card in place of the run, and so does
// the button that put the run there - including a click on the very entry that
// was chosen before.
//
// 🔴 That last one is the case the toolkit would lose. It selects nothing that
// is already selected, so a click on the entry chosen before the run was shown
// would say nothing to the screen, and the card would go on saying what the
// run was under a person clicking an entry to read it.
func TestChoosingAnEntryPutsItBackInPlaceOfTheRun(t *testing.T) {
	view, window := threeEntries(t)
	defer window.Close()
	points := rowPoints(t, view)
	view.entries.Tapped(&fyne.PointEvent{Position: points[1]})

	view.ShowRun()
	view.entries.Tapped(&fyne.PointEvent{Position: points[1]})
	if view.showingRun || view.selected != 1 {
		t.Errorf("a click on the entry chosen before left the card on the run %v, reading entry %d",
			view.showingRun, view.selected+1)
	}
	if !saying(view.cardHeading, i18n.English.T(i18n.ManifestHeadingEntry)) {
		t.Errorf("the card reads out an entry under a heading saying %q", texts(view.cardHeading))
	}

	view.ShowRun()
	view.about.Tapped(&fyne.PointEvent{})
	if view.showingRun || view.selected != 1 {
		t.Errorf("the button bringing the entry back left the card on the run %v, reading entry %d",
			view.showingRun, view.selected+1)
	}
	if !saying(view.detailBox, "entry 2 of 3") {
		t.Errorf("the entry is back and the card says %q", texts(view.detailBox))
	}
}
