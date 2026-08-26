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
	Kind       Kind
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
	// Limits is what this comparison could not see, given what the response
	// record carried. It travels with the answer for the same reason the
	// inference carries what it does not read: a report saying only what it
	// checked reads as a complete account of the run.
	Limits []string
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
func Compare(promised manifest.Manifest, actual response.Record) Report {
	report := Report{}
	missing := map[int]bool{}
	answers := byRow(actual)
	for _, defect := range promised.Defects() {
		if !defect.Applied {
			// Damage that never happened has nothing in the dataset for the
			// application to have reacted to, so there is nothing to compare.
			// The manifest already carries why, and repeating it here would
			// make the report about this build rather than about the run.
			continue
		}
		report.Compared++
		answer, reacted := answers[defect.Location.Row]
		if !reacted {
			if !missing[defect.Location.Row] {
				missing[defect.Location.Row] = true
				report.Findings = append(report.Findings, missingFinding(defect))
			}
			continue
		}
		report.add(judge(defect, answer))
	}
	report.addShortfall(promised, actual)
	report.Limits = limitsOf(actual)
	sort.SliceStable(report.Findings, func(i, j int) bool {
		return report.Findings[i].Row < report.Findings[j].Row
	})
	return report
}

// byRow indexes the answers once, because the comparison asks about one row per
// entry and a scan would make that quadratic.
//
// 🔴 The sizes are not hypothetical: the viewer of these documents calls a
// hundred thousand entries ordinary, and the largest record this build reads
// holds about four hundred thousand rows. Asking each of the first for one of
// the second by walking the list is ten billion comparisons for a verb that
// runs in somebody's pipeline.
//
// The first answer for a row wins, which is the answer the scan gave. A record
// naming one row twice is refused by the model before it reaches here, so the
// rule matters only for keeping the two the same.
func byRow(actual response.Record) map[int]response.Row {
	indexed := make(map[int]response.Row, len(actual.Rows))
	for _, row := range actual.Rows {
		if _, already := indexed[row.Number]; !already {
			indexed[row.Number] = row
		}
	}
	return indexed
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
			return &Finding{Kind: Hole, Row: defect.Location.Row, Field: defect.Location.Field,
				Mutator: defect.Mutator, Confidence: defect.Confidence,
				Detail: whatIsHeld(defect, answer)}, nil
		}
	case catalog.Acceptance:
		if answer.Verdict == response.Rejected {
			return &Finding{Kind: Overzealous, Row: defect.Location.Row, Field: defect.Location.Field,
				Mutator: defect.Mutator, Confidence: defect.Confidence,
				Detail: refusedWith(answer)}, nil
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
		return &Finding{Kind: Overzealous, Row: defect.Location.Row, Field: defect.Location.Field,
			Mutator: defect.Mutator, Confidence: defect.Confidence,
			Detail: "the manifest expected this to be taken and marked, and it was refused. " +
				refusedWith(answer)}, nil
	}
	marked, known := answer.Marked()
	if !known {
		return nil, &Unsettled{Row: defect.Location.Row, Field: defect.Location.Field,
			Mutator: defect.Mutator,
			Reason: "the manifest expected a mark and the response record does not say whether " +
				"anything was marked. Add warnings to the row, even as an empty list, to settle it"}
	}
	if !marked {
		return &Finding{Kind: Silent, Row: defect.Location.Row, Field: defect.Location.Field,
			Mutator: defect.Mutator, Confidence: defect.Confidence,
			Detail: "the value was taken and nothing was said about it"}, nil
	}
	return nil, nil
}

func missingFinding(defect manifest.Defect) Finding {
	return Finding{Kind: Missing, Row: defect.Location.Row, Field: defect.Location.Field,
		Mutator: defect.Mutator, Confidence: defect.Confidence,
		Detail: "the response record says nothing about this row, so the application neither " +
			"took it nor refused it"}
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
func (r *Report) addShortfall(promised manifest.Manifest, actual response.Record) {
	total := promised.Run().RowsTotal
	if total <= 0 || len(actual.Rows) >= total {
		return
	}
	r.Findings = append(r.Findings, Finding{
		Kind: Missing, Confidence: catalog.Hard,
		Detail: fmt.Sprintf("the dataset had %d records and the response record answers for %d, "+
			"so %d were never reacted to", total, len(actual.Rows), total-len(actual.Rows)),
	})
}

// limitsOf says what the response record did not carry.
func limitsOf(actual response.Record) []string {
	var limits []string
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

func anyRow(actual response.Record, holds func(response.Row) bool) bool {
	for _, row := range actual.Rows {
		if holds(row) {
			return true
		}
	}
	return false
}
