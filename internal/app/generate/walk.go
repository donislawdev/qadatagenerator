package generate

import (
	"strconv"
	"strings"
)

// walker finds a place in a known domain that no earlier row took.
//
// # Why a walk, and why it remembers
//
// A unique column drawn at random runs out of luck as it fills: with nine in
// ten values taken, twenty draws in a row land on taken ones about one row in
// eight. Until 2026-10-26 the run gave up there and said the field had fewer
// values than the rows asked for, which it did not know. Where the domain is
// known the run now walks from a drawn place to the next free one, which is a
// value of the same domain chosen by the seed, so a dataset that fits its
// domain is produced whole and the same every time.
//
// A plain walk grows faster than the table, because every row crosses the same
// long run of taken places again. The walker remembers each run it crossed as a
// jump straight to its end. Measured on 2026-10-26 on the built binary, filling
// every value of a unique number, median of three runs after a discarded first:
// a million values took 16.7 s without the jumps and 3.5 to 4.5 s with them, a
// hundred thousand 0.73 s and 0.40 s. That stays true because
// nothing is ever freed behind a jump: what a row releases when it is built
// again is only ever the value it claimed itself, and a jump never leads over
// a place claimed by the row that made it.
type walker struct {
	size int64
	jump map[int64]int64
}

func newWalker(size int64) *walker {
	return &walker{size: size, jump: make(map[int64]int64)}
}

// next returns the first free place at or after start, going round, and
// whether there is one.
func (w *walker) next(start int64, free func(int64) bool) (int64, bool) {
	var passed []int64
	at, travelled := start, int64(0)
	for travelled < w.size {
		if to, known := w.jump[at]; known {
			passed = append(passed, at)
			travelled += (to - at + w.size) % w.size
			at = to
			continue
		}
		if free(at) {
			for _, from := range passed {
				w.jump[from] = at
			}
			return at, true
		}
		passed = append(passed, at)
		at = (at + 1) % w.size
		travelled++
	}
	return 0, false
}

// keyOf writes a tuple of values as one text that no other tuple writes.
//
// Each value goes in with its length in front of it, so that a and bc and ab
// and c are two keys rather than one, whatever the values carry.
func keyOf(values []string) string {
	var joined strings.Builder
	for _, value := range values {
		joined.WriteString(strconv.Itoa(len(value)))
		joined.WriteByte(':')
		joined.WriteString(value)
	}
	return joined.String()
}
