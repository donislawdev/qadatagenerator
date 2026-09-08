// Package quick answers the shortest question this tool takes: one value, from
// one generator, with no recipe and nothing written anywhere.
//
// # Why this is a package of its own rather than part of a surface
//
// It sits where the take and corrupt mode sits: above the use cases, below both
// surfaces, knowing neither. The layer test forbids the two surfaces to know
// each other, and the rule is not bookkeeping: a screen doing the same work
// from its own code is how one surface quietly becomes a reimplementation of
// the other. What is here is everything the command line and the window have to
// agree on, and none of it is a word meant for a person to read.
//
// # What this deliberately does not do
//
// It writes nothing, it plans nothing and it produces one value. Everything a
// recipe brings with it, the rows, the tables, the relations, the manifest and
// the files, is absent by definition rather than by omission: docs/07 defines
// this as the shortest path to one value, without a recipe and without any
// configuration.
package quick

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Entity is the name a value produced here is addressed under.
//
// # Why an address is needed at all for a single value
//
// Every value in this tool comes from a seed and an address, and there is no
// other way to draw one. A recipe gives the address a meaning, an entity and a
// column, and here there is neither, so one is chosen and written down. What
// matters is that it is FIXED: the same seed and the same generator have to
// give the same value, which is the promise the seed flag makes, and a made up
// address that moved would break it silently.
//
// The address is also what lets one oracle serve both this and a run of a
// recipe. A recipe with an entity of this name and one column named after the
// generator produces the same value, and the guard beside this package checks
// that rather than trusting the sentence.
const Entity = "quick"

// FirstRow and Purpose complete the address. The rows are counted from the
// first one, and the purpose is the one the engine uses for a value.
//
// 🔴 # Why more than one value is a ROW rather than a repeated draw
//
// Every value in this tool has an address of four parts, and the row is the
// part that means "another one of the same kind". Asking the engine twice at
// the same address would give the same value twice, and drawing from a fresh
// seed each time would break the promise the seed flag makes. Counting the row
// up is neither: it is the address a recipe of that many rows would use, so a
// run of ten here and a recipe of ten rows produce the same ten values, in the
// same order. The guard beside this package holds that rather than trusting
// this sentence.
const (
	FirstRow = 0
	Purpose  = "value"
)

// Request is one question.
type Request struct {
	// Generator is the name of the value wanted, such as pesel.
	Generator string
	// Locale is which locale's data to draw from. Empty means the default.
	Locale string
	// Seed makes the answer repeatable. Empty draws from a fresh one, which
	// is the one place in this tool where that happens, and the surfaces say
	// so where somebody can read it.
	Seed string
	// Agrees carries the values the answer has to be consistent with, keyed
	// by the role each plays. A role the chosen generator does not read is
	// refused rather than ignored.
	Agrees map[string]string
	// Count is how many values are wanted. It has no default here on purpose:
	// a zero that WORKS is worse than one that is refused, because a surface
	// that forgot to say would go on producing one value and nobody would find
	// out. Check refuses anything below one, so forgetting is loud.
	Count int
}

// Genders are the two spellings a surface OFFERS, sorted.
//
// # Why offering two and accepting twelve is not an inconsistency
//
// A recipe may write a gender twelve ways, because a column of test data is
// often copied out of a system that already had one and the digits of ISO 5218
// are what a database export gives. Everything that reads a gender reads all
// twelve, through one vocabulary in the domain, which is what DEC-214 settled.
//
// A list on a screen is a different question. It is a choice being offered
// rather than a value being read, and offering k, kobieta, 2 and female as four
// separate things to pick would suggest they differ. They do not.
//
// 🔴 Until 2026-09-06 this list was also what this mode ACCEPTED, and the
// reason recorded here was that a first name compared the bound value against
// the word female itself while a PESEL took all twelve. That defect is gone,
// so that reason is gone with it, and this list now means only what its name
// says.
func Genders() []string {
	return []string{string(provider.Female), string(provider.Male)}
}

