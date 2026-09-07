package mutators_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

const checksumTransposedDigits catalog.ID = "L3.checksum.transposed_digits"

// 🔴 The claim of this entry, measured against every scheme rather than argued.
//
// It is deliberately weaker than the one its sibling makes, and the weakness is
// the point. Damage to a single digit can promise that every scheme stops
// accepting the result. A transposition cannot, because whether a scheme sees
// one depends on which two digits moved: the article number and the book number
// are blind to the pairs differing by five and the equipment identity is blind
// to nought against nine.
//
// So what is held here is the disjunction the manifest rests on. Either the
// damage happened and the scheme's own validator now rejects the value, or the
// damage did not happen and something says why. The combination this forbids is
// the one that would put a false expectation in front of a test: reported as
// done, and still accepted by the scheme it belongs to.
//
// A verdict of unsupported counts as a failure of the first branch exactly as a
// valid one does. It means the validator recognised a form it does not check,
// so nothing was rejected and nothing was approved.
func TestNoSchemeIsEverToldToRejectAValueItStillAccepts(t *testing.T) {
	entry := entryOf(t, checksumTransposedDigits)
	schemes, applied, declined := 0, 0, 0
	for _, id := range schemeNames() {
		if !entry.CoversProvider(id) {
			continue
		}
		schemes++
		scheme := schemesThisBuildHas[id]
		t.Run(id, func(t *testing.T) {
			for row := int64(0); row < 40; row++ {
				correct := produce(t, scheme, row)
				outcome := damage(t, checksumTransposedDigits, mutator.Target{
					Value:    correct,
					Field:    identifierField(id),
					Provider: scheme,
					At:       seed.Coordinate{Entity: "corrupt", Row: row, Field: "identifier"},
					Source:   seed.New("transposition corpus"),
					Params:   defaultsOf(entry),
				})
				if !outcome.Applied() {
					declined++
					if outcome.Reason() == "" {
						t.Fatalf("%s was left alone and nothing says why", correct)
					}
					continue
				}
				applied++
				if outcome.Value() == correct {
					t.Errorf("%s was reported as damaged and did not change", correct)
				}
				verdict := scheme.Validate(outcome.Value())
				if verdict.Valid {
					t.Errorf("%s still accepts %s, which came from %s and was reported as damaged",
						id, outcome.Value(), correct)
				}
				if verdict.Unsupported {
					t.Errorf("%s passed %s over rather than rejecting it, and it was reported "+
						"as damaged: %s", id, outcome.Value(), verdict.Reason)
				}
			}
		})
	}
	if schemes == 0 {
		t.Fatal("this damage covers no scheme this build carries, so nothing above ran")
	}
	if applied == 0 {
		t.Fatal("no value anywhere was damaged, so the branch that matters was never taken")
	}
	t.Logf("%d schemes, %d values damaged, %d declined", schemes, applied, declined)
}

// 🔴 The case the whole design of this mutator exists for.
//
// This value is chosen rather than drawn, and it is chosen so that the draw has
// no freedom: eleven of its twelve adjacent pairs hold one digit twice, so the
// only pair that can be swapped is the nought against the five at the eleventh
// and twelfth positions. Those two differ by five, which is precisely what this
// scheme's alternating weights cannot see.
//
// The test asserts both halves. The swap really does leave a value the scheme
// still accepts, which is what makes the case real rather than contrived, and
// the mutator declines rather than reporting damage that a manifest would then
// promise an application will reject.
func TestATranspositionTheSchemeCannotSeeIsDeclinedRatherThanPromisedAsARejection(t *testing.T) {
	const onlyPairDiffersByFive = "0000000000055"
	const afterTheSwap = "0000000000505"

	scheme := intl.EAN{}
	if verdict := scheme.Validate(onlyPairDiffersByFive); !verdict.Valid {
		t.Fatalf("the fixture is not a correct value to begin with: %s", verdict.Reason)
	}
	if verdict := scheme.Validate(afterTheSwap); !verdict.Valid {
		t.Fatalf("the blind spot this rests on is not there any more, so this case is not the "+
			"one it claims to be: %s", verdict.Reason)
	}

	outcome := damage(t, checksumTransposedDigits, mutator.Target{
		Value:    onlyPairDiffersByFive,
		Field:    identifierField("ean"),
		Provider: scheme,
		At:       seed.Coordinate{Entity: "corrupt", Row: 1, Field: "identifier"},
		Source:   seed.New("blind spot"),
		Params:   mutator.Parameters{"position": "random"},
	})
	if outcome.Applied() {
		t.Fatalf("a swap this scheme cannot see was reported as damage, and the value became %q",
			outcome.Value())
	}
	if !strings.Contains(outcome.Reason(), "still accepts") {
		t.Errorf("the reason does not say the scheme still accepts the value: %s", outcome.Reason())
	}
}

