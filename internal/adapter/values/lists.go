package values

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// How a generator drawing from a word list reaches its entries.
//
// Split out of the registry on 2026-10-26, when the domain of a unique column
// began reading the same entries the draw reads and the registry stood one line
// under the ceiling on the length of a file.

// listing is what a word list hands a draw: the entries, the name a message
// gives them, and the name of the domain they make.
//
// The last two differ for a draw from two lists at once. A message names the
// first list, which is the sentence the plan gives for the same list, while
// the domain has to be told apart from the domain of that first list alone,
// because a first name drawn without a gender and one drawn for a woman stand
// in the same column and draw from different sets.
type listing struct {
	entries []string
	list    string
	key     string
}

// listed names the entries a request draws from.
type listed func(Source, generate.Request) (listing, error)

// fromEntries draws one entry out of whatever a word list names, honouring a
// maximum length by choosing among the entries that fit.
//
// Choosing rather than truncating is the whole point. A truncated name is a
// value this tool invented, and inventing values is what the second pillar does
// on purpose and the first must never do by accident.
func fromEntries(entries listed) producer {
	return func(s Source, req generate.Request) (string, error) {
		found, err := entries(s, req)
		if err != nil {
			return "", err
		}
		return s.pick(found, req)
	}
}

// entriesOf names the entries of one list.
func (s Source) entriesOf(list string) (listing, error) {
	entries, found := corpus.Load(s.locale, list)
	if !found {
		return listing{}, fault.New(fault.Config, noList(list, s.locale))
	}
	return listing{entries: entries, list: list, key: list}, nil
}

// entriesEither names the entries of two lists as if they were one, without
// concatenating them on every call.
func (s Source) entriesEither(first, second string) (listing, error) {
	entries, found := corpus.Joined(s.locale, first, second)
	if !found {
		// Named one at a time rather than as a pair, so that the sentence is
		// the one the plan gives for the same list and the reader is told which
		// of the two is missing rather than that one of them is.
		if _, carried := corpus.Load(s.locale, first); !carried {
			return listing{}, fault.New(fault.Config, noList(first, s.locale))
		}
		return listing{}, fault.New(fault.Config, noList(second, s.locale))
	}
	return listing{entries: entries, list: first, key: first + "+" + second}, nil
}

// entriesDerived names the entries of a list built from others.
func (s Source) entriesDerived(list string, build func(string) ([]string, bool)) (listing, error) {
	entries, found := corpus.Derived(s.locale, list, build)
	if !found {
		// Named in the words the plan uses, so the run and the plan say the
		// same sentence about the same missing list.
		// A build that found both lists and kept nothing of them has no such
		// sentence, so it is named as the list that came out empty.
		if fine, why := surnameLists(s.locale); !fine {
			return listing{}, fault.New(fault.Config, why)
		}
		return listing{}, fault.New(fault.Config, noList(list, s.locale))
	}
	return listing{entries: entries, list: list, key: list}, nil
}

// pick chooses one entry, from those that fit when a limit is declared.
func (s Source) pick(found listing, req generate.Request) (string, error) {
	entries := fitting(found.entries, req)
	if len(entries) == 0 {
		return "", fault.New(fault.Data, fmt.Sprintf(
			"no entry of %s for %s is short enough for %s, which accepts %d %s",
			found.list, s.locale, req.At.Field, req.MaxLength, req.Unit))
	}
	return entries[s.seed.IntN(req.At, int64(len(entries)))], nil
}

// fitting returns the entries within the declared maximum length, all of them
// when none is declared.
func fitting(entries []string, req generate.Request) []string {
	if !req.HasMaxLength {
		return entries
	}
	kept := entries[:0:0]
	for _, entry := range entries {
		if measure(entry, req.Unit) <= req.MaxLength {
			kept = append(kept, entry)
		}
	}
	return kept
}
