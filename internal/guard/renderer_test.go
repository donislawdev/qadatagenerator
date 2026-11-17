package guard

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// # What this defends (B-414, DEC-317 point 4)
//
// The software renderer is two files of somebody else's that the window on
// Windows loads from a folder beside it. Nothing in a build knows whether they
// are there, because a script downloads them. So four things describe them,
// and each is read by something else: the pin, which the script downloads by,
// the notices, which travel in the folder with the files and which a person
// reads, the licence texts the notices name, and the window's own code, which
// asks for the files by name. A version raised in the pin alone would put
// files beside the window whose licences nobody read again, and a name changed
// in one place would give a folder the window does not look in.
const (
	rendererPin     = ".github/mesa-dist-win"
	rendererFetcher = ".github/scripts/fetch_software_renderer.sh"
	rendererNotices = ".github/opengl"
	rendererProject = "pal1000/mesa-dist-win"
	windowRenderer  = "cmd/qadg-gui/software.go"
)

// noticesFile is the file of the notices folder the others are named in.
const noticesFile = "NOTICES.md"

// A pinnedRenderer is what the pin names.
type pinnedRenderer struct {
	version string
	archive string
	sum     string
	files   []pinnedFile
}

// A pinnedFile is one file taken out of the archive.
type pinnedFile struct {
	inside string // the path inside the archive
	sum    string
	size   int64
}

