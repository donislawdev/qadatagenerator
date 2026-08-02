package pl_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// seeds are the shapes worth starting from: correct values, values broken one
// rule at a time, and the shapes that are not numbers at all.
var seeds = []string{
	"90041158028",
	"00810179668",
	"99723128814",
	"00222991748",
	"",
	"0",
	"00000000000",
	"99999999999",
	"9004115802",   // one short
	"900411580288", // one long
	"9004115802x",
	"85136000000", // a month in no century range
	"90023058021", // the thirtieth of February
	"           ",
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
	"٩٠٠٤١١٥٨٠٢٨", // digits of another script
	"9004115802é",
}

// The two validators are written from one description by one hand, so the
// interesting question is not whether they agree on the values we produced. It
// is whether anything at all makes them disagree.
//
// A differential fuzz is the only way to ask that without imagining the answer
// first. Four states of a path are written by somebody who has to think of
// them. A fuzzer thinks of nothing and therefore finds different things.
func FuzzValidateAgreesWithTheIndependentValidator(f *testing.F) {
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		ours := pl.PESEL{}.Validate(value)
		theirs := oracle.CheckPESEL(value)

		if ours.Valid != (len(theirs) == 0) {
			t.Fatalf("the two validators disagree about %q: ours says valid=%v (%s), the independent one says %v",
				value, ours.Valid, ours.Reason, theirs)
		}
		if !ours.Valid && ours.Reason == "" {
			t.Fatalf("%q was refused without a reason, which is the silence this tool argues against", value)
		}
	})
}

// nipSeeds are the shapes worth starting the tax number from: correct values in
// all three accepted spellings, values broken one rule at a time, the body no
// check digit can complete, and shapes that are not numbers at all.
var nipSeeds = []string{
	"1234563218",
	"123-456-32-18",
	"123-45-63-218",
	"2234567895", // the number the source works out by hand
	"5659396047",
	"9000000000", // nine hundred million leaves a remainder of ten
	"9000000005",
	"",
	"0",
	"0000000000",
	"9999999999",
	"123456321",   // one short
	"12345632188", // one long
	"123456321x",
	"1-2-3-4-5-6-7-8-9-0",
	"123 456 32 18",
	"----------",
	"\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00",
	"١٢٣٤٥٦٣٢١٨", // digits of another script
	"123456321é",
}

// The same differential question as for the personal number, asked about the
// tax one. Two validators written from one description by one hand agree by
// construction, so the only useful question is whether anything at all makes
// them disagree.
func FuzzTaxValidateAgreesWithTheIndependentValidator(f *testing.F) {
	for _, value := range nipSeeds {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, value string) {
		ours := pl.NIP{}.Validate(value)
		theirs := oracle.CheckNIP(value)

		if ours.Valid != (len(theirs) == 0) {
			t.Fatalf("the two validators disagree about %q: ours says valid=%v (%s), the independent one says %v",
				value, ours.Valid, ours.Reason, theirs)
		}
		if !ours.Valid && ours.Reason == "" {
			t.Fatalf("%q was refused without a reason, which is the silence this tool argues against", value)
		}
	})
}

// A value this build calls valid has to survive being rebuilt with its own
// check digit, in the shape it arrived in. Anything else means the reader and
// the writer disagree about the same number, and it is the written shape rather
// than the digits where that is most likely to go wrong.
func FuzzTheTaxReadersNeverPanic(f *testing.F) {
	for _, value := range nipSeeds {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, value string) {
		subject := pl.NIP{}

		// The results are deliberately unused: the property under test is that
		// each call answers rather than fails.
		_, _ = subject.CheckDigits(value)
		_, _ = subject.WithCheckDigits(value)

		if !subject.Validate(value).Valid {
			return
		}
		positions, ok := subject.CheckDigits(value)
		if !ok || len(positions) != 1 || positions[0] != len(value)-1 {
			t.Fatalf("%q is valid and its check digit was placed at %v", value, positions)
		}
		rebuilt, err := subject.WithCheckDigits(value)
		if err != nil {
			t.Fatalf("%q is valid and recomputing its own check digit failed: %v", value, err)
		}
		if rebuilt != value {
			t.Fatalf("%q rebuilt with its own check digit became %q", value, rebuilt)
		}
	})
}

// Nothing a person can put in a file may take the process down. A recipe comes
// from somebody else's repository, so the values it names are untrusted input
// rather than our own data.
func FuzzTheReadersNeverPanic(f *testing.F) {
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		subject := pl.PESEL{}

		// Each of these takes a value from outside and has to answer rather
		// than fail. The results are deliberately unused: the property under
		// test is that the call returns at all.
		_, _ = subject.EncodedDate(value)
		_, _ = subject.EncodedGender(value)
		_, _ = subject.CheckDigits(value)
		_, _ = subject.WithCheckDigits(value)
		_, _ = subject.WithDate(value, provider.Date{Year: 1990, Month: 4, Day: 11})
		_, _ = subject.WithGender(value, provider.Female)

		// A value this build calls valid has to survive being rebuilt with the
		// date and the gender it already carries. Anything else would mean the
		// reader and the writer disagree about the same number.
		if !subject.Validate(value).Valid {
			return
		}
		date, ok := subject.EncodedDate(value)
		if !ok {
			t.Fatalf("%q is valid and yet carries no readable date", value)
		}
		rebuilt, err := subject.WithDate(value, date)
		if err != nil {
			t.Fatalf("%q is valid and rebuilding it with its own date failed: %v", value, err)
		}
		if rebuilt != value {
			t.Fatalf("%q rebuilt with its own date became %q", value, rebuilt)
		}
	})
}
