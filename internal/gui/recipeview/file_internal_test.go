package recipeview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
)

// The answers of the questions this screen asks, in the order they stand.
const (
	answerCancel  = 0
	answerDiscard = 1
	answerSave    = 2
)

// filed is a screen whose reads and saves happen where they are asked for, with
// the closing of a window behind it that a test presses the cross of.
func filed(t *testing.T) (*View, *watchedWindow) {
	t.Helper()
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the catalog inside this binary could not be read: %v", err)
	}
	window := &watchedWindow{}
	closing := background.NewClosing(window, nil)
	view := New(i18n.English, Checker(described), nil, background.Work{
		Read: background.ReadHere, Hold: closing.Hold, AskFirst: closing.AskFirst,
	}, folderless, nil)
	view.Content()
	return view, window
}

func put(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func onDisk(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// answer gives an answer to the question the screen is waiting on, and fails
// the test when it waits on none.
func answer(t *testing.T, view *View, at int) {
	t.Helper()
	if view.Asked() == nil {
		t.Fatal("the screen asks nothing, so there is nothing to answer")
	}
	view.Asked().Answer(at)
}

// 🔴 A file opened here is saved back with the ends of its lines, and the line
// beside the heading says whether what is on the screen is in the file.
func TestARecipeFileOpensIntoTheFieldAndSavesBack(t *testing.T) {
	view, _ := filed(t)
	path := filepath.Join(t.TempDir(), "customers.yaml")
	put(t, path, strings.ReplaceAll(cleanEnough, "\n", "\r\n"))

	view.OpenFile(path)
	if view.editor.Text != cleanEnough {
		t.Fatalf("the field holds %q", view.editor.Text)
	}
	if view.File() != path || view.Unsaved() {
		t.Fatalf("the file of the screen is %q and unsaved is %v", view.File(), view.Unsaved())
	}
	if view.fileName.Text != "customers.yaml" {
		t.Errorf("the line beside the heading says %q", view.fileName.Text)
	}

	view.SetRecipe(cleanEnough + "# kept\n")
	if !view.Unsaved() || !strings.Contains(view.fileName.Text, "not saved") {
		t.Fatalf("an edit is not said: unsaved %v, the line says %q", view.Unsaved(), view.fileName.Text)
	}
	view.Save(nil)
	want := strings.ReplaceAll(cleanEnough+"# kept\n", "\n", "\r\n")
	if got := onDisk(t, path); got != want {
		t.Errorf("saved %q, want %q", got, want)
	}
	if view.Unsaved() || view.fileName.Text != "customers.yaml" {
		t.Errorf("after the save unsaved is %v and the line says %q", view.Unsaved(), view.fileName.Text)
	}
}

// 🔴 The cross asks before a recipe nobody saved is lost, and only the answer
// letting it go closes the window.
//
// Measured on 2026-11-25 before this existed (UX-GUI-001): the cross ended the
// window at once and the recipe on the screen was gone.
func TestTheCrossAsksBeforeAnUnsavedRecipeIsLost(t *testing.T) {
	view, window := filed(t)
	view.SetRecipe(cleanEnough)

	window.intercept()
	answer(t, view, answerCancel)
	if window.closed != 0 {
		t.Fatal("staying closed the window")
	}
	window.intercept()
	answer(t, view, answerDiscard)
	if window.closed != 1 {
		t.Errorf("letting the recipe go closed the window %d times rather than once", window.closed)
	}
}

// A recipe that is in its file closes without a question, as a window with
// nothing at stake should.
func TestASavedRecipeClosesWithoutAsking(t *testing.T) {
	view, window := filed(t)
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, cleanEnough)
	view.OpenFile(path)

	window.intercept()
	if view.Asked() != nil {
		t.Fatalf("a saved recipe was asked about: %q", view.Asked().Says())
	}
	if window.closed != 1 {
		t.Errorf("the window closed %d times rather than once", window.closed)
	}
}

// Saving from the question writes the file and then lets the window go.
func TestSavingFromTheQuestionClosesOnceSaved(t *testing.T) {
	view, window := filed(t)
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, cleanEnough)
	view.OpenFile(path)
	view.SetRecipe(cleanEnough + "# edited\n")

	window.intercept()
	if view.Asked() == nil {
		t.Fatal("an edited recipe was not asked about")
	}
	if !strings.HasPrefix(view.Asked().Says(), "r.yaml ") {
		t.Errorf("the question does not name the file: %q", view.Asked().Says())
	}
	answer(t, view, answerSave)
	if onDisk(t, path) != cleanEnough+"# edited\n" {
		t.Error("the answer saving the recipe did not save it")
	}
	if window.closed != 1 {
		t.Errorf("the window closed %d times after the save rather than once", window.closed)
	}
}

