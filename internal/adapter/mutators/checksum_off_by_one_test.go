package mutators_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/intl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/pl"
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

const checksumOffByOne catalog.ID = "L3.checksum.off_by_one"

// schemesThisBuildHas are the numbering schemes with an implementation to check
// against, keyed by the identifier the catalog uses for them.
//
// It is written out because a test needs the instance and the registry of
// generators answers with a name. What stops it falling behind is the check
// below, which asks that registry what this build actually carries and holds the
// two lists against each other in both directions. A list kept by hand alone is
// a list somebody eventually forgets to extend, and this project has already
// replaced one of those.
var schemesThisBuildHas = map[string]provider.Provider{
	"PL.pesel":  pl.PESEL{},
	"PL.nip":    pl.NIP{},
	"PL.regon":  pl.REGON{},
	"US.ssn":    us.SSN{},
	"US.aba":    us.ABA{},
	"US.postal": us.Postal{},
	"iban":      intl.IBAN{},
}

func TestTheSchemesCheckedHereAreTheSchemesThisBuildCarries(t *testing.T) {
	carried := map[string]bool{}
	source := values.New("pl-PL", "closure")
	for _, generator := range source.Names() {
		if id, draws := source.ProviderOf(generator); draws {
			carried[id] = true
		}
	}
	if len(carried) == 0 {
		t.Fatal("no generator draws on a scheme, so this check read nothing")
	}
	for id := range carried {
		if _, listed := schemesThisBuildHas[id]; !listed {
			t.Errorf("%s is in this build and is checked by nothing here", id)
		}
	}
	for id := range schemesThisBuildHas {
		if !carried[id] {
			t.Errorf("%s is listed here and no generator draws on it any more", id)
		}
	}
}

// 🔴 This is the claim the entry makes, measured against each scheme rather
// than argued in a comment.
//
// The mutator declares a hard expectation of rejection, which is the strongest
// thing this tool can say about a value. What earns it is not the arithmetic in
// the mutator but this: a correct value from each scheme is damaged and that
// scheme's own validator, written from the published rules and not from what the
// generator happens to produce, is asked whether it still accepts it.
//
// A verdict of unsupported fails as surely as a valid one. It means the
// validator recognised a form it does not check, so nothing was rejected and
// nothing was approved, and a manifest promising rejection on the strength of it
// would be promising something nobody verified.
func TestEverySchemeThisDamageCoversStopsAcceptingTheDamagedValue(t *testing.T) {
	entry := entryOf(t, checksumOffByOne)
	checked := 0
	for _, id := range schemeNames() {
		if !entry.CoversProvider(id) {
			continue
		}
		checked++
		scheme := schemesThisBuildHas[id]
		t.Run(id, func(t *testing.T) {
			for row := int64(0); row < 25; row++ {
				correct := produce(t, scheme, row)
				outcome := damage(t, checksumOffByOne, mutator.Target{
					Value:  correct,
					Field:  identifierField(id),
					Params: defaultsOf(entry),
				})
				if !outcome.Applied() {
					t.Fatalf("%s was left alone: %s", correct, outcome.Reason())
				}
				verdict := scheme.Validate(outcome.Value())
				if verdict.Valid {
					t.Errorf("%s still accepts %s after %s became %s",
						id, outcome.Value(), correct, outcome.Value())
				}
				if verdict.Unsupported {
					t.Errorf("%s passed %s over rather than rejecting it: %s",
						id, outcome.Value(), verdict.Reason)
				}
			}
		})
	}
	if checked == 0 {
		t.Fatal("this damage covers no scheme this build carries, so nothing above ran")
	}
}

// 🔴 The other direction, and it is the reason the catalog names schemes at all.
//
// The social security number is an identifier like the others and carries no
// check digit, so changing a digit of one leaves a number its own rules still
// accept. One surviving value is enough to prove that, and it is what a hard
// expectation of rejection would have been written over had the engine been
// allowed to aim this damage at the column.
//
// Not every damaged value survives: the serial nought nought nought nought is
// invalid on its own account, so a value ending in nine can fall into it. That
// is why this asks for one survivor rather than for all of them.
func TestAnIdentifierWithNoCheckDigitSurvivesThisDamage(t *testing.T) {
	entry := entryOf(t, checksumOffByOne)
	if entry.CoversProvider("US.ssn") {
		t.Fatal("the catalog now covers a scheme with no check digit to break")
	}
	scheme := schemesThisBuildHas["US.ssn"]
	survived := 0
	for row := int64(0); row < 25; row++ {
		correct := produce(t, scheme, row)
		outcome := damage(t, checksumOffByOne, mutator.Target{
			Value: correct, Field: identifierField("US.ssn"), Params: defaultsOf(entry),
		})
		if scheme.Validate(outcome.Value()).Valid {
			survived++
		}
	}
	if survived == 0 {
		t.Error("every damaged number was rejected, so this scheme behaves as though it had a check digit")
	}
}

