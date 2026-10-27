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
// # A file that already exists stops the run, including one another run is
// writing right now
//
// Overwriting somebody's reference dataset without saying so is the same class
// of failure as a corruption that vanished from the manifest. This tool goes
// into other people's pipelines, where the destination is often a working
// directory with something else in it.
//
// That promise used to hold only for a file already sitting on the disk. A
// destination name is now taken at the moment a file is created rather than
// asked about, so two runs started against one directory no longer both find
// the name free, work for minutes and let the second silently replace the
// first. What that costs, and why the obvious alternative is not available
// here, is written at claim below.
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
	destined  map[string]bool
	reserved  map[string]bool
	open      []*File
}

// File is one file of a set, under its temporary name until the set commits.
//
// # Why closing it is allowed before the set commits, and why twice is fine
//
// A writer closes the file of a table when the table ends, so that a recipe of
// many tables holds one handle at a time rather than one per table until the
// end of the run (found by an outside review on 2026-10-22). The set closes
// every file again when it commits or throws them away, because a writer that
// forgot would otherwise leave a handle behind. So a close happens once and is
// remembered, and every later one answers with what the first one said: a file
// that failed to close is a file that may not hold what was written, and the
// commit must still refuse it even when the writer that closed it said nothing.
type File struct {
	held   *os.File
	closed bool
	failed error
}

// Write puts bytes in the file.
func (f *File) Write(p []byte) (int, error) { return f.held.Write(p) }

// Close closes the file the first time and repeats that answer afterwards.
func (f *File) Close() error {
	if !f.closed {
		f.closed = true
		f.failed = f.held.Close()
	}
	return f.failed
}

// Name is the temporary name the file is written under.
func (f *File) Name() string { return f.held.Name() }

// New returns a set of files to be written into a directory.
func New(dir string, overwrite bool) *Set {
	return &Set{
		dir:       dir,
		overwrite: overwrite,
		pending:   make(map[string]string),
		destined:  make(map[string]bool),
		reserved:  make(map[string]bool),
	}
}

// Create opens a file under a temporary name and records where it belongs.
//
// The name is checked before anything is opened, so a name the file system
// would refuse is reported as a configuration failure rather than as a write
// error somebody has to interpret.
func (s *Set) Create(name string) (*File, error) {
	if err := usableName(name); err != nil {
		return nil, err
	}
	final := filepath.Join(s.dir, name)
	if err := s.ownName(name, final); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return nil, fault.Wrap(fault.IO, err, "cannot make the output directory "+s.dir)
	}
	if err := s.claim(name, final); err != nil {
		return nil, err
	}

	temporary, err := os.CreateTemp(s.dir, "."+name+".*.part")
	if err != nil {
		// The claim has to go back, or a run that failed to open its own
		// temporary file would leave a marker blocking every later run.
		s.release(s.marker(name))
		return nil, fault.Wrap(fault.IO, err, "cannot open a temporary file in "+s.dir)
	}
	s.destined[final] = true
	s.pending[temporary.Name()] = final
	file := &File{held: temporary}
	s.open = append(s.open, file)
	return file, nil
}

// ownName refuses a destination this run has already taken for something else.
//
// # Why this is separate from claim, which answers the same question
//
// 🔴 claim answers it about somebody else and takes no marker when
// overwriting is allowed, because overwriting says a file already on the disk
// may be replaced. It never said a run may replace its own work, and until
// 2026-08-26 it did: two things written under one name both reached Commit,
// which renamed one over the other and reported success. A recipe with an
// entity called run-note wrote two records, said so on the log, and left a
// directory holding only the note. Nothing failed, nothing was missing from any
// count, and the rows were gone.
//
// # Why the message tells the reader to rename the entity
//
// Of the two things colliding, only one is theirs. The names this tool reserves
// for the record of the run and for the manifest are fixed by the tool, so the
// only end of the collision anybody can move is the entity.
func (s *Set) ownName(name, final string) error {
	if !s.destined[final] {
		return nil
	}
	return fault.New(fault.Config, fmt.Sprintf(
		"this run writes %s twice, because an entity is named after a file the run itself "+
			"leaves beside the data. One of the two would replace the other and nothing "+
			"would say so", name)).
		DoInstead("rename the entity")
}

