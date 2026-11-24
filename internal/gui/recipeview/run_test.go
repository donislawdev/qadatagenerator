package recipeview_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/gui/screens"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// ready is the screen with the real check and the real way of writing, so that
// what these tests exercise is the whole path rather than a stand in agreeing
// with them.
func ready(t *testing.T) (*recipeview.View, string) {
	t.Helper()
	dir := t.TempDir()
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere}, nowhere, nil)
	view.Content()
	view.SetRecipe(screens.RelatedRecipe)
	view.SetOut(dir)
	return view, dir
}

// produceHere carries the run out where it is asked for, so that a test can
// look at the answer on the line after it asked for one.
func produceHere(work func() (produce.Written, error), done func(produce.Written, error)) {
	done(work())
}

// 🔴 The one test of this screen that goes the whole way.
//
// A real recipe, the real engine, a real directory. Every other guard here is
// about the drawing of an answer somebody handed the screen, and none of them
// would notice a button that draws a perfect answer and writes nothing.
func TestTheScreenWritesTheDatasetAndSaysWhatItWrote(t *testing.T) {
	view, dir := ready(t)

	view.Start()

	written, failed, ran := view.Written()
	if !ran {
		t.Fatal("the button did not run anything")
	}
	if failed != nil {
		t.Fatalf("the run failed: %v", failed)
	}
	// 🔴 The note beside the data is among them since 2026-09-24 (B-209): the
	// folder a window leaves has to say about itself that its values describe
	// nobody, the same as the folder a terminal leaves.
	for _, name := range []string{"customer.csv", "order.csv", produce.RunNoteName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("the run did not leave %s behind: %v", name, err)
		}
	}
	if len(written.Files) != 3 || written.Rows != 29 {
		t.Errorf("this recipe is four customers and twenty five orders with a note beside "+
			"them, and the run reported %v and %d rows", written.Files, written.Rows)
	}

	drawn := rendered(t, view)
	if !strings.Contains(drawn, i18n.English.T(i18n.RecipeResultWritten)) {
		t.Error("the screen does not say what the run wrote")
	}
	if !strings.Contains(drawn, dir) {
		t.Error("the screen does not say where the files are")
	}
}

// 🔴 A button that cannot run says what to do rather than only being grey, and
// since 2026-09-15 it says it in the card beside the button rather than under it
// (DEC-246).
//
// The empty screen says what to do, a recipe with problems is its own reason,
// and a clean recipe with nowhere to write is asked for the directory. A
// disabled control with no words beside it sends somebody looking for what they
// did wrong, and the card is where the one thing this screen knows is said.
func TestADisabledButtonSaysWhatIsMissingInTheCard(t *testing.T) {
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere}, nowhere, nil)
	view.Content()

	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, i18n.English.T(i18n.RecipeEmpty)) {
		t.Error("an empty screen does not say what to do")
	}

	view.SetRecipe(screens.BrokenRecipe)
	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, i18n.English.T(i18n.RecipeHeadingProblems)) {
		t.Error("a recipe with problems does not name them")
	}

	view.SetRecipe(screens.CleanRecipe)
	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, i18n.English.T(i18n.RecipeNeedDirectory)) {
		t.Error("a clean recipe with nowhere to write does not ask for a directory")
	}

	// The directory is drawn verbatim under the files a run would write, so the
	// path it was given is what says the screen now knows where to write. The
	// heading over it is set in capitals by the theme, which is why the path and
	// not the heading is what this looks for.
	dir := t.TempDir()
	view.SetOut(dir)
	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, dir) {
		t.Error("a screen with everything it needs does not say where it would write")
	}
}

// A run that was never started leaves nothing on the disk.
//
// The screen is asked to write with nowhere to write to, which is the state its
// own button refuses, and the point is that the refusal is the screen's rather
// than the button's: a caller reaching Start directly gets the same answer.
func TestAScreenWithNowhereToWriteStartsNothing(t *testing.T) {
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere}, nowhere, nil)
	view.Content()
	view.SetRecipe(screens.CleanRecipe)

	view.Start()

	if _, _, ran := view.Written(); ran {
		t.Error("a screen with nowhere to write started a run anyway")
	}
}

