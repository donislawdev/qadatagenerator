package guard

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// # What this defends (B-419 (f))
//
// verify-release.yml checks a release after it is published, by downloading
// what a person downloads and doing what the release notes tell them to do. It
// runs on a published release and nowhere else, so a defect in it surfaces on
// the day of a release, which is the day it has to be believed. Two of its
// defects say nothing at all: a check that stops the run hides every check
// behind it, and a check the verdict never reads turns a red line into no
// line. A verifier that can write can change what it checks. A file of a
// release it does not ask for is louder, and found only on that day. These
// guards read the file on every push instead.
//
// Every test that reads the real files first plants one fault in a copy of
// them and requires its check to report it (planted, in release_test.go).
//
// Adapted on 2026-11-23 from the release guards of Cert Doctor, a project of
// the same owner under the same licence.
const (
	verifyWorkflow = ".github/workflows/verify-release.yml"
	rendererCheck  = ".github/scripts/check_renderer.py"
)

// TestTheVerifierOnlyReads holds the verifier to reading. It starts on a
// published release and by hand, it is read only by default and no job asks
// for more, and it holds no command that makes, changes or removes a release.
func TestTheVerifierOnlyReads(t *testing.T) {
	root := rootOrFail(t)
	verify := readRepoFile(t, root, verifyWorkflow)

	broken := planted(t, verifyWorkflow, verify, "\npermissions:\n  contents: read\n", "\npermissions:\n  contents: write\n")
	broken = planted(t, verifyWorkflow, broken, "    types: [published]\n", "    types: [created]\n")
	broken = planted(t, verifyWorkflow, broken, "\n  workflow_dispatch:\n", "\n  push:\n  workflow_dispatch:\n")
	download := `gh release download "$TAG" --repo "$GITHUB_REPOSITORY"` + "\n"
	broken = planted(t, verifyWorkflow, broken, download, download+`          gh release upload "$TAG" extra.txt`+"\n")
	found := checkVerifierReads(broken)
	requireFinding(t, found, "is not read only by default")
	requireFinding(t, found, "can write")
	requireFinding(t, found, "does not start on a published release")
	requireFinding(t, found, "starts on push")
	requireFinding(t, found, "holds gh release upload")

	requireNone(t, checkVerifierReads(verify), "a verifier that only reads")
}

// changingARelease are the commands of gh that make, change or remove a
// release, beside those that publish one.
var changingARelease = slices.Concat(publishing, []string{"gh release upload", "gh release delete"})

func checkVerifierReads(verify string) []string {
	var problems []string
	if !strings.Contains(verify, "\npermissions:\n  contents: read\n") {
		problems = append(problems, verifyWorkflow+" is not read only by default")
	}
	if writePermission.MatchString(verify) {
		problems = append(problems, verifyWorkflow+" can write, and a verifier that can publish is not a verifier")
	}
	if !strings.Contains(verify, "\non:\n  release:\n    types: [published]\n") {
		problems = append(problems, verifyWorkflow+" does not start on a published release, and only a published release is what people download")
	}
	if start := strings.Index(verify, "\non:\n"); start >= 0 {
		for _, line := range strings.Split(verify[start+len("\non:\n"):], "\n") {
			if line != "" && !strings.HasPrefix(line, " ") {
				break
			}
			if m := triggerLine.FindStringSubmatch(line); m != nil && m[1] != "release" && m[1] != "workflow_dispatch" {
				problems = append(problems, verifyWorkflow+" starts on "+m[1]+", and it checks a published release")
			}
		}
	}
	for _, command := range changingARelease {
		if strings.Contains(verify, command) {
			problems = append(problems, verifyWorkflow+" holds "+command+", and a verifier changes nothing")
		}
	}
	sort.Strings(problems)
	return problems
}

// TestNoReleaseCheckCanHideTheOnesBehindIt holds the verdict of the verifier
// to every check. Each check carries continue-on-error, so a failing one does
// not stop the ones after it, and the last step reads every check's outcome
// and fails when any is not a success. Outcome and never conclusion, because
// continue-on-error rewrites conclusion to success, so a verdict reading it
// would call every check passed however it went.
func TestNoReleaseCheckCanHideTheOnesBehindIt(t *testing.T) {
	root := rootOrFail(t)
	verify := readRepoFile(t, root, verifyWorkflow)

	broken := planted(t, verifyWorkflow, verify, "        id: renderer\n        continue-on-error: true\n", "        id: renderer\n")
	broken = planted(t, verifyWorkflow, broken, "          INSTALLER: ${{ steps.installer.outcome }}\n", "")
	broken = planted(t, verifyWorkflow, broken, "${{ steps.latest.outcome }}", "${{ steps.latest.conclusion }}")
	_, found := checkVerdict(broken)
	requireFinding(t, found, "step renderer stops the run when it fails")
	requireFinding(t, found, "the verdict never reads step installer")
	requireFinding(t, found, "reads steps.latest.conclusion")

	checked, problems := checkVerdict(verify)
	requireNone(t, problems, "the verdict of the verifier")
	if checked < 7 {
		t.Errorf("only %d checks feed the verdict, so this is not reading the file it means to", checked)
	}
}