// marker is the hidden file that says a run holds this name.
func (s *Set) marker(name string) string {
	return filepath.Join(s.dir, "."+name+".claim")
}

// claim answers both ways a destination can be taken: a file already sitting
// there, and another run on its way to putting one there.
//
// # Why a marker beside the name rather than the name itself
//
// Asking whether a file exists and renaming onto it are two separate moments,
// and between them lies the whole run. Two runs against one directory both saw
// a free name, both worked for minutes, and the second rename replaced the
// first result without a word, because replacing on rename is what every
// supported system does.
//
// The obvious fix is to create the destination name itself with O_EXCL, so that
// the check and the taking are one operation. It was written that way first and
// an existing test caught it within a minute: it makes the destination name
// appear, empty, before the run has succeeded. That breaks a promise worth more
// than the one being fixed, because an empty file under the name somebody
// expects results at reads as an answer, while a leftover temporary reads as
// rubbish.
//
// So the atomic creation happens on a hidden marker beside the name. The
// destination stays absent until Commit, and a second run finds the marker and
// stops before doing any work.
//
// # What this does and does not cover, stated rather than implied
//
// It covers two runs of this tool, which is what the backlog entry was about.
// It does not cover somebody else creating the destination during our run, and
// it is not meant to: whoever can write to the output directory is outside the
// threat model in docs/15, and a rename that refuses to replace would need
// package syscall, which the architecture guard refuses outside one named file.
//
// A marker left behind by a run that was killed outright will stop the next
// run. That is deliberate and the message says which file to delete, because
// the alternative is a lock that expires and stops meaning anything.
func (s *Set) claim(name, final string) error {
	if s.overwrite {
		return replaceable(final)
	}

	marker := s.marker(name)
	holder, err := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return fault.New(fault.IO, "another run is writing "+final+" right now, or one was "+
				"stopped before it could tidy up, and nothing has been written").
				DoInstead("wait for that run to finish, or delete " + marker +
					" if nothing is running")
		}
		return fault.Wrap(fault.IO, err, "cannot take the name "+final)
	}
	if err := holder.Close(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot close "+marker)
	}
	s.reserved[marker] = true
	// 🔴 Asked only once the marker is ours, never before. Until 2026-10-26 it
	// was asked first, and that answer is older than the marker: a run that
	// found the name free, and then lost the moment before taking the marker to
	// another run that finished and tidied its own marker away, held a marker
	// for a name that was no longer free and replaced that run's result at
	// Commit. Reported by an outside review on 2026-10-22 from the order of the
	// calls, not from a race anybody saw. Asked after, the answer cannot be
	// overtaken by a run of this tool, because any other run has to get past
	// the same marker first.
	if err := absent(final); err != nil {
		s.release(marker)
		return err
	}
	return nil
}

// absent refuses a destination that already holds something.
func absent(final string) error {
	if _, err := os.Lstat(final); err == nil {
		return fault.New(fault.IO,
			final+" already exists, and nothing has been written").
			DoInstead("delete it, choose another directory, or allow replacing")
	}
	return nil
}

// replaceable refuses, before any work, a destination that replacing cannot
// reach.
//
// A directory under the name a file is meant to take cannot be renamed over,
// and until 2026-10-26 that was found at Commit, after the whole dataset had
// been written. It is a fact about the disk known at the start, so the start is
// where it is said.
func replaceable(final string) error {
	held, err := os.Lstat(final)
	if err != nil || !held.IsDir() {
		return nil
	}
	return fault.New(fault.IO,
		final+" is a directory, which a file of the dataset cannot replace, and nothing has been written").
		DoInstead("move the directory away, or choose another directory to write to")
}

// release gives a claimed name back. It reports nothing, because it runs on a
// path that is already failing for another reason.
func (s *Set) release(marker string) {
	if s.reserved[marker] {
		_ = os.Remove(marker)
		delete(s.reserved, marker)
	}
}

