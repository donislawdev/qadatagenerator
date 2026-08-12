package cli

import (
	"flag"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/corrupted"
	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/sourcecsv"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// The take and corrupt mode: somebody else's file goes in, the same file with
// damage in it and a manifest beside it come out.
//
// # What makes this different from the mode that generates
//
// There, five suppliers say what a run can do and a recipe declares what every
// column holds. Here nobody declared anything, so the shape is worked out by
// looking at the values, and the manifest carries the evidence for what was
// worked out. Everything downstream of that is the same engine, the same
// selector and the same writer, which is deliberate: damage that behaved
// differently depending on where the records came from would be a second
// product.

// takeFlags are the settings this verb takes beyond the cross cutting ones.
type takeFlags struct {
	out       *string
	seed      *string
	layers    *string
	rows      *string
	perRow    *string
	exclude   *string
	delimiter *string
	locale    *string
	overwrite *bool
	dryRun    *bool
	// manifest is the extra form the manifest is exported in, empty for none.
	//
	// 🔴 There is no default and that is a decision rather than an omission.
	// Adding a default later changes no call that exists today, because nobody
	// is passing the flag yet, while changing a default later changes every one
	// of them silently. The safer of the two therefore goes first.
	manifest *string
	given    map[string]bool
}

func declareCorrupt(set *flag.FlagSet) *takeFlags {
	return &takeFlags{
		out:       set.String("out", ".", ""),
		seed:      set.String("seed", defaultCorruptSeed, ""),
		layers:    set.String("layers", "", ""),
		rows:      set.String("rows", "", ""),
		perRow:    set.String("per-row", "", ""),
		exclude:   set.String("exclude-fields", "", ""),
		delimiter: set.String("delimiter", ",", ""),
		locale:    set.String("locale", defaultCorruptLocale, ""),
		overwrite: set.Bool("overwrite", false, ""),
		dryRun:    set.Bool("dry-run", false, ""),
		manifest:  set.String("manifest", "", ""),
		given:     map[string]bool{},
	}
}

// note records which flags the parser actually saw, which is how a flag that
// was left out is told from one given its own default.
func (f *takeFlags) note(set *flag.FlagSet) {
	set.Visit(func(seen *flag.Flag) { f.given[seen.Name] = true })
}

// runCorrupt damages a file somebody else made.
func runCorrupt(out *streams, arguments []string, global crossCutting) int {
	set := flag.NewFlagSet(commandCorrupt, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	local := declareCrossCutting(set)
	own := declareCorrupt(set)
	askedForHelp := declareVerbHelp(set)
	if err := set.Parse(arguments); err != nil {
		return usage(out, flagFailure(err, acceptedFlagsOf(commandCorrupt),
			problemHelpSubjectDo+" "+commandCorrupt))
	}
	if askedForHelp() {
		return printCommandHelp(out, commandCorrupt)
	}
	own.note(set)
	flags := global.merge(local)
	if code, ok := applyVerbosity(out, flags); !ok {
		return code
	}

	rest := set.Args()
	if len(rest) != 1 {
		return usage(out, problem{
			what:    problemCorruptArguments,
			why:     problemCorruptArgumentsWhy,
			allowed: problemCorruptArgumentsAllowed,
			instead: problemCorruptArgumentsDo,
		})
	}
	return corruptFile(out, rest[0], own)
}

// corruptFile is the whole path, from a file on a disk to a damaged file and a
// manifest beside it.
func corruptFile(out *streams, path string, own *takeFlags) int {
	described, err := catalogyaml.Load()
	if err != nil {
		return fail(out, err)
	}
	if refusal, wrong := unusablePolicyFlags(own, described); wrong {
		return usage(out, refusal)
	}
	written, refusal, ok := outputPath(path, own)
	if !ok {
		return usage(out, refusal)
	}

	source, err := sourcecsv.Open(path, []rune(*own.delimiter)[0])
	if err != nil {
		return fail(out, err)
	}
	defer func() { _ = source.Close() }()

	from := values.New(*own.locale, *own.seed)
	shape := infer.From(source.Table(), source.Samples(), from.Schemes())
	sayWhatWasWorkedOut(out, source, shape)
	if *own.dryRun {
		sayWhatWouldHappen(out, source, own, described)
		return finish(out, codeOK)
	}
	return writeDamaged(out, damageRequest{
		source: source, shape: shape, path: path, written: written,
		own: own, described: described, from: from,
	})
}

// damageRequest is everything the writing half needs, gathered so that the
// function taking it stays inside the limit on how many things one function may
// be handed.
type damageRequest struct {
	source    *sourcecsv.File
	shape     infer.Schema
	path      string
	written   string
	own       *takeFlags
	described catalog.Catalog
	from      values.Source
}

// writeDamaged runs the engine over the file and puts everything on the disk,
// or leaves the directory exactly as it found it.
func writeDamaged(out *streams, ask damageRequest) int {
	files := store.New(*ask.own.out, *ask.own.overwrite)
	committed := false
	defer func() {
		if !committed {
			files.Discard()
		}
	}()

	// The watch starts here rather than at the top, because up to this line
	// nothing has been written and there is nothing to take back.
	watcher := watchSignals()
	defer watcher.release()

	broken, err := buildDamage(ask, files)
	if err != nil {
		return fail(out, err)
	}
	if err := corrupt.Run(ask.source, broken.told.writer, watcher); err != nil {
		return fail(out, err)
	}
	if err := broken.closeManifest(ask, time.Now()); err != nil {
		return fail(out, err)
	}
	if err := files.Commit(); err != nil {
		return fail(out, err)
	}
	committed = true

	out.note(listOf(append([]string{ask.written}, manifestFiles(*ask.own.manifest)...)) +
		" written to " + *ask.own.out)
	for _, line := range broken.told.summary() {
		out.note(line)
	}
	return finish(out, codeOK)
}

// takenDamage is what a damaging run holds: the pieces every run of the engine
// has, and the guesses that only this mode makes.
//
// The first part is the same value the mode that generates uses, so the summary
// a person reads is written once and says the same thing whatever produced the
// records.
type takenDamage struct {
	told  damage
	shape infer.Schema
}

// buildDamage assembles the engine, the selector and the two files.
func buildDamage(ask damageRequest, files *store.Set) (*takenDamage, error) {
	selector, err := corrupt.NewSelector(policyOf(ask.own), ask.described, ask.shape,
		seed.New(*ask.own.seed))
	if err != nil {
		return nil, err
	}
	writer, err := format.NewPlainCSV(fileMaker{files}, ask.written, *ask.own.delimiter)
	if err != nil {
		return nil, err
	}
	file, err := files.Create(manifestjson.Name)
	if err != nil {
		return nil, err
	}
	entries := manifestjson.New(file, version.Manifest)
	// The export is a peer of the canonical manifest rather than a rendering of
	// it afterwards, so it is opened here and fed the same entries at the same
	// moment. Nothing reads what this run just wrote.
	sinks := corrupted.Tee{entries}
	var exported *manifestexport.Gherkin
	if *ask.own.manifest == manifestexport.FormGherkin {
		file, err := files.Create(manifestexport.GherkinName)
		if err != nil {
			return nil, err
		}
		exported = manifestexport.NewGherkin(file, ask.described, ask.source.Table().Rows)
		sinks = append(sinks, exported)
	}
	engine := corrupt.New(ask.described, mutatorRegistry{}, schemeRegistry{ask.from},
		canDoWithoutRecipe(ask.shape, ask.from))
	wrapped := corrupted.New(writer, engine, &selector, sinks, ask.shape)
	return &takenDamage{
		told: damage{writer: wrapped, entries: entries, selector: &selector,
			exported: exported, form: *ask.own.manifest},
		shape: ask.shape,
	}, nil
}

// canDoWithoutRecipe gathers what this run is able to do when nothing declared
// anything.
//
// Four of the five suppliers answer as they always do. The one that changes is
// the schema, which here is what was worked out by looking rather than what
// somebody wrote down, and it therefore brings much less: no maximum length, no
// required column, no key and no relation. That is the point rather than a
// shortfall, and every entry it leaves inapplicable says so in the manifest.
func canDoWithoutRecipe(shape infer.Schema, source values.Source) capability.Set {
	names := format.CapabilitiesOf(format.NameCSV)
	names = append(names, recipe.CapabilitiesOf(shape)...)
	names = append(names, values.CapabilitiesOfView(shape, source)...)
	names = append(names, capability.UnicodeNormalization)
	return capability.NewSet(names...)
}

// closeManifest writes the envelope, including the evidence for everything this
// run had to guess.
//
// The counts come from what the run did rather than from what it meant to do,
// and the whole envelope is held against the model before a byte of it is
// written: a manifest the domain would refuse must not reach a disk, because
// whoever reads it will read it as a promise.
func (d *takenDamage) closeManifest(ask damageRequest, at time.Time) error {
	run := manifest.Run{
		GeneratedAt:    at.UTC().Format(time.RFC3339),
		Seed:           *ask.own.seed,
		Locale:         *ask.own.locale,
		CatalogVersion: version.Catalog,
		Source:         ask.path,
		Inference:      inferenceRecord(d.shape),
		RowsTotal:      ask.source.Table().Rows,
		RowsMutated:    d.told.writer.RowsMutated(),
	}
	if problems := run.Problems(); len(problems) > 0 {
		return fault.New(fault.Runtime, "the manifest envelope is not usable: "+problems[0])
	}
	if err := d.told.entries.Finish(run, d.told.writer.Requested(),
		d.told.writer.Applied()); err != nil {
		return err
	}
	// The canonical manifest goes first, so that a failure to close the export
	// is reported after the document everything else is compared against.
	// Either failure ends the run, and a run that ends leaves nothing behind,
	// so neither file is ever shipped half written.
	return d.told.closeExport()
}

// inferenceRecord turns what was worked out into what the manifest carries.
func inferenceRecord(shape infer.Schema) *manifest.Inference {
	record := &manifest.Inference{
		Method:      inferenceMethod,
		NotInferred: infer.NotInferred(),
	}
	for _, finding := range shape.Findings() {
		record.Columns = append(record.Columns, manifest.InferredColumn{
			Field: finding.Field, Type: finding.Type,
			Scheme: finding.Scheme, Basis: finding.Basis,
		})
	}
	return record
}

// policyOf turns the flags into the policy the engine takes.
//
// It builds the same value a recipe would have produced, so that the engine
// cannot tell where the policy came from. A second shape for the same thing
// would be a second set of rules about what a policy means.
func policyOf(own *takeFlags) recipe.Policy {
	return recipe.Policy{
		Layers:        namesIn(*own.layers),
		Rows:          *own.rows,
		PerRow:        *own.perRow,
		ExcludeFields: namesIn(*own.exclude),
	}
}

// namesIn splits a comma separated list, dropping what is only whitespace.
func namesIn(written string) []string {
	var names []string
	for _, part := range strings.Split(written, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

// unusablePolicyFlags refuses an invocation that cannot order damage, and says
// what would have.
//
// # Why the layers have no default
//
// Choosing them would be deciding what "damage this file" means on somebody
// else's behalf, and the cheapest default to defend, everything, would take the
// bytes and the container apart on a file whose owner wanted to test an import.
// Giving this a default later breaks nothing, because every invocation without
// it is refused today. Changing a default later breaks every pipeline that
// relied on it, so the direction that keeps the door open is the one taken.
func unusablePolicyFlags(own *takeFlags, described catalog.Catalog) (problem, bool) {
	if refusal, unusable := unusableLayers(own, described); unusable {
		return refusal, true
	}
	return unusableFlagValues(own)
}

// unusableLayers refuses a run that named no layer, or named one this build
// carries no damage in.
//
// It is separate from the values below because it asks a different kind of
// question. A layer is answered against the catalog, and the answer changes as
// the catalog grows. The values below are answered against this build's own
// vocabulary and do not.
func unusableLayers(own *takeFlags, described catalog.Catalog) (problem, bool) {
	// The same answer validate gives about a recipe, from the same place. Two
	// lists of which layers carry damage would tell somebody their policy is
	// fine while the verb that runs it did nothing.
	carried := mutators.Coverage{Described: described}.Layers()
	named := namesIn(*own.layers)
	if len(named) == 0 {
		return problem{
			what:    problemCorruptNoLayers,
			why:     problemCorruptNoLayersWhy,
			allowed: strings.Join(carried, ", "),
			instead: problemCorruptNoLayersDo,
		}, true
	}
	carries := map[string]bool{}
	for _, layer := range carried {
		carries[layer] = true
	}
	for _, layer := range named {
		if !carries[layer] {
			return problem{
				what:    problemCorruptUnknownLayer + layer,
				why:     problemCorruptUnknownLayerWhy,
				allowed: strings.Join(carried, ", "),
				instead: problemCorruptNoLayersDo,
			}, true
		}
	}
	return problem{}, false
}

// unusableFlagValues refuses a value this build cannot read, one flag at a
// time.
//
// 🔴 Every one of these is asked here rather than where the value is used, and
// the reason is the same for all four: the file is opened after this, and a
// pass over a file the user chose the size of must not be the price of a typo.
func unusableFlagValues(own *takeFlags) (problem, bool) {
	if len([]rune(*own.delimiter)) != 1 {
		return problem{
			what:    problemCorruptDelimiter + *own.delimiter,
			why:     problemCorruptDelimiterWhy,
			allowed: problemCorruptDelimiterAllowed,
			instead: problemCorruptDelimiterDo,
		}, true
	}
	if !manifestexport.Usable(*own.manifest) {
		return problem{
			what:    problemCorruptManifest + *own.manifest,
			why:     problemCorruptManifestWhy,
			allowed: strings.Join(manifestexport.Forms(), ", "),
			instead: problemCorruptManifestDo,
		}, true
	}
	if err := corrupt.UsableShare(*own.rows); err != nil {
		return problem{
			what:    problemCorruptRows + *own.rows,
			why:     problemCorruptRowsWhy,
			allowed: problemCorruptRowsAllowed,
			instead: problemCorruptRowsDo,
		}, true
	}
	if err := corrupt.UsablePerRow(*own.perRow); err != nil {
		return problem{
			what:    problemCorruptPerRow + *own.perRow,
			why:     problemCorruptPerRowWhy,
			allowed: strings.Join(corrupt.PerRowChoices(), ", "),
			instead: problemCorruptPerRowDo,
		}, true
	}
	return problem{}, false
}

// outputPath works out what the damaged file is called, and refuses a run that
// would write over the file it is reading.
//
// # Why both the suffix and the check
//
// The suffix makes the collision impossible by construction, so the check can
// never fire while the name is built this way. It stays because it guards
// something that cannot be taken back: this is the only verb that reads and
// writes at once, and the day somebody changes how the name is built, a
// redundant check is what stands between that change and a person losing the
// file they handed over. It has a mutation of its own for the same reason.
func outputPath(source string, own *takeFlags) (string, problem, bool) {
	name := sourcecsv.EntityOf(source) + corruptedSuffix
	written := filepath.Join(*own.out, name)
	if sameFile(written, source) {
		return "", problem{
			what:    problemCorruptWouldOverwrite + source,
			why:     problemCorruptWouldOverwriteWhy,
			allowed: problemCorruptWouldOverwriteAllowed,
			instead: problemCorruptWouldOverwriteDo,
		}, false
	}
	return name, problem{}, true
}

// sameFile reports whether two paths name one place, as far as comparing text
// can tell.
//
// It compares cleaned absolute paths. That answers the case this guards, which
// is a directory and a name that meet, and it does not answer links or two
// mounts of one volume. Saying so here is cheaper than a reader assuming it
// covers more: what covers the rest is the suffix above, which keeps the two
// names apart whatever the file system thinks.
func sameFile(written, source string) bool {
	left, err := filepath.Abs(written)
	if err != nil {
		return false
	}
	right, err := filepath.Abs(source)
	if err != nil {
		return false
	}
	return filepath.Clean(left) == filepath.Clean(right)
}

// sayWhatWasWorkedOut puts the guesses on the log channel, where somebody
// running this interactively sees them without opening the manifest.
//
// The manifest is where they live for good. This is the same information for
// the person watching, and it is deliberately not shortened: a column nothing
// was recognised in is exactly the one they need to know about, because that is
// where the damage they ordered will have had nothing to work with.
func sayWhatWasWorkedOut(out *streams, source *sourcecsv.File, shape infer.Schema) {
	table := source.Table()
	out.note(table.Entity + ": " + counted(table.Rows, "row") + ", " +
		counted(len(table.Columns), "column") + ", nothing declared what any of them holds")
	for _, finding := range shape.Findings() {
		out.detail(finding.Field + " taken for " + describeFinding(finding) + ", because " +
			finding.Basis)
	}
	if source.HadByteOrderMark() {
		out.warn(noteByteOrderMark)
	}
	out.detail(noteRewritten)
	out.note(noteNotInferred)
}

// describeFinding names what a column was taken to hold, in words.
func describeFinding(finding infer.Finding) string {
	if finding.Scheme != "" {
		return "a column of " + finding.Scheme + " values"
	}
	return "a column of " + finding.Type + " values"
}

// sayWhatWouldHappen is the dry run: it says what the policy asks for and
// writes nothing.
//
// It reports what is certain rather than estimating. How many rows a share
// comes to is a draw, and a number here that the real run then missed would be
// worse than no number.
func sayWhatWouldHappen(out *streams, source *sourcecsv.File, own *takeFlags,
	described catalog.Catalog) {
	coverage := mutators.Coverage{Described: described}
	ordered, implemented := 0, 0
	for _, layer := range namesIn(*own.layers) {
		inLayer := len(described.InLayer(catalog.Layer(layer)))
		ordered += inLayer
		implemented += inLayer - coverage.Unimplemented(layer)
	}
	out.note(noteDryRunPolicy + kindsOfDamage(ordered) + " in " + *own.layers +
		", and this build implements " + counted(implemented, "kind") + " of them. " +
		noteDryRunEveryOrder)
	out.note(noteDryRunWrites + listOf(append(
		[]string{source.Table().Entity + corruptedSuffix}, manifestFiles(*own.manifest)...)) +
		" in " + *own.out)
}
