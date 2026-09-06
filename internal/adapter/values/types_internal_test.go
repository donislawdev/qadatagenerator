package values

import "testing"

// The two registers have to name the same generators, and the direction that
// matters is the one that fails quietly.
//
// # What went wrong once, on 2026-09-06
//
// A fourteenth generator was added to producers and not to types. Nothing
// failed. The column it fills carries no type, an entry of the catalog covers
// an untyped column only through the reserved word any, and two of the fifty
// two entries use it. So the new scheme silently left fifty entries
// inapplicable, which is the exact failure the comment above types was written
// about after it happened to a different generator.
//
// It was found by looking at the window, where the row read as a name, a dash
// and nothing. A register closed in one direction answers half the question,
// and this is the other half.
func TestEveryGeneratorSaysWhatKindOfValueItProduces(t *testing.T) {
	for name := range producers {
		kind, declared := types[name]
		if !declared {
			t.Errorf("the generator %s produces values and no register says of what kind, "+
				"so the column it fills is matched only by entries using the reserved word any", name)
			continue
		}
		if kind == "" {
			t.Errorf("the generator %s is registered with an empty kind, which the schema view "+
				"cannot tell from a generator nobody registered", name)
		}
	}
}

// The other direction. A kind registered for a generator that no longer exists
// is not harmless: it is a line that reads like knowledge and describes nothing,
// and the next person to touch this file will trust it.
func TestEveryKindInTheRegisterBelongsToAGeneratorThisBuildHas(t *testing.T) {
	for name := range types {
		if _, produced := producers[name]; !produced {
			t.Errorf("the register says %s produces a kind of value and this build has no such "+
				"generator, so the entry describes nothing", name)
		}
	}
}
