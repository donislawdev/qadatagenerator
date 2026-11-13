package mutators

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// The corpus of names this build carries, which is what the capability
// corpus.names-international stands for in a run.
//
// # Why real names rather than the value with something done to it
//
// Each entry about names expects the name it supplies to be accepted, and that
// is a claim about a name a person really has. Putting an apostrophe in front
// of Kowalski, or accents over the letters of Smith, makes text with the right
// characters in it and a name nobody is called, and a report reading "was
// Kowalski, now O'Kowalski" leaves its reader wondering whether this tool made
// a mistake. A real name replacing another real name reads as what it is.
//
// # Why a handful and not a list on disk
//
// The word lists on disk are chosen by locale and drawn to look like ordinary
// data. These are the opposite: few, chosen for the character each of them
// carries, and the same in every locale. Every non ASCII character is written
// as an escape, with the name spelt out beside it, so the bytes are the ones
// meant whatever the editor that opens the file does to them.
//
// # What every list has to keep true, and a test holds it
//
// Each name in a list carries what its entry is about, so drawing any of them
// is the damage the entry describes. A test reads every list against its entry,
// and against the published example where there is one.

// apostropheSurnames are surnames with an apostrophe, the plain one on every
// keyboard rather than the curly one a word processor puts in its place.
var apostropheSurnames = []string{"O'Brien", "O'Connor", "D'Angelo", "D'Arcy"}

// apostropheFullNames are the same surnames each after a given name of the
// language it comes from, for a column holding a whole name.
var apostropheFullNames = []string{"Sean O'Brien", "Mary O'Connor", "Marco D'Angelo", "Claire D'Arcy"}

// diacriticGivenNames are given names with a letter that carries a mark, each
// written as one composed character. The decomposed writing is another entry.
var diacriticGivenNames = []string{
	"\U00000141ukasz",        // Łukasz
	"Ma\U00000142gorzata",    // Małgorzata
	"J\U000000F3zef",         // Józef
	"Jos\U000000E9",          // José
	"Fran\U000000E7ois",      // François
	"Ji\U00000159\U000000ED", // Jiří
}

// diacriticSurnames are surnames with a letter that carries a mark, composed
// for the same reason.
var diacriticSurnames = []string{
	"W\U000000F3jcik",           // Wójcik
	"K\U00000119dzierski",       // Kędzierski
	"Ga\U00000142\U00000105zka", // Gałązka
	"Dvo\U00000159\U000000E1k",  // Dvořák
	"M\U000000FCller",           // Müller
	"Gon\U000000E7alves",        // Gonçalves
}

// diacriticFullNames are whole names with a marked letter, each a given name
// and a surname of one language, and a Polish surname in the form that agrees
// with the given name, so no pair reads as a name nobody could have.
var diacriticFullNames = []string{
	"\U00000141ukasz W\U000000F3jcik",                 // Łukasz Wójcik
	"Ma\U00000142gorzata K\U00000119dzierska",         // Małgorzata Kędzierska
	"J\U000000F3zef Ga\U00000142\U00000105zka",        // Józef Gałązka
	"Jos\U000000E9 Gon\U000000E7alves",                // José Gonçalves
	"Fran\U000000E7ois Lef\U000000E8vre",              // François Lefèvre
	"Ji\U00000159\U000000ED Dvo\U00000159\U000000E1k", // Jiří Dvořák
}

// cjkGivenNames are given names in Chinese, Japanese and Korean writing, each
// two characters long, which is the length the entry is about: complete, and
// shorter than a minimum of three.
var cjkGivenNames = []string{
	"\U00007F8E\U000054B2", // 美咲, Misaki
	"\U00007FD4\U0000592A", // 翔太, Shota
	"\U00004F1F\U0000660E", // 伟明, Weiming
	"\U0000BBFC\U0000C900", // 민준, Minjun
}

