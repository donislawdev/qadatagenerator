package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// generateFlags are the settings this verb takes beyond the cross cutting ones.
//
// # Why each one is a pointer and a name, rather than a value
//
// A flag that was not given must not overwrite what the recipe says, and a flag
// given the same value as the default must. Those two are the same value and
// different intentions, so the parser is asked afterwards which names it
// actually saw. Getting this wrong is the quietest kind of surprise in a tool
// configured from two sides, and it is cheaper to build now than to retrofit
// through a whole parser later.
type generateFlags struct {
	out       *string
	seed      *string
	format    *string
	overwrite *bool
	dryRun    *bool
	// manifest is the extra form the manifest is exported in, empty for none.
	// It is the same flag the damaging verb carries, because an export is a
	// property of a run rather than of one way of starting one.
	manifest *string
	given    map[string]bool
}

func declareGenerate(set *flag.FlagSet) *generateFlags {
	return &generateFlags{
		out:       set.String("out", "", ""),
		seed:      set.String("seed", "", ""),
		format:    set.String("format", "", ""),
		overwrite: set.Bool("overwrite", false, ""),
		dryRun:    set.Bool("dry-run", false, ""),
		manifest:  set.String("manifest", "", ""),
		given:     map[string]bool{},
	}
}

// note records which flags the parser actually saw.
func (f *generateFlags) note(set *flag.FlagSet) {
	set.Visit(func(seen *flag.Flag) { f.given[seen.Name] = true })
}

