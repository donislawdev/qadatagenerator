package cli

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
)

// The README is the first page a stranger reads, and what it shows was copied
// from runs of the program: a recipe, the lines validate printed, the first rows
// generate wrote, entries of a manifest, what corrupt and verify said. A new
// generator, a reworded report or another order of drawing would leave those
// copies wrong without anything noticing, so this runs the page (B-391).
//
// # How the page is read
//
// A comment GitHub does not show, <!-- readme-check: STEP ARGUMENTS -->, stands
// on the line right above what it checks. The steps run in the order of the
// page, in an empty directory, so each command sees the files the commands
// above it wrote, the way somebody following the page would.
//
//	run [exit=N]        runs the qadg commands of the block, one to a line. Each
//	                    ends with 0, the last with N when the marker names one
//	output [N]          the block is what the last run block printed, both
//	                    channels as a terminal shows them, or its Nth command
//	file NAME           writes the block to NAME
//	append NAME         adds the block to the end of NAME
//	head NAME           the block is how NAME starts
//	defects NAME        every row of the table is an applied entry of manifest NAME
//	fixture NAME        makes NAME, a file the page treats as the reader's own
//	response NAME FROM  writes the block to NAME as a response record, with every
//	                    other row of FROM accepted
//	generators          the table names every generator this build has
//	layers              the table names every group this build has damage in
//
// In a block of output a line holding only ... stands for any number of lines,
// and a line ending in " ..." for a line that starts with what comes before it.
// Every other line is compared exactly.
//
// # What keeps the check from shrinking
//
// Every line of a fenced block that starts with qadg has to be run by a step, so
// a deleted marker is a finding rather than a gap. A block of output made of
// nothing but ... is refused, because any output would match it. The Go version
// the page asks for is held to the go line of go.mod.
//
// # What it does not check
//
// Prose: counts and claims in sentences, the table of recipe keys, the exit
// codes and the build commands. A block of output with no marker is not found,
// because nothing tells it from any other block. There is no -update, because
// the page is written by hand: a difference comes with what the program printed,
// and the page is corrected by hand.
func TestTheReadmeShowsWhatTheProgramDoes(t *testing.T) {
	readme := readmeText(t, readmeFile)
	goMod := readmeText(t, goModFile)
	problems := goVersionProblems(readme, goMod)
	problems = append(problems, walkReadme(t, readme)...)
	for _, problem := range problems {
		t.Error(problem)
	}
}

const (
	readmeFile   = "../../README.md"
	goModFile    = "../../go.mod"
	readmeRecipe = "shop.yaml"
	markerOpen   = "<!-- readme-check: "
	markerClose  = " -->"
	fenceMark    = "```"
	tableMark    = "|"
	commandWord  = "qadg"
	elidedLines  = "..."
	elidedTail   = " ..."
	emptyCell    = "empty"
	bodyNone     = ""
	bodyBlock    = "block"
	bodyTable    = "table"
)

var (
	readmeGoVersion = regexp.MustCompile(`\[Go\]\(https://go\.dev/dl/\) (\S+) or newer`)
	goModVersion    = regexp.MustCompile(`(?m)^go (\S+)$`)
	backticked      = regexp.MustCompile("`([^`]+)`")
)

// readmeAnswers turns the last column of the table of defects into the
// expectation and the certainty the manifest writes.
var readmeAnswers = map[string][2]string{
	"reject":   {"rejection", "hard"},
	"~ reject": {"rejection", "soft"},
	"accept":   {"acceptance", "hard"},
	"~ accept": {"acceptance", "soft"},
}

// customersColumns are the columns of the file the page damages as the reader's
// own, taken from the customers of its recipe, with the card of a payment beside
// them.
var customersColumns = []string{"first_name", "last_name", "birth_date", "city", "zip"}

