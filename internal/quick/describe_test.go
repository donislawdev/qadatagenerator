package quick_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/quick"
)

// What a kind of value CLAIMS has to be what its provider declares, and this is
// the pair that holds it.
//
// # Why the comparison is against the provider rather than against a list here
//
// A list written in a test is a second register, and the two would agree on the
// day it was written and drift afterwards. Reading the same providers the
// description reads would prove nothing at all, so the test reaches them by the
// other route: the register of schemes, walked directly, rather than through
// the name of a generator.
//
// # Why this matters more than most wiring
//
// One of these fields is a claim about whether a generated value could belong
// to a real person. A description saying a scheme has a reserved block when it
// does not is not a display fault, it is the tool telling somebody their test
// data is safe to paste into a system that matters.

func TestEveryDescribedSchemeSaysWhatItsProviderDeclares(t *testing.T) {
	source := values.New(quick.DefaultLocale, "")
	checked := 0
	for _, generator := range quick.Generators(quick.DefaultLocale) {
		scheme, behind := source.ProviderOf(generator)
		if !behind {
			continue
		}
		numbering, found := source.ProviderNamed(scheme)
		if !found {
			t.Errorf("%s is produced by the scheme %q and no provider answers to that name",
				generator, scheme)
			continue
		}

		declares := map[capability.Name]bool{}
		for _, name := range numbering.Capabilities() {
			declares[name] = true
		}

		about := quick.Describe(generator, quick.DefaultLocale)
		if about.Scheme != scheme {
			t.Errorf("%s is produced by %q and its description names %q",
				generator, scheme, about.Scheme)
		}
		if about.CheckDigit != declares[capability.IdentifierCheckDigit] {
			t.Errorf("%s says its check digit is %v and %s declares %v",
				generator, about.CheckDigit, scheme, declares[capability.IdentifierCheckDigit])
		}
		if about.ReservedRange != declares[capability.IdentifierReservedRange] {
			t.Errorf("%s says its reserved range is %v and %s declares %v. This is the field that "+
				"tells somebody whether a value could belong to a real person",
				generator, about.ReservedRange, scheme, declares[capability.IdentifierReservedRange])
		}
		unissued := false
		if says, saysSo := numbering.(provider.Unissued); saysSo {
			unissued = says.Unissued()
		}
		if about.Unissued != unissued {
			t.Errorf("%s says nobody issues its values is %v and %s declares %v",
				generator, about.Unissued, scheme, unissued)
		}
		// A block set aside by an issuer and no issuer at all cannot both be
		// true, and the panel would draw only one of them.
		if about.Unissued && about.ReservedRange {
			t.Errorf("%s claims a reserved range and no issuer at once", generator)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no generator in this build is produced by a numbering scheme, so the comparison " +
			"above walked nothing and proves nothing")
	}
}

// A kind with no numbering scheme behind it must not claim one, and must not
// claim the two facts that only a scheme can carry.
//
// 🔴 The zero value of a bool is false, so this passes for free today and would
// keep passing if the fields were never filled at all. What makes it worth
// keeping is the other half: the scheme has to be EMPTY, which the zero value
// also gives, and the generator has to be NAMED, which it does not.
func TestAKindWithNoSchemeClaimsNothingAboutOne(t *testing.T) {
	source := values.New(quick.DefaultLocale, "")
	for _, generator := range quick.Generators(quick.DefaultLocale) {
		if _, behind := source.ProviderOf(generator); behind {
			continue
		}
		about := quick.Describe(generator, quick.DefaultLocale)
		if about.Generator != generator {
			t.Errorf("a description of %q names %q", generator, about.Generator)
		}
		if about.Scheme != "" || about.CheckDigit || about.ReservedRange || about.Unissued {
			t.Errorf("%s has no numbering scheme and its description claims scheme %q, "+
				"check digit %v, reserved range %v, no issuer %v",
				generator, about.Scheme, about.CheckDigit, about.ReservedRange, about.Unissued)
		}
	}
}

// A name the register does not carry answers with a description that claims
// nothing, rather than with a panic or with a confident blank.
func TestAnUnknownKindIsDescribedAsClaimingNothing(t *testing.T) {
	about := quick.Describe("no-such-generator", quick.DefaultLocale)
	if about.Generator != "no-such-generator" {
		t.Errorf("the description does not carry the name it was asked about, it says %q", about.Generator)
	}
	if about.Produces != "" || about.Scheme != "" || about.CheckDigit || about.ReservedRange {
		t.Errorf("a generator this build does not have is described as producing %q under scheme %q",
			about.Produces, about.Scheme)
	}
	if about.Group.Kind != "" {
		t.Errorf("a generator this build does not have is shelved under %q", about.Group.Kind)
	}
}

