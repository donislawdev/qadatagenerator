package guard

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// # What this defends (B-419 (c), DEC-329)
//
// The release is a workflow and three scripts that run on a tag, and a tag is
// the one moment a defect in them is most expensive: a version spent, a draft
// half made, a person waiting to sign. A push to main that changes them runs
// the workflow by hand, but not every defect shows on such a run, because the
// checks that compare the tag with the code and the job that writes are fenced
// behind the tag. These guards read the files on every push instead, and each
// asks one thing that, if it went wrong, would surface only then.
//
// Every test that reads the real files first plants one fault in a copy of
// them and requires its check to report it, so a check that reads nothing, or
// a file whose shape moved away from what the check reads, fails rather than
// passes.
//
// Adapted on 2026-11-18 from the release guards of Cert Doctor, a project of
// the same owner under the same licence.
const (
	releaseWorkflow = ".github/workflows/release.yml"
	releaseScript   = ".github/scripts/build_release.sh"
	windowScript    = ".github/scripts/build_window.sh"
	bundleScript    = ".github/scripts/make_app_bundle.sh"
	spdxValidator   = ".github/requirements-spdx.txt"
	draftNotes      = ".github/release-notes.md"
	workflowsDir    = ".github/workflows"
	windowResources = "cmd/qadg-gui/qadg-gui.rc"
	siteWorkflow    = ".github/workflows/site.yml"
)

// The jobs of the release workflow the checks below name.
const (
	windowJob  = "window"
	buildJob   = "build"
	publishJob = "publish"
)

// tagFence is the condition that keeps a job to a pushed tag.
const tagFence = "if: startsWith(github.ref, 'refs/tags/')"

// planted applies one replacement to a file and fails the test when the text
// to replace is not there, because a canary that changes nothing proves
// nothing, and the file has then moved away from what the check reads.
func planted(t *testing.T, rel, body, old, replacement string) string {
	t.Helper()
	if !strings.Contains(body, old) {
		t.Fatalf("%s no longer holds %q, so the fault this test plants in it cannot be planted. "+
			"Plant it where the file now says the same thing", rel, old)
	}
	return strings.Replace(body, old, replacement, 1)
}

// TestTheReleaseBuildsTheTargetsTheNoticesAreCheckedFor ties three lists to
// one decision. The notices guard asks go list what each of releaseTargets and
// windowTargets links, build_release.sh builds and packs its own two lists, and
// the window job of the release workflow builds a third. A target added to the
// script alone would ship notices nobody checked against what it links, and a
// window packed for a system no job builds stops the release at its end.
func TestTheReleaseBuildsTheTargetsTheNoticesAreCheckedFor(t *testing.T) {
	root := rootOrFail(t)
	script := readRepoFile(t, root, releaseScript)
	workflow := readRepoFile(t, root, releaseWorkflow)

	brokenScript := planted(t, releaseScript, script, "\n  windows/arm64\n", "\n  windows/386\n")
	brokenWorkflow := planted(t, releaseWorkflow, workflow, "goos: darwin\n            goarch: arm64", "goos: darwin\n            goarch: amd64")
	found := checkReleaseTargets(brokenScript, brokenWorkflow)
	requireFinding(t, found, "builds windows/386, which is not a release target")
	requireFinding(t, found, "does not build windows/arm64, a release target")
	requireFinding(t, found, "builds darwin/amd64, which is not a window target")

	requireNone(t, checkReleaseTargets(script, workflow), "the targets of the release")
}

func checkReleaseTargets(script, workflow string) []string {
	var problems []string
	for _, each := range []struct {
		list, what string
		want       [][2]string
	}{
		{"targets", "release target", releaseTargets},
		{"window_targets", "window target", windowTargets},
	} {
		got, err := scriptList(script, each.list)
		if err != nil {
			problems = append(problems, releaseScript+": "+err.Error())
			continue
		}
		problems = append(problems, compareTargets(releaseScript, got, each.want, each.what)...)
	}
	matrix, err := jobTargets(workflow, windowJob)
	if err != nil {
		problems = append(problems, releaseWorkflow+": "+err.Error())
	} else {
		problems = append(problems, compareTargets(releaseWorkflow, matrix, windowTargets, "window target")...)
	}
	return problems
}

