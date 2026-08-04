// Package store puts bytes on a disk, and refuses to do it in the two ways
// that would be quiet.
//
// # Atomic, because a half written dataset is worse than none
//
// Every file is written under a temporary name in the destination directory and
// renamed once the whole run has succeeded. A run stopped halfway leaves
// nothing that looks complete, which is the ban on silence applied to a file
// system. The temporary file shares the directory rather than living in the
// system temporary area, because a rename across volumes is a copy and a copy
// is not atomic.
//
// # A file that already exists stops the run
//
// Overwriting somebody's reference dataset without saying so is the same class
// of failure as a corruption that vanished from the manifest. This tool goes
// into other people's pipelines, where the destination is often a working
// directory with something else in it.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Set is the files of one run, none of which is visible under its real name
// until every one of them has been written.
type Set struct {
	dir       string
	overwrite bool
	pending   map[string]string
	open      []*os.File
}

// New returns a set of files to be written into a directory.
func New(dir string, overwrite bool) *Set {
	return &Set{dir: dir, overwrite: overwrite, pending: make(map[string]string)}
}

// Create opens a file under a temporary name and records where it belongs.
//
// The name is checked before anything is opened, so a name the file system
// would refuse is reported as a configuration failure rather than as a write
// error somebody has to interpret.
func (s *Set) Create(name string) (*os.File, error) {
	if err := usableName(name); err != nil {
		return nil, err
	}
	final := filepath.Join(s.dir, name)
	if err := s.refuseExisting(final); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fault.Wrap(fault.IO, err, "cannot make the output directory "+s.dir)
	}

	temporary, err := os.CreateTemp(s.dir, "."+name+".*.part")
	if err != nil {
		return nil, fault.Wrap(fault.IO, err, "cannot open a temporary file in "+s.dir)
	}
	s.pending[temporary.Name()] = final
	s.open = append(s.open, temporary)
	return temporary, nil
}

// refuseExisting stops the run rather than overwriting, unless told otherwise.
func (s *Set) refuseExisting(final string) error {
	if s.overwrite {
		return nil
	}
	if _, err := os.Stat(final); err == nil {
		return fault.New(fault.IO, final+" already exists. Delete it, choose another directory, "+
			"or pass the flag that allows overwriting. Nothing has been written")
	}
	return nil
}

// Commit closes every file and gives each its real name.
//
// It renames in a sorted order so that a run interrupted between two renames
// leaves the same partial state every time rather than one that depends on how
// Go happened to iterate a map.
func (s *Set) Commit() error {
	for _, file := range s.open {
		if err := file.Close(); err != nil {
			return fault.Wrap(fault.IO, err, "cannot close "+file.Name())
		}
	}
	s.open = nil

	for _, temporary := range sortedKeys(s.pending) {
		if err := os.Rename(temporary, s.pending[temporary]); err != nil {
			return fault.Wrap(fault.IO, err, "cannot put "+s.pending[temporary]+" in place")
		}
		delete(s.pending, temporary)
	}
	return nil
}

// Discard removes everything this run made. It is what a signal handler calls,
// and it reports nothing: the run is already ending for another reason and a
// second failure on the way out would hide the first.
func (s *Set) Discard() {
	for _, file := range s.open {
		_ = file.Close()
	}
	s.open = nil
	for temporary := range s.pending {
		_ = os.Remove(temporary)
		delete(s.pending, temporary)
	}
}

// forbidden are the characters no name may carry.
//
// The list is the union of what the supported systems refuse rather than what
// each one does, because a dataset that can be generated on one machine and not
// on another is a dataset whose recipe is not portable. Windows is the strict
// one here and so the union is its list.
const forbidden = `<>:"/\|?*`

// devices are the names Windows reserves whatever the extension.
//
// A file called CON is not a file, and a run that appeared to write one and then
// hung is the failure this refusal exists to prevent. Measured on this project
// already: reading such a name waits for the keyboard.
var devices = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// usableName refuses a name the file system would not take, or would take and
// turn into something else.
//
// It never repairs a name. Two entities differing only in a forbidden character
// would become one file, and the second dataset would overwrite the first
// without anybody being told, which is exactly the silence being avoided one
// layer up.
func usableName(name string) error {
	base := name
	if dot := strings.LastIndex(base, "."); dot > 0 {
		base = base[:dot]
	}
	switch {
	case name == "":
		return fault.New(fault.Config, "an entity with no name cannot become a file")
	case strings.ContainsAny(name, forbidden):
		return fault.New(fault.Config, fmt.Sprintf(
			"%q cannot be a file name because of the characters %s. Rename the entity", name, forbidden))
	case strings.ContainsRune(name, 0):
		return fault.New(fault.Config, "a name carrying a zero byte cannot become a file")
	case name != filepath.Base(name) || name == "." || name == "..":
		return fault.New(fault.Config, fmt.Sprintf(
			"%q is a path rather than a name, and a dataset never writes outside the directory "+
				"it was given. Rename the entity", name))
	case devices[strings.ToUpper(base)]:
		return fault.New(fault.Config, fmt.Sprintf(
			"%q is the name of a device rather than of a file. Rename the entity", name))
	case strings.HasSuffix(name, ".") || strings.HasSuffix(name, " "):
		return fault.New(fault.Config, fmt.Sprintf(
			"%q ends in a dot or a space, which some systems drop, so the file would end up under "+
				"a different name than the one asked for. Rename the entity", name))
	}
	return nil
}

func sortedKeys(pending map[string]string) []string {
	keys := make([]string, 0, len(pending))
	for key := range pending {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
