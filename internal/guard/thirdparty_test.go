package guard

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// # What this defends (B-414, DEC-317 point 3)
//
// The window takes its OpenGL binding from third_party/go-gl-gl rather than
// from the published module, because the published one makes the window
// import opengl32.dll when it starts, and then the software renderer it starts
// again with can never be the one it draws with. A copy is code nobody reviews
// line by line, so it is held here to what PATCH.md promises: the published
// module at the version go.mod requires, cut down to what the toolkit
// imports, with PATCH.diff applied and nothing else.
//
// The published bytes come from the module cache, through go mod download,
// and the sum written in PATCH.md is checked against them here rather than
// trusted. The sum was in go.sum until the replace directive took the module
// out of it. A function that hashed modules wrongly could not produce the
// sum the go command once recorded for this version, so that comparison is
// also the proof that moduleSum is right.

// carriedModule is the module carried in third_party.
const carriedModule = "github.com/go-gl/gl"

// carriedDir is where it is carried, relative to the module root.
const carriedDir = "third_party/go-gl-gl"

// carriedPatched are the files the change touches. PATCH.diff touching any
// other file is the change growing, which is a decision to write here rather
// than something the diff may do alone.
var carriedPatched = []string{"v2.1/gl/package.go", "v2.1/gl/procaddr.go"}

// carriedOurs are the files in the copy this project wrote.
var carriedOurs = map[string]bool{"PATCH.md": true, "PATCH.diff": true}

// carriedKept says which files of the published module the copy keeps: its
// go.mod and licence, and the two packages the toolkit imports.
func carriedKept(name string) bool {
	return name == "go.mod" || name == "LICENSE" ||
		strings.HasPrefix(name, "v2.1/gl/") || strings.HasPrefix(name, "v3.1/gles2/")
}

func TestTheOpenGLBindingIsThePublishedOnePlusOneChange(t *testing.T) {
	root := rootOrFail(t)
	version := requiredVersion(t, root)
	zipPath := downloadedModule(t, root, version)
	published, sum := readModuleZip(t, zipPath, carriedModule+"@"+version+"/")
	if pinned := pinnedSum(t, root); sum != pinned {
		t.Fatalf("the published %s@%s hashes to %s, and %s/PATCH.md pins %s. "+
			"The copy is checked against bytes that are not the version it claims.",
			carriedModule, version, sum, carriedDir, pinned)
	}
	carried := readTree(t, filepath.Join(root, filepath.FromSlash(carriedDir)))
	for _, finding := range copyDrift(published, carried) {
		t.Error(finding)
	}
}

func TestThirdPartyHoldsOnlyTheCarriedBinding(t *testing.T) {
	for _, finding := range thirdPartyDrift(rootOrFail(t)) {
		t.Error(finding)
	}
}

// thirdPartyDrift is everything in third_party other than the carried binding,
// and go.mod replacing anything other than that binding with it. The guards
// skip third_party, so whatever else lives there is checked by nothing.
func thirdPartyDrift(root string) []string {
	var findings []string
	entries, err := os.ReadDir(filepath.Join(root, "third_party"))
	if err != nil {
		return []string{"cannot read third_party: " + err.Error()}
	}
	for _, entry := range entries {
		if "third_party/"+entry.Name() != carriedDir {
			findings = append(findings, "third_party/"+entry.Name()+" is there, and the guards skip that "+
				"directory, so only "+carriedDir+" may live there")
		}
	}
	lines, err := goModLines(root)
	if err != nil {
		return append(findings, err.Error())
	}
	var replaces []string
	for _, line := range lines {
		if rest, ok := strings.CutPrefix(line, "replace"); ok {
			replaces = append(replaces, strings.Join(strings.Fields(rest), " "))
		}
	}
	if want := carriedModule + " => ./" + carriedDir; len(replaces) != 1 || replaces[0] != want {
		findings = append(findings, fmt.Sprintf("go.mod replaces %q, and the one replacement it may hold is %q", replaces, want))
	}
	return findings
}

// copyDrift is every way the copy differs from the published files with
// PATCH.diff applied: a file missing, a file added, a file changed beyond the
// patch, and a patch that touches files other than carriedPatched or does
// not apply.
func copyDrift(published, carried map[string][]byte) []string {
	patches, err := parseUnified(string(carried["PATCH.diff"]))
	if err != nil {
		return []string{"PATCH.diff cannot be read as a unified diff: " + err.Error()}
	}
	var findings []string
	want := map[string][]byte{}
	for name, body := range published {
		if carriedKept(name) {
			want[name] = body
		}
	}
	var touched []string
	for _, patch := range patches {
		touched = append(touched, patch.name)
		applied, err := patch.apply(string(want[patch.name]))
		if err != nil {
			findings = append(findings, "PATCH.diff does not apply to "+patch.name+": "+err.Error())
			continue
		}
		want[patch.name] = []byte(applied)
	}
	sort.Strings(touched)
	if strings.Join(touched, ",") != strings.Join(carriedPatched, ",") {
		findings = append(findings, fmt.Sprintf("PATCH.diff touches %q, and the change is to %q", touched, carriedPatched))
	}
	return append(findings, filesDrift(want, carried)...)
}

