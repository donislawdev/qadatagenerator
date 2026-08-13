// Package cli is the command line adapter.
//
// It is the only layer that knows about flags, exit codes, terminals and the
// two channels. Everything below it returns a failure that carries its kind,
// and turning that kind into a number happens here and nowhere else.
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Environment is everything a run learns from outside itself.
//
// It is a struct rather than a set of package level lookups so that the whole
// command line can be exercised without starting a process, without a
// terminal and without touching the environment of the machine running the
// tests.
type Environment struct {
	// Args are the arguments after the program name.
	Args []string
	// Stdout carries data.
	Stdout io.Writer
	// Stderr carries the log.
	Stderr io.Writer
	// StderrIsTerminal decides whether escape sequences are allowed on the
	// log channel. There is no counterpart for the data channel, because
	// data never carries them: colored output stops being parseable, and
	// that is the whole reason the two channels are apart.
	StderrIsTerminal bool
	// NoColorRequested reports the NO_COLOR variable of the environment.
	//
	// It is the only variable this tool reads. Everything that affects the
	// data comes from the command and from the files the command names,
	// because a variable is invisible in a CI log and would let the same
	// commit produce different results for two people.
	NoColorRequested bool
}

// Run executes one invocation and returns the exit code.
//
// It never calls os.Exit. The caller owns the process, this owns the decision,
// and that split is what makes every path below testable.
func Run(env Environment) (code int) {
	// A crash must not leave through Go's own exit code, which is 2, the
	// same number that means a wrong invocation here. A pipeline would read
	// a defect of ours as a mistake of its own and never see the difference.
	defer func() {
		if recovered := recover(); recovered != nil {
			code = reportCrash(env, recovered)
		}
	}()
	return dispatch(env)
}

// invocation is what one command line asked for, after the flags were read.
type invocation struct {
	showVersion *bool
	showLicense *bool
	help        *bool
	helpShort   *bool
	global      crossCutting
}

func (i invocation) wantsHelp() bool { return *i.help || *i.helpShort }

// reportsAsked counts the reports requested at once. Two of them together are
// legal and both get printed, which is why this is a count and not a choice.
func (i invocation) reportsAsked() int {
	asked := 0
	for _, requested := range []bool{i.wantsHelp(), *i.showVersion, *i.showLicense} {
		if requested {
			asked++
		}
	}
	return asked
}

// declareFlags builds the flag set for the whole invocation.
func declareFlags() (*flag.FlagSet, invocation) {
	set := flag.NewFlagSet("qadg", flag.ContinueOnError)
	// The flag package prints its own message and its own usage text. Both
	// would break the four part shape of our failures and the split between
	// the channels, so we take the error and report it ourselves.
	set.SetOutput(io.Discard)

	return set, invocation{
		showVersion: set.Bool("version", false, ""),
		showLicense: set.Bool("license", false, ""),
		help:        set.Bool("help", false, ""),
		helpShort:   set.Bool("h", false, ""),
		global:      declareCrossCutting(set),
	}
}

// dispatch decides what one invocation does, once the crash net is in place.
func dispatch(env Environment) int {
	set, asked := declareFlags()
	parseErr := set.Parse(env.Args)

	out := &streams{
		data:  env.Stdout,
		log:   env.Stderr,
		color: env.StderrIsTerminal && !colorTurnedOff(env.Args) && !env.NoColorRequested,
		level: levelNormal,
	}

	if parseErr != nil {
		return reportParseFailure(out, parseErr)
	}
	if code, ok := applyVerbosity(out, asked.global); !ok {
		return code
	}

	// The flag package stops at the first argument that is not a flag, so
	// anything after a command belongs to that command and never to this set.
	if command := set.Args(); len(command) > 0 {
		return runCommand(out, command, asked)
	}
	return runReports(out, asked)
}

