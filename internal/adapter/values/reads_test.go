package values

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// TestEveryGeneratorReadsExactlyTheRolesTheRegisterSays closes the register of
// consistency roles from both sides, by measurement rather than by trust.
//
// # Why this is not a list compared against a list
//
// Nothing in a provider declares which roles it reads, so a second written list
// would only say that somebody typed the same thing twice. What can be observed
// is the behaviour: bind two different values for one role at one address, and
// a generator that reads that role produces two different values while one that
// ignores it produces the same value twice.
//
// So the register is a claim about behaviour and this is the measurement of it.
// A generator that starts reading a role without the register being told fails
// here, and so does an entry claiming a role nothing reads.
//
// # What would make this test lie
//
// A generator that reads a role and happens to produce the same value for both
// bound values. That is why several addresses are tried rather than one: one
// draw is an anecdote, and this project has already measured what a single
// draw is worth. A generator that agreed with itself at all of them would be
// one that reads the role and ignores what it says, which is a defect this
// would report in the right direction anyway.
func TestEveryGeneratorReadsExactlyTheRolesTheRegisterSays(t *testing.T) {
	// Two values far apart for each role, so that a generator taking any
	// account of one has to move.
	contrasts := map[string][2]string{
		provider.RoleGender:    {"female", "male"},
		provider.RoleBirthDate: {"1950-01-31", "2001-12-01"},
		// Chicago lies in Illinois alone and Houston almost always in Texas,
		// so a region taking any account of the city has to move (DEC-278).
		provider.RoleCity: {"Chicago", "Houston"},
	}

	// Every locale, because the register answers per locale and a surname
	// reads the gender in one of them and not in the other. Measuring one
	// would have blessed half the register.
	for _, locale := range Locales() {
		measureOneLocale(t, locale, contrasts)
	}
}

func measureOneLocale(t *testing.T, locale string, contrasts map[string][2]string) {
	t.Helper()
	source := New(locale, "reads-register")
	for _, generator := range source.Names() {
		for role, pair := range contrasts {
			moved := false
			for row := int64(0); row < 8; row++ {
				at := seed.Coordinate{Entity: "quick", Row: row, Field: generator, Purpose: "value"}
				first, firstErr := source.Produce(generate.Request{
					Generator: generator, At: at,
					Agrees: provider.NewAgreement(map[string]string{role: pair[0]}),
				})
				second, secondErr := source.Produce(generate.Request{
					Generator: generator, At: at,
					Agrees: provider.NewAgreement(map[string]string{role: pair[1]}),
				})
				if firstErr != nil || secondErr != nil {
					// A generator that cannot be produced in this locale says
					// nothing about which roles it reads, so it is skipped
					// rather than counted as reading none.
					moved = source.ReadsRole(generator, role)
					break
				}
				if first != second {
					moved = true
					break
				}
			}

			claimed := source.ReadsRole(generator, role)
			if moved && !claimed {
				t.Errorf("in %s, %s changes when %s is bound and the register does not say it reads it. "+
					"A caller is told the value ignores that role and it does not",
					locale, generator, role)
			}
			if claimed && !moved {
				t.Errorf("in %s, %s claims to read %s and produces the same value whichever way it is "+
					"bound. The register promises an influence nothing has", locale, generator, role)
			}
		}
	}
}

// TestARoleTheRegisterNamesIsARoleThatExists holds the other half: a role
// spelled wrongly in the register would be a rule about nothing, and the
// measurement above cannot see it, because nothing reads a role that is not a
// role either.
func TestARoleTheRegisterNamesIsARoleThatExists(t *testing.T) {
	known := map[string]bool{provider.RoleBirthDate: true, provider.RoleGender: true, provider.RoleCity: true}
	for _, locale := range Locales() {
		source := New(locale, "")
		for _, generator := range source.Names() {
			for _, role := range source.RolesRead(generator) {
				if !known[role] {
					t.Errorf("in %s, %s is registered as reading %q, which is not a role a recipe "+
						"can bind", locale, generator, role)
				}
			}
		}
	}
}

// 🔴 TestEveryValueABoundGeneratorDrawsIsOneItsReadersTake holds the reason an
// identifier may refuse a bound value without saying what kind of failure it is.
//
// The scheme behind a PESEL refuses a bound date or gender it cannot read with
// a bare error, and a bare error reaches every surface as a defect of this tool.
// That is true only while nothing a person writes gets there. A person typing a
// value goes through the refusals of quick first, and a recipe binds only what
// a generator playing the role drew, because a list under values plays no role.
// Measured on 2026-11-03 on the binary (B-336): a gender of other and a date of
// 2023-02-30 from such a list were generated beside the PESEL with a warning that
// the two are independent, and neither reached it.
//
// So every value those generators draw is handed to every generator reading the
// role. One that starts drawing a value a reader cannot take fails here, rather
// than as a defect reported from somebody's run.
func TestEveryValueABoundGeneratorDrawsIsOneItsReadersTake(t *testing.T) {
	for _, locale := range Locales() {
		source := New(locale, "bound-values")
		for _, bound := range source.Names() {
			role, plays := source.RoleOf(bound)
			if !plays {
				continue
			}
			for _, reader := range source.Names() {
				if source.ReadsRole(reader, role) {
					bindEveryDraw(t, source, locale, bound, role, reader)
				}
			}
		}
	}
}

// boundRows is how many values of a bound generator are handed on. Enough for
// the dates to land in every year of their span many times over, which is where
// a value a scheme cannot carry would show first.
const boundRows = 2000

func bindEveryDraw(t *testing.T, source Source, locale, bound, role, reader string) {
	t.Helper()
	at := func(field string, row int64) seed.Coordinate {
		return seed.Coordinate{Entity: "bound", Row: row, Field: field, Purpose: "value"}
	}
	if _, err := source.Produce(generate.Request{Generator: reader, At: at(reader, 0)}); err != nil {
		// A generator this locale cannot produce at all is reached by no
		// binding here either, so there is nothing to hand it.
		return
	}
	for row := int64(0); row < boundRows; row++ {
		value, err := source.Produce(generate.Request{Generator: bound, At: at(bound, row)})
		if err != nil {
			t.Errorf("in %s, %s could not be drawn at row %d: %v", locale, bound, row, err)
			return
		}
		_, err = source.Produce(generate.Request{
			Generator: reader, At: at(reader, row),
			Agrees: provider.NewAgreement(map[string]string{role: value}),
		})
		if err != nil {
			t.Errorf("in %s, %s drew %q as the %s, and %s bound to it refused it: %v",
				locale, bound, value, role, reader, err)
			return
		}
	}
}