// cjkSurnames are surnames in the same three writings, two characters each.
var cjkSurnames = []string{
	"\U00004F50\U000085E4", // 佐藤, Sato
	"\U00006B27\U00009633", // 欧阳, Ouyang
	"\U00008BF8\U0000845B", // 诸葛, Zhuge
	"\U0000B0A8\U0000AD81", // 남궁, Namgung
}

// cjkFullNames are whole Chinese names of two characters, a surname of one and
// a given name of one, written as they are written at home: the surname
// first and nothing between. Two characters is the length the entry is about,
// and a whole Japanese or Korean name is three characters or more, so only
// Chinese ones are here.
var cjkFullNames = []string{
	"\U0000674E\U0000660E", // 李明, Li Ming
	"\U0000738B\U000082B3", // 王芳, Wang Fang
	"\U00005F20\U00004F1F", // 张伟, Zhang Wei
	"\U00005218\U00006D0B", // 刘洋, Liu Yang
}

// nullSurname is the real surname that reads as an absent value.
var nullSurname = []string{"Null"}

// nullFullNames carry the surname Null after a common given name.
var nullFullNames = []string{"Anna Null", "John Null"}

// nameLists are the names one entry supplies, a list for each part of a name a
// column can hold. A part the entry has nothing for is left empty, and the
// entry then supplies nothing to such a column.
type nameLists struct {
	given, surnames, whole []string
}

// For picks the list for the part of a name a column holds, and nothing for a
// column holding no names.
//
// A whole name is given for a whole name. A surname put alone into a column
// that holds whole names would be a value the column was not meant to hold,
// and the report would ask whether the application was right to refuse it
// for that rather than the one question the entry asks.
func (l nameLists) For(part recipe.NamePart) []string {
	switch part {
	case recipe.GivenName:
		return l.given
	case recipe.Surname:
		return l.surnames
	case recipe.FullName:
		return l.whole
	}
	return nil
}

// replacedByName puts one of a list of real names in place of a value.
//
// The name is drawn, so that the rows of one run carry different names and the
// same seed gives the same ones. It is drawn from the names that differ from
// the value and fit the column, counted twice rather than gathered into a new
// list, because this runs once per damaged cell and the lists are short.
//
// # Why a name that would overrun the column is never drawn
//
// The entry expects the name accepted. A column that declares a limit refuses a
// longer value for its length, rightly, and a manifest saying it should have
// been accepted would blame the application for a rule it was right to keep.
// The engine refuses such a value from any damage, and choosing among the names
// that fit is what lets a shorter one land here instead of a refusal.
//
// # Which refusal is which
//
// A value that already is every name on the list is unfit: the value shows it,
// and another column of the row may differ. A list nothing on which fits the
// column is a skip, because the limit is something the column declares rather
// than something the value shows.
func replacedByName(target mutator.Target, names []string, purpose string) mutator.Outcome {
	if len(names) == 0 {
		return mutator.Skipped("nothing says this column holds the kind of name this damage " +
			"supplies, and that name is expected accepted only in such a column")
	}
	differing, fitting := 0, 0
	for _, name := range names {
		if name == target.Value {
			continue
		}
		differing++
		if target.Field.Fits(name) {
			fitting++
		}
	}
	if differing == 0 {
		return mutator.Unfit(fmt.Sprintf("the value already is %s, so supplying it changes nothing",
			target.Value))
	}
	if fitting == 0 {
		return mutator.Skipped(fmt.Sprintf("the column allows at most %d %s, and every name this "+
			"damage can supply is longer, so the application would be right to refuse it",
			target.Field.MaxLength, target.Field.Unit()))
	}
	at := target.At
	at.Purpose += purpose
	wanted := target.Source.IntN(at, int64(fitting))
	for _, name := range names {
		if name == target.Value || !target.Field.Fits(name) {
			continue
		}
		if wanted == 0 {
			return mutator.Done(name)
		}
		wanted--
	}
	// Unreachable: the loop above walks the same names the count did, so the
	// drawn one is among them. Said rather than left as a panic, because a
	// mutator answers with an outcome and never with nothing.
	return mutator.Skipped("no name was left to supply after the draw")
}
