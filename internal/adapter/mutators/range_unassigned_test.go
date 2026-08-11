package mutators_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

const rangeUnassigned catalog.ID = "L3.range.unassigned"

// 🔴 The damaged value has to leave the block this tool generates in and land
// in one any competent validator refuses, and those are two different blocks.
//
// This is the whole applicability of the entry. Pillar one already generates
// inside a never issued range, because a value that could belong to a real
// person is not something a caller should have to avoid by hand. So a damaged
// value that stayed in the same block would be the value it started from, and
// the manifest would report damage over data nobody touched.
func TestTheMovedValueLeavesTheBlockThisToolGeneratesIn(t *testing.T) {
	entry := entryOf(t, rangeUnassigned)
	for _, id := range schemeNames() {
		if !entry.CoversProvider(id) {
			continue
		}
		scheme := schemesThisBuildHas[id]
		writable, writes := scheme.(provider.UnassignedWritable)
		if !writes {
			t.Fatalf("%s is covered by this damage and cannot be written into a spare block", id)
		}
		t.Run(id, func(t *testing.T) {
			for row := int64(0); row < 25; row++ {
				correct := produce(t, scheme, row)
				outcome := damage(t, rangeUnassigned, mutator.Target{
					Value: correct, Field: identifierField(id),
					Provider: scheme, Params: defaultsOf(entry),
				})
				if !outcome.Applied() {
					t.Fatalf("%s was left alone: %s", correct, outcome.Reason())
				}
				if outcome.Value() == correct {
					t.Fatalf("%s came back unchanged and was reported as damaged", correct)
				}
				// Moving it again into the same block has to be refused, which
				// is the same question asked from the other side: the value now
				// sits where the damage aims.
				again := damage(t, rangeUnassigned, mutator.Target{
					Value: outcome.Value(), Field: identifierField(id),
					Provider: scheme, Params: defaultsOf(entry),
				})
				if again.Applied() {
					t.Errorf("%s was moved into the same block twice, ending as %s",
						outcome.Value(), again.Value())
				}
				_ = writable
			}
		})
	}
}

// The fourth published example this build produces rather than only checks.
func TestTheCatalogExampleIsWhatTheRangeMutatorProduces(t *testing.T) {
	entry := entryOf(t, rangeUnassigned)
	outcome := damage(t, rangeUnassigned, mutator.Target{
		Value:    entry.Example.Before,
		Field:    identifierField("US.ssn"),
		Provider: us.SSN{},
		Params:   defaultsOf(entry),
	})
	if !outcome.Applied() {
		t.Fatalf("the published example was left alone: %s", outcome.Reason())
	}
	if outcome.Value() != entry.Example.After {
		t.Errorf("the specification promises %s -> %s and this build produces %s",
			entry.Example.Before, entry.Example.After, outcome.Value())
	}
}

func TestRangeDamageThatCannotBeDoneIsRefusedWithAReason(t *testing.T) {
	entry := entryOf(t, rangeUnassigned)
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"the scheme has no spare block", mutator.Target{
			Value: "85011412345", Field: identifierField("PL.pesel"),
			Provider: nil, Params: defaultsOf(entry)},
			"no numbering scheme with a block its issuer never gives out"},
		{"no target block was given", mutator.Target{
			Value: "666123456", Field: identifierField("US.ssn"),
			Provider: us.SSN{}, Params: mutator.Parameters{}},
			"nothing says where to move the value"},
		{"the block is not one this scheme has", mutator.Target{
			Value: "666123456", Field: identifierField("US.ssn"),
			Provider: us.SSN{}, Params: mutator.Parameters{"target": "no_such_block"}},
			"is not a block this scheme has"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			outcome := damage(t, rangeUnassigned, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %q", outcome.Value())
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}
