package values

import (
	"sync"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
)

// The entries of a word list that fit a limit on length, remembered for as long
// as one source lives.
//
// A file of its own because it is the one place in this package that shares
// something mutable, and the guard on concurrency names files rather than
// lines. Everything here is that and nothing else.

// entriesWithin is the entries of a word list within the declared maximum
// length, worked out the first time a run asks and remembered after that.
//
// # Why remembered, and why only for one source
//
// Until 2026-10-28 every value of a field with a limit measured the whole list
// again, which for the Polish surnames is about four thousand entries a value
// (found by an outside review on 2026-10-22). The draw and the domain of a
// unique column both read through this, so the two cannot be handed different
// lists.
//
// The key is the list, the limit and the unit. The locale is not in it because
// one source has one locale. Nothing is remembered beyond the source, which
// lives for one run, one preview or one call of the quick verb, so a limit the
// window was given once does not stay in memory after it.
func (s Source) entriesWithin(found listing, req generate.Request) []string {
	if !req.HasMaxLength || s.fitted == nil {
		return fitting(found.entries, req)
	}
	return s.fitted.of(fitKey{list: found.key, limit: req.MaxLength, unit: req.Unit},
		func() []string { return fitting(found.entries, req) })
}

// fittedLists remembers the entries of each list that fit each limit.
//
// A lock rather than a promise that a source is only ever used by one
// goroutine. Today it is, and a promise like that is broken by the first
// screen that shares one without anybody reading this.
type fittedLists struct {
	mu   sync.Mutex
	kept map[fitKey][]string
}

// fitKey names one list under one limit.
type fitKey struct {
	list  string
	limit int
	unit  string
}

// of answers the remembered entries for a key, working them out the first time.
func (f *fittedLists) of(key fitKey, work func() []string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if kept, known := f.kept[key]; known {
		return kept
	}
	if f.kept == nil {
		f.kept = map[fitKey][]string{}
	}
	kept := work()
	f.kept[key] = kept
	return kept
}