func reportParseFailure(out *streams, parseErr error) int {
	// Unreachable while both help flags are defined, and kept because the
	// flag package answers this way for a help flag that is not. Removing one
	// definition would otherwise turn a request for help into a failure.
	if errors.Is(parseErr, flag.ErrHelp) {
		printHelp(out)
		return finish(out, codeOK)
	}
	return usage(out, flagFailure(parseErr, acceptedFlags(), problemUnknownFlagDo))
}

// flagFailure turns a refusal from the flag package into our four parts.
//
// # Why the reason is chosen rather than fixed
//
// It used to be one reason for every parse failure, and that reason said the
// flag was not defined. For a flag given a value it was untrue, and untrue in
// the most confusing way available: the line naming what would be accepted
// listed the flag two rows further down, so the message contradicted itself and
// sent the reader looking for a typo in a name that was spelled correctly.
//
// The two classes are told apart by the wording the flag package uses, which is
// the only signal it offers. An unrecognised wording falls through to the
// original reason, which is the right answer for a flag nobody knows.
func flagFailure(err error, allowed, instead string) problem {
	if strings.Contains(err.Error(), "invalid boolean value") {
		return problem{
			what:    firstLine(err.Error()),
			why:     problemFlagTakesNoValueWhy,
			allowed: problemFlagTakesNoValueAllowed,
			instead: problemFlagTakesNoValueDo,
		}
	}
	return problem{
		what:    firstLine(err.Error()),
		why:     problemUnknownFlagWhy,
		allowed: allowed,
		instead: instead,
	}
}

// firstLine keeps a message from the flag package to one line, so that a
// failure stays four lines however the library chooses to word itself.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return strings.TrimSpace(line)
}

// runCommand hands the rest of the line to a verb.
//
// A command asked for together with a report would leave one of the two
// unanswered, so both are refused rather than one quietly dropped.
func runCommand(out *streams, command []string, asked invocation) int {
	if asked.reportsAsked() > 0 {
		return usage(out, problem{
			what:    problemCommandAndReport,
			why:     problemCommandAndReportWhy,
			allowed: problemCommandAndReportAllowed,
			instead: problemCommandAndReportDo,
		})
	}
	// Which verb is running decides the last part of a refusal, so it is
	// recorded before the verb can fail. Setting it inside each verb would
	// leave the one that forgot answering with the sentence of another.
	out.verb = command[0]
	switch command[0] {
	case commandValidate:
		return runValidate(out, command[1:], asked.global)
	case commandGenerate:
		return runGenerate(out, command[1:], asked.global)
	case commandCorrupt:
		return runCorrupt(out, command[1:], asked.global)
	case commandVerify:
		return runVerify(out, command[1:], asked.global)
	case commandHelp:
		return runHelp(out, command[1:])
	default:
		return usage(out, unknownCommand(command[0]))
	}
}

// unknownCommand builds the refusal for a verb this build does not have, and
// finishes the attempt when it can.
//
// The four parts stay four parts. What changes when the typed name is close to
// a real one is the reason and the suggestion, because a message that only
// says no leaves the reader to guess which of their letters was wrong.
func unknownCommand(typed string) problem {
	refusal := problem{
		what:    "unknown command " + typed,
		why:     problemUnknownCommandWhy,
		allowed: strings.Join(knownCommands(), ", "),
		instead: problemUnknownCommandDo,
	}
	if near, found := nearestCommand(typed); found {
		refusal.why = problemUnknownCommandNear + near + problemUnknownCommandNearTail
		refusal.instead = "qadg " + near
	}
	return refusal
}

// knownCommands are the verbs this build accepts, sorted.
//
// Sorted rather than in the order somebody happened to write them, because the
// list reaches the user and two runs must not disagree about it. Go randomises
// map iteration on purpose and this is the same rule applied to a slice.
func knownCommands() []string {
	commands := make([]string, 0, len(commandDocs))
	for _, doc := range commandDocs {
		commands = append(commands, doc.name)
	}
	sort.Strings(commands)
	return commands
}

