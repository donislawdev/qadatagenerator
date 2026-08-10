package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
)

// policyRecipe is the path of the recipe that carries a policy.
//
// It is worked out before anything changes the working directory, because a run
// writes into it and a second call would then resolve the path against the
// temporary directory of the first.
func policyRecipe(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("testdata", "policy", "recipe.yaml"))
	if err != nil {
		t.Fatalf("cannot find the recipe with a policy: %v", err)
	}
	return path
}

// readManifest runs the recipe that carries a policy and reads what came out.
func readManifest(t *testing.T, recipe string) (map[string]any, outcome, string) {
	t.Helper()
	made := t.TempDir()
	t.Chdir(made)

	got := run(commandGenerate, recipe)
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}
	raw, err := os.ReadFile(filepath.Join(made, manifestjson.Name))
	if err != nil {
		t.Fatalf("the run left no manifest: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("the manifest does not parse: %v", err)
	}
	return document, got, string(raw)
}

// The whole path, from a recipe with a policy to a manifest on disk.
//
// # What this covers that the pieces do not
//
// Every part of this has its own test: choosing the orders, applying one, and
// writing an entry. None of them notices if the parts are wired up wrongly,
// and that is the half a run consists of. This is also the shape of the check
// that found two real defects at the first attempt, both of them entries the
// manifest model refused for locations nobody had thought about.
func TestARecipeWithAPolicyProducesAManifestBesideTheData(t *testing.T) {
	document, got, _ := readManifest(t, policyRecipe(t))

	entries, ok := document["defects"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatal("the manifest carries no entries at all")
	}
	envelope, ok := document["run"].(map[string]any)
	if !ok {
		t.Fatal("the manifest carries no envelope")
	}
	if requested := envelope["defects_requested"].(float64); int(requested) != len(entries) {
		t.Errorf("the envelope says %v defects were ordered and the list holds %d",
			requested, len(entries))
	}
	if !strings.Contains(got.log, "defects ordered") {
		t.Errorf("the run said nothing about the damage it did: %q", got.log)
	}
}

// 🔴 Every kind of damage the catalog describes gets an entry, including the
// ones this build cannot carry out.
//
// This is the sentence the whole step was built to make true. A build that
// silently left out the fifty one it has no implementation for would produce a
// manifest that reads like a complete run, which is the exact failure this
// product exists to find in other people's tools.
func TestEveryDescribedKindOfDamageReachesTheManifest(t *testing.T) {
	document, _, _ := readManifest(t, policyRecipe(t))

	named := map[string]bool{}
	unapplied := 0
	for _, raw := range document["defects"].([]any) {
		entry := raw.(map[string]any)
		named[entry["mutator"].(string)] = true
		if applied, _ := entry["applied"].(bool); !applied {
			if reason, _ := entry["failure_reason"].(string); reason == "" {
				t.Fatalf("%v was not applied and does not say why", entry["id"])
			}
			unapplied++
		}
	}
	// The recipe names every layer the catalog carries entries in, and damages
	// every row, so each described kind has a turn before any repeats.
	if len(named) < 52 {
		t.Errorf("only %d kinds of damage reached the manifest, and the catalog describes 52", len(named))
	}
	if unapplied == 0 {
		t.Error("nothing was refused, which cannot be true while one mutator is implemented")
	}
}

// The same seed produces the same manifest, byte for byte. Damage that moved
// between runs would make the second pillar useless in a pipeline.
func TestTheManifestIsTheSameForTheSameSeed(t *testing.T) {
	recipe := policyRecipe(t)
	_, _, first := readManifest(t, recipe)
	_, _, second := readManifest(t, recipe)

	firstLines, secondLines := withoutTimestamp(first), withoutTimestamp(second)
	if firstLines != secondLines {
		t.Error("two runs of one seed produced different manifests")
	}
}

// withoutTimestamp drops the line carrying when the run happened, which is the
// one thing in a manifest that is allowed to differ between two runs.
func withoutTimestamp(document string) string {
	var kept []string
	for _, line := range strings.Split(document, "\n") {
		if strings.Contains(line, "generated_at") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
