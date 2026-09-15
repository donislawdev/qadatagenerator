package cli

import (
	"flag"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/infer"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// The take and corrupt mode as the command line offers it.
//
// # What is here and what moved out
//
// What is here is the flags, the four part refusals and the sentences. What the
// mode actually does moved to internal/take on 2026-08-19, when the window had
// to do the same work: the layer test forbids a screen to import this package,
// so an assembly living here could only be reached by copying it, and a second
// copy of the rules about damaging somebody else's file is the thing this
// project can least afford to have drift.
//
// The division is not a matter of taste. Everything both surfaces have to agree
// on is over there, and everything that is a property of speaking to a terminal
// is here.

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
	defaults := take.Defaults()
	return &takeFlags{
		out:       set.String("out", defaults.Out, ""),
		seed:      set.String("seed", defaults.Seed, ""),
		layers:    set.String("layers", "", ""),
		rows:      set.String("rows", "", ""),
		perRow:    set.String("per-row", "", ""),
		exclude:   set.String("exclude-fields", "", ""),
		delimiter: set.String("delimiter", defaults.Delimiter, ""),
		locale:    set.String("locale", defaults.Locale, ""),
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

// settings turns the flags into what the mode takes.
func (f *takeFlags) settings() take.Settings {
	return take.Settings{
		Out: *f.out, Seed: *f.seed, Layers: take.NamesIn(*f.layers), Rows: *f.rows,
		PerRow: *f.perRow, ExcludeFields: take.NamesIn(*f.exclude),
		Delimiter: *f.delimiter, Locale: *f.locale, Overwrite: *f.overwrite,
		Manifest: *f.manifest,
	}
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
	settings := own.settings()
	// Everything answerable without the file is answered first. The file is
	// somebody else's and they chose its size, so a pass over it must not be
	// the price of a typo.
	if refusal := take.Check(settings, described); refusal.Refused() {
		return usage(out, refusalOf(refusal))
	}
	if refusal := take.CheckOutput(path, settings); refusal.Refused() {
		return usage(out, refusalOf(refusal))
	}

	opened, err := take.Open(path, settings)
	if err != nil {
		return fail(out, err)
	}
	defer func() { _ = opened.Close() }()

	sayWhatWasWorkedOut(out, opened, *own.dryRun)
	if *own.dryRun {
		sayWhatWouldHappen(out, opened, own, described)
		return finish(out, codeOK)
	}
	return writeDamaged(out, opened, own, described)
}

// writeDamaged runs the engine over the file and says what was left behind.
func writeDamaged(out *streams, opened *take.Opened, own *takeFlags,
	described catalog.Catalog) int {
	// The watch starts here rather than at the top, because up to this line
	// nothing has been written and there is nothing to take back.
	watcher := watchSignals()
	defer watcher.release()

	result, err := opened.Corrupt(own.settings(), described, watcher, time.Now())
	if err != nil {
		return fail(out, err)
	}
	out.note(listOf(result.Written) + " written to " + result.Out)
	for _, line := range summaryOf(result.Counts, *own.manifest) {
		out.note(line)
	}
	// This verb is the one that damages somebody else's file, so it is the one
	// that reaches a manifest too large to read back first. The run says which
	// files those are rather than this surface measuring them again, because the
	// window shows the same thing and neither can import the other.
	for _, big := range result.Oversized {
		out.note(tooLargeSentence(big.Name, big.Bytes))
	}
	return finish(out, codeOK)
}

// refusalWords is what this surface says about one refusal of the mode.
//
// The first part is a prefix rather than a whole sentence, because most of these
// refusals name the value that was wrong and the value arrives with the refusal.
type refusalWords struct {
	// what is the opening, with the offending value appended to it.
	what string
	// why is the reason, which never varies.
	why string
	// allowed is what would have been taken, for the refusals whose answer is a
	// sentence. It is empty for the ones whose answer is a list, and the list
	// then comes from the refusal itself, because only the catalog knows it.
	allowed string
	// instead is what to do about it.
	instead string
}

// refusalSays is every refusal the mode can produce, in the words this surface
// uses for it.
//
// # Why a register and not the switch this used to be
//
// A switch has a last branch, and a kind added to the mode and forgotten here
// would land in it. What came out was not silence, which somebody would notice.
// It was a complete, confident, four part message about a different refusal
// entirely: somebody whose delimiter was wrong being told their run would write
// over the file it reads. No compiler reports that and no reader of the switch
// sees it.
//
// A register can be held against the list of kinds in both directions, which is
// what the guard beside it does. A kind with no words fails, and words for a
// kind that no longer exists fail too.
var refusalSays = map[take.RefusalKind]refusalWords{
	take.NoLayers: {
		what: problemCorruptNoLayers, why: problemCorruptNoLayersWhy,
		instead: problemCorruptNoLayersDo,
	},
	take.UnknownLayer: {
		what: problemCorruptUnknownLayer, why: problemCorruptUnknownLayerWhy,
		instead: problemCorruptNoLayersDo,
	},
	take.UnreadableDelimiter: {
		what: problemCorruptDelimiter, why: problemCorruptDelimiterWhy,
		allowed: problemCorruptDelimiterAllowed, instead: problemCorruptDelimiterDo,
	},
	take.UnknownManifestForm: {
		what: problemCorruptManifest, why: problemCorruptManifestWhy,
		instead: problemCorruptManifestDo,
	},
	take.UnreadableShare: {
		what: problemCorruptRows, why: problemCorruptRowsWhy,
		allowed: problemCorruptRowsAllowed, instead: problemCorruptRowsDo,
	},
	take.UnknownPerRow: {
		what: problemCorruptPerRow, why: problemCorruptPerRowWhy,
		instead: problemCorruptPerRowDo,
	},
	take.WouldOverwriteSource: {
		what: problemCorruptWouldOverwrite, why: problemCorruptWouldOverwriteWhy,
		allowed: problemCorruptWouldOverwriteAllowed,
		instead: problemCorruptWouldOverwriteDo,
	},
}

// refusalOf turns a refusal of the mode into the four parts a person reads.
//
// # Why the sentences are here rather than beside the rule
//
// The rule is shared with the window and the sentences cannot be: this channel
// is English always, by an unbreakable rule, while the window shows whatever it
// was translated into. What travels between the two surfaces is therefore which
// requests are refused, and each surface says it in its own words.
func refusalOf(refusal take.Refusal) problem {
	words, known := refusalSays[refusal.Kind]
	if !known {
		// Unreachable while the guard beside this holds, and it says so out
		// loud rather than describing this as some other refusal. A message
		// naming what it does not know is recoverable. One confidently naming
		// the wrong cause is not.
		return problem{
			what:    problemCorruptUnnamedRefusal + refusal.Kind.Name(),
			why:     problemCorruptUnnamedRefusalWhy,
			allowed: strings.Join(refusal.Allowed, ", "),
			instead: problemCorruptUnnamedRefusalDo,
		}
	}
	allowed := words.allowed
	if allowed == "" {
		allowed = strings.Join(refusal.Allowed, ", ")
	}
	return problem{
		what: words.what + refusal.Value, why: words.why,
		allowed: allowed, instead: words.instead,
	}
}

// sayWhatWasWorkedOut puts the guesses on the log channel, where somebody
// running this interactively sees them without opening the manifest.
//
// The manifest is where they live for good, and it holds all of them however
// many there are. This is the same information for the person watching, up to
// the point where there is more of it than a person watching can read.
func sayWhatWasWorkedOut(out *streams, opened *take.Opened, dryRun bool) {
	table := opened.Table()
	out.note(table.Entity + ": " + counted(table.Rows, "row") + ", " +
		counted(len(table.Columns), "column") + ", nothing declared what any of them holds")
	shape := opened.Shape()
	findings := shape.Findings()
	for _, finding := range findings[:named(len(findings))] {
		out.detail(finding.Field + " taken for " + describeFinding(finding) + ", because " +
			finding.Basis)
	}
	sayHowManyMore(out, len(findings), "column", dryRun)
	// A relation is what damage contradicting a neighbouring column stands on,
	// so it is said here beside the columns rather than left to the manifest.
	// Somebody watching a run decide things about their file is the reader most
	// able to say that a relation is wrong.
	//
	// It is counted separately from the columns rather than sharing one budget
	// with them, because the two answer different questions and a file with many
	// of one would otherwise hide the other entirely.
	relations := shape.Relations()
	for _, relation := range relations[:named(len(relations))] {
		out.detail(relation.Field + " taken to agree with " + relation.With + " on " +
			string(relation.Kind) + ", because " + relation.Basis)
	}
	sayHowManyMore(out, len(relations), "relation", dryRun)
	if opened.HadByteOrderMark() {
		out.warn(noteByteOrderMark)
	}
	out.detail(noteRewritten)
	out.note(noteNotInferred)
}

// named is how many of a list get a line to themselves.
func named(all int) int {
	if all > take.ColumnsListed {
		return take.ColumnsListed
	}
	return all
}

// sayHowManyMore counts what was left out, and says nothing at all when nothing
// was.
//
// # Why a dry run is told something different
//
// It writes no manifest, so sending its reader to one would be the last line of
// a message contradicting the line above it. What that reader can be told is
// what to run to get the whole list, which is what somebody in the middle of a
// dry run is deciding anyway.
func sayHowManyMore(out *streams, all int, what string, dryRun bool) {
	if all <= take.ColumnsListed {
		return
	}
	where := noteRestInManifest
	if dryRun {
		where = noteRestInManifestOfARealRun
	}
	out.detail(counted(all-take.ColumnsListed, "more "+what) + noteRestWorkedOut + where)
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
func sayWhatWouldHappen(out *streams, opened *take.Opened, own *takeFlags,
	described catalog.Catalog) {
	planned := opened.Plan(own.settings(), described)
	out.note(noteDryRunPolicy + kindsOfDamage(planned.Ordered) + " in " + *own.layers +
		", and this build implements " + counted(planned.Implemented, "kind") + " of them. " +
		noteDryRunEveryOrder)
	// One line per layer, because the sums above say that something is missing
	// and not what. The split is the mode's, so the window says the same thing
	// in its own words (DEC-245).
	for _, layer := range planned.ByLayer {
		out.note(noteDryRunLayer + layer.Layer + ": " + counted(layer.Ordered, "kind") +
			", of which this build implements " + strconv.Itoa(layer.Implemented))
	}
	out.note(noteDryRunWrites + listOf(planned.Writes) + " in " + planned.Out)
}
