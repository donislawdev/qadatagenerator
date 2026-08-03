package provider_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// # Why a verdict has three answers and not two
//
// A scheme can have a form a build recognises and does not check. Answering
// that with the same verdict as a broken value puts a correct number and a
// faulty one in the same bucket, with the difference carried only by prose that
// nothing reads programmatically.
//
// The distinction is not decoration. In the mode where a column arrives from
// somebody else's file, collapsing the two means reporting a defect in a value
// that has none, which is the failure this project avoided carefully elsewhere
// and reintroduced one file away.
func TestTheThreeVerdictsAreTellableApart(t *testing.T) {
	cases := []struct {
		name        string
		verdict     provider.Verdict
		valid       bool
		unsupported bool
	}{
		{"valid", provider.Valid(), true, false},
		{"invalid", provider.Invalid("the check digit is wrong"), false, false},
		{"unsupported", provider.Unsupported("this build checks only the short form"), false, true},
	}
	for _, one := range cases {
		t.Run(one.name, func(t *testing.T) {
			if one.verdict.Valid != one.valid {
				t.Errorf("Valid answered %v", one.verdict.Valid)
			}
			if one.verdict.Unsupported != one.unsupported {
				t.Errorf("Unsupported answered %v", one.verdict.Unsupported)
			}
		})
	}
}

// Valid stays false for the third answer, so a caller written before it existed
// keeps behaving the way it did. What it gains is the ability to ask the other
// question rather than being forced to guess from prose.
func TestTheThirdAnswerDoesNotQuietlyBecomeApproval(t *testing.T) {
	verdict := provider.Unsupported("a form this build does not check")
	if verdict.Valid {
		t.Fatal("a value nobody checked was reported as correct")
	}
	if verdict.Reason == "" {
		t.Fatal("the third answer has to name the form, or a report cannot tell a user what was passed over")
	}
}

// A refusal still has to say why. A validator that only says no is one nobody
// can act on, which is the reason this is a struct rather than a boolean.
func TestARefusalCarriesItsReason(t *testing.T) {
	if provider.Invalid("the check digit is 4 and should be 7").Reason == "" {
		t.Fatal("a refusal arrived with nothing to act on")
	}
	if provider.Valid().Reason != "" {
		t.Fatal("a correct value came with a complaint attached")
	}
}
