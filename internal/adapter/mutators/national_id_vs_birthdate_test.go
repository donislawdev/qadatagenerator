package mutators_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

const nationalIDVsBirthdate catalog.ID = "L4.coherence.national_id_vs_birthdate"

// The same property as the gender one: what comes out has to be correct, and
// wrong only beside the column it no longer agrees with.
func TestTheDateDriftedIdentifierIsStillCorrectOnItsOwn(t *testing.T) {
	entry := entryOf(t, nationalIDVsBirthdate)
	scheme := pl.PESEL{}
	moved := 0
	for row := int64(0); row < 25; row++ {
		correct := produce(t, scheme, row)
		before, readable := scheme.EncodedDate(correct)
		if !readable {
			t.Fatalf("the generator produced %s, which carries no date", correct)
		}

		outcome := damage(t, nationalIDVsBirthdate, mutator.Target{
			Value: correct, Field: identifierField("PL.pesel"),
			Provider: scheme, Params: defaultsOf(entry),
		})
		if !outcome.Applied() {
			// The twenty ninth of February has nowhere to land most years, and
			// refusing is the right answer. It is counted rather than ignored,
			// because a run where every value refused would otherwise look the
			// same as one that worked.
			if !strings.Contains(outcome.Reason(), "which is not a day") {
				t.Fatalf("%s was left alone for an unexpected reason: %s", correct, outcome.Reason())
			}
			continue
		}
		moved++
		if verdict := scheme.Validate(outcome.Value()); !verdict.Valid {
			t.Errorf("%s became %s, which its own scheme rejects: %s",
				correct, outcome.Value(), verdict.Reason)
		}
		after, readable := scheme.EncodedDate(outcome.Value())
		if !readable {
			t.Fatalf("%s became %s, which carries no date at all", correct, outcome.Value())
		}
		if after.Year == before.Year {
			t.Errorf("%s became %s and still says %d", correct, outcome.Value(), after.Year)
		}
		// 🔴 Only the year moves. A value whose month or day also changed would
		// give whoever reads the manifest three differences to explain when the
		// entry promises one.
		if after.Month != before.Month || after.Day != before.Day {
			t.Errorf("%s became %s, which moved from %v to %v", correct, outcome.Value(), before, after)
		}
	}
	if moved == 0 {
		t.Fatal("no value was moved at all, so nothing above was checked")
	}
}

// The gender must not move. It sits in the same value and this entry is about
// the date, so a mutator that shifted both would produce a row with two defects
// under one name.
func TestDriftingTheDateLeavesTheGenderWhereItWas(t *testing.T) {
	entry := entryOf(t, nationalIDVsBirthdate)
	scheme := pl.PESEL{}
	for row := int64(0); row < 25; row++ {
		correct := produce(t, scheme, row)
		before, _ := scheme.EncodedGender(correct)
		outcome := damage(t, nationalIDVsBirthdate, mutator.Target{
			Value: correct, Field: identifierField("PL.pesel"),
			Provider: scheme, Params: defaultsOf(entry),
		})
		if !outcome.Applied() {
			continue
		}
		if after, _ := scheme.EncodedGender(outcome.Value()); after != before {
			t.Errorf("%s became %s, whose gender went from %s to %s",
				correct, outcome.Value(), before, after)
		}
	}
}

// The third published example this build produces rather than only checks.
//
// 🔴 This pair was not reproducible until 2026-08-11. The two values were
// unrelated correct numbers twenty three years and several months apart, under
// an entry declaring a drift of five years, so nothing could have produced the
// second from the first. It passed every check there was, because the only ones
// that existed asked whether each value was a valid number.
func TestTheCatalogExampleIsWhatTheDateMutatorProduces(t *testing.T) {
	entry := entryOf(t, nationalIDVsBirthdate)
	outcome := damage(t, nationalIDVsBirthdate, mutator.Target{
		Value:    entry.Example.Before,
		Field:    identifierField("PL.pesel"),
		Provider: pl.PESEL{},
		Params:   defaultsOf(entry),
	})
	if !outcome.Applied() {
		t.Fatalf("the published example was left alone: %s", outcome.Reason())
	}
	if outcome.Value() != entry.Example.After {
		t.Errorf("the specification promises %s -> %s and this build produces %s",
			entry.Example.Before, entry.Example.After, outcome.Value())
	}
}

