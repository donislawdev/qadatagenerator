package corpus_test

import (
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
	"github.com/donislawdev/qadatagenerator/internal/license"
)

// The gate. A list added without its attribution fails the build, and an
// attribution left behind after a list was dropped fails it too. Taking
// somebody's collection is allowed on the condition that their notice travels
// with it, and a condition nobody checks is a condition nobody keeps.
func TestEveryListIsAttributedAndEveryAttributionHasAList(t *testing.T) {
	embedded := map[string]bool{}
	for _, entry := range corpus.All() {
		embedded[entry[0]+"/"+entry[1]] = true
	}
	described := map[string]bool{}
	for _, source := range license.Data {
		described[source.Corpus] = true
	}

	for name := range embedded {
		if !described[name] {
			t.Errorf("%s is built into the binary and named in no licence entry", name)
		}
	}
	for name := range described {
		if !embedded[name] {
			t.Errorf("%s is named in a licence entry and is not in the binary", name)
		}
	}
	if len(embedded) == 0 {
		t.Fatal("no lists were found at all, so this test is checking nothing")
	}
}

func TestEveryAttributionIsComplete(t *testing.T) {
	for _, source := range license.Data {
		if source.Project == "" || source.Version == "" || source.License == "" || source.Source == "" {
			t.Errorf("%s is described without one of project, version, licence or source", source.Corpus)
		}
	}
}

// The file has to say where it came from as well. The registry can be read
// from the binary and the file can be read on its own, and somebody looking at
// one should not have to find the other.
func TestEveryFileNamesItsSource(t *testing.T) {
	for _, entry := range corpus.All() {
		header, ok := corpus.Header(entry[0], entry[1])
		if !ok {
			t.Fatalf("%v has no header at all", entry)
		}
		joined := strings.Join(header, "\n")
		if !strings.Contains(joined, "Source:") {
			t.Errorf("%v does not name its source", entry)
		}
	}
}

// A value is chosen by turning a coordinate into an index, so the position of
// a name decides which record gets it. Reordering a list rewrites every
// dataset ever generated from it, which is why the order is asserted rather
// than restored at load.
func TestEveryListIsSortedByCodePointAndHasNoRepeats(t *testing.T) {
	for _, entry := range corpus.All() {
		name := entry[0] + "/" + entry[1]
		entries, ok := corpus.Load(entry[0], entry[1])
		if !ok {
			t.Fatalf("%s could not be read", name)
		}

		if !sort.SliceIsSorted(entries, func(i, j int) bool { return entries[i] < entries[j] }) {
			t.Errorf("%s is not sorted by code point", name)
		}
		seen := make(map[string]bool, len(entries))
		for _, value := range entries {
			if seen[value] {
				t.Errorf("%s repeats %q", name, value)
			}
			seen[value] = true
			if value != strings.TrimSpace(value) {
				t.Errorf("%s has %q with space around it", name, value)
			}
			if value == "" {
				t.Errorf("%s has an empty entry", name)
			}
		}
	}
}

// An extraction that quietly produced one line would pass every test above.
// This is the one that notices.
func TestEveryListIsBigEnoughToBeWorthHaving(t *testing.T) {
	const fewest = 15
	for _, entry := range corpus.All() {
		entries, _ := corpus.Load(entry[0], entry[1])
		if len(entries) < fewest {
			t.Errorf("%s/%s holds %d entries, which is too few to have come out right",
				entry[0], entry[1], len(entries))
		}
	}
}

// The Polish lists have to be Polish. An extraction that lost the encoding
// somewhere would produce readable ASCII and nothing else would notice.
func TestThePolishListsCarryPolishLetters(t *testing.T) {
	for _, name := range []string{"cities", "first-names-female", "last-names-unisex"} {
		entries, ok := corpus.Load("pl-PL", name)
		if !ok {
			t.Fatalf("pl-PL/%s is missing", name)
		}
		withDiacritics := 0
		for _, value := range entries {
			for _, letter := range value {
				if letter > unicode.MaxASCII {
					withDiacritics++
					break
				}
			}
		}
		if withDiacritics == 0 {
			t.Errorf("pl-PL/%s has no letter outside ASCII, so the encoding was lost", name)
		}
	}
}

// A missing list is a defect in the caller. Answering with an empty list would
// turn it into data nobody ordered.
func TestAListThatIsNotThereSaysSoRatherThanAnsweringEmpty(t *testing.T) {
	if _, ok := corpus.Load("pl-PL", "there-is-no-such-list"); ok {
		t.Error("a list that does not exist was reported as found")
	}
	if _, ok := corpus.Load("xx-XX", "cities"); ok {
		t.Error("a locale that does not exist was reported as found")
	}
	if entries, ok := corpus.Load("pl-PL", "cities"); !ok || len(entries) == 0 {
		t.Error("a list that does exist was not found")
	}
}

// The header is description, not data. A name that starts with a hash would
// be dropped, and nothing in these lists does.
func TestTheHeaderDoesNotLeakIntoTheEntries(t *testing.T) {
	for _, entry := range corpus.All() {
		entries, _ := corpus.Load(entry[0], entry[1])
		for _, value := range entries {
			if strings.HasPrefix(value, "#") {
				t.Errorf("%v: a description line was read as an entry: %q", entry, value)
			}
		}
	}
}
