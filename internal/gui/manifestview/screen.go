// Package manifestview builds the screen that shows what a run promised about
// a dataset, and what an application did about it.
//
// # Why the screen is a value before it is a widget
//
// The list of entries has to be a pooled list rather than a column of widgets:
// a neighbouring project measured one hundred thousand rows in a plain column
// at 1.353 s and 448.9 MB, and a manifest of that size is ordinary here. A
// pooled list builds its rows only when they are scrolled into view, so a test
// walking the widget tree sees an empty frame and would pass while saying
// nothing at all.
//
// So this file answers what the screen says, as plain values, and the files
// beside it answer how that is drawn. The first is checked entry by entry
// against a manifest, and the second is checked by driving one row of the list
// directly. Neither half alone would be worth much: a model nothing renders is
// prose, and a renderer with no model is a screen nobody can hold to anything.
//
// # What this file deliberately does not do
//
// It reads no file. A manifest and a response record arrive already parsed,
// which is what lets the whole screen be built against a manifest a test wrote
// by hand rather than against one that had to exist on a disk somewhere.
//
// # The shape, since 2026-09-15 (DEC-247)
//
// An entry is four cells: the damage, where it landed, what was expected of
// the record with a mark in front of a soft expectation, and what became of it.
// The sentences an entry carries beyond those four - why it could not be
// applied, the value before and after, what the application did and how that
// compares - are read out one entry at a time in a card under the table. Six
// columns added up to 1740 units and the window hands the table 828, so the
// two columns saying what became of an entry were never on the screen without
// scrolling sideways.
package manifestview

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Line is one phrase together with the token it is painted in.
//
// The colour is a name rather than a value, because the guard beside this
// package refuses a colour written anywhere in the window outside the theme,
// and because a name is what the bridge to docs/17 compares. An empty name
// means the ordinary text colour, which is what most of the screen is.
type Line struct {
	Text  string
	Color string
}

// Field is a label and the value beside it.
//
// The label is interface text and comes from the catalogue. The value is data:
// a seed, a locale, a count. Translating a seed would change what the screen is
// showing rather than what it says.
type Field struct {
	Label string
	Value string
}

// Row is one ordered damage as a person reads it.
//
// # Why an unapplied entry carries its reason here rather than in a tooltip
//
// docs/17 puts it plainly and the reason is the ban on silence: an entry that
// did not happen is the one a reader most needs to see, so its kind is in the
// table in a colour meant to stand out, and its reason is one arrow key away
// in the card under it rather than hidden behind a pointer. Fyne 2.8 has no
// tooltips in any case, which is a fact about the toolkit rather than a design.
type Row struct {
	// Mutator is the identifier and its layer, which are data.
	Mutator string
	// Place is where the damage sits, in words.
	Place string
	// Applied says whether the damage happened at all, and Status says so in
	// words: "applied", or the closed kind of failure when it did not.
	Applied bool
	Status  Line
	// Expect is what the application should do about it, with the mark of a
	// soft expectation in front when the answer belongs to the application,
	// and Certainty is the same thing as a sentence for the card.
	//
	// The mark rather than a second column, because a soft expectation is not
	// a weaker version of a hard one and the column that said so cost 120
	// units of a table that had 828. Rule five of the handbook refuses to let
	// the window suggest otherwise, and the mark in front of the phrase is a
	// shape, which is what docs/17 asks of anything carrying meaning.
	Expect    Line
	Certainty Line
	// Reason is why the damage could not be applied, empty when it was.
	Reason Line
	// Values is the original and what replaced it, empty when the mutator
	// keeps neither.
	Values Line
	// Answer is what the application did about this row in a word or two,
	// for the outcome column, and Finding is the same thing as a sentence for
	// the card: the verdict, and where the two disagree, how. Both are filled
	// only in the second state, and only for an applied entry, since a damage
	// that never happened is nothing an application could answer for.
	Answer  Line
	Finding Line
}

