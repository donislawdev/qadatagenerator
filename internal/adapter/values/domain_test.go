package values_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// Every value a request draws lies inside the domain the same request reports,
// and the domain names each of its values once.
//
// 🔴 This is what lets a unique column walk a domain without handing out a value
// the draw would never give. A word list answers the draw and the domain
// through one function since 2026-10-26, and this holds the two together from
// outside, under limits and agreements that change which entries are drawn.
func TestEveryDrawnValueLiesInsideTheDomainItsRequestReports(t *testing.T) {
	female := provider.NewAgreement(map[string]string{provider.RoleGender: "female"})
	male := provider.NewAgreement(map[string]string{provider.RoleGender: "male"})
	short := func(req generate.Request, length int) generate.Request {
		req.MaxLength, req.HasMaxLength, req.Unit = length, true, "characters"
		return req
	}
	cases := []struct {
		name   string
		locale string
		req    generate.Request
	}{
		{"a city", "pl-PL", generate.Request{Generator: values.City}},
		{"a city of six letters at most", "pl-PL", short(generate.Request{Generator: values.City}, 6)},
		{"a region", "pl-PL", generate.Request{Generator: values.Region}},
		{"a first name of anybody", "en-US", generate.Request{Generator: values.FirstName}},
		{"a first name of a woman", "pl-PL", generate.Request{Generator: values.FirstName, Agrees: female}},
		{"a surname of a man in Polish", "pl-PL", generate.Request{Generator: values.LastName, Agrees: male}},
		{"a surname of a woman in Polish", "pl-PL", generate.Request{Generator: values.LastName, Agrees: female}},
		{"a surname of anybody in Polish", "pl-PL", generate.Request{Generator: values.LastName}},
		{"a surname in English", "en-US", short(generate.Request{Generator: values.LastName}, 5)},
		{"a gender", "en-US", generate.Request{Generator: values.Gender}},
		{"a date of birth", "en-US", generate.Request{Generator: values.BirthDate}},
		{"a whole number of two digits", "en-US", short(generate.Request{Type: values.TypeInteger}, 2)},
		{"a decimal of five characters", "en-US", short(generate.Request{Type: values.TypeDecimal}, 5)},
		{"a list under values", "en-US", generate.Request{Values: []string{"p", "q", "r"}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			source := values.New(test.locale, "domain")
			domain, known := source.DomainOf(test.req)
			if !known || domain.Size == 0 {
				t.Fatalf("the domain is not known or empty: %v, %d", known, domain.Size)
			}
			members := make(map[string]bool, domain.Size)
			for place := int64(0); place < domain.Size; place++ {
				members[domain.At(place)] = true
			}
			if int64(len(members)) != domain.Size {
				t.Errorf("the domain names %d places and %d different values", domain.Size, len(members))
			}
			for row := int64(0); row < 300; row++ {
				req := test.req
				req.At = at("f", row)
				drawn, err := source.Produce(req)
				if err != nil {
					t.Fatal(err)
				}
				if !members[drawn] {
					t.Fatalf("row %d drew %q, which the domain does not hold", row, drawn)
				}
			}
		})
	}
}

// A field with a list draws from the list and from nothing else, the type
// beside it included, and reaches every entry.
func TestAFieldWithAListDrawsFromItAlone(t *testing.T) {
	source := values.New("en-US", "domain")
	listed := []string{"1", "2", "3"}
	seen := map[string]bool{}
	for row := int64(0); row < 200; row++ {
		drawn, err := source.Produce(generate.Request{Type: values.TypeInteger, Values: listed, At: at("tier", row)})
		if err != nil {
			t.Fatal(err)
		}
		if drawn != "1" && drawn != "2" && drawn != "3" {
			t.Fatalf("row %d drew %q, which the list does not hold", row, drawn)
		}
		seen[drawn] = true
	}
	if len(seen) != len(listed) {
		t.Errorf("two hundred rows reached %d of the %d entries", len(seen), len(listed))
	}
}

