package quick

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
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
	// Masks are the written forms of a value of this scheme, shortest first,
	// none for a kind that has no scheme behind it. Almost every scheme has
	// one. The payment card has two, sixteen digits and fifteen.
	//
	// # Why a surface is given the mask rather than a sentence about it
	//
	// Because the mask is the fact and the sentence is a rendering of it. A
	// window shows the mask itself - somebody comparing it against the values
	// beside it learns the shape faster than from any wording - and works out
	// what to say about it with Places below. The wording is then a matter for
	// the language of the interface, which is where wording belongs.
	Masks []string
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
	// Group is the shelf a surface listing every kind puts this one on.
	Group Group
}

// Group is where a kind of value stands in a list of all of them.
//
// # Why the mode decides this and not the window
//
// Because it is read off facts the mode already holds. A scheme carries a
// country in its identifier or carries none (DEC-47), and a generator with no
// scheme declares a topic in the register (values.TopicOf). A window sorting
// kinds into headings by its own reading of the names would be a second copy
// of that rule, and the two would part the day a scheme arrived whose name did
// not look like the ones before it.
//
// The words for a group are the window's, since a heading is read in the
// language of the interface. The mode hands over a kind and a code, never a
// sentence.
type Group struct {
	// Kind is one of the four constants below, or empty for a name this build
	// does not know.
	Kind string
	// Country is the ISO 3166-1 alpha-2 code the scheme carries, for
	// GroupCountry only.
	Country string
}

// The kinds of group.
const (
	// GroupCountry is a kind whose scheme belongs to one country.
	GroupCountry = "country"
	// GroupInternational is a kind whose scheme belongs to no country.
	GroupInternational = "international"
	// GroupPeople is a kind with no scheme that describes a person.
	GroupPeople = "people"
	// GroupPlaces is a kind with no scheme that describes a place.
	GroupPlaces = "places"
)

// groupOf reads the shelf off a scheme, or off a declared topic where there is
// no scheme.
func groupOf(scheme string, topic string) Group {
	if scheme != "" {
		if dot := strings.Index(scheme, "."); dot > 0 {
			return Group{Kind: GroupCountry, Country: scheme[:dot]}
		}
		return Group{Kind: GroupInternational}
	}
	switch topic {
	case values.TopicPeople:
		return Group{Kind: GroupPeople}
	case values.TopicPlaces:
		return Group{Kind: GroupPlaces}
	}
	return Group{}
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
	topic, _ := source.TopicOf(generator)
	about.Group = groupOf(scheme, topic)
	if !behind {
		return about
	}
	about.Scheme = scheme
	numbering, found := source.ProviderNamed(scheme)
	if !found {
		return about
	}
	if shaped, saysSo := numbering.(provider.Shaped); saysSo {
		about.Masks = shaped.Masks()
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

// Places counts what varies in a mask: how many characters a person would have
// to fill in, and whether any of them may be a letter.
//
// # Why the mode counts this and not the window
//
// Because it is a reading of a mask, and a mask has to be read the same way
// wherever it is read. Two surfaces counting characters separately would be two
// readings, and they would part company the first time a mask gained a
// separator - one of them would call it a place to fill in and the other would
// not, and both would look right.
//
// The window turns the pair into words, which is the half that depends on the
// language somebody reads.
func Places(mask string) (count int, letters bool) {
	for i := 0; i < len(mask); i++ {
		switch mask[i] {
		case provider.MaskDigit:
			count++
		case provider.MaskAlnum:
			count, letters = count+1, true
		}
	}
	return count, letters
}
