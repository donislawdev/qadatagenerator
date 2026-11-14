package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The website shows more of the program than the README holds: the ready-made
// recipes of examples/ with what each one writes, and any command a page names
// with {{< run "qadg ..." >}}. A web page cannot run the program, so this test
// runs it and keeps what it saw in the files the site is built from (B-402):
//
//	web/data/runs.json                 what every such command printed
//	web/data/examples/NAME.json        the files each example writes, and the
//	                                   first rows of each
//	web/assets/example-runs/NAME/      the manifest of an example that orders
//	                                   damage, offered for download
//
// Every command runs in an empty directory that holds the examples, the way
// somebody runs one they downloaded. Every example is run with generate, which
// its page shows. The site stops its build on a command or an example this
// file does not have, so a page cannot show what nobody ran.
//
// Run with -update after a deliberate change, then read the diff: it is what
// the pages now show differently. A manifest is compared without the time of
// its run, and -update leaves one alone when nothing else in it moved.
//
// What it does not check: the sentences written around these in every language,
// and that an example does what the comment at its top says.
func TestTheSiteShowsWhatTheProgramDoes(t *testing.T) {
	// Every command below runs in a directory of its own, so the repository
	// is found once, before the first of them moves away from it.
	root, err := filepath.Abs(siteRoot)
	if err != nil {
		t.Fatalf("cannot find the repository: %v", err)
	}
	examples, problems := exampleNames(inRepository(root, examplesDir))
	pages, err := sitePages(inRepository(root, siteContentDir))
	if err != nil {
		t.Fatalf("cannot read the pages of the site: %v", err)
	}
	commands, found := commandsOnPages(pages)
	problems = append(problems, found...)
	for _, name := range examples {
		commands = append(commands, exampleCommand(name))
	}
	slices.Sort(commands)
	commands = slices.Compact(commands)

	want, found := showRuns(t, root, examples, commands)
	problems = append(problems, found...)
	for _, problem := range problems {
		t.Error(problem)
	}
	if t.Failed() {
		return
	}
	if *updateDataset {
		writeShown(t, root, want, shownOnDisk(t, root))
		return
	}
	for _, problem := range shownProblems(want, shownOnDisk(t, root)) {
		t.Error(problem)
	}
}

// Each check of this test has its fault planted here and has to report it, and
// the same input without the fault has to pass.
func TestTheSiteRunsCheckGoesRedWhereTheSiteIsWrong(t *testing.T) {
	commands, problems := commandsOnPages(map[string]string{
		"a.md": "{{< run \"qadg quick vin --count 2\" >}}\n{{< run \"qadg quick vin --count 2\" >}}\n",
		"b.md": "{{< run qadg quick vin >}}\n{{< run \"echo hello\" >}}\n",
	})
	if !slices.Equal(commands, []string{"qadg quick vin --count 2"}) || len(problems) != 2 {
		t.Errorf("the pages gave the commands %q and the problems %q, and should give one command and two problems",
			commands, problems)
	}

	if _, problems := exampleNames(t.TempDir()); len(problems) != 1 {
		t.Errorf("a directory with no example gave %d problems rather than one", len(problems))
	}

	manifest := "{\n  \"run\": {\n    \"generated_at\": \"2026-11-14T00:00:00Z\",\n    \"seed\": \"s\"\n  }\n}\n"
	later := strings.Replace(manifest, "2026-11-14T00:00:00Z", "2026-11-15T09:30:00Z", 1)
	moved := strings.Replace(manifest, `"s"`, `"t"`, 1)
	runs := siteRunsPath
	kept := exampleRunsPath + "/a/" + manifestName
	want := map[string][]byte{runs: []byte("printed\n"), kept: []byte(manifest)}
	cases := []struct {
		name  string
		disk  map[string][]byte
		wrong bool
	}{
		{"the same files", map[string][]byte{runs: []byte("printed\r\n"), kept: []byte(manifest)}, false},
		{"a manifest of a later run", map[string][]byte{runs: []byte("printed\n"), kept: []byte(later)}, false},
		{"a manifest that moved", map[string][]byte{runs: []byte("printed\n"), kept: []byte(moved)}, true},
		{"another printout", map[string][]byte{runs: []byte("printed!\n"), kept: []byte(manifest)}, true},
		{"a missing file", map[string][]byte{runs: []byte("printed\n")}, true},
		{"a file nothing makes", map[string][]byte{runs: []byte("printed\n"), kept: []byte(manifest),
			exampleDataPath + "/gone.json": []byte("{}\n")}, true},
	}
	for _, c := range cases {
		if got := len(shownProblems(want, c.disk)) > 0; got != c.wrong {
			t.Errorf("%s: reported %v, and should report %v", c.name, got, c.wrong)
		}
	}
}

