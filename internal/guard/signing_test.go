package guard

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
)

// # What this defends (B-419 (d), DEC-330)
//
// A release is signed on the maintainer's machines by sign_release.py, which
// hands the macOS archives to sign_macos.sh on a Mac and then starts
// attest-release.yml. None of it runs on a push: the script runs once per
// release, by a person, with a card and a Mac, and the workflow only when the
// script starts it. A defect in any of them surfaces on the day of a release,
// with a version spent and a person waiting. These guards read the files on
// every push instead, and each asks one thing that would otherwise surface only
// then.
//
// Every test that reads the real files first plants one fault in a copy of
// them and requires its check to report it (planted, in release_test.go).
//
// Adapted on 2026-11-18 from the release guards of Cert Doctor, a project of
// the same owner under the same licence.
const (
	signingScript  = ".github/scripts/sign_release.py"
	macSigning     = ".github/scripts/sign_macos.sh"
	attestWorkflow = ".github/workflows/attest-release.yml"
	signingPins    = ".github/signing-certificates.txt"
	sbomDocument   = "cmd/sbom/document.go"

	// spdxPredicate is the statement type of an SPDX 2.3 document, the one
	// actions/attest gives a statement made with sbom-path.
	spdxPredicate = "https://spdx.dev/Document/v2.3"
)

// pinNames are the two certificates a release is signed with.
var pinNames = []string{"windows-code-signing", "apple-developer-id"}

// TestThePinnedCertificatesAreDigestsWithADate holds the one written copy of
// each certificate digest. A digest, because a name can be issued twice. One
// copy, because the signing script and a later check of a published release
// each carrying their own is exactly the drift a pin exists to catch, so the
// script must read the file and no other file of the release may hold a
// digest itself.
//
// The dates are logged, not asserted. A certificate expiring is a fact of life
// and a red suite on that day helps nobody. The signing script refuses to sign
// with an expired one, which is where that refusal belongs.
func TestThePinnedCertificatesAreDigestsWithADate(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	_, broken := parsePins("# a comment\n\n" +
		"windows-code-signing " + digest + " 2027-08-19\n" +
		"windows-code-signing " + digest + " 2027-08-19\n" +
		"apple-developer-id " + strings.ToUpper(digest) + " 2029-10-27\n" +
		"someone-else " + digest + " 2027-08-19\n" +
		"windows-code-signing " + digest + " 19.08.2027 extra\n")
	requireFinding(t, broken, "names windows-code-signing twice")
	requireFinding(t, broken, "apple-developer-id is not a lowercase sha256")
	requireFinding(t, broken, "names someone-else")
	requireFinding(t, broken, "has 4 fields where a pin has three")
	requireFinding(t, broken, "names no apple-developer-id")
	_, broken = parsePins("apple-developer-id " + digest + " 2029-13-27\n")
	requireFinding(t, broken, "is not a date")

	root := rootOrFail(t)
	pins, problems := parsePins(readRepoFile(t, root, signingPins))
	requireNone(t, problems, "the pinned certificates")
	for _, name := range pinNames {
		t.Logf("%s expires %s", name, pins[name].expires.Format("2006-01-02"))
	}

	files := workflowFiles(t, root)
	for _, rel := range []string{signingScript, macSigning, releaseScript, draftNotes} {
		files[rel] = readRepoFile(t, root, rel)
	}
	brokenFiles := maps.Clone(files)
	brokenFiles[macSigning] = planted(t, macSigning, files[macSigning], "PIN=\"$(echo \"$2\"",
		"PIN=\""+pins["apple-developer-id"].digest+"\"\nX=\"$(echo \"$2\"")
	brokenFiles[signingScript] = planted(t, signingScript, files[signingScript],
		`PIN_FILE = os.path.join(".github", "signing-certificates.txt")`, `PIN_FILE = "pins.txt"`)
	found := checkPinCopies(pins, brokenFiles)
	requireFinding(t, found, macSigning+" carries the apple-developer-id digest itself")
	requireFinding(t, found, signingScript+" does not read "+signingPins)

	requireNone(t, checkPinCopies(pins, files), "the copies of the pinned digests")
}

