package recipeyaml

import (
	"slices"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// What can be written at one place of a recipe that is still being typed.
//
// # Why this reads lines rather than asking the parser
//
// A recipe in the middle of being typed is broken most of the time. The line
// under the caret ends in half a word, and the parser refuses a whole document
// for less. What a person needs at that moment is the place of one line, and the
// lines above it say that by their indentation, which is also how a person reads
// YAML. So this reads indentation and the punctuation of the caret's own line,
// and never parses.
//
// # What it reads, and what it leaves without offers
//
// Mappings and lists written as blocks, and a mapping or a list written in
// braces or brackets on the caret's own line, which is how a recipe writes a
// field and a list of names. A collection spread over several lines, a mapping
// inside a list item other than a rule, a comment and a quoted value get no
// offers at all. A list that is not there costs a person nothing, and a list
// for the wrong place teaches them something false.
//
// # Where the words come from
//
// The keys and the fixed values are the lists the reader checks a recipe
// against, so the two cannot offer what the other refuses. The names of the
// generators, the types a build fills and the layers its catalog describes come
// from the caller, for the reason policy gives: they belong to the build, not to
// the file. Names of entities and fields come from the recipe itself.

// Vocabulary is what the build carries and a recipe does not say.
type Vocabulary struct {
	// Generators are the names a field's generator may take.
	Generators []string
	// Types are the types the build fills a field described by type alone with.
	Types []string
	// Layers are the layers the catalog describes damage in.
	Layers []string
}

// OfferKind says what an offer is, so that whoever shows it can describe it.
type OfferKind int

const (
	// OfferKey is a key of the mapping the caret is in.
	OfferKey OfferKind = iota
	// OfferValue is one of the values the key before the caret takes.
	OfferValue
	// OfferName is the name of an entity or a field written elsewhere in the
	// recipe.
	OfferName
)

// Offer is one thing that can be written at the caret.
type Offer struct {
	// Word is the offer as a list shows it, and what typing is matched against.
	Word string
	// Write is what replaces the word being typed when the offer is taken. A
	// key is written with its colon, so that its value can follow at once.
	Write string
	Kind  OfferKind
	// Key is the key a value or a name is written under, and empty for a key.
	Key string
	// Then says that writing the offer begins a place with offers of its own,
	// such as a key whose values are known.
	Then bool
}

// Suggestion is what can be written at the caret, and the word it replaces.
type Suggestion struct {
	// From and To are where the word at the caret begins and ends, counted in
	// characters. They are equal when nothing has been typed there yet.
	From, To int
	// Offers are the ones that match what has been typed, those beginning with
	// it first, each group in the order the format lists them.
	Offers []Offer
}

// unoffered are the keys the reader accepts and a list never offers. This build
// refuses only_when, and it accepts expression and leaves the column empty, so
// an offer of either leads a person to a recipe that does less than it says.
var unoffered = []string{"only_when", "expression"}

// blockKeys are the keys whose value is a block of lines under them, written
// with a colon alone so that Enter begins the block.
var blockKeys = []string{"entities", "output", "policy", "fields", "consistency", "belongs_to"}

// trueFalse are the two values of a key that is switched on or off.
var trueFalse = []string{"true", "false"}

// ruleKinds are the ways one field follows others in a rule of consistency.
var ruleKinds = []string{string(recipe.Matches), string(recipe.DerivedFrom)}

// Suggest says what can be written at the caret, which counts characters into
// the text. An answer with no offers is an ordinary answer: the caret is where
// anything may be written, or where this cannot tell.
func Suggest(text string, caret int, known Vocabulary) Suggestion {
	lines := readLines(text)
	index, column := lines.locate(caret)
	at, ok := lines.spotAt(index, column)
	if !ok {
		return Suggestion{}
	}
	offers := at.offers(lines, known)
	if at.keyed {
		// The word already has its colon after it, so a key replaces the word
		// and nothing more.
		for i := range offers {
			if offers[i].Kind == OfferKey {
				offers[i].Write, offers[i].Then = offers[i].Word, false
			}
		}
	}
	return Suggestion{From: at.from, To: at.to, Offers: matching(offers, at.typed, at.taken)}
}

// matching keeps the offers containing what was typed, ignoring case, those
// beginning with it first, and drops what the mapping or the list already holds.
func matching(all []Offer, typed string, taken []string) []Offer {
	wanted := strings.ToLower(typed)
	var starting, containing []Offer
	for _, offer := range all {
		if slices.Contains(taken, offer.Word) {
			continue
		}
		word := strings.ToLower(offer.Word)
		switch {
		case strings.HasPrefix(word, wanted):
			starting = append(starting, offer)
		case strings.Contains(word, wanted):
			containing = append(containing, offer)
		}
	}
	return append(starting, containing...)
}

// section is the kind of mapping the caret writes in.
type section int

const (
	elsewhere section = iota
	atTop
	inEntity
	inField
	inOutput
	inPolicy
	inParents
	inRules
)

// sectionOf reads the kind of mapping from the keys above it.
func sectionOf(path []string) section {
	switch {
	case len(path) == 0:
		return atTop
	case len(path) == 1 && path[0] == "output":
		return inOutput
	case len(path) == 1 && path[0] == "policy":
		return inPolicy
	case path[0] != "entities":
		return elsewhere
	case len(path) == 2:
		return inEntity
	case len(path) == 4 && path[2] == "fields":
		return inField
	case len(path) == 3 && path[2] == "belongs_to":
		return inParents
	case len(path) == 3 && path[2] == "consistency":
		return inRules
	}
	return elsewhere
}

// keysOf is the keys a section takes, in the order the format lists them.
func keysOf(where section) []string {
	switch where {
	case atTop:
		return topLevelKeys
	case inEntity:
		return entityKeys
	case inField:
		return fieldKeys
	case inOutput:
		return outputKeys
	case inPolicy:
		return policyKeys
	}
	return nil
}

// offers is everything that could be written at the spot, before matching.
func (at spot) offers(lines recipeLines, known Vocabulary) []Offer {
	if at.key != "" {
		return valueOffers(at.mapping, at.key, lines, known)
	}
	where := sectionOf(at.mapping)
	switch where {
	case inParents:
		return named(without(lines.entityNames(), at.mapping[1]), "belongs_to", ": ", false)
	case inRules:
		return ruleOffers(at, lines)
	}
	var offers []Offer
	for _, key := range keysOf(where) {
		if slices.Contains(unoffered, key) {
			continue
		}
		after := ": "
		if slices.Contains(blockKeys, key) {
			after = ":"
		}
		offers = append(offers, Offer{Word: key, Write: key + after, Kind: OfferKey,
			Then: len(valueOffers(at.mapping, key, lines, known)) > 0})
	}
	return offers
}

// valueOffers is what a key takes as its value, or as one item of its list.
func valueOffers(mapping []string, key string, lines recipeLines, known Vocabulary) []Offer {
	where := sectionOf(mapping)
	switch {
	case where == atTop && key == "version":
		return []Offer{{Word: version.Recipe, Write: strconv.Quote(version.Recipe), Kind: OfferValue, Key: key}}
	case where == inEntity && key == "primary_key":
		return named(lines.fieldNames(mapping[1]), key, "", false)
	case where == inPolicy && key == "exclude_fields":
		return named(lines.allFieldNames(), key, "", false)
	case where == inRules:
		// The value of a rule is the fields it follows, which are the
		// fields of the entity other than the one the rule is about.
		field, _, _ := strings.Cut(key, " ")
		return named(without(lines.fieldNames(mapping[1]), field), "consistency", "", false)
	}
	words := valuesOf(where, key, known)
	offers := make([]Offer, 0, len(words))
	for _, word := range words {
		offers = append(offers, Offer{Word: word, Write: word, Kind: OfferValue, Key: key})
	}
	return offers
}

// valuesOf is the values a key takes in a section, and nil for a key whose
// values are anybody's to choose, such as a count or a name.
func valuesOf(where section, key string, known Vocabulary) []string {
	switch where {
	case atTop:
		if key == "locale" {
			return knownLocales
		}
	case inField:
		return fieldValues(key, known)
	case inPolicy:
		switch key {
		case "layers":
			return known.Layers
		case "per_row":
			return perRowChoices
		}
	case inOutput:
		switch key {
		case "format":
			return writableFormats
		case "dialect":
			return writableDialects
		case "file_per_entity", "create_tables":
			return trueFalse
		case "encoding":
			return writableEncodings
		}
	}
	return nil
}

// fieldValues is the values a key of a field takes.
func fieldValues(key string, known Vocabulary) []string {
	switch key {
	case "type":
		return known.Types
	case "generator":
		return known.Generators
	case "unique", "required":
		return trueFalse
	case "counted_in":
		return lengthUnits
	}
	return nil
}

// ruleOffers is what begins an item of consistency: the field the rule is
// about, then the way it follows the others.
func ruleOffers(at spot, lines recipeLines) []Offer {
	if at.rule == "" {
		return named(lines.fieldNames(at.mapping[1]), "consistency", " ", true)
	}
	offers := make([]Offer, 0, len(ruleKinds))
	for _, kind := range ruleKinds {
		offers = append(offers, Offer{Word: kind, Write: kind + ": ", Kind: OfferValue,
			Key: "consistency", Then: true})
	}
	return offers
}

// named turns names written in the recipe into offers.
func named(names []string, key, after string, then bool) []Offer {
	offers := make([]Offer, 0, len(names))
	for _, name := range names {
		offers = append(offers, Offer{Word: name, Write: name + after, Kind: OfferName, Key: key, Then: then})
	}
	return offers
}

// without is a list with one name left out.
func without(names []string, left string) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		if name != left {
			kept = append(kept, name)
		}
	}
	return kept
}
