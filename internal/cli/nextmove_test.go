package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
