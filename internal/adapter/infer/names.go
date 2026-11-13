package infer

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// namedHeaders maps a header, as foldedHeader writes it, onto the part of a
// person's name the column under it holds.
//
// # Why a header is read at all, against the rule at the top of this package
//
// No value proves that it is somebody's name. Jordan is a given name, a
// surname and a country, and Nowak spelt in a column of product codes is a
// product code. A check digit is a proof and nothing about a name is one, so
// looking harder at the values would only make the guess more expensive.
//
// The damage that needs the answer supplies a real name, such as O'Brien, and
// expects it accepted. That holds for a column of surnames and says nothing
// about a column of towns, so without an answer here the damage could not run
// on a file at all. The owner chose on 2026-11-13 to read the header for this
// one question (DEC-295). A header is what the person who wrote the file says
// the column holds, which is the same kind of statement as a recipe, made
// with less care, and the basis of every such column says it was read from
// the header so that whoever reads the manifest can disagree.
//
// # Why these words and no others
//
// Each one names a part of a person's name, or the whole of it, and nothing
// else. A bare name is
// left out because a column called name holds products, companies and logins
// as often as people. Abbreviations such as fname are left out because a file
// name is abbreviated the same way. English and Polish because those are the
// languages of the data this build generates. A header in any other language
// is a column of text, and the list of what this mode never reads says so.
//
// The Polish words are written with and without their diacritics, because a
// file exported from a system that strips them is as likely as one that does
// not.
var namedHeaders = map[string]recipe.NamePart{
	"firstname":           recipe.GivenName,
	"givenname":           recipe.GivenName,
	"forename":            recipe.GivenName,
	"middlename":          recipe.GivenName,
	"imie":                recipe.GivenName,
	"imi\U00000119":       recipe.GivenName,
	"drugieimie":          recipe.GivenName,
	"drugieimi\U00000119": recipe.GivenName,
	"lastname":            recipe.Surname,
	"surname":             recipe.Surname,
	"familyname":          recipe.Surname,
	"nazwisko":            recipe.Surname,
	// A whole name, in the words of a form in either language. The Polish
	// ones in both orders, because an official list puts the surname first.
	"fullname":               recipe.FullName,
	"nameandsurname":         recipe.FullName,
	"firstandlastname":       recipe.FullName,
	"imi\U00000119inazwisko": recipe.FullName,
	"imieinazwisko":          recipe.FullName,
	"nazwiskoiimi\U00000119": recipe.FullName,
	"nazwiskoiimie":          recipe.FullName,
	"imi\U00000119nazwisko":  recipe.FullName,
	"imienazwisko":           recipe.FullName,
	"nazwiskoimi\U00000119":  recipe.FullName,
	"nazwiskoimie":           recipe.FullName,
}

// namePartOfHeader returns the part of a person's name a header says its column
// holds, and false when the header is not one this build reads.
func namePartOfHeader(header string) (recipe.NamePart, bool) {
	part, named := namedHeaders[foldedHeader(header)]
	return part, named
}

// foldedHeader writes a header the way namedHeaders is keyed: in small letters,
// composed, and without the spaces, underscores, hyphens, dots, commas and
// slashes that one file puts between the words of a header and another leaves
// out, such as Nazwisko, imię or Imię/Nazwisko.
//
// # Why composed
//
// A file written on a system that stores its accents apart spells imię as five
// code points rather than four, and the two look the same to the person who
// typed the header. Comparing the bytes as they came would read the one and
// not the other.
func foldedHeader(header string) string {
	var folded strings.Builder
	for _, r := range norm.NFC.String(strings.ToLower(header)) {
		if unicode.IsSpace(r) || strings.ContainsRune("_-.,/", r) {
			continue
		}
		folded.WriteRune(r)
	}
	return folded.String()
}

// nameBasis is what a column's basis adds when its header names a part of a
// person's name, or the whole of it.
func nameBasis(part recipe.NamePart) string {
	held := "given names"
	switch part {
	case recipe.Surname:
		held = "surnames"
	case recipe.FullName:
		held = "full names"
	}
	return ", and its header is one this build reads as naming " + held
}
