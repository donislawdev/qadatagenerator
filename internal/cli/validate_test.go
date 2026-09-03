package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
)

// # The finding this exists for
//
// A terminal obeys what it is sent, and a recipe comes from somebody else's
// repository. Measured on 2026-08-02: a recipe naming a key with an escape
// sequence put those bytes into one of our own messages, where the sequence
// cleared the line and repainted the text. A refusal could be made to read on
// screen as an approval.
//
// Neither --no-color nor NO_COLOR helped, which is the part worth remembering.
// Both govern the colour this tool adds, and neither has anything to do with
// what the data carries.
func TestControlCharactersFromARecipeNeverReachTheLogChannel(t *testing.T) {
	path := recipeFile(t, "version: \"0.1\"\nentities:\n  a:\n"+
		"    \"\x1b[2K\x1b[32mrecipe is valid\x1b[0m\": 1\n"+
		"    fields:\n      x: { generator: y }\n")

	// Every combination of the two switches that govern colour, because the
	// finding was that neither of them is the answer here.
	for _, env := range []Environment{
		{},
		{StderrIsTerminal: true},
		{StderrIsTerminal: true, NoColorRequested: true},
	} {
		got := invoke(env, "validate", path)

		// The colour this tool adds itself is removed first, so that what is
		// left is only what came out of the recipe. Checking the raw log would
		// fail on our own escapes and prove nothing about the recipe's.
		ours := strings.NewReplacer(ansiRed, "", ansiYellow, "", ansiReset, "")
		if strings.Contains(ours.Replace(got.log), "\x1b") {
			t.Errorf("an escape sequence from the recipe reached the log: %q", got.log)
		}
		if !strings.Contains(got.log, string(replacement)) {
			t.Errorf("the escape was dropped without a trace, so the name reads as though it were clean: %q", got.log)
		}
	}
}

// The other half of the same rule, and it has to be a test rather than a
// comment. Data is the product, layer L0 exists to inject control characters
// on purpose, and cleaning that channel would break what this tool is for.
func TestTheDataChannelIsNotCleanedOfControlCharacters(t *testing.T) {
	var log strings.Builder
	var data strings.Builder
	out := &streams{data: &data, log: &log, level: levelNormal}

	out.emit("a\x1bb")
	if !strings.Contains(data.String(), "\x1b") {
		t.Fatal("the data channel was cleaned, which would break every corruption that injects a control character")
	}
}

// The machine readable report was safe before any of this, because the JSON
// encoder writes the escape rather than the byte. Said out loud so that
// changing the encoder cannot take it away quietly.
func TestTheMachineReadableReportEscapesControlCharacters(t *testing.T) {
	path := recipeFile(t, "version: \"0.1\"\nentities:\n  a:\n"+
		"    \"\x1b[2Kfake\": 1\n    fields:\n      x: { generator: y }\n")

	got := invoke(Environment{}, "validate", "--json", path)
	if strings.Contains(got.data, "\x1b") {
		t.Errorf("a raw escape reached the data channel: %q", got.data)
	}
	if !strings.Contains(got.data, `\u001b`) {
		t.Errorf("the escape is not in the report at all, so the report lost information: %q", got.data)
	}
}

// # The finding this exists for, found by review rather than by a test
//
// The size limit refused an oversized recipe, and refused it after the whole
// file was already in memory. Measured on 2026-08-02: a 250 MB file cost
// 258 MB of memory, alike on three runs, on its way to being told it was over
// a 256 kB limit. The refusal message even quoted the file's size, which it
// could only know by holding it.
//
// A limit that spends the resource it guards is a limit in name. In somebody
// else's pipeline the outcome is a build killed by the machine rather than an
// exit code from the table, and a killed build says nothing about why.
func TestAnOversizedRecipeIsNotHeldInMemoryToBeRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.yaml")
	oversized := bytes.Repeat([]byte("#\n"), recipeyaml.MaxSize)
	if err := os.WriteFile(path, oversized, 0o644); err != nil {
		t.Fatalf("cannot write the sample: %v", err)
	}

	data, err := readRecipe(path)
	if err != nil {
		t.Fatalf("reading failed: %v", err)
	}
	if len(data) > recipeyaml.MaxSize+1 {
		t.Fatalf("read %d bytes of a %d byte file, so the whole thing was held in order to refuse it",
			len(data), len(oversized))
	}

	// The bytes that were read still have to be refused. A cap that turned an
	// oversized recipe into an acceptable one would be worse than no cap.
	if recipeyaml.Decode(data).Valid() {
		t.Fatal("the capped read produced a recipe this build accepts")
	}
}