// readyWith is the screen with the real check and the real way of writing, and
// a recipe of the test's choosing on it.
func readyWith(t *testing.T, recipe string) (*recipeview.View, string) {
	t.Helper()
	dir := t.TempDir()
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere}, nowhere, nil)
	view.Content()
	view.SetRecipe(recipe)
	view.SetOut(dir)
	return view, dir
}

// 🔴 A recipe with a policy is damaged by this screen, with a manifest beside
// the data, and the card says what the damage did.
//
// This is B-300 held against a disk from the window's side. Until 2026-09-24
// the screen wrote such a recipe clean and wrote no manifest, while the command
// line wrote it damaged, and nothing on either side said so. The run is now the
// one both surfaces call (DEC-252), so what is asked here is that the screen
// reaches it and says what came of it.
func TestARecipeWithAPolicyIsDamagedByTheScreenAndItSaysWhatTheDamageDid(t *testing.T) {
	view, dir := readyWith(t, screens.PolicyRecipe)

	view.Start()

	written, failed, ran := view.Written()
	if !ran || failed != nil {
		t.Fatalf("the run did not finish: ran %v, failed %v", ran, failed)
	}
	if !written.Damaged || written.Counts.Requested == 0 {
		t.Errorf("a recipe ordering damage to every record ran as %+v", written.Counts)
	}
	if _, err := os.Stat(filepath.Join(dir, manifestjson.Name)); err != nil {
		t.Errorf("a recipe with a policy left no manifest: %v", err)
	}
	drawn := spoken(rendered(t, view))
	for _, key := range []i18n.Key{i18n.TakeResultOrdered, i18n.TakeResultApplied} {
		if !strings.Contains(drawn, i18n.English.T(key)) {
			t.Errorf("the card after a damaging run does not say %q", i18n.English.T(key))
		}
	}
}

// Every setting of a run reaches it from this screen, which is what parity
// with generate --seed, --format and --manifest means here.
//
// # Why one test for three flags, and why two of them have no control
//
// The seed and the form of the output are written in the recipe, and the
// window edits the recipe, so a control for either would be a second place to
// say one thing two lines apart (DEC-199, DEC-252). What proves the parity for
// them is that the value written in the text is the value the run used. The
// extra form of the manifest belongs to a run rather than to a recipe
// (DEC-163), so it is a control, and what proves it is the file it adds. All
// three are read back out of what the run wrote, with values nobody would
// arrive at by default, and the names on the card before the button are held
// against the files that appeared.
func TestEverySettingOfARunReachesItFromTheScreen(t *testing.T) {
	const seed = "chosen-in-the-recipe"
	view, dir := readyWith(t, `version: "0.1"
seed: "`+seed+`"
locale: pl-PL
entities:
  customer:
    count: 3
    fields:
      pesel:
        generator: pesel
policy:
  layers: [L3]
  rows: 100%
output:
  format: json
`)
	form := manifestexport.Forms()[0]
	view.ChooseForm(form)
	before := spoken(rendered(t, view))

	view.Start()

	written, failed, ran := view.Written()
	if !ran || failed != nil {
		t.Fatalf("the run did not finish: ran %v, failed %v", ran, failed)
	}
	// The object form writes the whole dataset as one file, which no run of the
	// comma separated form writes, so its presence is the format arriving.
	if _, err := os.Stat(filepath.Join(dir, "dataset.json")); err != nil {
		t.Errorf("output.format in the recipe did not reach the run: %v", err)
	}
	if note := noteIn(t, dir); note.Seed != seed {
		t.Errorf("the seed in the recipe did not reach the run: the note carries %q", note.Seed)
	}
	if _, err := os.Stat(filepath.Join(dir, manifestexport.FileFor(form))); err != nil {
		t.Errorf("the extra form %q was chosen and its file is not there: %v", form, err)
	}
	for _, name := range written.Files {
		if !strings.Contains(before, name) {
			t.Errorf("the run wrote %q and the card did not say it would", name)
		}
	}
}

