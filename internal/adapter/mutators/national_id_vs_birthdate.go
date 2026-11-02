package mutators

import (
	"fmt"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// nationalIDVsBirthdateID is the catalog entry this implements.
const nationalIDVsBirthdateID catalog.ID = "L4.coherence.national_id_vs_birthdate"

// driftParameter is the knob the entry declares, in years. Its default lives in
// the catalog and arrives already applied.
const driftParameter = "drift_years"

// nationalIDVsBirthdate leaves a correct identifier encoding a birth date the
// column beside it does not carry.
//
// It is the sibling of the gender one and differs in the part that matters: a
// gender has two values and turning it round is the only move there is, while a
// date has as many as the calendar and the entry has to say how far to go. That
// is why this one carries a knob and that one does not.
//
// # Why the whole date does not move
//
// Only the year. The day and month stay where they were, so a reader comparing
// the identifier with the column beside it sees one difference rather than
// three, and a test written against the manifest can say what it expected. It
// also keeps the value inside the same part of the calendar, which matters for
// the case below.
type nationalIDVsBirthdate struct{}

// ID is the catalog identifier.
func (nationalIDVsBirthdate) ID() catalog.ID { return nationalIDVsBirthdateID }

// Damage rebuilds the identifier around a birth date some years away.
func (nationalIDVsBirthdate) Damage(target mutator.Target) mutator.Outcome {
	encoder, encodes := target.Provider.(provider.DateEncoder)
	if !encodes {
		return mutator.Skipped(
			"this column follows no numbering scheme that encodes a birth date, so there is none to move")
	}
	years, given := target.Params.Whole(driftParameter)
	if !given {
		return mutator.Skipped(fmt.Sprintf(
			"no whole number of %s was given, so nothing says how far to move the date", driftParameter))
	}
	if years == 0 {
		// A drift of nothing rebuilds the same value, and an entry claiming a
		// disagreement that is not there sends a test looking for one.
		return mutator.Skipped(fmt.Sprintf(
			"%s is nought, so the identifier would keep agreeing with the column beside it",
			driftParameter))
	}
	carried, readable := encoder.EncodedDate(target.Value)
	if !readable {
		return mutator.Skipped("no birth date can be read out of this value, so there is none to move")
	}

	moved := provider.Date{Year: carried.Year + years, Month: carried.Month, Day: carried.Day}
	rebuilt, err := encoder.WithDate(target.Value, moved)
	if err != nil {
		// 🔴 The twenty ninth of February is the case this exists for. Moved by
		// a whole number of years it lands on a day that is not in the calendar
		// three times out of four, and the scheme refuses to write one. Refusing
		// here is the honest answer: choosing a different drift instead would do
		// something the policy did not ask for, and choosing the nearest real
		// day would quietly change what the manifest promises.
		if !isDay(moved) {
			return mutator.Skipped(fmt.Sprintf(
				"moving the birth date by %d years lands on %04d-%02d-%02d, which is not a day",
				years, moved.Year, moved.Month, moved.Day))
		}
		// A day the calendar has and the scheme refused is a day it has no way
		// to write, such as one past the last century a PESEL covers. Until
		// 2026-11-02 the reason above was given for those too, so the manifest
		// said of a real day that it was not one.
		return mutator.Skipped(fmt.Sprintf(
			"moving the birth date by %d years lands on %04d-%02d-%02d, which the numbering scheme cannot write",
			years, moved.Year, moved.Month, moved.Day))
	}
	if rebuilt == target.Value {
		return mutator.Skipped(fmt.Sprintf(
			"moving the birth date by %d years left the value where it was", years))
	}
	return mutator.Done(rebuilt)
}

// isDay reports whether the calendar has that date. The standard library moves
// a day it does not have into the next month, so a date that comes back changed
// was not one.
func isDay(date provider.Date) bool {
	asked := time.Date(date.Year, time.Month(date.Month), date.Day, 0, 0, 0, 0, time.UTC)
	return asked.Year() == date.Year && int(asked.Month()) == date.Month && asked.Day() == date.Day
}