// Paths are slash separated and relative to the repository, which siteRoot is
// from this package, so a message names a file the way the repository does.
// inRepository turns one into a path on this system.
const (
	siteRoot        = "../../"
	examplesDir     = "examples"
	siteContentDir  = "web/content"
	siteRunsPath    = "web/data/runs.json"
	exampleDataPath = "web/data/examples"
	exampleRunsPath = "web/assets/example-runs"
	siteRunsRedo    = "go test ./internal/cli -run TestTheSiteShowsWhatTheProgramDoes -update"

	manifestName = "expectation-manifest.json"
	runNoteName  = "run-note.json"

	// exampleRowsShown is how many rows of a table the page of an example
	// shows, and exampleLinesShown how many lines of a JSON file. The page says
	// how many there are in all, and the reader runs the recipe for the rest.
	exampleRowsShown  = 5
	exampleLinesShown = 40
)

var (
	runShortcode    = regexp.MustCompile(`\{\{<\s*run\s+"([^"]*)"\s*>\}\}`)
	runShortcodeAny = regexp.MustCompile(`\{\{<\s*run\b`)
	exampleName     = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

// siteRun is one command as the site shows it.
type siteRun struct {
	Command string `json:"command"`
	Exit    int    `json:"exit"`
	Printed string `json:"printed"`
}

// siteExample is what one example writes, in the order of the file names, and
// the tables of the run with their rows as its note counts them. A JSON file
// holds every table at once, so the note is where the page reads how many
// rows each one has.
type siteExample struct {
	Files  []siteExampleFile  `json:"files"`
	Tables []siteExampleTable `json:"tables"`
}

type siteExampleTable struct {
	Entity  string `json:"entity"`
	Rows    int    `json:"rows"`
	Columns int    `json:"columns"`
}

// siteExampleFile is one file an example writes. Kind says how the page shows
// it: the first rows of a table, the first lines of JSON, or only its name with
// a sentence about it for the manifest and the note of the run.
type siteExampleFile struct {
	Name   string     `json:"name"`
	Kind   string     `json:"kind"`
	Header []string   `json:"header,omitempty"`
	Rows   [][]string `json:"rows,omitempty"`
	Lines  []string   `json:"lines,omitempty"`
	Total  int        `json:"total,omitempty"`
}

func inRepository(root, name string) string { return filepath.Join(root, filepath.FromSlash(name)) }

func exampleCommand(name string) string { return "qadg generate " + name + ".yaml" }

// exampleNames lists the recipes in dir by name, without .yaml.
func exampleNames(dir string) ([]string, []string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, []string{fmt.Sprintf("cannot read %s: %v", examplesDir, err)}
	}
	var names, problems []string
	for _, entry := range entries {
		name, isRecipe := strings.CutSuffix(entry.Name(), ".yaml")
		switch {
		case entry.IsDir() || !isRecipe:
			problems = append(problems, fmt.Sprintf("%s/%s is not a recipe, and %s holds recipes alone", examplesDir, entry.Name(), examplesDir))
		case !exampleName.MatchString(name):
			problems = append(problems, fmt.Sprintf("%s/%s is named for an address, so its name is lower case letters and digits with single hyphens",
				examplesDir, entry.Name()))
		default:
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		problems = append(problems, fmt.Sprintf("%s holds no recipe, and the site has a page for every one", examplesDir))
	}
	return names, problems
}

// sitePages reads every page of the site, in every language, by its path.
func sitePages(dir string) (map[string]string, error) {
	pages := map[string]string{}
	err := filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || filepath.Ext(file) != ".md" {
			return err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, file)
		pages[siteContentDir+"/"+filepath.ToSlash(relative)] = string(data)
		return err
	})
	return pages, err
}