// readmeStep is one marker of the page together with what stands under it.
type readmeStep struct {
	at     int // the line of the marker, counted from one
	verb   string
	args   []string
	body   []string
	bodyAt int // the line the body starts on, zero when the step has none
}

// stepRule says what a step takes and what it does.
type stepRule struct {
	minArgs, maxArgs int
	body             string
	do               func(*readmeWalk, readmeStep) []string
}

var readmeStepRules = map[string]stepRule{
	"run":        {0, 1, bodyBlock, (*readmeWalk).run},
	"output":     {0, 1, bodyBlock, (*readmeWalk).output},
	"file":       {1, 1, bodyBlock, (*readmeWalk).file},
	"append":     {1, 1, bodyBlock, (*readmeWalk).appendTo},
	"head":       {1, 1, bodyBlock, (*readmeWalk).head},
	"defects":    {1, 1, bodyTable, (*readmeWalk).defects},
	"fixture":    {1, 1, bodyNone, (*readmeWalk).fixture},
	"response":   {2, 2, bodyBlock, (*readmeWalk).response},
	"generators": {0, 0, bodyTable, (*readmeWalk).generators},
	"layers":     {0, 0, bodyTable, (*readmeWalk).layers},
}

var readmeFixtures = map[string]func(*readmeWalk) []string{
	"customers.csv": (*readmeWalk).customersFile,
}

// readmeWalk is the state a walk down the page carries from step to step.
type readmeWalk struct {
	dir     string
	written map[string]string // what file steps wrote, before anything was added
	printed []string          // what each command of the last run block printed
}

// walkReadme runs every step of the page in a fresh directory and returns what
// did not hold.
func walkReadme(t *testing.T, readme string) []string {
	t.Helper()
	lines := strings.Split(readme, "\n")
	steps, problems := readmeSteps(lines)
	problems = append(problems, commandsLeftUnrun(lines, steps)...)
	walk := &readmeWalk{dir: t.TempDir(), written: map[string]string{}}
	t.Chdir(walk.dir)
	for _, step := range steps {
		for _, problem := range readmeStepRules[step.verb].do(walk, step) {
			problems = append(problems, fmt.Sprintf("README line %d, %s: %s", step.at, step.verb, problem))
		}
	}
	return problems
}

// readmeSteps finds every marker and what stands under it.
func readmeSteps(lines []string) ([]readmeStep, []string) {
	var steps []readmeStep
	var problems []string
	for at, line := range lines {
		spec, isMarker := strings.CutPrefix(line, markerOpen)
		if !isMarker {
			continue
		}
		spec, closed := strings.CutSuffix(spec, markerClose)
		fields := strings.Fields(spec)
		if !closed || len(fields) == 0 {
			problems = append(problems, fmt.Sprintf("README line %d reads %q, and a marker reads %sSTEP ARGUMENTS%s",
				at+1, line, markerOpen, markerClose))
			continue
		}
		step := readmeStep{at: at + 1, verb: fields[0], args: fields[1:]}
		rule, known := readmeStepRules[step.verb]
		switch {
		case !known:
			problems = append(problems, fmt.Sprintf("README line %d names the step %q, which this check does not have", step.at, step.verb))
			continue
		case len(step.args) < rule.minArgs || len(step.args) > rule.maxArgs:
			problems = append(problems, fmt.Sprintf("README line %d gives %s %d arguments, and it takes %d to %d",
				step.at, step.verb, len(step.args), rule.minArgs, rule.maxArgs))
			continue
		}
		if rule.body != bodyNone {
			body, found := bodyUnder(lines, at+1, rule.body)
			if !found {
				problems = append(problems, fmt.Sprintf("README line %d: %s needs a %s on the very next line, and there is none",
					step.at, step.verb, rule.body))
				continue
			}
			step.body, step.bodyAt = body, at+2
		}
		steps = append(steps, step)
	}
	return steps, problems
}

