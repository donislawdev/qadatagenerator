package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