// An extra form of the manifest on a recipe that orders no damage says, before
// the run and after it, that there is nothing to export, and nothing is.
//
// 🔴 The command line ignored --manifest here without a word (B-301), and a
// window that greyed the choice out would say nothing either. The control
// stays alive and the card speaks (DEC-252).
func TestAnExtraFormOnARecipeWithoutAPolicySaysItHasNothingToExport(t *testing.T) {
	view, dir := readyWith(t, screens.CleanRecipe)
	view.ChooseForm(manifestexport.Forms()[0])
	said := i18n.English.T(i18n.RecipeFormUnused)

	if !strings.Contains(spoken(rendered(t, view)), said) {
		t.Error("before the run the card does not say that the form has nothing to export")
	}
	view.Start()
	if !strings.Contains(spoken(rendered(t, view)), said) {
		t.Error("after the run the card does not say that no manifest was written")
	}
	if _, err := os.Stat(filepath.Join(dir, manifestjson.Name)); err == nil {
		t.Error("a recipe without a policy left a manifest")
	}
}

// 🔴 The note a run from this screen leaves says that the directory came from
// outside the recipe, the way the note from a terminal says it of --out
// (DEC-255).
//
// Until 2026-10-13 it said nothing (B-311): one recipe through both surfaces
// gave the same data and two notes that differed in this record alone. The
// record has two sides, so there are two recipes: one naming no directory,
// whose side is empty, and one naming a directory the field moved away from.
// The second also asks that the run went where the field said, rather than
// where the recipe did.
func TestTheNoteSaysTheDirectoryCameFromTheScreenTheWayItSaysItOfTheFlag(t *testing.T) {
	for _, each := range []struct {
		name, recipe, fromRecipe string
	}{
		{"the recipe names no directory", screens.PolicyRecipe, ""},
		{"the recipe names another directory", screens.PolicyRecipe + "output:\n  dir: elsewhere\n", "elsewhere"},
	} {
		t.Run(each.name, func(t *testing.T) {
			view, dir := readyWith(t, each.recipe)

			view.Start()

			if _, failed, ran := view.Written(); !ran || failed != nil {
				t.Fatalf("the run did not finish: ran %v, failed %v", ran, failed)
			}
			want := produce.Override{Setting: "output.dir", FromRecipe: each.fromRecipe, FromFlag: dir}
			if note := noteIn(t, dir); len(note.Overrides) != 1 || note.Overrides[0] != want {
				t.Errorf("the note carries %+v rather than %+v", note.Overrides, want)
			}
		})
	}
}

// noteIn reads the record of the run this screen left beside its data.
func noteIn(t *testing.T, dir string) produce.RunNote {
	t.Helper()
	read, err := os.ReadFile(filepath.Join(dir, produce.RunNoteName))
	if err != nil {
		t.Fatalf("the run left no note: %v", err)
	}
	var note produce.RunNote
	if err := json.Unmarshal(read, &note); err != nil {
		t.Fatalf("the note cannot be read: %v", err)
	}
	return note
}

// 🔴 The names on screen before the button is pressed are the names the run
// leaves behind.
//
// A sentence promising customer.csv beside a run that writes dataset.json is
// the kind of wrong answer nobody checks, because both halves look right on
// their own.
func TestWhatTheScreenSaysItWouldWriteIsWhatTheRunWrites(t *testing.T) {
	view, dir := ready(t)

	before := rendered(t, view)
	view.Start()

	written, _, _ := view.Written()
	for _, name := range written.Files {
		if !strings.Contains(before, name) {
			t.Errorf("the run wrote %q and the screen did not say it would", name)
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("the screen named %q and it is not there: %v", name, err)
		}
	}
}
