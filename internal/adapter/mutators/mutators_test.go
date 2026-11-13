package mutators_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// This is the half of the closure that is a defect rather than progress.
//
// A mutator running without an entry has no declared expectation, so a manifest
// naming it tells a test nothing about what should happen next, and the
// published specification and the behaviour have parted company. The other
// direction, an entry with nobody implementing it, is the ordinary state of a
// catalog that grows faster than the engine.
func TestEveryImplementedMutatorHasACatalogEntry(t *testing.T) {
	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := catalog.Undescribed(loaded, mutators.IDs()); len(got) != 0 {
		t.Errorf("these have behaviour and no entry describing them: %v", got)
	}
	if len(mutators.IDs()) == 0 {
		t.Fatal("nothing is implemented, so this check read nothing")
	}
}

// implementedCount is how many kinds of damage this build carries out.
//
// 🔴 It is the half of the closure that can exist while the catalog is ahead of
// the engine. The architecture asks for the two sets to be equal, and they are
// not: fifty two kinds of damage are described and this build carries thirteen
// of them. A test declaring equality with a list of thirty nine exceptions would be
// green against a build with no mechanism at all, which is the shape that once
// left this project with three guards defending nothing.
//
// The direction that can fail on its own is downwards. Damage that quietly
// stopped being carried out would leave every other test green: the engine
// answers for what nobody implemented with an entry saying so, which is correct
// behaviour and indistinguishable from a build that lost a mutator.
//
// 🔴 # Why it fails upwards as well, since 2026-09-07
//
// It was written as a floor and compared with less than, which made it a
// ratchet somebody had to remember to raise. The eighth mutator arrived without
// that happening, and the floor of seven then accepted a build of seven, so the
// guard was disabled by a change that never touched it. Nothing said so for the
// rest of the day: every test stayed green and only a full mutation run found
// it, by removing a mutator this was supposed to notice.
//
// A number that fails in both directions cannot be left behind. Raising it is
// one line and it is part of adding damage, which is exactly when somebody is
// looking at this file anyway.
const implementedCount = 13

func TestThisBuildHasNotLostAMutator(t *testing.T) {
	got := len(mutators.IDs())
	if got < implementedCount {
		t.Errorf("this build carries %d kinds of damage and carried %d before: %v",
			got, implementedCount, mutators.IDs())
	}
	if got > implementedCount {
		t.Errorf("this build carries %d kinds of damage and %d are written down here, so "+
			"raise implementedCount to %d as part of adding one: %v",
			got, implementedCount, got, mutators.IDs())
	}
}

// A mutator filed under somebody else's identifier would be handed the wrong
// entry, and the manifest would carry the wrong expectation for the damage that
// actually happened.
func TestEveryMutatorIsFiledUnderItsOwnIdentifier(t *testing.T) {
	for _, id := range mutators.IDs() {
		found, ok := mutators.Lookup(id)
		if !ok {
			t.Errorf("%s is listed and cannot be looked up", id)
			continue
		}
		if found.ID() != id {
			t.Errorf("%s is filed under %s", found.ID(), id)
		}
	}
}

func TestDamageNobodyImplementedIsAnsweredAsMissing(t *testing.T) {
	if _, ok := mutators.Lookup("L5.invented.by_nobody"); ok {
		t.Error("an identifier nobody implemented was answered as implemented")
	}
}

func numeric() recipe.FieldInfo {
	return recipe.FieldInfo{Name: "amount", Type: "integer"}
}

func TestTextGoesWhereTheSchemaWantsANumber(t *testing.T) {
	damaged := damage(t, "L2.type.string_in_numeric", mutator.Target{
		Value:  "1250",
		Field:  numeric(),
		Params: mutator.Parameters{"value": "n/a"},
	})
	if !damaged.Applied() {
		t.Fatalf("nothing happened: %s", damaged.Reason())
	}
	if damaged.Value() != "n/a" {
		t.Errorf("the value became %q", damaged.Value())
	}
}

// Every one of these is damage that would have been recorded as done while
// leaving the data alone, which is a manifest sending a test to look for a
// difference that is not there.
func TestDamageThatWouldChangeNothingIsRefusedWithAReason(t *testing.T) {
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"no replacement given", mutator.Target{
			Value: "1250", Field: numeric(), Params: mutator.Parameters{}},
			"no value was given"},
		{"replacement is empty", mutator.Target{
			Value: "1250", Field: numeric(), Params: mutator.Parameters{"value": ""}},
			"a different defect with an entry of its own"},
		{"replacement is already there", mutator.Target{
			Value: "n/a", Field: numeric(), Params: mutator.Parameters{"value": "n/a"}},
			"would change nothing"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			outcome := damage(t, "L2.type.string_in_numeric", one.target)
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

// The default belongs to the catalog entry, not to the code. A copy here would
// be a published value with a second home, and the two would part company the
// first time one of them was edited.
func TestTheReplacementDefaultComesFromTheCatalogAndNotFromTheCode(t *testing.T) {
	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entry, ok := loaded.Lookup("L2.type.string_in_numeric")
	if !ok {
		t.Fatal("the entry this mutator implements is not in the catalog")
	}
	if len(entry.Parameters) != 1 || entry.Parameters[0].Name != "value" {
		t.Fatalf("the entry declares %v, and this mutator reads one named value", entry.Parameters)
	}
	written, ok := entry.Parameters[0].Default.(string)
	if !ok || written == "" {
		t.Fatalf("the declared default is %v, and there is nothing to write with it", entry.Parameters[0].Default)
	}

	outcome := damage(t, "L2.type.string_in_numeric", mutator.Target{
		Value: "1250", Field: numeric(), Params: mutator.Parameters{"value": written},
	})
	if !outcome.Applied() || outcome.Value() != written {
		t.Errorf("the catalog default %q did not reach the data: applied=%v value=%q",
			written, outcome.Applied(), outcome.Value())
	}
}

func damage(t *testing.T, id catalog.ID, target mutator.Target) mutator.Outcome {
	t.Helper()
	found, ok := mutators.Lookup(id)
	if !ok {
		t.Fatalf("%s is not implemented in this build", id)
	}
	return found.Damage(target)
}