// scriptList reads the list called name out of a shell script, one
// goos/goarch on a line between "name=(" and ")".
func scriptList(body, name string) ([][2]string, error) {
	body = "\n" + body
	opening := "\n" + name + "=(\n"
	start := strings.Index(body, opening)
	if start < 0 {
		return nil, fmt.Errorf("no %s=( list", name)
	}
	rest := body[start+len(opening):]
	end := strings.Index(rest, "\n)")
	if end < 0 {
		return nil, fmt.Errorf("the %s=( list is never closed", name)
	}
	var targets [][2]string
	for _, line := range strings.Split(rest[:end], "\n") {
		if goos, goarch, ok := strings.Cut(strings.TrimSpace(line), "/"); ok {
			targets = append(targets, [2]string{goos, goarch})
		}
	}
	return targets, nil
}

var (
	jobStart    = regexp.MustCompile(`^  ([A-Za-z0-9_-]+):\s*$`)
	matrixValue = regexp.MustCompile(`^\s+(?:- )?(goos|goarch):\s*(\S+)\s*$`)
)

// jobTargets reads the goos and goarch pairs of one job's matrix.
func jobTargets(workflow, job string) ([][2]string, error) {
	block, ok := jobBlocks(workflow)[job]
	if !ok {
		return nil, fmt.Errorf("no %s job", job)
	}
	var targets [][2]string
	for _, line := range strings.Split(block, "\n") {
		m := matrixValue.FindStringSubmatch(line)
		switch {
		case m == nil:
		case m[1] == "goos":
			targets = append(targets, [2]string{m[2], ""})
		case len(targets) > 0:
			targets[len(targets)-1][1] = m[2]
		}
	}
	return targets, nil
}

// jobBlocks splits a workflow into its jobs, by name.
func jobBlocks(workflow string) map[string]string {
	jobs := map[string]string{}
	_, after, found := strings.Cut(workflow, "\njobs:\n")
	if !found {
		return jobs
	}
	name := ""
	for _, line := range strings.Split(after, "\n") {
		if m := jobStart.FindStringSubmatch(line); m != nil {
			name = m[1]
		}
		if name != "" {
			jobs[name] += line + "\n"
		}
	}
	return jobs
}

// compareTargets compares one list of builds with the targets a decision
// ships.
func compareTargets(rel string, got, want [][2]string, what string) []string {
	wanted := map[[2]string]bool{}
	for _, target := range want {
		wanted[target] = true
	}
	seen := map[[2]string]bool{}
	var problems []string
	for _, target := range got {
		name := target[0] + "/" + target[1]
		switch {
		case seen[target]:
			problems = append(problems, rel+": builds "+name+" twice")
		case !wanted[target]:
			problems = append(problems, rel+": builds "+name+", which is not a "+what+" the notices are checked for")
		}
		seen[target] = true
	}
	for _, target := range want {
		if !seen[target] {
			problems = append(problems, rel+": does not build "+target[0]+"/"+target[1]+", a "+what)
		}
	}
	sort.Strings(problems)
	return problems
}

// TestTheWindowsAndTheirRendererReachTheArchives holds the hand over from the
// window job to the archives, and the software renderer to one copy of its
// notices. The window job runs build_window.sh and keeps what it left under
// the name build_release.sh looks for, the build job waits for it, fetches it
// and calls build_release.sh with the third argument, without which the
// script still succeeds, packs the command line alone and says so where
// nobody is reading. build_release.sh then holds the notices in the folder the
// window job made to .github/opengl of the commit, and hands the same folder
// to the bill of materials, which is the folder fetch_software_renderer.sh
// copies them from.
func TestTheWindowsAndTheirRendererReachTheArchives(t *testing.T) {
	root := rootOrFail(t)
	workflow := readRepoFile(t, root, releaseWorkflow)
	script := readRepoFile(t, root, releaseScript)
	fetcher := readRepoFile(t, root, rendererFetcher)

	call := `"$RUNNER_TEMP/release" "$RUNNER_TEMP/windows"`
	brokenWorkflow := planted(t, releaseWorkflow, workflow, call, `"$RUNNER_TEMP/release"`)
	requireFinding(t, checkWindowHandover(brokenWorkflow, script, fetcher), "hands build_release.sh 2 argument(s)")
	brokenScript := planted(t, releaseScript, script, `-renderer "$renderer_notices"`, `-renderer .github/elsewhere`)
	requireFinding(t, checkWindowHandover(workflow, brokenScript, fetcher), "does not hand the bill of materials the renderer's notices")

	requireNone(t, checkWindowHandover(workflow, script, fetcher), "the windows reaching the archives")
}