// Tests write to the directory the testing package hands out and nowhere else.
// The ignore file is a second line of defence rather than the first.
func recipeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recipe.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the recipe: %v", err)
	}
	return path
}

const usableRecipe = `version: "0.1"
entities:
  customer:
    count: 10
    fields:
      id: { type: uuid, unique: true }
`

func TestValidateAcceptsAUsableRecipe(t *testing.T) {
	got := run("validate", recipeFile(t, usableRecipe))
	if got.code != 0 {
		t.Fatalf("ended with %d, log: %s", got.code, got.log)
	}
	if got.data != "" {
		t.Errorf("the report belongs on the log channel, found %q on the data channel", got.data)
	}
	if !strings.Contains(got.log, "can be used") {
		t.Errorf("the log does not say the recipe is usable: %q", got.log)
	}
}

// A recipe with one of each, so that the sentence about what was read has to
// say entity and field rather than entities and fields.
const oneOfEachRecipe = usableRecipe

// A recipe with two entities and three fields between them, which is what
// stops the repair from being a spelling.
const severalOfEachRecipe = `version: "0.1"
entities:
  customer:
    count: 2
    fields:
      id:   { type: uuid, unique: true }
      note: { type: uuid }
  order:
    count: 3
    fields:
      id: { type: uuid }
`

// # What this guards, and why it is a test rather than a careful constant
//
// B-204: the window said 1 entity, 1 field and the command line said
// 1 entities and 1 fields about the same recipe, so the two surfaces answered
// one question in two ways over a single line.
//
// Both forms are checked in one test on purpose. A sentence that is always
// singular reads correctly about the first recipe below, and a sentence that
// is always plural reads correctly about the second, so either half alone is a
// guard that a wrong answer walks past.
func TestWhatWasUnderstoodIsCountedInTheSingularWhenThereIsOneOfIt(t *testing.T) {
	one := run(commandValidate, recipeFile(t, oneOfEachRecipe))
	if one.code != 0 {
		t.Fatalf("ended with %d, log: %s", one.code, one.log)
	}
	if !strings.Contains(one.log, "1 entity and 1 field.") {
		t.Errorf("one of each is not written in the singular: %q", one.log)
	}

	several := run(commandValidate, recipeFile(t, severalOfEachRecipe))
	if several.code != 0 {
		t.Fatalf("ended with %d, log: %s", several.code, several.log)
	}
	if !strings.Contains(several.log, "2 entities and 3 fields.") {
		t.Errorf("more than one of each is not written in the plural: %q", several.log)
	}
}

// A recipe that cannot be used says something about the recipe, so the run
// ends with the configuration code and not with a tool failure.
func TestValidateRefusesAnUnusableRecipeWithTheConfigurationCode(t *testing.T) {
	got := run("validate", recipeFile(t, "version: 1.0\n"))
	if got.code != int(codeConfig) {
		t.Fatalf("expected %d, got %d", codeConfig, got.code)
	}
	if got.data != "" {
		t.Errorf("a failed run writes nothing to the data channel, found %q", got.data)
	}
	if !strings.Contains(got.log, "version") {
		t.Errorf("the log does not name the problem: %q", got.log)
	}
}