// commandsOnPages collects what every run shortcode names, once each and in
// order. A shortcode written another way is a finding, so a page cannot show a
// command this test passed over.
func commandsOnPages(pages map[string]string) ([]string, []string) {
	var commands, problems []string
	for _, name := range slices.Sorted(maps.Keys(pages)) {
		for at, line := range strings.Split(pages[name], "\n") {
			whole := runShortcode.FindAllStringSubmatch(line, -1)
			if len(runShortcodeAny.FindAllString(line, -1)) != len(whole) {
				problems = append(problems, fmt.Sprintf("%s line %d reads %q, and a command is shown as {{< run \"qadg ...\" >}}", name, at+1, line))
				continue
			}
			for _, match := range whole {
				if !isCommand(match[1]) {
					problems = append(problems, fmt.Sprintf("%s line %d shows %q, which is not a qadg command", name, at+1, match[1]))
					continue
				}
				commands = append(commands, match[1])
			}
		}
	}
	slices.Sort(commands)
	return slices.Compact(commands), problems
}

// showRuns runs every command and returns every file the site is built from,
// by its path.
func showRuns(t *testing.T, root string, examples, commands []string) (map[string][]byte, []string) {
	t.Helper()
	shown := map[string][]byte{}
	var runs []siteRun
	var problems []string
	for _, command := range commands {
		if strings.ContainsAny(command, "\"'`") {
			problems = append(problems, fmt.Sprintf("%q quotes an argument, and this check splits a command at its spaces", command))
			continue
		}
		dir := t.TempDir()
		for _, name := range examples {
			data, err := os.ReadFile(inRepository(root, examplesDir+"/"+name+".yaml"))
			if err != nil {
				t.Fatalf("cannot read the example %s: %v", name, err)
			}
			if err := os.WriteFile(filepath.Join(dir, name+".yaml"), data, 0o644); err != nil {
				t.Fatalf("cannot copy the example %s: %v", name, err)
			}
		}
		t.Chdir(dir)
		printed, code := runInProcess(strings.Fields(command)[1:])
		runs = append(runs, siteRun{Command: command, Exit: code, Printed: printed})

		name, isExample := exampleOf(command, examples)
		if !isExample {
			continue
		}
		if code != 0 {
			problems = append(problems, fmt.Sprintf("%s ended with %d, and an example runs as it is:\n%s", command, code, printed))
			continue
		}
		made, found := exampleMade(dir, name, examples, shown)
		problems = append(problems, found...)
		shown[exampleDataPath+"/"+name+".json"] = encodeForSite(t, made)
	}
	shown[siteRunsPath] = encodeForSite(t, map[string][]siteRun{"runs": runs})
	return shown, problems
}

func exampleOf(command string, examples []string) (string, bool) {
	for _, name := range examples {
		if command == exampleCommand(name) {
			return name, true
		}
	}
	return "", false
}

// exampleMade reads what one example wrote into dir, and adds its manifest to
// what the site is built from.
func exampleMade(dir, name string, examples []string, shown map[string][]byte) (siteExample, []string) {
	var made siteExample
	var problems []string
	err := filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		written := filepath.ToSlash(relative)
		if recipe, isYAML := strings.CutSuffix(written, ".yaml"); isYAML && slices.Contains(examples, recipe) {
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		shownFile, problem := showFile(written, data)
		if problem != "" {
			problems = append(problems, fmt.Sprintf("%s wrote %s: %s", exampleCommand(name), written, problem))
			return nil
		}
		switch shownFile.Kind {
		case "manifest":
			shown[exampleRunsPath+"/"+name+"/"+manifestName] = data
		case "note":
			tables, problem := tablesOfNote(data)
			if problem != "" {
				problems = append(problems, fmt.Sprintf("%s wrote %s: %s", exampleCommand(name), written, problem))
			}
			made.Tables = tables
		}
		made.Files = append(made.Files, shownFile)
		return nil
	})
	if err != nil {
		problems = append(problems, fmt.Sprintf("cannot read what %s wrote: %v", exampleCommand(name), err))
	}
	return made, problems
}

// showFile is the part of a written file the page of its example shows.
func showFile(name string, data []byte) (siteExampleFile, string) {
	shown := siteExampleFile{Name: name}
	switch {
	case name == manifestName:
		shown.Kind = "manifest"
	case name == runNoteName:
		shown.Kind = "note"
	case path.Ext(name) == ".csv":
		records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil || len(records) == 0 {
			return shown, fmt.Sprintf("it does not read as a table: %v", err)
		}
		shown.Kind, shown.Header, shown.Total = "table", records[0], len(records)-1
		shown.Rows = records[1:min(len(records), exampleRowsShown+1)]
	case path.Ext(name) == ".json":
		lines := linesOf(string(data))
		shown.Kind, shown.Total = "json", len(lines)
		shown.Lines = lines[:min(len(lines), exampleLinesShown)]
	default:
		return shown, "the site has no way to show a file of this kind"
	}
	return shown, ""
}

