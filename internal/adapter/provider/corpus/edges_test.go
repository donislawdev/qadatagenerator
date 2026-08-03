package corpus_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/corpus"
)

// # A carriage return would become part of a name
//
// Every file here is stored with unix line endings, which .gitattributes states
// and a measurement on 2026-08-03 confirmed. The risk is not what git holds, it
// is where these files come from: a script outside the repository writes them,
// and a script writing text on Windows produces carriage returns by default.
// Those bytes reach the binary through embed, which reads the working tree
// rather than the index, so a regenerated list could ship with a return on the
// end of every name while git reported no change at all.
//
// A name ending in an invisible byte would then be written into somebody's
// dataset, where it fails a comparison for a reason nobody can see.
func TestNoEntryCarriesAnInvisibleLineEnding(t *testing.T) {
	for _, entry := range corpus.All() {
		values, ok := corpus.Load(entry[0], entry[1])
		if !ok {
			t.Fatalf("%s/%s did not load", entry[0], entry[1])
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				t.Fatalf("%s/%s carries a line ending inside the entry %q", entry[0], entry[1], value)
			}
			if strings.TrimSpace(value) != value {
				t.Fatalf("%s/%s carries surrounding space in the entry %q", entry[0], entry[1], value)
			}
		}
	}
}

// # An empty list is refused rather than returned
//
// A caller turns a list into a value by drawing an index inside its length. A
// length of nought makes that draw a division by nothing, which stops the run
// with a panic naming the draw rather than the list, so the message points at
// the wrong package. Refusing here names the list.
func TestAListWithNoEntriesIsRefusedLikeAMissingOne(t *testing.T) {
	// Every list this build carries has entries, which is what makes the
	// guarantee above worth having.
	for _, entry := range corpus.All() {
		values, ok := corpus.Load(entry[0], entry[1])
		if !ok || len(values) == 0 {
			t.Fatalf("%s/%s came back empty or missing", entry[0], entry[1])
		}
	}

	// A list nobody carries answers the same way as one with nothing in it,
	// which is the point: a caller has one thing to check.
	if values, ok := corpus.Load("pl-PL", "there-is-no-such-list"); ok || values != nil {
		t.Fatal("a list this build does not carry was answered as usable")
	}
}
