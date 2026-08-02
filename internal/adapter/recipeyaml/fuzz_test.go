package recipeyaml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
)

// A recipe arrives from somebody else's repository. It is untrusted input in
// the same sense a downloaded file is, so nothing in it may take the process
// down, and the reader has to answer rather than fail.
//
// This is the machine's half of the degenerate input case. A person writes the
// degenerate inputs they can imagine, which is why count zero and an entity
// with no fields have tests. A fuzzer imagines nothing and finds the rest.
func FuzzDecodeNeverPanics(f *testing.F) {
	documented, err := os.ReadFile(filepath.Join("testdata", "documented.yaml"))
	if err != nil {
		f.Fatalf("cannot read the fixture: %v", err)
	}
	f.Add(string(documented))

	for _, seed := range []string{
		"",
		"\n",
		"version: \"0.1\"\n",
		"version: 1.0\n",
		"entities:\n",
		"entities: {}\n",
		"version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y }\n",
		"version: \"0.1\"\nentities:\n  a:\n    count: 0\n    fields:\n      x: { type: uuid }\n",
		"version: \"0.1\"\nentities:\n  a:\n    primary_key: [x, x]\n    fields:\n      x: { type: uuid }\n",
		"version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { generator: y, max_length: 0 }\n",
		"version: \"0.1\"\nentities:\n  a:\n    belongs_to: { a: 5-0 }\n    fields:\n      x: { type: uuid }\n",
		"version: \"0.1\"\nentities:\n  a:\n    consistency:\n      - x matches: y\n    fields:\n      x: { type: uuid }\n",
		"version: \"0.1\"\nentities:\n  a:\n    fields:\n      x: { expression: \"${a.b.c.d}\" }\n",
		"\t\t\t",
		"a: &b\n  *b\n",
		"- - - - - - - -\n",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, source string) {
		result := recipeyaml.Decode([]byte(source))

		// Silence is the failure this product exists to argue against, so a
		// refusal that names nothing would be the same defect one layer down.
		if !result.Valid() && len(result.Problems) == 0 {
			t.Fatalf("the recipe was refused without a single problem named")
		}

		// Every problem has to say where it is and what is wrong. A report
		// missing either sends somebody looking through the whole file.
		for _, found := range result.Problems {
			if found.What == "" {
				t.Fatalf("a problem at %q says nothing about what is wrong", found.Where)
			}
		}
		for _, note := range result.Unchecked {
			if note.What == "" || note.Reason == "" {
				t.Fatalf("an unchecked note at %q is missing its subject or its reason", note.Where)
			}
		}

		// Reading is a function of the bytes and nothing else. A second pass
		// over the same input has to reach the same answer, or something in
		// here is holding state between runs.
		again := recipeyaml.Decode([]byte(source))
		if again.Valid() != result.Valid() || len(again.Problems) != len(result.Problems) {
			t.Fatalf("two reads of the same recipe disagreed")
		}
	})
}