func TestValidateReportsAMissingFileAsAReadFailure(t *testing.T) {
	got := run("validate", filepath.Join(t.TempDir(), "nowhere.yaml"))
	if got.code != int(codeIO) {
		t.Fatalf("expected %d, got %d", codeIO, got.code)
	}
	if !strings.Contains(got.log, "cannot read") {
		t.Errorf("the log does not say what failed: %q", got.log)
	}
}

func TestValidateNeedsExactlyOneRecipe(t *testing.T) {
	for _, arguments := range [][]string{
		{"validate"},
		{"validate", "a.yaml", "b.yaml"},
	} {
		got := run(arguments...)
		if got.code != int(codeUsage) {
			t.Errorf("%v: expected %d, got %d", arguments, codeUsage, got.code)
		}
		if !strings.Contains(got.log, problemValidateArguments) {
			t.Errorf("%v: unexpected log %q", arguments, got.log)
		}
	}
}

// Answering one of the two would leave the other unanswered, and dropping a
// request without a word is the silence this tool argues against.
func TestACommandAndAReportCannotBeAskedForAtOnce(t *testing.T) {
	for _, arguments := range [][]string{
		{"--version", "validate", "r.yaml"},
		{"--help", "validate", "r.yaml"},
		{"--license", "validate", "r.yaml"},
	} {
		got := run(arguments...)
		if got.code != int(codeUsage) {
			t.Errorf("%v: expected %d, got %d", arguments, codeUsage, got.code)
		}
		if got.data != "" {
			t.Errorf("%v: wrote %q to the data channel", arguments, got.data)
		}
	}
}

func TestAnUnknownCommandNamesTheOnesThatExist(t *testing.T) {
	got := run("frobnicate")
	if got.code != int(codeUsage) {
		t.Fatalf("expected %d, got %d", codeUsage, got.code)
	}
	if !strings.Contains(got.log, commandValidate) {
		t.Fatalf("the failure does not name the commands: %q", got.log)
	}
}

// A cross cutting flag that works before the verb and fails after it is worse
// than one that does not exist, because the failure looks like a typo.
func TestTheMachineReadableFlagWorksOnEitherSideOfTheVerb(t *testing.T) {
	path := recipeFile(t, usableRecipe)
	for _, arguments := range [][]string{
		{"validate", "--json", path},
		{"--json", "validate", path},
	} {
		got := run(arguments...)
		if got.code != 0 {
			t.Errorf("%v ended with %d, log %q", arguments[:2], got.code, got.log)
			continue
		}
		var document map[string]any
		if err := json.Unmarshal([]byte(got.data), &document); err != nil {
			t.Errorf("%v did not produce JSON: %v", arguments[:2], err)
			continue
		}
		if document["usable"] != true {
			t.Errorf("%v: expected a usable recipe, got %v", arguments[:2], document["usable"])
		}
		if got.log != "" {
			t.Errorf("%v: a pipeline reading the document should get nothing beside it, got %q", arguments[:2], got.log)
		}
	}
}

// The rule that a failed run writes no data exists so that half a dataset
// never looks complete. A report asked for by name is the answer to the
// question, and withholding it exactly when it is needed would make the flag
// useless.
func TestTheMachineReadableReportSurvivesAnUnusableRecipe(t *testing.T) {
	got := run("validate", "--json", recipeFile(t, "version: 1.0\n"))
	if got.code != int(codeConfig) {
		t.Fatalf("expected %d, got %d", codeConfig, got.code)
	}
	var document struct {
		Usable   bool `json:"usable"`
		Problems []struct {
			Where string `json:"where"`
			What  string `json:"what"`
		} `json:"problems"`
		Unchecked []struct{} `json:"unchecked"`
	}
	if err := json.Unmarshal([]byte(got.data), &document); err != nil {
		t.Fatalf("the findings are not readable by a machine: %v, data %q", err, got.data)
	}
	if document.Usable {
		t.Error("the document claims a recipe that was refused is usable")
	}
	if len(document.Problems) == 0 {
		t.Error("the document carries no problems, so it says nothing")
	}
	if document.Problems[0].Where == "" || document.Problems[0].What == "" {
		t.Error("a problem has to say where and what, or a pipeline cannot act on it")
	}
}

