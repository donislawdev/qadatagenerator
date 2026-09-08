package recipeview_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
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
		background.Work{Read: background.ReadHere, Produce: produceHere}, nil)
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
	for _, name := range []string{"customer.csv", "order.csv"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("the run did not leave %s behind: %v", name, err)
		}
	}
	if len(written.Files) != 2 || written.Rows != 29 {
		t.Errorf("this recipe is four customers and twenty five orders, and the run "+
			"reported %d files and %d rows", len(written.Files), written.Rows)
	}

	drawn := rendered(t, view)
	if !strings.Contains(drawn, i18n.English.T(i18n.RecipeWrote)) {
		t.Error("the screen does not say what the run wrote")
	}
	if !strings.Contains(drawn, dir) {
		t.Error("the screen does not say where the files are")
	}
}

// 🔴 A button that cannot run says what to do rather than only being grey.
//
// Three reasons and three sentences. A disabled control with no words beside it
// sends somebody looking for what they did wrong, and the one thing this screen
// knows is exactly which of the three it is.
func TestADisabledButtonSaysWhichOfTheThreeThingsIsMissing(t *testing.T) {
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere}, nil)
	view.Content()

	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, i18n.English.T(i18n.RecipeNeedRecipe)) {
		t.Error("an empty screen does not say that a recipe comes first")
	}

	view.SetRecipe(screens.BrokenRecipe)
	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, i18n.English.T(i18n.RecipeNeedFix)) {
		t.Error("a recipe with problems does not say that they come first")
	}

	view.SetRecipe(screens.CleanRecipe)
	if drawn := spoken(rendered(t, view)); !strings.Contains(drawn, i18n.English.T(i18n.RecipeNeedDirectory)) {
		t.Error("a screen with nowhere to write does not say so")
	}

	view.SetOut(t.TempDir())
	if drawn := rendered(t, view); !strings.Contains(drawn, i18n.English.T(i18n.RecipeWouldWrite)) {
		t.Error("a screen with everything it needs does not say what it would write")
	}
}

// A run that was never started leaves nothing on the disk.
//
// The screen is asked to write with nowhere to write to, which is the state its
// own button refuses, and the point is that the refusal is the screen's rather
// than the button's: a caller reaching Start directly gets the same answer.
func TestAScreenWithNowhereToWriteStartsNothing(t *testing.T) {
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere}, nil)
	view.Content()
	view.SetRecipe(screens.CleanRecipe)

	view.Start()

	if _, _, ran := view.Written(); ran {
		t.Error("a screen with nowhere to write started a run anyway")
	}
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
