package guard

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The register of what the window can do, against what the command line can be
// asked to do.
//
// # Why this exists, and why prose was not enough
//
// The sixth unbreakable rule says every function has to be describable from
// both surfaces, and says of itself that it is a test rather than a postulate.
// It was carried out by hand once, on 2026-08-11, after ten days in which six
// functions had arrived. The matrix that came out of it lives in docs/08 and is
// true on the day it was written, which is the property every piece of prose in
// this project has.
//
// So the list of what can be asked for is not written here. It is handed in by
// the caller, which reads it from the register the command line already uses
// for its help, its refusals and its dispatcher. A verb or a flag added there
// appears here on the next run without anybody remembering to add it, and an
// unclassified one turns the run red.
//
// # What this cannot answer
//
// It does not look at the window. Reachability is declared below rather than
// read out of the screens, because a screen is a tree of widgets and no reading
// of it would tell a capability from a label that mentions one. That is the one
// manual part, and it is the part the ceilings below are aimed at: the list of
// what is not reachable may only shrink.

// Claim is a capability the window has, together with the test that shows the
// screen really does it.
//
// 🔴 The test name is the whole point of this type. Without it the register
// would be a list of PROMISES: an entry could name a screen nobody built, and
// every check here would stay green, because the rest of this file only asks
// whether each capability was classified. Naming a test moves the claim onto
// something that fails when it stops being true, which is the same bargain the
// register of protected behaviours already makes.
type Claim struct {
	// Screen is what a person opens to get at it.
	Screen string
	// Test is the test that fails if the screen stops doing it.
	Test string
}