// The seed has to reach the draw, and this is the only thing that says so.
//
// 🔴 It is here because the source arrived later than the coordinate beside it.
// A mutator handed a coordinate and no source can still choose a pair, and the
// choice looks correct: it is the same for two runs of one seed, which is half
// of what determinism means. What it is not is different for two seeds, and a
// run that produced the same swap whatever seed it was given would say nothing
// about it anywhere.
//
// So both halves are held. One seed twice gives one answer, and a corpus of
// seeds gives more than one, which can only happen if the seed is consulted.
func TestOneSeedDrawsOnePairAndDifferentSeedsDoNotAllDrawTheSameOne(t *testing.T) {
	scheme := intl.EAN{}
	value := produce(t, scheme, 3)
	at := seed.Coordinate{Entity: "corrupt", Row: 3, Field: "identifier", Purpose: "corrupt"}

	swap := func(source seed.Source) string {
		outcome := damage(t, checksumTransposedDigits, mutator.Target{
			Value: value, Field: identifierField("ean"), Provider: scheme,
			At: at, Source: source, Params: mutator.Parameters{"position": "random"},
		})
		if !outcome.Applied() {
			return "declined: " + outcome.Reason()
		}
		return outcome.Value()
	}

	first := swap(seed.New("one"))
	if again := swap(seed.New("one")); again != first {
		t.Errorf("one seed drew two different pairs for one place: %s then %s", first, again)
	}

	seen := map[string]bool{first: true}
	for _, other := range []string{"two", "three", "four", "five", "six", "seven", "eight"} {
		seen[swap(seed.New(other))] = true
	}
	if len(seen) < 2 {
		t.Fatalf("eight seeds all produced %s, so the seed is not reaching the draw", first)
	}
}

// stated is a scheme that answers whatever a case needs it to.
//
// A double rather than a real provider, because these cases are about what this
// mutator does with each answer and not about any scheme's arithmetic. Two of
// them cannot be reached through a real one at all: every unsupported form this
// build has is decided by a length or by a letter, and swapping two digits
// changes neither.
type stated struct {
	before provider.Verdict
	after  provider.Verdict
}

func (stated) ID() string                                { return "test.scheme" }
func (stated) Capabilities() []capability.Name           { return nil }
func (stated) Generate(provider.Request) (string, error) { return "", nil }

// Validate answers for the value it was built around the first time and for
// everything else the second, which is how one double covers a before and an
// after that have to differ.
func (s *stated) Validate(string) provider.Verdict {
	answer := s.before
	s.before = s.after
	return answer
}

