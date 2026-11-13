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

// nullSurname is the real surname that reads as an absent value.
var nullSurname = []string{"Null"}

// namesFor picks the list for the part of a name a column holds, and nothing
// for a column holding no names.
func namesFor(part recipe.NamePart, given, surnames []string) []string {
	switch part {
	case recipe.GivenName:
		return given
	case recipe.Surname:
		return surnames
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
//
// # Which refusal is which
//
// A value that already is every name on the list is unfit: the value shows it,
// and another column of the row may differ. A list nothing on which fits the
// column is a skip, because the limit is something the column declares rather
// than something the value shows.
func replacedByName(target mutator.Target, names []string, purpose string) mutator.Outcome {
	if len(names) == 0 {
		return mutator.Skipped("nothing says this column holds a part of a person's name, " +
			"and the name this damage supplies is expected accepted only in such a column")
	}
	differing, fitting := 0, 0
	for _, name := range names {
		if name == target.Value {
			continue
		}
		differing++
		if fits(name, target.Field) {
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
		if name == target.Value || !fits(name, target.Field) {
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

// fits reports whether a name is within the limit a column declares, if it
// declares one.
func fits(name string, field recipe.FieldInfo) bool {
	return !field.HasMaxLength || lengthIn(name, field.Unit()) <= field.MaxLength
}
