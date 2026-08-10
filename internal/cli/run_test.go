package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

type outcome struct {
	code int
	data string
	log  string
}

func invoke(env Environment, args ...string) outcome {
	var data, log bytes.Buffer
	env.Args = args
	env.Stdout = &data
	env.Stderr = &log
	return outcome{code: Run(env), data: data.String(), log: log.String()}
}

func run(args ...string) outcome {
	return invoke(Environment{}, args...)
}

func TestHelpGoesToTheDataChannelAndSucceeds(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		got := run(flag)
		if got.code != 0 {
			t.Errorf("%s ended with %d", flag, got.code)
		}
		if got.log != "" {
			t.Errorf("%s wrote to the log channel: %q", flag, got.log)
		}
		for _, wanted := range []string{"Usage:", "Flags:", "Channels:", exitCodesHeading, "EXPECTATION"} {
			if !strings.Contains(got.data, wanted) {
				t.Errorf("%s left out %q", flag, wanted)
			}
		}
	}
}

func TestVersionReportsAllFourNumbers(t *testing.T) {
	got := run("--version")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}
	lines := strings.Split(strings.TrimSpace(got.data), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected four lines, got %d: %q", len(lines), got.data)
	}
	for _, label := range []string{versionLabelTool, versionLabelCatalog, versionLabelManifest, versionLabelRecipe} {
		if !strings.Contains(got.data, label) {
			t.Errorf("no line for %q", label)
		}
	}
}

func TestVersionHasAMachineReadableForm(t *testing.T) {
	got := run("--version", "--json")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}
	var decoded version.Set
	if err := json.Unmarshal([]byte(got.data), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if decoded != version.Current() {
		t.Fatalf("expected %+v, got %+v", version.Current(), decoded)
	}
}

func TestLicenseNamesBothLicensesAndFreesTheOutput(t *testing.T) {
	got := run("--license")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}
	for _, wanted := range []string{"GPL-3.0-only", "CC-BY-4.0", "belongs to you", licenseHeadingParts} {
		if !strings.Contains(got.data, wanted) {
			t.Errorf("the license report leaves out %q", wanted)
		}
	}
}

// The flag has to say that everything produced here is synthetic.
//
// Not because a licence requires it, but because several of these schemes have
// no block that is never issued, so a correct value cannot be shown not to
// belong to somebody. A README does not travel with a file of data and this
// does, which is the whole reason the sentence lives in the binary.
func TestLicenseSaysEveryValueIsSynthetic(t *testing.T) {
	got := run("--license")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}
	if !strings.Contains(got.data, license.SyntheticDataNotice) {
		t.Error("the license report no longer says that every value is synthetic")
	}
}

// The machine readable form has to carry everything the printed one does.
//
// # Why this is checked field by field rather than against a fixture
//
// A fixture would freeze the whole document and fail on every addition, which
// is the opposite of what DEC-146 promised: the shape grows by adding fields.
// What must not change is that a compliance scanner finds the answer it came
// for, so the test asks for exactly those answers and says nothing about the
// rest.
func TestTheMachineReadableLicenseCarriesWhatTheReadableOneDoes(t *testing.T) {
	got := run("--license", "--json")
	if got.code != 0 {
		t.Fatalf("ended with %d, log %q", got.code, got.log)
	}

	var report license.Report
	if err := json.Unmarshal([]byte(got.data), &report); err != nil {
		t.Fatalf("the license report is not a JSON document: %v", err)
	}
	if report.Code.License != license.Code {
		t.Errorf("the code license is %q, expected %q", report.Code.License, license.Code)
	}
	if report.Spec.License != license.Spec || report.Spec.Scope == "" {
		t.Error("the specification license or its scope is missing, and the two names read as a contradiction without the scope")
	}
	if report.Tool == "" {
		t.Error("the report does not say which build it describes")
	}
	if report.Notices.SyntheticData != license.SyntheticDataNotice {
		t.Error("the machine readable report drops the notice that every value is synthetic")
	}

	// Every row of both registries has to be there. A scanner that sees one of
	// two dependencies reports compliance it cannot claim, and the failure is
	// invisible on the printed form because prose reads complete either way.
	if len(report.Components) != len(license.ThirdParty) {
		t.Errorf("the report carries %d components, the registry has %d",
			len(report.Components), len(license.ThirdParty))
	}
	if len(report.Data) != len(license.Data) {
		t.Errorf("the report carries %d data sources, the registry has %d",
			len(report.Data), len(license.Data))
	}
	for _, source := range report.Data {
		if source.Checked == "" {
			t.Errorf("%s carries no date on which its terms were read, so nobody can tell a checked fact from a remembered one", source.Corpus)
		}
	}
}

