package takeview

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// shelf is a file manager that remembers which folders it was asked to show.
type shelf struct{ shown []string }

func (s *shelf) ShowFolder(path string) error {
	s.shown = append(s.shown, path)
	return nil
}

// folderButtonIn is the Open folder button in what the card shows about a
// run, or nothing when there is none.
func folderButtonIn(object fyne.CanvasObject) *parts.Button {
	for _, each := range test.LaidOutObjects(object) {
		if button, ok := each.(*parts.Button); ok && button.Visible() &&
			button.Text == i18n.English.T(i18n.FolderOpen) {
			return button
		}
	}
	return nil
}

// The folder of a run opens from the card once the run has written, and once
// it has failed at the disk, and a run somebody stopped offers none (DEC-336).
//
// The folder is named without its root, the way a person may type it, and the
// screen counts it from the folder of the file (take.Settings.Beside), so what
// the file manager must be handed is that folder, whole.
func TestTheFolderOfARunOpensFromTheCardAfterItWroteAndAfterItFailed(t *testing.T) {
	dir := t.TempDir()
	source := sourceFile(t, dir)
	manager := &shelf{}
	view, runs := screenWith(t, nil, outward.NewFolders(i18n.English, manager, background.ReadHere, nil))
	view.Open(source)
	if err := view.Unread(); err != nil {
		t.Fatalf("the screen could not read the file: %v", err)
	}
	view.delimiter.SetText(separator)
	view.ChooseLayers([]string{orderedLayers, otherLayer})
	view.out.SetText("written")
	want := view.asked().Out
	if !strings.HasPrefix(want, dir) {
		t.Fatalf("the folder of the run is %q, not beside the file in %q", want, dir)
	}

	// The canary: before any run there is no button, so a finder that answers
	// a button whatever is drawn cannot pass the checks below.
	if folderButtonIn(view.runBox) != nil {
		t.Fatal("the screen offers to open a folder before anything was written")
	}

	press := func(when string) {
		t.Helper()
		button := folderButtonIn(view.runBox)
		if button == nil {
			t.Fatalf("after a run that %s the card offers no way to open its folder", when)
		}
		before := len(manager.shown)
		test.Tap(button)
		if len(manager.shown) != before+1 || manager.shown[before] != want {
			t.Errorf("after a run that %s the file manager was handed %q, want %q", when, manager.shown, want)
		}
	}

	view.Start()
	runs.carryOut(t)
	if view.stage != stageWritten {
		t.Fatalf("the first run did not finish: %v", view.failed)
	}
	press("wrote")

	view.Start()
	runs.carryOut(t)
	if view.stage != stageFailed || !strings.Contains(view.failed.Error(), "already exists") {
		t.Fatalf("a second run into the same folder ended in stage %d with %v", view.stage, view.failed)
	}
	press("was refused for a file that exists")

	// Into a folder of its own, because a run into the first one is refused
	// for the files there before it reaches a record to stop between.
	view.out.SetText("stopped")
	view.Start()
	view.Stop()
	runs.carryOut(t)
	if view.stage != stageStopped {
		t.Fatalf("a run that was stopped ended in stage %d: %v", view.stage, view.failed)
	}
	if folderButtonIn(view.runBox) != nil {
		t.Error("a run that wrote nothing offers a folder to look in")
	}
}
