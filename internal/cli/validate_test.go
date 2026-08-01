package cli

import (
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

func TestJSONIsRefusedForACommandThatCannotProduceIt(t *testing.T) {
	got := run("--json", "validate", "r.yaml")
	if got.code != int(codeUsage) {
		t.Fatalf("expected %d, got %d", codeUsage, got.code)
	}
	if !strings.Contains(got.log, problemJSONUnsupported) {
		t.Fatalf("unexpected log %q", got.log)
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
