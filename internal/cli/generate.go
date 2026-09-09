package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
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

	source := values.New(built.Locale, built.Seed)
	plan, err := generate.Prepare(built, source)
	if err != nil {
		return fail(out, err)
	}
	noteUnreadSettings(built, &plan)

	if *own.dryRun {
		writePlan(out, plan, applied, flags, true)
		for _, line := range plannedDamage(built) {
			out.note(line)
		}
		return finish(out, codeOK)
	}

	// The watch starts here rather than at the top of the run, because up to
	// this line nothing has been written and there is nothing to take back.
	watcher := watchSignals()
	defer watcher.release()
	told, err := produce(built, plan, source, own, applied, watcher)
	if err != nil {
		return fail(out, err)
	}
	writePlan(out, plan, applied, flags, false)
	// What the damage did goes after the counts of what was produced, because
	// it is a statement about the same files and reads as one continuous answer.
	for _, line := range told {
		out.note(line)
	}
	return finish(out, codeOK)
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

// override is one setting the command line moved.
//
// It carries both sides rather than only the winner, because two runs of one
// recipe giving different data with nothing saying why is the surprise this
// record exists to remove.
type override struct {
	Setting    string `json:"setting"`
	FromRecipe string `json:"from_recipe"`
	FromFlag   string `json:"from_flag"`
}

// overrides applies the flags that carry a value and records what they moved.
func overrides(built *recipe.Recipe, own *generateFlags) []override {
	var applied []override
	if own.given["seed"] {
		applied = append(applied, override{"seed", built.Seed, *own.seed})
		built.Seed = *own.seed
	}
	if own.given["out"] {
		applied = append(applied, override{"output.dir", built.Output.Dir, *own.out})
		built.Output.Dir = *own.out
	}
	if own.given["format"] {
		applied = append(applied, override{settingFormat, built.Output.Format, *own.format})
		built.Output.Format = *own.format
	}
	return applied
}