func TestTheRendererBesideTheWindowIsTheOneItsNoticesDescribe(t *testing.T) {
	root := rootOrFail(t)
	pin, findings := parseRendererPin(readRepoFile(t, root, rendererPin))
	requireNone(t, findings, rendererPin)
	notices := readRepoFile(t, root, rendererNotices+"/"+noticesFile)
	folder, names, err := windowRendererNames(readRepoFile(t, root, windowRenderer))
	if err != nil {
		t.Fatalf("%s: %v", windowRenderer, err)
	}

	// The check is asked first about these notices with one word changed, so a
	// green run below cannot be a check that read nothing.
	if len(checkRendererNotices(pin, strings.Replace(notices, "`"+pin.version+"`", "`0.0.0`", 1), folder)) == 0 {
		t.Fatalf("the notices with another version passed the check, so it is not reading %s", noticesFile)
	}
	requireNone(t, checkRendererNotices(pin, notices, folder), rendererNotices+"/"+noticesFile)
	requireNone(t, checkRendererNames(pin, names), windowRenderer)

	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(rendererNotices)))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rendererNotices, err)
	}
	var travelling []string
	for _, entry := range entries {
		travelling = append(travelling, entry.Name())
	}
	requireNone(t, checkNoticeFolder(pin, travelling, notices), rendererNotices)

	// The one reader of the pin reads it, and nothing else under .github
	// carries a sum of the renderer: a second copy is the copy that drifts.
	fetcher := readRepoFile(t, root, rendererFetcher)
	for _, line := range []string{`pin="` + rendererPin + `"`, `notices="` + rendererNotices + `"`} {
		if !strings.Contains(fetcher, line) {
			t.Errorf("%s does not say %s, so it does not take what it downloads and what it copies from there", rendererFetcher, line)
		}
	}
	sums := append([]string{pin.sum}, sumsOf(pin.files)...)
	err = filepath.WalkDir(filepath.Join(root, ".github"), func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == rendererPin || path.Dir(rel) == rendererNotices {
			return nil
		}
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		for _, sum := range sums {
			if strings.Contains(string(body), sum) {
				t.Errorf("%s carries a sum of the software renderer, a second copy that can drift from %s", rel, rendererPin)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cannot read .github: %v", err)
	}
}

// Every way the four can drift apart is named, on a pin, notices and names a
// few lines long.
func TestTheRendererCheckNamesEveryWayItDrifts(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	other := strings.Repeat("cd", 32)
	good := "# a comment\nversion=1.2.3\narchive=mesa3d-1.2.3-release-msvc.7z\nsha256=" + sum + "\n" +
		"file=x64/first.dll " + other + " 10\nfile=x64/second.dll " + sum + " 20\n"
	pin, findings := parseRendererPin(good)
	requireNone(t, findings, "a well formed pin")

	_, findings = parseRendererPin("version=1.2.3\nversion=1.2.4\narchive=a.zip\nsha256=" + strings.ToUpper(sum) + "\n" +
		"file=../x64/first.dll " + other + " 10\nfile=x64/second.dll " + sum + "\nfile=x64/second.dll " + sum + " 0\ncolour=red\nloose\n")
	for _, want := range []string{"names version twice", "is not the name of a 7z archive", "is not a lowercase sha256",
		"is not a path inside the archive", "has 2 fields where a file has three", "is not a size", "names colour",
		"neither a comment nor key=value"} {
		requireFinding(t, findings, want)
	}
	_, findings = parseRendererPin("version=1.2.3\n")
	requireFinding(t, findings, "names no archive")
	requireFinding(t, findings, "names no file")

	notices := "# The software renderer in this folder\n\nMesa 3D, version `1.2.3`. MIT.\n\n" +
		"- `opengl/first.dll`, 10 bytes, SHA-256 `" + other + "`\n" +
		"- `opengl/second.dll`, 20 bytes, SHA-256 `" + sum + "`\n\n" +
		"Out of `mesa3d-1.2.3-release-msvc.7z`, SHA-256 `" + sum + "`, at\n" +
		"<https://github.com/pal1000/mesa-dist-win/releases/tag/1.2.3>.\n\n" +
		"| File in this folder | What it holds | Licence |\n|---|---|---|\n" +
		"| `LICENSE-One.txt` | one | MIT |\n| `LICENSE-Two.txt` | two | BSL-1.0 |\n"
	requireNone(t, checkRendererNotices(pin, notices, "opengl"), "notices that agree with the pin")
	requireFinding(t, checkRendererNotices(pin, "", "opengl"), "state no version")
	requireFinding(t, checkRendererNotices(pin, strings.Replace(notices, "`1.2.3`.", "`1.2.2`.", 1), "opengl"), "state 1.2.2")
	requireFinding(t, checkRendererNotices(pin, strings.Replace(notices, "10 bytes", "11 bytes", 1), "opengl"), "do not list opengl/first.dll")
	requireFinding(t, checkRendererNotices(pin, notices+"- `opengl/third.dll`, 5 bytes, SHA-256 `"+sum+"`\n", "opengl"), "list 3 file(s)")
	requireFinding(t, checkRendererNotices(pin, notices, "software"), "do not list software/first.dll")
	requireFinding(t, checkRendererNotices(pin, strings.Replace(notices, "release-msvc.7z`, SHA-256 `"+sum, "release-msvc.7z`, SHA-256 `"+other, 1), "opengl"),
		"do not name the archive")
	requireFinding(t, checkRendererNotices(pin, strings.Replace(notices, "/tag/1.2.3>", "/tag/1.2.2>", 1), "opengl"), "do not point at the release")

	requireNone(t, checkRendererNames(pin, []string{"first.dll", "second.dll"}), "names that agree with the pin")
	requireFinding(t, checkRendererNames(pin, []string{"second.dll", "first.dll"}), "loads second.dll as file 1")
	requireFinding(t, checkRendererNames(pin, []string{"first.dll"}), "asks for 1 file(s)")

	folder := []string{"LICENSE-One.txt", "LICENSE-Two.txt", noticesFile}
	requireNone(t, checkNoticeFolder(pin, folder, notices), "a folder that agrees with its notices")
	requireFinding(t, checkNoticeFolder(pin, []string{"LICENSE-One.txt", "LICENSE-Two.txt"}, notices), "holds no NOTICES.md")
	requireFinding(t, checkNoticeFolder(pin, []string{"LICENSE-One.txt", noticesFile}, notices), "names LICENSE-Two.txt, which is not in the folder")
	requireFinding(t, checkNoticeFolder(pin, append(folder, "LICENSE-Three.txt"), notices), "LICENSE-Three.txt is not named")
	requireFinding(t, checkNoticeFolder(pin, append(folder, "readme.txt"), notices), "readme.txt is neither")
	requireFinding(t, checkNoticeFolder(pin, append(folder, "second.dll"), notices), "second.dll has the name of a file of the renderer")

	if _, _, err := windowRendererNames("const (\n\tsoftwareDir = \"opengl\"\n)\n"); err == nil {
		t.Error("a file that names no renderer was read as one that does")
	}
	got, names, err := windowRendererNames("const (\n\tsoftwareDir      = \"opengl\"\n\tsoftwareRenderer = \"a.dll\"\n\tsoftwareLoader   = \"b.dll\"\n)\n")
	if err != nil || got != "opengl" || !slices.Equal(names, []string{"a.dll", "b.dll"}) {
		t.Errorf("the window's names were read as %q %q (%v)", got, names, err)
	}
}

var (
	rendererVersion = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)
	rendererArchive = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*\.7z$`)
	rendererInside  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*(/[A-Za-z0-9][A-Za-z0-9._-]*)*$`)
	rendererSum     = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// parseRendererPin reads the pin the way the fetching script does, and refuses
