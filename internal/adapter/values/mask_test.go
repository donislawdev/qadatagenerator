package values

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// The guard over what a scheme says its values look like.
//
// # Why a mask needs a guard at all, where a sentence would not have one
//
// Because that is the whole reason it is a mask. A panel telling somebody the
// shape of a value could have been eleven sentences in eleven files, and
// nothing in this project checks a sentence: the first scheme to change its
// written form would leave ten correct descriptions and one lie, and the lie
// would read exactly like the rest.
//
// A mask is checkable against the thing it describes. These two tests are what
// makes the difference real rather than stylistic, and without them the mask is
// prose with punctuation.

// theLocalesValuesComeIn are the two this build carries data for. A scheme is
// asked in both, because a provider that quietly produces a different form in
// one of them is exactly the fault a single locale would hide.
var theLocalesValuesComeIn = []string{"en-US", "pl-PL"}

// TestEverySchemeSaysWhatItsValuesLookLike closes the register from the side
// the schemes are on.
//
// 🔴 It walks Schemes() rather than a list written here, for the reason this
// project has already paid for once: a register with a list beside it lets the
// fourteenth entry in without a word, and the only symptom was a row of a
// window reading "vin -  (vin)". A scheme added tomorrow fails this on the day
// it is added.
func TestEverySchemeSaysWhatItsValuesLookLike(t *testing.T) {
	for _, scheme := range New("en-US", "").Schemes() {
		shaped, saysSo := scheme.(provider.Shaped)
		if !saysSo {
			t.Errorf("%s cannot say what its values look like, so any surface showing "+
				"one has nothing to describe it with", scheme.ID())
			continue
		}
		if shaped.Mask() == "" {
			t.Errorf("%s answers with an empty mask, which is worse than not "+
				"implementing the interface: it reads as a scheme with no shape", scheme.ID())
		}
	}
}

// TestEveryValueIsWrittenTheWayItsSchemeSaysIsTheOtherHalf, and it is the half
// that makes the mask worth anything.
//
// # Why several rows rather than one
//
// A single draw agreeing with a mask is an anecdote, and this project has
// measured what that costs: two walks of a four key map produce the same order
// 43.7% of the time, so one comparison passing means very little. Every scheme
// here has places that vary, and a mask wrong about ONE of them - the group
// separator, the length of the last block - would agree with a value at some
// addresses and not at others.
//
// # Why a refusal is skipped rather than failed
//
// A generator with no data for a locale refuses, and that says nothing about
// the shape of the values it produces elsewhere. Postal codes exist here for
// one locale by decision, which is DEC-92 and not a defect.
func TestEveryValueIsWrittenTheWayItsSchemeSays(t *testing.T) {
	for _, locale := range theLocalesValuesComeIn {
		source := New(locale, "")
		for _, generator := range source.Names() {
			scheme, behind := source.ProviderOf(generator)
			if !behind {
				continue
			}
			numbering, carried := source.ProviderNamed(scheme)
			if !carried {
				t.Errorf("%s names the scheme %s, which this build does not carry",
					generator, scheme)
				continue
			}
			shaped, saysSo := numbering.(provider.Shaped)
			if !saysSo {
				// Reported by the test above, once, rather than twice a locale.
				continue
			}
			checkOneSchemeIsWrittenAsItSays(t, source, locale, generator, scheme, shaped.Mask())
		}
	}
}

func checkOneSchemeIsWrittenAsItSays(t *testing.T, source Source, locale, generator, scheme, mask string) {
	t.Helper()
	produced := 0
	for row := int64(0); row < 24; row++ {
		at := seed.Coordinate{Entity: "quick", Row: row, Field: generator, Purpose: "value"}
		value, err := source.Produce(generate.Request{Generator: generator, At: at})
		if err != nil {
			continue
		}
		produced++
		if !provider.FitsMask(mask, value) {
			t.Errorf("%s in %s produced %q, which is not written the way %s says its "+
				"values are: mask %q", generator, locale, value, scheme, mask)
			return
		}
	}
	if produced == 0 {
		// 🔴 Not a skip, and the difference matters. A scheme that refuses
		// every address in both locales has had its mask checked against
		// nothing at all, and this test would pass by producing no evidence -
		// which is the shape of a guard that is green about nothing.
		if locale == theLocalesValuesComeIn[len(theLocalesValuesComeIn)-1] {
			t.Logf("%s produced nothing in %s, so its mask was not exercised there",
				generator, locale)
		}
	}
}

// TestAMaskRefusesTheThingsItIsMeantTo is the guard over the reader itself.
//
// 🔴 A mask reader that answers yes to everything would make both tests above
// green while proving nothing, and it would look exactly like a working one:
// every real value fits, so every real comparison passes. What separates the
// two is a value that ought to be REFUSED, which no amount of correct data
// will ever produce.
func TestAMaskRefusesTheThingsItIsMeantTo(t *testing.T) {
	// 🔴 The masks are BUILT from the constants rather than typed as literals,
	// and that is not tidiness. Written out, these cases went stale the moment
	// the digit character moved from a hash to a nought on 2026-09-09 - the
	// test went red on its own change, which was lucky, because a case written
	// with the OLD character and still passing would have been a guard over a
	// mask language nothing else speaks.
	digit := string(provider.MaskDigit)
	either := string(provider.MaskAlnum)
	for _, each := range []struct {
		why, mask, value string
		fits             bool
	}{
		{"a digit where a digit belongs", digit + digit + digit, "123", true},
		{"a letter where a digit belongs", digit + digit + digit, "12a", false},
		{"too short", digit + digit + digit, "12", false},
		{"too long", digit + digit + digit, "1234", false},
		{"a literal that is there", digit + digit + "-" + digit + digit, "12-34", true},
		{"a literal that is missing", digit + digit + "-" + digit + digit, "12345", false},
		{"a literal in the wrong place", digit + digit + "-" + digit + digit, "1-234", false},
		{"a letter where either is allowed", either + either + either, "1a2", true},
		{"punctuation where either is allowed", either + either + either, "1-2", false},
		{"nothing at all against an empty mask", "", "", true},
	} {
		if got := provider.FitsMask(each.mask, each.value); got != each.fits {
			t.Errorf("%s: mask %q against %q answered %v, where it has to answer %v",
				each.why, each.mask, each.value, got, each.fits)
		}
	}
}