// Cells is one entry as the columns of a table, in column order.
//
// # Why the row hands out cells rather than the view reaching into it
//
// A table asks for one cell at a time, by row and column, and it asks only for
// the cells somebody can see. Reaching into the row from the view would put the
// order of the columns in two places, so a column inserted later would show the
// heading of one thing above the values of another and go on compiling.
func (r Row) Cells() []Line {
	return []Line{
		{Text: r.Mutator},
		{Text: r.Place, Color: palette.TextSecondary},
		r.Expect,
		r.Outcome(),
	}
}

// Cell is one column of this entry, and an empty line for a column that does
// not exist. A table that asked for a column nobody defined would otherwise
// take the view down with it.
func (r Row) Cell(column int) Line {
	cells := r.Cells()
	if column < 0 || column >= len(cells) {
		return Line{}
	}
	return cells[column]
}

// Outcome is what became of the ordered damage, in the one column that says so.
//
// # Why one column carries both halves
//
// A damage that was not applied has no answer from any application, and one
// that was applied said only "applied" in the first drawing, in a column
// beside a second one carrying the answer. Two columns held one fact each and
// never both, so the column that said the least in every row was the one the
// table could not afford. Here the kind of failure stands for an entry that
// did not happen, the application's verdict for one that did once a response
// is open, and "applied" until then.
func (r Row) Outcome() Line {
	if !r.Applied {
		return r.Status
	}
	if r.Answer.Text != "" {
		return r.Answer
	}
	return r.Status
}