// releaseScriptCall is build_release.sh called with its arguments, each in
// double quotes, as the build job writes it.
var releaseScriptCall = regexp.MustCompile(`build_release\.sh((?:\s+"[^"]*")*)\s*$`)

func checkWindowHandover(workflow, script, fetcher string) []string {
	var problems []string
	at := func(rel, detail string) { problems = append(problems, rel+": "+detail) }
	jobs := jobBlocks(workflow)
	window, build := jobs[windowJob], jobs[buildJob]
	if window == "" {
		at(releaseWorkflow, "has no "+windowJob+" job, so no release carries the window")
	} else {
		if !strings.Contains(window, "bash "+windowScript+" ") {
			at(releaseWorkflow, "the "+windowJob+" job never runs "+windowScript)
		}
		if !strings.Contains(window, "name: qadg-gui_${{ matrix.goos }}_${{ matrix.goarch }}") {
			at(releaseWorkflow, "the "+windowJob+" job keeps the window under another name than qadg-gui_<goos>_<goarch>, which build_release.sh looks for")
		}
	}
	if !regexp.MustCompile(`(?m)^\s+needs:\s*\[[^\]]*\b` + windowJob + `\b[^\]]*\]\s*$`).MatchString(build) {
		at(releaseWorkflow, "the "+buildJob+" job does not wait for the "+windowJob+" job")
	}
	if !strings.Contains(build, "pattern: qadg-gui_*") {
		at(releaseWorkflow, "the "+buildJob+" job never fetches the windows")
	}
	calls := 0
	for _, line := range strings.Split(build, "\n") {
		m := releaseScriptCall.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		calls++
		if n := strings.Count(m[1], `"`) / 2; n != 3 {
			at(releaseWorkflow, fmt.Sprintf("the %s job hands build_release.sh %d argument(s), and without the third, the windows, it packs the command line alone", buildJob, n))
		}
	}
	if calls != 1 {
		at(releaseWorkflow, fmt.Sprintf("the %s job calls build_release.sh %d time(s), and a release is packed by one call", buildJob, calls))
	}

	for _, want := range []struct{ sign, why string }{
		{`from="$windows/qadg-gui_${os}_${arch}"`, "does not read the windows under the name the window job keeps them"},
		{`renderer_notices="` + rendererNotices + `"`, "does not take the renderer's notices from " + rendererNotices},
		{`-renderer "$renderer_notices"`, "does not hand the bill of materials the renderer's notices it packs"},
		{`cmp -s "$to/opengl/$name" "$renderer_notices/$name"`, "does not hold the notices the window job put beside the renderer to the ones the bill of materials reads"},
	} {
		if !strings.Contains(script, want.sign) {
			at(releaseScript, want.why)
		}
	}
	if !strings.Contains(fetcher, `notices="`+rendererNotices+`"`) {
		at(rendererFetcher, "does not copy the renderer's notices from "+rendererNotices+", the folder the bill of materials reads")
	}
	sort.Strings(problems)
	return problems
}

