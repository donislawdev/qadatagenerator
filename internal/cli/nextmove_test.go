package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// The fourth part of a refusal says what to do instead. These two hold the
// cases where the sentence chosen by the kind of failure was not merely thin
// but wrong, each in its own way: one contradicted the line above it, and the
// other sent the reader to the command they had just run.

// 🔴 A destination that is taken is a write failure, and the sentence for a
// write failure asks the reader to check that the path exists and that this
// account may write there. Both are true here and neither is the matter: the
// run refused because something is already standing where it wants to put a
// file, and it had just named the file to delete on the line above.
func TestARefusalThatKnowsTheNextMoveDoesNotGetTheOneForItsKind(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	path := recipeIn(t, dir, smallRecipe(out))

	if first := run(commandGenerate, path); first.code != 0 {
		t.Fatalf("the first run ended with %d: %s", first.code, first.log)
	}
	second := run(commandGenerate, path)

	if second.code != int(codeIO) {
		t.Fatalf("the second run ended with %d rather than %d: %s",
			second.code, codeIO, second.log)
	}
	if strings.Contains(second.log, doAboutIO) {
		t.Errorf("the refusal names the file in the way and then asks the reader to "+
			"check their permissions: %q", second.log)
	}
	if !strings.Contains(second.log, "allow replacing") {
		t.Errorf("the refusal carries no next move of its own: %q", second.log)
	}
}

// 🔴 Standard output has no path. Until 2026-11-02 a write to it that failed,
// because the program reading it had left, ended with the sentence for every
// write failure, asking the reader to check that a path exists. Both ways a run
// ends are asked, because each wrapped the failure on its own: the version ends
// through finish, and quick, whose drawing stops at the failed write, through
// finishFailed.
//
// 🔴 The count is past one batch of the data channel on purpose. Three values
// fit in one batch, which is written only after the drawing is done, so the
// drawing itself never fails and the run ends through finish like the version
// does. A mutation of finishFailed survived exactly that on 2026-11-02.
func TestAFailedWriteToTheOutputDoesNotSendTheReaderToAPath(t *testing.T) {
	for _, args := range [][]string{
		{"--version"},
		{"quick", "pesel", "--count", "100", "--seed", "the-reader-left"},
	} {
		var log bytes.Buffer
		code := Run(Environment{Args: args, Stdout: refusingWriter{}, Stderr: &log})

		if code != int(codeIO) {
			t.Errorf("%v ended with %d rather than %d: %s", args, code, codeIO, log.String())
		}
		if strings.Contains(log.String(), doAboutIO) {
			t.Errorf("%v sends the reader to check a path: %q", args, log.String())
		}
		if !strings.Contains(log.String(), doAboutOutput) {
			t.Errorf("%v does not say what to do about output nobody took: %q", args, log.String())
		}
	}
}

// 🔴 Measured on two different recipes on 2026-09-02: validate ended its own
// refusal with the sentence telling the reader that qadg validate reports every
// problem a recipe has. It is a good sentence for the verb that writes data and
// an empty one for the verb that is already the answer.
func TestValidateDoesNotSendTheReaderToValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recipe.yaml")
	// A document that is valid YAML and is not a recipe, so that the refusal
	// comes from the reader rather than from the report of problems.
	if err := os.WriteFile(path, []byte("- one\n- two\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := run(commandValidate, path)

	if got.code != int(codeConfig) {
		t.Fatalf("the run ended with %d rather than %d: %s", got.code, codeConfig, got.log)
	}
	if strings.Contains(got.log, doAboutConfig) {
		t.Errorf("validate tells the reader to run validate: %q", got.log)
	}
	if !strings.Contains(got.log, doAboutConfigValidate) {
		t.Errorf("the refusal has no fourth part of its own: %q", got.log)
	}
}

// A request that cannot be met sends each verb to what it was given (B-351).
// Until 2026-11-03 every verb without a recipe was told to change "the file",
// which is the corrupting verb's file, and the verb drawing values reads none.
// No refusal of that verb reaches this today, because the one it can meet
// carries its own move, so the rule is asked of the function that decides it,
// the same way the recipe advice is.
func TestADataFailureSendsEachVerbToWhatItWasGiven(t *testing.T) {
	for _, one := range []struct {
		verb string
		want string
	}{
		{commandGenerate, doAboutData},
		{commandCorrupt, doAboutDataCorrupt},
		{commandVerify, doAboutConfigDocument},
		{commandQuick, doAboutConfigFlags + commandQuick + doAboutConfigFlagsTail},
	} {
		if said := whatToDoAbout(fault.Data, one.verb); said != one.want {
			t.Errorf("a request %s cannot meet is followed by %q, want %q", one.verb, said, one.want)
		}
	}
}