// The account number stands under Poland, and says so, because Polish numbers
// are the only ones it produces (DEC-281). Named rather than found by the walk
// below, because the walk accepts a kind on either shelf as long as its facts
// agree, and the decision is about which shelf.
func TestTheAccountNumberStandsUnderTheOneCountryItProducesFor(t *testing.T) {
	about := quick.Describe("iban", quick.DefaultLocale)
	if about.Group != (quick.Group{Kind: quick.GroupCountry, Country: "PL"}) {
		t.Errorf("the account number stands on %+v rather than under Poland", about.Group)
	}
	if !slices.Equal(about.Countries, []string{"PL"}) {
		t.Errorf("the account number says it produces for %v rather than for Poland", about.Countries)
	}
}

// Every kind this build has stands on a shelf a list can name, and the shelf
// follows the scheme where there is one.
//
// # Why the country is checked against the scheme rather than against a list
//
// DEC-47 fixes the shape of a scheme identifier: a country code in ISO 3166-1
// alpha-2 before the dot, or no dot for an international standard. The group is
// read off that shape, so the test reads it off the same shape by the other
// route and holds the two together. A list of expected countries here would be
// a second register, and the next country would have to be added twice.
func TestEveryKindStandsOnAShelfAListCanName(t *testing.T) {
	for _, locale := range quick.Locales() {
		for _, generator := range quick.Generators(locale) {
			about := quick.Describe(generator, locale)
			switch about.Group.Kind {
			case quick.GroupCountry:
				if len(about.Group.Country) != 2 || about.Group.Country != strings.ToUpper(about.Group.Country) {
					t.Errorf("%s is shelved under the country %q, which is not an alpha-2 code", generator, about.Group.Country)
				}
				// The country comes from the scheme's identifier, or from
				// the one country a scheme of no country produces for.
				fromScheme := strings.HasPrefix(about.Scheme, about.Group.Country+".")
				fromValues := !strings.Contains(about.Scheme, ".") &&
					slices.Equal(about.Countries, []string{about.Group.Country})
				if !fromScheme && !fromValues {
					t.Errorf("%s is shelved under %s, its scheme is %q and it produces for %v",
						generator, about.Group.Country, about.Scheme, about.Countries)
				}
			case quick.GroupInternational:
				if about.Scheme == "" || strings.Contains(about.Scheme, ".") {
					t.Errorf("%s is shelved as international and its scheme is %q", generator, about.Scheme)
				}
				if len(about.Countries) == 1 {
					t.Errorf("%s is shelved as international and produces for %s alone",
						generator, about.Countries[0])
				}
			case quick.GroupPeople, quick.GroupPlaces, quick.GroupAccounts:
				if about.Scheme != "" {
					t.Errorf("%s is shelved under %s and still has the scheme %q", generator, about.Group.Kind, about.Scheme)
				}
			default:
				t.Errorf("%s in %s stands on no shelf a list can name: %+v", generator, locale, about.Group)
			}
		}
	}
}

// The locale decides whether a kind can be produced at all, and the description
// has to say so rather than describing something this build cannot make.
//
// Every locale rather than the default one since 2026-11-12, when the cities
// and regions of en-US got their data (DEC-278) and the default locale stopped
// lacking any. What still lacks it is the postal code of pl-PL (DEC-92).
func TestAKindWithoutDataInThisLocaleSaysWhyNot(t *testing.T) {
	found := false
	for _, locale := range quick.Locales() {
		source := values.New(locale, "")
		for _, generator := range quick.Generators(locale) {
			fine, why := source.CanProduce(generator, locale)
			about := quick.Describe(generator, locale)
			if fine {
				if about.Unavailable != "" {
					t.Errorf("%s can be produced in %s and the description says it cannot: %q",
						generator, locale, about.Unavailable)
				}
				continue
			}
			found = true
			if about.Unavailable != why {
				t.Errorf("%s cannot be produced in %s because %q and the description says %q",
					generator, locale, why, about.Unavailable)
			}
		}
	}
	// 🔴 A failure and not a skip (B-319). The day the last kind gets its data
	// is a good day, and it is also the day this guard stops guarding, so it
	// says so rather than going quiet.
	if !found {
		t.Fatal("every kind this build carries can be produced in every locale, so nothing " +
			"here is described as unavailable. This guard has nothing left to guard")
	}
}