// TestEveryWorkflowThatCanWritePinsItsActionsToACommit holds a workflow that
// can write to this repository or publish from it to naming each action by the
// commit it runs, with the release it is beside it. A tag can be moved to
// other code by whoever owns the action, and that code would hold the token.
// The workflows that only read name their actions by a release, and the
// difference is the token.
func TestEveryWorkflowThatCanWritePinsItsActionsToACommit(t *testing.T) {
	root := rootOrFail(t)
	workflows := workflowFiles(t, root)
	release := workflows[releaseWorkflow]
	broken := planted(t, releaseWorkflow, release, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1", "actions/checkout@v7")
	requireFinding(t, checkPinnedActions(releaseWorkflow, broken), "actions/checkout@v7 is not pinned to a commit")

	var problems []string
	writers := 0
	for _, rel := range slices.Sorted(maps.Keys(workflows)) {
		if !writePermission.MatchString(workflows[rel]) {
			continue
		}
		writers++
		problems = append(problems, checkPinnedActions(rel, workflows[rel])...)
	}
	requireNone(t, problems, "actions pinned to a commit")
	for _, rel := range []string{releaseWorkflow, siteWorkflow} {
		if !writePermission.MatchString(workflows[rel]) {
			t.Errorf("%s no longer asks for a permission to write, so this guard no longer reads it. Read why, then change this list", rel)
		}
	}
	if writers < 2 {
		t.Errorf("only %d workflow(s) can write, so this read fewer files than it means to", writers)
	}
}

var (
	// writePermission is a permission granted to write, the line and not a
	// comment that happens to hold the word.
	writePermission = regexp.MustCompile(`(?m)^\s+[a-z-]+:\s*write\s*(?:#.*)?$`)

	usesLine   = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*(\S+)(.*)$`)
	pinnedUses = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+@[0-9a-f]{40}$`)
	releaseTag = regexp.MustCompile(`^\s+#\s+v[0-9]`)
)

// checkPinnedActions reports every action named by anything but a commit with
// its release written beside it. An action of this repository, ./ and a path,
// is the code of the commit being run and is pinned by it.
func checkPinnedActions(rel, body string) []string {
	var problems []string
	for i, line := range strings.Split(body, "\n") {
		m := usesLine.FindStringSubmatch(line)
		if m == nil || strings.HasPrefix(m[1], "./") {
			continue
		}
		switch {
		case !pinnedUses.MatchString(m[1]):
			problems = append(problems, fmt.Sprintf("%s:%d: %s is not pinned to a commit", rel, i+1, m[1]))
		case !releaseTag.MatchString(m[2]):
			problems = append(problems, fmt.Sprintf("%s:%d: %s does not say which release that commit is", rel, i+1, m[1]))
		}
	}
	return problems
}

// workflowFiles reads every workflow, by path from the root.
func workflowFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(workflowsDir)))
	if err != nil {
		t.Fatalf("cannot list the workflows: %v", err)
	}
	out := map[string]string{}
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yml") || strings.HasSuffix(entry.Name(), ".yaml")) {
			rel := workflowsDir + "/" + entry.Name()
			out[rel] = readRepoFile(t, root, rel)
		}
	}
	if _, ok := out[releaseWorkflow]; !ok {
		t.Fatalf("the workflows read do not include %s, so this proves nothing about it", releaseWorkflow)
	}
	return out
}

// TestAScriptAWorkflowRunsIsHandedToItsInterpreter keeps the release off the
// executable bit. A file committed from Windows is recorded without it, and a
// workflow that runs such a script by its path stops with "permission denied",
// on the tag. Handed to bash or python, the bit does not matter. A script a
// workflow or a release script names also has to be in the tree.
func TestAScriptAWorkflowRunsIsHandedToItsInterpreter(t *testing.T) {
	root := rootOrFail(t)
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		return err == nil
	}
	files := map[string]string{}
	for rel, body := range workflowFiles(t, root) {
		files[rel] = body
	}
	for _, rel := range []string{releaseScript, windowScript} {
		files[rel] = readRepoFile(t, root, rel)
	}

	broken := planted(t, releaseWorkflow, files[releaseWorkflow], "run: bash "+releaseScript, "run: "+releaseScript)
	_, plantedProblems := checkScriptCalls(releaseWorkflow, broken, exists)
	requireFinding(t, plantedProblems, "runs "+releaseScript+" by its path")
	_, plantedProblems = checkScriptCalls(releaseWorkflow, files[releaseWorkflow]+"run: bash .github/scripts/gone.sh\n", exists)
	requireFinding(t, plantedProblems, "names .github/scripts/gone.sh, which is not in the tree")

	var problems []string
	calls := 0
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		found, p := checkScriptCalls(rel, files[rel], exists)
		calls += found
		problems = append(problems, p...)
	}
	requireNone(t, problems, "scripts handed to their interpreter")
	if calls < 5 {
		t.Errorf("only %d script(s) are handed to an interpreter anywhere, so this is not reading the files it means to", calls)
	}
}