// nearestCommand returns the known verb closest to what was typed, and whether
// it is close enough to be worth suggesting.
//
// The threshold is deliberately mean. Suggesting a command for anything vaguely
// similar turns a clear refusal into a misleading one, and somebody who typed a
// word we simply do not have is better served by the list of what exists.
func nearestCommand(typed string) (string, bool) {
	best, distance := "", -1
	for _, candidate := range knownCommands() {
		measured := editDistance(strings.ToLower(typed), candidate)
		if distance < 0 || measured < distance {
			best, distance = candidate, measured
		}
	}
	if best == "" {
		return "", false
	}
	// Two edits, and never more than a third of the word.
	//
	// Which of the two binds depends on the length of the verb, and a mutation
	// made that worth writing down: for anything six letters or longer the
	// ceiling of two is the binding one, so with validate as the only command
	// the division does nothing. It starts to matter the day a short verb
	// arrives, where two edits out of three letters would match almost
	// anything.
	limit := len(best) / 3
	if limit > 2 {
		limit = 2
	}
	return best, distance <= limit
}

// editDistance counts single character insertions, deletions and substitutions
// between two words, over bytes.
//
// Bytes rather than runes is correct here and worth saying why: command names
// are a public contract and this project writes them in the ASCII range, so a
// rune walk would cost a conversion and answer the same thing.
func editDistance(from, to string) int {
	previous := make([]int, len(to)+1)
	current := make([]int, len(to)+1)
	for column := range previous {
		previous[column] = column
	}
	for row := 1; row <= len(from); row++ {
		current[0] = row
		for column := 1; column <= len(to); column++ {
			cost := 1
			if from[row-1] == to[column-1] {
				cost = 0
			}
			current[column] = minimumOf(
				previous[column]+1,
				current[column-1]+1,
				previous[column-1]+cost,
			)
		}
		previous, current = current, previous
	}
	return previous[len(to)]
}

func minimumOf(first, second, third int) int {
	smallest := first
	if second < smallest {
		smallest = second
	}
	if third < smallest {
		smallest = third
	}
	return smallest
}

// unusableJSON reports what is wrong with --json in this invocation, and
// whether anything is.
//
// # Why the two refusals are separate
//
// --json is a cross cutting flag, and two reports answer to it since DEC-146:
// --version and --license. A run that asked for machine readable output and got
// prose has been quietly ignored, which is worse than refusing.
//
// 🔴 Two reports at once are refused as well, even though both now speak JSON,
// and that reason changed rather than went away. Printing them together would
// put two documents on one channel, and two JSON documents in a row are not a
// JSON document: the parser on the other side stops after the first and never
// learns that a second was there. That is the silence this tool argues against,
// one layer up from the data.
//
// Both tests are limited to runs that asked for a report, so that --json on its
// own is answered with the reason that is actually true: nothing was asked for
// at all.
func unusableJSON(asked invocation) (problem, bool) {
	if !*asked.global.json {
		return problem{}, false
	}
	count := asked.reportsAsked()
	if count > 1 {
		return problem{
			what:    problemJSONTwoReports,
			why:     problemJSONTwoReportsWhy,
			allowed: problemJSONTwoReportsAllowed,
			instead: problemJSONTwoReportsDo,
		}, true
	}
	if count > 0 && !*asked.showVersion && !*asked.showLicense {
		return problem{
			what:    problemJSONUnsupported,
			why:     problemJSONUnsupportedWhy,
			allowed: problemJSONUnsupportedAllowed,
			instead: problemJSONUnsupportedDo,
		}, true
	}
	return problem{}, false
}

// runReports prints everything the invocation asked for, or refuses the run.
func runReports(out *streams, asked invocation) int {
	if refusal, unusable := unusableJSON(asked); unusable {
		return usage(out, refusal)
	}

	// Every report that was asked for gets printed, in a fixed order. A
	// precedence rule where one flag swallows another is the smallest
	// possible silence, and this tool exists to argue against silence.
	printed := 0
	separate := func() {
		if printed > 0 {
			out.emit("")
		}
		printed++
	}
	if asked.wantsHelp() {
		separate()
		printHelp(out)
	}
	if *asked.showVersion {
		separate()
		printVersion(out, *asked.global.json)
	}
	if *asked.showLicense {
		separate()
		printLicense(out, *asked.global.json)
	}
	if printed > 0 {
		return finish(out, codeOK)
	}

	// A run with nothing to do ends as a usage failure rather than as
	// success. A pipeline that called this with an empty variable has to find
	// out now, not from an empty output file three steps later.
	return usage(out, problem{
		what:    problemNoCommand,
		why:     problemNoCommandWhy,
		allowed: acceptedInvocations(),
		instead: problemNoCommandDo,
	})
}