// The two forms have to answer with the same facts.
//
// They are built by different code, and the printed one groups the lists under
// their project while this one does not. A difference in arrangement is fine, a
// difference in what is named is not, and only one of the two is read by the
// person who can block adoption.
//
// # Why this walks the registry and not the report
//
// The first version of this test iterated over the document and asked whether
// each row appeared in the printed form. A mutation emptying the document made
// it iterate over nothing, so it passed against a build with no mechanism at
// all: the exact shape that left three guards green in point 5. Walking the
// registry means an empty document fails on the first row, because the registry
// is what both forms are supposed to be showing.
func TestBothLicenseFormsNameTheSameProjects(t *testing.T) {
	printed := run("--license")
	machine := run("--license", "--json")

	var report license.Report
	if err := json.Unmarshal([]byte(machine.data), &report); err != nil {
		t.Fatalf("the license report is not a JSON document: %v", err)
	}
	inDocument := func(name string) bool {
		for _, component := range report.Components {
			if component.Module == name {
				return true
			}
		}
		for _, source := range report.Data {
			if source.Project == name {
				return true
			}
		}
		return false
	}

	for _, component := range license.ThirdParty {
		if !strings.Contains(printed.data, component.Module) {
			t.Errorf("%s is in the registry and not in the printed form", component.Module)
		}
		if !inDocument(component.Module) {
			t.Errorf("%s is in the registry and not in the document", component.Module)
		}
	}
	for _, source := range license.Data {
		if !strings.Contains(printed.data, source.Project) {
			t.Errorf("%s is in the registry and not in the printed form", source.Project)
		}
		if !inDocument(source.Project) {
			t.Errorf("%s is in the registry and not in the document", source.Project)
		}
	}
}

// Asking for two reports and getting one is the smallest possible silence, and
// silence is the thing this product argues against.
func TestBothReportsPrintWhenBothAreAsked(t *testing.T) {
	got := run("--version", "--license")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}
	if !strings.Contains(got.data, versionLabelCatalog) || !strings.Contains(got.data, "CC-BY-4.0") {
		t.Fatalf("one of the two reports is missing: %q", got.data)
	}
}

func TestFailingRunsWriteNothingToTheDataChannel(t *testing.T) {
	cases := map[string][]string{
		"no arguments":      {},
		"unknown flag":      {"--nope"},
		"unknown command":   {"frobnicate"},
		"quiet and verbose": {"--quiet", "--verbose"},
		// A report that is prose, asked for as a document. The case used to be
		// --json --license, which stopped being a refusal when the license
		// report gained a machine readable form in DEC-146.
		"json over a report that is prose": {"--json", "--help"},
		// Both of these speak JSON, and that is exactly why they are refused
		// together: two documents on one channel give a reader that stops after
		// the first and never learns the second was there.
		"json over two documents at once": {"--json", "--version", "--license"},
	}
	for name, args := range cases {
		got := run(args...)
		if got.code != int(codeUsage) {
			t.Errorf("%s: expected %d, got %d", name, codeUsage, got.code)
		}
		if got.data != "" {
			t.Errorf("%s: wrote %q to the data channel", name, got.data)
		}
		if !strings.HasPrefix(got.log, prefixError) {
			t.Errorf("%s: the log does not start with the error prefix: %q", name, got.log)
		}
		if !strings.Contains(got.log, "try: ") {
			t.Errorf("%s: the failure does not say what to do instead: %q", name, got.log)
		}
	}
}

