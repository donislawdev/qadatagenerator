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
	known := map[string]bool{provider.RoleBirthDate: true, provider.RoleGender: true}
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
