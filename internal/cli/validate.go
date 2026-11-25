package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// validateWith reads a recipe and says whether it can be used, once its
// arguments have been read.
//
// This is the only place where the pieces are wired together: the file is read
// here, the decoding is handed to the use case as a port, and the exit code
// comes from the failure the use case returned rather than from a decision
// made here.
func validateWith(out *streams, given parsed[noFlags]) int {
	if len(given.rest) != 1 {
		return usage(out, problem{
			what:    problemValidateArguments,
			why:     problemValidateArgumentsWhy,
			allowed: problemValidateArgumentsAllowed,
			instead: problemValidateArgumentsDo,
		})
	}
	return validateRecipeAt(out, given.rest[0], given.flags)
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
	// Read once and handed to both, the way the window already does it. The use
	// case takes a decoder rather than a document, and the engine below needs
	// the document, so a closure over what was read keeps the use case from
	// knowing that recipes are YAML while costing one reading instead of two.
	// A recipe may be 256 KB and reading one that size is measured in hundreds
	// of megabytes, so the difference is not academic.
	decoded := decodeRecipe(data)
	report, runErr := validate.Run(func([]byte) validate.Decoded { return decoded },
		validate.Source{Name: path, Data: data},
		values.New("", ""), mutators.Coverage{Described: described})

	// The second half of the question, and it is put only to a recipe the first
	// half approved: a plan of a document with problems in it would report the
	// same problems again in worse words.
	outlook := whatARunWouldDo{}
	if runErr == nil {
		outlook, runErr = alsoAskTheEngine(&report, decoded.Recipe, path)
	}

	if *flags.json {
		writeJSONReport(out, report, outlook)
	} else {
		writeReport(out, report, outlook)
	}
	if runErr != nil {
		return finishFailed(out, runErr)
	}
	return finish(out, codeOK)
}

// alsoAskTheEngine puts the second half of the question and folds the answer
// into the report.
//
// A refusal that says the recipe is wrong joins the problems, so that it
// arrives in the same list and the same shape as every other one, in both the
// prose and the machine readable form. The count in the failure is rebuilt
// rather than taken from the use case, because the list it counts has just
// grown.
//
// Split from the caller because that function reached the length this project
// holds a function to, and the limit is bent for nothing.
func alsoAskTheEngine(report *validate.Report, rec recipe.Recipe,
	path string) (whatARunWouldDo, error) {
	outlook := askThePlan(rec)
	if outlook.wrong == nil {
		return outlook, nil
	}
	// The refusal's own next step goes with it, since one engine refusal can
	// be fixed in several ways and only the refusal knows which.
	report.Problems = append(report.Problems, recipe.Problem{What: outlook.wrong.Error(),
		Hint: fault.InsteadOf(outlook.wrong)})
	return outlook, fault.New(fault.Config,
		counted(len(report.Problems), "problem")+" in "+path)
}

// whatARunWouldDo is what the engine says about a recipe that reads correctly.
//
// # Why this verb asks the engine at all
//
// 🔴 Because half of what makes a recipe unusable is only known there, and
// until 2026-09-02 this verb said nothing about that half. Measured with the
// real binary on that day: a recipe whose relations form a loop was answered
// can be used, and so was one whose entity belongs to two others, and so was
// one asking for a relation wider than the arithmetic behind it. The first two
// are refusals nobody could argue with, and the sentences refusing them exist
// in one file in the engine and nowhere else, so this verb could not have
// produced them. The README promises that this verb reports every problem a
// recipe has, and it was reporting the problems one reader could see.
//
// # Why the kind of the failure decides what happens to it, rather than a rule
// written here
//
// The engine already refuses on both sides of a line this project drew and
// guards: exit code 3 says a document is not valid, and 5 says a valid one
// cannot be carried out. DEC-116 put the second of those outside this verb on
// purpose, because whether a run is worth attempting is the engine's question
// and has its own code. So a refusal that says the recipe is wrong becomes a
// problem here, and one that says it cannot be satisfied becomes a warning
// beside an answer that is still yes.
//
// Nothing new decides which is which. The register in internal/guard holds
// every refusal either side of that boundary together with why it is there,
// and a new one that says neither turns the build red.
type whatARunWouldDo struct {
	tables, rows int
	// notes are what a run of this recipe would not do, and implied are the
	// agreements it would keep with no rule written for them.
	//
	// 🔴 Both were worked out here and thrown away until 2026-10-21, so a rule
	// such as pesel matches: [first_name, gender] was answered can be used by
	// this verb while generate warned that half of it does nothing. Two verbs
	// asked about one recipe answered differently, which is the drift asking
	// the plan here exists to prevent.
	notes   []generate.Note
	implied []generate.Implied
	// wrong is a fault of the recipe that only the engine asks about.
	wrong error
	// cannot is a recipe that is right and could not be carried out.
	cannot error
}