// bodyUnder reads the fenced block or the table that starts at line from.
func bodyUnder(lines []string, from int, kind string) ([]string, bool) {
	if from >= len(lines) {
		return nil, false
	}
	if kind == bodyTable {
		end := from
		for end < len(lines) && strings.HasPrefix(lines[end], tableMark) {
			end++
		}
		return lines[from:end], end-from >= 2
	}
	if !strings.HasPrefix(lines[from], fenceMark) {
		return nil, false
	}
	for end := from + 1; end < len(lines); end++ {
		if strings.HasPrefix(lines[end], fenceMark) {
			return lines[from+1 : end], true
		}
	}
	return nil, false
}

// commandsLeftUnrun finds every command of a fenced block that no run step
// covers.
func commandsLeftUnrun(lines []string, steps []readmeStep) []string {
	covered := map[int]bool{}
	for _, step := range steps {
		if step.verb == "run" {
			covered[step.bodyAt] = true
		}
	}
	var problems []string
	inside, openedAt := false, 0
	for at, line := range lines {
		if strings.HasPrefix(line, fenceMark) {
			inside, openedAt = !inside, at+1
			continue
		}
		if inside && isCommand(line) && !covered[openedAt] {
			problems = append(problems, fmt.Sprintf("README line %d shows %q, and no run step runs it", at+1, line))
		}
	}
	return problems
}

func isCommand(line string) bool {
	fields := strings.Fields(line)
	return len(fields) > 0 && fields[0] == commandWord
}

func (w *readmeWalk) run(step readmeStep) []string {
	last := 0
	if len(step.args) == 1 {
		text, named := strings.CutPrefix(step.args[0], "exit=")
		code, err := strconv.Atoi(text)
		if !named || err != nil {
			return []string{fmt.Sprintf("%q should read exit=N", step.args[0])}
		}
		last = code
	}
	var commands [][]string
	var problems []string
	for _, line := range step.body {
		switch {
		case strings.TrimSpace(line) == "":
		case !isCommand(line):
			problems = append(problems, fmt.Sprintf("%q is not a qadg command, and a run block holds nothing else", line))
		case strings.ContainsAny(line, "\"'`"):
			problems = append(problems, fmt.Sprintf("%q quotes an argument, and this check splits a command at its spaces", line))
		default:
			commands = append(commands, strings.Fields(line)[1:])
		}
	}
	if len(commands) == 0 {
		problems = append(problems, "the block runs nothing")
	}
	w.printed = nil
	if len(problems) > 0 {
		return problems
	}
	for at, args := range commands {
		printed, code := runInProcess(args)
		w.printed = append(w.printed, printed)
		want := 0
		if at == len(commands)-1 {
			want = last
		}
		if code != want {
			problems = append(problems, fmt.Sprintf("qadg %s ended with %d rather than %d, and printed:\n%s",
				strings.Join(args, " "), code, want, printed))
		}
	}
	return problems
}

func (w *readmeWalk) output(step readmeStep) []string {
	if len(w.printed) == 0 {
		return []string{"no run block above it printed anything to compare with"}
	}
	printed := strings.Join(w.printed, "")
	if len(step.args) == 1 {
		n, err := strconv.Atoi(step.args[0])
		if err != nil || n < 1 || n > len(w.printed) {
			return []string{fmt.Sprintf("%q names no command of the run block above, which ran %d", step.args[0], len(w.printed))}
		}
		printed = w.printed[n-1]
	}
	if onlyElided(step.body) {
		return []string{"the block shows nothing but ..., which any output would match"}
	}
	if !linesMatch(step.body, linesOf(printed)) {
		return []string{"the block is not what the program printed, which was:\n" + printed}
	}
	return nil
}

func (w *readmeWalk) file(step readmeStep) []string {
	text := strings.Join(step.body, "\n") + "\n"
	w.written[step.args[0]] = text
	return writeProblems(step.args[0], []byte(text))
}