func TestWhatThisRefusesAndTheSentenceItRefusesWith(t *testing.T) {
	accepted := provider.Valid()
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"no position given", mutator.Target{
			Value: "0000000000055", Provider: intl.EAN{}, Params: mutator.Parameters{}},
			"nothing says which pair of digits to swap"},
		{"a position this build cannot aim at", mutator.Target{
			Value: "0000000000055", Provider: intl.EAN{},
			Params: mutator.Parameters{"position": "first"}},
			"is not a position this build can swap at"},
		{"the column follows no scheme", mutator.Target{
			Value: "0000000000055", Params: mutator.Parameters{"position": "random"}},
			"no numbering scheme this build can check"},
		{"the value was already broken", mutator.Target{
			Value:    "0000000000054",
			Provider: &stated{before: provider.Invalid("the check digit is wrong"), after: accepted},
			Params:   mutator.Parameters{"position": "random"}},
			"did not satisfy test.scheme before this ran"},
		{"nothing in the value is a pair of digits", mutator.Target{
			Value:    "AB-CD",
			Provider: &stated{before: accepted, after: provider.Invalid("unreachable")},
			Params:   mutator.Parameters{"position": "random"}},
			"no two adjacent characters of the value are both digits"},
		{"every adjacent pair holds one digit twice", mutator.Target{
			Value: "0000000000000", Provider: intl.EAN{},
			Params: mutator.Parameters{"position": "random"}},
			"holds one digit twice"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			one.target.Source = seed.New("refusals")
			outcome := damage(t, checksumTransposedDigits, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %q", outcome.Value())
			}
			if outcome.Reason() == "" {
				t.Fatal("nothing happened and nothing says why, which is the silence this forbids")
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}

// A pair holding one digit twice is left out of the draw rather than drawn and
// then refused, and the difference shows on a value that has one of each.
//
// Eleven of the twelve pairs here hold two noughts. If they were in the draw,
// eleven times in twelve a seed would land on one and the answer would be a
// refusal about identical digits, which says nothing about the value and
// nothing about the scheme. Every seed has to reach the one pair that is a
// transposition at all.
func TestAPairHoldingOneDigitTwiceIsNeverWhatGetsDrawn(t *testing.T) {
	scheme := intl.EAN{}
	// The nought and the two differ by two, so this scheme does see the swap
	// and the outcome is damage rather than the decline a blind spot produces.
	// Both values were computed rather than written from memory, by walking
	// every thirteen digit value whose digits form two runs.
	const onlyPairDiffersByTwo = "0000000000222"
	const afterTheOnlySwap = "0000000002022"
	if verdict := scheme.Validate(onlyPairDiffersByTwo); !verdict.Valid {
		t.Fatalf("the fixture is not a correct value to begin with: %s", verdict.Reason)
	}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"} {
		outcome := damage(t, checksumTransposedDigits, mutator.Target{
			Value: onlyPairDiffersByTwo, Field: identifierField("ean"), Provider: scheme,
			At:     seed.Coordinate{Entity: "corrupt", Row: 1, Field: "identifier"},
			Source: seed.New(name), Params: mutator.Parameters{"position": "random"},
		})
		if !outcome.Applied() {
			t.Fatalf("seed %q drew nothing usable from a value with one transposable pair: %s",
				name, outcome.Reason())
		}
		if outcome.Value() != afterTheOnlySwap {
			t.Fatalf("seed %q swapped somewhere other than the one pair that differs: %s",
				name, outcome.Value())
		}
	}
}

// 🔴 The two directions of the third verdict, which is the one a caller forgets.
//
// A scheme has three answers and not two. Beside correct and broken there is a
// form it recognises and does not check, and the provider contract says in so
// many words that damaging such a value would put a claim in the manifest that
// nobody verified. Both directions matter and they are different sentences: a
// value that arrives in such a form is not damaged at all, and a swap that
// lands in one is not called a rejection.
//
// Neither can be reached through a real scheme in this build. Every unsupported
// form it has is decided by a length or by a letter, and swapping two digits
// changes neither, which is why the scheme here is a double. That is a fact
// about today rather than about the contract, and the day a scheme reads a
// value rather than its shape, these are already standing.
func TestAFormTheSchemeDoesNotCheckIsNotDamagedInTheFirstPlace(t *testing.T) {
	outcome := damage(t, checksumTransposedDigits, mutator.Target{
		Value:    "0000000000055",
		Provider: &stated{before: provider.Unsupported("this form carries no check digit")},
		Source:   seed.New("unsupported before"),
		Params:   mutator.Parameters{"position": "random"},
	})
	if outcome.Applied() {
		t.Fatalf("a form nobody checks was damaged anyway, and became %q", outcome.Value())
	}
	if !strings.Contains(outcome.Reason(), "recognises this form of its scheme and does not check it") {
		t.Errorf("the reason does not say the scheme passed the value over: %s", outcome.Reason())
	}
}

func TestASwapLandingInAFormTheSchemeDoesNotCheckIsNotCalledARejection(t *testing.T) {
	outcome := damage(t, checksumTransposedDigits, mutator.Target{
		Value:    "0000000000055",
		Provider: &stated{before: provider.Valid(), after: provider.Unsupported("a form with no check digit")},
		Source:   seed.New("unsupported after"),
		Params:   mutator.Parameters{"position": "random"},
	})
	if outcome.Applied() {
		t.Fatalf("a swap into a form nobody checks was reported as damage, and the value became %q",
			outcome.Value())
	}
	if !strings.Contains(outcome.Reason(), "produced a form test.scheme does not check") {
		t.Errorf("the reason does not say the swap landed in a form nobody checks: %s", outcome.Reason())
	}
}
