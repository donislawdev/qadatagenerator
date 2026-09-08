package manifestview

import (
	"errors"
	"io"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Every column of the entries table is wide enough for its heading and for the
// widest value it can ever hold.
//
// 🔴 WRITTEN AFTER THE DEFECT IT GUARDS, measured on 2026-09-08. The column for
// the certainty of an expectation had 340 pixels to hold one of two words, the
// wider of which renders 36 across, while the column beside it had 220 to hold a
// phrase that renders 210. Ten pixels of room on the longest fixed phrase in the
// table, next to two hundred and seventy five spare on the shortest, on a table
// already wider than the window it sits in.
//
// The comment above those numbers described the proportion correctly in words
// and had done since the day it was written. Prose was checked and arithmetic
// was not, which is the thing this asks about.
//
// # Why only these three columns
//
// They are the ones whose values come from a closed set, so the widest value is
// a fact rather than a sample. What a damage is called and what changed in a
// value are as long as somebody's data, and a column for those is a judgement
// about how much to show rather than a size that can be right or wrong. Every
// heading is checked, because there are six of those and they are all fixed.
func TestEveryColumnOfTheEntriesTableFitsWhatItCanHold(t *testing.T) {
	// Room to spare rather than merely fitting. Text touching the edge of its
	// cell reads as text that was cut, so a column that fits to the pixel is one
	// nobody can tell from a column that does not.
	spare := palette.StepOf(palette.NameStepTight)

	// 🔴 Measured with the text that does NOT trim itself, and the first version
	// of this used the one that does. A cell of this table trims, so it reports
	// the width it can be squeezed to rather than the width of what is in it,
	// and asking it how wide a phrase is answers a different question in a
	// perfectly plausible number. That version was green against the very
	// widths it was written to catch, which only came out because it was run
	// with the defect put back.
	fits := func(t *testing.T, column int, what, words string) {
		t.Helper()
		width := plain(Line{Text: words}, false).MinSize().Width
		if width+spare > palette.EntryColumnWidths[column] {
			t.Errorf("column %d is %.0f wide and %s renders %.0f, which leaves %.0f "+
				"of room where %.0f is asked for", column,
				palette.EntryColumnWidths[column], what, width,
				palette.EntryColumnWidths[column]-width, spare)
		}
	}

	for column := range columns {
		fits(t, column, "its heading", i18n.English.T(columns[column].Name))
	}

	// The closed sets, each named by the column it belongs to, taken from the
	// catalogue rather than written out here so that a phrase reworded in the
	// catalogue is measured in its new wording.
	for _, held := range []struct {
		column int
		keys   []i18n.Key
	}{
		{2, []i18n.Key{i18n.ManifestExpectationRejection, i18n.ManifestExpectationAcceptance,
			i18n.ManifestExpectationWarning}},
		{3, []i18n.Key{i18n.ManifestConfidenceHard, i18n.ManifestConfidenceSoft}},
		{4, []i18n.Key{i18n.ManifestEntryApplied, i18n.ManifestEntryNotApplied}},
	} {
		for _, key := range held.keys {
			words := i18n.English.T(key)
			fits(t, held.column, "the value "+words, words)
		}
	}
}

// chosenFile stands in for what a file chooser hands back.
//
// A test cannot open a chooser, and the states worth checking here are the ones
// around it: a choice made, a choice cancelled, and a chooser that failed.
type chosenFile struct{ path string }

func (c chosenFile) Read([]byte) (int, error) { return 0, io.EOF }
func (c chosenFile) Close() error             { return nil }
func (c chosenFile) URI() fyne.URI            { return storage.NewFileURI(c.path) }

// The screen draws in both of its states, and every part of the drawing is
// reached.
//
// It is worth having as much for what it does not assert as for what it does:
// it renders rather than inspects, so it holds the drawing code to running at
// all against a real manifest. What each part says is checked next door,
// against the model, which is where those questions can be asked precisely.
func TestTheScreenDrawsInBothStates(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	if view.Content() == nil {
		t.Fatal("the screen builds nothing at all")
	}

	promised, problems := manifest.New("0.1", manifest.Run{
		GeneratedAt: "2026-08-18T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 2, RowsMutated: 1,
	}, []manifest.Defect{{
		ID:       "one",
		Location: manifest.Location{Entity: "customer", Row: 1, Field: "pesel"},
		Mutator:  "L3.checksum.off_by_one", Layer: catalog.Domain, Applied: true,
		ValueBefore: "44051401359", HasValueBefore: true,
		ValueAfter: "44051401358", HasValueAfter: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the check digit no longer agrees",
	}})
	if len(problems) > 0 {
		t.Fatalf("the manifest this test is built on is not valid: %v", problems)
	}

	if view.rendered(Build(i18n.English, promised, nil)) == nil {
		t.Error("the first state draws nothing")
	}

	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Accepted}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	second := Build(i18n.English, promised, &comparison)
	if view.rendered(second) == nil {
		t.Error("the second state draws nothing")
	}
	// The footer of the second state carries the legend and whatever the
	// comparison could not settle, so it is never absent once a response is in.
	if view.foot(second) == nil {
		t.Error("the second state draws no footer, so the legend and the limits have nowhere to be")
	}
}

