package generate

import "testing"

// A walk goes round the end of the domain to the one place still free, keeps
// what it crossed as jumps that stay true, and says so when nothing is left.
func TestAWalkGoesRoundToTheLastFreePlace(t *testing.T) {
	free := map[int64]bool{3: true, 8: true}
	walker := newWalker(10)
	isFree := func(at int64) bool { return free[at] }

	if place, found := walker.next(5, isFree); !found || place != 8 {
		t.Fatalf("from 5 the walk found %d (%v), and the first free place on is 8", place, found)
	}
	delete(free, 8)
	if place, found := walker.next(5, isFree); !found || place != 3 {
		t.Fatalf("from 5 the walk found %d (%v), and the only free place left is 3, round the end", place, found)
	}
	delete(free, 3)
	if place, found := walker.next(0, isFree); found {
		t.Fatalf("the walk found %d in a domain with nothing free", place)
	}
}
