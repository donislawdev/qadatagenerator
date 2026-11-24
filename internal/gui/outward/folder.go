package outward

import (
	"errors"
	"path/filepath"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// The folder a run wrote into, shown in the file manager of the system when a
// person presses Open folder (DEC-336).
//
// # Why the system call is not in this package
//
// A folder is opened differently on each system, and on Windows not the way
// the toolkit opens a page: a folder under a Polish or Japanese name did not
// open that way and nothing said so (measured 2026-11-24). The call for each system
// lives in the program that opens the window, under the guards that allow it
// (cmd/qadg-gui, folder_*.go), and this package is handed the result as a
// FileManager. Asking whether the folder is there belongs with that call too:
// no package of the window reads the disk itself.

// FileManager shows a folder in the file manager of the system.
//
// The path handed in is whole. A path that names no folder is answered with
// ErrNoFolder rather than handed on, so the window says in its own words that
// the folder is gone, instead of whatever each system answers about a path it
// cannot find.
type FileManager interface {
	ShowFolder(path string) error
}

// ErrNoFolder is the answer of a file manager asked to show a folder that is
// not there, or a path that is a file.
var ErrNoFolder = errors.New("there is no folder at this path")

// Folders is the Open folder button of the two screens that write, and what
// pressing it does.
//
// One value handed to both screens, built once where the window is assembled,
// so the button looks and answers the same on both and a third screen that
// writes gets it by asking for it rather than by copying it.
type Folders struct {
	text    i18n.Catalogue
	manager FileManager
	read    background.Read
	refused Refused
}

// NewFolders is the button maker for one window: its words, the file manager
// of its system, its place of work off the thread that draws, and the dialog
// that says why a folder did not open.
func NewFolders(text i18n.Catalogue, manager FileManager, read background.Read, refused Refused) Folders {
	return Folders{text: text, manager: manager, read: read, refused: refused}
}

// Button makes the button for one screen.
//
// where is asked at the press rather than at the making, because the button is
// made once with the screen and the folder is the one the last run wrote into.
func (f Folders) Button(where func() string) *parts.Button {
	button := parts.NewButton(f.text.T(i18n.FolderOpen), func() { f.open(where()) })
	button.ShowIcon(paint.IconFolder)
	return button
}

// open shows one folder, off the thread that draws.
//
// # Why off the thread that draws
//
// Asking whether a folder on a network share is there can take as long as the
// share takes to answer, and Windows may wait on the file manager it hands the
// folder to. Either on the thread that draws would stop the window answering.
//
// # Why the path is made whole here
//
// A run writes to the path in its field as it was typed, and a path without a
// root counts from where this program stands, so making it whole here names the
// same folder. The file manager is another program standing somewhere else, and
// the dialog shows the whole path because a part of one is no way to find a
// folder by hand.
func (f Folders) open(dir string) {
	whole, err := filepath.Abs(dir)
	if err != nil {
		whole = dir
	}
	tell := func(reason error) func() {
		message := f.text.T(i18n.FolderRefusedBody) + paragraph + whole +
			paragraph + f.text.T(i18n.FolderRefusedReason) + line + reason.Error()
		if errors.Is(reason, ErrNoFolder) {
			message = f.text.T(i18n.FolderRefusedMissing) + paragraph + whole
		}
		return func() { f.refused(f.text.T(i18n.FolderRefusedTitle), message) }
	}
	f.read(func() func() {
		if err := f.manager.ShowFolder(whole); err != nil {
			return tell(err)
		}
		return nil
	}, tell)
}
