package background

import (
	"slices"
	"testing"
	"time"
)

// looks plays a window forward one look at a time, from the moment its tidy
// was made until until, with allocated saying the running total of allocations
// each look reads. It answers with what the tidy asked for, as "frame at 3m10s"
// and "release at 3m25s", in order.
func looks(until time.Duration, allocated func(at time.Duration) uint64) []string {
	start := time.Date(2026, 11, 12, 18, 0, 0, 0, time.UTC)
	tidy := newTidy(allocated(0), start)
	var asked []string
	for at := tidyLook; at <= until; at += tidyLook {
		switch tidy.look(allocated(at), start.Add(at)) {
		case stepFrame:
			asked = append(asked, "frame at "+at.String())
		case stepRelease:
			asked = append(asked, "release at "+at.String())
		}
	}
	return asked
}

// settled is the first look at least tidySettle after a frame asked for at
// framed, which is when the release comes.
func settled(framed time.Duration) time.Duration {
	wait := (tidySettle + tidyLook - 1) / tidyLook * tidyLook
	return framed + wait
}

// tidiedAt is the frame at framed and its release.
func tidiedAt(framed time.Duration) []string {
	return []string{"frame at " + framed.String(), "release at " + settled(framed).String()}
}

// A window used and then left alone is tidied once: one frame after the quiet,
// the memory back a settling later, and nothing more however long it stays
// alone. The last part is the half that matters most to somebody who leaves the
// window open for a day: a tidy that came back every few minutes would be work
// done for nothing, on a machine that is meant to be idle.
func TestAWindowLeftAloneIsTidiedOnce(t *testing.T) {
	// Used at every look for two minutes, then left alone for an hour.
	used := 2 * time.Minute
	got := looks(time.Hour, func(at time.Duration) uint64 {
		return uint64(min(at, used)/tidyLook) * 2 * tidyBudget
	})
	if want := tidiedAt(used + tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("a window used for %v and then left alone for an hour asked for %v, want %v",
			used, got, want)
	}
}

// Somebody coming back before the quiet is over starts it again, so the frame
// comes a whole quiet after the last thing they did and never while they work.
func TestUseDuringTheQuietStartsItOver(t *testing.T) {
	back := 50 * time.Second
	got := looks(10*time.Minute, func(at time.Duration) uint64 {
		if at >= back {
			return 2 * tidyBudget
		}
		return 0
	})
	if want := tidiedAt(back + tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("used as the window opened and again at %v, the window asked for %v, want %v",
			back, got, want)
	}
}

// The window is built as it opens, which is the most it ever allocates at once,
// so a window nobody touches after that is tidied once as well.
func TestAWindowNobodyTouchesIsTidiedOnceAfterItOpens(t *testing.T) {
	got := looks(time.Hour, func(time.Duration) uint64 { return 0 })
	if want := tidiedAt(tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("a window left alone from the start asked for %v, want %v", got, want)
	}
}

// What a window allocates while nobody uses it is not use, for as long as it
// stays under the budget across the quiet.
func TestAllocationsUnderTheBudgetAreNotUse(t *testing.T) {
	looksInAQuiet := uint64(tidyQuiet/tidyLook) + 1
	got := looks(time.Hour, func(at time.Duration) uint64 {
		return uint64(at/tidyLook) * ((tidyBudget - 1) / looksInAQuiet)
	})
	if want := tidiedAt(tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("a window allocating just under the budget in every quiet asked for %v, want %v",
			got, want)
	}
}

// A trickle is never use, however long the window stands. Counted from the
// last thing that happened rather than over the quiet, a few kilobytes a look
// add up to any budget within hours, and the window would then be tidied again
// and again in a quiet nobody broke.
func TestASlowTrickleNeverAddsUpToUse(t *testing.T) {
	got := looks(10*time.Hour, func(at time.Duration) uint64 {
		return uint64(at/tidyLook) * (tidyBudget / 40)
	})
	if want := tidiedAt(tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("a window allocating a fortieth of the budget at every look for ten hours asked for "+
			"%v, want %v", got, want)
	}
}

// The budget allocated at once is use. The boundary is written down because it
// is the one a changed comparison moves without anything else noticing.
func TestTheBudgetAtOnceIsUse(t *testing.T) {
	once := 10 * time.Minute
	got := looks(time.Hour, func(at time.Duration) uint64 {
		if at >= once {
			return tidyBudget
		}
		return 0
	})
	if want := append(tidiedAt(tidyQuiet), tidiedAt(once+tidyQuiet)...); !slices.Equal(got, want) {
		t.Fatalf("the budget allocated at once at %v asked for %v, want %v", once, got, want)
	}
}

// The frame the tidy asks for allocates, and that is not somebody using the
// window. Counted as use, a second frame would follow the first a quiet later,
// and a third after that, for as long as the window stood open.
func TestTheFramesOwnAllocationsAreNotUse(t *testing.T) {
	afterTheFrame := tidyQuiet + tidyLook
	got := looks(time.Hour, func(at time.Duration) uint64 {
		if at >= afterTheFrame {
			return 100 * tidyBudget
		}
		return 0
	})
	if want := tidiedAt(tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("a window whose only allocations came from the tidy's own frame asked for %v, want %v",
			got, want)
	}
}

// Giving the memory back allocates too, and that is not use either.
//
// 🔴 Found in the real window rather than here. The release collects, and
// what it lets go of allocates on the way out: 0.5 to 0.8 MB in the second of
// the release, two runs of two. The first version took that for somebody
// using the window and tidied every ninety seconds for as long as it stood
// alone.
func TestTheReleasesOwnAllocationsAreNotUse(t *testing.T) {
	releasedAt := settled(tidyQuiet)
	got := looks(time.Hour, func(at time.Duration) uint64 {
		if at > releasedAt {
			return tidyBudget
		}
		return 0
	})
	if want := tidiedAt(tidyQuiet); !slices.Equal(got, want) {
		t.Fatalf("a window whose only allocations after the frame came from the release asked for %v, "+
			"want %v", got, want)
	}
}

// A window in use without a pause, a run writing for an hour for one, is never
// tidied while it works: the memory is what the work is using. Half the budget
// a look is less than a fading caret allocates.
func TestAWindowInUseIsNeverTidied(t *testing.T) {
	got := looks(time.Hour, func(at time.Duration) uint64 {
		return uint64(at/tidyLook) * (tidyBudget / 2)
	})
	if len(got) != 0 {
		t.Fatalf("a window allocating half the budget at every look for an hour asked for %v, want nothing",
			got)
	}
}
