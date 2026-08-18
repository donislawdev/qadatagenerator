package manifestview

import (
	"regexp"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// A run with everything the envelope has to carry, so that a test can say what
// it is about rather than repeat the required fields.
func run() manifest.Run {
	return manifest.Run{
		GeneratedAt:    "2026-08-17T10:00:00Z",
		Seed:           "a-seed",
		Locale:         "pl-PL",
		CatalogVersion: "0.1",
		RowsTotal:      3,
		RowsMutated:    2,
	}
}

// applied is an entry that happened, in a row and a column.
func applied(id string, row int, expectation catalog.Expectation, confidence catalog.Confidence) manifest.Defect {
	return manifest.Defect{
		ID:             id,
		Location:       manifest.Location{Entity: "customer", Row: row, Field: "pesel"},
		Mutator:        "L3.checksum.off_by_one",
		Layer:          catalog.Domain,
		Applied:        true,
		ValueBefore:    "44051401359",
		HasValueBefore: true,
		ValueAfter:     "44051401358",
		HasValueAfter:  true,
		Expectation:    expectation,
		Confidence:     confidence,
		Rationale:      "the check digit no longer agrees with the number",
	}
}

// unapplied is an entry that did not happen, with both halves of the answer.
func unapplied(id string, kind manifest.FailureKind, why string) manifest.Defect {
	return manifest.Defect{
		ID:            id,
		Location:      manifest.Location{Entity: "customer", Row: 2, Field: "iban"},
		Mutator:       "L3.checksum.off_by_one",
		Layer:         catalog.Domain,
		Expectation:   catalog.Rejection,
		Confidence:    catalog.Hard,
		Rationale:     "nothing was changed",
		FailureReason: why,
		FailureKind:   kind,
	}
}

func built(t *testing.T, defects ...manifest.Defect) manifest.Manifest {
	t.Helper()
	promised, problems := manifest.New("0.1", run(), defects)
	if len(problems) > 0 {
		t.Fatalf("the manifest this test is built on is not valid: %v", problems)
	}
	return promised
}

func screenOf(t *testing.T, comparison *Comparison, defects ...manifest.Defect) Screen {
	t.Helper()
	return Build(i18n.English, built(t, defects...), comparison)
}

// everything a person could read off the screen, so that a test asks about what
// is shown rather than about which member happens to hold it.
func said(screen Screen) []string {
	var all []string
	for _, field := range append(append([]Field{}, screen.Run...), screen.Counts...) {
		all = append(all, field.Label, field.Value)
	}
	for _, line := range append(append(append([]Line{}, screen.Notes...), screen.Unsettled...), screen.Limits...) {
		all = append(all, line.Text)
	}
	for _, row := range screen.Rows {
		all = append(all, row.Mutator, row.Place, row.Status.Text, row.Expect.Text,
			row.Confidence.Text, row.Reason.Text, row.Values.Text, row.Answer.Text)
	}
	return all
}

func says(screen Screen, want string) bool {
	for _, one := range said(screen) {
		if want != "" && strings.Contains(one, want) {
			return true
		}
	}
	return false
}

// Every ordered damage is on the screen, applied or not.
//
// 🔴 This is the ban on silence at the last step before a person. The model
// already refuses an entry that says nothing, and a screen quietly showing only
// the entries that worked would undo that at the one place it matters: a
// dataset where twenty seven of forty ordered defects happened looks exactly
// like a complete one, and it will be trusted as a complete one.
func TestEveryOrderedDamageIsOnTheScreenWhateverBecameOfIt(t *testing.T) {
	screen := screenOf(t, nil,
		applied("d1", 1, catalog.Rejection, catalog.Hard),
		unapplied("d2", manifest.SchemeNotCovered, "the column follows no scheme this damage suits"),
		applied("d3", 3, catalog.Acceptance, catalog.Soft),
	)
	if len(screen.Rows) != 3 {
		t.Fatalf("three damages were ordered and the screen shows %d rows", len(screen.Rows))
	}
	for _, count := range screen.Counts {
		if count.Value == "" {
			t.Errorf("the count %q has no value beside it", count.Label)
		}
	}
	if !says(screen, "3") {
		t.Error("the screen does not say how many damages were ordered")
	}
}

// An entry that did not happen says why, in both the words a person reads and
// the closed kind a machine acts on.
//
// DEC-152 asks for the kind beside every unapplied damage in the window, and it
// is the promise of the whole product: the reason is on the screen rather than
// behind a pointer, and Fyne 2.8 has no tooltip to hide it in anyway.
func TestAnUnappliedEntrySaysWhyOnTheScreen(t *testing.T) {
	why := "the column follows no scheme this damage suits"
	screen := screenOf(t, nil, unapplied("d1", manifest.SchemeNotCovered, why))

	row := screen.Rows[0]
	if row.Reason.Text == "" {
		t.Fatal("an entry that did not happen carries no reason on the screen")
	}
	// The sentence the run recorded and the closed kind are both on the screen
	// and in different columns, so neither repeats the other in every row.
	if !strings.Contains(row.Reason.Text, why) {
		t.Errorf("the reason on screen is %q and does not carry what the run recorded", row.Reason.Text)
	}
	if row.Status.Text != i18n.English.T(i18n.ManifestFailureSchemeNotCovered) {
		t.Errorf("the outcome on screen is %q and does not name the kind of failure", row.Status.Text)
	}
	if strings.Contains(row.Reason.Text, row.Status.Text) {
		t.Errorf("the reason %q repeats the outcome %q, so a column of sixty entries says the same "+
			"thing twice in every row", row.Reason.Text, row.Status.Text)
	}
	// It has to stand out rather than fade. A row that did not happen is the
	// one a reader most needs to find without reading.
	if row.Reason.Color != palette.NameNotApplied || row.Status.Color != palette.NameNotApplied {
		t.Errorf("an unapplied entry is painted %q and %q rather than in the colour meant to stand out",
			row.Status.Color, row.Reason.Color)
	}
}

// A soft expectation looks different from a hard one, and not merely paler.
//
// Rule five of the handbook says the manifest does not pretend to a certainty
// it does not have, and this is that rule at the pixel. Paler reads as less
// important. Soft does not mean less important, it means the right answer
// belongs to the application being tested.
func TestASoftExpectationDoesNotLookLikeAHardOne(t *testing.T) {
	screen := screenOf(t, nil,
		applied("d1", 1, catalog.Rejection, catalog.Hard),
		applied("d2", 2, catalog.Rejection, catalog.Soft),
	)
	hard, soft := screen.Rows[0].Confidence, screen.Rows[1].Confidence
	if hard.Text == soft.Text {
		t.Fatalf("both confidences read %q, so nothing on screen tells them apart", hard.Text)
	}
	if !strings.ContainsAny(soft.Text, "~") {
		t.Errorf("the soft confidence reads %q and carries no sign of its own, so it differs by "+
			"wording alone where the rule asks for a mark", soft.Text)
	}
}

// Every value of every closed vocabulary has a phrase, and every phrase belongs
// to a value.
//
// 🔴 Closed in both directions and against the domain rather than against a
// list kept here. A register answering only one of those two questions answers
// half of it, which this project has already paid for. The half that would hurt
// here is the first: a seventh failure kind would reach a person as an empty
// space, and the ban on silence would be undone at the last step by the surface
// they actually read.
func TestEveryClosedVocabularyOfTheDomainHasAPhrase(t *testing.T) {
	expectations := map[catalog.Expectation]bool{}
	for _, expectation := range catalog.Expectations() {
		expectations[expectation] = true
		if _, known := expectationKeys[expectation]; !known {
			t.Errorf("the expectation %q has no phrase, so it would reach a screen as an empty space", expectation)
		}
		if _, known := expectationColors[expectation]; !known {
			t.Errorf("the expectation %q has no colour", expectation)
		}
	}
	for expectation := range expectationKeys {
		if !expectations[expectation] {
			t.Errorf("a phrase names the expectation %q, which the domain does not have", expectation)
		}
	}

	confidences := map[catalog.Confidence]bool{}
	for _, confidence := range catalog.Confidences() {
		confidences[confidence] = true
		if _, known := confidenceKeys[confidence]; !known {
			t.Errorf("the confidence %q has no phrase", confidence)
		}
	}
	for confidence := range confidenceKeys {
		if !confidences[confidence] {
			t.Errorf("a phrase names the confidence %q, which the domain does not have", confidence)
		}
	}

	kinds := map[manifest.FailureKind]bool{}
	for _, kind := range manifest.FailureKinds() {
		kinds[kind] = true
		if _, known := failureKeys[kind]; !known {
			t.Errorf("the failure kind %q has no phrase, so an unapplied entry carrying it would say "+
				"nothing a person can read", kind)
		}
	}
	for kind := range failureKeys {
		if !kinds[kind] {
			t.Errorf("a phrase names the failure kind %q, which the format does not define", kind)
		}
	}

	disagreements := map[verify.Kind]bool{}
	for _, kind := range verify.Kinds() {
		disagreements[kind] = true
		if _, known := kindKeys[kind]; !known {
			t.Errorf("the disagreement %q has no phrase", kind)
		}
	}
	for kind := range kindKeys {
		if !disagreements[kind] {
			t.Errorf("a phrase names the disagreement %q, which the comparison does not report", kind)
		}
	}
}

// The envelope is on the screen, because it is what tells one manifest from the
// one beside it a week later.
func TestTheEnvelopeSaysWhatTheRunWas(t *testing.T) {
	screen := screenOf(t, nil, applied("d1", 1, catalog.Rejection, catalog.Hard))
	for _, want := range []string{"a-seed", "pl-PL", "0.1", "2026-08-17T10:00:00Z"} {
		if !says(screen, want) {
			t.Errorf("the screen does not carry %q from the envelope", want)
		}
	}
}

// An optional reference the run did not carry is left out rather than shown
// empty, because a label against nothing is a reader wondering which of the two
// it means.
func TestAReferenceTheRunNeverCarriedIsNotShownEmpty(t *testing.T) {
	screen := screenOf(t, nil, applied("d1", 1, catalog.Rejection, catalog.Hard))
	for _, field := range screen.Run {
		if field.Label == i18n.English.T(i18n.ManifestRunTag) {
			t.Error("the run carried no tag and the screen shows a row for one")
		}
	}
}

// Damage to the whole file names no place, and the screen says so.
//
// An empty cell would read as a place somebody forgot to record, which is the
// opposite of what an empty location means at the byte layer.
func TestDamageToTheWholeFileSaysSoRatherThanShowingNothing(t *testing.T) {
	wholeFile := manifest.Defect{
		ID: "d1", Mutator: "L0.encoding.bom_added", Layer: catalog.Bytes, Applied: true,
		Expectation: catalog.Rejection, Confidence: catalog.Hard,
		Rationale: "a byte order mark was put in front of the document",
	}
	screen := screenOf(t, nil, wholeFile)
	if screen.Rows[0].Place != i18n.English.T(i18n.ManifestPlaceWholeFile) {
		t.Errorf("damage to the whole file shows its place as %q", screen.Rows[0].Place)
	}
}

// A manifest that ordered nothing is a screen that says so through its counts
// rather than a screen with a hole in it.
func TestAManifestThatOrderedNothingStillShowsItsRun(t *testing.T) {
	screen := screenOf(t, nil)
	if len(screen.Rows) != 0 {
		t.Errorf("nothing was ordered and the screen shows %d rows", len(screen.Rows))
	}
	if !says(screen, "a-seed") {
		t.Error("a manifest with no entries shows nothing about the run it describes")
	}
}

// The second state shows what the application did with every row that could be
// judged.
//
// 🔴 This is the test the parity register names for the verify capability. The
// window reaching verify means exactly this: the same list of entries, each one
// carrying how the application answered, with a hard disagreement told apart
// from a soft one. A screen showing only the disagreements would leave a reader
// unable to separate a row that agreed from one nothing was said about, and
// those two are what this project most insists on keeping apart.
func TestTheSecondStateShowsWhatTheApplicationDidWithEveryJudgedRow(t *testing.T) {
	promised := built(t,
		applied("d1", 1, catalog.Rejection, catalog.Hard),
		applied("d2", 2, catalog.Rejection, catalog.Hard),
	)
	actual := response.Record{
		SpecVersion: "0.1",
		Rows: []response.Row{
			// Row one was taken although the manifest expected a refusal, which
			// is the hole this tool exists to find.
			{Number: 1, Verdict: response.Accepted},
			{Number: 2, Verdict: response.Rejected},
		},
	}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	screen := Build(i18n.English, promised, &comparison)

	for index, row := range screen.Rows {
		if row.Answer.Text == "" {
			t.Errorf("row %d carries nothing about what the application did", index+1)
		}
	}
	hole := screen.Rows[0].Answer
	if !strings.Contains(hole.Text, i18n.English.T(i18n.VerifyKindHole)) {
		t.Errorf("the row the application wrongly accepted reads %q and does not name the disagreement",
			hole.Text)
	}
	if hole.Color != palette.NameRejection {
		t.Errorf("a disagreement is painted %q rather than in the colour of a rejection", hole.Color)
	}
	agreed := screen.Rows[1].Answer
	if strings.Contains(agreed.Text, i18n.English.T(i18n.VerifyKindHole)) {
		t.Errorf("the row the application answered as expected reads %q and names a disagreement", agreed.Text)
	}
	if !says(screen, i18n.English.T(i18n.VerifyVerdictRejected)) {
		t.Error("the screen does not say that the application rejected the row it was expected to reject")
	}
}

// A row the response record never mentions is said to be unanswered rather than
// left looking like agreement.
func TestARowTheApplicationNeverAnsweredForSaysSo(t *testing.T) {
	promised := built(t, applied("d1", 1, catalog.Rejection, catalog.Hard))
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 2, Verdict: response.Rejected}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	screen := Build(i18n.English, promised, &comparison)

	if screen.Rows[0].Answer.Text != i18n.English.T(i18n.VerifyNoAnswer) {
		t.Errorf("a row nothing was said about reads %q", screen.Rows[0].Answer.Text)
	}
}

