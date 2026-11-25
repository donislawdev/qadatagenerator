package recipeyaml_test

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
)

// caretMark stands where the caret is in the texts below. It is a character no
// recipe here holds, so a text cannot carry one by accident.
const caretMark = "‸"

// known is a small build: three generators, the two types, four layers.
var known = recipeyaml.Vocabulary{
	Generators: []string{"first_name", "last_name", "pesel"},
	Types:      []string{"integer", "decimal"},
	Layers:     []string{"L0", "L1", "L2", "L3"},
}

// suggestAt asks for the offers at the mark, and says where the mark was, in
// characters.
func suggestAt(t *testing.T, marked string) (recipeyaml.Suggestion, int) {
	t.Helper()
	at := strings.Index(marked, caretMark)
	if at < 0 || strings.Count(marked, caretMark) != 1 {
		t.Fatalf("the text has to carry the caret exactly once: %q", marked)
	}
	caret := utf8.RuneCountInString(marked[:at])
	return recipeyaml.Suggest(strings.Replace(marked, caretMark, "", 1), caret, known), caret
}

func words(offers []recipeyaml.Offer) []string {
	found := make([]string, 0, len(offers))
	for _, offer := range offers {
		found = append(found, offer.Word)
	}
	return found
}

// A recipe the way people write one: blocks of entities, fields in braces, a
// rule, a parent, a policy with a list in brackets and one of items.
const shop = `version: "0.1"
entities:
  customer:
    count: 10
    primary_key: [id]
    fields:
      id: { type: integer }
      imię: { generator: first_name }
      pesel: { generator: pesel }
      birth_date: { generator: first_name }
    consistency:
      - pesel matches: [birth_date]
  payment:
    belongs_to: { customer: 0-5 }
    fields:
      amount: { type: decimal }
`

