// Package seed turns the address of a value into randomness for it.
//
// Nothing here reads a stream. The value at a place is a function of that
// place alone, so generating the same dataset twice gives the same bytes, and
// the order in which values are produced stops mattering. Four things follow
// from that, and each is worth the choice on its own: refactoring cannot move
// anybody's data, adding a field leaves the other fields where they were,
// parallel work needs no shared state, and the choice of which rows to corrupt
// is reproducible by the same mechanism as the rows themselves.
//
// The price is one hash per value rather than one read from a stream. It is
// paid knowingly and there is a documented way out if measurement ever asks
// for it: hash once per row and run a cheap derived stream inside the row.
// That trade keeps three of the four properties and gives up part of the
// second, so it waits for a measurement rather than a feeling.
//
// # What is frozen
//
// The message layout below and the choice of SHA-256 are a contract with every
// dataset ever generated. Changing either changes every value the tool has
// ever produced, so it is a major version and an entry in the changelog, never
// a quiet improvement.
//
// The message is built from length framed fields:
//
//	frame(s)  = uint64 big endian length of s, then the bytes of s
//	message   = frame(seed) frame(entity) uint64(row) frame(field)
//	            frame(purpose) uint32(round)
//
// The lengths are what makes the message unambiguous. Plain concatenation
// would give the same bytes for the entity "ab" with field "c" and the entity
// "a" with field "bc", so two unrelated places would share a value. That kind
// of collision is rare enough to read as coincidence, which is exactly why it
// has to be impossible rather than unlikely.
//
// No version of the tool, the catalog or any format enters the message. An
// upgrade therefore moves nobody's bytes. Byte stability is a promise held
// within a major version and guarded by the golden values in testdata.
package seed

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

// Size is the length of a derived digest in bytes.
const Size = sha256.Size

// Coordinate is the address of one value: which entity, which row, which
// field, and what the value is for.
//
// Purpose separates two draws that share the other three. Generating a value
// and deciding whether to corrupt it are different questions about the same
// place, and they must not receive the same answer.
type Coordinate struct {
	Entity  string
	Row     int64
	Field   string
	Purpose string
}

// Source derives values for one run.
//
// It carries no mutable state, which is the property the whole package rests
// on. A counter shared between draws would tie the value at one place to how
// many draws happened before it, and that is the model this package exists to
// avoid. It is also the easiest thing to reintroduce by accident, so a test
// asks about it directly.
type Source struct {
	seed string
}

// New returns a source for the given seed.
//
// The seed is a string rather than a number on purpose. A recipe is YAML, and
// YAML reads 010 as the number 8 and 1.0 as a floating point value, so a seed
// written as a number can quietly become a different seed than the one on the
// page.
func New(seed string) Source {
	return Source{seed: seed}
}

// Bytes returns the digest for a coordinate.
func (s Source) Bytes(c Coordinate) [Size]byte {
	return s.derive(c, 0)
}

// Uint64 returns an unsigned number for a coordinate.
func (s Source) Uint64(c Coordinate) uint64 {
	digest := s.Bytes(c)
	return binary.BigEndian.Uint64(digest[:8])
}

// IntN returns a number in the half open range from zero to bound.
//
// Values are drawn again rather than folded with a remainder, because folding
// makes the low results more likely than the high ones whenever the bound does
// not divide evenly. The bias is small and would never show up as a failure,
// only as data that leans one way. Redrawing uses the next round of the same
// coordinate, so the answer stays a function of the place.
//
// It panics when the bound is not positive. That is a defect in the caller
// rather than bad input from a user, and it surfaces as a tool failure.
func (s Source) IntN(c Coordinate, bound int64) int64 {
	if bound <= 0 {
		panic(fmt.Sprintf("seed: IntN needs a bound above zero, got %d", bound))
	}
	limit := uint64(bound)
	// Draws below this are the ones that would tilt the result, so they are
	// the ones thrown away.
	threshold := (0 - limit) % limit
	for round := uint32(0); ; round++ {
		digest := s.derive(c, round)
		drawn := binary.BigEndian.Uint64(digest[:8])
		if drawn >= threshold {
			return int64(drawn % limit)
		}
	}
}

// derive builds the message described in the package comment and hashes it.
func (s Source) derive(c Coordinate, round uint32) [Size]byte {
	hash := sha256.New()
	writeFramed(hash, s.seed)
	writeFramed(hash, c.Entity)
	writeUint64(hash, uint64(c.Row))
	writeFramed(hash, c.Field)
	writeFramed(hash, c.Purpose)
	writeUint32(hash, round)

	var digest [Size]byte
	hash.Sum(digest[:0])
	return digest
}

// The three helpers below ignore the write result. A hash never fails a write
// and never writes short, which its own documentation states, so there is
// nothing to report and nothing to recover from.

func writeFramed(w io.Writer, value string) {
	writeUint64(w, uint64(len(value)))
	io.WriteString(w, value)
}

func writeUint64(w io.Writer, value uint64) {
	var buffer [8]byte
	binary.BigEndian.PutUint64(buffer[:], value)
	w.Write(buffer[:])
}

func writeUint32(w io.Writer, value uint32) {
	var buffer [4]byte
	binary.BigEndian.PutUint32(buffer[:], value)
	w.Write(buffer[:])
}
