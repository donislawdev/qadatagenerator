package values

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// TestEverySpellingOfAGenderProducesTheValueTheCanonicalOneDoes holds the rule
// the whole product now shares: one vocabulary for a gender, read the same way
// by everything that reads one.
//
// # What this is about, measured rather than imagined
//
// Until 2026-09-06 the scheme behind a PESEL accepted eleven spellings while
// the generator behind a first name compared the bound value against the word
// female itself. A recipe binding f therefore produced a PESEL of a woman
// beside a name drawn from both lists, in the same row, under one declared
// rule, AND NOTHING SAID SO. That is the class of defect this product exists to
// find in other people's systems.
//
// # Why every spelling against the canonical one, rather than a list of pairs
//
// The property is not that some particular spelling works. It is that no
// spelling of a gender can produce a different value from any other spelling of
// the same gender, at the same address. A reader that starts comparing raw text
// again fails here on ten of the twelve spellings at once, whichever reader it
// is, because none of them is named in this test.
func TestEverySpellingOfAGenderProducesTheValueTheCanonicalOneDoes(t *testing.T) {
	canonical := map[provider.Gender]string{
		provider.Female: string(provider.Female),
		provider.Male:   string(provider.Male),
	}

	for _, locale := range Locales() {
		source := New(locale, "one-vocabulary")
		for _, generator := range source.Names() {
			if !source.ReadsRole(generator, provider.RoleGender) {
				continue
			}
			for _, spelling := range provider.GenderSpellings() {
				gender, known := provider.ParseGender(spelling)
				if !known {
					t.Fatalf("%q is offered as a spelling and cannot be read back, "+
						"so the two halves of the vocabulary disagree", spelling)
				}
				compareOneSpelling(t, source, locale, generator, spelling, canonical[gender])
			}
		}
	}
}

// compareOneSpelling holds one spelling against the canonical one at several
// addresses.
//
// Several rather than one, because a word list of two hundred entries answers
// the same at one address by coincidence often enough to make a single draw an
// anecdote, which this project has measured before.
func compareOneSpelling(t *testing.T, source Source, locale, generator, spelling, canonical string) {
	t.Helper()
	for row := int64(0); row < 8; row++ {
		at := seed.Coordinate{Entity: "quick", Row: row, Field: generator, Purpose: "value"}
		written, writtenErr := source.Produce(generate.Request{
			Generator: generator, At: at,
			Agrees: provider.NewAgreement(map[string]string{provider.RoleGender: spelling}),
		})
		plain, plainErr := source.Produce(generate.Request{
			Generator: generator, At: at,
			Agrees: provider.NewAgreement(map[string]string{provider.RoleGender: canonical}),
		})
		if writtenErr != nil || plainErr != nil {
			// A generator with no data for this locale says nothing about the
			// vocabulary, and refusing both ways is agreement of the kind this
			// test is about. One of them alone would not be.
			if (writtenErr == nil) != (plainErr == nil) {
				t.Errorf("in %s, %s answered %q one way and %q the other for the same gender",
					locale, generator, writtenErr, plainErr)
			}
			return
		}
		if written != plain {
			t.Errorf("in %s, %s produced %q for a gender written %q and %q for the same gender "+
				"written %q. One declared rule, two answers, and nothing says so",
				locale, generator, written, spelling, plain, canonical)
			return
		}
	}
}

// TestAnUnreadableGenderIsRefusedByTheSchemeAndDrawnAroundByAWordList holds the
// one place the readers are ALLOWED to differ, so that the difference is a
// decision rather than the next thing somebody unifies by accident.
//
// A word list draws from both of its lists, because a name without a gender is
// an ordinary name. A numbering scheme refuses, because an identifier that
// cannot encode what it was told to encode is a value nobody asked for. Where
// both are in one rule the run is refused, which is the answer that matters.
func TestAnUnreadableGenderIsRefusedByTheSchemeAndDrawnAroundByAWordList(t *testing.T) {
	source := New("pl-PL", "one-vocabulary")
	at := seed.Coordinate{Entity: "quick", Row: 0, Field: "x", Purpose: "value"}
	nonsense := provider.NewAgreement(map[string]string{provider.RoleGender: "yes"})

	if _, err := source.Produce(generate.Request{Generator: FirstName, At: at, Agrees: nonsense}); err != nil {
		t.Errorf("a first name refused an unreadable gender: %v. A name without one is an ordinary name", err)
	}
	if _, err := source.Produce(generate.Request{Generator: Pesel, At: at, Agrees: nonsense}); err == nil {
		t.Error("a PESEL accepted an unreadable gender, so a value nobody asked for was produced " +
			"under a rule that said otherwise")
	}

	// 🔴 Drawing around an unreadable gender means drawing from BOTH lists, not
	// quietly picking a side, and the difference is the whole worth of this
	// test. A mutation making the unreadable case answer Female went unnoticed
	// until this part existed: the name was still produced, nothing was
	// refused, and every dataset with a nonsense gender column would have read
	// as one sex.
	female := provider.NewAgreement(map[string]string{provider.RoleGender: string(provider.Female)})
	sameAsFemale := 0
	const addresses = 24
	for row := int64(0); row < addresses; row++ {
		where := seed.Coordinate{Entity: "quick", Row: row, Field: "x", Purpose: "value"}
		unreadable, oneErr := source.Produce(generate.Request{
			Generator: FirstName, At: where, Agrees: nonsense})
		bound, otherErr := source.Produce(generate.Request{
			Generator: FirstName, At: where, Agrees: female})
		if oneErr != nil || otherErr != nil {
			t.Fatalf("a first name could not be produced at all: %v %v", oneErr, otherErr)
		}
		if unreadable == bound {
			sameAsFemale++
		}
	}
	if sameAsFemale == addresses {
		t.Errorf("a first name with an unreadable gender produced the same value as one bound to "+
			"female at all %d addresses, so it picks a side quietly and every dataset with a "+
			"nonsense gender column reads as one sex", addresses)
	}
}
