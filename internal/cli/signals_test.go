package cli

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// stopsAfter is a halt that stops a run once it has been asked a given number
// of times. It is how a test reaches the ending a signal reaches, on every
// system and without a process to send anything to.
type stopsAfter struct {
	asked int
	after int
}

func (s *stopsAfter) Stopped() error {
	s.asked++
	if s.asked > s.after {
		return fault.New(fault.Interrupted, stoppedByUser)
	}
	return nil
}

// The promise a signal handler exists to keep. A run stopped part way through
// leaves the output directory as it found it, with no half written table and no
// temporary file for somebody to find next week and wonder about.
func TestARunStoppedPartWayLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "data")
	built := decodedRecipe(t, recipeIn(t, dir, smallRecipe(out)))

	source := values.New(built.Locale, built.Seed)
	plan, err := generate.Prepare(built, source)
	if err != nil {
		t.Fatalf("the plan was refused: %v", err)
	}

	own := declareGenerate(flag.NewFlagSet(commandGenerate, flag.ContinueOnError))
	stopped := produce(built, plan, source, own, &stopsAfter{after: 2})

	if kind := fault.KindOf(stopped); kind != fault.Interrupted {
		t.Fatalf("a stopped run ended as %v rather than as an interrupted one: %v", kind, stopped)
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		// The directory may not exist at all, which is the same answer.
		return
	}
	for _, entry := range entries {
		t.Errorf("the stopped run left %s behind", entry.Name())
	}
}

// An interrupted run ends with the kind that carries the code a shell would
// have reported anyway, rather than looking like a tool failure of ours.
func TestTheWatchEndsTheRunAsAnInterruptedOne(t *testing.T) {
	watcher := watchSignals()
	defer watcher.release()
	if before := watcher.Stopped(); before != nil {
		t.Fatalf("a watch that has seen nothing already stops the run: %v", before)
	}

	deliver(t, watcher, os.Interrupt)
	if kind := fault.KindOf(watcher.Stopped()); kind != fault.Interrupted {
		t.Errorf("the run ended as %v rather than as an interrupted one", kind)
	}
}

// 🔴 The answer has to survive being asked again. The run asks between every
// row and between every table, so a stop that could be taken once would let the
// next question carry on as if nothing had happened.
func TestTheWatchGivesTheSameAnswerEveryTimeItIsAsked(t *testing.T) {
	watcher := watchSignals()
	defer watcher.release()
	deliver(t, watcher, os.Interrupt)

	for asked := 1; asked <= 3; asked++ {
		if kind := fault.KindOf(watcher.Stopped()); kind != fault.Interrupted {
			t.Fatalf("question %d got %v rather than an interrupted run", asked, kind)
		}
	}
}

// A run that ends on its own stops nothing, and releasing the watch is safe
// even though no signal ever arrived.
//
// 🔴 This is the one test here that waits, and the wait is the test rather than
// a delay somebody forgot to remove. What has to be shown is that nothing
// happens, and nothing happening has no moment to observe. A quarter of a
// second is far longer than the microseconds a goroutine needs to answer a
// closed channel, so a build that got this wrong fails on the first branch
// rather than intermittently. Asking straight away instead measured the
// scheduler: the wrong build was reported as correct because the goroutine had
// not run yet.
//
// What this does not prove: that the goroutine behind the watch has actually
// ended. Counting goroutines to find out would be a test of the runtime's
// scheduling rather than of this code.
func TestAWatchReleasedBeforeAnySignalStopsNothing(t *testing.T) {
	watcher := watchSignals()
	watcher.release()

	select {
	case <-watcher.stopping:
		t.Fatal("a run nobody interrupted was reported as stopped")
	case <-time.After(250 * time.Millisecond):
	}
	if stopped := watcher.Stopped(); stopped != nil {
		t.Errorf("a run nobody interrupted was stopped anyway: %v", stopped)
	}
}

// deliver puts a signal into the watch the way the runtime would and waits for
// it to be noticed.
func deliver(t *testing.T, watcher *halt, sent os.Signal) {
	t.Helper()
	watcher.signals <- sent
	select {
	case <-watcher.stopping:
	case <-time.After(5 * time.Second):
		t.Fatal("the watch never noticed a signal that was handed straight to it")
	}
}

// decodedRecipe reads a recipe file the way the verb does.
func decodedRecipe(t *testing.T, path string) recipe.Recipe {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read the recipe: %v", err)
	}
	decoded := recipeyaml.Decode(data)
	if !decoded.Valid() {
		t.Fatalf("the recipe these tests use is not valid: %+v", decoded.Problems)
	}
	return decoded.Recipe
}
