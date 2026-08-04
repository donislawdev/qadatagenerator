// Package corpus carries the reference word lists compiled into this binary.
//
// The lists are data rather than a dependency. Nothing here links anybody
// else's code, and the contents and the order belong to this project from the
// moment they were written down, so an upstream release cannot move anybody's
// generated data.
//
// # Where they came from, and why that is stated rather than assumed
//
// Most were read out of open source projects for fake data and written here
// under their terms. Reading somebody's list and keeping it is taking a
// substantial part of their collection, whether it arrived through a function
// or through a file, so the licences require their notice to travel with it.
// Each file names its source in a header, the registry beside the licences
// repeats it, and --license prints it from the binary, because that is where
// the data actually ships.
//
// One of them came from somewhere else and the difference is worth knowing.
// The postal registry was built from two files of the United States census,
// which carry no copyright at all, so the notice it ships is a courtesy rather
// than a condition. It is shipped anyway, because a reader cannot tell the two
// cases apart by looking, and a registry that lists only the collections it was
// forced to list teaches the next reader the wrong rule.
//
// # Not every list is a list of words
//
// A postal registry is a table, and its lines carry fields separated by a pipe
// rather than one value each. Nothing here parses them: a caller that needs the
// fields splits them itself, because a loader that knew about columns would have
// to know which file has which, and that is the sort of knowledge that goes out
// of date silently.
//
// # Order is part of the contract
//
// A value is chosen by turning a coordinate into an index, so the position of
// a name in a list decides which record gets it. Sorting is by code point and
// is done once, when the list is written, never at load and never by the
// collation of whichever machine happens to run this. A test asserts it,
// because reordering a list quietly rewrites every dataset ever generated
// from it.
package corpus

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed en-US pl-PL
var files embed.FS

// Comment marks a line that describes the file rather than belonging to it.
const comment = "#"

// Load returns one list, in the order it is stored.
//
// The second result says whether a usable list came back. There is no fallback
// to another locale and no empty list standing in for a missing one: a provider
// asking for a list this build does not carry is a defect, and answering with
// silence would turn it into data nobody ordered.
//
// A file that exists and carries no entries answers false as well. The caller
// turns a list into a value by drawing an index inside its length, and a length
// of nought makes that draw a division by nothing, which stops the run with a
// panic naming the draw rather than the list. Refusing here names the list.
func Load(locale, name string) ([]string, bool) {
	raw, err := files.ReadFile(path.Join(locale, name+".txt"))
	if err != nil {
		return nil, false
	}
	entries := parse(string(raw))
	if len(entries) == 0 {
		return nil, false
	}
	return entries, true
}

// parse reads the entries out of a list file.
//
// The carriage return is trimmed although every file in this repository is
// stored with unix line endings, checked in .gitattributes and measured on
// 2026-08-03. The reason is where these files come from: they are written by a
// script outside the repository, and a script writing text on Windows produces
// carriage returns by default. Those bytes reach the binary through embed,
// which reads the working tree rather than what git holds, so a regenerated
// list could ship with a return on the end of every name while git showed no
// change at all. One trim closes that, and the name Anna never becomes Anna
// followed by an invisible byte in somebody's dataset.
func parse(content string) []string {
	lines := strings.Split(content, "\n")
	entries := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, comment) {
			continue
		}
		entries = append(entries, line)
	}
	return entries
}

// Header returns the descriptive lines of a list, which name where it came
// from. It exists so that a test can check every file says so.
func Header(locale, name string) ([]string, bool) {
	raw, err := files.ReadFile(path.Join(locale, name+".txt"))
	if err != nil {
		return nil, false
	}
	var header []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, comment) {
			header = append(header, line)
		}
	}
	return header, true
}

// All returns every list carried by this build, as locale and name pairs,
// sorted so that two runs walk them the same way.
func All() [][2]string {
	var found [][2]string
	_ = fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(name, ".txt") {
			return err
		}
		locale, file := path.Split(name)
		found = append(found, [2]string{
			strings.TrimSuffix(locale, "/"),
			strings.TrimSuffix(file, ".txt"),
		})
		return nil
	})
	sort.Slice(found, func(i, j int) bool {
		if found[i][0] != found[j][0] {
			return found[i][0] < found[j][0]
		}
		return found[i][1] < found[j][1]
	})
	return found
}
