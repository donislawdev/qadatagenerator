// Package verify compares what a run promised against what an application did.
//
// # What this answers and what it refuses to answer
//
// The manifest says, for each record of a dataset, what a system reading it
// should do. The response record says what one system actually did. This puts
// the two side by side and reports where they disagree, in four kinds that are
// kept apart because they mean different things to whoever reads them.
//
// It does not decide whether the application is correct. An expectation this
// project calls soft is one where the right reaction depends on rules belonging
// to the application rather than to the data, so a disagreement there is
// reported and never made into a failure. Recorded as DEC-166.
package verify

import (
	"fmt"
	"sort"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/response"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Kind is the closed set of disagreements this build reports.
type Kind string

const (
	// Hole is a value that should have been refused and was taken.
	Hole Kind = "accepted although it should have been rejected"
	// Overzealous is a value that should have been taken and was refused.
	// This is what the REV module exists to find, and almost nobody tests it.
	Overzealous Kind = "rejected although it should have been accepted"
	// Silent is a value taken without the complaint the manifest expected.
	Silent Kind = "expected a mark, got silence"
	// Missing is a record the application never reacted to. It was neither
	// accepted nor refused, it disappeared on the way, and no check of one
	// row against one verdict finds it.
	Missing Kind = "the dataset arrived incomplete"
)

// kinds is every disagreement this build reports, which is what lets a surface
// be held to the whole set rather than to the ones it happened to think of.
var kinds = map[Kind]bool{Hole: true, Overzealous: true, Silent: true, Missing: true}

// Kinds is the closed set written out, sorted.
//
// # Why a reader needs the set and not only the value it was handed
//
// The command line is English by contract and prints these as they are. The
// window is translated, so it needs one phrase per kind, and a register of
// phrases can only be closed against a list. A fifth kind would otherwise reach
// a screen as an empty space rather than as a failing test.
//
// The detail beside a finding stays as this package writes it, in English, and
// that is a limit rather than an oversight. It names rows, columns and values
// in a sentence composed for the case at hand, so it is not a closed set and
// cannot become one without this package writing prose for every language.
func Kinds() []Kind {
	all := make([]Kind, 0, len(kinds))
	for kind := range kinds {
		all = append(all, kind)
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	return all
}

// Finding is one disagreement.
type Finding struct {
	Kind Kind
	// Entity is the table the finding is in, as the manifest names it. A
	// surface shows it only when Report.Entities names more than one, so a
	// run over one table reads as it always did.
	Entity     string
	Row        int
	Field      string
	Mutator    catalog.ID
	Confidence catalog.Confidence
	// Detail says what makes this one what it is, in words a person reads.
	Detail string
}

// Hard reports whether this is a disagreement the tool is entitled to fail on.
func (f Finding) Hard() bool { return f.Confidence == catalog.Hard }

// Unsettled is an entry the comparison could not judge, and why.
//
// Kept apart from the findings because it is not a disagreement. It is a
// question the response record did not carry the evidence to answer, and
// counting it as either a pass or a failure would be inventing the answer.
type Unsettled struct {
	Entity  string
	Row     int
	Field   string
	Mutator catalog.ID
	Reason  string
}

// Report is everything the comparison found.
type Report struct {
	Findings  []Finding
	Unsettled []Unsettled
	// Compared is how many applied entries were judged.
	Compared int
	// NeverReacted is how many records of the dataset the response record
	// answers for none of, which is the number inside the one Missing finding
	// that names no row. Carried as a number as well, so that a surface
	// saying it in fewer words than the sentence does not subtract a second
	// time and drift from the sentence.
	NeverReacted int
	// Limits is what this comparison could not see, given what the response
	// record carried. It travels with the answer for the same reason the
	// inference carries what it does not read: a report saying only what it
	// checked reads as a complete account of the run.
	Limits []string
	// Entities is every table of the dataset, in the order the manifest lists
	// them. One name, or one empty name for a manifest that named none, is a
	// dataset of one table.
	Entities []string

	// answers and tables are how the comparison placed the rows, kept so that
	// a surface asking what the application said about one entry asks the same
	// index the comparison judged with rather than building a second one.
	answers map[response.Place]response.Row
	tables  dataset
}

// Answer is what the application said about the row an entry stands in, and
// whether it said anything.
//
// The window asks this rather than the record, because the record knows rows
// by the names its producer wrote and only the comparison knows which table
// each of them is in. Asked of the record, row 1 of a customer table would be
// answered with whatever row 1 of an order table said.
func (r Report) Answer(defect manifest.Defect) (response.Row, bool) {
	if len(r.tables.tables) == 0 {
		return response.Row{}, false
	}
	answer, found := r.answers[r.tables.key(defect.Location.Entity, defect.Location.Row)]
	return answer, found
}

// Failed reports whether the run should end with the expectation code.
//
// Only hard disagreements count. A soft one means the correct reaction depends
// on the application's own rules, so failing on it would be this tool asserting
// something it was never entitled to assert, and the first run at anybody's
// site would be red for a reason nobody can act on.
func (r Report) Failed() bool {
	for _, finding := range r.Findings {
		if finding.Hard() {
			return true
		}
	}
	return false
}

// Compare puts a manifest beside a response record.
//
// It refuses a record whose rows cannot all be placed in the dataset the
// manifest describes: a row naming no entity in a dataset of several, a row
// naming an entity the manifest does not list, a row past the end of its table
// and two rows landing in one place. Each of those makes a verdict about some
// other record than the one the application saw, which is worse than no
// verdict, because it will be believed.
//
// # Why the answers are indexed by place
//
// The comparison asks about one row per entry, and a scan would make that
// quadratic. 🔴 The sizes are not hypothetical: the viewer of these documents
// calls a hundred thousand entries ordinary, and the largest record this build
// reads holds about four hundred thousand rows. Asking each of the first for
// one of the second by walking the list is ten billion comparisons for a verb
// that runs in somebody's pipeline.
func Compare(promised manifest.Manifest, actual response.Record) (Report, error) {
	tables := datasetOf(promised)
	answers, err := tables.place(actual)
	if err != nil {
		return Report{}, err
	}
	report := Report{Entities: tables.names(), answers: answers, tables: tables}
	missing := map[response.Place]bool{}
	for _, defect := range promised.Defects() {
		if !defect.Applied {
			// Damage that never happened has nothing in the dataset for the
			// application to have reacted to, so there is nothing to compare.
			// The manifest already carries why, and repeating it here would
			// make the report about this build rather than about the run.
			continue
		}
		report.Compared++
		at := tables.key(defect.Location.Entity, defect.Location.Row)
		answer, reacted := answers[at]
		if !reacted {
			if !missing[at] {
				missing[at] = true
				report.Findings = append(report.Findings, missingFinding(defect))
			}
			continue
		}
		report.add(judge(defect, answer))
	}
	report.addShortfall(tables, answers)
	report.Limits = limitsOf(promised, actual, tables)
	// Table by table in the order the manifest lists them, and row by row
	// inside one, so that the findings about one table are read together. A
	// dataset of one table sorts by row alone, as it always did.
	sort.SliceStable(report.Findings, func(i, j int) bool {
		left, right := report.Findings[i], report.Findings[j]
		if tables.order[left.Entity] != tables.order[right.Entity] {
			return tables.order[left.Entity] < tables.order[right.Entity]
		}
		return left.Row < right.Row
	})
	return report, nil
}

func (r *Report) add(finding *Finding, unsettled *Unsettled) {
	if finding != nil {
		r.Findings = append(r.Findings, *finding)
	}
	if unsettled != nil {
		r.Unsettled = append(r.Unsettled, *unsettled)
	}
}

// judge decides one entry against one verdict.
func judge(defect manifest.Defect, answer response.Row) (*Finding, *Unsettled) {
	switch defect.Expectation {
	case catalog.Rejection:
		if answer.Verdict == response.Accepted {
			return findingAbout(defect, Hole, whatIsHeld(defect, answer)), nil
		}
	case catalog.Acceptance:
		if answer.Verdict == response.Rejected {
			return findingAbout(defect, Overzealous, refusedWith(answer)), nil
		}
	case catalog.Warning:
		return judgeWarning(defect, answer)
	}
	return nil, nil
}

// judgeWarning decides an entry expecting the application to take the value and
// say something about it.
func judgeWarning(defect manifest.Defect, answer response.Row) (*Finding, *Unsettled) {
	if answer.Verdict == response.Rejected {
		return findingAbout(defect, Overzealous,
			"the manifest expected this to be taken and marked, and it was refused. "+
				refusedWith(answer)), nil
	}
	marked, known := answer.Marked()
	if !known {
		return nil, &Unsettled{Entity: defect.Location.Entity, Row: defect.Location.Row,
			Field: defect.Location.Field, Mutator: defect.Mutator,
			Reason: "the manifest expected a mark and the response record does not say whether " +
				"anything was marked. Add warnings to the row, even as an empty list, to settle it"}
	}
	if !marked {
		return findingAbout(defect, Silent, "the value was taken and nothing was said about it"), nil
	}
	return nil, nil
}

func missingFinding(defect manifest.Defect) Finding {
	return *findingAbout(defect, Missing, "the response record says nothing about this row, "+
		"so the application neither took it nor refused it")
}

// findingAbout is a disagreement about one entry, placed where the entry is.
func findingAbout(defect manifest.Defect, kind Kind, detail string) *Finding {
	return &Finding{Kind: kind, Entity: defect.Location.Entity, Row: defect.Location.Row,
		Field: defect.Location.Field, Mutator: defect.Mutator, Confidence: defect.Confidence,
		Detail: detail}
}

// whatIsHeld says what the application is holding now, when it said.
//
// 🔴 The distinction this draws is measured rather than imagined. A widely
// deployed database configuration takes a row, turns text into zero and cuts an
// overlong string to the column width, reporting no error at all. A system
// holding something other than what it was sent did not take the value, it took
// a different one, and calling both of those "accepted" would hide the second.
func whatIsHeld(defect manifest.Defect, answer response.Row) string {
	if !defect.HasValueAfter {
		return "the value was taken"
	}
	changed, known := answer.Changed(defect.Location.Field, defect.ValueAfter)
	switch {
	case !known:
		return "the value was taken. The response record does not say what is held now, so " +
			"whether it survived as sent or was quietly changed is not settled here"
	case changed:
		return "the value was taken and what is held now is not what was sent, so it was not " +
			"refused and not kept either"
	default:
		return "the value was taken and is held exactly as sent"
	}
}

func refusedWith(answer response.Row) string {
	if answer.Code == "" && answer.Message == "" {
		return "the response record does not say why"
	}
	if answer.Code == "" {
		return "the application said: " + answer.Message
	}
	return "the application answered " + answer.Code
}

// addShortfall reports records the application never reacted to at all.
//
// This is the case no comparison of one row against one verdict finds, because
// both sides of such a comparison assume the row was seen. It is counted from
// the dataset rather than from the damaged rows, since a receiver dropping
// everything past a limit drops undamaged records just as readily.
//
// # Why it is counted from what was placed and not from the length of the list
//
// Until 2026-10-27 the shortfall was the rows of the dataset less the rows of
// the record, so a record answering for row 61 of a dataset of sixty made up
// for a row that never arrived and the run reported nothing missing (R21). Every
// row is placed in its table before this runs, and a row past the end or a
// second answer for one place refuses the record, so what was placed in a table
// is exactly how much of it was answered.
//
// One finding per table that came up short, naming the table when the dataset
// has several. A sum would say records went missing without saying from where,
// and the table is the first thing whoever reads it has to go and look at.
func (r *Report) addShortfall(tables dataset, answers map[response.Place]response.Row) {
	if !tables.counted {
		r.addShort("", tables.total, len(answers))
		return
	}
	answered := make(map[string]int, len(tables.tables))
	for at := range answers {
		answered[at.Entity]++
	}
	for _, table := range tables.tables {
		entity := ""
		if tables.several() {
			entity = table.Name
		}
		r.addShort(entity, table.Rows, answered[table.Name])
	}
}

// addShort reports the records of one table, or of the whole dataset when the
// entity is empty, that the application never reacted to.
func (r *Report) addShort(entity string, total, answered int) {
	if total <= 0 || answered >= total {
		return
	}
	// The verb follows the count, which the real window showed reading "so 1
	// were never reacted to" (B-289). So does the noun before it: with one
	// table a record answering for no row is refused by the reader, but a
	// table of one record nobody answered for is ordinary once there are two.
	missing := total - answered
	r.NeverReacted += missing
	were := "were"
	if missing == 1 {
		were = "was"
	}
	records := "records"
	if total == 1 {
		records = "record"
	}
	whose := fmt.Sprintf("the dataset had %d %s", total, records)
	if entity != "" {
		whose += " of " + entity
	}
	r.Findings = append(r.Findings, Finding{
		Kind: Missing, Entity: entity, Confidence: catalog.Hard,
		Detail: fmt.Sprintf("%s and the response record answers for %d, so %d %s never "+
			"reacted to", whose, answered, missing, were),
	})
}

// limitsOf says what the comparison could not see, given what the two
// documents carried and the versions they were written to.
//
// # Why a newer version is said here and not only by the command line
//
// The command line said it before the report until 2026-10-27, and the window
// never did, so the same record compared in both was a complete account in one
// and a partial one in the other. From response 0.2 the reader skips members of
// a newer minor version it does not know rather than refusing them (DEC-266),
// which makes saying so the only trace of what was skipped. A limit is what
// both surfaces already carry to the end.
func limitsOf(promised manifest.Manifest, actual response.Record, tables dataset) []string {
	limits := newerThanThisBuild(promised, actual)
	if !tables.counted {
		limits = append(limits, "the manifest does not say how many rows each entity had, which "+
			"a manifest written to 0.1 never did, so a row past the end of its entity and the "+
			"records that went missing are counted against the dataset as a whole")
	}
	if !anyRow(actual, func(row response.Row) bool { return row.HasStored }) {
		limits = append(limits, "no row says what the application holds now, so a value taken as "+
			"sent and a value quietly changed on the way in look the same here")
	}
	if !anyRow(actual, func(row response.Row) bool { return row.HasWarnings }) {
		limits = append(limits, "no row says whether anything was marked, so an entry expecting a "+
			"mark cannot be settled")
	}
	if !actual.HasRowsTotal {
		limits = append(limits, "the response record does not say how many rows it was given, so "+
			"the count of records that went missing rests on the manifest alone")
	}
	return limits
}

// newerThanThisBuild says which of the two documents is written to a minor
// version this build has never seen, and so was read only as far as this build
// understands it. An older version says nothing: the formats grow by adding, so
// everything an older document can say is something this build knows.
func newerThanThisBuild(promised manifest.Manifest, actual response.Record) []string {
	var limits []string
	for _, document := range []struct{ name, speaks, claimed string }{
		{"manifest", version.Manifest, promised.SpecVersion()},
		{"response record", version.Response, actual.SpecVersion},
	} {
		if version.Compatibility(document.speaks, document.claimed) == version.Later {
			limits = append(limits, fmt.Sprintf("the %s is written to version %s of its format and "+
				"this build speaks %s, so it is read as far as this build understands it and "+
				"anything the newer version added is not part of this comparison",
				document.name, fault.Shorten(document.claimed), document.speaks))
		}
	}
	return limits
}

func anyRow(actual response.Record, holds func(response.Row) bool) bool {
	for _, row := range actual.Rows {
		if holds(row) {
			return true
		}
	}
	return false
}
