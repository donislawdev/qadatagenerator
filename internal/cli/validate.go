package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
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
	askedForHelp := declareVerbHelp(set)
	if err := set.Parse(arguments); err != nil {
		return usage(out, flagFailure(err, acceptedFlagsOf(commandValidate), problemHelpSubjectDo+" "+commandValidate))
	}
	if askedForHelp() {
		return printCommandHelp(out, commandValidate)
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
	return validateRecipeAt(out, arguments[0], flags)
}

// validateRecipeAt reads one recipe and reports what is wrong with it.
//
// Split from the line above it rather than nested inside, so that reading the
// command line and doing what it asked stay two things. It is the same shape
// the other verb already has, where generateFrom carries the work.
func validateRecipeAt(out *streams, path string, flags crossCutting) int {
	data, err := readRecipe(path)
	if err != nil {
		// A directory is answered in our own words rather than the operating
		// system's. Left to the system it read "Incorrect function." on Windows
		// and "is a directory" elsewhere, so the same mistake produced two
		// messages and neither said what to do next.
		if errors.Is(err, errIsDirectory) {
			out.report(problem{
				what:    problemRecipeIsDirectory + path,
				why:     problemRecipeIsDirectoryWhy,
				allowed: problemRecipeIsDirectoryAllowed,
				instead: problemRecipeIsDirectoryDo,
			})
			return int(codeIO)
		}
		// Nothing was validated, so there is no report to write in either
		// shape. A reader that got no document on the data channel and a
		// read failure on the log channel has the whole story.
		return fail(out, fault.Wrap(fault.IO, err, "cannot read "+path))
	}

	// Both registries travel in as ports, so that this verb answers the same
	// questions generate and corrupt would about the same file. The catalog
	// arrives beside the generators rather than later, because a validator
	// that could see only one of them is how the two verbs came to disagree
	// once already.
	described, catalogErr := catalogyaml.Load()
	if catalogErr != nil {
		return fail(out, catalogErr)
	}
	report, runErr := validate.Run(decodeRecipe, validate.Source{Name: path, Data: data},
		values.New("", ""), mutators.Coverage{Described: described})
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

// readRecipe reads a recipe and stops one byte past the largest one this build
// accepts.
//
// # Why not simply read the file
//
// Reading it whole and refusing it afterwards holds exactly the memory the
// limit exists to refuse. The limit stopped the parser, which is what it was
// written for, and did nothing about the buffer the bytes arrived in.
//
// Measured on 2026-08-02, before this: a 250 MB file cost 258 MB of memory,
// three runs alike, on its way to being told it was over a 256 kB limit. The
// message even quoted its size, which it could only know by holding it. In a
// pipeline that is a build killed by the machine rather than an exit code from
// the table, and a killed build says nothing about why.
//
// The one extra byte is what lets the decoder tell a recipe sitting exactly on
// the limit from one past it.
func readRecipe(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	// Opening a directory succeeds on both systems this runs on and only the
	// read fails, with a message from the operating system rather than from
	// here. Asking first is what lets the refusal be ours.
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errIsDirectory
	}

	return io.ReadAll(io.LimitReader(file, recipeyaml.MaxSize+1))
}

// errIsDirectory marks the one read failure this command can describe better
// than the operating system does.
var errIsDirectory = errors.New("the path is a directory")

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
			what = found.Where + placeSuffix(found.Line) + ": " + what
		}
		out.report(problem{what: what, allowed: found.Allowed, instead: found.Hint})
	}

	// The headline carries the count and, when the detail below it is being
	// suppressed, says how to see it.
	//
	// It used to be a fixed sentence with the entries underneath it on a
	// channel that --quiet silences. That left the one combination worth
	// avoiding: a warning saying something was skipped, with nothing anywhere
	// saying what. A warning has to be complete on its own, because the line
	// beneath it is not guaranteed to arrive.
	if len(report.Unchecked) > 0 {
		out.warn(uncheckedHeadline(len(report.Unchecked), out.hidesDetail()))
		for _, skipped := range report.Unchecked {
			out.detail(skipped.Where + ": " + skipped.What + " - " + skipped.Reason)
		}
	}

	if report.Usable() {
		out.note(report.Name + summaryValid + understood(report.Entities, report.Fields))
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

// jsonProblem is a problem as a build reads it.
//
// 🔴 It is converted from the domain type by a plain struct conversion, which
// is a guard rather than a convenience: the day that type gains a field, this
// stops compiling and somebody has to decide whether the field belongs on a
// surface other people read. That is what happened when the line arrived.
//
// The line is omitted when there is none, because a problem about the file as a
// whole belongs to no line and a zero there would read as the first one.
type jsonProblem struct {
	Where   string `json:"where"`
	What    string `json:"what"`
	Allowed string `json:"allowed,omitempty"`
	Hint    string `json:"hint,omitempty"`
	Line    int    `json:"line,omitempty"`
}

type jsonNote struct {
	Where  string `json:"where"`
	What   string `json:"what"`
	Reason string `json:"reason"`
}

// placeSuffix says which line a place was written on, or nothing when it was
// never written.
//
// The line comes after the place rather than before it, because the list is
// ordered by place: problems about one field belong together, and a number in
// front of each would read as an ordering the list does not have.
func placeSuffix(line int) string {
	if line == 0 {
		return ""
	}
	return " (line " + strconv.Itoa(line) + ")"
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