// produce writes the dataset and the note beside it, or leaves nothing behind.
//
// The halt comes in as a parameter rather than being started here, so that this
// path can be exercised with a stop somebody triggers on purpose. A run stopped
// part way is one of the two endings this function has, and an ending that only
// a real signal can reach is an ending nothing checks.
func produce(built recipe.Recipe, plan generate.Plan, source values.Source,
	own *generateFlags, applied []override, stop generate.Halt) ([]string, error) {
	dir := built.Output.Dir
	if dir == "" {
		dir = "."
	}
	files := store.New(dir, *own.overwrite)
	// Nothing this run made survives a failure. The rename that gives a file its
	// real name happens once, at the end, so a run that stops here leaves no
	// file that looks complete.
	committed := false
	defer func() {
		if !committed {
			files.Discard()
		}
	}()

	writer, err := format.For(fileMaker{files}, built, stop)
	if err != nil {
		return nil, err
	}
	// A recipe with a policy has its damage done on the way to the disk, and
	// the manifest joins this same set of files. The first pillar is untouched
	// by this: a recipe without a policy gets the writer it always got.
	writer, broken, err := withDamage(built, source, writer, files, *own.manifest)
	if err != nil {
		return nil, err
	}
	if err := generate.Run(plan, built, source, writer, stop); err != nil {
		return nil, err
	}
	var told []string
	if broken != nil {
		if err := closeRecipeManifest(broken, built, plan, time.Now()); err != nil {
			return nil, err
		}
		told = summaryOf(broken.Counts(), *own.manifest)
	}
	if err := writeRunNote(files, built, plan, applied); err != nil {
		return nil, err
	}
	// Nothing is asked about a signal from here on. Everything is written and
	// what remains is giving the files their real names, and a rename stopped
	// half way through the set would leave exactly the half finished dataset
	// this whole arrangement exists to prevent.
	if err := files.Commit(); err != nil {
		return nil, err
	}
	committed = true
	if broken != nil {
		told = append(told, tooLargeToReadBack(dir, *own.manifest)...)
	}
	return told, nil
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
func usableFormat(built recipe.Recipe, applied []override) error {
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

// noteUnreadSettings records the settings this run carries and does not read.
//
// 🔴 file_per_entity decides the shape of the object form and means nothing to
// the comma separated one, which writes a file per table whatever it says. The
// setting did nothing and said nothing about doing nothing, in the format a
// recipe gets when it asks for none, which is the silence this product exists
// to find. The plan is the right place for it: the same sentence reaches the
// log, the plan on the data channel and the record left beside the data.
func noteUnreadSettings(built recipe.Recipe, plan *generate.Plan) {
	if built.Output.FilePerEntity {
		return
	}
	if built.Output.Format != "" && built.Output.Format != formatCSV {
		return
	}
	plan.Notes = append(plan.Notes, generate.Note{
		Where: "output.file_per_entity",
		What: "the " + formatCSV + " form writes one file per table whatever this says, so it " +
			"changes nothing here. The " + formatJSON + " form is the one that reads it",
	})
}

// fileMaker adapts the set of files to what a writer asks for. The two
// interfaces differ by one concrete type, and converting here keeps the writers
// from knowing what a disk is.
type fileMaker struct{ set *store.Set }

func (m fileMaker) Create(name string) (format.WriteCloser, error) {
	return m.set.Create(name)
}

// runNote is what travels beside the data.
//
// 🔴 It is not a manifest. The expectation manifest is the public contract of
// the second pillar and has its own specification. This says what a run made,
// what it did not do, and what the values in the files are, which is a smaller
// thing with a different reader.
type runNote struct {
	Tool          string      `json:"tool_version"`
	Catalog       string      `json:"catalog_version"`
	Manifest      string      `json:"manifest_version"`
	RecipeVersion string      `json:"recipe_schema_version"`
	Seed          string      `json:"seed"`
	Locale        string      `json:"locale"`
	Tag           string      `json:"tag,omitempty"`
	Tables        []noteTable `json:"tables"`
	NotDone       []noteEntry `json:"not_done"`
	Overrides     []override  `json:"overrides"`
	SyntheticData string      `json:"synthetic_data"`
}

type noteTable struct {
	Entity  string   `json:"entity"`
	Rows    int      `json:"rows"`
	Columns []string `json:"columns"`
	Parent  string   `json:"parent,omitempty"`
}

type noteEntry struct {
	Where string `json:"where"`
	What  string `json:"what"`
}

// writeRunNote puts the record of the run beside the data.
//
// It goes into a file rather than only onto the log, because a log stays in the
// pipeline and the data travels. Somebody holding a folder of records has to be
// able to learn from the folder itself that the values in it describe nobody.
func writeRunNote(files *store.Set, built recipe.Recipe, plan generate.Plan,
	applied []override) error {
	note := runNote{
		Tool: version.Tool, Catalog: version.Catalog,
		Manifest: version.Manifest, RecipeVersion: version.Recipe,
		Seed: built.Seed, Locale: built.Locale, Tag: built.Tag,
		Tables:        make([]noteTable, 0, len(plan.Entities)),
		NotDone:       make([]noteEntry, 0, len(plan.Notes)),
		Overrides:     applied,
		SyntheticData: license.SyntheticDataNotice,
	}
	if note.Overrides == nil {
		// A run with nothing overridden writes an empty list rather than a
		// null, because somebody reading these files reads them all the same
		// way and a type that changes with the flags is a trap for them.
		note.Overrides = make([]override, 0)
	}
	for _, planned := range plan.Entities {
		note.Tables = append(note.Tables, noteTable{
			Entity: planned.Entity, Rows: planned.Rows,
			Columns: planned.Columns, Parent: planned.Parent,
		})
	}
	for _, skipped := range plan.Notes {
		note.NotDone = append(note.NotDone, noteEntry{skipped.Where, skipped.What})
	}

	encoded, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		return fault.Wrap(fault.Runtime, err, "cannot write the run note")
	}
	file, err := files.Create(runNoteName)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the run note")
	}
	return nil
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
func writePlan(out *streams, plan generate.Plan, applied []override, flags crossCutting, trial bool) {
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

type jsonPlan struct {
	Seed      string      `json:"seed"`
	Locale    string      `json:"locale"`
	Tables    []noteTable `json:"tables"`
	NotDone   []noteEntry `json:"not_done"`
	Overrides []override  `json:"overrides"`
}

func writeJSONPlan(out *streams, plan generate.Plan, applied []override) {
	document := jsonPlan{
		Seed: plan.Seed, Locale: plan.Locale,
		Tables:    make([]noteTable, 0, len(plan.Entities)),
		NotDone:   make([]noteEntry, 0, len(plan.Notes)),
		Overrides: applied,
	}
	if document.Overrides == nil {
		document.Overrides = make([]override, 0)
	}
	for _, planned := range plan.Entities {
		document.Tables = append(document.Tables, noteTable{
			Entity: planned.Entity, Rows: planned.Rows,
			Columns: planned.Columns, Parent: planned.Parent,
		})
	}
	for _, skipped := range plan.Notes {
		document.NotDone = append(document.NotDone, noteEntry{skipped.Where, skipped.What})
	}

	encoded, err := json.Marshal(document)
	if err != nil {
		out.dataErr = err
		return
	}
	out.emit(string(encoded))
}