// AcceptedGenders is every spelling a request may carry, sorted, for a refusal
// that has to name what it would have taken.
func AcceptedGenders() []string { return provider.GenderSpellings() }

// DateLayout is how a bound date of birth is written, which is what the scheme
// reading it accepts. DateShape is the same thing as a person writes it,
// which is the only form of it fit to put on a screen or in a message.
const (
	DateLayout = "2006-01-02"
	DateShape  = "YYYY-MM-DD"
)

// NewSeed draws a seed nobody chose.
//
// # Why this exists, when everything else in this tool is repeatable
//
// A person asking for a PESEL twice needs two of them, not the same one twice.
// The seventh rule says the same seed gives the same result, and it stays whole:
// this draws a DIFFERENT seed, and naming one brings the repeatability back
// entirely. It is the only place in this tool where a run is not repeatable by
// default, and both surfaces say so where it can be read rather than found out.
func NewSeed() (string, error) {
	// Sixteen bytes, because that is wider than anything that could collide by
	// accident and short enough to paste into a command when somebody wants
	// the same value again.
	drawn := make([]byte, 16)
	if _, err := rand.Read(drawn); err != nil {
		// Loud rather than falling back to a clock. A seed taken from the
		// clock would be repeatable by anybody who knows when the run was, and
		// this failing at all means the machine has no randomness, which is
		// news rather than something to work around.
		return "", fault.Wrap(fault.Runtime, err, "cannot draw a seed")
	}
	return hex.EncodeToString(drawn), nil
}

// The roles a value can be bound to, named here so that a surface talks to one
// package rather than reaching past this one into the domain for two constants.
const (
	RoleGender    = provider.RoleGender
	RoleBirthDate = provider.RoleBirthDate
)

// DefaultLocale is the locale a surface starts on when nobody has said.
//
// It is the one the mode itself falls back to, asked of the mode rather than
// spelled again on a screen, so that a screen and a command line cannot start
// somewhere different.
const DefaultLocale = "en-US"

// Locales are the locales this build carries data for, sorted.
func Locales() []string { return values.Locales() }

// Generators are the values this build can produce in that locale, sorted.
//
// Asked of the mode rather than listed on a screen, so that a screen offering a
// value this build cannot produce is not a thing that can be built.
func Generators(locale string) []string {
	return values.New(locale, "").Names()
}

// Reads reports whether that generator takes account of that role in that
// locale.
//
// 🔴 It takes a locale because the answer depends on one, which was measured
// rather than assumed: a surname reads the gender in Polish, where surnames
// inflect, and reads nothing in English. A surface asking without the locale
// would offer a field that does nothing for half of the runs.
func Reads(generator, role, locale string) bool {
	return values.New(locale, "").ReadsRole(generator, role)
}

// Run produces the values the request asks for, in order.
//
// Check above has to have passed. A request that would be refused reaches the
// engine here and is answered by whatever the engine says about it, which is a
// worse message than the refusal beside it and, for an unknown generator, a
// different exit code. The two surfaces both ask first, and the test beside
// this package holds that they cannot stop.
//
// # Why the values are not promised to differ from each other
//
// They are drawn, not dealt. Ten values of a generator whose data is a short
// list will repeat, and that is the same thing a recipe of ten rows does with
// the same seed. Uniqueness is a promise a recipe makes with a rule of its own,
// and making it quietly here would be a second spelling of it that nobody
// asked for. Both surfaces say so where it can be read.
func Run(req Request) ([]string, error) {
	source := values.New(req.Locale, req.Seed)
	produced := make([]string, 0, req.Count)
	for row := int64(FirstRow); row < int64(FirstRow)+int64(req.Count); row++ {
		value, err := source.Produce(generate.Request{
			Generator: req.Generator,
			At:        seed.Coordinate{Entity: Entity, Row: row, Field: req.Generator, Purpose: Purpose},
			Agrees:    provider.NewAgreement(req.Agrees),
		})
		if err != nil {
			return nil, err
		}
		produced = append(produced, value)
	}
	return produced, nil
}
