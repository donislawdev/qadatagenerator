package quick

import (
	"sort"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// RefusalKind is a request this mode will not carry out, named rather than
// worded.
//
// # Why a kind and not the sentence itself
//
// The same reason the mode next door gives: the command line is English always,
// by an unbreakable rule, while the window shows whatever it was translated
// into and takes its words from a catalogue of keys. A shared sentence would
// force one of the two to be wrong, and the one it would break is the window,
// quietly, by putting an untranslated string on a translated screen.
//
// What both surfaces share is which requests are refused and why, which is what
// a kind carries. Each surface then owns its own four parts.
type RefusalKind int

const (
	// NotRefused is the zero value, so that a refusal nobody filled in cannot
	// be mistaken for a real one.
	NotRefused RefusalKind = iota
	// UnknownGenerator names a value this build cannot produce.
	UnknownGenerator
	// RoleNotRead binds a value to something the chosen generator takes no
	// account of, such as a gender beside a city.
	RoleNotRead
	// UnreadableGender is a gender outside the spellings this mode takes.
	UnreadableGender
	// UnreadableDate is a date of birth that is not written as YYYY-MM-DD.
	UnreadableDate
)

// Refusal is why a value will not be produced, with what it needs to be said
// properly.
//
// Value and Allowed are here for the reason every refusal in this project
// carries them: a message has to name the offending value and what would have
// been taken instead, and working that out again on each surface is how the two
// come to disagree.
type Refusal struct {
	// Kind is what was wrong.
	Kind RefusalKind
	// Value is the offending value.
	Value string
	// Role is which role was bound, for a refusal about one.
	Role string
	// Locale is named only where it is what makes the answer no, which is a
	// role read in one locale and not in another.
	Locale string
	// Allowed is what this build would have taken: the generators it has, the
	// spellings of a gender, or the generators that do read the role.
	Allowed []string
}

// Refused reports whether this is a refusal at all.
func (r Refusal) Refused() bool { return r.Kind != NotRefused }

// Check says whether this request will be carried out, before anything is
// drawn.
//
// # Why a bound role nothing reads is refused rather than ignored
//
// A recipe binds a rule across a whole table and some columns of that table act
// on it, so a column that does not is ordinary. Here there is one value, so a
// bound role the generator ignores has no other column to be for. Accepting it
// would answer a question nobody asked and say nothing about it, which is the
// silence this product refuses everywhere else.
//
// # Why the gender is checked here at all
//
// The vocabulary is the domain's and every reader of a gender shares it, so
// this is not a second opinion about what the words mean. What it adds is WHEN
// the answer comes: a word this product has no meaning for is refused before
// anything is drawn, with the accepted spellings named, rather than reaching a
// scheme that refuses it in its own words or a word list that quietly draws
// from both.
//
// 🔴 This mode took two spellings until 2026-09-06, and the reason was that a
// first name compared the bound value against the word female itself while a
// PESEL took all twelve, so a narrower verb was one that did not inherit the
// disagreement. DEC-214 removed the disagreement, so the narrowness lost its
// reason and went with it.
func Check(req Request) Refusal {
	source := values.New(req.Locale, req.Seed)
	if !source.Known(req.Generator) {
		return Refusal{Kind: UnknownGenerator, Value: req.Generator, Allowed: source.Names()}
	}
	for _, role := range sortedRoles(req.Agrees) {
		if !source.ReadsRole(req.Generator, role) {
			return Refusal{
				Kind: RoleNotRead, Value: req.Generator, Role: role,
				Locale: req.Locale, Allowed: readersOf(source, role),
			}
		}
	}
	if bound, ok := req.Agrees[provider.RoleGender]; ok {
		if _, known := provider.ParseGender(bound); !known {
			return Refusal{Kind: UnreadableGender, Value: bound, Role: provider.RoleGender,
				Allowed: AcceptedGenders()}
		}
	}
	if bound, ok := req.Agrees[provider.RoleBirthDate]; ok {
		if _, err := time.Parse(DateLayout, bound); err != nil {
			// No Allowed here, and that is the rule of this file working: the
			// layout a person reads is YYYY-MM-DD, which is a word for a
			// person, and DateLayout is how Go spells the same thing. Each
			// surface writes the readable one in its own words.
			return Refusal{Kind: UnreadableDate, Value: bound, Role: provider.RoleBirthDate}
		}
	}
	return Refusal{}
}

// readersOf names the generators that would act on that role, so that a refusal
// points somewhere rather than only saying no.
func readersOf(source values.Source, role string) []string {
	var readers []string
	for _, generator := range source.Names() {
		if source.ReadsRole(generator, role) {
			readers = append(readers, generator)
		}
	}
	return readers
}

func sortedRoles(agrees map[string]string) []string {
	roles := make([]string, 0, len(agrees))
	for role := range agrees {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}
