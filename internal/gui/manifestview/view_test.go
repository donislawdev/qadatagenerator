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
)

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
	view := New(i18n.English, nil, Readers{})
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
	view := New(i18n.English, nil, Readers{})
	for _, absent := range []int{-1, len(columnNames)} {
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
	view := New(i18n.English, nil, Readers{})
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
	})
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
	view := New(i18n.English, nil, Readers{})
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
	view := New(i18n.English, nil, Readers{})
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Accepted}}}
	if view.Compare(actual) {
		t.Error("a record was compared against a manifest that is not there")
	}
}