// 🔴 A birth date that cannot move is refused rather than moved somewhere else.
//
// The twenty ninth of February exists one year in four, so drifting it by a
// whole number of years usually lands on a day the calendar does not have. The
// scheme refuses to write one, and this passes that refusal on. Picking a
// nearby day instead would put a date in the data that the policy never asked
// for, and the manifest would describe damage nobody ordered.
func TestABirthDateWithNowhereToLandIsRefusedWithTheDateNamed(t *testing.T) {
	// Built by the scheme rather than written down, for the same reason no
	// correct identifier is written down anywhere in these tests: a valid one
	// cannot be shown not to be somebody's. It also cannot go stale, which a
	// check digit typed by hand can and did.
	leapDay, err := (pl.PESEL{}).WithDate(produce(t, pl.PESEL{}, 0),
		provider.Date{Year: 2004, Month: 2, Day: 29})
	if err != nil {
		t.Fatalf("the scheme refused to build the twenty ninth of February: %v", err)
	}
	if got, _ := (pl.PESEL{}).EncodedDate(leapDay); got.Month != 2 || got.Day != 29 {
		t.Fatalf("the fixture is not the twenty ninth of February: %v", got)
	}

	outcome := damage(t, nationalIDVsBirthdate, mutator.Target{
		Value: leapDay, Field: identifierField("PL.pesel"), Provider: pl.PESEL{},
		Params: mutator.Parameters{driftYears: 5},
	})
	if outcome.Applied() {
		t.Fatalf("a date with nowhere to land became %q", outcome.Value())
	}
	if !strings.Contains(outcome.Reason(), "2009-02-29") {
		t.Errorf("the reason does not name the day that does not exist: %s", outcome.Reason())
	}
}

// The catalog owns the drift, and it arrives as a whole number rather than as
// text. This is the first parameter of that kind, so the reader that takes it
// is held against what the catalog actually carries.
func TestTheDriftComesFromTheCatalogAsAWholeNumber(t *testing.T) {
	entry := entryOf(t, nationalIDVsBirthdate)
	params := defaultsOf(entry)
	years, whole := params.Whole(driftYears)
	if !whole {
		t.Fatalf("the declared default is %#v, which is not a whole number", params[driftYears])
	}
	if years == 0 {
		t.Error("the declared drift is nought, which would leave every value agreeing with its row")
	}
}

func TestDateDamageThatCannotBeDoneIsRefusedWithAReason(t *testing.T) {
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"the scheme encodes no date", mutator.Target{
			Value: "1234563218", Field: identifierField("PL.nip"), Provider: pl.NIP{},
			Params: mutator.Parameters{driftYears: 5}},
			"follows no numbering scheme that encodes a birth date"},
		{"no drift was given", mutator.Target{
			Value: "85011412345", Field: identifierField("PL.pesel"), Provider: pl.PESEL{},
			Params: mutator.Parameters{}},
			"nothing says how far to move the date"},
		{"the drift is a fraction", mutator.Target{
			Value: "85011412345", Field: identifierField("PL.pesel"), Provider: pl.PESEL{},
			Params: mutator.Parameters{driftYears: 2.5}},
			"nothing says how far to move the date"},
		{"the drift is nought", mutator.Target{
			Value: "85011412345", Field: identifierField("PL.pesel"), Provider: pl.PESEL{},
			Params: mutator.Parameters{driftYears: 0}},
			"is nought, so the identifier would keep agreeing"},
		{"no date can be read out of the value", mutator.Target{
			Value: "85011", Field: identifierField("PL.pesel"), Provider: pl.PESEL{},
			Params: mutator.Parameters{driftYears: 5}},
			"no birth date can be read out of this value"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			outcome := damage(t, nationalIDVsBirthdate, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %q", outcome.Value())
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}

// The same requirement as its sibling, and for the same reason: this mutator
// cannot see the column it is supposed to disagree with.
func TestDateCoherenceDamageRequiresARelationToDisagreeWith(t *testing.T) {
	entry := entryOf(t, nationalIDVsBirthdate)
	for _, needed := range entry.Requires {
		if needed == capability.SchemaFieldRelations {
			return
		}
	}
	t.Errorf("the entry requires %v, so a recipe with no birth date column would get this "+
		"applied and be told its rows are incoherent", entry.Requires)
}

// driftYears is the parameter name, written once here so that a test naming it
// wrongly fails to compile rather than silently passing no drift at all.
const driftYears = "drift_years"

// Contract check: the scheme used throughout this file really does encode a
// date, so these tests exercise the path they claim to.
var _ provider.DateEncoder = pl.PESEL{}