// 🔴 The first published example this build can produce rather than only check.
//
// The architecture turns every before and after pair into a test case so that
// the specification cannot drift away from the behaviour. Until there was an
// engine, only half of that was possible: the values could be held against the
// scheme they belong to, but nothing could produce the second from the first.
// This entry is the first where the whole pair runs.
func TestTheCatalogExampleIsWhatThisMutatorProduces(t *testing.T) {
	entry := entryOf(t, checksumOffByOne)
	outcome := damage(t, checksumOffByOne, mutator.Target{
		Value:  entry.Example.Before,
		Field:  identifierField("PL.nip"),
		Params: defaultsOf(entry),
	})
	if !outcome.Applied() {
		t.Fatalf("the published example was left alone: %s", outcome.Reason())
	}
	if outcome.Value() != entry.Example.After {
		t.Errorf("the specification promises %s -> %s and this build produces %s",
			entry.Example.Before, entry.Example.After, outcome.Value())
	}
}

// Nine wraps round to nought, so there is one rule rather than a digit that
// sometimes counts up and sometimes down.
func TestTheDigitAfterNineIsNought(t *testing.T) {
	outcome := damage(t, checksumOffByOne, mutator.Target{
		Value:  "1234563219",
		Field:  identifierField("PL.nip"),
		Params: mutator.Parameters{"position": "last"},
	})
	if !outcome.Applied() || outcome.Value() != "1234563210" {
		t.Errorf("a value ending in nine became %q: applied=%v", outcome.Value(), outcome.Applied())
	}
}

// Every one of these would otherwise be damage recorded as done over a value
// nothing had changed, or a digit invented out of a character that is not one.
func TestChecksumDamageThatCannotBeDoneIsRefusedWithAReason(t *testing.T) {
	cases := []struct {
		what   string
		target mutator.Target
		says   string
	}{
		{"no position given", mutator.Target{
			Value: "1234563218", Params: mutator.Parameters{}},
			"nothing says which digit"},
		{"a position this build cannot aim at", mutator.Target{
			Value: "1234563218", Params: mutator.Parameters{"position": "first"}},
			`"first" is not a position this build can change`},
		{"the value is empty", mutator.Target{
			Value: "", Params: mutator.Parameters{"position": "last"}},
			"the value is empty"},
		// A real shape rather than an invented one: the check digit of a ten
		// digit book number is the remainder modulo eleven, and the remainder of
		// ten is written X. There is nothing there to add one to, and writing Y
		// beside it would be this tool inventing a character the scheme has no
		// use for.
		{"the value does not end in a digit", mutator.Target{
			Value: "080442957X", Params: mutator.Parameters{"position": "last"}},
			"does not end in a digit"},
		// A digit of another script is a digit to a reader and not to this
		// arithmetic. Adding one to it would produce a character from a
		// calculation that means nothing in that script.
		{"the value ends in a digit of another script", mutator.Target{
			Value: "٩٠٠٤١١٥٨٠٢٨", Params: mutator.Parameters{"position": "last"}},
			"does not end in a digit"},
	}
	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			one.target.Field = identifierField("PL.nip")
			outcome := damage(t, checksumOffByOne, one.target)
			if outcome.Applied() {
				t.Fatalf("damage was reported as done, and the value became %q", outcome.Value())
			}
			if outcome.Reason() == "" {
				t.Fatal("nothing happened and nothing says why, which is the silence this forbids")
			}
			if !strings.Contains(outcome.Reason(), one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, outcome.Reason())
			}
		})
	}
}

// entryOf reads one entry out of the catalog this build ships.
func entryOf(t *testing.T, id catalog.ID) catalog.Entry {
	t.Helper()
	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	entry, ok := loaded.Lookup(id)
	if !ok {
		t.Fatalf("%s is implemented and the catalog does not describe it", id)
	}
	return entry
}

// defaultsOf takes the knobs the catalog declares, so that no default is written
// down twice. A copy here would be a published value with a second home.
func defaultsOf(entry catalog.Entry) mutator.Parameters {
	params := make(mutator.Parameters, len(entry.Parameters))
	for _, declared := range entry.Parameters {
		params[declared.Name] = declared.Default
	}
	return params
}

// produce makes one correct value of a scheme.
//
// Computed rather than written down. A valid identifier can belong to a real
// person or body, and a handful of them sitting in a repository that is going to
// be public reads like a curated list.
func produce(t *testing.T, scheme provider.Provider, row int64) string {
	t.Helper()
	value, err := scheme.Generate(provider.Request{
		Source: seed.New("checksum corpus"),
		At:     seed.Coordinate{Entity: "corrupt", Row: row, Field: "identifier", Purpose: "value"},
	})
	if err != nil {
		t.Fatalf("row %d: %s refused to produce a correct value: %v", row, scheme.ID(), err)
	}
	return value
}

// identifierField is the column such a value sits in, as the engine sees it.
func identifierField(scheme string) recipe.FieldInfo {
	return recipe.FieldInfo{Name: "identifier", Type: "identifier", Provider: scheme}
}

// schemeNames returns the schemes in a fixed order, because Go randomises map
// iteration and a test that reports its findings in a different order every run
// is harder to read than one that does not.
func schemeNames() []string {
	names := make([]string, 0, len(schemesThisBuildHas))
	for id := range schemesThisBuildHas {
		names = append(names, id)
	}
	sort.Strings(names)
	return names
}