var scriptPath = regexp.MustCompile(`\.github/scripts/[A-Za-z0-9_.-]+`)

// checkScriptCalls reports a script run by its path rather than handed to an
// interpreter, and a script named that is not there. A script named after
// anything else, cp or a list of paths, is not run, and only its existence is
// asked. It returns how many scripts it read handed to an interpreter.
func checkScriptCalls(rel, body string, exists func(string) bool) (int, []string) {
	var problems []string
	calls := 0
	for i, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, at := range scriptPath.FindAllStringIndex(line, -1) {
			script := line[at[0]:at[1]]
			before := strings.TrimRight(strings.TrimSuffix(line[:at[0]], "./"), ` "'`)
			fields := strings.Fields(before)
			last := ""
			if len(fields) > 0 {
				last = fields[len(fields)-1]
			}
			switch last {
			case "bash", "python", "python3":
				calls++
			case "", "run:", "&&", "||", "$(", "then", "do":
				problems = append(problems, fmt.Sprintf("%s:%d: runs %s by its path, which needs the executable bit git does not record from Windows. Hand it to bash or python", rel, i+1, script))
			}
			if !exists(script) {
				problems = append(problems, fmt.Sprintf("%s:%d: names %s, which is not in the tree", rel, i+1, script))
			}
		}
	}
	return calls, problems
}

