package mutators

import (
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// rangeUnassignedID is the catalog entry this implements.
const rangeUnassignedID catalog.ID = "L3.range.unassigned"

// targetParameter is the knob the entry declares: which never issued block the
// value moves into. Its default lives in the catalog.
const targetParameter = "target"

// rangeUnassigned moves an identifier into a block its issuer never gives out.
//
// # Why the scheme decides what that means and this does not
//
// A numbering scheme with unissued blocks usually has more than one, and they
// are not interchangeable. This tool already generates inside such a block
// wherever one exists, because a value that could belong to a real person is
// not something a caller should have to remember to avoid. So a mutator that
// asked whether a value sits in a reserved range would get yes for every value
// this tool produces and would never apply to anything.
//
// The question that means something is whether the value already sits in the
// particular block the damage aims at, and only the scheme knows how many
// blocks it has or how to write one. That is why the refusal below is the
// scheme's own sentence rather than a judgement made here: the two ranges are
// two different lines of defence, and telling them apart is what makes this
// applicable at all.
type rangeUnassigned struct{}

// ID is the catalog identifier.
func (rangeUnassigned) ID() catalog.ID { return rangeUnassignedID }

// Damage moves the value into the named block.
func (rangeUnassigned) Damage(target mutator.Target) mutator.Outcome {
	writable, writes := target.Provider.(provider.UnassignedWritable)
	if !writes {
		return mutator.Skipped(
			"this column follows no numbering scheme with a block its issuer never gives out")
	}
	where, given := target.Params.Text(targetParameter)
	if !given {
		return mutator.Skipped("no target block was given, so nothing says where to move the value")
	}
	moved, err := writable.WithUnassigned(target.Value, where)
	if err != nil {
		// The scheme's own words, passed through rather than reworded. It knows
		// which blocks it has and whether this value already sits in the one
		// asked for, and both of those are sentences about the data that a
		// person reading the manifest can act on. Rewording them here would put
		// a second description of somebody else's rules in this file.
		return mutator.Skipped(err.Error())
	}
	// 🔴 There is deliberately no check here for a value that did not move, and
	// the reason is worth keeping. The contract makes that refusal the scheme's
	// job and says so: WithUnassigned refuses when the value already sits in the
	// target block, and that refusal IS the applicability answer. A second check
	// here could never run against a conforming scheme, and a condition nobody
	// can break is a condition nobody can prove.
	//
	// The gender mutator keeps the equivalent check because nothing in the
	// contract promises the same thing there, so its two halves can disagree.
	return mutator.Done(moved)
}