func TestColorOnlyWhenTheLogIsATerminalAndNobodySaidNo(t *testing.T) {
	onTerminal := Environment{StderrIsTerminal: true}
	if got := invoke(onTerminal, "--nope"); !strings.Contains(got.log, ansiRed) {
		t.Error("a terminal should get color")
	}
	if got := invoke(onTerminal, "--nope", "--no-color"); strings.Contains(got.log, "\x1b[") {
		t.Error("--no-color was ignored")
	}
	refused := Environment{StderrIsTerminal: true, NoColorRequested: true}
	if got := invoke(refused, "--nope"); strings.Contains(got.log, "\x1b[") {
		t.Error("NO_COLOR was ignored")
	}
	if got := run("--nope"); strings.Contains(got.log, "\x1b[") {
		t.Error("a pipe must never see an escape sequence")
	}
}

// Input that is odd but legal, which is the kind this will actually receive.
func TestArgumentTerminatorIsRespected(t *testing.T) {
	// A bare terminator leaves nothing behind it, so there is still no
	// command.
	if got := run("--"); got.code != int(codeUsage) || !strings.Contains(got.log, problemNoCommand) {
		t.Errorf("expected the no command failure, got %d and %q", got.code, got.log)
	}

	// After the terminator a flag is an argument. It must not turn color
	// off, and it must be reported as a command rather than as a flag.
	got := invoke(Environment{StderrIsTerminal: true}, "--", "--no-color")
	if !strings.Contains(got.log, "unknown command") {
		t.Errorf("past the terminator a flag is an argument, got %q", got.log)
	}
	if !strings.Contains(got.log, ansiRed) {
		t.Error("a flag past the terminator must not switch color off")
	}
}

func TestColorCanBeAskedForExplicitly(t *testing.T) {
	got := invoke(Environment{StderrIsTerminal: true}, "--nope", "--no-color=false")
	if !strings.Contains(got.log, ansiRed) {
		t.Error("--no-color=false says color is wanted")
	}
}

func TestDataChannelNeverCarriesEscapeSequences(t *testing.T) {
	got := invoke(Environment{StderrIsTerminal: true}, "--version")
	if strings.Contains(got.data, "\x1b[") {
		t.Fatal("data must stay parseable even on a terminal")
	}
}

// One flag swallowing another is the smallest possible silence.
func TestHelpDoesNotSwallowTheOtherReports(t *testing.T) {
	got := run("--help", "--version")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}
	if !strings.Contains(got.data, exitCodesHeading) {
		t.Error("the help is missing")
	}
	if !strings.Contains(got.data, versionLabelCatalog) {
		t.Error("the version report was dropped")
	}
}

// JSON followed by prose on one channel is unparseable, which is worse than
// refusing to print it.
func TestJSONIsRefusedWhenASecondReportWouldFollowIt(t *testing.T) {
	for _, args := range [][]string{
		{"--version", "--license", "--json"},
		{"--version", "--help", "--json"},
	} {
		got := run(args...)
		if got.code != int(codeUsage) {
			t.Errorf("%v: expected %d, got %d", args, codeUsage, got.code)
		}
		if got.data != "" {
			t.Errorf("%v: wrote %q to the data channel", args, got.data)
		}
	}
}

// A failure has to give the reason that is true, not the first one that
// matches. Nothing was asked for here, and --json is beside the point.
func TestJSONAloneIsReportedAsNoCommand(t *testing.T) {
	got := run("--json")
	if got.code != int(codeUsage) {
		t.Fatalf("expected %d, got %d", codeUsage, got.code)
	}
	if !strings.Contains(got.log, problemNoCommand) {
		t.Fatalf("expected the no command failure, got %q", got.log)
	}
}