// tablesOfNote reads the tables a run made from its note.
func tablesOfNote(data []byte) ([]siteExampleTable, string) {
	var note struct {
		Tables []struct {
			Entity  string   `json:"entity"`
			Rows    int      `json:"rows"`
			Columns []string `json:"columns"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(data, &note); err != nil {
		return nil, fmt.Sprintf("it does not read as a note of the run: %v", err)
	}
	if len(note.Tables) == 0 {
		return nil, "it names no table, and the page says how many rows each one has"
	}
	tables := make([]siteExampleTable, 0, len(note.Tables))
	for _, table := range note.Tables {
		tables = append(tables, siteExampleTable{Entity: table.Entity, Rows: table.Rows, Columns: len(table.Columns)})
	}
	return tables, ""
}

// encodeForSite writes a value the way the site's data is committed: indented,
// with a final newline, and with < and > left as they are, because the site
// prints <recipe> and a reader of the diff should see it too.
func encodeForSite(t *testing.T, value any) []byte {
	t.Helper()
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		t.Fatalf("cannot encode what the site shows: %v", err)
	}
	return b.Bytes()
}

// shownOnDisk reads every file this test writes, as the repository holds them
// now: the runs, and everything under the two directories it owns.
func shownOnDisk(t *testing.T, root string) map[string][]byte {
	t.Helper()
	disk := map[string][]byte{}
	if data, err := os.ReadFile(inRepository(root, siteRunsPath)); err == nil {
		disk[siteRunsPath] = data
	}
	for _, owned := range []string{exampleDataPath, exampleRunsPath} {
		dir := inRepository(root, owned)
		err := filepath.WalkDir(dir, func(file string, entry fs.DirEntry, err error) error {
			if os.IsNotExist(err) && file == dir {
				return fs.SkipDir
			}
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(dir, file)
			disk[owned+"/"+filepath.ToSlash(relative)] = data
			return err
		})
		if err != nil {
			t.Fatalf("cannot read %s: %v", owned, err)
		}
	}
	return disk
}

// shownProblems compares what the program made with what the repository holds,
// both ways.
func shownProblems(want, disk map[string][]byte) []string {
	var problems []string
	for _, name := range slices.Sorted(maps.Keys(want)) {
		had, there := disk[name]
		switch {
		case !there:
			problems = append(problems, fmt.Sprintf("%s is missing, and the site is built from it. Write it with\n  %s", name, siteRunsRedo))
		case !sameShown(name, had, want[name]):
			problems = append(problems, firstDifference(name, had, want[name], siteRunsRedo))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(disk)) {
		if _, wanted := want[name]; !wanted {
			problems = append(problems, fmt.Sprintf("%s is in the repository, and no example or page makes it any more. Remove it with\n  %s",
				name, siteRunsRedo))
		}
	}
	return problems
}

// sameShown compares two versions of one file, a manifest without the time of
// its run.
func sameShown(name string, had, now []byte) bool {
	had = bytes.ReplaceAll(had, []byte("\r\n"), []byte("\n"))
	if path.Base(name) == manifestName {
		return withoutTimestamp(string(had)) == withoutTimestamp(string(now))
	}
	return bytes.Equal(had, now)
}

// writeShown makes the repository hold what the program made: it writes every
// file that changed, keeps a manifest that moved only in time, and removes what
// nothing makes any more.
func writeShown(t *testing.T, root string, want, disk map[string][]byte) {
	t.Helper()
	changed := 0
	for _, name := range slices.Sorted(maps.Keys(want)) {
		if had, there := disk[name]; there && sameShown(name, had, want[name]) {
			continue
		}
		file := inRepository(root, name)
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatalf("cannot make the directory of %s: %v", name, err)
		}
		if err := os.WriteFile(file, want[name], 0o644); err != nil {
			t.Fatalf("cannot write %s: %v", name, err)
		}
		changed++
	}
	for _, name := range slices.Sorted(maps.Keys(disk)) {
		if _, wanted := want[name]; wanted {
			continue
		}
		if err := os.Remove(inRepository(root, name)); err != nil {
			t.Fatalf("cannot remove %s: %v", name, err)
		}
		changed++
	}
	t.Logf("%d files of the site written or removed. Read the diff before committing it", changed)
}