// TestTheReleaseWorkflowOpensADraftAndPublishesNothing is the fence. A tag
// starts a release and a person finishes it: the build writes nothing but an
// empty draft, the only job holding a token that can write is behind the tag,
// and a tagged tree has to pass the three checks before anything is built. The
// draft opens with a note that tells a person how to check a download, for
// this repository as go.mod names it.
func TestTheReleaseWorkflowOpensADraftAndPublishesNothing(t *testing.T) {
	root := rootOrFail(t)
	workflow := readRepoFile(t, root, releaseWorkflow)
	notes := readRepoFile(t, root, draftNotes)
	module, err := modulePathOf(readRepoFile(t, root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	repository := strings.TrimPrefix(module, "github.com/")

	// Four spaces in, the condition of a job. The steps that share it are
	// eight spaces in.
	broken := planted(t, releaseWorkflow, workflow, "\n    "+tagFence+"\n", "\n")
	requireFinding(t, checkReleaseFences(broken, notes, repository), "job publish can write and is not fenced behind a tag")
	silent := checkReleaseFences(workflow, "", repository)
	requireFinding(t, silent, "never mentions gh attestation verify")
	requireFinding(t, silent, "never mentions --predicate-type "+spdxPredicate)
	requireFinding(t, silent, "never mentions verify-SHA256SUMS.txt")

	requireNone(t, checkReleaseFences(workflow, notes, repository), "the release fences")
}

func modulePathOf(gomod string) (string, error) {
	for _, line := range strings.Split(gomod, "\n") {
		if path, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(path), nil
		}
	}
	return "", fmt.Errorf("go.mod names no module")
}

func checkReleaseFences(workflow, notes, repository string) []string {
	var problems []string
	at := func(rel, detail string) { problems = append(problems, rel+": "+detail) }
	if !strings.Contains(workflow, "\npermissions:\n  contents: read\n") {
		at(releaseWorkflow, "is not read only by default")
	}
	for name, job := range jobBlocks(workflow) {
		if writePermission.MatchString(job) && !strings.Contains(job, tagFence) {
			at(releaseWorkflow, "job "+name+" can write and is not fenced behind a tag")
		}
	}
	if strings.Contains(workflow, "gh release upload") {
		at(releaseWorkflow, "uploads to a release, and the build is not what a person downloads")
	}
	if !strings.Contains(workflow, "gh release create") || !strings.Contains(workflow, "--draft") {
		at(releaseWorkflow, "opens a release that is not a draft")
	}
	for promise, sign := range map[string]string{
		"never asks whether ci passed on the commit":              "--workflow=ci.yml",
		"never asks whether the changelog is closed":              `sed -n '/^## \[Unreleased\]/`,
		"never asks whether the tag is the version the code says": "GITHUB_REF_NAME#v",
		"does not open the draft with its note":                   "--notes-file " + draftNotes,
	} {
		if !strings.Contains(workflow, sign) {
			at(releaseWorkflow, promise)
		}
	}
	// What a person landing on the release page has to be told: the command
	// that checks a download for this repository, the statement type the
	// statement over the signed files really carries, and the checksum file
	// the signing script writes.
	for _, promised := range []string{"gh attestation verify", "-R " + repository, "--predicate-type " + spdxPredicate, "verify-SHA256SUMS.txt"} {
		if !strings.Contains(notes, promised) {
			at(draftNotes, "never mentions "+promised)
		}
	}
	sort.Strings(problems)
	return problems
}

// TestTheBillOfMaterialsIsReadBySPDXsOwnValidator keeps the second reading of
// the bill of materials, and keeps what it installs from widening what the
// release trusts. The build job leaves what publish attests, so a package it
// installs by name alone is code nobody read, running where the release is
// made, the reason the actions there are pinned to a commit. pip holds the
// hashes itself, but only when the workflow runs, and this one runs on a tag.
// A validator that can fail without stopping the job validates nothing.
func TestTheBillOfMaterialsIsReadBySPDXsOwnValidator(t *testing.T) {
	root := rootOrFail(t)
	files := map[string]string{spdxValidator: readRepoFile(t, root, spdxValidator)}
	for rel, body := range workflowFiles(t, root) {
		files[rel] = body
	}

	broken := map[string]string{
		releaseWorkflow: "on: push\njobs:\n  build:\n    steps:\n" +
			"      - name: validate\n        continue-on-error: true\n        run: |\n" +
			"          \"$VENV/bin/pip\" install spdx-tools==0.8.5\n" +
			"          pyspdxtools --novalidation \\\n            -i dist/x.spdx.json\n",
		".github/workflows/ci.yml": "on: push\njobs:\n  lint:\n    steps:\n      - run: python -m pip install --require-hashes -r tools.txt\n",
		spdxValidator:              "# a comment\nclick==8.5.0\nrdflib>=7 \\\n    --hash=sha256:" + strings.Repeat("a", 64) + "\n",
	}
	plantedProblems := checkSPDXValidator(broken)
	for _, want := range []string{
		"runs pip install without --require-hashes",
		"runs pip install without --only-binary :all:",
		"without --version SPDX-2.3",
		"--novalidation",
		"does not install the validator from " + spdxValidator,
		"carries continue-on-error",
		"click==8.5.0 is not pinned",
		"rdflib>=7",
		"never pins spdx-tools",
	} {
		requireFinding(t, plantedProblems, want)
	}
	silent := checkSPDXValidator(map[string]string{releaseWorkflow: "on: push\njobs:\n  build:\n    steps:\n      - run: echo\n"})
	requireFinding(t, silent, "never hands the bill of materials to pyspdxtools")

	requireNone(t, checkSPDXValidator(files), "the validator of the bill of materials")
}

var (
	continuedLine = regexp.MustCompile(`[ \t]*\\\n\s*`)
	pinnedWheel   = regexp.MustCompile(`^([A-Za-z0-9._-]+)==[A-Za-z0-9.+!-]+( --hash=sha256:[0-9a-f]{64})+$`)
	// pipInstall is pip asked to install, by any spelling a workflow uses: a
	// bare pip, pip3, python -m pip, or a path in quotes.
	pipInstall = regexp.MustCompile(`\bpip3?["']?\s+install\b`)
)

// checkSPDXValidator reports a release that stops handing its bill of
// materials to the validator, reads it in another version than SPDX 2.3, or
// goes on when it is refused, and anything a workflow installs with pip that
// is not held to the bytes of its file. files holds every workflow and the
// validator's list, by path. A line a shell continues is read as one.
func checkSPDXValidator(files map[string]string) []string {
	var problems []string
	at := func(rel, detail string) { problems = append(problems, rel+": "+detail) }
	commands := func(body string) []string {
		var lines []string
		for _, line := range strings.Split(continuedLine.ReplaceAllString(body, " "), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "#") {
				lines = append(lines, line)
			}
		}
		return lines
	}
	for _, rel := range slices.Sorted(maps.Keys(files)) {
		if !strings.HasPrefix(rel, workflowsDir+"/") {
			continue
		}
		for _, line := range commands(files[rel]) {
			if !pipInstall.MatchString(line) {
				continue
			}
			for _, flag := range []string{"--require-hashes", "--only-binary :all:", "-r "} {
				if !strings.Contains(line, flag) {
					at(rel, "runs pip install without "+flag)
				}
			}
		}
	}

	build := jobBlocks(files[releaseWorkflow])[buildJob]
	validates, installs := false, false
	for _, line := range commands(build) {
		installs = installs || (pipInstall.MatchString(line) && strings.Contains(line, "-r "+spdxValidator))
		if !strings.Contains(line, "pyspdxtools") || !strings.Contains(line, ".spdx.json") {
			continue
		}
		validates = true
		if !strings.Contains(line, "--version SPDX-2.3") {
			at(releaseWorkflow, "reads the bill of materials without --version SPDX-2.3, so a document in another version passes")
		}
		if strings.Contains(line, "--novalidation") {
			at(releaseWorkflow, "hands the bill of materials over with --novalidation, which parses it and checks nothing")
		}
	}
	if !validates {
		at(releaseWorkflow, "the build job never hands the bill of materials to pyspdxtools")
	}
	if !installs {
		at(releaseWorkflow, "the build job does not install the validator from "+spdxValidator)
	}
	if strings.Contains(build, "continue-on-error") {
		at(releaseWorkflow, "the build job carries continue-on-error, so a refused build or bill of materials goes on to be attested")
	}

	pinned := map[string]bool{}
	for _, line := range commands(files[spdxValidator]) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := pinnedWheel.FindStringSubmatch(line)
		if m == nil {
			at(spdxValidator, line+" is not pinned to a version and the bytes of its file")
			continue
		}
		pinned[strings.ToLower(m[1])] = true
	}
	if !pinned["spdx-tools"] {
		at(spdxValidator, "never pins spdx-tools, the validator")
	}
	sort.Strings(problems)
	return problems
}