type pin struct {
	digest  string
	expires time.Time
}

var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// parsePins reads the pin file: a name, a lowercase sha256 and a date on each
// line, both certificates, each once.
func parsePins(body string) (map[string]pin, []string) {
	pins := map[string]pin{}
	var problems []string
	for i, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		at := func(detail string) { problems = append(problems, fmt.Sprintf("%s:%d: %s", signingPins, i+1, detail)) }
		fields := strings.Fields(trimmed)
		if len(fields) != 3 {
			at(fmt.Sprintf("has %d fields where a pin has three: a name, a sha256, an expiry date", len(fields)))
			continue
		}
		name, digest, date := fields[0], fields[1], fields[2]
		if !slices.Contains(pinNames, name) {
			at("names " + name + ", which is not a certificate a release is signed with")
			continue
		}
		if _, twice := pins[name]; twice {
			at("names " + name + " twice")
			continue
		}
		if !sha256Hex.MatchString(digest) {
			at(name + " is not a lowercase sha256")
			continue
		}
		expires, err := time.Parse("2006-01-02", date)
		if err != nil {
			at(name + " expires on " + date + ", which is not a date")
			continue
		}
		pins[name] = pin{digest, expires}
	}
	for _, name := range pinNames {
		if _, ok := pins[name]; !ok {
			problems = append(problems, signingPins+": names no "+name)
		}
	}
	return pins, problems
}

// checkPinCopies reports a file of the release that carries a digest itself,
// in either case, and a signing script that does not read the pin file. The
// line that reads the file is asked for, not a mention of its name, because
// the script names the file in its messages too, and those would survive the
// reading going away.
func checkPinCopies(pins map[string]pin, files map[string]string) []string {
	var problems []string
	for rel, body := range files {
		lower := strings.ToLower(body)
		for name, p := range pins {
			if strings.Contains(lower, p.digest) {
				problems = append(problems, rel+" carries the "+name+" digest itself, a second copy that can drift from "+signingPins)
			}
		}
	}
	if !strings.Contains(files[signingScript], `PIN_FILE = os.path.join(".github", "signing-certificates.txt")`) {
		problems = append(problems, signingScript+" does not read "+signingPins+", so it carries its own idea of which certificate signs")
	}
	sort.Strings(problems)
	return problems
}