// Every place a recipe offers something, and the places it must not.
func TestEachPlaceOffersWhatTheFormatTakesThere(t *testing.T) {
	inShop := func(at string) string { return strings.Replace(shop, "count: 10", at, 1) }
	for _, one := range []struct {
		name   string
		marked string
		want   []string
	}{
		{"an empty recipe offers the keys of the top", caretMark,
			[]string{"version", "seed", "locale", "name", "tag", "entities", "output", "policy"}},
		{"a key at the top already written is not offered again", "version: \"0.1\"\n" + caretMark,
			[]string{"seed", "locale", "name", "tag", "entities", "output", "policy"}},
		{"a key is matched by its beginning", "ver" + caretMark, []string{"version"}},
		{"the version is the one this build reads", "version: " + caretMark, []string{"0.1"}},
		{"a locale is matched anywhere in its name", "locale: US" + caretMark, []string{"en-US"}},
		{"an entity takes the keys it does not hold yet", inShop(caretMark),
			[]string{"count", "belongs_to"}},
		{"an entity's keys leave out what it holds", inShop("c" + caretMark), []string{"count"}},
		{"a field in braces takes the keys of a field, and not expression",
			strings.Replace(shop, "{ type: integer }", "{ "+caretMark+" }", 1),
			[]string{"type", "generator", "unique", "required", "max_length", "counted_in", "empty",
				"values", "min_value", "max_value"}},
		{"a field in braces leaves out the keys it holds",
			strings.Replace(shop, "{ type: integer }", "{ type: integer, "+caretMark+" }", 1),
			[]string{"generator", "unique", "required", "max_length", "counted_in", "empty",
				"values", "min_value", "max_value"}},
		{"a generator is matched by its beginning",
			strings.Replace(shop, "{ generator: pesel }", "{ generator: pe"+caretMark+" }", 1),
			[]string{"pesel"}},
		{"a generator is matched inside its name, after those beginning with it",
			strings.Replace(shop, "{ generator: pesel }", "{ generator: name"+caretMark+" }", 1),
			[]string{"first_name", "last_name"}},
		{"a type is one the build fills",
			strings.Replace(shop, "{ type: integer }", "{ type: "+caretMark+" }", 1),
			[]string{"integer", "decimal"}},
		{"a flag is true or false",
			strings.Replace(shop, "{ type: integer }", "{ type: integer, unique: t"+caretMark+" }", 1),
			[]string{"true"}},
		{"a key is offered among the fields of the entity",
			strings.Replace(shop, "primary_key: [id]", "primary_key: ["+caretMark+"]", 1),
			[]string{"id", "imię", "pesel", "birth_date"}},
		{"a rule begins with a field",
			strings.Replace(shop, "- pesel matches: [birth_date]", "- pe"+caretMark, 1),
			[]string{"pesel"}},
		{"a rule goes on with the way it follows",
			strings.Replace(shop, "- pesel matches: [birth_date]", "- pesel "+caretMark, 1),
			[]string{"matches", "derived_from"}},
		{"a rule follows the other fields of its entity",
			strings.Replace(shop, "- pesel matches: [birth_date]", "- pesel matches: [birth_date, "+caretMark+"]", 1),
			[]string{"id", "imię"}},
		{"a parent is another entity of the recipe",
			strings.Replace(shop, "belongs_to: { customer: 0-5 }", "belongs_to:\n      "+caretMark, 1),
			[]string{"customer"}},
		{"the layers in brackets leave out those named",
			"policy:\n  layers: [L1, " + caretMark + "]", []string{"L0", "L2", "L3"}},
		{"the word at the caret does not leave itself out",
			"policy:\n  layers: [L1, L2" + caretMark + "]", []string{"L2"}},
		{"the layers as items leave out those named",
			"policy:\n  layers:\n    - L1\n    - " + caretMark, []string{"L0", "L2", "L3"}},
		{"the items of a list may stand at the indent of its key",
			"policy:\n  layers:\n  - L1\n  - L" + caretMark, []string{"L0", "L2", "L3"}},
		{"a policy takes its keys, and not only_when", "policy:\n  " + caretMark,
			[]string{"layers", "rows", "per_row", "exclude_fields"}},
		{"defects per row are the two the format names", "policy:\n  per_row: " + caretMark,
			[]string{"exactly_one", "any"}},
		{"columns to leave alone are the fields of every entity",
			shop + "policy:\n  exclude_fields: [" + caretMark + "]",
			[]string{"id", "imię", "pesel", "birth_date", "amount"}},
		{"the output takes its keys", "output:\n  " + caretMark,
			[]string{"format", "dialect", "create_tables", "dir", "file_per_entity", "encoding", "delimiter"}},
		{"a form is one this build writes", "output:\n  format: " + caretMark, []string{"csv", "json", "sql"}},
		{"a dialect is a database the sql form writes for", "output:\n  dialect: " + caretMark,
			[]string{"postgresql", "mysql", "sqlserver", "sqlite"}},
		{"a line ending in a carriage return is one line", "version: \"0.1\"\r\nent" + caretMark,
			[]string{"entities"}},
	} {
		t.Run(one.name, func(t *testing.T) {
			got, _ := suggestAt(t, one.marked)
			if !slices.Equal(words(got.Offers), one.want) {
				t.Errorf("offered %q, want %q", words(got.Offers), one.want)
			}
		})
	}
}

// Where anything may be written, or where this cannot tell, nothing is offered,
// because a list for the wrong place teaches a person something false.
func TestNothingIsOfferedWhereTheFormatTakesAnyWordOrTheShapeIsNotRead(t *testing.T) {
	for _, one := range []struct{ name, marked string }{
		{"the name of an entity", "entities:\n  " + caretMark},
		{"the name of a field", "entities:\n  customer:\n    fields:\n      " + caretMark},
		{"a count", "entities:\n  customer:\n    count: " + caretMark},
		{"a comment", "# gen" + caretMark},
		{"a comment after a value", "version: \"0.1\" # ver" + caretMark},
		{"a quoted value", "seed: \"ab" + caretMark},
		{"after a list that closed", "policy:\n  layers: [L1] " + caretMark},
		{"under a key with a value", "seed: x\n  " + caretMark},
		{"a key the format does not take", "policy:\n  on" + caretMark},
		{"expression, which this build leaves empty",
			"entities:\n  customer:\n    fields:\n      a: { ex" + caretMark + " }"},
		{"a line under a rule's item", "entities:\n  c:\n    consistency:\n      - a matches: [b]\n        " + caretMark},
		{"a key under the rules, which are items", "entities:\n  c:\n    consistency:\n      " + caretMark},
		{"two words where one is typed", "policy:\n  per_row: exactly one" + caretMark},
	} {
		t.Run(one.name, func(t *testing.T) {
			if got, _ := suggestAt(t, one.marked); len(got.Offers) > 0 {
				t.Errorf("offered %q where nothing belongs", words(got.Offers))
			}
		})
	}
}

