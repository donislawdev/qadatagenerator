package cli

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// TestQuickWritesEveryValueItDrewInOrder holds that writing as the values come
// loses none of them, and that the last batch, the one no later value pushes
// out, reaches the data channel too.
//
// Five hundred PESELs are six thousand bytes, so the run crosses the batch
// limit eleven times and ends on a batch that is not full.
func TestQuickWritesEveryValueItDrewInOrder(t *testing.T) {
	const count = 500
	got := run("quick", "pesel", "--count", "500", "--seed", "every-value")
	if got.code != int(codeOK) {
		t.Fatalf("expected %d, got %d: %s", codeOK, got.code, got.log)
	}
	want, err := quick.Run(quick.Request{
		Generator: "pesel", Seed: "every-value", Agrees: map[string]string{}, Count: count,
	})
	if err != nil {
		t.Fatalf("the mode produced nothing: %v", err)
	}
	if got.data != strings.Join(want, "\n")+"\n" {
		lines := strings.Count(got.data, "\n")
		t.Errorf("the data channel carried %d lines where %d values were drawn, or carried them "+
			"out of order", lines, count)
	}
}

// writesKept records every write the data channel makes, separately.
type writesKept struct{ writes []string }

func (w *writesKept) Write(p []byte) (int, error) {
	w.writes = append(w.writes, string(p))
	return len(p), nil
}

// TestABatchIsWholeLinesAndAsLargeAsAPipeTakesWhole holds the two promises
// beside batchBytes: no write ends in the middle of a line, and no write is
// larger than the size up to which a pipe takes it whole.
//
// 🔴 The lines are sixty four bytes with their end, so lines fit a batch
// EXACTLY. A batch that stopped one line short of the limit would still be
// whole lines, and only its length tells it apart.
func TestABatchIsWholeLinesAndAsLargeAsAPipeTakesWhole(t *testing.T) {
	if batchBytes > 512 {
		t.Errorf("a batch is %d bytes, more than _POSIX_PIPE_BUF (512), so a kill while the "+
			"reader is slow can cut the last line in half on a system that promises no more", batchBytes)
	}
	kept := &writesKept{}
	out := &streams{data: kept}
	line := strings.Repeat("x", 63)
	lines := 2*batchBytes/64 + 3
	for range lines {
		out.emitBatched(line)
	}
	// A line longer than a batch, between two short ones, is written whole
	// and on its own.
	long := strings.Repeat("y", 2*batchBytes)
	out.emitBatched(long)
	out.emitBatched("z")
	out.flushBatch()

	if len(kept.writes) == 0 || len(kept.writes[0]) != batchBytes {
		t.Fatalf("the first write carried %d bytes where %d whole lines of 64 fill a batch of %d "+
			"exactly", lengthOfFirst(kept.writes), batchBytes/64, batchBytes)
	}
	for _, written := range kept.writes {
		if !strings.HasSuffix(written, "\n") {
			t.Errorf("a write ends in the middle of a line: %q", tail(written))
		}
		if len(written) > batchBytes && written != long+"\n" {
			t.Errorf("a write of %d bytes is larger than a batch and is not one long line", len(written))
		}
	}
	want := strings.Repeat(line+"\n", lines) + long + "\n" + "z\n"
	if strings.Join(kept.writes, "") != want {
		t.Error("the writes put together are not the lines that were emitted, in their order")
	}
}

func lengthOfFirst(writes []string) int {
	if len(writes) == 0 {
		return 0
	}
	return len(writes[0])
}

func tail(text string) string {
	if len(text) <= 20 {
		return text
	}
	return text[len(text)-20:]
}

// refusingAfterCounting refuses every write and counts how many were tried.
type refusingAfterCounting struct{ tried int }

func (r *refusingAfterCounting) Write([]byte) (int, error) {
	r.tried++
	return 0, errors.New("the other end of the pipe is gone")
}

// TestQuickStopsDrawingAtTheFirstWriteThatFails holds the other half of R12:
// once a write has failed, nothing more is drawn or written, because nothing
// after it can arrive.
//
// Until 2026-10-28 the loop went on writing every value into the failure,
// about a second after the reader of a million values left.
//
// 🔴 The count is the largest an int holds, because a run that went on drawing
// after the failure would write nothing more either - the batch is dropped once
// a write has failed - and would end with the same code. Only a count that
// never ends tells the two apart, by the run not ending: the test then fails
// at the timeout of go test rather than passing.
func TestQuickStopsDrawingAtTheFirstWriteThatFails(t *testing.T) {
	var log bytes.Buffer
	refusing := &refusingAfterCounting{}
	code := Run(Environment{
		Args:   []string{"quick", "pesel", "--count", strconv.Itoa(math.MaxInt), "--seed", "the-reader-left"},
		Stdout: refusing,
		Stderr: &log,
	})
	if code != int(codeIO) {
		t.Fatalf("expected %d, got %d: %s", codeIO, code, log.String())
	}
	if refusing.tried != 1 {
		t.Errorf("the data channel was written %d times after its first write failed", refusing.tried)
	}
	if !strings.Contains(log.String(), "standard output") {
		t.Errorf("the log does not say what failed: %q", log.String())
	}
}

// TestADrawnSeedIsSaidAndBringsTheSameValuesBack holds the one line this verb
// writes beside its values: a run that drew its own seed names it, and naming
// it gives the same values again.
func TestADrawnSeedIsSaidAndBringsTheSameValuesBack(t *testing.T) {
	first := run("quick", "pesel", "--count", "3")
	if first.code != int(codeOK) {
		t.Fatalf("expected %d, got %d: %s", codeOK, first.code, first.log)
	}
	_, after, said := strings.Cut(first.log, quickSeedDrawn)
	if !said {
		t.Fatalf("a run that drew its own seed did not say which: %q", first.log)
	}
	drawn := strings.TrimSpace(after)
	again := run("quick", "pesel", "--count", "3", "--seed", drawn)
	if again.data != first.data {
		t.Errorf("the seed the log named gave %q where the run gave %q", again.data, first.data)
	}
	if strings.Contains(again.log, quickSeedDrawn) {
		t.Errorf("a run given its seed still says it drew one: %q", again.log)
	}
}

// 🔴 A run whose engine refused the very first value drew nothing, so there is
// no value for a seed to bring back and the line is not said.
//
// The refusal used is a date of birth a PESEL cannot carry, which the mode's
// Check does not catch today and the engine does (B-345). The first assertion
// says so, so that the day Check starts refusing it this test asks for another
// case instead of quietly passing without reaching the engine.
func TestNoSeedIsSaidWhenTheEngineRefusedTheFirstValue(t *testing.T) {
	asked := quick.Request{
		Generator: "pesel", Locale: quick.DefaultLocale, Seed: "x", Count: 1,
		Agrees: map[string]string{quick.RoleBirthDate: "1700-01-01"},
	}
	if refusal := quick.Check(asked); refusal.Refused() {
		t.Fatalf("Check refuses this request now (%v), so it no longer reaches the engine. "+
			"Choose a refusal only the engine makes", refusal.Kind)
	}
	got := run("quick", "pesel", "--birth-date", "1700-01-01")
	if got.code == int(codeOK) {
		t.Fatalf("a PESEL born in 1700 was produced: %q", got.data)
	}
	if got.data != "" {
		t.Errorf("a refused run wrote data: %q", got.data)
	}
	if strings.Contains(got.log, quickSeedDrawn) {
		t.Errorf("a run that drew nothing named a seed to draw it again: %q", got.log)
	}
}