var (
	// verifyStep is the first line of a step.
	verifyStep = regexp.MustCompile(`^\s+- (?:name|uses):`)
	// verifyStepID is the id a step is read by.
	verifyStepID = regexp.MustCompile(`(?m)^\s+id:\s*([A-Za-z0-9_-]+)\s*$`)
	// conclusionRead is a step's conclusion read anywhere in the file.
	conclusionRead = regexp.MustCompile(`steps\.([A-Za-z0-9_-]+)\.conclusion`)
)

// checkVerdict reports a check that can hide the ones behind it, or that the
// verdict never reads, and returns how many checks it read. which is the step
// that names the tag, not a check.
func checkVerdict(verify string) (int, []string) {
	var steps []string
	for _, line := range strings.Split(verify, "\n") {
		if verifyStep.MatchString(line) || len(steps) == 0 {
			steps = append(steps, "")
		}
		steps[len(steps)-1] += line + "\n"
	}
	var problems []string
	checked := 0
	for _, step := range steps {
		m := verifyStepID.FindStringSubmatch(step)
		if m == nil || m[1] == "which" {
			continue
		}
		checked++
		if !strings.Contains(step, "continue-on-error: true") {
			problems = append(problems, verifyWorkflow+": step "+m[1]+" stops the run when it fails, and hides every check behind it")
		}
		if !strings.Contains(verify, "${{ steps."+m[1]+".outcome }}") {
			problems = append(problems, verifyWorkflow+": the verdict never reads step "+m[1])
		}
	}
	for _, m := range conclusionRead.FindAllStringSubmatch(verify, -1) {
		problems = append(problems, verifyWorkflow+": reads steps."+m[1]+".conclusion, which continue-on-error turns into success")
	}
	sort.Strings(problems)
	return checked, problems
}

// TestTheVerifierChecksWhatTheSigningScriptMakes ties the verifier to the
// signing script, which another guard ties to what the release builds. Every
// archive the script signs or passes on and every file it names with the
// verify- prefix is a file the verifier asks for by name. In each Windows
// archive the verifier expects the programs and libraries the script signs
// there. It reads both certificates and the software renderer's files from
// their pins, the files the script reads, and hands the renderer check the
// window's archive and that pin. A file the verifier does not ask for would be
// refused as not part of a release, on the day of a release.
func TestTheVerifierChecksWhatTheSigningScriptMakes(t *testing.T) {
	root := rootOrFail(t)
	files := map[string]string{}
	for _, rel := range []string{verifyWorkflow, signingScript} {
		files[rel] = readRepoFile(t, root, rel)
	}
	broken := maps.Clone(files)
	broken[verifyWorkflow] = planted(t, verifyWorkflow, files[verifyWorkflow], `            "qadg_${version}_linux_arm64.tar.gz"`+"\n", "")
	broken[verifyWorkflow] = planted(t, verifyWorkflow, broken[verifyWorkflow], `= @('qadg-gui.exe', 'qadg.exe') + $renderer`, `= @('qadg-gui.exe', 'qadg.exe')`)
	broken[verifyWorkflow] = planted(t, verifyWorkflow, broken[verifyWorkflow], `"verify-qadg_${version}.sbom.sigstore.json"`+"\n          )", `"verify-qadg_${version}.sbom.json"`+"\n          )")
	broken[verifyWorkflow] = planted(t, verifyWorkflow, broken[verifyWorkflow], `awk '$1 == "apple-developer-id" { print $2 }' .github/signing-certificates.txt`, `echo pinned`)
	broken[verifyWorkflow] = planted(t, verifyWorkflow, broken[verifyWorkflow], `_windows_amd64.zip" .github/mesa-dist-win`, `_windows_amd64.zip" mesa`)
	found := checkVerifierFiles(broken)
	requireFinding(t, found, `does not ask for "qadg_${version}_linux_arm64.tar.gz"`)
	requireFinding(t, found, `does not expect qadg-gui_${version}_windows_amd64.zip to hold what the signing script signs in it`)
	requireFinding(t, found, `does not ask for "verify-qadg_${version}.sbom.sigstore.json"`)
	requireFinding(t, found, "does not read the apple-developer-id pin")
	requireFinding(t, found, "does not hand "+rendererCheck+" the window's archive and "+rendererPin)

	requireNone(t, checkVerifierFiles(files), "what the verifier asks for")
}

// signedInsideRow is one line of the SIGNED_INSIDE table of the signing script.
var signedInsideRow = regexp.MustCompile(`^"([a-z-]+)": \(([^)]*)\),$`)