// what that script refuses: a key given twice, a value of the wrong shape, a
// line that is neither.
func parseRendererPin(body string) (pinnedRenderer, []string) {
	var pin pinnedRenderer
	var findings []string
	seen := map[string]bool{}
	single := map[string]*string{"version": &pin.version, "archive": &pin.archive, "sha256": &pin.sum}
	for i, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			findings = append(findings, fmt.Sprintf("line %d, %q, is neither a comment nor key=value", i+1, line))
			continue
		}
		if key == "file" {
			file, problem := parsePinnedFile(value)
			if problem != "" {
				findings = append(findings, fmt.Sprintf("line %d: %s", i+1, problem))
				continue
			}
			pin.files = append(pin.files, file)
			continue
		}
		target, known := single[key]
		switch {
		case !known:
			findings = append(findings, fmt.Sprintf("line %d names %s, which is not something the pin says", i+1, key))
		case seen[key]:
			findings = append(findings, fmt.Sprintf("line %d names %s twice, so the line a reviewer read may not be the line that is used", i+1, key))
		default:
			seen[key], *target = true, value
		}
	}
	for _, key := range []string{"version", "archive", "sha256"} {
		if !seen[key] {
			findings = append(findings, "the pin names no "+key)
		}
	}
	if seen["version"] && !rendererVersion.MatchString(pin.version) {
		findings = append(findings, fmt.Sprintf("%q is not a version", pin.version))
	}
	if seen["archive"] && !rendererArchive.MatchString(pin.archive) {
		findings = append(findings, fmt.Sprintf("%q is not the name of a 7z archive", pin.archive))
	}
	if seen["sha256"] && !rendererSum.MatchString(pin.sum) {
		findings = append(findings, "the sum of the archive is not a lowercase sha256")
	}
	if len(pin.files) == 0 {
		findings = append(findings, "the pin names no file to take out of the archive")
	}
	return pin, findings
}

