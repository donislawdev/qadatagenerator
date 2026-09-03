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
	"sync"
)

//go:embed en-US pl-PL
var files embed.FS

// Comment marks a line that describes the file rather than belonging to it.
const comment = "#"

// held is every list this build has already read, keyed by locale and name.
//
// # Why this is here at all
//
// 🔴 A list becomes a value by drawing an index inside its length, so a run
// asks for the whole list once per cell. Reading and splitting the file every
// time made a surname cost 3695 string allocations, and a dataset of 100000
// rows with three word lists took thirty times as long as the same dataset
// without them, measured on 2026-08-26 and written up in the backlog. Nothing
// in the result changes: the same bytes come back in the same order, which is
// the whole reason this is safe to do.
//
// # What every caller must therefore not do
//
// 🔴 The slice handed back is shared by every later caller. Nobody may write
// through it, and nobody may append to it without taking a copy first. The one
// caller that filters a list does so with a zero capacity slice of it, which
// forces append to allocate rather than to write into the shared array. A
// change from that to a plain reslice would corrupt every row drawn afterwards,
// in a way no compiler and no type notices, so a test stands in front of it.
//
// A sync.Map rather than a mutex because the reads outnumber the writes by the
// number of rows in the dataset, and there are as many writes as this build has
// files.
var held sync.Map

// joined is every pair of lists that has already been put end to end, keyed by
// locale and both names. It is separate from held because the join is a
// different list with a different length, and rebuilding it per value costs two
// copies of both lists.
var joined sync.Map

// Locales are the locales this build carries word lists for, sorted.
//
// # Why this is read from the files rather than written down
//
// A list here would be a second copy of what the embed directive above already
// says, and the day a third locale arrived the two would part company without a
// word. The answer instead comes from the same bytes the lists come from.
//
// 🔴 It is not the set of locales a recipe may name, and whoever prints it has
// to say so. A generator that draws on no list at all, such as an identifier
// built by arithmetic, is produced in any locale, so a recipe naming one this
// build has no words for is refused per field rather than outright.
func Locales() []string {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		// Unreachable: the directory is embedded in the binary, so reading its
		// root cannot fail for a reason a caller could act on. Answering with
		// nothing beats a panic in a function whose only job is to describe
		// this build to a reader.
		return nil
	}
	var locales []string
	for _, entry := range entries {
		if entry.IsDir() {
			locales = append(locales, entry.Name())
		}
	}
	sort.Strings(locales)
	return locales
}

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
	key := locale + "/" + name
	if kept, found := held.Load(key); found {
		entries, _ := kept.([]string)
		return entries, len(entries) > 0
	}
	raw, err := files.ReadFile(path.Join(locale, name+".txt"))
	if err != nil {
		held.Store(key, []string(nil))
		return nil, false
	}
	entries := parse(string(raw))
	held.Store(key, entries)
	if len(entries) == 0 {
		return nil, false
	}
	return entries, true
}

// Joined returns two lists end to end, in that order.
//
// A generator drawing from two lists as if they were one asks for the join once
// per value, and building it there copies both lists every time. The order is
// the order of the arguments, because it decides which index reaches which
// entry and therefore what every dataset drawn this way holds.
//
// It answers false when either list is missing, and the caller names which by
// asking Load. The same rule applies to what comes back as to Load: it is
// shared, and nobody may write through it.
func Joined(locale, first, second string) ([]string, bool) {
	key := locale + "/" + first + "+" + second
	if kept, found := joined.Load(key); found {
		entries, _ := kept.([]string)
		return entries, len(entries) > 0
	}
	left, foundLeft := Load(locale, first)
	right, foundRight := Load(locale, second)
	if !foundLeft || !foundRight {
		joined.Store(key, []string(nil))
		return nil, false
	}
	entries := make([]string, 0, len(left)+len(right))
	entries = append(entries, left...)
	entries = append(entries, right...)
	joined.Store(key, entries)
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