func (w *readmeWalk) appendTo(step readmeStep) []string {
	before, err := os.ReadFile(step.args[0])
	if err != nil {
		return []string{fmt.Sprintf("nothing above it wrote %s: %v", step.args[0], err)}
	}
	return writeProblems(step.args[0], []byte(string(before)+"\n"+strings.Join(step.body, "\n")+"\n"))
}

func (w *readmeWalk) head(step readmeStep) []string {
	data, err := os.ReadFile(step.args[0])
	if err != nil {
		return []string{fmt.Sprintf("reading %s: %v", step.args[0], err)}
	}
	actual := linesOf(string(data))
	if len(actual) < len(step.body) || !slices.Equal(actual[:len(step.body)], step.body) {
		shown := min(len(step.body), len(actual))
		return []string{fmt.Sprintf("%s starts with:\n%s", step.args[0], strings.Join(actual[:shown], "\n"))}
	}
	return nil
}

// manifestEntry is the part of a manifest entry the table of defects shows.
type manifestEntry struct {
	Location struct {
		Entity string `json:"entity"`
		Row    int    `json:"row"`
		Field  string `json:"field"`
	} `json:"location"`
	Applied     bool   `json:"applied"`
	ValueBefore string `json:"value_before"`
	ValueAfter  string `json:"value_after"`
	Expectation string `json:"expectation"`
	Confidence  string `json:"confidence"`
}

func (w *readmeWalk) defects(step readmeStep) []string {
	var manifest struct {
		Defects []manifestEntry `json:"defects"`
	}
	data, err := os.ReadFile(step.args[0])
	if err != nil {
		return []string{fmt.Sprintf("reading %s: %v", step.args[0], err)}
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return []string{fmt.Sprintf("reading %s: %v", step.args[0], err)}
	}
	header := tableCells(step.body[0])
	at, problem := columnsOf(header, "Record", "Column", "Before", "After", "Your app should")
	if problem != "" {
		return []string{problem}
	}
	rows := step.body[2:]
	if len(rows) == 0 {
		return []string{"the table has no rows, so it checks nothing"}
	}
	var problems []string
	for _, row := range rows {
		cells := tableCells(row)
		if len(cells) != len(header) {
			problems = append(problems, fmt.Sprintf("the row %q has %d cells and the header %d", row, len(cells), len(header)))
			continue
		}
		shown, problem := shownEntry(cells, at)
		if problem != "" {
			problems = append(problems, fmt.Sprintf("the row %q: %s", row, problem))
			continue
		}
		if !slices.ContainsFunc(manifest.Defects, func(entry manifestEntry) bool { return entry == shown }) {
			problems = append(problems, fmt.Sprintf("the row %q is no applied entry of %s", row, step.args[0]))
		}
	}
	return problems
}

// shownEntry reads one row of the table of defects as the entry it claims.
func shownEntry(cells []string, at []int) (manifestEntry, string) {
	var entry manifestEntry
	entry.Applied = true
	record := strings.Fields(cells[at[0]])
	if len(record) != 2 {
		return entry, fmt.Sprintf("the record %q should read ENTITY ROW", cells[at[0]])
	}
	row, err := strconv.Atoi(record[1])
	if err != nil {
		return entry, fmt.Sprintf("the record %q should read ENTITY ROW", cells[at[0]])
	}
	entry.Location.Entity, entry.Location.Row, entry.Location.Field = record[0], row, cells[at[1]]
	var fine bool
	if entry.ValueBefore, fine = shownValue(cells[at[2]]); !fine {
		return entry, fmt.Sprintf("the value %q is neither in backticks nor %q", cells[at[2]], emptyCell)
	}
	if entry.ValueAfter, fine = shownValue(cells[at[3]]); !fine {
		return entry, fmt.Sprintf("the value %q is neither in backticks nor %q", cells[at[3]], emptyCell)
	}
	answer, known := readmeAnswers[cells[at[4]]]
	if !known {
		return entry, fmt.Sprintf("the answer %q is none of the words this check knows", cells[at[4]])
	}
	entry.Expectation, entry.Confidence = answer[0], answer[1]
	return entry, ""
}

