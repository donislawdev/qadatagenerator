package manifestview

import (
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

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

// everyKindOfScreen is both states built with every kind of entry the model
// can produce: hard and soft, applied and not, agreed with, disagreed with,
// unanswered and unsettled.
func everyKindOfScreen(t *testing.T) []Screen {
	t.Helper()
	// Three rows damaged rather than the two the shared run says, because the
	// entries below land on three rows and a manifest counts them.
	envelope := run()
	envelope.RowsMutated = 3
	promised, problems := manifest.New("0.1", envelope, []manifest.Defect{
		applied("d1", 1, catalog.Rejection, catalog.Hard),
		applied("d2", 2, catalog.Rejection, catalog.Soft),
		applied("d3", 3, catalog.Warning, catalog.Soft),
		unapplied("d4", manifest.Unimplemented, "this build does not carry that damage yet"),
	})
	if len(problems) > 0 {
		t.Fatalf("the manifest this test is built on is not valid: %v", problems)
	}
	actual := response.Record{
		SpecVersion: "0.1",
		Rows: []response.Row{
			{Number: 1, Verdict: response.Accepted},
			{Number: 3, Verdict: response.Accepted},
		},
	}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	return []Screen{
		Build(i18n.English, promised, nil),
		Build(i18n.English, promised, &comparison),
	}
}

// everything a person could read off the screen, so that a test asks about what
// is shown rather than about which member happens to hold it.
func said(screen Screen) []string {
	var all []string
	for _, field := range append(append([]Field{}, screen.Run...), screen.Counts...) {
		all = append(all, field.Label, field.Value)
	}
	for _, line := range lines(screen) {
		all = append(all, line.Text)
	}
	for _, row := range screen.Rows {
		all = append(all, row.Mutator, row.Place)
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
// behind a pointer, and Fyne 2.8 has no tooltip to hide it in anyway. The kind
// stands in the table, where it differs between entries, and the card under
// the table repeats it in front of the sentence, because the card is read on
// its own and a reason with nothing saying what it is a reason for is half a
// reason.
func TestAnUnappliedEntrySaysWhyOnTheScreen(t *testing.T) {
	why := "the column follows no scheme this damage suits"
	screen := screenOf(t, nil, unapplied("d1", manifest.SchemeNotCovered, why))

	row := screen.Rows[0]
	if row.Reason.Text == "" {
		t.Fatal("an entry that did not happen carries no reason on the screen")
	}
	if !strings.Contains(row.Reason.Text, why) {
		t.Errorf("the reason on screen is %q and does not carry what the run recorded", row.Reason.Text)
	}
	kind := i18n.English.T(i18n.ManifestFailureSchemeNotCovered)
	if row.Outcome().Text != kind {
		t.Errorf("the outcome column reads %q and does not name the kind of failure", row.Outcome().Text)
	}
	if !strings.Contains(row.Reason.Text, kind) {
		t.Errorf("the card reads %q, which does not say what kind of failure the sentence is about",
			row.Reason.Text)
	}
	// It has to stand out rather than fade. A row that did not happen is the
	// one a reader most needs to find without reading.
	if row.Reason.Color != palette.StateWarning || row.Outcome().Color != palette.StateWarning {
		t.Errorf("an unapplied entry is painted %q and %q rather than in the token meant to stand out",
			row.Outcome().Color, row.Reason.Color)
	}
}

// A soft expectation looks different from a hard one, and not merely paler.
//
// Rule five of the handbook says the manifest does not pretend to a certainty
// it does not have, and this is that rule at the pixel. Paler reads as less
// important. Soft does not mean less important, it means the right answer
// belongs to the application being tested. So the soft one carries a mark in
// front of the same words in the table, and the card says in a sentence what
// the mark means.
func TestASoftExpectationDoesNotLookLikeAHardOne(t *testing.T) {
	screen := screenOf(t, nil,
		applied("d1", 1, catalog.Rejection, catalog.Hard),
		applied("d2", 2, catalog.Rejection, catalog.Soft),
	)
	hard, soft := screen.Rows[0].Expect, screen.Rows[1].Expect
	if hard.Text == soft.Text {
		t.Fatalf("both expectations read %q, so nothing in the table tells them apart", hard.Text)
	}
	mark := i18n.English.T(i18n.ManifestConfidenceMark)
	if !strings.HasPrefix(soft.Text, mark) {
		t.Errorf("the soft expectation reads %q and does not begin with the mark %q", soft.Text, mark)
	}
	if strings.HasPrefix(hard.Text, mark) {
		t.Errorf("the hard expectation reads %q and carries the mark of a soft one", hard.Text)
	}
	if screen.Rows[0].Certainty.Text == screen.Rows[1].Certainty.Text {
		t.Error("the card says the same thing about a hard certainty and a soft one")
	}
	// The mark in the table and the mark in the phrase for the certainty are
	// the same mark, or the one line explaining it explains something else.
	if !strings.HasPrefix(i18n.English.T(i18n.ManifestConfidenceSoft), mark) {
		t.Errorf("the soft certainty reads %q and does not begin with the mark %q the table uses",
			i18n.English.T(i18n.ManifestConfidenceSoft), mark)
	}
	if !strings.HasPrefix(i18n.English.T(i18n.ManifestLegendMark), mark) {
		t.Errorf("the legend reads %q and does not begin with the mark %q it explains",
			i18n.English.T(i18n.ManifestLegendMark), mark)
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
			t.Errorf("the confidence %q has no sentence", confidence)
		}
	}
	for confidence := range confidenceKeys {
		if !confidences[confidence] {
			t.Errorf("a sentence names the confidence %q, which the domain does not have", confidence)
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
// carrying how the application answered, with a disagreement told apart from
// agreement. A screen showing only the disagreements would leave a reader
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
		if row.Answer.Text == "" || row.Finding.Text == "" {
			t.Errorf("row %d carries nothing about what the application did", index+1)
		}
		if row.Outcome().Text != row.Answer.Text {
			t.Errorf("row %d shows %q in the outcome column rather than the application's answer %q",
				index+1, row.Outcome().Text, row.Answer.Text)
		}
	}
	hole := screen.Rows[0]
	if !strings.Contains(hole.Finding.Text, i18n.English.T(i18n.VerifyKindHole)) {
		t.Errorf("the row the application wrongly accepted reads %q and does not name the disagreement",
			hole.Finding.Text)
	}
	if hole.Answer.Color != palette.StateDanger || hole.Finding.Color != palette.StateDanger {
		t.Errorf("a disagreement is painted %q and %q rather than in the danger token",
			hole.Answer.Color, hole.Finding.Color)
	}
	agreed := screen.Rows[1]
	if strings.Contains(agreed.Finding.Text, i18n.English.T(i18n.VerifyKindHole)) {
		t.Errorf("the row the application answered as expected reads %q and names a disagreement",
			agreed.Finding.Text)
	}
	if agreed.Answer.Color == palette.StateDanger {
		t.Error("a row that agreed is painted in the danger token, so agreement and disagreement look alike")
	}
	if !says(screen, i18n.English.T(i18n.VerifyVerdictRejected)) {
		t.Error("the screen does not say that the application rejected the row it was expected to reject")
	}
	if screen.Compared != 2 || screen.Disagreed != 1 {
		t.Errorf("the tally says %d compared and %d disagreed for two judged rows with one hole",
			screen.Compared, screen.Disagreed)
	}
}

// A row the response record never mentions is said to be unanswered rather than
// left looking like agreement.
func TestARowTheApplicationNeverAnsweredForSaysSo(t *testing.T) {
	promised := built(t, applied("d1", 1, catalog.Rejection, catalog.Hard))
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 2, Verdict: response.Rejected}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	screen := Build(i18n.English, promised, &comparison)

	if screen.Rows[0].Answer.Text != i18n.English.T(i18n.ManifestAnswerNone) {
		t.Errorf("a row nothing was said about reads %q in the outcome column", screen.Rows[0].Answer.Text)
	}
	if screen.Rows[0].Finding.Text != i18n.English.T(i18n.VerifyNoAnswer) {
		t.Errorf("a row nothing was said about reads %q in the card", screen.Rows[0].Finding.Text)
	}
}

// The second state carries what it could not settle and what it could not see,
// and an entry it could not settle says so in its own row.
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
	if screen.Rows[0].Answer.Text != i18n.English.T(i18n.ManifestAnswerNotSettled) {
		t.Errorf("the entry the comparison could not settle reads %q in the outcome column", screen.Rows[0].Answer.Text)
	}
	if screen.Rows[0].Answer.Color != palette.StateWarning {
		t.Errorf("the entry the comparison could not settle is painted %q rather than in the warning token",
			screen.Rows[0].Answer.Color)
	}
	if len(screen.Limits) == 0 {
		t.Error("the screen states no limit against a record that carries neither stored values nor warnings")
	}
	if len(screen.Notes) == 0 {
		t.Error("the second state says nothing about the run as a whole")
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

	var whole []string
	for _, finding := range comparison.Report.Findings {
		if finding.Row == 0 {
			whole = append(whole, finding.Detail)
		}
	}
	if len(whole) == 0 {
		t.Fatal("the comparison found nothing about the whole dataset, so this test checks nothing")
	}
	for _, detail := range whole {
		found := false
		for _, note := range screen.Notes {
			if note.Text == sentence(detail) && note.Color == palette.StateDanger {
				found = true
			}
		}
		if !found {
			t.Errorf("the screen does not say, as a note in the danger token, that %q", detail)
		}
		for _, line := range []Line{screen.Rows[0].Answer, screen.Rows[0].Finding} {
			if strings.Contains(line.Text, detail) {
				t.Error("a statement about the whole dataset is shown against one row of it")
			}
		}
	}
	if screen.Disagreed != 0 {
		t.Errorf("a finding about the whole dataset is counted as %d entries that disagreed", screen.Disagreed)
	}
}

// A sentence of the comparison that stands on a line of its own reads as one:
// a capital at the front and a full stop at the end.
//
// 🔴 Seen on the real window on 2026-09-23 (B-289): the note over the table
// read "the dataset had 24 records...", because the comparison writes its
// phrases to follow the words the command line puts in front of them. The
// limits are the same kind of phrase on the same kind of line.
func TestASentenceOfTheComparisonOnALineOfItsOwnReadsAsOne(t *testing.T) {
	promised := built(t, applied("d1", 1, catalog.Rejection, catalog.Hard))
	actual := response.Record{SpecVersion: "0.1", Rows: []response.Row{{Number: 1, Verdict: response.Rejected}}}
	comparison := Comparison{Record: actual, Report: verify.Compare(promised, actual)}
	screen := Build(i18n.English, promised, &comparison)

	standing := append(append([]Line{}, screen.Notes...), screen.Limits...)
	if len(screen.Notes) == 0 || len(screen.Limits) == 0 {
		t.Fatalf("the comparison left %d notes and %d limits, so this test checks nothing",
			len(screen.Notes), len(screen.Limits))
	}
	for _, line := range standing {
		first, _ := utf8.DecodeRuneInString(line.Text)
		last, _ := utf8.DecodeLastRuneInString(line.Text)
		if !unicode.IsUpper(first) || last != '.' {
			t.Errorf("a line standing on its own reads %q, which does not begin and end as a sentence", line.Text)
		}
	}
}

// The reason an entry could not be applied is joined to its kind by a word,
// and keeps the first letter it was written with.
//
// 🔴 The window read "Not applied: not built yet. this build carries..." (B-289).
// The reason is not raised to a capital to mend that, because several reasons
// begin with the name of a knob or with a value, and the capital would change
// what somebody types into a recipe. The reason below is one of those.
func TestTheReasonAnEntryWasNotAppliedIsJoinedRatherThanCapitalised(t *testing.T) {
	why := "drift_years is nought, so the identifier would keep agreeing with the column beside it"
	screen := screenOf(t, nil, unapplied("d1", manifest.DeclinedByMutator, why))

	reason := screen.Rows[0].Reason.Text
	want := i18n.English.T(i18n.ManifestFailureDeclinedByMutator) + ", " +
		i18n.English.T(i18n.ManifestDetailBecause) + " " + why
	if !strings.HasSuffix(reason, want) {
		t.Errorf("the reason reads %q where it should end %q", reason, want)
	}
	if strings.Contains(reason, ". "+why) || strings.Contains(reason, "Drift_years") {
		t.Errorf("the reason reads %q, which begins a sentence with a phrase written to follow words", reason)
	}
}

// The first state says nothing about an application, because none has seen the
// data yet.
func TestTheFirstStateClaimsNothingAboutAnyApplication(t *testing.T) {
	screen := screenOf(t, nil, applied("d1", 1, catalog.Rejection, catalog.Hard))
	if screen.Rows[0].Answer.Text != "" || screen.Rows[0].Finding.Text != "" {
		t.Errorf("a manifest shown on its own carries %q and %q about an application",
			screen.Rows[0].Answer.Text, screen.Rows[0].Finding.Text)
	}
	if len(screen.Notes) != 0 || len(screen.Unsettled) != 0 || len(screen.Limits) != 0 {
		t.Error("a manifest shown on its own carries the sections that belong to a comparison")
	}
	if screen.Compared != 0 || screen.Disagreed != 0 {
		t.Error("a manifest shown on its own carries a tally of a comparison nobody made")
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
	for _, screen := range everyKindOfScreen(t) {
		for _, phrase := range said(screen) {
			if looksLikeAKey.MatchString(phrase) {
				t.Errorf("the phrase %q reads as a translation key rather than as a sentence", phrase)
			}
		}
	}
}