// The list of accepted flags in a failure comes from the same place as the
// help text, so the two cannot drift apart.
func TestAcceptedFlagsMatchTheHelpText(t *testing.T) {
	accepted := acceptedFlags()
	for _, doc := range flagDocs {
		for _, name := range doc.names {
			if !strings.Contains(accepted, name) {
				t.Errorf("%s is documented and not listed as accepted", name)
			}
		}
		if !strings.Contains(renderFlags(), doc.shown) {
			t.Errorf("%s is accepted and not documented", doc.shown)
		}
	}
	if got := run("--nope"); !strings.Contains(got.log, accepted) {
		t.Errorf("the failure does not list the accepted flags: %q", got.log)
	}
}

// Without this a crash leaves through Go's own exit code, which is the number
// that means a wrong invocation here.
func TestACrashEndsAsAToolFailureAndNotAsAUsageFailure(t *testing.T) {
	var log bytes.Buffer
	// A missing data channel is the degenerate input that reaches the write
	// itself, so the recovery is exercised through a real path.
	code := Run(Environment{Args: []string{"--version"}, Stderr: &log})
	if code != int(codeRuntime) {
		t.Fatalf("expected %d, got %d", codeRuntime, code)
	}
	if !strings.Contains(log.String(), problemCrash) {
		t.Fatalf("a crash has to say so: %q", log.String())
	}
}

func TestACrashWithNowhereToReportItStillEndsWithTheRightCode(t *testing.T) {
	if code := Run(Environment{Args: []string{"--version"}}); code != int(codeRuntime) {
		t.Fatalf("expected %d, got %d", codeRuntime, code)
	}
}

type refusingWriter struct{}

func (refusingWriter) Write([]byte) (int, error) {
	return 0, errors.New("the other end of the pipe is gone")
}

// Output that went nowhere is not a successful run. Without this the exit code
// would say the data was produced while the file stayed empty.
func TestAFailedWriteEndsAsAnInputOutputFailure(t *testing.T) {
	var log bytes.Buffer
	code := Run(Environment{
		Args:   []string{"--version"},
		Stdout: refusingWriter{},
		Stderr: &log,
	})
	if code != int(codeIO) {
		t.Fatalf("expected %d, got %d", codeIO, code)
	}
	if !strings.Contains(log.String(), "standard output") {
		t.Fatalf("the log does not say what failed: %q", log.String())
	}
}

func TestAFailureWithoutAKindIsReportedAsADefectOfOurOwn(t *testing.T) {
	var log bytes.Buffer
	out := &streams{data: &bytes.Buffer{}, log: &log}
	code := fail(out, errors.New("something went wrong"))
	if code != int(codeRuntime) {
		t.Fatalf("expected %d, got %d", codeRuntime, code)
	}
	if !strings.Contains(log.String(), problemMissingKind) {
		t.Fatalf("a kindless failure has to say so: %q", log.String())
	}
}

func TestAFailureWithAKindGetsItsOwnCode(t *testing.T) {
	var log bytes.Buffer
	out := &streams{data: &bytes.Buffer{}, log: &log}
	code := fail(out, fault.New(fault.Config, "the recipe names a mutator that does not exist"))
	if code != int(codeConfig) {
		t.Fatalf("expected %d, got %d", codeConfig, code)
	}
	if strings.Contains(log.String(), problemMissingKind) {
		t.Fatal("a failure that carried its kind must not be reported as a defect")
	}
}

func TestQuietHidesProgressAndKeepsWarnings(t *testing.T) {
	var log bytes.Buffer
	out := &streams{data: &bytes.Buffer{}, log: &log, level: levelQuiet}
	out.info("reading the recipe")
	out.warn("one mutator could not be applied")
	if strings.Contains(log.String(), "reading the recipe") {
		t.Error("progress detail should be quiet")
	}
	if !strings.Contains(log.String(), "one mutator could not be applied") {
		t.Error("a warning that --quiet hides is a warning that never happened")
	}
}

func TestVerboseShowsProgress(t *testing.T) {
	var log bytes.Buffer
	out := &streams{data: &bytes.Buffer{}, log: &log, level: levelVerbose}
	out.info("reading the recipe")
	if !strings.Contains(log.String(), "reading the recipe") {
		t.Error("--verbose asked for detail and got none")
	}
}

