package cli

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/guard"
)

// capabilities is everything this tool can be asked to do from the command
// line, read from the register the help, the refusals and the dispatcher
// already use.
//
// 🔴 It is READ rather than listed. A guard that needs somebody to add a line
// by hand is a guard somebody will one day not add a line to, and this project
// has already replaced one such list with a rule that reads the register.
//
// A flag is qualified by its verb, because --out means one thing beside
// generate and another beside corrupt, and a window may well reach one without
// the other.
func capabilities() []string {
	var all []string
	for _, verb := range commandDocs {
		all = append(all, verb.name)
		for _, flag := range verb.flags {
			all = append(all, verb.name+" "+strings.Join(flag.names, "/"))
		}
	}
	for _, flag := range append(append([]flagDoc{}, reportFlagDocs...), crossCuttingFlagDocs...) {
		all = append(all, strings.Join(flag.names, "/"))
	}
	return all
}

// Every capability of the command line is answered for from the side of the
// window, and the distance between the two surfaces is a number.
//
// 🔴 The sixth unbreakable rule says a function has to be describable from both
// surfaces and calls itself a test rather than a postulate. It was carried out
// by hand exactly once, on 2026-08-11, after ten days in which six functions
// had arrived and nobody had run it. This is that rule as a mechanism: what can
// be asked for is read from the code, so a verb or a flag added tomorrow turns
// this red until somebody says what the window does about it.
func TestEveryCapabilityOfTheCommandLineIsAnsweredForFromTheWindow(t *testing.T) {
	all := capabilities()
	if len(all) == 0 {
		t.Fatal("the register of verbs answered with nothing, so this guard would check nothing")
	}

	for _, finding := range guard.Parity(all) {
		t.Error(finding)
	}

	unreached, total := guard.ParityDistance(all)
	t.Logf("the window reaches %d of %d capabilities, %d are not yet there and %d belong to a terminal",
		total-unreached-len(guard.MediumOnly), total, unreached, len(guard.MediumOnly))

	// 🔴 The two ceilings are the distance written down. They may fall and they
	// may not rise by accident: a screen that closes a gap lowers the first, and
	// anything landing in the exceptions has to be a decision rather than the
	// easiest place to put a function nobody wants to build.
	if unreached > guard.NotYetCeiling {
		t.Errorf("%d capabilities are out of the window's reach and the ceiling is %d. "+
			"A new one has to be reachable or the ceiling has to be raised on purpose",
			unreached, guard.NotYetCeiling)
	}
	if len(guard.MediumOnly) > guard.MediumOnlyCeiling {
		t.Errorf("%d capabilities are excused as belonging to a terminal and the ceiling is %d. "+
			"The rule this guard serves is recorded as one that erodes by one exception at a time",
			len(guard.MediumOnly), guard.MediumOnlyCeiling)
	}
}

// Every claim that the window reaches something names a test that is in the
// tree.
//
// 🔴 Without this the first register would be a list of promises. An entry
// could name a screen nobody built, or one deleted last week, and the guard
// above would stay green because it only asks whether each capability was
// classified. What it still cannot tell is whether the named test checks what
// its name says, and that is what the mutation register is for.
func TestEveryClaimAboutTheWindowNamesATestThatExists(t *testing.T) {
	if len(guard.WindowReach) == 0 {
		t.Fatal("the window reaches nothing at all, which was true before 2026-08-12 and is not now")
	}
	root, err := guard.ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := guard.ParityProofs(root)
	if err != nil {
		t.Fatalf("the claims about the window could not be checked: %v", err)
	}
	for _, finding := range findings {
		t.Error(finding)
	}
}