// filesDrift compares the copy with what it should hold, file by file.
func filesDrift(want, carried map[string][]byte) []string {
	var findings []string
	for name, body := range want {
		got, present := carried[name]
		switch {
		case !present:
			findings = append(findings, name+" is published and missing from the copy")
		case string(got) != string(body):
			findings = append(findings, name+" differs from the published file with PATCH.diff applied")
		}
	}
	for name := range carried {
		if _, expected := want[name]; !expected && !carriedOurs[name] {
			findings = append(findings, name+" is in the copy and is neither a published file it keeps nor one of ours")
		}
	}
	sort.Strings(findings)
	return findings
}

// filePatch is the part of a unified diff that changes one file.
type filePatch struct {
	name  string
	hunks []hunk
}

// hunk is one @@ block: where it starts in the original, and its lines with
// their first character, a space for context, - for removed and + for added.
type hunk struct {
	oldStart, oldCount, newCount int
	lines                        []string
}

var hunkHeader = regexp.MustCompile(`^@@ -(\d+),(\d+) \+\d+,(\d+) @@`)

// parseUnified reads a unified diff in the shape diff -u writes, with paths
// that begin a/ and b/. Anything it does not recognise is refused rather than
// skipped, since a line skipped here is a change nobody compared.
//
// An empty line inside a hunk is an empty line of context. diff writes it as
// one space, editors strip that space, and GNU patch reads the empty line the
// same way, so a diff that patch applies is a diff this applies.
func parseUnified(diff string) ([]filePatch, error) {
	var patches []filePatch
	for _, line := range strings.Split(strings.TrimSuffix(diff, "\n"), "\n") {
		if line == "" && len(patches) > 0 && len(patches[len(patches)-1].hunks) > 0 {
			line = " "
		}
		switch {
		case strings.HasPrefix(line, "--- a/"):
			patches = append(patches, filePatch{name: strings.TrimPrefix(line, "--- a/")})
		case strings.HasPrefix(line, "+++ b/"):
			if len(patches) == 0 || strings.TrimPrefix(line, "+++ b/") != patches[len(patches)-1].name {
				return nil, fmt.Errorf("%q does not follow the line naming the same file", line)
			}
		case strings.HasPrefix(line, "@@"):
			if len(patches) == 0 {
				return nil, fmt.Errorf("%q comes before any file", line)
			}
			parsed, err := newHunk(line)
			if err != nil {
				return nil, err
			}
			last := &patches[len(patches)-1]
			last.hunks = append(last.hunks, parsed)
		case line != "" && strings.ContainsRune(" -+", rune(line[0])):
			if len(patches) == 0 || len(patches[len(patches)-1].hunks) == 0 {
				return nil, fmt.Errorf("%q comes before any hunk", line)
			}
			last := &patches[len(patches)-1].hunks
			(*last)[len(*last)-1].lines = append((*last)[len(*last)-1].lines, line)
		default:
			return nil, fmt.Errorf("%q is not a line of a unified diff", line)
		}
	}
	return patches, nil
}

// newHunk reads the header of one hunk.
func newHunk(header string) (hunk, error) {
	match := hunkHeader.FindStringSubmatch(header)
	if match == nil {
		return hunk{}, fmt.Errorf("%q is not a hunk header with both counts", header)
	}
	numbers := make([]int, 3)
	for i := range numbers {
		numbers[i], _ = strconv.Atoi(match[i+1])
	}
	return hunk{oldStart: numbers[0], oldCount: numbers[1], newCount: numbers[2]}, nil
}

// apply applies the patch to original, and refuses where a line the patch
// expects is not the line that is there.
func (patch filePatch) apply(original string) (string, error) {
	lines := strings.SplitAfter(original, "\n")
	var out strings.Builder
	next := 0
	for _, each := range patch.hunks {
		start := each.oldStart - 1
		if start < next || start > len(lines) {
			return "", fmt.Errorf("a hunk starts at line %d, which is not after the one before", each.oldStart)
		}
		for _, kept := range lines[next:start] {
			out.WriteString(kept)
		}
		at, err := each.applyAt(lines, start, &out)
		if err != nil {
			return "", err
		}
		next = at
	}
	for _, kept := range lines[next:] {
		out.WriteString(kept)
	}
	return out.String(), nil
}

