package quick_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
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
		if about.Scheme != "" || about.CheckDigit || about.ReservedRange {
			t.Errorf("%s has no numbering scheme and its description claims scheme %q, "+
				"check digit %v, reserved range %v",
				generator, about.Scheme, about.CheckDigit, about.ReservedRange)
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
}

// The locale decides whether a kind can be produced at all, and the description
// has to say so rather than describing something this build cannot make.
func TestAKindWithoutDataInThisLocaleSaysWhyNot(t *testing.T) {
	source := values.New(quick.DefaultLocale, "")
	found := false
	for _, generator := range quick.Generators(quick.DefaultLocale) {
		fine, why := source.CanProduce(generator, quick.DefaultLocale)
		about := quick.Describe(generator, quick.DefaultLocale)
		if fine {
			if about.Unavailable != "" {
				t.Errorf("%s can be produced in %s and the description says it cannot: %q",
					generator, quick.DefaultLocale, about.Unavailable)
			}
			continue
		}
		found = true
		if about.Unavailable != why {
			t.Errorf("%s cannot be produced in %s because %q and the description says %q",
				generator, quick.DefaultLocale, why, about.Unavailable)
		}
	}
	if !found {
		t.Skip("every kind this build carries can be produced in the default locale, so there is " +
			"nothing here to describe as unavailable")
	}
}