// reportCrash turns a crash into the exit code for a tool failure.
//
// It writes straight to the log channel rather than through the usual type,
// because whatever stopped the run may have been that type. It stays silent
// when there is no log channel to write to, which is the only case where
// silence is the honest answer.
// The results are dropped for the reason above: this is the path taken when
// the run has already failed in a way nobody planned for, and a second failure
// while describing the first has nowhere left to be reported.
func reportCrash(env Environment, recovered any) int {
	// The recovered value goes through plain for the same reason every other
	// line on this channel does. Nothing in this program panics with text that
	// came from outside today, so this is closing a door rather than a hole,
	// and it is one word.
	if env.Stderr != nil {
		_, _ = fmt.Fprintln(env.Stderr, prefixError+problemCrash)
		_, _ = fmt.Fprintln(env.Stderr, prefixDetail+plain(fmt.Sprint(recovered)))
		_, _ = fmt.Fprintln(env.Stderr, prefixDetail+labelTry+problemCrashDo)
	}
	return int(codeRuntime)
}

// crossCutting holds the flags that mean the same thing wherever they appear.
//
// They are declared twice, once for the whole invocation and once for the
// command, because the flag package stops reading flags at the first argument
// that is not one. Without the second declaration --json would work before the
// command and be an error after it, which is exactly the trap a cross cutting
// flag is supposed to avoid.
type crossCutting struct {
	json    *bool
	quiet   *bool
	verbose *bool
}

func declareCrossCutting(set *flag.FlagSet) crossCutting {
	// no-color is declared so that it is accepted and then never read here.
	// The answer comes from colorTurnedOff, for the reason written beside it.
	set.Bool("no-color", false, "")
	return crossCutting{
		json:    set.Bool("json", false, ""),
		quiet:   set.Bool("quiet", false, ""),
		verbose: set.Bool("verbose", false, ""),
	}
}

// merge folds what the command said into what the whole invocation said. The
// flags are all switches, so either position turning one on turns it on.
func (c crossCutting) merge(other crossCutting) crossCutting {
	*c.json = *c.json || *other.json
	*c.quiet = *c.quiet || *other.quiet
	*c.verbose = *c.verbose || *other.verbose
	return c
}

// colorTurnedOff reads --no-color from the arguments themselves rather than
// from the parsed flags.
//
// Parsing fails exactly when a failure has to be printed, and a failed parse
// leaves every flag at its default. Reading the parsed value would therefore
// put escape sequences into the output of every invocation that also had a
// typo in it, which is the one case where nobody is watching a terminal.
// The last spelling wins, which is what the flag package does with a flag given
// twice. Taking the first would make this and the parser disagree about the
// same command line, and the disagreement would only show in colour, which is
// the hardest kind of difference to notice in a log.
func colorTurnedOff(args []string) bool {
	off := false
	for _, arg := range args {
		if arg == "--" {
			// Everything after this is an argument, not a flag.
			return off
		}
		name, value, hasValue := strings.Cut(arg, "=")
		if name != "-no-color" && name != "--no-color" {
			continue
		}
		if !hasValue {
			off = true
			continue
		}
		if parsed, err := strconv.ParseBool(value); err == nil {
			off = parsed
		}
	}
	return off
}

