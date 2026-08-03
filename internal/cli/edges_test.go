package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The edges of the command line, each one found by review or by running the
// built binary rather than by a test failing.

// writeRecipe puts a recipe in a directory of this test and returns its path.
func writeRecipe(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "recipe.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("could not write the recipe: %v", err)
	}
	return path
}

// uncheckedRecipe declares a generator setting, which is the shape this build
// carries and cannot check.
const uncheckedRecipe = `version: "0.1"
entities:
  c:
    fields:
      dob:
        generator: date_of_birth
        min_age: 18
`

// # A warning has to be complete on its own
//
// The headline printed through a channel that survives --quiet and its entries
// through one that does not, so asking for less output left a warning saying
// something had been skipped with nothing anywhere saying what. Measured on the
// built binary before this: one line, no detail, exit nought.
func TestTheUncheckedWarningSurvivesQuietWithItsMeaningIntact(t *testing.T) {
	path := writeRecipe(t, uncheckedRecipe)

	loud := run("validate", path)
	if !strings.Contains(loud.log, "1 declaration was not checked") {
		t.Errorf("the ordinary run did not count what it skipped: %q", loud.log)
	}
	if !strings.Contains(loud.log, "min_age") {
		t.Errorf("the ordinary run did not name what it skipped: %q", loud.log)
	}

	quiet := run("validate", "--quiet", path)
	if !strings.Contains(quiet.log, "1 declaration was not checked") {
		t.Errorf("the quiet run lost the count: %q", quiet.log)
	}
	if !strings.Contains(quiet.log, "--quiet") {
		t.Errorf("the quiet run hid the detail without saying how to see it: %q", quiet.log)
	}
	if strings.Contains(quiet.log, "min_age") {
		t.Errorf("--quiet printed the detail after all: %q", quiet.log)
	}
}

// The count is a count, so more than one has to read as more than one.
func TestTheUncheckedWarningAgreesWithItselfAboutNumber(t *testing.T) {
	path := writeRecipe(t, uncheckedRecipe+"        max_age: 65\n")

	got := run("validate", path)
	if !strings.Contains(got.log, "2 declarations were not checked") {
		t.Fatalf("expected two to read as a plural, got %q", got.log)
	}
}

// # A flag given a value used to be told it did not exist
//
// Every parse failure carried one reason, and that reason said the flag was not
// defined. For a flag handed a value it was untrue, and the line naming what
// would be accepted listed the same flag two rows further down, so the message
// contradicted itself and sent the reader hunting a typo in a correct name.
func TestAFlagGivenAValueIsToldItIsASwitch(t *testing.T) {
	got := run("--no-color=nonsense", "--version")

	if got.code != int(codeUsage) {
		t.Fatalf("ended with %d", got.code)
	}
	if !strings.Contains(got.log, problemFlagTakesNoValueWhy) {
		t.Errorf("the reason did not say the flag is a switch: %q", got.log)
	}
	if strings.Contains(got.log, problemUnknownFlagWhy) {
		t.Errorf("a flag that exists was reported as undefined: %q", got.log)
	}
}

// A flag nobody defines still gets the reason that is true for it.
func TestAFlagNobodyDefinesIsStillReportedAsUndefined(t *testing.T) {
	got := run("--invented")

	if got.code != int(codeUsage) {
		t.Fatalf("ended with %d", got.code)
	}
	if !strings.Contains(got.log, problemUnknownFlagWhy) {
		t.Errorf("expected the undefined reason, got %q", got.log)
	}
}

// The same treatment after the verb, because a cross cutting flag means the
// same thing on either side of it.
func TestAFlagGivenAValueAfterTheVerbIsAnsweredTheSameWay(t *testing.T) {
	got := run("validate", "--json=nonsense", "recipe.yaml")

	if got.code != int(codeUsage) {
		t.Fatalf("ended with %d", got.code)
	}
	if !strings.Contains(got.log, problemFlagTakesNoValueWhy) {
		t.Errorf("the reason did not say the flag is a switch: %q", got.log)
	}
}

// # A directory is answered in our own words
//
// Left to the operating system this read "Incorrect function." on Windows and
// "is a directory" elsewhere, so one mistake produced two messages and neither
// said what to do next.
func TestADirectoryGivenAsARecipeIsRefusedInOurOwnWords(t *testing.T) {
	got := run("validate", t.TempDir())

	if got.code != int(codeIO) {
		t.Fatalf("ended with %d, expected the read failure code", got.code)
	}
	for _, wanted := range []string{
		problemRecipeIsDirectoryWhy,
		problemRecipeIsDirectoryAllowed,
		problemRecipeIsDirectoryDo,
	} {
		if !strings.Contains(got.log, wanted) {
			t.Errorf("the refusal left out %q: %q", wanted, got.log)
		}
	}
	if strings.Contains(got.log, "Incorrect function") {
		t.Errorf("the message from the operating system reached the user: %q", got.log)
	}
}

// A file that genuinely is not there keeps the ordinary read failure, or the
// branch above has swallowed a different case.
func TestAMissingRecipeIsStillAnOrdinaryReadFailure(t *testing.T) {
	got := run("validate", filepath.Join(t.TempDir(), "nothing.yaml"))

	if got.code != int(codeIO) {
		t.Fatalf("ended with %d", got.code)
	}
	if strings.Contains(got.log, problemRecipeIsDirectoryWhy) {
		t.Errorf("a missing file was reported as a directory: %q", got.log)
	}
}

// # The last spelling wins
//
// The flag package takes the last of a repeated flag and this scan took the
// first, so the two disagreed about the same command line. The disagreement
// showed only in colour, which is the hardest kind of difference to notice.
func TestTheLastSpellingOfNoColourWins(t *testing.T) {
	cases := []struct {
		args []string
		off  bool
	}{
		{[]string{"--no-color"}, true},
		{[]string{"--no-color=false"}, false},
		{[]string{"--no-color=false", "--no-color"}, true},
		{[]string{"--no-color", "--no-color=false"}, false},
		{[]string{"--no-color=true", "--no-color=false"}, false},
		{[]string{"--", "--no-color"}, false},
		{[]string{"--no-color", "--", "--no-color=false"}, true},
	}
	for _, one := range cases {
		if got := colorTurnedOff(one.args); got != one.off {
			t.Errorf("%v answered %v, expected %v", one.args, got, one.off)
		}
	}
}