// An empty list has to arrive as an empty list. A null teaches every consumer
// to check for it.
func TestEmptyListsAreListsAndNotNothing(t *testing.T) {
	got := run("validate", "--json", recipeFile(t, usableRecipe))
	if !strings.Contains(got.data, `"problems":[]`) || !strings.Contains(got.data, `"unchecked":[]`) {
		t.Fatalf("expected empty lists rather than null: %s", got.data)
	}
}

func TestAnUnknownFlagAfterTheVerbIsAUsageFailure(t *testing.T) {
	got := run("validate", "--nope", "recipe.yaml")
	if got.code != int(codeUsage) {
		t.Fatalf("expected %d, got %d", codeUsage, got.code)
	}
	if got.data != "" {
		t.Errorf("a failed run writes nothing to the data channel, found %q", got.data)
	}
}

// The report has to stay visible when the run is quiet about progress, and the
// summary of a clean recipe is exactly what --quiet was asked to drop.
func TestQuietKeepsTheProblemsAndDropsTheSummary(t *testing.T) {
	quiet := run("--quiet", "validate", recipeFile(t, usableRecipe))
	if quiet.code != 0 {
		t.Fatalf("ended with %d", quiet.code)
	}
	if strings.Contains(quiet.log, "can be used") {
		t.Errorf("--quiet still printed the summary: %q", quiet.log)
	}

	loud := run("--quiet", "validate", recipeFile(t, "version: 1.0\n"))
	if !strings.Contains(loud.log, "error: ") {
		t.Errorf("--quiet hid a failure: %q", loud.log)
	}
}

// 🔴 The two verbs have to answer the same question the same way.
//
// Until 2026-08-05 a recipe naming a generator this build lacks passed
// validation with the words "can be used" and no warning of any kind, while
// generate refused the same file. That is worse than a gap: the verb whose whole
// purpose is the sentence "this recipe will work" was answering it wrongly.
// Found by running the binary, with every test green.
func TestValidateRefusesARecipeGenerateWouldRefuse(t *testing.T) {
	dir := t.TempDir()
	// 🔴 TWO unknown generators, not one, and that is the whole strength of this
	// guard. The engine refuses such a recipe as well, and it stops at the first
	// problem, so with one generator a mutation removing this command's own
	// check passed unnoticed: something else refused instead and the exit code
	// was the same. Measured on 2026-09-03, on the full run that first reached
	// this entry. What only this command does is answer for the whole document
	// at once, which is the sentence its own refusal ends with.
	body := `version: "0.1"
seed: "s"
entities:
  customer:
    count: 3
    fields:
      imie: { generator: nosuchthing }
      inne: { generator: alsomissing }
`
	path := recipeIn(t, dir, body)

	checked := run(commandValidate, path)
	produced := run(commandGenerate, path)

	if !strings.Contains(checked.log, "alsomissing") {
		t.Errorf("the report stops at the first problem, and this command exists to give "+
			"every one of them at once: %s", checked.log)
	}

	if checked.code != int(codeConfig) {
		t.Errorf("validate ended with %d and a recipe that cannot be used is %d: %s",
			checked.code, codeConfig, checked.log)
	}
	if checked.code != produced.code {
		t.Errorf("validate says %d and generate says %d about the same file",
			checked.code, produced.code)
	}
	if !strings.Contains(checked.log, "nosuchthing") {
		t.Errorf("the refusal does not name the generator: %s", checked.log)
	}
	if !strings.Contains(checked.log, "first_name") {
		t.Errorf("the refusal does not list what this build does have: %s", checked.log)
	}
}

