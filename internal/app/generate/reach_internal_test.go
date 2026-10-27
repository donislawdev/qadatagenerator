package generate

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// # Why these are inside the package while the rest of the plan is tested from
// outside it
//
// Everything this guard refuses is something that costs tens of gigabytes, or
// since the plan holds a word per parent (2026-10-27) ends in a panic of the
// arithmetic, if it is not refused. A test driving it through Prepare would
// therefore be a test that allocates all of it the moment somebody breaks the
// guard, and the tool that breaks guards on purpose runs here every time
// anybody changes this package. A mutation is meant to turn a test red, not to
// ask the machine for 32 GB.
//
// withinReach is arithmetic and nothing else, so asked directly it answers the
// same question for nothing. The way through Prepare is covered next door with
// numbers small enough to survive being allowed.

func relation(low, high int) (recipe.Entity, recipe.Parent) {
	return recipe.Entity{Name: "order"}, recipe.Parent{Entity: "customer", Min: low, Max: high}
}

// A range wider than this build can plan for is refused before anything is
// drawn from it.
//
// 🔴 Measured with the real binary on 2026-09-02, before this existed: the
// widest range an int can hold reached the seed as a bound of minus nine
// million million million, because the width was worked out in the arithmetic
// that had just overflowed, and the run ended with the message that begins
// internal defect and asks for a bug report. A recipe somebody wrote was
// answered by this tool blaming itself.
func TestARangeWiderThanThisBuildCanPlanForIsRefused(t *testing.T) {
	entity, parent := relation(0, int(^uint(0)>>1))

	err := withinReach(entity, parent, 3)

	if err == nil {
		t.Fatal("the widest range this machine can count was accepted, so the plan was " +
			"about to draw from it")
	}
	if fault.KindOf(err) != fault.Data {
		t.Errorf("the refusal carries kind %v rather than the one meaning the recipe reads "+
			"correctly and cannot be carried out, and the two leave by different exit codes",
			fault.KindOf(err))
	}
	// 🔴 Until 2026-10-27 this asked for the cost in memory, one machine word
	// for every child row, because that was what the ceiling stood on. The plan
	// holds a word per parent now and the ceiling stands on its arithmetic, so
	// what the reader is owed is which tables it applies to: a person who reads
	// only a number thinks every count in the recipe is capped, and none is.
	if !strings.Contains(err.Error(), "plans for a table that belongs to another") {
		t.Errorf("the refusal does not say which tables the ceiling applies to, so it reads "+
			"as a cap on every count: %v", err)
	}
}

// More parents than this build can share children out between is refused even
// when the range asks for no children at all.
//
// 🔴 This is the case a single check written as a product cannot see. A share
// is kept for every parent while the draw goes on, so nought to nought
// children of five thousand million parents allocates nothing for the children
// and one word for each parent. Multiplying by a range of nought gives nought.
func TestMoreParentsThanThisBuildCanShareOutBetweenIsRefused(t *testing.T) {
	entity, parent := relation(0, 0)

	err := withinReach(entity, parent, MaxChildRows+1)

	if err == nil {
		t.Fatal("a parent table larger than this build can share children out between " +
			"was accepted, and a share is held for every one of them")
	}
	if !strings.Contains(err.Error(), "parents") {
		t.Errorf("the refusal does not say that the number of parents is what is too "+
			"large, so somebody would go looking at the range: %v", err)
	}
}

// A reach too large to count is compared by dividing rather than by
// multiplying, because the product is the overflow being refused.
//
// 🔴 The number of parents here is exactly the ceiling and not one above it,
// and that is the whole of what makes this test about the rule it names. Its
// first version handed in the largest countable number of parents, which the
// rule ABOVE this one refuses on its own, so the test passed while the rule it
// was written for never ran. The mutation that forms the product said so, and
// nothing else would have.
//
// With these two numbers the product wraps to minus four thousand million,
// which is smaller than the ceiling, so a check that multiplied would let
// through the one recipe it exists to refuse.
func TestTheReachIsComparedWithoutFormingTheProduct(t *testing.T) {
	const widest = int(^uint(0) >> 1)
	entity, parent := relation(0, widest)

	if err := withinReach(entity, parent, MaxChildRows); err == nil {
		t.Error("the widest countable range between as many parents as this build allows " +
			"was accepted, so the comparison wrapped instead of answering")
	}
}

// An ordinary relation is not refused, which is the failure mode of a ceiling
// put in the wrong place.
func TestTheReachOfAnOrdinaryRelationIsNotRefused(t *testing.T) {
	entity, parent := relation(0, 4)

	if err := withinReach(entity, parent, 3); err != nil {
		t.Errorf("three parents of up to four children each were refused: %v", err)
	}
}