// The help has to finish somebody's first attempt, not describe it.
//
// A usage pattern like qadg <command> [arguments] tells a reader who already
// knows the tool what shape a line has. Somebody meeting it for the first time
// wants a line to paste, and clig.dev puts examples first for that reason. The
// test pins the order rather than the presence, because an example buried
// under the flag list answers the question too late.
func TestTheHelpBeginsWithAnExampleBeforeTheUsagePattern(t *testing.T) {
	got := run("--help")
	if got.code != 0 {
		t.Fatalf("ended with %d", got.code)
	}

	example := strings.Index(got.data, "Examples:")
	usage := strings.Index(got.data, "Usage:")
	flags := strings.Index(got.data, flagsHeading)

	if example < 0 {
		t.Fatal("the help carries no examples section")
	}
	if usage >= 0 && example > usage {
		t.Error("the usage pattern comes before the examples")
	}
	if flags >= 0 && example > flags {
		t.Error("the flag list comes before the examples")
	}
	// An example nobody can paste is a heading, not an example.
	if !strings.Contains(got.data, "qadg validate recipe.yaml") {
		t.Error("no example carries a whole command somebody could copy")
	}
}

// A slip of the fingers gets the answer, not only the refusal.
func TestACommandOneEditAwayIsSuggested(t *testing.T) {
	for _, typed := range []string{"vlaidate", "validat", "Validate", "valdiate"} {
		got := run(typed, "recipe.yaml")
		if got.code != 2 {
			t.Errorf("%q ended with %d, and a wrong verb is a usage failure", typed, got.code)
		}
		if !strings.Contains(got.log, commandValidate) {
			t.Errorf("%q was refused without naming the command it is near: %q", typed, got.log)
		}
		if !strings.Contains(got.log, "try: qadg "+commandValidate) {
			t.Errorf("%q was refused without a line to try instead: %q", typed, got.log)
		}
	}
}

// The threshold has to be mean, and this is the half of the pair that keeps it
// mean. Suggesting a command for any vaguely similar word turns a clear
// refusal into a misleading one.
// 🔴 The words here have to be ones this tool will never have as a verb. It
// used to say generate, which was a plausible absent command until the day it
// stopped being absent, and then this test failed for a reason that had nothing
// to do with what it protects. A fixture naming something the product might
// grow into is a fixture with a fuse in it.
func TestAWordNothingLikeACommandIsRefusedWithoutASuggestion(t *testing.T) {
	for _, typed := range []string{"frobnicate", "elephant", "list", "rm"} {
		got := run(typed, "recipe.yaml")
		if got.code != 2 {
			t.Errorf("%q ended with %d", typed, got.code)
		}
		if strings.Contains(got.log, "one small edit away") {
			t.Errorf("%q was offered as a near miss, and it is not one: %q", typed, got.log)
		}
		// The refusal still has to say what does exist, or it is a dead end.
		if !strings.Contains(got.log, labelAllowed) {
			t.Errorf("%q was refused without naming what is allowed: %q", typed, got.log)
		}
	}
}

// Every refusal keeps its four parts, suggestion or not. The contract in
// docs/09 is about the shape of the message, and a branch that improves the
// wording is exactly where a part goes missing.
func TestBothShapesOfTheUnknownCommandRefusalKeepFourParts(t *testing.T) {
	for _, typed := range []string{"vlaidate", "frobnicate"} {
		got := run(typed, "recipe.yaml")
		for _, part := range []string{prefixError, labelAllowed, labelTry} {
			if !strings.Contains(got.log, part) {
				t.Errorf("the refusal of %q is missing %q: %q", typed, part, got.log)
			}
		}
		if strings.Count(got.log, "\n") < 3 {
			t.Errorf("the refusal of %q is not four lines: %q", typed, got.log)
		}
	}
}

