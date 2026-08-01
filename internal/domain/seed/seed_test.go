package seed_test

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// This is the one test that does not trust the package.
//
// It writes the message out by hand from the layout in the package comment and
// hashes it here. Everything else in this file compares the package with
// itself or with what it produced yesterday, and neither of those can tell a
// correct derivation from a wrong one that never changes. The golden values
// rest on this test.
func TestTheMessageIsBuiltTheWayTheContractSays(t *testing.T) {
	var message []byte
	frame := func(value string) {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		message = append(message, length[:]...)
		message = append(message, value...)
	}
	frame("run-1")
	frame("customer")
	var row [8]byte
	binary.BigEndian.PutUint64(row[:], 42)
	message = append(message, row[:]...)
	frame("email")
	frame("value")
	message = append(message, 0, 0, 0, 0) // round zero

	want := sha256.Sum256(message)
	got := seed.New("run-1").Bytes(seed.Coordinate{
		Entity:  "customer",
		Row:     42,
		Field:   "email",
		Purpose: "value",
	})
	if got != want {
		t.Fatalf("the derivation does not match the documented layout\n got %x\nwant %x", got, want)
	}
}

// Plain concatenation would give one value to two unrelated places. The
// collision would be rare enough to read as coincidence, which is why it has
// to be impossible rather than unlikely.
func TestPlacesThatConcatenateAlikeStayApart(t *testing.T) {
	pairs := []struct {
		name  string
		left  func() [seed.Size]byte
		right func() [seed.Size]byte
	}{
		{
			name:  "entity and field",
			left:  func() [seed.Size]byte { return seed.New("s").Bytes(seed.Coordinate{Entity: "ab", Field: "c"}) },
			right: func() [seed.Size]byte { return seed.New("s").Bytes(seed.Coordinate{Entity: "a", Field: "bc"}) },
		},
		{
			name:  "field and purpose",
			left:  func() [seed.Size]byte { return seed.New("s").Bytes(seed.Coordinate{Field: "ab", Purpose: "c"}) },
			right: func() [seed.Size]byte { return seed.New("s").Bytes(seed.Coordinate{Field: "a", Purpose: "bc"}) },
		},
		{
			name:  "seed and entity",
			left:  func() [seed.Size]byte { return seed.New("ab").Bytes(seed.Coordinate{Entity: "c"}) },
			right: func() [seed.Size]byte { return seed.New("a").Bytes(seed.Coordinate{Entity: "bc"}) },
		},
	}
	for _, pair := range pairs {
		if pair.left() == pair.right() {
			t.Errorf("%s: two different places share a value", pair.name)
		}
	}
}

// The same place answers the same way however many times it is asked, in
// whatever order, from however many goroutines.
func TestOnePlaceAlwaysAnswersTheSame(t *testing.T) {
	source := seed.New("run-1")
	place := seed.Coordinate{Entity: "order", Row: 7, Field: "total", Purpose: "value"}
	want := source.Bytes(place)

	for i := 0; i < 100; i++ {
		source.Bytes(seed.Coordinate{Entity: "noise", Row: int64(i), Field: "x"})
	}
	if got := source.Bytes(place); got != want {
		t.Fatal("the answer moved after other places were asked about")
	}

	results := make([][seed.Size]byte, 64)
	var wait sync.WaitGroup
	for i := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			results[index] = source.Bytes(place)
		}(i)
	}
	wait.Wait()
	for index, got := range results {
		if got != want {
			t.Fatalf("goroutine %d got a different answer, so something is shared between draws", index)
		}
	}
}

// This is the guard for the promise that adding a field leaves the other
// fields where they were. The way to break it is one counter shared between
// draws, and that is the easiest thing to reintroduce by accident.
func TestAskingAboutMorePlacesLeavesTheOthersAlone(t *testing.T) {
	source := seed.New("run-1")
	watched := []seed.Coordinate{
		{Entity: "customer", Row: 0, Field: "first_name", Purpose: "value"},
		{Entity: "customer", Row: 0, Field: "last_name", Purpose: "value"},
		{Entity: "customer", Row: 999, Field: "first_name", Purpose: "value"},
		{Entity: "order", Row: 4812903, Field: "total", Purpose: "corrupt"},
	}

	before := make(map[seed.Coordinate][seed.Size]byte, len(watched))
	for _, place := range watched {
		before[place] = source.Bytes(place)
	}

	// The field somebody just added to the recipe, drawn for every row
	// between the watched ones, in a different order than before.
	for row := int64(1000); row >= 0; row-- {
		source.Bytes(seed.Coordinate{Entity: "customer", Row: row, Field: "phone", Purpose: "value"})
	}
	for row := int64(0); row < 1000; row++ {
		source.Bytes(seed.Coordinate{Entity: "order", Row: row, Field: "discount", Purpose: "value"})
	}

	for _, place := range watched {
		if got := source.Bytes(place); got != before[place] {
			t.Errorf("%+v moved after unrelated places were drawn", place)
		}
	}
}

