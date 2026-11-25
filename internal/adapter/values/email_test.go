package values_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// dotAtom is the local part this generator promises: lowercase letters and
// digits in runs joined by single dots or underscores, never a dot at either
// end, which RFC 5322 section 3.2.3 allows as a dot-atom and which forms accept.
var dotAtom = regexp.MustCompile(`^[a-z0-9]+([._][a-z0-9]+)*$`)

// reserved are the domains the owner chose on 2026-11-25, written here a second
// time so that a domain slipping into the generator turns this red.
var reserved = []string{"example.com", "example.net", "example.org"}

// splitAddress is the local part and the domain of an address, failing on
// anything that is not one at sign between two non empty parts.
func splitAddress(t *testing.T, address string) (string, string) {
	t.Helper()
	local, domain, found := strings.Cut(address, "@")
	if !found || local == "" || strings.Contains(domain, "@") {
		t.Fatalf("%q is not one local part and one domain", address)
	}
	if !dotAtom.MatchString(local) {
		t.Fatalf("the local part of %q is not a dot-atom of letters and digits", address)
	}
	if !slices.Contains(reserved, domain) {
		t.Fatalf("%q is at %s, which is not one of the reserved domains %v", address, domain, reserved)
	}
	return local, domain
}

// An address bound to the name of its row is made of that name written in
// plain letters: marks taken off, ł as l, an apostrophe and a hyphen left out.
func TestAnAddressIsMadeOfTheNameOfItsRow(t *testing.T) {
	source := values.New("pl-PL", "emails")
	cases := []struct{ given, surname, first, last string }{
		{"Łucja", "Wróbel", "lucja", "wrobel"},
		{"Żaneta", "Gąsiorowska", "zaneta", "gasiorowska"},
		{"Liam", "D'Arcy", "liam", "darcy"},
		{"Ann", "Smith-Jones", "ann", "smithjones"},
	}
	for _, each := range cases {
		for row := range int64(40) {
			req := ask(values.Email, "email", row)
			req.Agrees = provider.NewAgreement(map[string]string{
				provider.RoleGivenName: each.given, provider.RoleSurname: each.surname,
			})
			address, err := source.Produce(req)
			if err != nil {
				t.Fatalf("%s %s, row %d: %v", each.given, each.surname, row, err)
			}
			local, _ := splitAddress(t, address)
			if !strings.Contains(local, each.last) {
				t.Fatalf("%s %s gave %q, which leaves out the surname %s", each.given, each.surname, address, each.last)
			}
			if !strings.Contains(local, each.first) && !strings.HasPrefix(local, each.first[:1]+each.last) {
				t.Fatalf("%s %s gave %q, which leaves out the given name %s", each.given, each.surname, address, each.first)
			}
		}
	}
}

// A row with no name to follow gets an address of a name drawn for it, in
// every shape, at every reserved domain, with and without digits, the same at
// the same place.
func TestAnAddressWithNoNameToFollowDrawsOne(t *testing.T) {
	for _, locale := range []string{"en-US", "pl-PL"} {
		source := values.New(locale, "emails")
		domains := map[string]int{}
		separators := map[string]int{}
		numbered := 0
		for row := range int64(2000) {
			req := ask(values.Email, "email", row)
			address, err := source.Produce(req)
			if err != nil {
				t.Fatalf("%s, row %d: %v", locale, row, err)
			}
			if again, _ := source.Produce(req); again != address {
				t.Fatalf("%s, row %d gave %q and then %q at the same place", locale, row, address, again)
			}
			local, domain := splitAddress(t, address)
			domains[domain]++
			switch {
			case strings.Contains(local, "."):
				separators["."]++
			case strings.Contains(local, "_"):
				separators["_"]++
			default:
				separators[""]++
			}
			if strings.ContainsAny(local[len(local)-1:], "0123456789") {
				numbered++
			}
		}
		if len(domains) != len(reserved) {
			t.Errorf("%s: 2000 addresses used %v, and all of %v were wanted", locale, domains, reserved)
		}
		if len(separators) != 3 {
			t.Errorf("%s: 2000 addresses joined their names only by %v", locale, separators)
		}
		if numbered < 600 || numbered > 1400 {
			t.Errorf("%s: %d of 2000 addresses end in digits, and about half were wanted", locale, numbered)
		}
	}
}

// max_length cuts the local part, never into a dot at its end, down to the
// shortest address, and below that the field is refused before a row is
// written with the sentence the draw would give.
func TestMaxLengthCutsAnAddressAndBelowTheShortestItIsRefused(t *testing.T) {
	source := values.New("en-US", "emails")
	for _, limit := range []int{13, 15, 18, 24} {
		for row := range int64(300) {
			req := ask(values.Email, "email", row)
			req.HasMaxLength, req.MaxLength, req.Unit = true, limit, "characters"
			address, err := source.Produce(req)
			if err != nil {
				t.Fatalf("max_length %d, row %d: %v", limit, row, err)
			}
			if len(address) > limit {
				t.Fatalf("max_length %d, row %d: %q is %d characters", limit, row, address, len(address))
			}
			splitAddress(t, address)
		}
	}
	short := ask(values.Email, "email", 0)
	short.HasMaxLength, short.MaxLength, short.Unit = true, 12, "characters"
	if _, err := source.Produce(short); err == nil || !strings.Contains(err.Error(), "at least 13") {
		t.Errorf("max_length 12 was answered with %v, which does not say an address needs 13", err)
	}
	if domain, known := source.DomainOf(short); !known || domain.Size != 0 {
		t.Errorf("below the shortest address the plan is told known=%v and %d values", known, domain.Size)
	}
	if _, known := source.DomainOf(ask(values.Email, "email", 0)); known {
		t.Error("an unlimited address claims a domain the plan could count")
	}
}

// A recipe with a first name, a last name and an email needs no rule for the
// address to follow the name: the run keeps one the recipe did not write, the
// way a first name follows the gender column.
func TestAnEmailFollowsTheNameColumnsWithoutARule(t *testing.T) {
	source := values.New("en-US", "emails")
	entity := recipe.Entity{Name: "customer", Fields: []recipe.Field{
		{Name: "first", Generator: values.FirstName},
		{Name: "last", Generator: values.LastName},
		{Name: "mail", Generator: values.Email},
	}}
	found := false
	for _, rule := range generate.Rules(entity, source) {
		if rule.Field == "mail" {
			found = true
			if !slices.Equal(sorted(rule.With), []string{"first", "last"}) {
				t.Errorf("the address follows %v, and first and last were wanted", rule.With)
			}
		}
	}
	if !found {
		t.Error("the run keeps no rule binding the address to the name columns")
	}
}

func sorted(names []string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return out
}
