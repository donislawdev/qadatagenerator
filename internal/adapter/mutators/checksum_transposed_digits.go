package mutators

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// checksumTransposedDigitsID is the catalog entry this implements.
const checksumTransposedDigitsID catalog.ID = "L3.checksum.transposed_digits"

// randomPosition is the one value of the entry's position knob this build
// carries. The name of the knob itself is shared with the damage that changes a
// single digit, because it is one published parameter and not two.
const randomPosition = "random"

// drawPurpose separates this draw from every other one made about the same
// place. The engine already draws twice about a row, for whether to damage it
// and for which damage to do, and a third draw sharing their address would
// return one of their answers.
const drawPurpose = ":transposed-digits"

// checksumTransposedDigits swaps two adjacent digits, which is the mistake a
// person makes most often when copying a number by hand.
//
// # Why this one asks the scheme afterwards, when its sibling does not
//
// The damage that changes a single digit narrowed itself to the last position,
// because that is the one place where the rejection its entry promises is true
// of every scheme. The same narrowing is not available here, and the reason is
// not that nobody looked for it.
//
// Whether a transposition is detectable depends on the two digits and not on
// where they sit. Measured exhaustively while the schemes were written: the
// article number and the book number miss exactly the pairs differing by five,
// and the equipment identity misses exactly nought against nine. Two blind
// spots of different sizes under one entry, and neither of them is a position
// this could refuse to aim at.
//
// So the expectation is settled per value rather than per scheme. The swap is
// made in a buffer, the scheme's own validator is asked whether it still
// accepts the result, and a value it still accepts is put back. What reaches
// the manifest then is damage that did not happen, with a sentence saying why,
// rather than a promise that an application will reject a value which is
// arithmetically correct.
//
// # Why it does not go looking for a swap that works
//
// It would be a short edit and it would destroy the point. A mutator that tried
// positions until one of them broke the check would report success on every row
// and the blind spot would never appear in a manifest, which is the silence
// this product exists to break. Whether this row was caught is a fact about
// this row, and the manifest is where facts about rows go.
//
// Pairs holding the same digit twice are left out of the draw, and that is a
// different thing. Swapping a five with a five is not damage the scheme failed
// to see, it is not damage at all, and no validator is asked about it. The
// value alone settles that, which is why it is settled before the draw.
type checksumTransposedDigits struct{}

// ID is the catalog identifier.
func (checksumTransposedDigits) ID() catalog.ID { return checksumTransposedDigitsID }

// Damage swaps one pair of adjacent digits and keeps the result only if the
// scheme stopped accepting it.
//
// Whether the field is an identifier, whether its scheme is one the entry
// covers, and whether the run can compute a check digit at all are settled
// before this is called. What is left here is the part only this can know.
func (checksumTransposedDigits) Damage(target mutator.Target) mutator.Outcome {
	scheme, refusal := whatStopsATransposition(target)
	if scheme == nil {
		return refusal
	}
	pairs, identical := transposablePairs(target.Value)
	if len(pairs) == 0 {
		return nothingHereToTranspose(identical)
	}
	// Drawn rather than taken from either end, because the entry says the
	// position is random and a person's slip is not fixed to one place. The
	// bound is the number of pairs and never nought, which the branch above
	// guarantees and the draw requires.
	at := pairs[target.Source.IntN(drawnAt(target.At), int64(len(pairs)))]
	return verdictOnTheSwap(scheme, transpose(target.Value, at), at)
}

// whatStopsATransposition answers with the scheme to damage against, or with
// nothing and the refusal to record instead.
//
// 🔴 The value has to be correct before this runs, or the entry would credit a
// swap with a rejection that was already going to happen. In the mode that
// damages somebody else's file the incoming value is whatever they had, so this
// is a real state and not a defensive one.
//
// Unsupported is refused for the reason the provider contract gives: a form the
// build recognises and does not check is neither approved nor faulted, and
// damaging it would put a claim in the manifest that nobody verified.
func whatStopsATransposition(target mutator.Target) (provider.Provider, mutator.Outcome) {
	where, given := target.Params.Text(positionParameter)
	if !given {
		return nil, mutator.Skipped(fmt.Sprintf(
			"no %s was given, so nothing says which pair of digits to swap", positionParameter))
	}
	if where != randomPosition {
		return nil, mutator.Skipped(fmt.Sprintf(
			"%q is not a position this build can swap at, because the only one it carries is %q, "+
				"which draws the pair from the seed", where, randomPosition))
	}
	scheme := target.Provider
	if scheme == nil {
		return nil, mutator.Skipped(
			"the column follows no numbering scheme this build can check, and a swap nobody can " +
				"check is a rejection nobody can promise")
	}
	before := scheme.Validate(target.Value)
	switch {
	case before.Unsupported:
		return nil, mutator.Skipped(fmt.Sprintf(
			"%s recognises this form of its scheme and does not check it, so a swap in it would "+
				"break nothing anybody could verify. %s", scheme.ID(), before.Reason))
	case !before.Valid:
		return nil, mutator.Skipped(fmt.Sprintf(
			"the value did not satisfy %s before this ran, so a swap cannot be what makes an "+
				"application reject it. %s", scheme.ID(), before.Reason))
	}
	return scheme, mutator.Outcome{}
}

