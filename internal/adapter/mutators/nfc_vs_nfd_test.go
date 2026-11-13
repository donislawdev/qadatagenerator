package mutators_test

import (
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
)

// 🔴 The decomposed form is written out code point by code point here, because
// the mutator computes it with the same library a test would otherwise ask.
// Asking it twice would only show that it agrees with itself.
//
// Five kinds of character, chosen because a hand written table would get each
// one wrong in a different way: Polish letters whose accent comes apart, the l
// with a stroke, which is a letter of its own and has no decomposition, a Korean
// syllable that decomposes into its three letters, the angstrom sign, whose
// decomposition is a different character from the one it looks like, and a
// superscript two, which the compatibility form would fold into a plain two and
// the canonical form leaves alone.
func TestTheDecomposedFormHasEveryAccentApart(t *testing.T) {
	for _, each := range []struct{ name, before, after string }{
		{"Polish", "\U0000017B\U000000F3\U00000142\U00000107", "Z\U00000307o\U00000301\U00000142c\U00000301"},
		{"Korean syllable", "\U0000D55C", "\U00001112\U00001161\U000011AB"},
		{"angstrom sign", "\U0000212B", "A\U0000030A"},
		{"name with a plain part", "Zo\U000000EB", "Zoe\U00000308"},
		{"superscript kept", "\U000000E9\U000000B2", "e\U00000301\U000000B2"},
	} {
		t.Run(each.name, func(t *testing.T) {
			outcome := withDefaults(t, nfcVsNFD, each.before)
			if !outcome.Applied() {
				t.Fatalf("%+q was left alone: %s", each.before, outcome.Reason())
			}
			if outcome.Value() != each.after {
				t.Errorf("%+q became %+q, want %+q", each.before, outcome.Value(), each.after)
			}
		})
	}
}

// The entry says the damage can be undone, and for text that arrived composed
// the composed form of the result is the value it started from.
func TestTheDecomposedValueComposesBackToTheOriginal(t *testing.T) {
	for _, before := range []string{
		"\U0000017B\U000000F3\U00000142\U00000107", "Fran\U000000E7ois", "\U0000D55C\U0000AE00",
		"Ren\U000000E9e Gr\U000000FCn", "\U00000130stanbul",
	} {
		outcome := withDefaults(t, nfcVsNFD, before)
		if !outcome.Applied() {
			t.Fatalf("%+q was left alone: %s", before, outcome.Reason())
		}
		if back := norm.NFC.String(outcome.Value()); back != before {
			t.Errorf("%+q composes back to %+q", before, back)
		}
		if outcome.Value() == before {
			t.Errorf("%+q was reported as damaged and did not change", before)
		}
	}
}

// 🔴 A value that already reads the same in both forms is unfit rather than
// skipped, so the engine can try another column of the row. Text of plain
// Latin letters, an address and a code are all like that, and they are most of
// what a file holds.
func TestAValueWithNothingToDecomposeIsUnfit(t *testing.T) {
	for _, each := range []struct{ value, says string }{
		{"Maria", "same in NFC and NFD"},
		{"maria@example.test", "same in NFC and NFD"},
		{"90041158021", "same in NFC and NFD"},
		{"Z\U00000307o\U00000301", "already decomposed"},
		{"", "empty"},
	} {
		outcome := withDefaults(t, nfcVsNFD, each.value)
		if outcome.Applied() {
			t.Errorf("%+q was damaged into %+q, and it has nothing to decompose", each.value, outcome.Value())
			continue
		}
		if !outcome.ValueUnfit() {
			t.Errorf("%+q was refused for a reason that does not move the order on: %s",
				each.value, outcome.Reason())
		}
		if !strings.Contains(outcome.Reason(), each.says) {
			t.Errorf("%+q was refused with %q, which does not say %q", each.value, outcome.Reason(), each.says)
		}
	}
}

// A form other than the one this build writes is refused where it stands. It
// is about the knob rather than the value, so no other column would fare
// better. The compatibility forms in particular fold characters that mean
// different things, and the entry promises the same text.
func TestAFormThisBuildDoesNotWriteIsRefusedWhereItStands(t *testing.T) {
	for _, params := range []mutator.Parameters{{"form": "NFKD"}, {"form": "NFC"}, {}} {
		outcome := damage(t, nfcVsNFD, mutator.Target{
			Value: "\U0000017B\U000000F3\U00000142\U00000107", Field: textColumn(), Params: params,
		})
		if outcome.Applied() || outcome.ValueUnfit() {
			t.Errorf("with %v the outcome was applied %v, unfit %v: %s",
				params, outcome.Applied(), outcome.ValueUnfit(), outcome.Reason())
		}
	}
}
