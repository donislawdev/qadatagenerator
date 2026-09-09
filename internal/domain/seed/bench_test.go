package seed_test

import (
	"strconv"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// The measurements this file exists for are the ones DEC-65 left open.
//
// That decision chose to derive a value from its address rather than from a
// stream, wrote down that the hash costs more than a stream read, and left the
// way out described but unmeasured: hash once per row and run a cheap derived
// stream inside the row. Nobody has ever taken the number, so the trade has
// been carried on the strength of the reasoning alone.
//
// What is reported here is a cost per draw and a cost per row. Neither is a
// verdict on the trade. A verdict needs the streaming mode to exist, because
// the question is what this costs against everything else a row does, and
// today a row does nothing else.

var sink [seed.Size]byte

func BenchmarkOneDraw(b *testing.B) {
	source := seed.New("release-2026-08")
	at := seed.Coordinate{Entity: "customer", Row: 0, Field: "pesel", Purpose: "value"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		at.Row = int64(i)
		sink = source.Bytes(at)
	}
}

var sinkNumber int64

func BenchmarkOneBoundedDraw(b *testing.B) {
	source := seed.New("release-2026-08")
	at := seed.Coordinate{Entity: "customer", Row: 0, Field: "pesel", Purpose: "value"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		at.Row = int64(i)
		sinkNumber = source.IntN(at, 1000)
	}
}

// A row of ten fields, which is the shape the crossing example uses. This is
// the number that matters for the streaming mode, because it is what gets
// multiplied by tens of millions.
func BenchmarkRowOfTenFields(b *testing.B) {
	source := seed.New("release-2026-08")
	fields := []string{
		"id", "first_name", "last_name", "gender", "birth_date",
		"pesel", "email", "phone", "city", "postal_code",
	}
	b.ReportAllocs()
	b.ResetTimer()
	for row := 0; b.Loop(); row++ {
		for _, field := range fields {
			sink = source.Bytes(seed.Coordinate{
				Entity: "customer", Row: int64(row), Field: field, Purpose: "value",
			})
		}
	}
}

// allocationCeiling is a ratchet. It may only ever be lowered.
//
// It stands at nought, which is where a pure computation belongs. It did not
// start there: the first measurement on 2026-08-02 found nine allocations and
// two hundred bytes per derived value, because the message was written into
// the hash through an io.Writer and a call through an interface sends every
// small array to the heap. Assembling the message first and hashing it once
// removed all nine and made the whole thing more than twice as fast, without
// moving a single byte of the result.
//
// # Why an allocation count is asserted while a duration is only reported
//
// An allocation count is the same on every machine and in every run, so it is
// a fact and a test can hold it. A duration on a shared runner is a sample,
// and a build gated on one fails for reasons that have nothing to do with the
// code, which is how a guard earns its way to being switched off.
const allocationCeiling = 0

// This is the hottest path in the product. In the streaming mode it runs once
// per field per row, so nine allocations become hundreds of millions.
func TestDerivingAValueDoesNotAllocateMoreThanItDid(t *testing.T) {
	source := seed.New("release-2026-08")
	at := seed.Coordinate{Entity: "customer", Row: 7, Field: "pesel", Purpose: "value"}

	for _, test := range []struct {
		name string
		call func()
	}{
		{"one draw", func() { sink = source.Bytes(at) }},
		{"one bounded draw", func() { sinkNumber = source.IntN(at, 1000) }},
	} {
		got := testing.AllocsPerRun(1000, test.call)
		if got > allocationCeiling {
			t.Errorf("%s allocates %v times, above the ceiling of %d. The ceiling may be lowered, never raised",
				test.name, got, allocationCeiling)
		}
		t.Logf("%s allocates %v times", test.name, got)
	}
}

// BenchmarkDrawBySeedLength is what settled the length a surface will take.
//
// The message is assembled in a fixed array on the stack, and a seed long
// enough to overflow it pushes the whole derivation onto the heap - once per
// VALUE, not once per run. So the length of a seed is not only a question of
// what a person can reasonably type: past a point it changes the cost of every
// value the tool produces. This measures where that point is.
func BenchmarkDrawBySeedLength(b *testing.B) {
	at := seed.Coordinate{Entity: "customer", Row: 0, Field: "pesel", Purpose: "value"}
	for _, length := range []int{32, 128, 192, 256, 1024} {
		grain := make([]byte, length)
		for i := range grain {
			grain[i] = 'a'
		}
		source := seed.New(string(grain))
		b.Run(strconv.Itoa(length), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; b.Loop(); i++ {
				at.Row = int64(i)
				sink = source.Bytes(at)
			}
		})
	}
}