// WindowReach is what a person can do in the window today.
var WindowReach = map[string]Claim{
	"--version": {
		Screen: "the About screen names the four versions this build speaks",
		Test:   "TestTheScreenCarriesAllFourVersions",
	},
	"--license": {
		Screen: "the About screen carries both licenses, every module and what each one covers",
		Test:   "TestTheScreenCarriesEveryFactTheLicenceFlagPrints",
	},
	"verify": {
		Screen: "the second state of the manifest browser: the same entries, each carrying how the " +
			"application answered, with what could not be settled and what could not be seen",
		Test: "TestTheSecondStateShowsWhatTheApplicationDidWithEveryJudgedRow",
	},
	"corrupt": {
		Screen: "the Take and corrupt screen: a file chosen or dropped, what it holds, what a run " +
			"would do, and the button that does it",
		Test: "TestTheScreenDamagesTheFileAndSaysWhatItWrote",
	},
	"corrupt --layers": {
		Screen: "the kinds of damage, as tick boxes built from the catalog this build carries",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --rows": {
		Screen: "the share of records, as a field beside the kinds",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --per-row": {
		Screen: "the defects per record, as a list of what the engine knows rather than a field",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --exclude-fields": {
		Screen: "the columns to leave alone, as a field taking names the way the flag does",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --delimiter": {
		Screen: "the field separator, which reads the file again when it changes rather than only " +
			"the run",
		Test: "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --locale": {
		Screen: "the data locale, beside the interface language and never the same control",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --seed": {
		Screen: "the seed, which the screen fills with the one the mode uses when nobody says",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --out": {
		Screen: "the directory to write to, as a field, and the answer above the button names it",
		Test:   "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --manifest": {
		Screen: "the extra form of the manifest, as a list whose first entry names the absence of " +
			"one rather than leaving it blank",
		Test: "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --overwrite": {
		Screen: "replacing files already there, as a tick box that is off until somebody says " +
			"otherwise",
		Test: "TestEverySettingOnTheScreenReachesTheRun",
	},
	"corrupt --dry-run": {
		Screen: "the state the screen is in before anybody presses the button. A window writes " +
			"nothing until it is told to, so this is what it does rather than a switch",
		Test: "TestTheScreenSaysWhatARunWouldDoBeforeItRunsAnything",
	},
}

// NotYetInWindow is what the command line can be asked for and the window
// cannot, with what each one waits for.
//
// 🔴 This map may only get smaller. It is the distance between the two
// surfaces, and the ceiling below is that distance written as a number so that
// it cannot grow by accident.
var NotYetInWindow = map[string]string{
	"generate":             "waits for the recipe editor with the live preview",
	"validate":             "waits for the same editor, which shows problems beside the field they belong to",
	"help":                 "waits for the first control carrying a hint. No help screen is planned, and that is recorded, but a window with no hints answers nothing",
	"-h/--help":            "the same as the verb above it",
	"--json":               "waits for Save as JSON on the About screen. The reader here is a compliance scanner, so the window has to PRODUCE the file rather than show it",
	"generate --out":       "waits for the directory picker of the editor",
	"generate --seed":      "waits for the editor",
	"generate --format":    "waits for the editor",
	"generate --manifest":  "waits for the editor",
	"generate --overwrite": "waits for the editor",
	"generate --dry-run":   "waits for the preview. In the window the answer is the default state rather than a switch, because a window writes nothing until it is told to",
}

// MediumOnly is what belongs to a terminal rather than to the product, with the
// reason each one has no counterpart in a window.
//
// 🔴 This is the escape hatch, so it is named, reasoned and capped. The sixth
// rule is recorded as one that erodes by one exception per session, and an
// uncapped list of exceptions is how that happens: a function nobody wants to
// build lands here instead of in the list above, and the distance stops
// telling the truth. Every entry answers HOW output appears rather than WHAT
// the tool does.
var MediumOnly = map[string]string{
	"--no-color": "color in a window comes from the theme rather than from a flag, and there is no escape sequence to turn off",
	"--quiet":    "a window has no log channel to quieten. What the command line writes to the log, a window either shows or does not",
	"--verbose":  "the same reason as the one above it",
}

// NotYetCeiling is how far apart the two surfaces are allowed to be.
//
// It is the length of NotYetInWindow written down, so that the distance is a
// number in the code rather than an impression. Lower it when a screen closes
// the gap. Raising it is a decision about the product, not housekeeping.
//
// It stood at 24 from 2026-08-17, when this register was written, until later
// that day, when the manifest browser took verify out of the list above. It
// stood at 23 until 2026-08-19, when the Take and corrupt screen learned to
// write and took the verb and its eleven flags with it in one step. That is the
// largest single fall available today, and what is left is one verb, its five
// flags and the four that stand alone.
const NotYetCeiling = 11

// MediumOnlyCeiling caps the exceptions for the same reason, one number lower
// than nothing: three flags say how the output looks, and a fourth would be a
// new kind of thing rather than one more of the same.
const MediumOnlyCeiling = 3

// Parity holds every capability of the command line against the register above
// and reports what does not line up.
//
// 🔴 It closes in both directions. A capability nobody classified is the
// failure this guard exists for, and an entry naming a capability that no
// longer exists is the other half: a register that only answers one of those
// two questions answers half of the question asked, which this project has
// already paid for once.
func Parity(capabilities []string) []string {
	var findings []string
	known := map[string]bool{}
	for _, capability := range capabilities {
		known[capability] = true
		where := 0
		if _, listed := WindowReach[capability]; listed {
			where++
		}
		for _, register := range []map[string]string{NotYetInWindow, MediumOnly} {
			if _, listed := register[capability]; listed {
				where++
			}
		}
		switch where {
		case 0:
			findings = append(findings, fmt.Sprintf(
				"%q can be asked for on the command line and no register says whether the window "+
					"can do it. Put it in WindowReach, in NotYetInWindow or in MediumOnly", capability))
		case 1:
		default:
			findings = append(findings, fmt.Sprintf(
				"%q is in %d registers at once, so nothing says which answer holds", capability, where))
		}
	}

	for name, register := range map[string]map[string]string{
		"NotYetInWindow": NotYetInWindow, "MediumOnly": MediumOnly,
	} {
		for capability := range register {
			if !known[capability] {
				findings = append(findings, fmt.Sprintf(
					"%s names %q, which the command line no longer offers. Remove it rather than "+
						"leaving a register that answers about something nobody can ask for",
					name, capability))
			}
		}
	}
	for capability, reach := range WindowReach {
		if !known[capability] {
			findings = append(findings, fmt.Sprintf(
				"WindowReach names %q, which the command line no longer offers. Remove it rather "+
					"than leaving a register that answers about something nobody can ask for",
				capability))
		}
		if reach.Test == "" {
			findings = append(findings, fmt.Sprintf(
				"WindowReach says the window does %q and names no test, so the claim rests on "+
					"nothing", capability))
		}
	}

	sort.Strings(findings)
	return findings
}

// ParityDistance is how many capabilities the window cannot reach, and how many
// there are in total. It exists so that the number can be printed rather than
// counted by whoever reads the register.
func ParityDistance(capabilities []string) (int, int) {
	return len(NotYetInWindow), len(capabilities)
}

// ParityProofs reports every claim about the window that names a test which is
// not in the tree.
//
// 🔴 This is the half that makes the first register more than a wish. A screen
// can be deleted, renamed or quietly stop being reached, and the claim above
// would go on being true looking. Reading the tree for the test by name is the
// cheapest thing that fails when that happens.
//
// # What it does not cover
//
// Whether the named test still checks what its name says. A test can be
// hollowed out and keep its name, and no reading of the tree would tell. That
// is what the mutation register is for, and it is said here rather than left
// for somebody to assume.
func ParityProofs(root string) ([]string, error) {
	tests, err := testNames(root)
	if err != nil {
		return nil, err
	}
	if len(tests) == 0 {
		return nil, fmt.Errorf("no test at all was found under %s, so this guard would check nothing", root)
	}

	var findings []string
	for capability, reach := range WindowReach {
		if reach.Test != "" && !tests[reach.Test] {
			findings = append(findings, fmt.Sprintf(
				"the window is said to do %q, proved by %s, and no such test is in the tree",
				capability, reach.Test))
		}
	}
	sort.Strings(findings)
	return findings, nil
}

// testNames collects every test function in the repository, by name.
func testNames(root string) (map[string]bool, error) {
	found := map[string]bool{}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (skippedDirs[name] || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && strings.HasPrefix(function.Name.Name, "Test") {
				found[function.Name.Name] = true
			}
		}
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return found, nil
}
