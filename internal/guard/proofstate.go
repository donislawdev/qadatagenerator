package guard

// What a guard is, and how we know it can fail.
//
// Split out of the register beside it on 2026-09-03, when that file reached the
// hard ceiling on the length of a file. The ceiling may only be lowered, so the
// answer is to divide rather than to raise it, and this is the seam that costs
// nothing: what a guard IS does not change when one is added, and the register
// changes every time it does. It also leaves the register as the whole of the
// file two of the tools read, which they find by that path.

// ProofState is how we know a guard can go red.
type ProofState string

const (
	// ByMutation means the mutation list carries an entry that breaks the
	// protected behaviour, and the named test goes red when it runs. This is
	// the strongest of the three.
	ByMutation ProofState = "mutation"
	// ByProbe means there is no product code under the guard to break. It
	// reads the tree, compares two lists or checks a document, so the fault
	// is planted in a fixture inside the test itself.
	ByProbe ProofState = "probe"
	// Unproven means nobody has shown this one can fail. Said out loud so
	// that it does not read like the other two. This list may only shrink.
	Unproven ProofState = "unproven"
)

// Guard is one protected behaviour together with the test that protects it.
type Guard struct {
	// Behaviour is what stops being true if this guard is removed. One
	// sentence, in the present tense, about the product rather than about the
	// test.
	Behaviour string
	// Test is the name of the function that goes red.
	Test string
	// Package is where that function lives, relative to the module root.
	Package string
	// Proof is how we know it can go red.
	Proof ProofState
}

// UnprovenCeiling is a ratchet. It may only be lowered.
//
// Raising it to make a run green is the same act as editing a golden value for
// the same reason. Lowering it is ordinary work.
//
// It stood at one until the window existed, and the reason was written here
// rather than left to be rediscovered: the rule that the command line binary
// never reaches the window could not be broken on purpose while there was no
// window to reach, so a mutation adding that import reported a broken tree
// instead of a caught fault. The condition for the return was stated as the
// day the graphical half exists, that day came with the theme package, and
// the mutation now compiles and is caught. This is what a condition for a
// return is for.
const UnprovenCeiling = 0

// CountByProof returns how many guards are in each state.
func CountByProof() map[ProofState]int {
	counts := make(map[ProofState]int, 3)
	for _, guard := range Guards {
		counts[guard.Proof]++
	}
	return counts
}
