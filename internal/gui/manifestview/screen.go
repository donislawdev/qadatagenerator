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
// So this file answers what the screen says, as plain values, and the file
// beside it answers how that is drawn. The first is checked entry by entry
// against a manifest, and the second is checked by driving one row of the list
// directly. Neither half alone would be worth much: a model nothing renders is
// prose, and a renderer with no model is a screen nobody can hold to anything.
//
// # What this file deliberately does not do
//
// It reads no file. A manifest and a response record arrive already parsed,
// which is what lets the whole screen be built against a manifest a test wrote
// by hand rather than against one that had to exist on a disk somewhere.
package manifestview

import (
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// Line is one phrase together with the palette entry it is painted in.
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
// did not happen is the one a reader most needs to see, so it is on the screen
// in a colour meant to stand out rather than hidden behind a pointer. Fyne 2.8
// has no tooltips in any case, which is a fact about the toolkit rather than a
// design.
type Row struct {
	// Mutator is the identifier and its layer, which are data.
	Mutator string
	// Place is where the damage sits, in words.
	Place string
	// Status says whether it happened at all.
	Status Line
	// Expect is what the application should do about it, and Confidence is
	// how much that expectation is worth.
	//
	// Two lines rather than one, because a soft expectation is not a weaker
	// version of a hard one. It is a statement that the right answer belongs
	// to the application, and rule five of the handbook refuses to let the
	// window suggest otherwise.
	Expect     Line
	Confidence Line
	// Reason is why the damage could not be applied, empty when it was.
	Reason Line
	// Values is the original and what replaced it, empty when the mutator
	// keeps neither.
	Values Line
	// Answer is what the application did about this row, and is filled only in
	// the second state.
	Answer Line
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
		{Text: r.Place, Color: palette.NameTextSubdued},
		r.Expect,
		r.Confidence,
		r.Status,
		r.Detail(),
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

// Detail is the one line of a row that carries a sentence rather than a value.
//
// # Why the three are one line rather than three
//
// A reason belongs to an entry that did not happen and values belong to one
// that did, so the two never occur together. The answer of an application joins
// the values because it is about the same thing: what became of damage that was
// really done. Keeping them on separate lines meant an entry with no values
// left a hole in the middle of its own row, and the line below that hole read
// as though it belonged to the next entry.
func (r Row) Detail() Line {
	if r.Reason.Text != "" {
		return r.Reason
	}
	if r.Values.Text == "" {
		return r.Answer
	}
	if r.Answer.Text == "" {
		return r.Values
	}
	return Line{Text: r.Values.Text + separator + r.Answer.Text, Color: r.Answer.Color}
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
	// Unsettled is every entry the comparison could not judge, and Limits is
	// what it could not see at all. Both are empty in the first state.
	//
	// 🔴 Neither is optional in the second. A comparison reporting only its
	// findings reads as a complete account of the run, and the whole argument
	// of this project is that a quiet gap is worse than a loud one.
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
func runFields(text i18n.Catalogue, promised manifest.Manifest) []Field {
	run := promised.Run()
	fields := []Field{
		{text.T(i18n.ManifestRunGenerated), run.GeneratedAt},
		{text.T(i18n.ManifestRunSeed), run.Seed},
		{text.T(i18n.ManifestRunLocale), run.Locale},
		{text.T(i18n.ManifestRunCatalog), run.CatalogVersion},
		{text.T(i18n.ManifestRunSpec), promised.SpecVersion()},
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
		Mutator:    string(defect.Mutator),
		Place:      placeOf(text, defect.Location),
		Status:     statusOf(text, defect),
		Expect:     expectationOf(text, defect.Expectation),
		Confidence: confidenceOf(text, defect.Confidence),
		Values:     valuesOf(text, defect),
	}
	if !defect.Applied {
		row.Reason = failureOf(text, defect)
	}
	if comparison != nil && defect.Applied {
		row.Answer = answerOf(text, *comparison, defect)
	}
	return row
}

// separator is what stands between two facts on one line.
//
// It is here rather than typed at each place that needs it, so that changing
// how a line reads is one edit rather than a search, and so that no screen
// grows its own spacing habit.
const separator = "   "

// statusOf says what became of ordered damage.
//
// # Why the kind of failure stands here rather than beside the reason
//
// Looking at a real manifest showed it: fifty three of sixty entries were not
// applied, so a column reading NOT APPLIED fifty three times and a column
// repeating the same sentence beside it gave a reader nothing to compare. The
// kind is a closed vocabulary and short, so it belongs in the narrow column,
// where it differs between entries. The sentence the run recorded names the
// column or the capability, so it belongs in the wide one.
//
// The colour still marks the row out at a glance, which is what rule 4.4 asks:
// an entry that did not happen has to stand out rather than fade.
func statusOf(text i18n.Catalogue, defect manifest.Defect) Line {
	if defect.Applied {
		return Line{Text: text.T(i18n.ManifestEntryApplied), Color: palette.NameTextSubdued}
	}
	phrase := text.T(i18n.ManifestEntryNotApplied)
	if key, known := failureKeys[defect.FailureKind]; known {
		phrase = text.T(key)
	}
	return Line{Text: phrase, Color: palette.NameNotApplied}
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

// expectationKeys and the two registers below turn a closed vocabulary of the
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

var expectationColors = map[catalog.Expectation]string{
	catalog.Rejection:  palette.NameRejection,
	catalog.Acceptance: palette.NameAcceptance,
	catalog.Warning:    palette.NameWarning,
}

var confidenceKeys = map[catalog.Confidence]i18n.Key{
	catalog.Hard: i18n.ManifestConfidenceHard,
	catalog.Soft: i18n.ManifestConfidenceSoft,
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

// expectationOf is what the application should do, in words and in colour.
//
// The word carries the meaning and the colour only reinforces it. That is WCAG
// 1.4.1, and here it rests on a measurement rather than on the guideline alone:
// the three colours differ from each other by a contrast of 1.18 to 1.41, so
// somebody who does not separate hues sees one shade of grey.
func expectationOf(text i18n.Catalogue, expectation catalog.Expectation) Line {
	key, known := expectationKeys[expectation]
	if !known {
		return Line{Text: string(expectation)}
	}
	return Line{Text: text.T(key), Color: expectationColors[expectation]}
}

// confidenceOf says how much the expectation is worth.
//
// A soft one gets a sign of its own in front of the phrase rather than a paler
// shade of the same thing. Paler reads as less important, and soft does not
// mean less important. It means the answer belongs to the application being
// tested and this tool is not entitled to assert it.
func confidenceOf(text i18n.Catalogue, confidence catalog.Confidence) Line {
	key, known := confidenceKeys[confidence]
	if !known {
		return Line{Text: string(confidence)}
	}
	return Line{Text: text.T(key), Color: palette.NameTextSubdued}
}

// failureOf is what the run recorded about damage it could not do.
//
// The closed kind is not repeated here, because it is already in the column
// beside this one. What is left is the sentence the run wrote for the case at
// hand: which column, which capability, which value. It is not a key and cannot
// become one, since it is composed per entry, and that limit is recorded rather
// than hidden.
//
// The kind falls back to standing here when this build has no phrase for it,
// which keeps an unrecognised kind visible rather than silent.
func failureOf(text i18n.Catalogue, defect manifest.Defect) Line {
	phrase := defect.FailureReason
	if _, known := failureKeys[defect.FailureKind]; !known {
		phrase = string(defect.FailureKind) + separator + phrase
	}
	if phrase == "" {
		phrase = text.T(i18n.ManifestEntryNotApplied)
	}
	return Line{Text: phrase, Color: palette.NameNotApplied}
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
	return Line{Text: phrase, Color: palette.NameCorrupted}
}

// quoted puts a value in marks so that an empty one and a missing one do not
// look the same on screen, which is the whole reason the model keeps them
// apart.
func quoted(value string) string { return "\"" + value + "\"" }

// answerOf is what the application did about this row.
//
// The verdict is shown even where the two agreed. A screen that spoke only
// about disagreements would leave a reader unable to tell a row that agreed
// from one nothing was said about, and those are the two answers this project
// most insists on separating.
func answerOf(text i18n.Catalogue, comparison Comparison, defect manifest.Defect) Line {
	answer, reacted := comparison.Record.ByRow(defect.Location.Row)
	if !reacted {
		return Line{Text: text.T(i18n.VerifyNoAnswer), Color: palette.NameRejection}
	}
	phrase := verdictOf(text, answer.Verdict)
	if finding, found := findingFor(comparison.Report, defect); found {
		return Line{Text: phrase + separator + kindOf(text, finding.Kind) + separator + finding.Detail,
			Color: palette.NameRejection}
	}
	return Line{Text: phrase, Color: palette.NameTextSubdued}
}

func verdictOf(text i18n.Catalogue, verdict response.Verdict) string {
	switch verdict {
	case response.Accepted:
		return text.T(i18n.VerifyVerdictAccepted)
	case response.Rejected:
		return text.T(i18n.VerifyVerdictRejected)
	}
	return string(verdict)
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

// notesOf is what the comparison says about the run as a whole.
//
// A finding with no row belongs here rather than beside an entry, and there is
// one that behaves that way: records the application never reacted to at all
// are counted from the dataset rather than from the damaged rows. Attaching it
// to an entry would put a statement about the whole file on one arbitrary line
// of it.
func notesOf(text i18n.Catalogue, report verify.Report) []Line {
	notes := []Line{{
		Text:  text.T(i18n.VerifyCompared) + " " + strconv.Itoa(report.Compared),
		Color: palette.NameTextSubdued,
	}}
	for _, finding := range report.Findings {
		if finding.Row == 0 {
			notes = append(notes, Line{
				Text:  kindOf(text, finding.Kind) + separator + finding.Detail,
				Color: palette.NameRejection,
			})
		}
	}
	if len(report.Findings) == 0 {
		notes = append(notes, Line{Text: text.T(i18n.VerifyAgreed), Color: palette.NameAcceptance})
	}
	return notes
}

func unsettledOf(report verify.Report) []Line {
	lines := make([]Line, 0, len(report.Unsettled))
	for _, open := range report.Unsettled {
		lines = append(lines, Line{
			Text:  placeOfUnsettled(open) + open.Reason,
			Color: palette.NameWarning,
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
		lines = append(lines, Line{Text: limit, Color: palette.NameTextSubdued})
	}
	return lines
}
