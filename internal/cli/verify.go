package cli

import (
	"flag"
	"io"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/responsejson"
	"github.com/donislawdev/qadatagenerator/internal/app/verify"
)

// runVerify compares a manifest against what an application did with the data.
//
// # The one verb that fails on somebody else's behaviour
//
// Every other verb reports on its own work. This one reports on an application
// it never ran, using two files it did not both write, and it is the only verb
// that returns the expectation code. That code has been in the table since the
// first day with nothing to return it, which is what made this verb a
// requirement rather than an idea.
func runVerify(out *streams, arguments []string, global crossCutting) int {
	set := flag.NewFlagSet(commandVerify, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	local := declareCrossCutting(set)
	askedForHelp := declareVerbHelp(set)
	if err := set.Parse(arguments); err != nil {
		return usage(out, flagFailure(err, acceptedFlagsOf(commandVerify),
			problemHelpSubjectDo+" "+commandVerify))
	}
	if askedForHelp() {
		return printCommandHelp(out, commandVerify)
	}
	flags := global.merge(local)
	if code, ok := applyVerbosity(out, flags); !ok {
		return code
	}

	arguments = set.Args()
	if len(arguments) != 2 {
		return usage(out, problem{
			what:    problemVerifyArguments,
			why:     problemVerifyArgumentsWhy,
			allowed: problemVerifyArgumentsAllowed,
			instead: problemVerifyArgumentsDo,
		})
	}
	return compareAt(out, arguments[0], arguments[1])
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

	report := verify.Compare(promised, actual)
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
	for _, finding := range report.Findings {
		out.warn(placeOf(finding) + string(finding.Kind) + ". " + finding.Detail)
	}
	for _, open := range report.Unsettled {
		out.warn("row " + strconv.Itoa(open.Row) + ", " + open.Field + ": " + open.Reason)
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

// placeOf names where a finding is, and says nothing when it is about the
// dataset rather than about one row.
func placeOf(finding verify.Finding) string {
	if finding.Row == 0 {
		return ""
	}
	return "row " + strconv.Itoa(finding.Row) + ", " + finding.Field + ": "
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
