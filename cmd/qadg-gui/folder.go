package main

import (
	"errors"
	"io/fs"
	"os"

	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
)

// fileManager shows the folder a run wrote into, for the Open folder button of
// both screens that write (DEC-336).
//
// Whether the folder is there is asked here, for every system alike, and how it
// is shown is asked of the system: folder_windows.go on Windows and
// folder_other.go elsewhere, each in its own file because each is an exception
// to a different guard.
type fileManager struct {
	// show hands a whole path that names a folder to the system.
	show func(path string) error
}

// newFileManager is the file manager of the system this window runs on.
func newFileManager(desktop outward.Desktop) fileManager {
	return fileManager{show: systemShowsFolder(desktop)}
}

// ShowFolder shows one folder, or says that there is none to show.
//
// A path that is not there, or that is a file, is answered with
// outward.ErrNoFolder and never handed to the system. Any other failure to ask,
// a permission on the way to it among them, is handed back as it is, because
// the folder may well be there and the reason is what a person can act on.
func (m fileManager) ShowFolder(path string) error {
	held, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return outward.ErrNoFolder
	case err != nil:
		return err
	case !held.IsDir():
		return outward.ErrNoFolder
	}
	return m.show(path)
}
