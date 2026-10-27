package generate_test

import (
	"errors"
	"runtime"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// A plan holds a word for every parent and nothing for every child.
//
// 🔴 Until 2026-10-27 it held the parent of every child row, so four parents of
// half a million children each cost 16 MB before a value was drawn, and the
// preview on the recipe screen paid it at every key press for the ten rows it
// then showed (outside review, 2026-10-22, DEC-266). What is counted here is
// every byte the plan asks for, rather than what it keeps, because a plan that
// built the long form and threw it away would cost the same at every key press.
//
// The sizes are kept small enough that the long form coming back turns this
// red without asking the machine for much. Measured on 2026-10-27, three runs
// each: 16 045 808 bytes on the commit before, 1 536 after, against a ceiling
// of a megabyte.
func TestAPlanHoldsAWordPerParentRatherThanPerChild(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}
	const parents, each = 4, 500_000
	rec := related(parents, each, each)

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	plan, err := generate.Prepare(rec, values, generate.NeverStops{})
	runtime.ReadMemStats(&after)

	if err != nil {
		t.Fatalf("the relation was refused: %v", err)
	}
	if asked := after.TotalAlloc - before.TotalAlloc; asked > 1<<20 {
		t.Errorf("planning %d children of %d parents asked for %d bytes, which is a word "+
			"for every child rather than for every parent", parents*each, parents, asked)
	}
	child := plan.Entities[1]
	if child.Rows != parents*each {
		t.Fatalf("the plan makes %d children and the relation asks for %d", child.Rows, parents*each)
	}
	for row, parent := range map[int]int{0: 0, each - 1: 0, each: 1, parents*each - 1: parents - 1} {
		if got := child.Ownership.ParentOf(row); got != parent {
			t.Errorf("child %d belongs to parent %d and the plan says %d", row, parent, got)
		}
	}
}

// stopsAfter answers `carry on` a fixed number of times and then ends the plan,
// which is a window asked to close while the plan is still working.
type stopsAfter struct {
	left  int
	asked int
	why   error
}

func (s *stopsAfter) Stopped() error {
	s.asked++
	if s.left > 0 {
		s.left--
		return nil
	}
	return s.why
}

var errClosed = errors.New("the window was asked to close")

// A plan is stopped before a table it has not started, which for a recipe with
// no relation is the only place it can be.
func TestAStopBeforeATableEndsThePlan(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}
	rec := oneEntity(recipe.Field{Name: "first_name", Generator: "first_name"})

	_, err := generate.Prepare(rec, values, &stopsAfter{why: errClosed})

	if !errors.Is(err, errClosed) {
		t.Errorf("the plan was asked to stop before its only table and answered %v", err)
	}
}

// A plan is stopped while it shares children out between the parents.
//
// 🔴 This is the loop that grows with the parents: a draw for every one of them.
// Ten thousand parents and a stop on the tenth question is a stop that has to
// arrive in the middle of it, because the two tables ask only twice between
// them before it starts.
func TestAStopWhileChildrenAreSharedOutEndsThePlan(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}
	halt := &stopsAfter{left: 10, why: errClosed}

	_, err := generate.Prepare(related(10_000, 0, 4), values, halt)

	if !errors.Is(err, errClosed) {
		t.Fatalf("the plan was asked to stop while sharing out and answered %v", err)
	}
	if halt.asked != 11 {
		t.Errorf("the plan carried on for %d questions after the stop", halt.asked-11)
	}
}

// repairParents is how many parents the relations below share their children
// between.
const repairParents = 1000

// repairing is a relation whose count lies a few rounds of repair beyond what
// the draws add up to.
//
// 🔴 The total is taken from a plan of the same relation without a count, whose
// draws are the same ones, rather than guessed. The first version guessed: it
// asked for nearly a million children of parents drawn between nought and a
// thousand, the repair took about a thousand rounds instead of the five hundred
// the guess said, and the stop below arrived on the rounds alone, so taking the
// question out of the drawing of the order went unnoticed. Each round adds at
// most one child to every parent, so three times the parents is at least three
// rounds and, with the draws spread over the whole range, only a few more.
func repairing(t *testing.T, values knownValues) recipe.Recipe {
	t.Helper()
	rec := related(repairParents, 0, 1000)
	drawn, err := generate.Prepare(rec, values, generate.NeverStops{})
	if err != nil {
		t.Fatalf("the relation was refused: %v", err)
	}
	rec.Entities[1].Count = drawn.Entities[1].Rows + 3*repairParents
	rec.Entities[1].HasCount = true
	return rec
}

// A plan is stopped while it repairs a count, which is the loop that grows with
// the width of the range.
//
// The halt lets through every question before the second round of the repair:
// one per table, one per parent for the shares, one per parent for the order the
// repair walks in, and the first round.
func TestAStopWhileACountIsRepairedEndsThePlan(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}
	rec := repairing(t, values)
	before := 2 + repairParents + repairParents + 1
	halt := &stopsAfter{left: before, why: errClosed}

	_, err := generate.Prepare(rec, values, halt)

	if !errors.Is(err, errClosed) {
		t.Fatalf("the plan was asked to stop while repairing a count and answered %v", err)
	}
	if halt.asked != before+1 {
		t.Errorf("the stop was answered %d questions after it was given", halt.asked-before-1)
	}
}

// A plan is stopped while it draws the order the repair walks the parents in,
// which is a draw for every parent and a sort of all of them.
//
// The stop arrives on the eleventh question of that draw. Without the question
// there, the only ones left are a round each of the repair, and a few rounds
// are fewer than eleven, so the plan would finish instead.
func TestAStopWhileTheOrderOfARepairIsDrawnEndsThePlan(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}
	rec := repairing(t, values)
	before := 2 + repairParents + 10
	halt := &stopsAfter{left: before, why: errClosed}

	_, err := generate.Prepare(rec, values, halt)

	if !errors.Is(err, errClosed) {
		t.Fatalf("the plan was asked to stop while drawing the order of a repair and answered %v", err)
	}
	if halt.asked != before+1 {
		t.Errorf("the stop was answered %d questions after it was given", halt.asked-before-1)
	}
}

// The same recipe planned without a stop gets the count it asked for, so the
// stops above are the stops and not a recipe the plan cannot carry out.
func TestTheRepairedCountIsTheOneAskedFor(t *testing.T) {
	values := knownValues{generators: []string{"first_name"}}
	rec := repairing(t, values)
	want := rec.Entities[1].Count

	plan, err := generate.Prepare(rec, values, generate.NeverStops{})

	if err != nil {
		t.Fatalf("the count was refused: %v", err)
	}
	if rows := plan.Entities[1].Rows; rows != want {
		t.Errorf("the plan makes %d children and the recipe asks for %d", rows, want)
	}
	if last := plan.Entities[1].Ownership.ParentOf(want - 1); last == dataset.NoParent {
		t.Error("the last child of the count belongs to no parent")
	}
}