// applyVerbosity turns the two switches into one level, or refuses the pair.
// It is shared with the commands so that asking for both after the verb fails
// the same way as asking for both before it.
func applyVerbosity(out *streams, flags crossCutting) (int, bool) {
	switch {
	case *flags.quiet && *flags.verbose:
		return usage(out, problem{
			what:    problemQuietAndVerbose,
			why:     problemQuietAndVerboseWhy,
			allowed: problemQuietAndVerboseAllowed,
			instead: problemQuietAndVerboseDo,
		}), false
	case *flags.quiet:
		out.level = levelQuiet
	case *flags.verbose:
		out.level = levelVerbose
	}
	return 0, true
}

// usage reports a wrong invocation and returns its code.
func usage(out *streams, p problem) int {
	out.report(p)
	return int(codeUsage)
}

// finish turns a successful path into an exit code, unless writing the data
// itself failed. Output that went nowhere is not a successful run.
func finish(out *streams, code exitCode) int {
	if out.dataErr != nil {
		return fail(out, fault.Wrap(fault.IO, out.dataErr, "cannot write to standard output"))
	}
	return int(code)
}

// finishFailed ends a run that has both a failure to report and output that
// was already written.
//
// A failed write wins over the failure being reported, because a report that
// never arrived leaves the reader with nothing, while a report that arrived
// alongside a bad exit code leaves them with everything.
func finishFailed(out *streams, err error) int {
	if out.dataErr != nil {
		return fail(out, fault.Wrap(fault.IO, out.dataErr, "cannot write to standard output"))
	}
	return fail(out, err)
}

// fail reports a failure on the log channel and returns the exit code that
// belongs to it.
//
// The code comes from the failure itself. Choosing it here with a switch over
// the message is how every new failure mode silently ends up as 1.
func fail(out *streams, err error) int {
	kind := fault.KindOf(err)
	code, known := codeFor(kind)
	if !known {
		out.report(problem{
			what:    problemMissingKind + ": " + err.Error(),
			why:     problemMissingKindWhy,
			allowed: problemMissingKindAllowed,
			instead: problemMissingKindDo,
		})
		return int(code)
	}
	out.report(problem{what: err.Error(), instead: whatToDoAbout(kind, out.verb)})
	return int(code)
}

// whatToDoAbout returns the fourth part of a refusal, chosen by the kind the
// failure carried.
//
// # Why by kind rather than by message
//
// The message already says what went wrong, and often what would have been
// accepted. What it never said is the next move, and the kind is exactly the
// thing that decides it: a path that cannot be written needs different action
// from a recipe that cannot be satisfied. One sentence per kind is also the
// only version that cannot drift, because a sentence per message would be
// written by whoever adds the message and forgotten by whoever adds the next.
//
// 🔴 What this does not do: it does not add the other two parts. Where the
// message carries them it would repeat itself, and where it does not, a
// sentence invented here would be guessing at a failure it cannot see.
//
// # Why the verb is a second key, since 2026-08-11
//
// Two of these sentences send the reader to their recipe, which was the right
// next move while every verb had one. The verb that damages somebody else's
// file does not, so both sentences became untrue for it on the day it arrived,
// and untrue in the worst available way: they name a file the reader does not
// have and a command that cannot help them. Found by running the binary rather
// than by reading it, with every test green.
//
// The pair rather than a sentence per message. What went wrong is in the
// message already, and what to do next is decided by exactly these two things:
// the kind of the failure, and whether this run was told what the data should
// look like.
// 🔴 The test is which verbs DO have a recipe, rather than which one does not.
// The first version listed the single verb without one, and the next verb
// without one inherited advice naming a file it has no idea about and a help
// page for somebody else's command. Asking the other question means a verb
// added tomorrow is right by default and only a verb that really reads a
// recipe has to say so.
func whatToDoAbout(kind fault.Kind, verb string) string {
	hasRecipe := verb == commandValidate || verb == commandGenerate
	// 🔴 The same question asked a second time, on 2026-08-13. Widening code 3
	// to cover any input document that is not valid (DEC-172) gave this verb a
	// Config failure about a FILE, and the advice for a verb without a recipe
	// names the flags. Which is exactly the shape of the earlier mistake: true
	// of every caller on the day it was written, false the moment another kind
	// of caller arrived.
	readsDocuments := verb == commandVerify
	switch kind {
	case fault.IO:
		return doAboutIO
	case fault.Config:
		switch {
		case hasRecipe:
			return doAboutConfig
		case readsDocuments:
			return doAboutConfigDocument
		default:
			return doAboutConfigFlags + verb + doAboutConfigFlagsTail
		}
	case fault.Data:
		if !hasRecipe {
			return doAboutDataCorrupt
		}
		return doAboutData
	case fault.Interrupted, fault.Terminated:
		return doAboutStopped
	default:
		return doAboutRuntime
	}
}