func shownValue(cell string) (string, bool) {
	if cell == emptyCell {
		return "", true
	}
	inner, opened := strings.CutPrefix(cell, "`")
	inner, closed := strings.CutSuffix(inner, "`")
	return inner, opened && closed
}

func (w *readmeWalk) fixture(step readmeStep) []string {
	build, known := readmeFixtures[step.args[0]]
	if !known {
		return []string{fmt.Sprintf("this check has no way to make %s", step.args[0])}
	}
	return build(w)
}

// customersFile makes the file the page damages as if it were the reader's
// own: the customers of the recipe it saves as shop.yaml, before damage was
// ordered, with five of their columns and the card of the payment in the same
// row. The page shows the command and what it printed, and this is the file it
// printed that for.
func (w *readmeWalk) customersFile() []string {
	recipe, written := w.written[readmeRecipe]
	if !written {
		return []string{"nothing above it wrote " + readmeRecipe + ", which the file is made from"}
	}
	dir := filepath.Join(w.dir, "fixture")
	if problems := writeProblems(filepath.Join(dir, readmeRecipe), []byte(recipe)); problems != nil {
		return problems
	}
	if printed, code := runInProcess([]string{"generate", "--out", dir, filepath.Join(dir, readmeRecipe)}); code != 0 {
		return []string{fmt.Sprintf("generating the customers ended with %d:\n%s", code, printed)}
	}
	customers, problem := readTable(filepath.Join(dir, "customer.csv"))
	if problem != "" {
		return []string{problem}
	}
	payments, problem := readTable(filepath.Join(dir, "payment.csv"))
	if problem != "" {
		return []string{problem}
	}
	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	_ = writer.Write(append(slices.Clone(customersColumns), "card"))
	for row := range min(len(customers), len(payments)) {
		var record []string
		for _, column := range customersColumns {
			record = append(record, customers[row][column])
		}
		_ = writer.Write(append(record, payments[row]["card"]))
	}
	writer.Flush()
	return writeProblems("customers.csv", out.Bytes())
}