// Generating a value and deciding whether to corrupt it are two questions
// about the same place. They must not receive the same answer.
func TestPurposeSeparatesTwoQuestionsAboutOnePlace(t *testing.T) {
	source := seed.New("run-1")
	place := seed.Coordinate{Entity: "customer", Row: 3, Field: "national_id"}

	value := place
	value.Purpose = "value"
	corrupt := place
	corrupt.Purpose = "corrupt"

	if source.Bytes(value) == source.Bytes(corrupt) {
		t.Fatal("the value and the decision to corrupt it share an answer")
	}
}

// Input that is odd but legal, which is the kind this will actually receive.
func TestDegenerateCoordinatesAreLegalAndStayApart(t *testing.T) {
	source := seed.New("")
	empty := source.Bytes(seed.Coordinate{})
	if empty == source.Bytes(seed.Coordinate{Row: 1}) {
		t.Error("row zero and row one share an answer")
	}
	if empty == source.Bytes(seed.Coordinate{Row: -1}) {
		t.Error("row zero and row minus one share an answer")
	}
	if source.Bytes(seed.Coordinate{Row: -1}) == source.Bytes(seed.Coordinate{Row: 1}) {
		t.Error("a negative row and a positive one share an answer")
	}
	if empty == seed.New(" ").Bytes(seed.Coordinate{}) {
		t.Error("an empty seed and a blank seed share an answer")
	}
}

func TestIntNStaysInsideItsBound(t *testing.T) {
	source := seed.New("run-1")
	for _, bound := range []int64{1, 2, 7, 256, 1000000} {
		for row := int64(0); row < 500; row++ {
			got := source.IntN(seed.Coordinate{Entity: "e", Row: row, Field: "f"}, bound)
			if got < 0 || got >= bound {
				t.Fatalf("bound %d gave %d", bound, got)
			}
		}
	}
}

// A bound that does not divide evenly makes the low results more likely
// whenever the remainder is simply folded away. The bias is invisible in any
// single value and shows up as data leaning one way, which nobody would report
// as a defect.
//
// The bound below is large enough for the lean to be obvious: folding would
// give the lowest quarter roughly 27 percent of the draws and the highest
// roughly 20, instead of 25 each.
func TestIntNDoesNotLeanTowardsLowValues(t *testing.T) {
	const (
		bound   = int64(5_000_000_000_000_000_000)
		draws   = 20000
		buckets = 4
	)
	source := seed.New("distribution")
	counts := make([]int, buckets)
	for row := int64(0); row < draws; row++ {
		drawn := source.IntN(seed.Coordinate{Entity: "e", Row: row, Field: "f"}, bound)
		counts[drawn/(bound/buckets)]++
	}

	const share = 1.0 / float64(buckets)
	for index, count := range counts {
		got := float64(count) / float64(draws)
		if got < share-0.03 || got > share+0.03 {
			t.Errorf("quarter %d holds %.1f%% of the draws, expected about %.0f%%", index, got*100, share*100)
		}
	}
}

// Without this the redraw is unreachable code that looks like it works.
func TestIntNActuallyRedrawsWhenItHasTo(t *testing.T) {
	const bound = int64(5_000_000_000_000_000_000)
	source := seed.New("distribution")
	redrawn := 0
	for row := int64(0); row < 20000; row++ {
		place := seed.Coordinate{Entity: "e", Row: row, Field: "f"}
		folded := int64(source.Uint64(place) % uint64(bound))
		if source.IntN(place, bound) != folded {
			redrawn++
		}
	}
	if redrawn == 0 {
		t.Fatal("no draw was ever thrown away, so the redraw never runs")
	}
}

func TestIntNRefusesABoundThatIsNotPositive(t *testing.T) {
	for _, bound := range []int64{0, -1} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("bound %d was accepted", bound)
				}
			}()
			seed.New("s").IntN(seed.Coordinate{}, bound)
		}()
	}
}