// applyAt writes the hunk's result for the original lines from start, and
// answers the first original line after it.
func (each hunk) applyAt(lines []string, start int, out *strings.Builder) (int, error) {
	at, removed, added := start, 0, 0
	for _, line := range each.lines {
		kind, text := line[0], line[1:]+"\n"
		if kind == '+' {
			out.WriteString(text)
			added++
			continue
		}
		if at >= len(lines) || lines[at] != text {
			return 0, fmt.Errorf("line %d is not %q", at+1, strings.TrimSuffix(text, "\n"))
		}
		if kind == ' ' {
			out.WriteString(text)
			added++
		}
		removed++
		at++
	}
	if removed != each.oldCount || added != each.newCount {
		return 0, fmt.Errorf("the hunk at line %d counts %d and %d lines and holds %d and %d",
			each.oldStart, each.oldCount, each.newCount, removed, added)
	}
	return at, nil
}

// requiredVersion is the version of the carried module go.mod requires, which
// the replace directive leaves in place.
func requiredVersion(t *testing.T, root string) string {
	t.Helper()
	lines, err := goModLines(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		if fields := strings.Fields(line); len(fields) >= 2 && fields[0] == carriedModule {
			return fields[1]
		}
	}
	t.Fatalf("go.mod requires no %s, so there is nothing for the copy to be a copy of", carriedModule)
	return ""
}

// goModLines are the lines of go.mod without comments.
func goModLines(root string) ([]string, error) {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("cannot read go.mod: %w", err)
	}
	var lines []string
	for _, line := range strings.Split(string(content), "\n") {
		code, _, _ := strings.Cut(line, "//")
		if code = strings.TrimSpace(code); code != "" {
			lines = append(lines, code)
		}
	}
	return lines, nil
}

// pinnedSum is the module sum PATCH.md names, which has to be exactly one.
func pinnedSum(t *testing.T, root string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(carriedDir), "PATCH.md"))
	if err != nil {
		t.Fatalf("cannot read the notes of the copy: %v", err)
	}
	sums := regexp.MustCompile(`h1:[A-Za-z0-9+/]+=*`).FindAllString(string(content), -1)
	if len(sums) != 1 {
		t.Fatalf("PATCH.md names %d module sums, and it pins the published version with exactly one", len(sums))
	}
	return sums[0]
}

// downloadedModule is the path of the published module's zip in the module
// cache, fetched through the module proxy when it is not there yet.
func downloadedModule(t *testing.T, root, version string) string {
	t.Helper()
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("this test reads the published module through the go command, which go test puts on PATH: %v", err)
	}
	command := exec.Command(goTool, "mod", "download", "-json", carriedModule+"@"+version)
	command.Dir = root
	output, err := command.Output()
	var answer struct{ Zip, Error string }
	if decodeErr := json.Unmarshal(output, &answer); decodeErr != nil || answer.Zip == "" {
		t.Fatalf("go mod download %s@%s gave no zip: %v %s\n%s", carriedModule, version, err, answer.Error, output)
	}
	return answer.Zip
}

// readModuleZip reads every file of a module zip, keyed by its path inside the
// module, and the module sum of the zip as go.sum writes it.
func readModuleZip(t *testing.T, path, prefix string) (map[string][]byte, string) {
	t.Helper()
	archive, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("cannot open the published module %s: %v", path, err)
	}
	defer func() { _ = archive.Close() }()
	files := map[string][]byte{}
	named := map[string][]byte{}
	for _, entry := range archive.File {
		body := readZipEntry(t, entry)
		named[entry.Name] = body
		if inside, ok := strings.CutPrefix(entry.Name, prefix); ok {
			files[inside] = body
		}
	}
	return files, moduleSum(named)
}

// readZipEntry reads one file of a zip.
func readZipEntry(t *testing.T, entry *zip.File) []byte {
	t.Helper()
	reader, err := entry.Open()
	if err != nil {
		t.Fatalf("cannot open %s in the published module: %v", entry.Name, err)
	}
	defer func() { _ = reader.Close() }()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("cannot read %s in the published module: %v", entry.Name, err)
	}
	return body
}

// moduleSum is the h1 sum go.sum keeps for a module: the SHA-256 of a summary
// listing the SHA-256 of every file by name, sorted, in base64. The go
// command calls it Hash1, in golang.org/x/mod/sumdb/dirhash.
func moduleSum(files map[string][]byte) string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	summary := sha256.New()
	for _, name := range names {
		_, _ = fmt.Fprintf(summary, "%x  %s\n", sha256.Sum256(files[name]), name)
	}
	return "h1:" + base64.StdEncoding.EncodeToString(summary.Sum(nil))
}

// readTree reads every file under dir, keyed by its path with forward slashes.
func readTree(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(path)
		files[filepath.ToSlash(relative)] = body
		return err
	})
	if err != nil {
		t.Fatalf("cannot read the copy in %s: %v", dir, err)
	}
	return files
}

// rootOrFail is the module root, or the end of the test.
func rootOrFail(t *testing.T) string {
	t.Helper()
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	return root
}