// A column nobody defined is headed with nothing rather than taking the view
// down, which is the same bargain the row makes about its cells.
func TestAColumnWithNoHeadingIsEmptyRatherThanFatal(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	for _, absent := range []int{-1, len(columns)} {
		if got := view.columnName(absent); got != "" {
			t.Errorf("column %d is headed %q", absent, got)
		}
	}
}

// A chooser that failed says so, a cancelled one changes nothing, and a choice
// made hands back its path.
//
// 🔴 The middle case is the one worth writing down. Turning a cancellation into
// an error message would tell a person something went wrong when they were the
// one who decided nothing should.
func TestACancelledChooserChangesNothingAndAFailedOneSaysSo(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	before := view.body.Objects[0]

	if _, ok := view.pathOf(nil, nil); ok {
		t.Error("a cancelled chooser was treated as a choice")
	}
	if view.body.Objects[0] != before {
		t.Error("a cancelled chooser changed the screen")
	}

	if _, ok := view.pathOf(nil, errors.New("the chooser could not open")); ok {
		t.Error("a chooser that failed was treated as a choice")
	}
	if view.body.Objects[0] == before {
		t.Error("a chooser that failed left the screen saying nothing about it")
	}

	path, ok := view.pathOf(chosenFile{path: "/tmp/expectation-manifest.json"}, nil)
	if !ok {
		t.Fatal("a file that was chosen was not treated as a choice")
	}
	if !strings.HasSuffix(path, "expectation-manifest.json") {
		t.Errorf("the chosen file came back as %q", path)
	}
}

// A document that cannot be read leaves the screen saying why, and leaves the
// control for the second state unavailable, because there is nothing to compare
// a response against.
func TestADocumentThatCannotBeReadLeavesTheScreenSayingWhy(t *testing.T) {
	view := New(i18n.English, nil, Readers{
		Manifest: func(string) (manifest.Manifest, error) {
			return manifest.Manifest{}, errors.New("this is not a manifest")
		},
	}, background.ReadHere)
	promised, err := view.read.Manifest("whatever")
	if err == nil {
		t.Fatal("the reader handed in for this test did not fail, so the test checks nothing")
	}
	_ = promised
	if !view.respond.Disabled() {
		t.Error("the control for the second state is available with no manifest open")
	}
}

// Opening a manifest from outside reaches the same state as the chooser does.
//
// 🔴 This is the guard for a drift that happened within the hour on 2026-08-18.
// The rendering probe in tools needs the states of this screen without a file
// chooser, and the first exported way in only drew the screen. Drawing is not
// the state: the button that opens a response record stays disabled, so the
// probe photographed a screen no person can reach by opening a manifest, and
// the picture was wrong in a way that reads as correct.
func TestOpeningAManifestFromOutsideReachesTheSameStateAsTheChooser(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	if !view.respond.Disabled() {
		t.Fatal("the response button is available before there is a manifest to judge against")
	}

	promised, problems := manifest.New("0.1", manifest.Run{
		GeneratedAt: "2026-08-18T00:00:00Z", Seed: "s", Locale: "pl-PL",
		CatalogVersion: "0.1", RowsTotal: 1, RowsMutated: 1,
	}, []manifest.Defect{{
		ID:       "one",
		Location: manifest.Location{Entity: "customer", Row: 1, Field: "pesel"},
		Mutator:  "L3.checksum.off_by_one", Layer: catalog.Domain, Applied: true,
		ValueBefore: "44051401359", HasValueBefore: true,
		ValueAfter: "44051401358", HasValueAfter: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "the check digit no longer agrees",
	}})
	if len(problems) > 0 {
		t.Fatalf("the manifest this test is built on is not valid: %v", problems)
	}

	view.Open(promised)
	if view.respond.Disabled() {
		t.Error("a manifest is open and the response button is still unavailable, so this way in " +
			"reaches a state the chooser never leaves behind")
	}
	if !view.hasPromise {
		t.Error("a manifest is on the screen and the screen does not know it holds one")
	}

	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Rejected}}}
	if !view.Compare(actual) {
		t.Error("a record was refused although there is a manifest to judge it against")
	}
}

// A record with no manifest beside it is refused rather than drawn.
//
// The comparison has nothing to say without a manifest, and a screen showing it
// would be saying it anyway.
func TestAResponseRecordWithNoManifestIsRefused(t *testing.T) {
	view := New(i18n.English, nil, Readers{}, background.ReadHere)
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Accepted}}}
	if view.Compare(actual) {
		t.Error("a record was compared against a manifest that is not there")
	}
}
