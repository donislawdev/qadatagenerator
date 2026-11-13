package infer_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// headerReading is the sentence a column's basis carries when its part of a
// name was read off its header, which is how a reader of the manifest tells a
// statement read from a proof found.
const headerReading = "its header is one this build reads"

// 🔴 A header from the short list marks its column as holding that part of a
// name, however the file spells the words apart, and no other header does.
//
// The headers that must not match are the point as much as the ones that must.
// A bare name holds products as often as people, fname is also a file name, and
// a column holding whole names has no part this build can damage, so each of
// those would land a hard expectation of acceptance on a column it does not
// describe.
func TestAHeaderNamingAPartOfANameMarksItsColumn(t *testing.T) {
	for header, want := range map[string]recipe.NamePart{
		"first_name":               recipe.GivenName,
		"First Name":               recipe.GivenName,
		"FIRSTNAME":                recipe.GivenName,
		"given-name":               recipe.GivenName,
		"Middle.Name":              recipe.GivenName,
		"forename":                 recipe.GivenName,
		"imie":                     recipe.GivenName,
		"Imi\U00000119":            recipe.GivenName,
		"IMI\U00000118":            recipe.GivenName,
		"imie\U00000328":           recipe.GivenName,
		"drugie imi\U00000119":     recipe.GivenName,
		"last_name":                recipe.Surname,
		"LastName":                 recipe.Surname,
		"surname":                  recipe.Surname,
		"Family Name":              recipe.Surname,
		"Nazwisko":                 recipe.Surname,
		"name":                     "",
		"fname":                    "",
		"Nachname":                 "",
		"full_name":                "",
		"imi\U00000119 i nazwisko": "",
		"nazwiska":                 "",
		"first_name_2":             "",
		"city":                     "",
	} {
		shape := shapeOf(t, tableOf(header),
			[]dataset.Sample{sampleOf(header, []string{"Anna", "Ewa", "Jan"})}, schemes())
		finding := findingFor(t, shape, header)
		if finding.NamePart != want {
			t.Errorf("the header %+q was read as holding %q, want %q", header, finding.NamePart, want)
		}
		if read := strings.Contains(finding.Basis, headerReading); read != (want != "") {
			t.Errorf("the header %+q: the basis says it was read off the header %v, want %v: %s",
				header, read, want != "", finding.Basis)
		}
		if fields := shape.FieldsOf("records"); len(fields) != 1 || fields[0].NamePart != want {
			t.Errorf("the header %+q reaches the engine as %+v, want a column holding %q",
				header, fields, want)
		}
	}
}

// A check digit is a proof and a header is a statement, so a column whose
// values prove a numbering scheme keeps the scheme whatever its header says.
// Taken for surnames it would invite damage expecting O'Brien accepted where a
// national identifier belongs.
func TestAColumnProvedByACheckDigitIsNotTakenForNamesByItsHeader(t *testing.T) {
	shape := shapeOf(t, tableOf("nazwisko"),
		[]dataset.Sample{sampleOf("nazwisko", produced(t, values.Pesel, 40))}, schemes())
	finding := findingFor(t, shape, "nazwisko")
	if finding.Scheme != "PL.pesel" || finding.NamePart != "" {
		t.Errorf("a column of national identifiers headed nazwisko came out as scheme %q holding %q",
			finding.Scheme, finding.NamePart)
	}
	if strings.Contains(finding.Basis, headerReading) {
		t.Errorf("its basis speaks of the header: %s", finding.Basis)
	}
}
