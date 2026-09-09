package quick

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
)

// Description is what this build knows ABOUT a kind of value, as opposed to how
// to produce one.
//
// # Why this lives in the mode rather than on the screen that shows it
//
// Every field below is a fact about the product, and one of them is a claim
// about whether a generated value could belong to somebody. A window working
// that out for itself would be product logic inside a surface, which the
// dependency rule forbids for a reason this answer makes concrete: the same
// question has one answer, and two surfaces asking it separately would drift
// the first time a scheme changed what it declares.
//
// # Why every field is DERIVED and none of it is written down
//
// A register of sentences about each scheme would be prose, and prose is the
// one thing in this project that nothing checks. Each of these is read from
// what the provider itself declares, so a scheme that stops carrying a check
// digit stops saying it does, in the same commit, without anybody remembering
// to edit a description.
type Description struct {
	// Generator is the name asked about, so a caller holding several of these
	// does not have to remember which is which.
	Generator string
	// Produces is the kind of column this makes, in the vocabulary of the
	// recipe format rather than of any one surface.
	Produces string
	// Scheme is the numbering scheme behind it, empty where there is none.
	// A first name has no scheme. A PESEL is PL.pesel.
	Scheme string
	// CheckDigit says the value carries a digit computed from the rest of it,
	// so that one wrong character can be caught by the number alone.
	CheckDigit bool
	// ReservedRange says this build generates inside a block the issuer never
	// assigns.
	//
	// 🔴 It is false for every scheme that does not DECLARE it, and false is
	// the honest answer rather than a gap. Two schemes here narrow what they
	// generate without claiming a harbour, because the block they narrow to is
	// one codes are still issued from, and reading that narrowing as a promise
	// is exactly the mistake the providers refuse to invite. Whoever renders
	// this must say what false means rather than leaving it out, since a
	// missing line reads as reassurance.
	ReservedRange bool
	// ReadsGender and ReadsBirthDate are the settings this kind takes account
	// of, which is why a surface offers those fields beside it.
	ReadsGender    bool
	ReadsBirthDate bool
	// Unavailable is why this build cannot produce the kind in the locale
	// asked about, and is empty when it can.
	Unavailable string
}

// Describe answers what this build knows about one kind of value in one locale.
//
// An unknown name answers with a zero description carrying the name, rather
// than with a second return value nobody would check. A surface asking about a
// generator that is not in the register has already gone wrong somewhere the
// list of names came from, and the honest thing on screen is a kind that
// claims nothing at all.
func Describe(generator, locale string) Description {
	source := values.New(locale, "")
	about := Description{
		Generator:      generator,
		ReadsGender:    source.ReadsRole(generator, RoleGender),
		ReadsBirthDate: source.ReadsRole(generator, RoleBirthDate),
	}
	if !source.Known(generator) {
		return about
	}
	about.Produces, _ = source.TypeOf(generator)
	if fine, why := source.CanProduce(generator, locale); !fine {
		about.Unavailable = why
	}

	scheme, behind := source.ProviderOf(generator)
	if !behind {
		return about
	}
	about.Scheme = scheme
	numbering, found := source.ProviderNamed(scheme)
	if !found {
		return about
	}
	for _, declared := range numbering.Capabilities() {
		switch declared {
		case capability.IdentifierCheckDigit:
			about.CheckDigit = true
		case capability.IdentifierReservedRange:
			about.ReservedRange = true
		}
	}
	return about
}