// 🔴 The same rule, and the second recipe this project found it broken on.
//
// B-205, measured on 2026-09-02 with the binary: a child carrying a field named
// after the entity it belongs to was called usable by validate, counted by a
// dry run, and refused by generate. The refusal told the reader to run validate,
// which had just approved the file, so the tool sent people to the answer that
// was wrong.
//
// Three surfaces are checked rather than two, because the dry run is the half
// that was still disagreeing after the first repair: the refusal used to live in
// the writer, and a dry run builds no writer.
func TestTheThreeAnswersAboutARelationCollidingWithAFieldAgree(t *testing.T) {
	dir := t.TempDir()
	body := `version: "0.1"
seed: "s"
entities:
  customer:
    count: 2
    fields:
      id: { generator: first_name }
  order:
    count: 2
    belongs_to: { customer: 0-2 }
    fields:
      customer: { generator: first_name }
`
	path := recipeIn(t, dir, body)

	checked := run(commandValidate, path)
	planned := run(commandGenerate, "--dry-run", path)
	// 🔴 The output directory is named although this run is expected to refuse
	// before it writes anything. A test that relies on the refusal to keep the
	// disk clean stops being a test the moment the refusal breaks, and it stops
	// being one by writing its files into this package. Found by a mutation of
	// the very rule this test guards, which wrote customer.csv beside the code.
	produced := run(commandGenerate, "--out", filepath.Join(dir, "written"), path)

	for name, got := range map[string]outcome{
		commandValidate: checked, "generate --dry-run": planned, commandGenerate: produced,
	} {
		if got.code != int(codeConfig) {
			t.Errorf("%s ended with %d and a recipe that cannot be used is %d: %s",
				name, got.code, codeConfig, got.log)
		}
		if !strings.Contains(got.log, "customer") {
			t.Errorf("the refusal from %s does not name the column both things want: %s",
				name, got.log)
		}
	}

	// The place is named as the field to rename rather than as the entity,
	// because renaming the entity would be the other way to resolve it and it
	// is not the one this tool recommends.
	if !strings.Contains(checked.log, "entities.order.fields.customer") {
		t.Errorf("validate does not point at the field to rename: %s", checked.log)
	}
}

// 🔴 Every refusal says what to do next, whichever layer it came from.
//
// The rule has always said four parts. The command line kept it where a refusal
// was written out by hand and lost it everywhere a failure arrived carrying a
// kind, which is most of them.
func TestAFailureFromBelowStillSaysWhatToDo(t *testing.T) {
	dir := t.TempDir()
	for _, sample := range []struct {
		name string
		args []string
	}{
		{"a recipe that is not there", []string{commandValidate, filepath.Join(dir, "no.yaml")}},
		{"a recipe that cannot be used", []string{commandGenerate, recipeIn(t, dir, "version: \"0.1\"\nseed: \"s\"\nentities:\n  c:\n    count: 1\n    fields:\n      f: { generator: nope }\n")}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			got := run(sample.args...)
			if !strings.Contains(got.log, "try: ") {
				t.Errorf("the refusal does not say what to do instead:\n%s", got.log)
			}
		})
	}
}

// 🔴 The third recipe this rule was found broken on, and the one that closed a
// class rather than a case.
//
// Measured on 2026-09-02 with the binary: a recipe whose relations form a loop
// was answered can be used. So was one whose entity belongs to two others.
// Neither of those is a matter of taste, and both are refused in a file the
// engine owns and this verb was not asking. Half of what makes a recipe
// unusable was known one function away and never reported.
func TestValidateRefusesARelationTheEngineWillNotOrder(t *testing.T) {
	loop := recipeFile(t, `version: "0.1"
seed: "s"
locale: pl-PL
entities:
  a:
    belongs_to: { b: 0-2 }
    fields:
      x: { generator: city }
  b:
    belongs_to: { a: 0-2 }
    fields:
      y: { generator: city }
`)

	checked := run(commandValidate, loop)
	produced := run(commandGenerate, "--dry-run", loop)

	if checked.code != int(codeConfig) {
		t.Errorf("validate ended with %d and a recipe that cannot be used is %d: %s",
			checked.code, codeConfig, checked.log)
	}
	if checked.code != produced.code {
		t.Errorf("validate says %d and generate says %d about the same file",
			checked.code, produced.code)
	}
	if !strings.Contains(checked.log, "loop") {
		t.Errorf("the refusal does not say what is wrong with the relations: %s", checked.log)
	}
}

