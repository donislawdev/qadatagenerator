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

	set := flag.NewFlagSet("qadg", flag.ContinueOnError)
	// The flag package prints its own message and its own usage text. Both
	// would break the four part shape of our failures and the split between
	// the channels, so we take the error and report it ourselves.
	set.SetOutput(io.Discard)

	showVersion := set.Bool("version", false, "")
	showLicense := set.Bool("license", false, "")
	help := set.Bool("help", false, "")
	helpShort := set.Bool("h", false, "")
	global := declareCrossCutting(set)

	parseErr := set.Parse(env.Args)

	out := &streams{
		data:  env.Stdout,
		log:   env.Stderr,
		color: env.StderrIsTerminal && !colorTurnedOff(env.Args) && !env.NoColorRequested,
		level: levelNormal,
	}

	if parseErr != nil {
		// Unreachable while both help flags are defined above, and kept
		// because the flag package answers this way for a help flag that
		// is not. Removing one definition would otherwise turn a request
		// for help into a failure.
		if errors.Is(parseErr, flag.ErrHelp) {
			printHelp(out)
			return finish(out, codeOK)
		}
		return usage(out, problem{
			what:    parseErr.Error(),
			why:     problemUnknownFlagWhy,
			allowed: acceptedFlags(),
			instead: problemUnknownFlagDo,
		})
	}

	if code, ok := applyVerbosity(out, global); !ok {
		return code
	}

	wantsHelp := *help || *helpShort
	asked := 0
	for _, requested := range []bool{wantsHelp, *showVersion, *showLicense} {
		if requested {
			asked++
		}
	}

	// The flag package stops at the first argument that is not a flag, so
	// anything after a command belongs to that command and never to this
	// set. A command asked for together with a report would leave one of the
	// two unanswered, so both are refused rather than one dropped.
	command := set.Args()
	if len(command) > 0 {
		if asked > 0 {
			return usage(out, problem{
				what:    problemCommandAndReport,
				why:     problemCommandAndReportWhy,
				allowed: problemCommandAndReportAllowed,
				instead: problemCommandAndReportDo,
			})
		}
		switch command[0] {
		case commandValidate:
			return runValidate(out, command[1:], global)
		default:
			return usage(out, problem{
				what:    "unknown command " + command[0],
				why:     problemUnknownCommandWhy,
				allowed: commandValidate,
				instead: problemUnknownCommandDo,
			})
		}
	}

	// --json is a cross cutting flag with one report behind it so far. A run
	// that asked for machine readable output and got prose has been quietly
	// ignored, and asking for two reports at once with --json would put JSON
	// and prose on the same channel. Both are worse than refusing.
	// The test is limited to runs that asked for a report, so that --json on
	// its own is answered with the reason that is actually true: nothing was
	// asked for at all.
	if *global.json && asked > 0 && !(*showVersion && asked == 1) {
		return usage(out, problem{
			what:    problemJSONUnsupported,
			why:     problemJSONUnsupportedWhy,
			allowed: problemJSONUnsupportedAllowed,
			instead: problemJSONUnsupportedDo,
		})
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
	if wantsHelp {
		separate()
		printHelp(out)
	}
	if *showVersion {
		separate()
		printVersion(out, *global.json)
	}
	if *showLicense {
		separate()
		printLicense(out)
	}
	if printed > 0 {
		return finish(out, codeOK)
	}

	// A run with nothing to do ends as a usage failure rather than as
	// success. A pipeline that called this with an empty variable has to
	// find out now, not from an empty output file three steps later.
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
func reportCrash(env Environment, recovered any) int {
	if env.Stderr != nil {
		fmt.Fprintln(env.Stderr, prefixError+problemCrash)
		fmt.Fprintln(env.Stderr, prefixDetail+fmt.Sprint(recovered))
		fmt.Fprintln(env.Stderr, prefixDetail+labelTry+problemCrashDo)
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
func colorTurnedOff(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			// Everything after this is an argument, not a flag.
			return false
		}
		name, value, hasValue := strings.Cut(arg, "=")
		if name != "-no-color" && name != "--no-color" {
			continue
		}
		if !hasValue {
			return true
		}
		if off, err := strconv.ParseBool(value); err == nil {
			return off
		}
	}
	return false
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
	out.report(problem{what: err.Error()})
	return int(code)
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

func printLicense(out *streams) {
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
	out.emit("")
	out.emit(license.GeneratedDataNotice)
}

func printHelp(out *streams) {
	blocks := []string{
		summaryLine,
		usageBlock,
		commandsBlock,
		renderFlags(),
		channelsBlock,
		exitCodesHeading + "\n" + renderExitCodes(),
	}
	out.emit(strings.Join(blocks, "\n\n"))
}
