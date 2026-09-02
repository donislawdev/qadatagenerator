package preview

import (
	"sort"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
)

// held is a set of files that never reaches a disk.
//
// # Why a preview writes at all, rather than being computed
//
// It is the promise the window makes and the one this project keeps in writing:
// the answer on screen before anybody presses a button leaves no trace. Nothing
// is created, nothing is replaced, and a person reading a recipe on a screen
// cannot destroy a file by reading it. Writing into memory is how the real
// writer stays the real writer while that promise holds.
//
// # Why the size is not capped here
//
// What bounds this is the plan above it, cut to a few rows a table before the
// engine ever runs. A cap here would be a second limit in a second unit, and
// the two would disagree the day either moved.
type held struct {
	written map[string]*strings.Builder
}

func newHeld() *held { return &held{written: make(map[string]*strings.Builder)} }

// Create takes a file by name and hands back somewhere to put it.
func (h *held) Create(name string) (format.WriteCloser, error) {
	built := &strings.Builder{}
	h.written[name] = built
	return kept{built}, nil
}

// files returns what was written, sorted by name.
//
// Sorted because a map in Go is walked in a deliberately unstable order and
// this answer travels to a screen. Two previews of one recipe listing its files
// in two orders would be the determinism rule broken in the one place nobody
// would think to test.
func (h *held) files() []File {
	names := make([]string, 0, len(h.written))
	for name := range h.written {
		names = append(names, name)
	}
	sort.Strings(names)

	files := make([]File, 0, len(names))
	for _, name := range names {
		files = append(files, File{Name: name, Text: h.written[name].String()})
	}
	return files
}

// kept is a builder that can be closed, because a writer closes what it opens
// and a builder in memory has nothing to close.
type kept struct{ *strings.Builder }

func (kept) Close() error { return nil }