// A recipe that reads correctly and cannot be carried out is warned about, and
// the answer is still yes.
//
// 🔴 The two verbs deliberately end with different codes here, which is the
// opposite of the test above and rests on DEC-116. Whether a recipe is written
// correctly and whether a run of it is worth attempting are two questions, the
// second belongs to the engine and has an exit code of its own, and a verb that
// refused it would be answering the first question with the second one's
// answer. What this verb owes is that it does not say can be used about it.
func TestARecipeThatCannotBeRunIsWarnedAboutRatherThanRefused(t *testing.T) {
	wide := recipeFile(t, `version: "0.1"
seed: "s"
locale: pl-PL
entities:
  customer:
    count: 3
    fields:
      x: { generator: city }
  order:
    belongs_to: { customer: "0-9223372036854775807" }
    fields:
      y: { generator: city }
`)

	checked := run(commandValidate, wide)
	produced := run(commandGenerate, "--dry-run", wide)

	if checked.code != int(codeOK) {
		t.Errorf("validate ended with %d about a recipe that reads correctly: %s",
			checked.code, checked.log)
	}
	if produced.code != int(codeData) {
		t.Errorf("the dry run ended with %d and a valid recipe that cannot be satisfied "+
			"is %d: %s", produced.code, codeData, produced.log)
	}
	if strings.Contains(checked.log, "can be used") {
		t.Errorf("validate says a recipe can be used when the run of it would not "+
			"start: %s", checked.log)
	}
	if !strings.Contains(checked.log, "would not be carried out") {
		t.Errorf("validate does not say that a run of it would not happen: %s", checked.log)
	}
	if !strings.Contains(checked.log, "4000000000") {
		t.Errorf("the warning does not carry the arithmetic, so it reads as a rule "+
			"somebody chose: %s", checked.log)
	}
}

// What validate says a run would write is what the dry run says it would
// write.
//
// 🔴 The oracle is the other verb rather than a number written here. A count
// spelled out in a test freezes whatever this build does today, including a
// mistake, and the whole worth of this sentence is that somebody can trust it
// before running anything. Two sentences about one number are also how somebody
// comes to believe there are two numbers.
func TestWhatValidateSaysARunWouldWriteIsWhatTheDryRunSays(t *testing.T) {
	path := recipeFile(t, `version: "0.1"
seed: "s"
locale: pl-PL
entities:
  customer:
    count: 3
    fields:
      x: { generator: city }
  order:
    belongs_to: { customer: 0-4 }
    fields:
      y: { generator: city }
`)

	checked := run(commandValidate, path)
	produced := run(commandGenerate, "--dry-run", path)

	rows := rowsIn(t, produced.log, `([0-9]+) rows would be written`)
	tables := rowsIn(t, produced.log, `([0-9]+) tables?, `)
	wanted := "would write " + counted(rows, "row") + " in " + counted(tables, "table")
	if !strings.Contains(checked.log, wanted) {
		t.Errorf("validate does not say %q while the dry run writes %d rows in %d "+
			"tables.\nvalidate: %s\ndry run: %s", wanted, rows, tables,
			checked.log, produced.log)
	}
}

// rowsIn pulls the one number a pattern captures out of a log.
func rowsIn(t *testing.T, log, pattern string) int {
	t.Helper()
	found := regexp.MustCompile(pattern).FindStringSubmatch(log)
	if found == nil {
		t.Fatalf("the dry run said nothing matching %q, so there is nothing to compare "+
			"against: %s", pattern, log)
	}
	number, err := strconv.Atoi(found[1])
	if err != nil {
		t.Fatalf("the dry run wrote %q where a number was expected: %v", found[1], err)
	}
	return number
}
