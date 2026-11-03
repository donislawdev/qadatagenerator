package cli

import (
	"flag"
	"io"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// 🔴 # Flags after the path, B-97 and DEC-272
//
// Until 2026-11-03 four verbs of five stopped reading flags at the first path,
// so qadg generate recipe.yaml --out ./data was refused as two recipes, with a
// suggestion to type what had just been typed. Measured on the built binary
// before the change: generate, validate, corrupt and verify all ended with 2,
// and verify read a -x after the manifest as the name of the response record.

// The split itself, against a set with one flag of each kind.
func TestTheFlagsOfAVerbAreReadWhereverTheyStand(t *testing.T) {
	for _, given := range []struct {
		what      string
		arguments []string
		paths     []string
		out       string
		dryRun    bool
	}{
		{"a flag after the path", []string{"r.yaml", "--out", "data"}, []string{"r.yaml"}, "data", false},
		{"a flag before the path", []string{"--out", "data", "r.yaml"}, []string{"r.yaml"}, "data", false},
		{"a flag between two paths", []string{"m.json", "--dry-run", "p.json"}, []string{"m.json", "p.json"}, "", true},
		{"a value that looks like a path", []string{"--out", "r.yaml", "s.yaml"}, []string{"s.yaml"}, "r.yaml", false},
		{"a value written with an equals sign", []string{"r.yaml", "--out=data"}, []string{"r.yaml"}, "data", false},
		{"one dash rather than two", []string{"r.yaml", "-out", "data"}, []string{"r.yaml"}, "data", false},
		{"a switch leaves the path after it alone", []string{"--dry-run", "r.yaml"}, []string{"r.yaml"}, "", true},
		{"a double dash taken as the value it follows", []string{"--out", "--", "r.yaml"}, []string{"r.yaml"}, "--", false},
		{"a double dash ends the flags", []string{"--dry-run", "--", "-r.yaml", "--out"}, []string{"-r.yaml", "--out"}, "", true},
		{"a dash on its own is a path", []string{"-", "--dry-run"}, []string{"-"}, "", true},
		{"nothing but flags", []string{"--dry-run"}, nil, "", true},
	} {
		t.Run(given.what, func(t *testing.T) {
			set, out, dryRun := splitSet()
			paths, err := parseAnywhere(set, given.arguments)
			if err != nil {
				t.Fatalf("%v was refused: %v", given.arguments, err)
			}
			if !reflect.DeepEqual(paths, given.paths) || *out != given.out || *dryRun != given.dryRun {
				t.Errorf("%v gave the paths %q, --out %q and --dry-run %v, want %q, %q and %v",
					given.arguments, paths, *out, *dryRun, given.paths, given.out, given.dryRun)
			}
		})
	}
}

// A flag the parse refuses is refused whichever side of the path it stands.
func TestAFlagAfterThePathIsRefusedForWhatIsWrongWithIt(t *testing.T) {
	for _, given := range []struct {
		arguments []string
		says      string
	}{
		{[]string{"r.yaml", "--out"}, "flag needs an argument: -out"},
		{[]string{"r.yaml", "--outt", "data"}, "flag provided but not defined: -outt"},
	} {
		set, out, _ := splitSet()
		paths, err := parseAnywhere(set, given.arguments)
		if err == nil || !strings.Contains(err.Error(), given.says) {
			t.Errorf("%v: want a refusal saying %q, got %v", given.arguments, given.says, err)
		}
		if *out != "" || len(paths) == 0 || paths[0] != "r.yaml" {
			t.Errorf("%v: the path went into a flag, --out is %q and the paths %q", given.arguments, *out, paths)
		}
	}
}

func splitSet() (*flag.FlagSet, *string, *bool) {
	set := flag.NewFlagSet("split", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	return set, set.String("out", "", ""), set.Bool("dry-run", false, "")
}

// --help after a path is answered by every verb that reads flags, which is a
// question about every verb at once: one parsing its flags the old way prints
// no help here. help itself is left out because it reads no flags at all.
func TestEveryVerbAnswersHelpAskedForAfterItsPath(t *testing.T) {
	for _, doc := range commandDocs {
		if doc.name == commandHelp {
			continue
		}
		before := run(doc.name, "--help")
		after := run(doc.name, "anything", "--help")
		if after.code != int(codeOK) || after.data != before.data || after.data == "" {
			t.Errorf("qadg %s anything --help ended with %d and printed %q, want the help it prints for "+
				"qadg %s --help. Log %q", doc.name, after.code, after.data, doc.name, after.log)
		}
	}
}

// The same flags on the other side of the path are the same run, down to the
// last byte of both channels.
func TestAFlagAfterThePathMeansWhatItMeansBeforeIt(t *testing.T) {
	dir := t.TempDir()
	recipe := recipeIn(t, dir, smallRecipe(filepath.Join(dir, "out")))
	records := somebodyElsesFile(t, 3)
	manifest := emptyManifest(t, dir)
	response := responseFor(t, dir, `{"spec_version":"0.2","rows":[]}`)
	for _, pair := range [][2][]string{
		{{"generate", recipe, "--dry-run"}, {"generate", "--dry-run", recipe}},
		{{"validate", recipe, "--json"}, {"validate", "--json", recipe}},
		{{"corrupt", records, "--layers", "L3", "--seed", "s", "--dry-run"},
			{"corrupt", "--layers", "L3", "--seed", "s", "--dry-run", records}},
		{{"verify", manifest, response, "--json"}, {"verify", "--json", manifest, response}},
		{{"verify", manifest, "--json", response}, {"verify", "--json", manifest, response}},
		{{"quick", "pesel", "--count", "2", "--seed", "s"}, {"quick", "--count", "2", "--seed", "s", "pesel"}},
		{{"quick", "--count", "2", "pesel", "--seed", "s"}, {"quick", "--count", "2", "--seed", "s", "pesel"}},
	} {
		after, before := run(pair[0]...), run(pair[1]...)
		if after.code != int(codeOK) || after != before {
			t.Errorf("qadg %v ended with %d, data %q and log %q, and qadg %v with %d, data %q and log %q",
				pair[0], after.code, after.data, after.log, pair[1], before.code, before.data, before.log)
		}
	}
}

// A misspelt flag after the path is reported as the flag it is, where it used
// to be counted as a second recipe and never named.
func TestAMisspeltFlagAfterThePathIsNamed(t *testing.T) {
	got := run("generate", "recipe.yaml", "--outt", "data")
	if got.code != int(codeUsage) || !strings.Contains(got.log, "-outt") ||
		strings.Contains(got.log, problemGenerateArguments) {
		t.Errorf("ended with %d and said %q, want the misspelt flag named and no word about recipes", got.code, got.log)
	}
}

// A flag missing its value was told it does not exist, beside a list of
// accepted flags naming it. Reading flags after the path made this the likely
// slip, because the value is the last thing typed.
func TestAFlagWrittenLastWithoutItsValueIsToldItTakesOne(t *testing.T) {
	for _, arguments := range [][]string{
		{"generate", "recipe.yaml", "--out"},
		{"corrupt", "records.csv", "--layers"},
		{"quick", "pesel", "--count"},
	} {
		got := run(arguments...)
		if got.code != int(codeUsage) || !strings.Contains(got.log, problemFlagNeedsValueWhy) ||
			strings.Contains(got.log, problemUnknownFlagWhy) {
			t.Errorf("%v ended with %d and said %q, want the reason that the flag takes a value",
				arguments, got.code, got.log)
		}
	}
}