// askThePlan works out what a run would produce, without producing it.
//
// The values carry the recipe's own locale and seed, unlike the registry the
// use case above is given, which is asked only which names exist. A generator
// this build cannot fill in the recipe's locale is a refusal that depends on
// the locale, so asking with an empty one would answer a question nobody put.
//
// # What the count of rows is an answer about
//
// The recipe as it is written, and this verb has no flag that could change
// that. A relation with a range in it has its shares drawn from the seed, so
// generate given a different seed writes a different number of rows and both
// numbers are right. The three flags that override a recipe belong to the verb
// that runs it, which is where somebody who overrode one will be reading.
//
// Nothing here can stop the plan, and nothing has to: this verb watches no
// signal, so an interrupt ends the process the way the system ends any other,
// and there is nothing on the disk to take back.
func askThePlan(rec recipe.Recipe) whatARunWouldDo {
	// The plan generate and the window follow, notes about settings the form
	// asked for ignores included, so that this verb cannot be the one surface
	// silent about them.
	prepared, err := produce.Prepare(rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
	// The writer of each form asks questions of its own, such as whether a
	// parent field is named after a child in the tree or whether a parent of
	// the SQL form has a key of one field, and validate asks them too so that
	// the two verbs agree.
	if err == nil {
		err = format.Fits(rec)
	}
	switch {
	case err == nil:
		plan := prepared.Plan()
		return whatARunWouldDo{tables: len(plan.Entities), rows: plan.TotalRows(),
			notes: plan.Notes, implied: plan.Implied}
	case fault.KindOf(err) == fault.Config:
		return whatARunWouldDo{wrong: err}
	default:
		return whatARunWouldDo{cannot: err}
	}
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
// writeUnchecked says what was carried without being read, in one form for
// validate and for generate.
//
// The headline carries the count and, when the detail below it is being
// suppressed, says how to see it.
//
// It used to be a fixed sentence with the entries underneath it on a channel
// that --quiet silences. That left the one combination worth avoiding: a
// warning saying something was skipped, with nothing anywhere saying what. A
// warning has to be complete on its own, because the line beneath it is not
// guaranteed to arrive.
func writeUnchecked(out *streams, unchecked []recipe.Unchecked) {
	if len(unchecked) == 0 {
		return
	}
	out.warn(uncheckedHeadline(len(unchecked), out.hidesDetail()))
	for _, skipped := range unchecked {
		out.detail(skipped.Where + ": " + skipped.What + " - " + skipped.Reason)
	}
}

func writeReport(out *streams, report validate.Report, outlook whatARunWouldDo) {
	for _, found := range report.Problems {
		// A problem with the file as a whole has no place inside it, and
		// prefixing it with an empty place reads as a missing word.
		what := found.What
		if found.Where != "" {
			what = found.Where + placeSuffix(found.Line) + ": " + what
		}
		out.report(problem{what: what, allowed: found.Allowed, instead: found.Hint})
	}

	writeUnchecked(out, report.Unchecked)

	if !report.Usable() {
		return
	}
	if outlook.cannot != nil {
		// A warning rather than a problem, and the answer below it is still
		// yes. The recipe is right, and whether a run of it can be carried out
		// is a question with an exit code of its own on another verb.
		out.warn(outlook.cannot.Error())
		out.note(report.Name + summaryValidUnrunnable)
		return
	}
	// The same lines generate writes, in the same order, for the reason the
	// count below uses the dry run's words.
	for _, skipped := range outlook.notes {
		out.warn(skipped.Where + ": " + skipped.What)
	}
	for _, kept := range outlook.implied {
		out.detail(impliedLine(kept))
	}
	out.note(report.Name + summaryValid + understood(report.Entities, report.Fields))
	// What a run would write, said in the same words the dry run uses, because
	// two sentences about one number are how somebody comes to believe there
	// are two numbers.
	out.note(summaryWouldWrite + counted(outlook.rows, "row") + " in " +
		counted(outlook.tables, "table") + ".")
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

	// Tables and Rows are what a run of this recipe would write, and they mean
	// something only when usable is true and unrunnable is absent. That is
	// said here rather than left to be worked out, because nought rows and
	// nothing worked out are the same number.
	Tables int `json:"tables"`
	Rows   int `json:"rows"`
	// Unrunnable is why a recipe that reads correctly could not be carried
	// out. It is absent when it could, which is the ordinary case, and a
	// pipeline that wants this to fail a build reads it rather than the exit
	// code, because the recipe is not what is wrong.
	Unrunnable string `json:"unrunnable,omitempty"`
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
func writeJSONReport(out *streams, report validate.Report, outlook whatARunWouldDo) {
	document := jsonReport{
		Recipe:    report.Name,
		Usable:    report.Usable(),
		Entities:  report.Entities,
		Fields:    report.Fields,
		Problems:  make([]jsonProblem, 0, len(report.Problems)),
		Unchecked: make([]jsonNote, 0, len(report.Unchecked)),
		Tables:    outlook.tables,
		Rows:      outlook.rows,
	}
	if outlook.cannot != nil {
		document.Unrunnable = outlook.cannot.Error()
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
