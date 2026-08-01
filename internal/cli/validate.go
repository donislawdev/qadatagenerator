package cli

import (
	"encoding/json"
	"flag"
	"io"
	"os"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// runValidate reads a recipe and says whether it can be used.
//
// This is the only place where the pieces are wired together: the file is read
// here, the decoding is handed to the use case as a port, and the exit code
// comes from the failure the use case returned rather than from a decision
// made here.
func runValidate(out *streams, arguments []string, global crossCutting) int {
	// The command declares the cross cutting flags again, so that they mean
	// the same thing after the verb as before it.
	set := flag.NewFlagSet(commandValidate, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	local := declareCrossCutting(set)
	if err := set.Parse(arguments); err != nil {
		return usage(out, problem{
			what:    err.Error(),
			why:     problemUnknownFlagWhy,
			allowed: acceptedFlags(),
			instead: problemUnknownFlagDo,
		})
	}
	flags := global.merge(local)
	if code, ok := applyVerbosity(out, flags); !ok {
		return code
	}

	arguments = set.Args()
	if len(arguments) != 1 {
		return usage(out, problem{
			what:    problemValidateArguments,
			why:     problemValidateArgumentsWhy,
			allowed: problemValidateArgumentsAllowed,
			instead: problemValidateArgumentsDo,
		})
	}

	path := arguments[0]
	data, err := os.ReadFile(path)
	if err != nil {
		// Nothing was validated, so there is no report to write in either
		// shape. A reader that got no document on the data channel and a
		// read failure on the log channel has the whole story.
		return fail(out, fault.Wrap(fault.IO, err, "cannot read "+path))
	}

	report, runErr := validate.Run(decodeRecipe, validate.Source{Name: path, Data: data})
	if *flags.json {
		writeJSONReport(out, report)
	} else {
		writeReport(out, report)
	}
	if runErr != nil {
		return finishFailed(out, runErr)
	}
	return finish(out, codeOK)
}

// decodeRecipe is the port the use case is given. The conversion holds because
// both structures carry the same three things, and writing it out here keeps
// the use case from knowing that recipes are YAML.
func decodeRecipe(data []byte) validate.Decoded {
	return validate.Decoded(recipeyaml.Decode(data))
}

// writeReport puts the findings on the log channel.
//
// Not on the data channel, and that is deliberate. A report about a file is
// not the data the tool produces, and a later --json has to be able to put
// machine readable findings on standard output without a word of prose above
// them.
func writeReport(out *streams, report validate.Report) {
	for _, found := range report.Problems {
		// A problem with the file as a whole has no place inside it, and
		// prefixing it with an empty place reads as a missing word.
		what := found.What
		if found.Where != "" {
			what = found.Where + ": " + what
		}
		out.report(problem{what: what, allowed: found.Allowed, instead: found.Hint})
	}

	if len(report.Unchecked) > 0 {
		out.warn(problemUnchecked)
		for _, skipped := range report.Unchecked {
			out.detail(skipped.Where + ": " + skipped.What + " - " + skipped.Reason)
		}
	}

	if report.Usable() {
		out.note(report.Name + summaryValid +
			strconv.Itoa(report.Entities) + summaryEntities +
			strconv.Itoa(report.Fields) + summaryFields)
	}
}

// The shape below is what a pipeline parses, so it is a public contract and
// not an implementation detail. The four parts of a problem stay four fields
// rather than one sentence, because a pipeline that wants to decide which
// findings block a build should not have to take prose apart.
type jsonReport struct {
	Recipe    string        `json:"recipe"`
	Usable    bool          `json:"usable"`
	Entities  int           `json:"entities"`
	Fields    int           `json:"fields"`
	Problems  []jsonProblem `json:"problems"`
	Unchecked []jsonNote    `json:"unchecked"`
}

type jsonProblem struct {
	Where   string `json:"where"`
	What    string `json:"what"`
	Allowed string `json:"allowed,omitempty"`
	Hint    string `json:"hint,omitempty"`
}

type jsonNote struct {
	Where  string `json:"where"`
	What   string `json:"what"`
	Reason string `json:"reason"`
}

// writeJSONReport puts the findings on the data channel, whether or not the
// recipe turned out to be usable.
//
// A run that ends badly writes no data, and this does not break that rule: the
// rule is there so that a half written dataset never looks complete. A report
// asked for by name is the answer to the question, not a piece of a dataset,
// and withholding it exactly when it is needed would make the flag useless.
func writeJSONReport(out *streams, report validate.Report) {
	document := jsonReport{
		Recipe:    report.Name,
		Usable:    report.Usable(),
		Entities:  report.Entities,
		Fields:    report.Fields,
		Problems:  make([]jsonProblem, 0, len(report.Problems)),
		Unchecked: make([]jsonNote, 0, len(report.Unchecked)),
	}
	for _, found := range report.Problems {
		document.Problems = append(document.Problems, jsonProblem(found))
	}
	for _, skipped := range report.Unchecked {
		document.Unchecked = append(document.Unchecked, jsonNote(skipped))
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		out.dataErr = err
		return
	}
	out.emit(string(encoded))
}