func parsePinnedFile(value string) (pinnedFile, string) {
	fields := strings.Fields(value)
	if len(fields) != 3 {
		return pinnedFile{}, fmt.Sprintf("%q has %d fields where a file has three: a path, a sha256 and a size", value, len(fields))
	}
	if !rendererInside.MatchString(fields[0]) {
		return pinnedFile{}, fmt.Sprintf("%q is not a path inside the archive", fields[0])
	}
	if !rendererSum.MatchString(fields[1]) {
		return pinnedFile{}, fmt.Sprintf("the sum of %s is not a lowercase sha256", fields[0])
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || size <= 0 {
		return pinnedFile{}, fmt.Sprintf("%q is not a size in bytes", fields[2])
	}
	return pinnedFile{inside: fields[0], sum: fields[1], size: size}, ""
}

func sumsOf(files []pinnedFile) []string {
	var sums []string
	for _, file := range files {
		sums = append(sums, file.sum)
	}
	return sums
}

var (
	// noticesVersion is the sentence the notices open with.
	noticesVersion = regexp.MustCompile("(?m)^Mesa 3D, version `([^`]+)`")
	// listedFile is one line of the list of the files beside the window.
	listedFile = regexp.MustCompile("(?m)^- `([^`]+)`, ([0-9]+) bytes, SHA-256 `([0-9a-f]{64})`$")
	// listedLicence is one row of the table of the licence texts in the folder.
	listedLicence = regexp.MustCompile("(?m)^\\| `([^`]+)` \\|")
)

// checkRendererNotices asks the notices for the version, for each file under
// the name it lies beside the window by, for no file more, and for where the
// files come from.
func checkRendererNotices(pin pinnedRenderer, notices, folder string) []string {
	var findings []string
	stated := noticesVersion.FindStringSubmatch(notices)
	switch {
	case stated == nil:
		return []string{"the notices state no version of Mesa, in the sentence \"Mesa 3D, version `...`\""}
	case stated[1] != pin.version:
		findings = append(findings, fmt.Sprintf("the notices state %s and the pin downloads %s. Read the licences of that version again, then change both",
			stated[1], pin.version))
	}
	listed := map[string]bool{}
	for _, match := range listedFile.FindAllStringSubmatch(notices, -1) {
		listed[match[1]+" "+match[2]+" "+match[3]] = true
	}
	for _, file := range pin.files {
		beside := folder + "/" + path.Base(file.inside)
		if !listed[fmt.Sprintf("%s %d %s", beside, file.size, file.sum)] {
			findings = append(findings, fmt.Sprintf("the notices do not list %s with the %d bytes and the sum the pin names", beside, file.size))
		}
	}
	if len(listed) != len(pin.files) {
		findings = append(findings, fmt.Sprintf("the notices list %d file(s) and the pin takes %d", len(listed), len(pin.files)))
	}
	if !strings.Contains(notices, "`"+pin.archive+"`, SHA-256 `"+pin.sum+"`") {
		findings = append(findings, fmt.Sprintf("the notices do not name the archive %s with its sum", pin.archive))
	}
	if !strings.Contains(notices, "<https://github.com/"+rendererProject+"/releases/tag/"+pin.version+">") {
		findings = append(findings, fmt.Sprintf("the notices do not point at the release %s of %s", pin.version, rendererProject))
	}
	return findings
}

// checkRendererNames holds the files the window loads, in the order it loads
// them, to the files the pin takes out of the archive, in the order it names
// them. The order matters to the window, because the second file imports the
// first by name.
func checkRendererNames(pin pinnedRenderer, names []string) []string {
	if len(names) != len(pin.files) {
		return []string{fmt.Sprintf("the window asks for %d file(s) and the pin takes %d", len(names), len(pin.files))}
	}
	for i, file := range pin.files {
		if want := path.Base(file.inside); names[i] != want {
			return []string{fmt.Sprintf("the window loads %s as file %d where the pin names %s", names[i], i+1, want)}
		}
	}
	return nil
}

// checkNoticeFolder holds what the script copies beside the two files: the
// notices, and the licence texts their table names, each one there and none
// unnamed. A file with the name of a file of the renderer would replace one
// that was checked.
func checkNoticeFolder(pin pinnedRenderer, folder []string, notices string) []string {
	var findings []string
	if !slices.Contains(folder, noticesFile) {
		findings = append(findings, "the folder holds no "+noticesFile)
	}
	named := map[string]bool{}
	for _, match := range listedLicence.FindAllStringSubmatch(notices, -1) {
		if strings.HasPrefix(match[1], "LICENSE-") {
			named[match[1]] = true
		}
	}
	renderer := map[string]bool{}
	for _, file := range pin.files {
		renderer[path.Base(file.inside)] = true
	}
	for _, name := range folder {
		switch {
		case renderer[name]:
			findings = append(findings, name+" has the name of a file of the renderer, and would replace it")
		case name == noticesFile:
		case !strings.HasPrefix(name, "LICENSE-"):
			findings = append(findings, name+" is neither "+noticesFile+" nor a licence text, and travels beside the window unread")
		case !named[name]:
			findings = append(findings, name+" is not named in the table of "+noticesFile+", so nobody reading it learns what it covers")
		}
	}
	for name := range named {
		if !slices.Contains(folder, name) {
			findings = append(findings, noticesFile+" names "+name+", which is not in the folder")
		}
	}
	slices.Sort(findings)
	return findings
}

var windowRendererConst = regexp.MustCompile(`(?m)^\s*(softwareDir|softwareRenderer|softwareLoader)\s*=\s*"([^"]+)"\s*$`)

// windowRendererNames reads the folder and the two file names out of the
// window's source, in the order the window loads them.
func windowRendererNames(src string) (string, []string, error) {
	found := map[string]string{}
	for _, match := range windowRendererConst.FindAllStringSubmatch(src, -1) {
		found[match[1]] = match[2]
	}
	for _, name := range []string{"softwareDir", "softwareRenderer", "softwareLoader"} {
		if found[name] == "" {
			return "", nil, fmt.Errorf("no constant %s, so the names the window asks for cannot be read", name)
		}
	}
	return found["softwareDir"], []string{found["softwareRenderer"], found["softwareLoader"]}, nil
}

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rel, err)
	}
	return string(body)
}

func requireNone(t *testing.T, findings []string, what string) {
	t.Helper()
	if len(findings) != 0 {
		t.Errorf("%s:\n%s", what, strings.Join(findings, "\n"))
	}
}

func requireFinding(t *testing.T, findings []string, want string) {
	t.Helper()
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return
		}
	}
	t.Errorf("no finding saying %q in:\n%s", want, strings.Join(findings, "\n"))
}