// TestTheMacBundleSaysWhatTheWindowsResourcesSay holds the two places the
// author of a program is shown by the system rather than by the program: the
// LegalCopyright of the Windows resources and the NSHumanReadableCopyright the
// bundle on macOS is given. The resources are held to the author in
// internal/license by another guard, and this holds the bundle to them, so the
// author reads the same on both systems. The icon the bundle carries has to be
// in the tree, or the script refuses the release at its end.
func TestTheMacBundleSaysWhatTheWindowsResourcesSay(t *testing.T) {
	root := rootOrFail(t)
	bundle := readRepoFile(t, root, bundleScript)
	resources := readRepoFile(t, root, windowResources)

	broken := planted(t, bundleScript, bundle, "Licensed under GPL-3.0-only.</string>", "All rights reserved.</string>")
	requireFinding(t, checkBundleCopyright(broken, resources), "does not give the bundle the LegalCopyright")

	requireNone(t, checkBundleCopyright(bundle, resources), "the bundle on macOS")
	icon := regexp.MustCompile(`icon="\$\{here\}/\.\./\.\./([^"]+)"`).FindStringSubmatch(bundle)
	if icon == nil {
		t.Fatalf("%s names no icon this test can read", bundleScript)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(icon[1]))); err != nil {
		t.Errorf("%s packs %s into every bundle, and it is not in the tree: %v", bundleScript, icon[1], err)
	}
}

var legalCopyright = regexp.MustCompile(`VALUE "LegalCopyright", "([^"]+)"`)

func checkBundleCopyright(bundle, resources string) []string {
	m := legalCopyright.FindStringSubmatch(resources)
	if m == nil {
		return []string{windowResources + ": has no LegalCopyright to compare with"}
	}
	if !strings.Contains(bundle, "<key>NSHumanReadableCopyright</key>\n\t<string>"+m[1]+"</string>") {
		return []string{bundleScript + ": does not give the bundle the LegalCopyright of " + windowResources + ", " + m[1]}
	}
	return nil
}
