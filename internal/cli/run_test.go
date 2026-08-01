package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
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
		"no arguments":       {},
		"unknown flag":       {"--nope"},
		"unknown command":    {"frobnicate"},
		"quiet and verbose":  {"--quiet", "--verbose"},
		"json without a use": {"--json", "--license"},
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
