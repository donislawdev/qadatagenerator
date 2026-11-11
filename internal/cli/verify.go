package cli

import (
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/responsejson"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
)

// verifyWith compares a manifest against what an application did with the data,
// once its arguments have been read.
//
// # The one verb that fails on somebody else's behaviour
//
// Every other verb reports on its own work. This one reports on an application
// it never ran, using two files it did not both write, and it is the only verb
// that returns the expectation code. That code has been in the table since the
// first day with nothing to return it, which is what made this verb a
// requirement rather than an idea.
func verifyWith(out *streams, given parsed[noFlags]) int {
	paths := given.rest
	if len(paths) != 2 {
		return usage(out, problem{
			what:    problemVerifyArguments,
			why:     problemVerifyArgumentsWhy,
			allowed: problemVerifyArgumentsAllowed,
			instead: problemVerifyArgumentsDo,
		})
	}
	return compareAt(out, paths[0], paths[1])
}

// compareAt reads both files and reports where they disagree.
func compareAt(out *streams, manifestPath, responsePath string) int {
	promised, err := manifestjson.Load(manifestPath)
	if err != nil {
		return fail(out, err)
	}
	actual, err := responsejson.Load(responsePath)
	if err != nil {
		return fail(out, err)
	}

	// The refusal for a version this build cannot read happened in the
	// readers. A document written to a newer minor version is read, and the
	// comparison says so among its limits, where the window says it too.
	report, err := verify.Compare(promised, actual)
	if err != nil {
		return fail(out, err)
	}
	say(out, report)
	if report.Failed() {
		return finish(out, codeExpectation)
	}
	return finish(out, codeOK)
}

// say writes the report on the log channel.
//
// 🔴 The order is what a reader needs rather than what is easiest to write.
// The disagreements come first because they are why somebody ran this. What
// could not be settled comes next, because an unanswered question read as a
// pass is the failure this whole project argues against. The limits come last
// and always, because a report saying only what it checked reads as a complete
// account of the run.
func say(out *streams, report verify.Report) {
	// Written out rather than handed to the general pluraliser, which appends
	// to what it is given and says "entrys".
	word := noteVerifyEntries
	if report.Compared == 1 {
		word = noteVerifyOneEntry
	}
	out.note(noteVerifyCompared + strconv.Itoa(report.Compared) + word)
	several := len(report.Entities) > 1
	for _, finding := range report.Findings {
		out.warn(placeOf(several, finding.Entity, finding.Row, finding.Field) +
			followedBy(string(finding.Kind), finding.Detail))
	}
	for _, open := range report.Unsettled {
		out.warn(placeOf(several, open.Entity, open.Row, open.Field) + open.Reason)
	}
	if len(report.Findings) == 0 {
		out.note(noteVerifyAgreed)
	}
	if soft := softCount(report); soft > 0 {
		out.note(counted(soft, "disagreement") + noteVerifySoft)
	}
	for _, limit := range report.Limits {
		out.detail(noteVerifyLimit + limit)
	}
}

// followedBy puts the detail of a finding after its kind as a sentence of its
// own.
//
// The detail is written to follow other words, so it begins in lower case, and
// the line read "the dataset arrived incomplete. the dataset had 24 records..."
// (B-298). Raising its first letter is safe here and not everywhere: every
// detail the comparison writes begins with its own words ("the value", "the
// application", "the manifest", "the response record", "the dataset"), never with a value from
// somebody's data, where a capital would change what was sent.
func followedBy(kind, detail string) string {
	if detail == "" {
		return kind
	}
	first, width := utf8.DecodeRuneInString(detail)
	return kind + ". " + string(unicode.ToUpper(first)) + detail[width:]
}

// placeOf names where a finding is, and says nothing when it is about the
// dataset rather than about one row.
//
// The entity is named only in a dataset of several, where row 3 alone could be
// in any of them. A run over one table reads as it did before the dataset could
// have more than one.
func placeOf(several bool, entity string, row int, field string) string {
	if row == 0 {
		return ""
	}
	if several {
		return "row " + strconv.Itoa(row) + " of " + entity + ", " + field + ": "
	}
	return "row " + strconv.Itoa(row) + ", " + field + ": "
}

func softCount(report verify.Report) int {
	soft := 0
	for _, finding := range report.Findings {
		if !finding.Hard() {
			soft++
		}
	}
	return soft
}