// The second state carries what it could not settle and what it could not see.
//
// 🔴 Both, and neither is decoration. A comparison reporting only its findings
// reads as a complete account of the run, and an unanswered question read as a
// pass is the failure this whole project argues against.
func TestTheSecondStateCarriesWhatItCouldNotSettleAndCouldNotSee(t *testing.T) {
	promised := built(t, applied("d1", 1, catalog.Warning, catalog.Soft))
	// A row with no warnings recorded cannot settle an expectation of a mark,
	// and a record that says nothing about what is held now cannot tell a value
	// taken as sent from one quietly changed.
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Accepted}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	screen := Build(i18n.English, promised, &comparison)

	if len(screen.Unsettled) == 0 {
		t.Error("an expectation of a mark against a record carrying no warnings is settled on screen, " +
			"and nothing could have settled it")
	}
	if len(screen.Limits) == 0 {
		t.Error("the screen states no limit against a record that carries neither stored values nor warnings")
	}
	if len(screen.Notes) == 0 {
		t.Error("the second state says nothing about how many entries were compared")
	}
}

// A finding about the dataset as a whole is not attached to one arbitrary row
// of it.
func TestAFindingAboutTheWholeDatasetIsNotHungOnOneRow(t *testing.T) {
	promised := built(t, applied("d1", 1, catalog.Rejection, catalog.Hard))
	// The manifest says three rows and the application answered for one, so two
	// records never arrived at all.
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Rejected}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	screen := Build(i18n.English, promised, &comparison)

	if !says(screen, i18n.English.T(i18n.VerifyKindMissing)) {
		t.Fatal("the screen does not say that records never reached the application")
	}
	if strings.Contains(screen.Rows[0].Answer.Text, i18n.English.T(i18n.VerifyKindMissing)) {
		t.Error("a statement about the whole dataset is shown against one row of it")
	}
}

