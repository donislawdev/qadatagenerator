package corrupt_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// 🔴 A refusal for want of something says what was missing in words, all of
// it, and never the name of a capability.
//
// The person reading it in Check results has never seen the catalog, and the
// names are not stable. Measured before this: "this run cannot schema.surname,
// which this damage needs" on every file without a column of surnames.
func TestARefusalForWantOfSomethingSaysWhatInWords(t *testing.T) {
	needsThree := entry()
	needsThree.AppliesTo.FieldTypes = []string{"string"}
	needsThree.Requires = []capability.Name{
		capability.CorpusUnicode, capability.SchemaSurname, capability.OutputByteLevelWrite,
	}
	engine := corrupt.New(described(t, needsThree),
		registry{damageID: stated{damageID, mutator.Done("x")}}, noSchemes{},
		capability.NewSet(), seed.New("words"))

	_, defect := engine.Apply(order(), recipe.FieldInfo{Name: "city", Type: "string"}, "Krakow",
		seed.Coordinate{})
	if defect.Applied {
		t.Fatal("damage needing three things this run lacks was applied")
	}
	// In the order of the names, which is sorted so that a seed reads the same
	// twice.
	want := "this damage needs a set of Unicode characters (such as emoji and invisible characters), " +
		"a way to write bytes that are not valid text and a column known to hold surnames, " +
		"which this run does not have"
	if defect.FailureReason != want {
		t.Errorf("the reason is\n  %q\nwant\n  %q", defect.FailureReason, want)
	}
	for _, name := range capability.All() {
		if strings.Contains(defect.FailureReason, string(name)) {
			t.Errorf("the reason names %s: %s", name, defect.FailureReason)
		}
	}
}

// One missing thing is said alone, without a list around it.
func TestARefusalForOneMissingThingSaysJustThat(t *testing.T) {
	needsOne := entry()
	needsOne.Requires = []capability.Name{capability.IdentifierEncodesDate}
	engine := corrupt.New(described(t, needsOne),
		registry{damageID: stated{damageID, mutator.Done("x")}}, noSchemes{},
		capability.NewSet(), seed.New("words"))

	_, defect := engine.Apply(order(), numeric(), "1250", seed.Coordinate{})
	want := "this damage needs an identifier that encodes a date (such as a PESEL), which this " +
		"run does not have"
	if defect.FailureReason != want {
		t.Errorf("the reason is %q, want %q", defect.FailureReason, want)
	}
}