// Commit closes every file and gives each its real name.
//
// A file its writer closed already answers with what that close said, so a
// failure there stops the commit as a failure here would.
//
// It renames in a sorted order so that a run interrupted between two renames
// leaves the same partial state every time rather than one that depends on how
// Go happened to iterate a map.
//
// # Why a rename that fails takes the earlier ones back
//
// 🔴 Renaming several files is not one operation, and no system offers one. A
// failure part way through used to return the error and stop, leaving the files
// already renamed under their real names: half a dataset, no record of the run,
// no manifest, and the header of this package promising the opposite. The
// triggers are ordinary rather than exotic, and on Windows they are daily: a
// file held open by another program, a full disk, a permission.
//
// So the failure is undone. Each rename that succeeded is put back under the
// temporary name it came from, which returns the directory to what it held
// before, and Discard then removes those temporary files as it always did. What
// this does not buy is atomicity: a machine losing power between two renames
// still leaves the halfway state, because nothing short of writing the whole set
// into one directory and renaming that directory would fix it. That is stated
// rather than implied, and the alternative is in the backlog.
//
// # Why a file being replaced is moved aside first rather than renamed over
//
// 🔴 Putting a run back only returns the directory to what it held before when
// nothing was there before. With replacing allowed, the first rename used to
// land on the earlier file and destroy it, and putting the new file back then
// left NOTHING under that name: a failure on the second file of a set deleted
// the first file of the previous result. Found by an outside review on
// 2026-10-22, reproduced there with a directory in the way of the second name.
//
// So the earlier file is renamed to a hidden name beside it before the new one
// takes its place, put back if the run is undone, and removed once every file
// is in place. A machine losing power between those two renames leaves the
// earlier file under the hidden name rather than nowhere, which is the halfway
// state worth having: rubbish beside a result can be tidied, a result that is
// gone cannot be had back.
func (s *Set) Commit() error {
	for _, file := range s.open {
		if err := file.Close(); err != nil {
			return fault.Wrap(fault.IO, err, "cannot close "+file.Name())
		}
	}
	s.open = nil

	moved := make([]placed, 0, len(s.pending))
	for _, temporary := range sortedKeys(s.pending) {
		final := s.pending[temporary]
		earlier, err := s.setAside(final)
		if err != nil {
			return fault.Wrap(fault.IO, err, "cannot move the earlier "+final+" aside"+
				s.putBack(moved))
		}
		if err := os.Rename(temporary, final); err != nil {
			undone := append(moved, placed{temporary: "", earlier: earlier, final: final})
			return fault.Wrap(fault.IO, err, "cannot put "+final+" in place"+s.putBack(undone))
		}
		moved = append(moved, placed{temporary: temporary, earlier: earlier, final: final})
	}

	for _, done := range moved {
		delete(s.pending, done.temporary)
		// The earlier file has been replaced for good, which is what replacing
		// was allowed to do. Failing to remove the copy kept on the way is not
		// a failure of the run: every file is in place, and the copy is a
		// hidden file beside them rather than a missing one.
		if done.earlier != "" {
			_ = os.Remove(done.earlier)
		}
		// Once the real file is in place it is the thing that stops the next
		// run, so the marker has done its job and goes. Leaving it would block
		// every later run against a directory that now legitimately holds a
		// result.
		s.release(s.marker(filepath.Base(done.final)))
	}
	return nil
}

// placed is one file Commit has put under its real name, together with where
// the file it replaced is waiting.
type placed struct {
	// temporary is where the new file came from, and empty for the file whose
	// own rename failed, which has nothing to take back.
	temporary string
	// earlier is the hidden name of the file this one replaced, and empty when
	// the name was free.
	earlier string
	final   string
}