func (w *readmeWalk) response(step readmeStep) []string {
	var shown struct {
		SpecVersion string           `json:"spec_version"`
		Rows        []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal([]byte(strings.Join(step.body, "\n")), &shown); err != nil {
		return []string{fmt.Sprintf("the block is not a response record: %v", err)}
	}
	records, problem := readTable(step.args[1])
	if problem != "" {
		return []string{problem}
	}
	byRow := map[int]map[string]any{}
	for _, entry := range shown.Rows {
		number, numbered := entry["row"].(float64)
		if !numbered || int(number) < 1 || int(number) > len(records) {
			return []string{fmt.Sprintf("the entry %v names no row of %s, which has %d", entry, step.args[1], len(records))}
		}
		byRow[int(number)] = entry
	}
	rows := make([]map[string]any, 0, len(records))
	for number := 1; number <= len(records); number++ {
		entry, given := byRow[number]
		if !given {
			entry = map[string]any{"row": number, "verdict": "accepted"}
		}
		rows = append(rows, entry)
	}
	data, err := json.Marshal(map[string]any{"spec_version": shown.SpecVersion, "rows": rows})
	if err != nil {
		return []string{err.Error()}
	}
	return writeProblems(step.args[0], data)
}

func (w *readmeWalk) generators(step readmeStep) []string {
	named, problem := namedInColumn(step.body, "Generator")
	if problem != "" {
		return []string{problem}
	}
	return sameNames("generator", named, values.New("", "guard").Names())
}

func (w *readmeWalk) layers(step readmeStep) []string {
	named, problem := namedInColumn(step.body, "Code")
	if problem != "" {
		return []string{problem}
	}
	var groups []string
	for _, id := range mutators.IDs() {
		group, _, _ := strings.Cut(string(id), ".")
		if !slices.Contains(groups, group) {
			groups = append(groups, group)
		}
	}
	return sameNames("group of damage", named, groups)
}

// namedInColumn collects every name in backticks in one column of a table.
func namedInColumn(table []string, column string) ([]string, string) {
	at, problem := columnsOf(tableCells(table[0]), column)
	if problem != "" {
		return nil, problem
	}
	var named []string
	for _, row := range table[2:] {
		cells := tableCells(row)
		if at[0] >= len(cells) {
			return nil, fmt.Sprintf("the row %q has no %s column", row, column)
		}
		for _, match := range backticked.FindAllStringSubmatch(cells[at[0]], -1) {
			named = append(named, match[1])
		}
	}
	return named, ""
}

// sameNames reports what the program has and the table leaves out, and what the
// table names and the program does not have.
func sameNames(kind string, named, has []string) []string {
	var problems []string
	for _, name := range has {
		if !slices.Contains(named, name) {
			problems = append(problems, fmt.Sprintf("this build has the %s %s, and the table leaves it out", kind, name))
		}
	}
	for _, name := range named {
		if !slices.Contains(has, name) {
			problems = append(problems, fmt.Sprintf("the table names the %s %s, which this build does not have", kind, name))
		}
	}
	return problems
}

func columnsOf(header []string, names ...string) ([]int, string) {
	at := make([]int, 0, len(names))
	for _, name := range names {
		index := slices.Index(header, name)
		if index < 0 {
			return nil, fmt.Sprintf("the table has no column %q, and its header is %v", name, header)
		}
		at = append(at, index)
	}
	return at, ""
}

func tableCells(row string) []string {
	cells := strings.Split(strings.Trim(strings.TrimSpace(row), tableMark), tableMark)
	for i := range cells {
		cells[i] = strings.TrimSpace(cells[i])
	}
	return cells
}

// readTable reads a CSV file into its records, each keyed by the header.
func readTable(path string) ([]map[string]string, string) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Sprintf("reading %s: %v", path, err)
	}
	defer file.Close()
	all, err := csv.NewReader(file).ReadAll()
	if err != nil || len(all) == 0 {
		return nil, fmt.Sprintf("reading %s as CSV: %v", path, err)
	}
	records := make([]map[string]string, 0, len(all)-1)
	for _, fields := range all[1:] {
		record := map[string]string{}
		for at, column := range all[0] {
			record[column] = fields[at]
		}
		records = append(records, record)
	}
	return records, ""
}

func writeProblems(path string, data []byte) []string {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return []string{err.Error()}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return []string{err.Error()}
	}
	return nil
}

// linesMatch compares a block of output with what was printed, line by line,
// with the two elisions the page uses.
func linesMatch(shown, actual []string) bool {
	if len(shown) == 0 {
		return len(actual) == 0
	}
	if strings.TrimSpace(shown[0]) == elidedLines {
		for skipped := 0; skipped <= len(actual); skipped++ {
			if linesMatch(shown[1:], actual[skipped:]) {
				return true
			}
		}
		return false
	}
	if len(actual) == 0 || !lineMatches(shown[0], actual[0]) {
		return false
	}
	return linesMatch(shown[1:], actual[1:])
}

func lineMatches(shown, actual string) bool {
	if start, cut := strings.CutSuffix(shown, elidedTail); cut {
		return strings.HasPrefix(actual, start)
	}
	return shown == actual
}

func onlyElided(block []string) bool {
	for _, line := range block {
		if trimmed := strings.TrimSpace(line); trimmed != "" && trimmed != elidedLines {
			return false
		}
	}
	return true
}