// TestTheSigningScriptSignsTheArchivesTheReleaseBuilds ties the signing
// script's table of archives to the targets the release builds, which another
// guard ties to the targets the notices are checked for. An archive the table
// does not name would be refused by the script as not part of a release on
// the day of the release, and a name the build does not make would be asked
// for and never found. It also holds the script to the repository it fetches
// from and uploads to, read from go.mod, and to the pin the software
// renderer's files are read from, the pin build_release.sh packs them by.
func TestTheSigningScriptSignsTheArchivesTheReleaseBuilds(t *testing.T) {
	root := rootOrFail(t)
	script := readRepoFile(t, root, signingScript)
	module, err := modulePathOf(readRepoFile(t, root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	repository := strings.TrimPrefix(module, "github.com/")

	broken := planted(t, signingScript, script, `("qadg", "windows/arm64"),`, `("qadg", "windows/386"),`)
	broken = planted(t, signingScript, broken, `("qadg-gui", "darwin/arm64"),`, `("qadg-gui", "darwin/arm64"),
    ("qadg-gui", "darwin/arm64"),`)
	broken = planted(t, signingScript, broken, `RENDERER_PIN = os.path.join(".github", "mesa-dist-win")`, `RENDERER_PIN = "mesa"`)
	broken = planted(t, signingScript, broken, `REPO = "`+repository+`"`, `REPO = "someone/else"`)
	found := checkSigningTable(broken, repository)
	requireFinding(t, found, "signs windows/386 of qadg, which the release does not build")
	requireFinding(t, found, "does not sign windows/arm64 of qadg, which the release builds")
	requireFinding(t, found, "names darwin/arm64 of qadg-gui twice")
	requireFinding(t, found, "does not read the software renderer's files from "+rendererPin)
	requireFinding(t, found, "does not name "+repository)
	if _, err := signingTable("no table here"); err == nil {
		t.Error("a script without a RELEASE table was read as one")
	}

	requireNone(t, checkSigningTable(script, repository), "what the signing script signs")
}

// signingRow is one line of the RELEASE table of the signing script.
var signingRow = regexp.MustCompile(`^\("([a-z-]+)", "([a-z0-9]+)/([a-z0-9]+)"\),$`)

// signingTable reads the RELEASE table out of the signing script: program,
// goos and goarch of each archive.
func signingTable(script string) ([][3]string, error) {
	opening := "\nRELEASE = (\n"
	start := strings.Index(script, opening)
	if start < 0 {
		return nil, fmt.Errorf("no RELEASE = ( table")
	}
	rest := script[start+len(opening):]
	end := strings.Index(rest, "\n)")
	if end < 0 {
		return nil, fmt.Errorf("the RELEASE table is never closed")
	}
	var rows [][3]string
	for _, line := range strings.Split(rest[:end], "\n") {
		m := signingRow.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			return nil, fmt.Errorf("the RELEASE table has a line that is not a program and a target: %q", line)
		}
		rows = append(rows, [3]string{m[1], m[2], m[3]})
	}
	return rows, nil
}

func checkSigningTable(script, repository string) []string {
	var problems []string
	at := func(detail string) { problems = append(problems, signingScript+": "+detail) }
	rows, err := signingTable(script)
	if err != nil {
		at(err.Error())
	}
	want := map[string][][2]string{"qadg": releaseTargets, "qadg-gui": windowTargets}
	seen := map[[3]string]bool{}
	for _, row := range rows {
		target := [2]string{row[1], row[2]}
		name := row[1] + "/" + row[2] + " of " + row[0]
		switch {
		case seen[row]:
			at("names " + name + " twice")
		case !slices.Contains(want[row[0]], target):
			at("signs " + name + ", which the release does not build")
		}
		seen[row] = true
	}
	for _, program := range []string{"qadg", "qadg-gui"} {
		for _, target := range want[program] {
			if !seen[[3]string{program, target[0], target[1]}] {
				at("does not sign " + target[0] + "/" + target[1] + " of " + program + ", which the release builds")
			}
		}
	}
	if !strings.Contains(script, `RENDERER_PIN = os.path.join(".github", "mesa-dist-win")`) {
		at("does not read the software renderer's files from " + rendererPin + ", so it signs by a list of its own")
	}
	if !strings.Contains(script, `REPO = "`+repository+`"`) {
		at("does not name " + repository + " as the repository it fetches from and uploads to")
	}
	sort.Strings(problems)
	return problems
}

// TestEveryFileNamesThePredicateTheDocumentIs ties the statement type to the
// document cmd/sbom writes. The attestation workflow checks it against the
// statement it made, but only on a release. This asks on every push, and asks
// the document too, so moving it to another SPDX version moves the type with
// it, in the note a person copies the command from as well.
func TestEveryFileNamesThePredicateTheDocumentIs(t *testing.T) {
	root := rootOrFail(t)
	files := map[string]string{}
	for _, rel := range []string{draftNotes, attestWorkflow, sbomDocument} {
		files[rel] = readRepoFile(t, root, rel)
	}
	broken := maps.Clone(files)
	broken[attestWorkflow] = planted(t, attestWorkflow, files[attestWorkflow], `promised="`+spdxPredicate+`"`, `promised="https://spdx.dev/Document/v2.2"`)
	broken[sbomDocument] = planted(t, sbomDocument, files[sbomDocument], `"SPDX-2.3"`, `"SPDX-3.0"`)
	found := checkPredicate(broken)
	requireFinding(t, found, attestWorkflow+" never names "+spdxPredicate)
	requireFinding(t, found, sbomDocument+` writes no "SPDX-2.3"`)

	requireNone(t, checkPredicate(files), "the predicate type")
}

// checkPredicate reports a file that does not name the predicate, and a
// document of another SPDX version than the predicate names.
func checkPredicate(files map[string]string) []string {
	var problems []string
	for _, rel := range []string{draftNotes, attestWorkflow} {
		if !strings.Contains(files[rel], spdxPredicate) {
			problems = append(problems, rel+" never names "+spdxPredicate)
		}
	}
	version := `"SPDX-` + spdxPredicate[strings.LastIndex(spdxPredicate, "/v")+2:] + `"`
	if !strings.Contains(files[sbomDocument], version) {
		problems = append(problems, sbomDocument+" writes no "+version+", the version "+spdxPredicate+" names")
	}
	sort.Strings(problems)
	return problems
}

// TestTheSigningLeavesTheReleaseADraft is the fence after the build. A person
// publishes a release, and nothing else: the attestation workflow starts only
// by hand, which is how the signing script starts it, and neither of them
// holds a command that publishes. Each asks whether the release is still a
// draft before it puts a file onto it, because a file added to a published
// release changes a release people may have downloaded.
func TestTheSigningLeavesTheReleaseADraft(t *testing.T) {
	root := rootOrFail(t)
	workflow := readRepoFile(t, root, attestWorkflow)
	script := readRepoFile(t, root, signingScript)

	brokenWorkflow := planted(t, attestWorkflow, workflow, "\non:\n  workflow_dispatch:\n", "\non:\n  push:\n    branches: [main]\n  workflow_dispatch:\n")
	brokenWorkflow = planted(t, attestWorkflow, brokenWorkflow, "--json isDraft", "--json name")
	brokenScript := planted(t, signingScript, script, `run(["gh", "workflow", "run", ATTEST_WORKFLOW,`,
		`run(["gh", "release", "edit", tag, "--draft=false"])
    run(["gh", "workflow", "run", ATTEST_WORKFLOW,`)
	found := checkDraftFence(brokenWorkflow, brokenScript)
	requireFinding(t, found, attestWorkflow+" starts on push, and it is started by hand alone")
	requireFinding(t, found, attestWorkflow+" never asks whether the release is still a draft")
	requireFinding(t, found, signingScript+` holds "gh", "release", "edit"`)

	requireNone(t, checkDraftFence(workflow, script), "the fence after the build")
}

// publishing are the ways gh publishes a release or makes one, each as the
// workflow writes it and as the script does.
var publishing = []string{
	"gh release edit", "gh release create", "--draft=false",
	`"gh", "release", "edit"`, `"gh", "release", "create"`, `"--draft=false"`,
}

// triggerLine is a key at two spaces in the on: block of a workflow.
var triggerLine = regexp.MustCompile(`^  ([a-z_]+):`)

func checkDraftFence(workflow, script string) []string {
	var problems []string
	start := strings.Index(workflow, "\non:\n")
	if start < 0 {
		problems = append(problems, attestWorkflow+" has no on: block")
	} else {
		for _, line := range strings.Split(workflow[start+len("\non:\n"):], "\n") {
			if line != "" && !strings.HasPrefix(line, " ") {
				break
			}
			if m := triggerLine.FindStringSubmatch(line); m != nil && m[1] != "workflow_dispatch" {
				problems = append(problems, attestWorkflow+" starts on "+m[1]+", and it is started by hand alone")
			}
		}
	}
	for rel, body := range map[string]string{attestWorkflow: workflow, signingScript: script} {
		for _, command := range publishing {
			if strings.Contains(body, command) {
				problems = append(problems, rel+" holds "+command+", and nothing after the build publishes")
			}
		}
	}
	if !strings.Contains(workflow, "--json isDraft") {
		problems = append(problems, attestWorkflow+" never asks whether the release is still a draft")
	}
	if !strings.Contains(script, `"--json", "assets,isDraft"`) {
		problems = append(problems, signingScript+" never asks whether the release is still a draft")
	}
	sort.Strings(problems)
	return problems
}
