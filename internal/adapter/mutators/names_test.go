package mutators_test

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

const (
	nameApostrophe  catalog.ID = "REV.name.apostrophe"
	nameDiacritics  catalog.ID = "REV.name.diacritics"
	nameNullSurname catalog.ID = "REV.name.null_surname"
	nameCJK         catalog.ID = "REV.name.cjk"
)

// nameRows is how many rows each draw below is asked over. Enough that a list
// of a few names drawn uniformly shows every one of them, so a draw stuck on
// the first name is seen rather than missed.
const nameRows = 200

// nameEntries are the entries about names, each with the parts of a name its
// catalog entry lets it land on and what every name it supplies has to carry.
var nameEntries = []struct {
	id      catalog.ID
	parts   []recipe.NamePart
	carries func(string) bool
	about   string
}{
	{nameApostrophe, []recipe.NamePart{recipe.Surname, recipe.FullName}, carriesApostrophe,
		"an apostrophe, the plain one, in a name of letters"},
	{nameDiacritics, []recipe.NamePart{recipe.GivenName, recipe.Surname, recipe.FullName},
		carriesMarkedLetter, "a Latin letter outside ASCII, composed"},
	{nameNullSurname, []recipe.NamePart{recipe.Surname, recipe.FullName},
		func(name string) bool { return name == "Null" }, "the surname Null"},
	{nameCJK, []recipe.NamePart{recipe.GivenName, recipe.Surname, recipe.FullName},
		carriesTwoCJKCharacters, "two Chinese, Japanese or Korean characters"},
}

// carriesAsPart asks a part of a name whether it carries what its entry is
// about, and a whole name whether it does or one of its words does, because
// the apostrophe of Sean O'Brien and the Null of Anna Null are in the surname.
func carriesAsPart(carries func(string) bool, name string, part recipe.NamePart) bool {
	if carries(name) {
		return true
	}
	if part != recipe.FullName {
		return false
	}
	for _, word := range strings.Fields(name) {
		if carries(word) {
			return true
		}
	}
	return false
}

func nameColumn(part recipe.NamePart) recipe.FieldInfo {
	return recipe.FieldInfo{Name: "who", Type: "string", NamePart: part}
}

// supplied is every name an entry gives in place of before, over many rows of
// one seed, in the order of the rows.
func supplied(t *testing.T, id catalog.ID, field recipe.FieldInfo, before, run string) []string {
	t.Helper()
	names := make([]string, 0, nameRows)
	source := seed.New(run)
	for row := 0; row < nameRows; row++ {
		outcome := damage(t, id, mutator.Target{
			Value: before, Field: field, Source: source,
			At: seed.Coordinate{Entity: "people", Row: int64(row), Field: field.Name, Purpose: "corrupt"},
		})
		if !outcome.Applied() {
			t.Fatalf("%s left %q alone in row %d: %s", id, before, row, outcome.Reason())
		}
		names = append(names, outcome.Value())
	}
	return names
}

func distinct(names []string) map[string]bool {
	seen := map[string]bool{}
	for _, name := range names {
		seen[name] = true
	}
	return seen
}

// 🔴 Every name an entry supplies carries what the entry is about, differs from
// the value it replaced, and more than one name is drawn where the list has
// more than one.
//
// The manifest promises the name is accepted because of what it carries. A
// list holding one name without an apostrophe would make that promise for a
// value that tests nothing, and every row it was drawn in would say so.
func TestEveryNameSuppliedCarriesWhatItsEntryIsAbout(t *testing.T) {
	for _, entry := range nameEntries {
		for _, part := range entry.parts {
			names := supplied(t, entry.id, nameColumn(part), "Nowak", "names")
			for _, name := range names {
				if !carriesAsPart(entry.carries, name, part) {
					t.Errorf("%s gave %q for a column of %s, which is not %s",
						entry.id, name, part, entry.about)
				}
				if name == "Nowak" {
					t.Errorf("%s gave back the value it was asked to replace", entry.id)
				}
			}
			if many := len(distinct(names)); entry.id != nameNullSurname && many < 2 {
				t.Errorf("%s drew %d distinct names over %d rows for a column of %s",
					entry.id, many, nameRows, part)
			}
		}
	}
}

// The part of a name is kept: a given name replaces a given name and a surname
// a surname, so the report asks only the question the entry asks. Two lists
// sharing a name would make that unprovable from the outside, so they are held
// apart here.
func TestAGivenNameReplacesAGivenNameAndASurnameASurname(t *testing.T) {
	for _, id := range []catalog.ID{nameDiacritics, nameCJK} {
		given := distinct(supplied(t, id, nameColumn(recipe.GivenName), "Anna", "parts"))
		for name := range distinct(supplied(t, id, nameColumn(recipe.Surname), "Nowak", "parts")) {
			if given[name] {
				t.Errorf("%s supplies %q both as a given name and as a surname", id, name)
			}
		}
	}
}