func linesOf(text string) []string {
	text = strings.TrimSuffix(strings.ReplaceAll(text, "\r", ""), "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func goVersionProblems(readme, goMod string) []string {
	floor := goModVersion.FindStringSubmatch(goMod)
	if floor == nil {
		return []string{"go.mod has no go line to hold the README to"}
	}
	asked := readmeGoVersion.FindAllStringSubmatch(readme, -1)
	if len(asked) != 1 {
		return []string{fmt.Sprintf("the README says %d times which Go it needs, and should say it once, as [Go](https://go.dev/dl/) %s or newer",
			len(asked), floor[1])}
	}
	if asked[0][1] != floor[1] {
		return []string{fmt.Sprintf("the README asks for Go %s and go.mod for %s", asked[0][1], floor[1])}
	}
	return nil
}

// sharedLog is both channels at once, in the order they were written, the way
// a terminal shows a run.
type sharedLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *sharedLog) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *sharedLog) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func runInProcess(args []string) (string, int) {
	var log sharedLog
	code := Run(Environment{Args: args, Stdout: &log, Stderr: &log})
	return log.String(), code
}

func readmeText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(data), "\r", "")
}

// Every check of the page has its fault planted here, on a page of a few lines,
// and has to report it. The same page without the fault has to pass, so a check
// that reported everything would not pass for one that works.
func TestTheReadmeCheckGoesRedWhereThePageIsWrong(t *testing.T) {
	quick := "qadg quick vin --count 2 --seed canary"
	printed, code := runInProcess(strings.Fields(quick)[1:])
	if code != 0 {
		t.Fatalf("%s ended with %d:\n%s", quick, code, printed)
	}
	shown := linesOf(printed)
	changed := slices.Clone(shown)
	changed[0] += "X"

	manifest := `{"defects":[{"location":{"entity":"customer","row":3,"field":"card"},"applied":true,` +
		`"value_before":"4111111111111111","value_after":"4111111111111112","expectation":"rejection","confidence":"hard"}]}`
	defectsTable := func(after string) string {
		return pageOf(markerOf("file m.json"), fenced(manifest), markerOf("defects m.json"),
			"| Record | Column | Before | After | Your app should |", "|---|---|---|---|---|",
			"| customer 3 | card | `4111111111111111` | `"+after+"` | reject |")
	}
	names := values.New("", "guard").Names()
	generatorsTable := func(listed []string) string {
		rows := []string{markerOf("generators"), "| Generator | Makes |", "|---|---|"}
		for _, name := range listed {
			rows = append(rows, "| `"+name+"` | values |")
		}
		return pageOf(rows...)
	}
	var groups []string
	for _, id := range mutators.IDs() {
		group, _, _ := strings.Cut(string(id), ".")
		if !slices.Contains(groups, group) {
			groups = append(groups, group)
		}
	}
	layersTable := func(listed []string) string {
		rows := []string{markerOf("layers"), "| Group | Code | Your app should |", "|---|---|---|"}
		for _, group := range listed {
			rows = append(rows, "| a group | `"+group+"` | reject |")
		}
		return pageOf(rows...)
	}

	cases := []struct {
		name, good, bad, says string
	}{
		{"a changed line of output",
			pageOf(markerOf("run"), fenced(quick), markerOf("output"), fenced(shown...)),
			pageOf(markerOf("run"), fenced(quick), markerOf("output"), fenced(changed...)),
			"is not what the program printed"},
		{"a command nobody runs",
			pageOf(markerOf("run"), fenced(quick)),
			pageOf(fenced(quick)),
			"no run step runs it"},
		{"output that is nothing but elisions",
			pageOf(markerOf("run"), fenced(quick), markerOf("output"), fenced(shown[0], elidedLines)),
			pageOf(markerOf("run"), fenced(quick), markerOf("output"), fenced(elidedLines)),
			"any output would match"},
		{"a step this check does not have",
			pageOf(markerOf("run"), fenced(quick)),
			pageOf(markerOf("rnu"), fenced(quick)),
			"which this check does not have"},
		{"a marker with nothing under it",
			pageOf(markerOf("run"), fenced(quick)),
			pageOf(markerOf("run"), "", fenced(quick)),
			"on the very next line"},
		{"an exit code other than the page says",
			pageOf(markerOf("run exit=2"), fenced("qadg verify")),
			pageOf(markerOf("run"), fenced("qadg verify")),
			"ended with 2 rather than 0"},
		{"a file that starts otherwise",
			pageOf(markerOf("file a.txt"), fenced("one", "two"), markerOf("head a.txt"), fenced("one")),
			pageOf(markerOf("file a.txt"), fenced("one", "two"), markerOf("head a.txt"), fenced("two")),
			"a.txt starts with"},
		{"a defect the manifest does not carry",
			defectsTable("4111111111111112"),
			defectsTable("4111111111111113"),
			"is no applied entry"},
		{"a generator left out of the table",
			generatorsTable(names),
			generatorsTable(names[:len(names)-1]),
			"the table leaves it out"},
		{"a group of damage this build does not have",
			layersTable(groups),
			layersTable(append(slices.Clone(groups), "L9")),
			"which this build does not have"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if problems := walkReadme(t, c.good); len(problems) != 0 {
				t.Fatalf("the page without the fault was refused: %v", problems)
			}
			problems := walkReadme(t, c.bad)
			if !slices.ContainsFunc(problems, func(problem string) bool { return strings.Contains(problem, c.says) }) {
				t.Fatalf("the planted fault was not reported as %q, the check said %v", c.says, problems)
			}
		})
	}
}

