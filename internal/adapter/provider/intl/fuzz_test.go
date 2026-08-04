package intl_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/oracle"
)

// correctSeeds are correct values for the fuzzer to start from, produced here
// rather than written down.
//
// This scheme has no block that is never issued, so a correct value cannot be
// shown not to be somebody's account. Deriving them changes the one thing that
// can be changed: nothing is published. It does not make them less likely to
// resemble a real account, because no correct value can be, and that is exactly
// why the answer is to store none.
func correctSeeds(f *testing.F) []string {
	f.Helper()
	source := seed.New("fuzz corpus")
	values := make([]string, 0, 4)
	for row := int64(0); row < 4; row++ {
		value, err := intl.IBAN{}.Generate(provider.Request{
			Source: source,
			At:     seed.Coordinate{Entity: "fuzz", Row: row, Field: "iban", Purpose: "value"},
		})
		if err != nil {
			f.Fatalf("row %d: the generator refused to produce a seed: %v", row, err)
		}
		values = append(values, value)
	}
	return values
}

// seeds are the shapes worth starting from: values broken one rule at a time,
// the two check numbers no calculation reaches, and shapes that are not account
// numbers at all. Correct values come from correctSeeds and are deliberately
// absent here.
var seeds = []string{
	"",
	"PL",
	"PL00",
	"PL0000000000000000000000000",   // one short
	"PL000000000000000000000000000", // one long
	"PL0000000000000000000000000A",  // a letter where Poland has digits
	"ZZ000000000000000000000000",
	"ZZ01ABCDEFGHIJKLMNOPQRSTUVWXYZ", // the check number that shares a residue with 98
	"ZZ00ABCDEFGHIJKLMNOPQRSTUVWXYZ", // and the one that shares it with 97
	"pl611090101400000712198128",
	"PL 61 1090 1014 0000 0712 1981 2874",
	"PL61\t1090",
	"           ",
	"\x00\x00\x00\x00\x00\x00",
	"ＰＬ６１", // the letters of another width
	"PL61109010140000071219812é",
}

// The two validators do not restate one another here, which is unusual in this
// project. The act sets out two methods and calls them equivalent, so the
// provider divides and the independent one weighs. A differential fuzz is the
// only way to ask whether anything at all separates them.
func FuzzValidateAgreesWithTheIndependentValidator(f *testing.F) {
	for _, value := range seeds {
		f.Add(value)
	}
	for _, value := range correctSeeds(f) {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, value string) {
		ours := intl.IBAN{}.Validate(value)
		theirs := oracle.CheckIBAN(value)

		// The agreement is about faults, not about approval. This build checks a
		// value from another country only as far as its check digits, and both
		// sides say so in their own way: one with the third verdict, the other
		// by answering a separate question.
		if faulted := !ours.Valid && !ours.Unsupported; faulted != (len(theirs) != 0) {
			t.Fatalf("the two validators disagree about %q: ours says valid=%v unsupported=%v (%s), "+
				"the independent one says %v", value, ours.Valid, ours.Unsupported, ours.Reason, theirs)
		}
		if ours.Valid && !oracle.IBANCheckedInFull(value) {
			t.Fatalf("%q was approved and the independent validator only half checks it", value)
		}
		if !ours.Valid && ours.Reason == "" {
			t.Fatalf("%q was refused without a reason, which is the silence this tool argues against", value)
		}
	})
}

// Nothing a person can put in a file may take the process down. A recipe comes
// from somebody else's repository, so the values it names are untrusted input
// rather than our own data.
func FuzzTheReadersNeverPanic(f *testing.F) {
	for _, value := range seeds {
		f.Add(value)
	}
	for _, value := range correctSeeds(f) {
		f.Add(value)
	}

	f.Fuzz(func(t *testing.T, value string) {
		subject := intl.IBAN{}

		// The results are deliberately unused: the property under test is that
		// each call answers rather than fails.
		_, _ = subject.CheckDigits(value)
		_, _ = subject.WithCheckDigits(value)

		// A value this build reads at all has to survive being rebuilt with the
		// check digits it already carries, in the text it arrived as. That holds
		// for a foreign value too, because recomputing needs the arithmetic and
		// not the length.
		verdict := subject.Validate(value)
		if !verdict.Valid && !verdict.Unsupported {
			return
		}
		positions, ok := subject.CheckDigits(value)
		if !ok || len(positions) != 2 {
			t.Fatalf("%q was read and its check digits were placed at %v", value, positions)
		}
		rebuilt, err := subject.WithCheckDigits(value)
		if err != nil {
			t.Fatalf("%q was read and recomputing its own check digits failed: %v", value, err)
		}
		if rebuilt != value {
			t.Fatalf("%q rebuilt with its own check digits became %q", value, rebuilt)
		}
	})
}