// signedInside reads the SIGNED_INSIDE table: for each program, what the card
// signs in its Windows archive, RENDERER standing for the renderer's files.
func signedInside(script string) (map[string][]string, error) {
	opening := "\nSIGNED_INSIDE = {\n"
	start := strings.Index(script, opening)
	if start < 0 {
		return nil, fmt.Errorf("no SIGNED_INSIDE = { table")
	}
	rest := script[start+len(opening):]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		return nil, fmt.Errorf("the SIGNED_INSIDE table is never closed")
	}
	table := map[string][]string{}
	for _, line := range strings.Split(rest[:end], "\n") {
		m := signedInsideRow.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			return nil, fmt.Errorf("the SIGNED_INSIDE table has a line that is not a program and its files: %q", line)
		}
		for _, item := range strings.Split(m[2], ",") {
			if item = strings.TrimSpace(item); item != "" {
				table[m[1]] = append(table[m[1]], strings.Trim(item, `"`))
			}
		}
	}
	return table, nil
}

// powershellList is how the verifier writes what an archive holds: the names
// in quotes, and the renderer's files added when the script signs them.
func powershellList(items []string) string {
	var quoted []string
	renderer := false
	for _, item := range items {
		if item == "RENDERER" {
			renderer = true
			continue
		}
		quoted = append(quoted, "'"+item+"'")
	}
	list := "@(" + strings.Join(quoted, ", ") + ")"
	if renderer {
		list += " + $renderer"
	}
	return list
}

// expectedFiles is the list of the files of a release the verifier compares
// the published ones with, the expected=( ) array of its step, or "". Asked of
// that list rather than the whole file, because the same names stand in its
// other steps too, and those would survive a name dropping out of the list.
func expectedFiles(verify string) string {
	_, after, found := strings.Cut(verify, "\n          expected=(\n")
	if !found {
		return ""
	}
	list, _, closed := strings.Cut(after, "\n          )\n")
	if !closed {
		return ""
	}
	return "\n" + list + "\n"
}

func checkVerifierFiles(files map[string]string) []string {
	verify, script := files[verifyWorkflow], files[signingScript]
	var problems []string
	at := func(detail string) { problems = append(problems, verifyWorkflow+" "+detail) }
	expected := expectedFiles(verify)
	if expected == "" {
		at("has no expected=( ) list of the files of a release")
	}
	asks := func(name string) bool {
		return strings.Contains(expected, "\n            \""+name+"\"\n")
	}

	rows, err := signingTable(script)
	if err != nil {
		problems = append(problems, signingScript+": "+err.Error())
	}
	inside, err := signedInside(script)
	if err != nil {
		problems = append(problems, signingScript+": "+err.Error())
	}
	for _, row := range rows {
		program, goos, goarch := row[0], row[1], row[2]
		system, ending := goos, ".tar.gz"
		if goos == "darwin" {
			system = "macos"
		}
		if goos == "windows" {
			ending = ".zip"
		}
		name := program + "_${version}_" + system + "_" + goarch + ending
		if !asks(name) {
			at(`does not ask for "` + name + `", an archive the signing script makes`)
		}
		if goos != "windows" {
			continue
		}
		held := regexp.MustCompile(`"` + regexp.QuoteMeta(name) + `"\s+= ` + regexp.QuoteMeta(powershellList(inside[program])) + `\n`)
		if !held.MatchString(verify) {
			at("does not expect " + name + " to hold what the signing script signs in it, " + powershellList(inside[program]))
		}
	}

	// The files a person checks a download with, named by the signing script
	// with its prefix and by the verifier with the prefix written out.
	if !strings.Contains(script, `AUX_PREFIX = "verify-"`) || !strings.Contains(script, `SUMS = AUX_PREFIX + "SHA256SUMS.txt"`) {
		problems = append(problems, signingScript+" no longer names its files verify- and its checksums verify-SHA256SUMS.txt, which the verifier asks for")
	}
	if !asks("verify-SHA256SUMS.txt") {
		at(`does not ask for "verify-SHA256SUMS.txt"`)
	}
	for _, end := range []string{".spdx.json", ".provenance.sigstore.json", ".sbom.sigstore.json"} {
		if !strings.Contains(script, `"%sqadg_%s`+end+`"`) {
			problems = append(problems, signingScript+" no longer names a file qadg_<version>"+end)
		}
		if !asks("verify-qadg_${version}" + end) {
			at(`does not ask for "verify-qadg_${version}` + end + `"`)
		}
	}

	// The pins, read from the files the signing script reads.
	if !strings.Contains(verify, `awk '$1 == "apple-developer-id" { print $2 }' .github/signing-certificates.txt`) {
		at("does not read the apple-developer-id pin from " + signingPins)
	}
	if !strings.Contains(verify, `Select-String -Path ../.github/signing-certificates.txt -Pattern '^windows-code-signing\s+([0-9a-f]{64})\s'`) {
		at("does not read the windows-code-signing pin from " + signingPins)
	}
	if !strings.Contains(verify, `Select-String -Path ../.github/mesa-dist-win -Pattern '^file=`) {
		at("does not read the software renderer's files from " + rendererPin)
	}
	if !strings.Contains(verify, `python `+rendererCheck+` "published/qadg-gui_${TAG#v}_windows_amd64.zip" `+rendererPin+"\n") {
		at("does not hand " + rendererCheck + " the window's archive and " + rendererPin)
	}
	sort.Strings(problems)
	return problems
}