// Every documented flag is actually accepted, asked of the parser rather than
// of the list the help was printed from.
//
// This exists because the test above cannot answer it. That one compares
// acceptedFlags() with flagDocs, and both are projections of the same slice, so
// it holds two views of one list against each other and passes by construction.
// What nobody checked is the step where a flag becomes real: the names are
// registered by hand, in two separate places, and a documented flag missing its
// registration would make the help promise something the tool refuses.
//
// It is the same shape of gap the neighbouring project found in its own usage
// contract test, where the guard proved the list was self consistent while the
// most important entry was absent from it.
func TestEveryDocumentedFlagIsAcceptedByTheParser(t *testing.T) {
	for _, doc := range flagDocs {
		for _, name := range doc.names {
			got := run(name)
			if strings.Contains(got.log, problemUnknownFlagWhy) {
				t.Errorf("%s is in the help and the parser refuses it: %q", name, got.log)
			}
		}
	}
}

// 🔴 Every surface that lists what this tool accepts has to list all of it.
//
// This guard exists because two of them were wrong at once on 2026-08-04, and
// both had been wrong since the day the second verb arrived. The help said no
// command generates data in this build, while the command that does was already
// released, and the refusal shown to somebody who runs qadg with nothing named
// one verb out of two. Neither is reachable by a compiler and neither was
// covered by a test: they are prose about the code, sitting inside the code.
func TestEverySurfaceListingCommandsNamesAllOfThem(t *testing.T) {
	help := run("--help").data
	commands, _, found := strings.Cut(strings.SplitN(help, "Commands:", 2)[1], "Flags:")
	if !found {
		t.Fatalf("the help no longer has a block of commands followed by one of flags:\n%s", help)
	}

	empty := invoke(Environment{}).log
	for _, command := range knownCommands() {
		if !strings.Contains(commands, command) {
			t.Errorf("the help does not list the command %q:\n%s", command, commands)
		}
		if !strings.Contains(empty, command) {
			t.Errorf("a run with no arguments does not mention the command %q:\n%s", command, empty)
		}
		// The other direction of the same promise. A command the help offers and
		// the dispatcher does not know would be worse than one nobody mentions.
		if got := run(command); strings.Contains(got.log, "unknown command") {
			t.Errorf("the help offers %q and the dispatcher refuses it:\n%s", command, got.log)
		}
	}
}

// 🔴 What the guard above does not cover, said out loud rather than left to be
// assumed: a verb the dispatcher accepts and the register does not carry is
// invisible to it, because both of its lists are read from that register.
// Closing that would mean reading the switch itself, which is a check against
// the syntax tree rather than against behaviour. It is written down in the
// backlog instead of being quietly counted as covered.

// 🔴 A flag named in the help of a command has to be one that command takes.
//
// The reverse of the guard above and the same failure mode. When the help of a
// single command was written, the list of shared flags it showed included
// --version and --license, which no verb has ever accepted: the parser of a verb
// refuses them, and asking for a report together with a command is refused
// before that. The help would have sent somebody to type a flag that cannot work
// where they were standing.
func TestEveryFlagAVerbDocumentsIsOneItAccepts(t *testing.T) {
	for _, doc := range commandDocs {
		if doc.name == commandHelp {
			// The help verb takes a command name rather than flags, and the
			// dispatcher answers it before any flag set exists.
			continue
		}
		help := run(doc.name, "--help").data
		for _, flag := range append(append([]flagDoc{}, doc.flags...), crossCuttingFlagDocs...) {
			// Listed rather than merely mentioned. The description of one flag
			// names another, so searching the whole text would answer yes about
			// a flag that is only being talked about.
			if !strings.Contains(help, "\n  "+flag.shown) {
				t.Errorf("the help of %s does not list %s", doc.name, flag.shown)
			}
			for _, spelling := range flag.names {
				if got := run(doc.name, spelling); strings.Contains(got.log, "not defined") {
					t.Errorf("the help of %s shows %s and the parser refuses it:\n%s",
						doc.name, spelling, got.log)
				}
			}
		}
		for _, refused := range reportFlagDocs {
			if strings.Contains(help, "\n  "+refused.shown) {
				t.Errorf("the help of %s offers %s, which no command accepts",
					doc.name, refused.shown)
			}
		}
	}
}
