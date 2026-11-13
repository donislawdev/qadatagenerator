package recipe_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// 🔴 A column of surnames answers for names and for surnames, and so does a
// column of whole names, a column of given names for names only, and a column
// holding no names for neither.
//
// The engine asks the column rather than the run for both, so this is the
// whole of what keeps O'Brien off a column of first names and off a column of
// towns. Both directions are asked, because a column answering too much lands
// the damage where its expectation does not hold, and one answering too little
// leaves it refused where it does.
func TestAColumnSaysWhichPartOfANameItHolds(t *testing.T) {
	for _, given := range []struct {
		part        recipe.NamePart
		personName  bool
		surnameOnly bool
	}{
		{recipe.Surname, true, true},
		{recipe.FullName, true, true},
		{recipe.GivenName, true, false},
		{"", false, false},
	} {
		supplied := capability.NewSet(recipe.CapabilitiesOfField(
			recipe.FieldInfo{Name: "who", Type: "string", NamePart: given.part})...)
		if got := len(supplied.Missing([]capability.Name{capability.SchemaPersonName})) == 0; got != given.personName {
			t.Errorf("a column holding %q answers %v for a person's name, want %v",
				given.part, got, given.personName)
		}
		if got := len(supplied.Missing([]capability.Name{capability.SchemaSurname})) == 0; got != given.surnameOnly {
			t.Errorf("a column holding %q answers %v for surnames, want %v",
				given.part, got, given.surnameOnly)
		}
	}
}

// Both are a property of the column, so a run carrying them because one column
// does can never let the damage land on another.
func TestThePartsOfANameAreAskedOfTheColumn(t *testing.T) {
	for _, name := range []capability.Name{capability.SchemaPersonName, capability.SchemaSurname} {
		if scope, known := capability.ScopeOf(name); !known || scope != capability.Field {
			t.Errorf("%s is of scope %q, known %v, and has to be a property of the column",
				name, scope, known)
		}
	}
}