// The first state says nothing about an application, because none has seen the
// data yet.
func TestTheFirstStateClaimsNothingAboutAnyApplication(t *testing.T) {
	screen := screenOf(t, nil, applied("d1", 1, catalog.Rejection, catalog.Hard))
	if screen.Rows[0].Answer.Text != "" {
		t.Errorf("a manifest shown on its own carries %q about an application", screen.Rows[0].Answer.Text)
	}
	if len(screen.Notes) != 0 || len(screen.Unsettled) != 0 || len(screen.Limits) != 0 {
		t.Error("a manifest shown on its own carries the sections that belong to a comparison")
	}
}

// No phrase on the screen renders as its own translation key.
//
// The shape is what is checked rather than a list of known keys. A key nobody
// declared renders as a dotted identifier and a test comparing against the
// declared list would not know to look for it, which is a mistake this project
// has already made once on the About screen.
func TestNoPhraseRendersAsItsOwnKey(t *testing.T) {
	looksLikeAKey := regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9-]+)+$`)
	promised := built(t, applied("d1", 1, catalog.Rejection, catalog.Hard),
		unapplied("d2", manifest.DeclinedByMutator, "the value already sits in the block this aims at"))
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Accepted}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}

	for _, screen := range []Screen{
		Build(i18n.English, promised, nil),
		Build(i18n.English, promised, &comparison),
	} {
		for _, phrase := range said(screen) {
			if looksLikeAKey.MatchString(phrase) {
				t.Errorf("the phrase %q reads as a translation key rather than as a sentence", phrase)
			}
		}
	}
}