// nothingHereToTranspose tells the two empty cases apart, because they are two
// different things to read in a report.
func nothingHereToTranspose(identical int) mutator.Outcome {
	if identical > 0 {
		return mutator.Skipped(
			"every pair of adjacent digits in the value holds one digit twice, so swapping " +
				"any of them would leave the value exactly as it is")
	}
	return mutator.Skipped(
		"no two adjacent characters of the value are both digits, so there is no pair here " +
			"to transpose")
}

// verdictOnTheSwap keeps the damage only where the scheme rejects the result,
// and every scheme it lies inside rejects it too.
//
// # Why the wider schemes are asked as well
//
// A book number is an article number with a fixed prefix, and the swap of an
// eight and a three right after the 978 of a German book number moves the
// prefix and leaves the arithmetic whole. The book number validator refuses
// that value for its prefix, the article number validator accepts it, and an
// application reading the column as article numbers takes it without a word.
// The inference in corrupt chooses the narrower scheme of such a pair
// (DEC-269), so the reader it did not choose is exactly the one this asks.
func verdictOnTheSwap(scheme provider.Provider, damaged string, at int) mutator.Outcome {
	after := scheme.Validate(damaged)
	switch {
	case after.Valid:
		return mutator.Skipped(fmt.Sprintf(
			"swapping the digits at positions %d and %d leaves a value %s still accepts, so a "+
				"test given this row would find nothing to reject", at+1, at+2, scheme.ID()))
	case after.Unsupported:
		return mutator.Skipped(fmt.Sprintf(
			"swapping the digits at positions %d and %d produced a form %s does not check, so "+
				"nothing here was verified as rejected", at+1, at+2, scheme.ID()))
	}
	if narrower, ok := scheme.(provider.Narrower); ok {
		for _, wider := range narrower.Within() {
			if verdict := wider.Validate(damaged); verdict.Valid || verdict.Unsupported {
				return mutator.Skipped(fmt.Sprintf(
					"swapping the digits at positions %d and %d breaks %s but not %s, and every %s "+
						"is valid as %s too, so an application reading this column as %s would find "+
						"nothing to reject", at+1, at+2, scheme.ID(), wider.ID(), scheme.ID(),
					wider.ID(), wider.ID()))
			}
		}
	}
	return mutator.Done(damaged)
}

// drawnAt is the address of this mutator's draw about a value.
func drawnAt(at seed.Coordinate) seed.Coordinate {
	at.Purpose += drawPurpose
	return at
}

// transposablePairs returns the offsets where two adjacent digits differ, and
// how many adjacent pairs held the same digit twice.
//
// The second result is not a statistic. It is what lets a refusal tell the
// difference between a value with no digits to swap and one whose digits are
// all the same, and those are two different things to read in a report.
//
// 🔴 Digits are the ten of ASCII, compared as bytes, for the reason written out
// beside the damage that changes one digit: the digits of a dozen other scripts
// would swap into a value whose arithmetic means nothing. Every byte of a
// character outside ASCII is above this range, so a pair is never split down
// the middle of one.
func transposablePairs(value string) (pairs []int, identical int) {
	for at := 0; at+1 < len(value); at++ {
		left, right := value[at], value[at+1]
		if !isDigit(left) || !isDigit(right) {
			continue
		}
		if left == right {
			identical++
			continue
		}
		pairs = append(pairs, at)
	}
	return pairs, identical
}

// isDigit reports whether the byte is one of the ten ASCII digits.
func isDigit(of byte) bool { return of >= '0' && of <= '9' }

// transpose returns the value with the bytes at the offset and the next one
// exchanged.
//
// A fresh slice, so the value handed in is the one still held by whoever handed
// it in. That matters more here than it looks: this refuses after damaging, and
// a refusal beside a value that had already been changed in place would be a
// manifest disagreeing with the file it describes.
func transpose(value string, at int) string {
	bytes := []byte(value)
	bytes[at], bytes[at+1] = bytes[at+1], bytes[at]
	return string(bytes)
}