// setAside moves a file already sitting under the name aside, and says where
// it went. A name holding nothing gives an empty answer and no error.
//
// The hidden name is created rather than made up, so that it cannot be a file
// somebody else already has, and the rename then replaces that empty file,
// which every supported system does on rename.
//
// A directory is left where it is. Create refuses one that is there at the
// start, so a directory here arrived during the run, and moving somebody's
// directory to a hidden name is not what replacing a file was allowed to do. The
// rename onto it fails and the run is undone, as it was before this existed.
func (s *Set) setAside(final string) (string, error) {
	held, err := os.Lstat(final)
	if err != nil || held.IsDir() {
		return "", nil
	}
	holder, err := os.CreateTemp(s.dir, "."+filepath.Base(final)+".*.old")
	if err != nil {
		return "", err
	}
	earlier := holder.Name()
	if err := holder.Close(); err != nil {
		_ = os.Remove(earlier)
		return "", err
	}
	if err := os.Rename(final, earlier); err != nil {
		_ = os.Remove(earlier)
		return "", err
	}
	return earlier, nil
}

// putBack undoes the renames that succeeded, and says what it could not undo.
//
// It reports by returning text rather than an error, because it runs on a path
// that is already failing and the reader needs one sentence naming both the
// failure and what is still on the disk because of it. Silence here would be the
// worst of the three: a directory holding files nobody was told about.
func (s *Set) putBack(moved []placed) string {
	var stuck, kept []string
	for index := len(moved) - 1; index >= 0; index-- {
		done := moved[index]
		if done.temporary != "" {
			if err := os.Rename(done.final, done.temporary); err != nil {
				stuck = append(stuck, done.final)
				if done.earlier != "" {
					kept = append(kept, done.earlier)
				}
				continue
			}
		}
		if done.earlier == "" {
			continue
		}
		if err := os.Rename(done.earlier, done.final); err != nil {
			kept = append(kept, done.earlier)
		}
	}
	said := ""
	if len(stuck) > 0 {
		sort.Strings(stuck)
		said += ". These were written and could not be taken back: " + strings.Join(stuck, ", ")
	}
	if len(kept) > 0 {
		sort.Strings(kept)
		said += ". The files they were to replace are kept under these names: " +
			strings.Join(kept, ", ")
	}
	return said
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
	// The markers go too. A marker left behind stops every later run against
	// this directory, and a run that has just been told to throw its work away
	// is exactly the one that must not leave that behind.
	for marker := range s.reserved {
		_ = os.Remove(marker)
		delete(s.reserved, marker)
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
	// 🔴 The FIRST dot, not the last, and the difference is the whole of this
	// line. Windows reserves a device name by the part before the first dot, so
	// CON.corrupted.csv is the console just as CON.csv is. Cutting at the last
	// dot refused the one and let the other through, measured on 2026-08-11.
	//
	// It went unnoticed because it was unreachable: a name from a recipe entity
	// carries one dot. The verb that damages somebody else's file always
	// produces two, so that verb made an old hole reachable without touching
	// this file.
	base := name
	if dot := strings.Index(base, "."); dot > 0 {
		base = base[:dot]
	}
	switch {
	case name == "":
		return fault.New(fault.Config, "an entity with no name cannot become a file")
	case strings.ContainsAny(name, forbidden):
		return fault.New(fault.Config, fmt.Sprintf(
			"%q cannot be a file name because of the characters %s", name, forbidden)).
			DoInstead("rename the entity")
	case strings.ContainsRune(name, 0):
		return fault.New(fault.Config, "a name carrying a zero byte cannot become a file")
	case name != filepath.Base(name) || name == "." || name == "..":
		return fault.New(fault.Config, fmt.Sprintf(
			"%q is a path rather than a name, and a dataset never writes outside the directory "+
				"it was given", name)).
			DoInstead("rename the entity")
	case devices[strings.ToUpper(base)]:
		return fault.New(fault.Config, fmt.Sprintf(
			"%q is the name of a device rather than of a file", name)).
			DoInstead("rename the entity")
	case strings.HasSuffix(name, ".") || strings.HasSuffix(name, " "):
		return fault.New(fault.Config, fmt.Sprintf(
			"%q ends in a dot or a space, which some systems drop, so the file would end up under "+
				"a different name than the one asked for", name)).
			DoInstead("rename the entity")
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