// The Go version the page asks for is held to go.mod, and a page that says it
// twice or not at all is refused rather than read at its first sentence.
func TestTheReadmeAsksForTheGoThatGoModNames(t *testing.T) {
	goMod := "module example\n\ngo 1.27.3\n"
	sentence := func(version string) string { return "You need [Go](https://go.dev/dl/) " + version + " or newer.\n" }
	if problems := goVersionProblems(sentence("1.27.3"), goMod); len(problems) != 0 {
		t.Fatalf("the same version was refused: %v", problems)
	}
	for name, page := range map[string]string{
		"an older version": sentence("1.27.2"),
		"no sentence":      "Build it with Go.\n",
		"two sentences":    sentence("1.27.3") + sentence("1.27.3"),
	} {
		if problems := goVersionProblems(page, goMod); len(problems) == 0 {
			t.Errorf("%s passed", name)
		}
	}
}

// The two elisions of a block of output stand for what they say and for no
// more.
func TestTheReadmeElisionsMatchWhatTheySayAndNoMore(t *testing.T) {
	printed := []string{"first", "second line, longer", "third", "fourth"}
	for _, c := range []struct {
		shown []string
		match bool
	}{
		{[]string{"first", "second line, longer", "third", "fourth"}, true},
		{[]string{"first", elidedLines}, true},
		{[]string{"first", elidedLines, "fourth"}, true},
		{[]string{elidedLines, "first", "second line, longer", "third", "fourth"}, true},
		{[]string{"first", "second" + elidedTail, "third", "fourth"}, true},
		{[]string{"first", "second line, longer", "third"}, false},
		{[]string{"first", "third", "fourth"}, false},
		{[]string{"first", "other" + elidedTail, elidedLines}, false},
		{[]string{elidedLines, "fifth"}, false},
		{[]string{"first", "second line, longer", "third", "fourth", "fifth"}, false},
	} {
		if got := linesMatch(c.shown, printed); got != c.match {
			t.Errorf("%q against the printed lines matched %v, and should have %v", c.shown, got, c.match)
		}
	}
}

func pageOf(lines ...string) string { return strings.Join(lines, "\n") + "\n" }

func markerOf(spec string) string { return markerOpen + spec + markerClose }

func fenced(lines ...string) string {
	return fenceMark + "\n" + strings.Join(lines, "\n") + "\n" + fenceMark
}