func printVersion(out *streams, machineReadable bool) {
	current := version.Current()
	if machineReadable {
		encoded, err := json.Marshal(current)
		if err != nil {
			// A struct of four strings cannot fail to encode. Reporting
			// it anyway costs one branch and beats a silent empty line.
			out.dataErr = err
			return
		}
		out.emit(string(encoded))
		return
	}
	out.emit(versionLabelTool + " " + current.Tool)
	out.emit(versionLabelCatalog + " " + current.Catalog)
	out.emit(versionLabelManifest + " " + current.Manifest)
	out.emit(versionLabelRecipe + " " + current.Recipe)
}

// printLicense says what is inside this binary and under what terms.
//
// The machine readable form is not a convenience. The reader named in DEC-30 is
// a legal department, and the tool it runs is a compliance scanner, which reads
// documents rather than paragraphs. Left without one it would parse the text
// below, and text written for a person changes wording without changing meaning.
func printLicense(out *streams, machineReadable bool) {
	if machineReadable {
		encoded, err := json.Marshal(license.Current())
		if err != nil {
			// Nothing in the report is unencodable, so this cannot fire. It is
			// reported rather than dropped for the same reason as in the
			// version report: a silent empty line is the one output nobody can
			// act on.
			out.dataErr = err
			return
		}
		out.emit(string(encoded))
		return
	}
	out.emit(versionLabelTool + " " + version.Tool)
	out.emit("")
	out.emit(licenseHeadingCode + " " + license.Code)
	out.emit(licenseHeadingSpec + " " + license.Spec)
	out.emit("  " + license.SpecScope)
	out.emit("")
	out.emit(licenseHeadingParts)
	if len(license.ThirdParty) == 0 {
		out.emit(licenseNoComponents)
	}
	for _, component := range license.ThirdParty {
		out.emit("  " + component.Module + " " + component.Version)
		out.emit("    " + component.License + "  " + component.Source)
	}

	// The reference data ships inside this binary, so whoever received the
	// binary is the person its terms bind. Naming it only in the repository
	// would put the notice where they will not look.
	out.emit("")
	out.emit(licenseHeadingData)
	if len(license.Data) == 0 {
		out.emit(licenseNoData)
	}
	for _, source := range groupData(license.Data) {
		out.emit("  " + source.project + " " + source.version)
		out.emit("    " + source.license + "  " + source.source)
		for _, list := range source.lists {
			out.emit("      " + list)
		}
	}

	out.emit("")
	out.emit(license.GeneratedDataNotice)
	out.emit("")
	out.emit(license.SyntheticDataNotice)
}

// takenFrom is one project and everything read from it. The registry keeps a
// row per list, because that is what a gate can compare with the files, and a
// person reading the licences wants the project named once.
type takenFrom struct {
	project string
	version string
	license string
	source  string
	lists   []string
}

func groupData(sources []license.DataSource) []takenFrom {
	var grouped []takenFrom
	index := map[string]int{}
	for _, entry := range sources {
		key := entry.Project + " " + entry.Version
		at, seen := index[key]
		if !seen {
			index[key] = len(grouped)
			grouped = append(grouped, takenFrom{
				project: entry.Project,
				version: entry.Version,
				license: entry.License,
				source:  entry.Source,
				lists:   []string{entry.Corpus},
			})
			continue
		}
		grouped[at].lists = append(grouped[at].lists, entry.Corpus)
	}
	return grouped
}