// Detail is everything the card under the table reads out about this entry
// beyond its four cells, in reading order, with nothing empty in it.
func (r Row) Detail() []Line {
	var lines []Line
	for _, line := range []Line{r.Certainty, r.Values, r.Reason, r.Finding} {
		if line.Text != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// Comparison is the second state: a response record and the report of putting
// it beside the manifest.
type Comparison struct {
	Record response.Record
	Report verify.Report
}

// Screen is everything the manifest browser shows.
type Screen struct {
	// Run is the envelope, and Counts is how much of what was ordered happened.
	Run    []Field
	Counts []Field
	// Notes are the sentences of the second state that belong to the run as a
	// whole rather than to one row.
	Notes []Line
	// Rows is one entry of the manifest each, in the order damage was
	// attempted, which is deterministic for one seed.
	Rows []Row
	// Compared is how many applied entries the second state could judge,
	// Disagreed how many of those the application answered against the
	// manifest, and Unsettled every entry the comparison could not judge.
	// Limits is what it could not see at all. All are empty in the first
	// state.
	//
	// 🔴 None is optional in the second. A comparison reporting only its
	// findings reads as a complete account of the run, and the whole argument
	// of this project is that a quiet gap is worse than a loud one.
	Compared  int
	Disagreed int
	Unsettled []Line
	Limits    []Line
}

// Build turns a manifest, and optionally a comparison, into what the screen
// shows.
//
// A nil comparison is the first state: the manifest on its own. That is not a
// lesser version of the second one. It is what somebody reads straight after a
// run, before any application has seen the data.
func Build(text i18n.Catalogue, promised manifest.Manifest, comparison *Comparison) Screen {
	screen := Screen{
		Run:    runFields(text, promised),
		Counts: countFields(text, promised),
		Rows:   rowsOf(text, promised, comparison),
	}
	if comparison != nil {
		screen.Notes = notesOf(text, comparison.Report)
		screen.Compared = comparison.Report.Compared
		screen.Disagreed = disagreedOf(comparison.Report)
		screen.Unsettled = unsettledOf(comparison.Report)
		screen.Limits = limitLines(comparison.Report)
	}
	return screen
}

// runFields is the envelope, with the optional references left out when the run
// carried none.
//
// Left out rather than shown empty: a row reading "Tag" against nothing is a
// reader wondering whether the tag was lost or never set, and the manifest does
// not distinguish those either.
//
// The order is the order of the columns the envelope is laid out in, and the
// format version stands before the catalog version for a reason the render
// showed: the columns share the width equally and a value has what its column's
// widest label leaves, so the timestamp, the longest value every manifest
// carries, is kept in a column whose labels are short. "Corruption catalog" is
// the widest label of the envelope and stood in that column until 2026-09-15,
// where it cut the timestamp to an ellipsis.
func runFields(text i18n.Catalogue, promised manifest.Manifest) []Field {
	run := promised.Run()
	fields := []Field{
		{text.T(i18n.ManifestRunGenerated), run.GeneratedAt},
		{text.T(i18n.ManifestRunSeed), run.Seed},
		{text.T(i18n.ManifestRunLocale), run.Locale},
		{text.T(i18n.ManifestRunSpec), promised.SpecVersion()},
		{text.T(i18n.ManifestRunCatalog), run.CatalogVersion},
	}
	for _, optional := range []Field{
		{text.T(i18n.ManifestRunSource), run.Source},
		{text.T(i18n.ManifestRunRecipe), run.Recipe},
		{text.T(i18n.ManifestRunPolicy), run.Policy},
		{text.T(i18n.ManifestRunTag), run.Tag},
	} {
		if optional.Value != "" {
			fields = append(fields, optional)
		}
	}
	return append(fields,
		Field{text.T(i18n.ManifestRunRowsTotal), strconv.Itoa(run.RowsTotal)},
		Field{text.T(i18n.ManifestRunRowsMutated), strconv.Itoa(run.RowsMutated)})
}

// countFields is how much of what was ordered happened.
//
// The three are derived from the entries rather than carried beside them, which
// is the same reason the model derives them: a summary that can disagree with
// the list under it is the one disagreement nobody would check.
func countFields(text i18n.Catalogue, promised manifest.Manifest) []Field {
	return []Field{
		{text.T(i18n.ManifestCountRequested), strconv.Itoa(promised.Requested())},
		{text.T(i18n.ManifestCountApplied), strconv.Itoa(promised.Applied())},
		{text.T(i18n.ManifestCountNotApplied), strconv.Itoa(len(promised.Unapplied()))},
	}
}

// Waiting is the envelope before any manifest is open: the field names, with no
// value beside any of them.
//
// 🔴 # Why the values are absent rather than a placeholder
//
// The first attempt put a hyphen where each value goes, so that the waiting
// screen kept the two column shape of the state that has a manifest. Rendered
// on 2026-09-08 it read wrong: a form column is only as wide as what is in it,
// so a column of single characters closed up and the eye paired the name on the
// right with the hyphen on its left. "Generated - Rows" is a sentence about
// nothing. A name on its own cannot be misread that way, and the sentence above
// the list already says the values arrive with a file.
//
// # Why this is not Build of an empty manifest
//
// An empty manifest is a manifest, so Build would answer with a run whose seed
// is the empty string and whose counts are all zero. Those are values, and they
// would be values about a file nobody has chosen. This says the same thing the
// settings on the screen next door say while they are dead: here is what will
// stand here.
//
// # Why only the fields every manifest carries
//
// runFields drops an optional field when the run carried none, so a source
// file, a recipe, a policy and a tag are on the screen for some manifests and
// absent for others. Naming them here would promise rows that the file somebody
// is about to open may not have. What is listed is what the format requires,
// which makes this the shape of every manifest rather than the shape of one.
func Waiting(text i18n.Catalogue) Screen {
	waiting := func(names ...i18n.Key) []Field {
		fields := make([]Field, 0, len(names))
		for _, name := range names {
			fields = append(fields, Field{Label: text.T(name)})
		}
		return fields
	}
	return Screen{
		Run: waiting(i18n.ManifestRunGenerated, i18n.ManifestRunSeed, i18n.ManifestRunLocale,
			i18n.ManifestRunSpec, i18n.ManifestRunCatalog,
			i18n.ManifestRunRowsTotal, i18n.ManifestRunRowsMutated),
		Counts: waiting(i18n.ManifestCountRequested, i18n.ManifestCountApplied,
			i18n.ManifestCountNotApplied),
	}
}

func rowsOf(text i18n.Catalogue, promised manifest.Manifest, comparison *Comparison) []Row {
	defects := promised.Defects()
	rows := make([]Row, 0, len(defects))
	for _, defect := range defects {
		rows = append(rows, rowOf(text, defect, comparison))
	}
	return rows
}

func rowOf(text i18n.Catalogue, defect manifest.Defect, comparison *Comparison) Row {
	// The layer is not written beside the identifier, because the identifier
	// begins with it: every one is LAYER.area.variant, so a row saying
	// "L0.bom.unexpected L0" is repeating itself in the one place a person
	// scans first.
	row := Row{
		Mutator:   string(defect.Mutator),
		Place:     placeOf(text, defect.Location),
		Applied:   defect.Applied,
		Status:    statusOf(text, defect),
		Expect:    expectationOf(text, defect.Expectation, defect.Confidence),
		Certainty: certaintyOf(text, defect.Expectation, defect.Confidence),
		Values:    valuesOf(text, defect),
	}
	if !defect.Applied {
		row.Reason = failureOf(text, defect)
	}
	if comparison != nil && defect.Applied {
		row.Answer, row.Finding = answerOf(text, *comparison, defect)
	}
	return row
}

// separator is what stands between two facts on one line.
//
// It is here rather than typed at each place that needs it, so that changing
// how a line reads is one edit rather than a search, and so that no screen
// grows its own spacing habit.
const separator = "   "

// statusOf says what became of ordered damage, before any application has.
//
// # Why the kind of failure stands here rather than beside the reason
//
// Looking at a real manifest showed it: fifty three of sixty entries were not
// applied, so a column reading NOT APPLIED fifty three times and a column
// repeating the same sentence beside it gave a reader nothing to compare. The
// kind is a closed vocabulary and short, so it belongs in the table, where it
// differs between entries. The sentence the run recorded names the column or
// the capability, so it belongs in the card under the table, one entry at a
// time.
//
// The colour still marks the row out at a glance, which is what rule 4.4 asks:
// an entry that did not happen has to stand out rather than fade. It is the
// warning token rather than the danger one, because a damage this build could
// not do is a limit of the run and not a fault of the application under test.
func statusOf(text i18n.Catalogue, defect manifest.Defect) Line {
	if defect.Applied {
		return Line{Text: text.T(i18n.ManifestEntryApplied), Color: palette.TextSecondary}
	}
	phrase := text.T(i18n.ManifestEntryNotApplied)
	if key, known := failureKeys[defect.FailureKind]; known {
		phrase = text.T(key)
	}
	return Line{Text: phrase, Color: palette.StateWarning}
}

// placeOf says where the damage sits, in the terms the layer uses.
//
// A location that names nothing is legal and means the whole file, which is
// what damage to the bytes of a document is. Saying so is the point: an empty
// cell there would read as a place somebody forgot to record.
func placeOf(text i18n.Catalogue, location manifest.Location) string {
	if location.Empty() {
		return text.T(i18n.ManifestPlaceWholeFile)
	}
	place := ""
	for _, part := range []string{location.Entity, rowPart(text, location), location.Field, location.Path} {
		if part == "" {
			continue
		}
		if place != "" {
			place += separator
		}
		place += part
	}
	if location.HasByteOffset {
		place += separator + text.T(i18n.ManifestPlaceByte) + " " + strconv.Itoa(location.ByteOffset)
	}
	return place
}

func rowPart(text i18n.Catalogue, location manifest.Location) string {
	if location.Row == 0 {
		return ""
	}
	return text.T(i18n.ManifestPlaceRow) + " " + strconv.Itoa(location.Row)
}

// expectationKeys and the registers below turn a closed vocabulary of the
// domain into phrases a person reads.
//
// 🔴 Each one is closed against the domain in both directions by a test, so a
// value the domain gains and this file has not heard of fails the run instead
// of arriving on screen as an empty space. That is the same bargain every other
// register in this project makes, and it is the one that matters most here:
// this is the last step before a person, so a gap here is a gap they see.
var expectationKeys = map[catalog.Expectation]i18n.Key{
	catalog.Rejection:  i18n.ManifestExpectationRejection,
	catalog.Acceptance: i18n.ManifestExpectationAcceptance,
	catalog.Warning:    i18n.ManifestExpectationWarning,
}

var confidenceKeys = map[catalog.Confidence]i18n.Key{
	catalog.Hard: i18n.ManifestDetailHard,
	catalog.Soft: i18n.ManifestDetailSoft,
}

var failureKeys = map[manifest.FailureKind]i18n.Key{
	manifest.Undescribed:         i18n.ManifestFailureUndescribed,
	manifest.Unimplemented:       i18n.ManifestFailureUnimplemented,
	manifest.RunLacksCapability:  i18n.ManifestFailureRunLacksCapability,
	manifest.FieldTypeNotCovered: i18n.ManifestFailureFieldTypeNotCovered,
	manifest.SchemeNotCovered:    i18n.ManifestFailureSchemeNotCovered,
	manifest.DeclinedByMutator:   i18n.ManifestFailureDeclinedByMutator,
}

var kindKeys = map[verify.Kind]i18n.Key{
	verify.Hole:        i18n.VerifyKindHole,
	verify.Overzealous: i18n.VerifyKindOverzealous,
	verify.Silent:      i18n.VerifyKindSilent,
	verify.Missing:     i18n.VerifyKindMissing,
}

// expectationOf is what the application should do, with the mark of a soft
// expectation in front of it.
//
// In the ordinary text colour since DEC-247. Three colours over sixty rows were
// noise, and WCAG 1.4.1 asks that the word carry the meaning in any case: the
// three colours differed from each other by a contrast of 1.18 to 1.41, so
// somebody who does not separate hues saw one shade of grey. Colour on this
// screen is kept for what needs attention, which is what did not happen and
// where an application disagreed.
func expectationOf(text i18n.Catalogue, expectation catalog.Expectation, confidence catalog.Confidence) Line {
	phrase := string(expectation)
	if key, known := expectationKeys[expectation]; known {
		phrase = text.T(key)
	}
	if confidence == catalog.Soft {
		phrase = text.T(i18n.ManifestConfidenceMark) + " " + phrase
	}
	return Line{Text: phrase}
}

// certaintyOf says how much the expectation is worth, as a sentence for the
// card: the expectation again, and then what the certainty means.
//
// A sentence rather than one of two words, because the card is the one place
// with room for one, and because "soft" on its own has told nobody what soft
// means. It means the answer belongs to the application being tested and this
// tool is not entitled to assert it.
func certaintyOf(text i18n.Catalogue, expectation catalog.Expectation, confidence catalog.Confidence) Line {
	key, known := confidenceKeys[confidence]
	if !known {
		return Line{Text: expectationOf(text, expectation, confidence).Text + ", " + string(confidence),
			Color: palette.TextSecondary}
	}
	return Line{Text: expectationOf(text, expectation, confidence).Text + ", " + text.T(key),
		Color: palette.TextSecondary}
}

// failureOf is what the run recorded about damage it could not do, as the card
// reads it out: that it was not applied, the closed kind, and the sentence the
// run wrote for the case at hand.
//
// The kind IS repeated here, though it already stands in the table, because the
// card is read on its own: an arrow key later it is about another entry, and a
// line reading only "the column follows no scheme" has lost what it is a
// reason for. The sentence is not a key and cannot become one, since it is
// composed per entry, and that limit is recorded rather than hidden.
//
// An unrecognised kind falls back to standing here by its own name, which keeps
// it visible rather than silent.
//
// 🔴 # Why the sentence is joined by a word rather than begun after a full stop
//
// Every reason this build records is written to follow other words - the
// command line puts it after a colon - so it begins in lower case, and the card
// read "Not applied: not built yet. this build carries..." on the real window
// (B-289). Raising its first letter would fix that reason and break others:
// several begin with a value or the name of a knob ("drift_years is nought"
// from national_id_vs_birthdate), and a capital there changes what somebody
// would type into a recipe. A joining word lets every reason be read as it was
// written.
func failureOf(text i18n.Catalogue, defect manifest.Defect) Line {
	kind := string(defect.FailureKind)
	if key, known := failureKeys[defect.FailureKind]; known {
		kind = text.T(key)
	}
	phrase := text.T(i18n.ManifestDetailNotApplied) + ": " + kind
	if defect.FailureReason != "" {
		phrase += ", " + text.T(i18n.ManifestDetailBecause) + " " + defect.FailureReason
	}
	return Line{Text: phrase, Color: palette.StateWarning}
}

// valuesOf is what was there and what took its place.
//
// Each half is shown only when the manifest carries it, because an empty string
// is a value one entry of the catalog produces on purpose and the model keeps a
// flag to tell that from nothing recorded.
func valuesOf(text i18n.Catalogue, defect manifest.Defect) Line {
	phrase := ""
	if defect.HasValueBefore {
		phrase = text.T(i18n.ManifestEntryBefore) + " " + quoted(defect.ValueBefore)
	}
	if defect.HasValueAfter {
		if phrase != "" {
			phrase += separator
		}
		phrase += text.T(i18n.ManifestEntryAfter) + " " + quoted(defect.ValueAfter)
	}
	if phrase == "" {
		return Line{}
	}
	return Line{Text: phrase, Color: palette.TextAccent}
}

// quoted puts a value in marks so that an empty one and a missing one do not
// look the same on screen, which is the whole reason the model keeps them
// apart.
func quoted(value string) string { return "\"" + value + "\"" }

// answerOf is what the application did about this row: a word or two for the
// table, and the sentence for the card.
//
// The verdict is shown even where the two agreed. A screen that spoke only
// about disagreements would leave a reader unable to tell a row that agreed
// from one nothing was said about, and those are the two answers this project
// most insists on separating. A disagreement is painted in the danger token
// and an entry the comparison could not settle in the warning one, because
// the first is a claim about the application and the second is a question
// the response record did not carry the evidence to answer.
func answerOf(text i18n.Catalogue, comparison Comparison, defect manifest.Defect) (short, long Line) {
	answer, reacted := comparison.Record.ByRow(defect.Location.Row)
	if !reacted {
		return Line{Text: text.T(i18n.ManifestAnswerNone), Color: palette.StateDanger},
			Line{Text: text.T(i18n.VerifyNoAnswer), Color: palette.StateDanger}
	}
	verdict, word := verdictOf(text, answer.Verdict)
	if finding, found := findingFor(comparison.Report, defect); found {
		return Line{Text: word, Color: palette.StateDanger},
			Line{Text: verdict + separator + kindOf(text, finding.Kind) + separator + finding.Detail,
				Color: palette.StateDanger}
	}
	if open, found := unsettledFor(comparison.Report, defect); found {
		return Line{Text: text.T(i18n.ManifestAnswerNotSettled), Color: palette.StateWarning},
			Line{Text: verdict + separator + open.Reason, Color: palette.StateWarning}
	}
	return Line{Text: word, Color: palette.TextSecondary}, Line{Text: verdict, Color: palette.TextSecondary}
}

// verdictOf is the verdict of the application as a sentence and as a word.
func verdictOf(text i18n.Catalogue, verdict response.Verdict) (sentence, word string) {
	switch verdict {
	case response.Accepted:
		return text.T(i18n.VerifyVerdictAccepted), text.T(i18n.ManifestAnswerAccepted)
	case response.Rejected:
		return text.T(i18n.VerifyVerdictRejected), text.T(i18n.ManifestAnswerRejected)
	}
	return string(verdict), string(verdict)
}

func kindOf(text i18n.Catalogue, kind verify.Kind) string {
	if key, known := kindKeys[kind]; known {
		return text.T(key)
	}
	return string(kind)
}

// findingFor is the disagreement about one entry, if the comparison found one.
//
// Matched on the row, the column and the mutator together. Any two of those can
// repeat inside one manifest, since one row carries several damages and one
// mutator is used in many rows, so a match on fewer would attach somebody
// else's finding to an entry that agreed.
func findingFor(report verify.Report, defect manifest.Defect) (verify.Finding, bool) {
	for _, finding := range report.Findings {
		if finding.Row == defect.Location.Row &&
			finding.Field == defect.Location.Field &&
			finding.Mutator == defect.Mutator {
			return finding, true
		}
	}
	return verify.Finding{}, false
}

// unsettledFor is the question about one entry the comparison could not
// answer, if there is one, matched the way a finding is and for the same
// reason.
func unsettledFor(report verify.Report, defect manifest.Defect) (verify.Unsettled, bool) {
	for _, open := range report.Unsettled {
		if open.Row == defect.Location.Row &&
			open.Field == defect.Location.Field &&
			open.Mutator == defect.Mutator {
			return open, true
		}
	}
	return verify.Unsettled{}, false
}

// notesOf is what the comparison says about the run as a whole.
//
// A finding with no row belongs here rather than beside an entry, and there is
// one that behaves that way: records the application never reacted to at all
// are counted from the dataset rather than from the damaged rows. Attaching it
// to an entry would put a statement about the whole file on one arbitrary line
// of it. How many entries were compared is not a note any more: it stands at
// the heading of the table with the other counts.
//
// The detail stands alone, without the phrase for the kind in front of it. The
// phrases are written for one row ("the application never answered for this
// row"), and rendering showed one at the head of a sentence about the whole
// dataset. The detail of a finding about the whole file is a sentence already.
func notesOf(text i18n.Catalogue, report verify.Report) []Line {
	var notes []Line
	for _, finding := range report.Findings {
		if finding.Row == 0 {
			words := finding.Detail
			if words == "" {
				words = kindOf(text, finding.Kind)
			}
			notes = append(notes, Line{Text: sentence(words), Color: palette.StateDanger})
		}
	}
	if len(report.Findings) == 0 {
		notes = append(notes, Line{Text: text.T(i18n.VerifyAgreed), Color: palette.StateSuccess})
	}
	return notes
}

// disagreedOf counts the findings that belong to one entry each, which is what
// the tally at the heading of the table reports beside how many were compared.
// A finding about the whole file is a note and is not an entry that disagreed.
func disagreedOf(report verify.Report) int {
	disagreed := 0
	for _, finding := range report.Findings {
		if finding.Row != 0 {
			disagreed++
		}
	}
	return disagreed
}

func unsettledOf(report verify.Report) []Line {
	lines := make([]Line, 0, len(report.Unsettled))
	for _, open := range report.Unsettled {
		lines = append(lines, Line{
			Text:  placeOfUnsettled(open) + open.Reason,
			Color: palette.StateWarning,
		})
	}
	return lines
}

func placeOfUnsettled(open verify.Unsettled) string {
	if open.Row == 0 {
		return ""
	}
	return strconv.Itoa(open.Row) + separator + open.Field + separator
}

func limitLines(report verify.Report) []Line {
	lines := make([]Line, 0, len(report.Limits))
	for _, limit := range report.Limits {
		lines = append(lines, Line{Text: sentence(limit), Color: palette.TextSecondary})
	}
	return lines
}

// sentence is a phrase of the comparison made to stand on its own line: a
// capital at the front and a full stop at the end.
//
// # Why here and not where the phrase is written
//
// The comparison writes its phrases to follow other words, because the command
// line prints them after its own: a limit after "not settled here:", where a
// capital would be the fault. The screen is where a phrase stops following
// anything and becomes a line of its own, which read "the dataset had 24
// records..." on the real window (B-289).
//
// # Why this cannot raise a value
//
// It is used only on the notes and the limits, and every one of those begins
// with words of the comparison itself ("the dataset", "no row", "the response
// record") or with a phrase of the catalogue, never with a value or a name from
// somebody's data. The reason an entry was not applied is the counter-example,
// and that one is joined by a word instead (failureOf).
func sentence(words string) string {
	if words == "" {
		return words
	}
	first, size := utf8.DecodeRuneInString(words)
	if unicode.IsLower(first) {
		words = string(unicode.ToUpper(first)) + words[size:]
	}
	last, _ := utf8.DecodeLastRuneInString(words)
	if !strings.ContainsRune(".!?", last) {
		words += "."
	}
	return words
}
