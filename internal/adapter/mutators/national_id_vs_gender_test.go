package mutators_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

const nationalIDVsGender catalog.ID = "L4.coherence.national_id_vs_gender"

// 🔴 This is the property that puts this damage at the layer of business rules
// rather than the one below, and it is the opposite of what a checksum mutator
// has to prove.
//
// The value that comes out has to be CORRECT. Its own scheme must accept it,
// because the defect is not in the value at all: it is in the pair of the value
// and the column beside it. A build that broke the number here would produce a
// defect an application catches immediately, which is the one this entry says
// applications almost never catch.
func TestTheGenderFlippedIdentifierIsStillCorrectOnItsOwn(t *testing.T) {
	scheme := pl.PESEL{}
	for row := int64(0); row < 25; row++ {
		correct := produce(t, scheme, row)
		before, readable := scheme.EncodedGender(correct)
		if !readable {
			t.Fatalf("the generator produced %s, which carries no gender", correct)
		}

		outcome := damage(t, nationalIDVsGender, mutator.Target{
			Value: correct, Field: identifierField("PL.pesel"), Provider: scheme,
		})
		if !outcome.Applied() {
			t.Fatalf("%s was left alone: %s", correct, outcome.Reason())
		}
		if verdict := scheme.Validate(outcome.Value()); !verdict.Valid {
			t.Errorf("%s became %s, which its own scheme rejects: %s",
				correct, outcome.Value(), verdict.Reason)
		}
		after, readable := scheme.EncodedGender(outcome.Value())
		if !readable || after == before {
			t.Errorf("%s became %s, which still reads as %s", correct, outcome.Value(), after)
		}
	}
}

// The second published example this build produces rather than only checks.
func TestTheCatalogExampleIsWhatTheGenderMutatorProduces(t *testing.T) {
	entry := entryOf(t, nationalIDVsGender)
	outcome := damage(t, nationalIDVsGender, mutator.Target{
		Value:    entry.Example.Before,
		Field:    identifierField("PL.pesel"),
		Provider: pl.PESEL{},
	})
	if !outcome.Applied() {
		t.Fatalf("the published example was left alone: %s", outcome.Reason())
	}
	if outcome.Value() != entry.Example.After {
		t.Errorf("the specification promises %s -> %s and this build produces %s",
			entry.Example.Before, entry.Example.After, outcome.Value())
	}
}

// 🔴 The entry has to require a declared relation, because this mutator cannot
// check for one and its promise depends on it.
//
// What it produces is a correct identifier encoding the other gender. That is a
// defect only if some column beside it says otherwise. In a dataset with no
// gender column, or one no consistency rule binds to the identifier, the damage
// would apply and the manifest would describe an incoherence that is not in the
// data. The mutator sees one value and cannot tell, so the applicability has to
// be declared where the engine can read it before calling.
func TestGenderCoherenceDamageRequiresARelationToDisagreeWith(t *testing.T) {
	entry := entryOf(t, nationalIDVsGender)
	for _, needed := range entry.Requires {
		if needed == capability.SchemaFieldRelations {
			return
		}
	}
	t.Errorf("the entry requires %v, so a recipe with no gender column would get this "+
		"applied and be told its rows are incoherent", entry.Requires)
}

// A column whose scheme cannot say anything about gender is refused in words
// rather than damaged. The tax number is the case that actually arises: it is an
// identifier, this build carries it, and it encodes nothing about its holder.
func TestGenderDamageWithoutASchemeThatEncodesOneIsRefused(t *testing.T) {
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"the column follows no scheme at all", mutator.Target{
			Value: "85011412345", Field: identifierField("")},
			"follows no numbering scheme that encodes a gender"},
		{"the scheme encodes no gender", mutator.Target{
			Value: "1234563218", Field: identifierField("PL.nip"), Provider: pl.NIP{}},
			"follows no numbering scheme that encodes a gender"},
		{"no gender can be read out of the value", mutator.Target{
			Value: "85011", Field: identifierField("PL.pesel"), Provider: pl.PESEL{}},
			"no gender can be read out of this value"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			outcome := damage(t, nationalIDVsGender, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %q", outcome.Value())
			}
			if outcome.Reason() == "" {
				t.Fatal("nothing happened and nothing says why, which is the silence this forbids")
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}

// A scheme reading a gender it cannot write is refused rather than trusted.
//
// The two halves of the contract are separate methods and a provider can
// implement one badly. Answering unknown and then being asked to write it is the
// case that would otherwise reach the arithmetic with nothing to do.
func TestAValueCarryingNoGenderIsNotTurnedRound(t *testing.T) {
	outcome := damage(t, nationalIDVsGender, mutator.Target{
		Value: "85011412345", Field: identifierField("PL.pesel"), Provider: genderless{},
	})
	if outcome.Applied() {
		t.Fatalf("a value carrying no gender was rebuilt as %q", outcome.Value())
	}
	if !strings.Contains(outcome.Reason(), "carries no gender to turn round") {
		t.Errorf("the reason does not name the case: %s", outcome.Reason())
	}
}

// genderless reads every value as carrying no gender, which the contract allows
// a scheme to answer and this build's own provider never does.
type genderless struct{ pl.PESEL }

func (genderless) EncodedGender(string) (provider.Gender, bool) {
	return provider.Unknown, true
}

// A scheme whose two halves disagree is refused rather than recorded as damage.
//
// 🔴 This is what keeps the guard against a value that did not move from being a
// condition nobody can break. A conforming provider never triggers it, so
// without a scheme that misbehaves on purpose it would sit there unprovable,
// which is the dead weight this project has found twice before.
func TestASchemeThatRebuildsNothingIsRefused(t *testing.T) {
	outcome := damage(t, nationalIDVsGender, mutator.Target{
		Value: "85011412345", Field: identifierField("PL.pesel"), Provider: unmoved{},
	})
	if outcome.Applied() {
		t.Fatalf("a value that did not move was recorded as damaged: %q", outcome.Value())
	}
	if !strings.Contains(outcome.Reason(), "changed nothing") {
		t.Errorf("the reason does not name the case: %s", outcome.Reason())
	}
}

// unmoved reports a gender and then rebuilds the value unchanged, which is a
// provider whose two halves disagree with each other.
type unmoved struct{ pl.PESEL }

func (unmoved) EncodedGender(string) (provider.Gender, bool) {
	return provider.Female, true
}

func (unmoved) WithGender(value string, _ provider.Gender) (string, error) {
	return value, nil
}