// A first name drawn for a woman and one drawn for nobody in particular stand
// in the same column and draw from different sets, so the two domains carry
// different names and a walk of one learns nothing about the other.
func TestTwoDomainsOfOneColumnAreToldApart(t *testing.T) {
	source := values.New("pl-PL", "domain")
	female := provider.NewAgreement(map[string]string{provider.RoleGender: "female"})
	ofAWoman, _ := source.DomainOf(generate.Request{Generator: values.FirstName, Agrees: female})
	ofAnybody, _ := source.DomainOf(generate.Request{Generator: values.FirstName})
	if ofAWoman.Key == ofAnybody.Key {
		t.Errorf("both domains are called %q", ofAWoman.Key)
	}
	if ofAWoman.Size >= ofAnybody.Size {
		t.Errorf("the names of women (%d) are not fewer than all names (%d)", ofAWoman.Size, ofAnybody.Size)
	}
}

// A number described by its type alone is counted rather than listed, and the
// count and the order are the ones the draw uses.
func TestTheDomainOfANumberIsCounted(t *testing.T) {
	source := values.New("en-US", "domain")
	cases := []struct {
		name  string
		req   generate.Request
		size  int64
		place int64
		value string
	}{
		{"two digits", generate.Request{Type: values.TypeInteger, MaxLength: 2, HasMaxLength: true}, 100, 42, "42"},
		{"no limit", generate.Request{Type: values.TypeInteger}, 1_000_000, 0, "0"},
		{"a decimal of four", generate.Request{Type: values.TypeDecimal, MaxLength: 4, HasMaxLength: true}, 1000, 123, "1.23"},
		{"a decimal of five", generate.Request{Type: values.TypeDecimal, MaxLength: 5, HasMaxLength: true}, 10000, 4207, "42.07"},
		{"a date of birth", generate.Request{Generator: values.BirthDate}, 28489, 28488, "2007-12-31"},
		{"a gender of four letters", generate.Request{Generator: values.Gender, MaxLength: 4, HasMaxLength: true,
			Unit: "characters"}, 1, 0, "male"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			domain, known := source.DomainOf(test.req)
			if !known || domain.Size != test.size {
				t.Fatalf("the domain holds %d values (known %v), and it should hold %d", domain.Size, known, test.size)
			}
			if got := domain.At(test.place); got != test.value {
				t.Errorf("place %d holds %q, and it should hold %q", test.place, got, test.value)
			}
		})
	}
	if _, known := source.DomainOf(generate.Request{Generator: values.Pesel}); known {
		t.Error("the domain of an identifier is reported as known, and this build does not count it")
	}
}

// An entry under values is held against the type beside it and against the
// length, in the one spelling the rest of the tool writes.
func TestAnEntryUnderValuesIsHeldAgainstItsTypeAndLength(t *testing.T) {
	source := values.New("en-US", "domain")
	cases := []struct {
		kind  string
		value string
		fine  bool
	}{
		{values.TypeInteger, "42", true},
		{values.TypeInteger, "-7", true},
		{values.TypeInteger, "+7", false},
		{values.TypeInteger, "4.2", false},
		{values.TypeInteger, "-", false},
		{values.TypeDecimal, "4.25", true},
		{values.TypeDecimal, "4", true},
		{values.TypeDecimal, "-0.5", true},
		{values.TypeDecimal, "4,25", false},
		{values.TypeDecimal, ".5", false},
		{values.TypeDecimal, "1.", false},
		{values.TypeDecimal, "1.-5", false},
		{values.TypeDate, "1984-02-29", true},
		{values.TypeDate, "1983-02-29", false},
		{values.TypeDate, "1984-2-29", false},
		{"email", "not checked here", true},
	}
	for _, test := range cases {
		fine, why := source.Accepts(generate.Request{Type: test.kind}, test.value)
		if fine != test.fine {
			t.Errorf("%s %q: accepted %v (%s), and it should be %v", test.kind, test.value, fine, why, test.fine)
		}
	}
	fine, why := source.Accepts(generate.Request{MaxLength: 3, HasMaxLength: true, Unit: "characters"}, "Łódź")
	if fine || !strings.Contains(why, "4 characters") {
		t.Errorf("a value of four characters in a field of three: %v, %q", fine, why)
	}
}