// The answer says which characters an offer replaces: the whole word at the
// caret, the part after it included, counted in characters rather than bytes.
func TestAnOfferReplacesTheWholeWordAtTheCaret(t *testing.T) {
	marked := "entities:\n  klient:\n    fields:\n      imię: { generator: pe" + caretMark + "sel }"
	got, caret := suggestAt(t, marked)
	if from, to := caret-len("pe"), caret+len("sel"); got.From != from || got.To != to {
		t.Errorf("the word runs from %d to %d, want %d to %d", got.From, got.To, from, to)
	}
	if !slices.Equal(words(got.Offers), []string{"pesel"}) {
		t.Errorf("offered %q, want pesel", words(got.Offers))
	}
}

// A key is written with its colon, and says whether its value has offers of
// its own. A key whose value is a block takes the colon alone, so that Enter
// begins the block. A key typed in front of its own colon is written bare.
func TestAKeyIsWrittenSoThatItsValueCanFollow(t *testing.T) {
	inEntity := "entities:\n  customer:\n    "
	for _, one := range []struct {
		name, marked, word, write string
		then                      bool
	}{
		{"a key with values", inEntity + "fields:\n      a: { gen" + caretMark + " }", "generator", "generator: ", true},
		{"a key with values in a block", "out" + caretMark, "output", "output:", false},
		{"a key with free values", inEntity + "cou" + caretMark, "count", "count: ", false},
		{"a key before its own colon", inEntity + "cou" + caretMark + "nt: 5", "count", "count", false},
		{"a field of a rule", inEntity + "fields:\n      a: {}\n    consistency:\n      - " + caretMark, "a", "a ", true},
		{"the way a rule follows", inEntity + "consistency:\n      - a m" + caretMark, "matches", "matches: ", true},
		{"a version, in quotes", "version: " + caretMark, "0.1", "\"0.1\"", false},
	} {
		t.Run(one.name, func(t *testing.T) {
			got, _ := suggestAt(t, one.marked)
			for _, offer := range got.Offers {
				if offer.Word != one.word {
					continue
				}
				if offer.Write != one.write || offer.Then != one.then {
					t.Errorf("%s writes %q and goes on %v, want %q and %v",
						offer.Word, offer.Write, offer.Then, one.write, one.then)
				}
				return
			}
			t.Errorf("%s was not offered among %q", one.word, words(got.Offers))
		})
	}
}

// bigRecipe is a recipe of entities of ten fields each, as many as fit under
// the largest recipe this build reads, ending where the caret is put.
func bigRecipe(entities int, ending string) string {
	var text strings.Builder
	text.WriteString("version: \"0.1\"\nentities:\n")
	for entity := range entities {
		text.WriteString("  entity_" + strconv.Itoa(entity) + ":\n    count: 10\n    fields:\n")
		for field := range 10 {
			text.WriteString("      field_" + strconv.Itoa(field) + ": { generator: first_name, required: true }\n")
		}
	}
	return text.String() + ending
}

// The cost of one answer at the largest recipe this build reads, which is
// paid on every key pressed while the list is open. Two shapes: a value under
// the caret, which reads the lines above it, and the columns a policy leaves
// alone, which reads the fields of every entity.
func BenchmarkSuggestAtTheLargestRecipe(b *testing.B) {
	for _, one := range []struct {
		name, ending string
	}{
		{"a generator", "  last:\n    fields:\n      a: { generator: pe"},
		{"the columns of every entity", "policy:\n  exclude_fields: [f"},
	} {
		entities := 1
		for len(bigRecipe(entities*2, one.ending)) < recipeyaml.MaxSize {
			entities *= 2
		}
		text := bigRecipe(entities, one.ending)
		caret := utf8.RuneCountInString(text)
		b.Run(one.name, func(b *testing.B) {
			for b.Loop() {
				recipeyaml.Suggest(text, caret, known)
			}
			b.ReportMetric(float64(len(text))/1024, "KB")
			b.ReportMetric(float64(entities), "entities")
		})
	}
}

// A caret outside the text is held at its nearer end rather than read past it.
func TestACaretOutsideTheTextIsHeldInside(t *testing.T) {
	for _, caret := range []int{-5, 1000} {
		got := recipeyaml.Suggest("ver", caret, known)
		if caret > 0 && !slices.Equal(words(got.Offers), []string{"version"}) {
			t.Errorf("a caret past the end offered %q, want version", words(got.Offers))
		}
		if got.From > got.To {
			t.Errorf("a caret at %d gave a word from %d to %d", caret, got.From, got.To)
		}
	}
}
