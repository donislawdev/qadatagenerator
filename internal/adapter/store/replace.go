package store

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Replace puts one file on the disk whole, or leaves what was there.
//
// # Why this is not a Set of one file
//
// A Set writes the files of a run. It holds a name with a marker so that two
// runs cannot both write it, and its refusals speak of a dataset. A file a
// person saves from the window is neither: the person chose the name, either
// because it is the file they opened or because they agreed to replace it when
// they chose it, and the sentence that fails has to be about their file. What
// the two share is the part that matters, and it is the same here: the bytes go
// under a temporary name in the same directory and reach the real name by one
// rename, so a save that fails half way leaves the earlier file as it was.
//
// # Why the temporary file is flushed before the rename
//
// A rename can reach the disk before the bytes of the file it names do, and a
// machine losing power in between then leaves an empty file under the name of
// a recipe somebody kept for months. A dataset is written again by running the
// recipe, so a Set does not pay for this. A saved recipe is the one copy.
//
// # What is kept of the file being replaced
//
// Its mode, because a temporary file is created readable by its owner alone
// and a recipe shared with a build account would stop being readable to it.
// And its place: when the name is a link, the file the link leads to is the
// one replaced, so the link stays a link. A file marked read-only is refused
// rather than replaced, which is what an editor does with one.
//
// A file somebody else writes at the same moment is outside this, as it is
// outside a Set: the last rename wins, and nothing short of a lock every editor
// honours would change that.
func Replace(final string, data []byte) error {
	target := final
	if resolved, err := filepath.EvalSymlinks(final); err == nil {
		target = resolved
	}
	mode, err := keptMode(target)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".*.part")
	if err != nil {
		return fault.Wrap(fault.IO, err, "cannot write "+target+", and nothing has changed").
			DoInstead("check that the folder exists and that this account may write in it")
	}
	if err := filled(temporary, data, mode); err != nil {
		_ = os.Remove(temporary.Name())
		return fault.Wrap(fault.IO, err, "cannot write "+target+", and the earlier file is as it was").
			DoInstead("check that the disk has room and try again")
	}
	if err := os.Rename(temporary.Name(), target); err != nil {
		_ = os.Remove(temporary.Name())
		return fault.Wrap(fault.IO, err, "cannot put "+target+" in place, and the earlier file is as it was").
			DoInstead("close any program that holds the file open and try again")
	}
	return nil
}

// newFileMode is the mode a file nobody wrote before is given: readable by
// everybody and written by its owner, which is what an editor leaves.
const newFileMode fs.FileMode = 0o644

// keptMode is the mode the file being replaced has, or the mode of a new file
// when there is none, and refuses a name that cannot be replaced.
func keptMode(target string) (fs.FileMode, error) {
	held, err := os.Stat(target)
	switch {
	case err != nil:
		return newFileMode, nil
	case held.IsDir():
		return 0, fault.New(fault.IO, target+" is a folder, and nothing has been written").
			DoInstead("choose a file name rather than a folder")
	case held.Mode().Perm()&0o200 == 0:
		return 0, fault.New(fault.IO, target+" is read-only, and nothing has been written").
			DoInstead("save it under another name, or make the file writable first")
	}
	return held.Mode().Perm(), nil
}

// filled writes the bytes, flushes them to the disk, gives the file its mode
// and closes it. The close is asked about as well, because a file that failed
// to close may not hold what was written.
func filled(file *os.File, data []byte, mode fs.FileMode) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