// runGenerate turns a recipe into files.
//
// This is the only place the pieces are wired together. The use case is handed
// a source of values and a writer as ports, the exit code comes from the kind
// the failure carried rather than from a decision made here, and nothing is
// visible under its real name until the whole run has succeeded.
func runGenerate(out *streams, arguments []string, global crossCutting) int {
	set := flag.NewFlagSet(commandGenerate, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	local := declareCrossCutting(set)
	own := declareGenerate(set)
	askedForHelp := declareVerbHelp(set)
	if err := set.Parse(arguments); err != nil {
		return usage(out, flagFailure(err, acceptedFlagsOf(commandGenerate), problemHelpSubjectDo+" "+commandGenerate))
	}
	if askedForHelp() {
		return printCommandHelp(out, commandGenerate)
	}
	own.note(set)
	flags := global.merge(local)
	if code, ok := applyVerbosity(out, flags); !ok {
		return code
	}

	rest := set.Args()
	if len(rest) != 1 {
		return usage(out, problem{
			what:    problemGenerateArguments,
			why:     problemGenerateArgumentsWhy,
			allowed: problemGenerateArgumentsAllowed,
			instead: problemGenerateArgumentsDo,
		})
	}
	// Asked before the recipe is read, because a form nobody writes is a typo
	// and a typo must not cost a pass over somebody's file.
	if !manifestexport.Usable(*own.manifest) {
		return usage(out, problem{
			what:    problemCorruptManifest + *own.manifest,
			why:     problemCorruptManifestWhy,
			allowed: strings.Join(manifestexport.Forms(), ", "),
			instead: problemGenerateManifestDo,
		})
	}
	return generateFrom(out, rest[0], own, flags)
}

// generateFrom reads the recipe and produces the dataset it describes.
func generateFrom(out *streams, path string, own *generateFlags, flags crossCutting) int {
	// Before the recipe is read, because a refusal about the command somebody
	// typed should not wait behind the parsing of a file. The seed inside the
	// recipe is refused by the reader instead, so that validate reports it
	// among all the other problems at once rather than one at a time.
	if own.given["seed"] {
		if refusal, tooLong := seedRefused(*own.seed); tooLong {
			return usage(out, refusal)
		}
	}
	built, code, ok := recipeAt(out, path)
	if !ok {
		return code
	}
	applied := overrides(&built, own)
	if err := usableFormat(built, applied); err != nil {
		return fail(out, err)
	}

	prepared, err := produce.Prepare(built, values.New(built.Locale, built.Seed))
	if err != nil {
		return fail(out, err)
	}
	plan := prepared.Plan()
	settings := produce.Settings{
		Into: outputDir(built), Replace: *own.overwrite, Manifest: *own.manifest,
		Overrides: applied,
	}

	if *own.dryRun {
		writePlan(out, plan, applied, flags, true)
		for _, line := range plannedDamage(built) {
			out.note(line)
		}
		for _, line := range unusedForm(built, settings) {
			out.note(line)
		}
		return finish(out, codeOK)
	}

	// The watch starts here rather than at the top of the run, because up to
	// this line nothing has been written and there is nothing to take back.
	watcher := watchSignals()
	defer watcher.release()
	written, err := prepared.Write(settings, watcher, time.Now())
	if err != nil {
		return fail(out, err)
	}
	writePlan(out, plan, applied, flags, false)
	// What the damage did goes after the counts of what was produced, because
	// it is a statement about the same files and reads as one continuous answer.
	for _, line := range damageTold(built, written, settings) {
		out.note(line)
	}
	return finish(out, codeOK)
}

// outputDir is where a run of this recipe writes, which is the working
// directory when nothing says otherwise.
func outputDir(built recipe.Recipe) string {
	if built.Output.Dir == "" {
		return "."
	}
	return built.Output.Dir
}

// damageTold is what the log says about the damage a finished run did, in the
// order a person reads it: what was ordered and done, which manifest will not
// be read back, and an extra form that had nothing to export.
//
// The facts come from the run and only the words are made here, because the
// window says the same facts in its own language (DEC-252).
func damageTold(built recipe.Recipe, written produce.Written, settings produce.Settings) []string {
	var told []string
	if written.Damaged {
		told = summaryOf(written.Counts, settings.Manifest)
	}
	for _, big := range written.Oversized {
		told = append(told, tooLargeSentence(big.Name, big.Bytes))
	}
	return append(told, unusedForm(built, settings)...)
}

// unusedForm says that an extra form of the manifest was asked for by a run
// that writes no manifest.
//
// 🔴 The flag used to be ignored in silence: the run ended with nought, wrote no
// manifest and said nothing about the flag (B-301). It is a correct setting
// with nothing to act on, so it is said rather than refused, and the sentence
// names what would give it something to act on.
func unusedForm(built recipe.Recipe, settings produce.Settings) []string {
	if !settings.FormUnused(built) {
		return nil
	}
	return []string{fmt.Sprintf(noteFormUnused, settings.Manifest)}
}

// recipeAt reads a recipe and refuses one that cannot be used.
//
// A recipe with problems is a configuration failure here rather than a report,
// because this verb was asked to produce data and cannot. The problems are
// still listed, so that the answer is the same one validate would have given.
func recipeAt(out *streams, path string) (recipe.Recipe, int, bool) {
	data, err := readRecipe(path)
	if err != nil {
		if errors.Is(err, errIsDirectory) {
			out.report(problem{
				what:    problemRecipeIsDirectory + path,
				why:     problemRecipeIsDirectoryWhy,
				allowed: problemRecipeIsDirectoryAllowed,
				instead: problemRecipeIsDirectoryDo,
			})
			return recipe.Recipe{}, int(codeIO), false
		}
		return recipe.Recipe{}, fail(out, fault.Wrap(fault.IO, err, "cannot read "+path)), false
	}

	decoded := recipeyaml.Decode(data)
	if !decoded.Valid() {
		for _, found := range decoded.Problems {
			what := found.What
			if found.Where != "" {
				what = found.Where + ": " + what
			}
			out.report(problem{what: what, allowed: found.Allowed, instead: found.Hint})
		}
		return recipe.Recipe{}, fail(out, fault.New(fault.Config,
			counted(len(decoded.Problems), "problem")+" in "+path)), false
	}
	return decoded.Recipe, 0, true
}

// settingFormat is the name the record of the run gives the output format, in
// one place because the refusal for an unwritable form reads it to say whether
// the recipe or the command line asked.
const settingFormat = "output.format"

// overrides applies the flags that carry a value and records what they moved.
//
// The record goes to the run, which writes it into the note beside the data,
// because two runs of one recipe giving different data with nothing saying why
// is the surprise that note exists to remove.
func overrides(built *recipe.Recipe, own *generateFlags) []produce.Override {
	var applied []produce.Override
	if own.given["seed"] {
		applied = append(applied, moved("seed", built.Seed, *own.seed))
		built.Seed = *own.seed
	}
	if own.given["out"] {
		// Through the run rather than here, because the window moves the same
		// setting with its own field and the note has to say it the same way.
		applied = append(applied, produce.MoveDir(built, *own.out))
	}
	if own.given["format"] {
		applied = append(applied, moved(settingFormat, built.Output.Format, *own.format))
		built.Output.Format = *own.format
	}
	return applied
}

// moved is one override, named field by field so that the calls above read as
// the setting and its two sides.
func moved(setting, fromRecipe, fromFlag string) produce.Override {
	return produce.Override{Setting: setting, FromRecipe: fromRecipe, FromFlag: fromFlag}
}

// usableFormat refuses a form this build cannot write, before anything is
// planned, and names whoever asked for it.
//
// # Why this is here and not only where the writer is made
//
// 🔴 The reader of recipes refuses an unknown format, so the refusal beside the
// writer was reached only through the flag, which arrives after the reader has
// finished. Two things followed from that and both were measured on 2026-08-26.
// A dry run returns before the writer is ever made, so generate --dry-run
// --format parquet printed how many rows it would write and ended with nought,
// while the same command without --dry-run ended with three. And the refusal
// said the recipe had asked for a word that came from the command line, which
// sends somebody to edit a file that is correct.
func usableFormat(built recipe.Recipe, applied []produce.Override) error {
	switch built.Output.Format {
	case "", formatCSV, formatJSON:
		return nil
	}
	asked := "the recipe asks for "
	for _, moved := range applied {
		if moved.Setting == settingFormat {
			asked = "the command line asks for "
		}
	}
	return fault.New(fault.Config, "this build writes "+formatCSV+" and "+formatJSON+
		", and "+asked+built.Output.Format)
}

// counted writes a number with its word, and the word agrees with the number.
//
// A line reading one columns is a small thing that tells a reader nobody looked
// at the output, and this text is the first thing anybody sees of the tool.
func counted(number int, word string) string {
	if number == 1 {
		return "1 " + word
	}
	return strconv.Itoa(number) + " " + word + "s"
}

// writePlan says what was produced, or what would be.
//
// The counts go on the log channel and not on the data channel, because they
// describe the dataset rather than being part of it. A pipeline that wants them
// as a document asks with the machine readable flag.
//
// trial says the run wrote nothing, and it changes the last word of the summary
// rather than the counts. A trial run saying rows written is a sentence that
// sends somebody looking for files nobody wrote.
func writePlan(out *streams, plan generate.Plan, applied []produce.Override, flags crossCutting, trial bool) {
	if *flags.json {
		writeJSONPlan(out, plan, applied)
		return
	}
	for _, skipped := range plan.Notes {
		out.warn(skipped.Where + ": " + skipped.What)
	}
	for _, moved := range applied {
		out.detail(moved.Setting + " came from the command line as " + moved.FromFlag +
			" rather than from the recipe as " + moved.FromRecipe)
	}
	for _, planned := range plan.Entities {
		out.detail(planned.Entity + ": " + counted(planned.Rows, "row") + ", " +
			counted(len(planned.Columns), "column"))
	}
	ending := summaryWritten
	if trial {
		ending = summaryPlanned
	}
	out.note(counted(len(plan.Entities), "table") + ", " +
		counted(plan.TotalRows(), "row") + ending)
}

// jsonPlan is the plan as a document, which is the part of the note beside the
// data that a pipeline wants before a run rather than after it. It is built
// from the same pieces as the note, so the two cannot describe one plan in two
// shapes.
type jsonPlan struct {
	Seed      string              `json:"seed"`
	Locale    string              `json:"locale"`
	Tables    []produce.NoteTable `json:"tables"`
	NotDone   []produce.NoteEntry `json:"not_done"`
	Overrides []produce.Override  `json:"overrides"`
}

func writeJSONPlan(out *streams, plan generate.Plan, applied []produce.Override) {
	document := jsonPlan{
		Seed: plan.Seed, Locale: plan.Locale,
		Tables:    produce.Tables(plan),
		NotDone:   produce.NotDone(plan),
		Overrides: produce.Overridden(applied),
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		out.dataErr = err
		return
	}
	out.emit(string(encoded))
}