// A recipe somebody wrote is asked about before a dropped file takes the field,
// and stays when the person stays.
func TestADroppedFileAsksBeforeItTakesTheField(t *testing.T) {
	view, _ := filed(t)
	path := filepath.Join(t.TempDir(), "other.yaml")
	put(t, path, cleanEnough)
	view.SetRecipe("mine: 1\n")
	dropped := []fyne.URI{storage.NewFileURI(path)}

	view.Dropped(dropped)
	answer(t, view, answerCancel)
	if view.editor.Text != "mine: 1\n" || view.File() != "" {
		t.Fatalf("staying let the dropped file in: the field holds %q", view.editor.Text)
	}
	view.Dropped(dropped)
	answer(t, view, answerDiscard)
	if view.editor.Text != cleanEnough || view.File() != path {
		t.Errorf("the dropped file did not take the field: %q from %q", view.editor.Text, view.File())
	}
}

// 🔴 A file changed by another program after it was opened here is asked about
// before a save throws that change away.
func TestAFileChangedBehindTheWindowIsAskedAboutBeforeASave(t *testing.T) {
	view, _ := filed(t)
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, cleanEnough)
	view.OpenFile(path)
	put(t, path, "theirs: 1\n")
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
	view.SetRecipe(cleanEnough + "# mine\n")

	view.Save(nil)
	answer(t, view, 0)
	if onDisk(t, path) != "theirs: 1\n" {
		t.Fatal("cancelling replaced the change another program made")
	}
	view.Save(nil)
	answer(t, view, 1)
	if onDisk(t, path) != cleanEnough+"# mine\n" {
		t.Error("agreeing to replace the file did not save the recipe")
	}
}

// A file the field cannot hold leaves the screen as it was and says why, in the
// words of the decoder.
func TestAFileThatCannotBeOpenedSaysWhyAndChangesNothing(t *testing.T) {
	view, _ := filed(t)
	path := filepath.Join(t.TempDir(), "ansi.yaml")
	put(t, path, "name: za\xbf\xf3\xb3\xe6\n")

	view.OpenFile(path)
	if view.Asked() == nil || !strings.Contains(view.Asked().Says(), "UTF-8") {
		t.Fatalf("the refusal was not said, or not why: %v", view.Asked())
	}
	if view.editor.Text != "" || view.File() != "" {
		t.Errorf("a refused file reached the screen: %q from %q", view.editor.Text, view.File())
	}
}

// Ctrl+S in the field saves, which is where a person writing presses it.
func TestControlSInTheFieldSaves(t *testing.T) {
	view, _ := filed(t)
	path := filepath.Join(t.TempDir(), "r.yaml")
	put(t, path, cleanEnough)
	view.OpenFile(path)
	view.SetRecipe(cleanEnough + "# typed\n")

	view.editor.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault})
	if onDisk(t, path) != cleanEnough+"# typed\n" {
		t.Error("Ctrl+S in the field did not save")
	}
}

// The line for a terminal names the real file once there is one, can then be
// copied, and says when it would run something other than the screen shows.
func TestTheCommandNamesTheFileOnceThereIsOne(t *testing.T) {
	view, _ := filed(t)
	view.SetRecipe(cleanEnough)
	if !view.copyCommand.Disabled() || !strings.Contains(view.commandText.Text, i18n.English.T(i18n.RecipeCommandFile)) {
		t.Fatal("before any file the line should carry the example name and the copy be out of reach")
	}

	path := filepath.Join(t.TempDir(), "saved.yaml")
	view.SaveTo(path)
	if onDisk(t, path) != cleanEnough {
		t.Fatal("Save as wrote something else")
	}
	if !strings.Contains(view.commandText.Text, path) || view.copyCommand.Disabled() {
		t.Errorf("after a save the line is %q and the copy disabled is %v",
			view.commandText.Text, view.copyCommand.Disabled())
	}
	if view.commandNote.Text != i18n.English.T(i18n.RecipeCommandSaved) {
		t.Errorf("the note says %q", view.commandNote.Text)
	}
	view.SetRecipe(cleanEnough + "# more\n")
	if view.commandNote.Text != i18n.English.T(i18n.RecipeCommandUnsaved) {
		t.Errorf("with changes not saved the note says %q", view.commandNote.Text)
	}
}
