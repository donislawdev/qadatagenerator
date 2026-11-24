package recipeview_test

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/gui/screens"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// shelf is a file manager that remembers which folders it was asked to show.
type shelf struct{ shown []string }

func (s *shelf) ShowFolder(path string) error {
	s.shown = append(s.shown, path)
	return nil
}

// folderButtonOn is the Open folder button a person can see on the screen, or
// nothing when there is none.
//
// It is looked for in the laid out window rather than taken from the screen,
// so that what is pressed is the button that is drawn.
func folderButtonOn(t *testing.T, view *recipeview.View) *parts.Button {
	t.Helper()
	window := test.NewWindow(view.Content())
	t.Cleanup(window.Close)
	window.Resize(fyne.NewSize(1120, 780))
	for _, object := range test.LaidOutObjects(window.Canvas().Content()) {
		if button, ok := object.(*parts.Button); ok && button.Visible() &&
			button.Text == i18n.English.T(i18n.FolderOpen) {
			return button
		}
	}
	return nil
}

// The folder of a run opens from the card once the run has written, and once
// it has failed at the disk, which is where every reason it gives sends a
// person (DEC-336).
//
// The second run is the one the owner named: the files of the first are
// already there and replacing is off, so the run is refused for a file that
// exists, and the button stands under the sentence saying so.
func TestTheFolderOfARunOpensFromTheCardAfterItWroteAndAfterItFailed(t *testing.T) {
	dir := t.TempDir()
	manager := &shelf{}
	view := recipeview.New(i18n.English, real(t), recipeview.Writer(),
		background.Work{Read: background.ReadHere, Produce: produceHere},
		outward.NewFolders(i18n.English, manager, background.ReadHere, nil), nil)
	view.SetRecipe(screens.RelatedRecipe)
	view.SetOut(dir)

	// The canary: before any run there is no button, so a finder that answers
	// a button whatever is drawn cannot pass the checks below.
	if folderButtonOn(t, view) != nil {
		t.Fatal("the screen offers to open a folder before anything was written")
	}

	view.Start()
	if _, failed, _ := view.Written(); failed != nil {
		t.Fatalf("the first run failed: %v", failed)
	}
	pressedFolder(t, view, manager, dir)

	view.Start()
	_, failed, _ := view.Written()
	if failed == nil || !strings.Contains(failed.Error(), "already exists") {
		t.Fatalf("a second run into the same folder ended with %v, want a refusal for a file that exists", failed)
	}
	pressedFolder(t, view, manager, dir)
	if len(manager.shown) != 2 {
		t.Errorf("two presses showed %d folders", len(manager.shown))
	}
}

// pressedFolder presses the button on the screen and asks that the folder the
// run wrote into is the one the file manager was handed last.
func pressedFolder(t *testing.T, view *recipeview.View, manager *shelf, dir string) {
	t.Helper()
	button := folderButtonOn(t, view)
	if button == nil {
		t.Fatal("the card offers no way to open the folder of the run")
	}
	test.Tap(button)
	if len(manager.shown) == 0 || manager.shown[len(manager.shown)-1] != dir {
		t.Errorf("the file manager was handed %q, want %q last", manager.shown, dir)
	}
}

// A run somebody stopped wrote nothing, and the card says so without offering
// a folder to look in.
func TestARunSomebodyStoppedOffersNoFolder(t *testing.T) {
	stopped := func([]byte, produce.Settings, generate.Halt) (produce.Written, error) {
		return produce.Written{}, background.ErrStopped
	}
	view := recipeview.New(i18n.English, real(t), stopped,
		background.Work{Read: background.ReadHere, Produce: produceHere}, nowhere, nil)
	view.SetRecipe(screens.RelatedRecipe)
	view.SetOut(t.TempDir())
	view.Start()
	if _, failed, ran := view.Written(); !ran || !errors.Is(failed, background.ErrStopped) {
		t.Fatalf("the run did not end as stopped: ran %v, %v", ran, failed)
	}
	if folderButtonOn(t, view) != nil {
		t.Error("a run that wrote nothing offers a folder to look in")
	}
}