// 🔴 A whole name replaces a whole name: a given name and a surname with one
// space between, or two Chinese characters with none, and never a name the
// same entry gives a column of surnames.
//
// A column of whole names that got O'Brien alone would hold a value it was not
// meant to hold, and the report would ask whether the application was right to
// refuse it for that rather than the one question the entry asks.
func TestAWholeNameReplacesAWholeName(t *testing.T) {
	if isWholeName("O'Brien") || isWholeName("Sean  O'Brien") || isWholeName("Sean O'Brien Jr") ||
		!isWholeName("Sean O'Brien") || !isWholeName("\U0000674E\U0000660E") {
		t.Fatal("the shape of a whole name is not told apart from a part of one")
	}
	for _, entry := range nameEntries {
		surnames := distinct(supplied(t, entry.id, nameColumn(recipe.Surname), "Nowak", "whole"))
		for name := range distinct(supplied(t, entry.id, nameColumn(recipe.FullName), "Anna Nowak", "whole")) {
			if !isWholeName(name) {
				t.Errorf("%s gave %q to a column of whole names, which is not a whole name",
					entry.id, name)
			}
			if surnames[name] {
				t.Errorf("%s supplies %q both as a surname and as a whole name", entry.id, name)
			}
		}
	}
}

// The published example is a name the entry really supplies, in the part of a
// name the example shows. A specification promising Łukasz from an
// implementation that cannot produce it would be read by another
// implementation as the behaviour to copy.
func TestThePublishedExampleIsANameItsEntrySupplies(t *testing.T) {
	for _, given := range []struct {
		id   catalog.ID
		part recipe.NamePart
	}{
		{nameApostrophe, recipe.Surname},
		{nameDiacritics, recipe.GivenName},
		{nameNullSurname, recipe.Surname},
		{nameCJK, recipe.Surname},
	} {
		example := entryOf(t, given.id).Example
		names := distinct(supplied(t, given.id, nameColumn(given.part), example.Before, "example"))
		if !names[example.After] {
			t.Errorf("%s publishes %q -> %q and supplies only %v in a column of %s",
				given.id, example.Before, example.After, names, given.part)
		}
	}
}

// The same seed gives the same names in the same rows, and another seed gives
// others somewhere, which is what lets a run be repeated in CI and still vary.
func TestTheNameDrawnIsTheSeedsToChoose(t *testing.T) {
	first := supplied(t, nameApostrophe, nameColumn(recipe.Surname), "Nowak", "repeat")
	again := supplied(t, nameApostrophe, nameColumn(recipe.Surname), "Nowak", "repeat")
	other := supplied(t, nameApostrophe, nameColumn(recipe.Surname), "Nowak", "another seed")
	if strings.Join(first, "|") != strings.Join(again, "|") {
		t.Error("one seed drew different names in two runs")
	}
	if strings.Join(first, "|") == strings.Join(other, "|") {
		t.Error("two seeds drew the same name in every one of the rows")
	}
}

// A value that already is a name on the list is never given back to itself,
// and a value that is the only name on the list is unfit, so the engine may
// try another column of the row instead of reporting damage that changed
// nothing.
func TestANameIsNeverSuppliedInPlaceOfItself(t *testing.T) {
	for _, name := range supplied(t, nameApostrophe, nameColumn(recipe.Surname), "O'Brien", "itself") {
		if name == "O'Brien" {
			t.Fatal("O'Brien was replaced by O'Brien")
		}
	}
	outcome := damage(t, nameNullSurname, mutator.Target{
		Value: "Null", Field: nameColumn(recipe.Surname), Source: seed.New("itself"),
	})
	if outcome.Applied() || !outcome.ValueUnfit() {
		t.Errorf("Null in place of Null came back applied %v, unfit %v: %s",
			outcome.Applied(), outcome.ValueUnfit(), outcome.Reason())
	}
}

