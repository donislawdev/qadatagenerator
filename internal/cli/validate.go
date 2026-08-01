package cli

import (
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
func runValidate(out *streams, arguments []string) int {
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
		return fail(out, fault.Wrap(fault.IO, err, "cannot read "+path))
	}

	report, runErr := validate.Run(decodeRecipe, validate.Source{Name: path, Data: data})
	writeReport(out, report)
	if runErr != nil {
		return fail(out, runErr)
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