// 🔴 A name longer than the column declares is never drawn, and a column no name
// fits is skipped with its limit named.
//
// The entry expects the name accepted, and a column with a limit refuses a
// longer value for its length, rightly. Counted in the unit the column
// declares, because a name of two Chinese characters is two characters and six
// bytes, and a limit of five bytes refuses it.
func TestANameLongerThanTheColumnAllowsIsNeverSupplied(t *testing.T) {
	for _, given := range []struct {
		id    catalog.ID
		part  recipe.NamePart
		limit int
		unit  recipe.LengthUnit
	}{
		{nameApostrophe, recipe.Surname, 7, recipe.Characters},
		{nameDiacritics, recipe.GivenName, 4, recipe.Characters},
		{nameCJK, recipe.Surname, 6, recipe.Bytes},
		{nameCJK, recipe.GivenName, 2, recipe.Characters},
	} {
		field := nameColumn(given.part)
		field.MaxLength, field.HasMaxLength, field.LengthUnit = given.limit, true, given.unit
		for _, name := range supplied(t, given.id, field, "Nowak", "limit") {
			length := utf8.RuneCountInString(name)
			if given.unit == recipe.Bytes {
				length = len(name)
			}
			if length > given.limit {
				t.Errorf("%s supplied %q, %d %s, to a column allowing %d",
					given.id, name, length, given.unit, given.limit)
			}
		}
	}
	for _, given := range []struct {
		id    catalog.ID
		limit int
		unit  recipe.LengthUnit
	}{
		{nameApostrophe, 5, recipe.Characters},
		{nameCJK, 5, recipe.Bytes},
	} {
		field := nameColumn(recipe.Surname)
		field.MaxLength, field.HasMaxLength, field.LengthUnit = given.limit, true, given.unit
		outcome := damage(t, given.id, mutator.Target{Value: "Nowak", Field: field, Source: seed.New("limit")})
		if outcome.Applied() || outcome.ValueUnfit() || !strings.Contains(outcome.Reason(), "at most") {
			t.Errorf("%s with a limit of %d %s came back applied %v, unfit %v: %s", given.id,
				given.limit, given.unit, outcome.Applied(), outcome.ValueUnfit(), outcome.Reason())
		}
	}
}

// A column holding no names is refused by the damage itself as well as by the
// engine, which stops it first. The engine is the one that decides, and this
// is what a caller that skipped it would meet instead of a name in a column of
// towns.
//
// The same holds for a column of given names and the two entries about
// surnames, which have no given name to supply.
func TestAColumnHoldingNoNamesIsRefused(t *testing.T) {
	for _, given := range []struct {
		id   catalog.ID
		part recipe.NamePart
	}{
		{nameDiacritics, ""},
		{nameCJK, ""},
		{nameApostrophe, ""},
		{nameNullSurname, ""},
		{nameApostrophe, recipe.GivenName},
		{nameNullSurname, recipe.GivenName},
	} {
		outcome := damage(t, given.id, mutator.Target{Value: "Szczyrk", Field: nameColumn(given.part),
			Source: seed.New("towns")})
		if outcome.Applied() || outcome.ValueUnfit() {
			t.Errorf("%s in a column holding %q came back applied %v, unfit %v",
				given.id, given.part, outcome.Applied(), outcome.ValueUnfit())
		}
	}
}

// The corpus of names travels with the damage, so neither mode refuses these
// entries for want of it, and what they need of the column is asked of the
// column rather than of the run.
func TestTheNameEntriesNeedTheCorpusOfTheRunAndTheNamesOfTheColumn(t *testing.T) {
	carried := capability.NewSet(mutators.Brings()...)
	for _, entry := range nameEntries {
		required := entryOf(t, entry.id).Requires
		for _, missing := range carried.Missing(required) {
			if scope, _ := capability.ScopeOf(missing); scope != capability.Field {
				t.Errorf("%s requires %s of the run, which the damage does not carry",
					entry.id, missing)
			}
		}
		if len(capability.FieldScoped(required)) == 0 {
			t.Errorf("%s requires nothing of the column, so it would land on a column of towns",
				entry.id)
		}
	}
}

func carriesApostrophe(name string) bool {
	if !strings.Contains(name, "'") || strings.ContainsRune(name, '\U00002019') {
		return false
	}
	for _, r := range name {
		if r != '\'' && !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func carriesMarkedLetter(name string) bool {
	if !norm.NFC.IsNormalString(name) {
		return false
	}
	for _, r := range name {
		if r > unicode.MaxASCII && unicode.Is(unicode.Latin, r) {
			return true
		}
	}
	return false
}

// isWholeName says whether a name is a given name and a surname: two words of
// letters with one space between, or two Chinese characters written with
// nothing between, as a whole Chinese name is.
func isWholeName(name string) bool {
	if carriesTwoCJKCharacters(name) {
		return true
	}
	words := strings.Split(name, " ")
	if len(words) != 2 {
		return false
	}
	for _, word := range words {
		if word == "" {
			return false
		}
		for _, r := range word {
			if r != '\'' && !unicode.IsLetter(r) {
				return false
			}
		}
	}
	return true
}

func carriesTwoCJKCharacters(name string) bool {
	if utf8.RuneCountInString(name) != 2 {
		return false
	}
	for _, r := range name {
		if !unicode.In(r, unicode.Han, unicode.Hangul) {
			return false
		}
	}
	return true
}
