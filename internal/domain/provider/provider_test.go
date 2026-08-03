package provider_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// stub is a provider that declares whatever a test tells it to and implements
// nothing beyond the base contract.
type stub struct {
	id      string
	claimed []capability.Name
}

func (s stub) ID() string                       { return s.id }
func (s stub) Capabilities() []capability.Name  { return s.claimed }
func (s stub) Validate(string) provider.Verdict { return provider.Valid() }
func (s stub) Generate(provider.Request) (string, error) {
	return "", nil
}

// checked adds the interface that identifier.check-digit implies.
type checked struct{ stub }

func (checked) CheckDigits(string) ([]int, bool)         { return []int{10}, true }
func (checked) WithCheckDigits(v string) (string, error) { return v, nil }

// readOnlyRanged implements only the reading half of what
// identifier.reserved-range implies.
type readOnlyRanged struct{ stub }

func (readOnlyRanged) InReservedRange(string) bool { return true }
func (readOnlyRanged) ReservedRanges() []string    { return []string{"a block"} }

// ranged has a harbour and can write into unassigned blocks, which is the
// shape of a scheme like the Social Security number.
type ranged struct{ readOnlyRanged }

func (ranged) UnassignedTargets() []string                    { return []string{"zero_area"} }
func (ranged) WithUnassigned(value, _ string) (string, error) { return value, nil }

// harbourless can write into unassigned blocks and claims no harbour, which is
// the shape of a scheme whose registrar may still issue from its spare blocks.
type harbourless struct{ stub }

func (harbourless) UnassignedTargets() []string                    { return []string{"spare_prefix"} }
func (harbourless) WithUnassigned(value, _ string) (string, error) { return value, nil }

func findings(t *testing.T, p provider.Provider) string {
	t.Helper()
	return strings.Join(provider.Check(p), " | ")
}

// The point of declaring a capability is that somebody can rely on it. A
// provider claiming to know where the check digits sit, without being able to
// answer, would make a mutator that reports it can work and then cannot.
func TestDeclaringACapabilityWithoutImplementingItFails(t *testing.T) {
	got := findings(t, stub{
		id:      "PL.pesel",
		claimed: []capability.Name{capability.IdentifierCheckDigit},
	})
	if !strings.Contains(got, "the interface it implies is not implemented") {
		t.Fatalf("expected the missing interface to be reported, got %q", got)
	}
}

// The quieter half of the same defect. Everything works except that no mutator
// ever asks, so nothing fails and a capability silently goes unused.
func TestImplementingACapabilityWithoutDeclaringItFails(t *testing.T) {
	got := findings(t, checked{stub{id: "PL.nip"}})
	if !strings.Contains(got, "implemented but not declared") {
		t.Fatalf("expected the undeclared interface to be reported, got %q", got)
	}
}

// A provider must not claim what a provider cannot supply. This is the same
// defect the capability registry exists to prevent, caught at the other end.
func TestAProviderCannotClaimSomebodyElsesCapability(t *testing.T) {
	got := findings(t, stub{
		id:      "PL.postal",
		claimed: []capability.Name{capability.FormatCSV, capability.SchemaFieldType},
	})
	if !strings.Contains(got, "supplied by writer, not by a provider") {
		t.Errorf("expected the writer capability to be refused, got %q", got)
	}
	if !strings.Contains(got, "supplied by schema, not by a provider") {
		t.Errorf("expected the schema capability to be refused, got %q", got)
	}
}

func TestAnUnknownOrRepeatedCapabilityIsReported(t *testing.T) {
	got := findings(t, stub{
		id: "US.ssn",
		claimed: []capability.Name{
			"identifier.check-digits", // a near miss of a real name
			capability.ProviderEmail,
			capability.ProviderEmail,
		},
	})
	if !strings.Contains(got, "not a capability in the registry") {
		t.Errorf("expected the unknown name to be reported, got %q", got)
	}
	if !strings.Contains(got, "declared twice") {
		t.Errorf("expected the repeat to be reported, got %q", got)
	}
}

func TestAProviderWithoutAnIdentifierIsReported(t *testing.T) {
	if got := findings(t, stub{}); !strings.Contains(got, "no identifier") {
		t.Fatalf("expected the missing identifier to be reported, got %q", got)
	}
}

// A contributor writing a provider for their country sees every failing point
// at once rather than one per build, which is the same rule the recipe
// validator follows.
func TestEveryFailingPointIsReportedAtOnce(t *testing.T) {
	got := provider.Check(stub{
		claimed: []capability.Name{capability.FormatCSV, capability.IdentifierCheckDigit},
	})
	if len(got) < 3 {
		t.Fatalf("expected the missing identifier and both capability faults, got %v", got)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("the findings are not sorted: %v", got)
		}
	}
}

// The shape a real provider has to reach, so that the checks above are known
// to be refusing something rather than everything.
func TestAWellFormedProviderPasses(t *testing.T) {
	good := checked{stub{
		id:      "PL.nip",
		claimed: []capability.Name{capability.IdentifierCheckDigit},
	}}
	if got := provider.Check(good); len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

// # The half that was missing until the first provider to declare it arrived
//
// A scheme with never issued blocks generates inside one of them, so asking
// whether a value is reserved returns true for every correct value the tool
// produces. A corruption aimed at unassigned ranges therefore cannot use that
// question to decide whether it applies, and it has nothing to build a value
// with either. The write side is what it calls, so claiming that capability
// without implementing it hands the mutator something empty while it is
// corrupting, and this is the test that says so.
func TestClaimingUnassignedTargetsWithoutTheWriteSideIsRefused(t *testing.T) {
	got := findings(t, readOnlyRanged{stub{
		id:      "US.ssn",
		claimed: []capability.Name{capability.IdentifierUnassignedTarget},
	}})
	if !strings.Contains(got, "the interface it implies is not implemented") {
		t.Fatalf("expected the missing write side to be reported, got %q", got)
	}
}

// The shape that passes, so the check above is known to be refusing something
// rather than everything.
func TestAProviderWithBothRangeCapabilitiesPasses(t *testing.T) {
	good := ranged{readOnlyRanged{stub{
		id: "US.ssn",
		claimed: []capability.Name{
			capability.IdentifierReservedRange,
			capability.IdentifierUnassignedTarget,
		},
	}}}
	if got := provider.Check(good); len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

// The reason those two capabilities are separate, kept as a test so that
// merging them back fails here rather than in somebody's manifest.
//
// A scheme whose registrar names spare blocks it may still issue from has no
// harbour, so it cannot claim its own output could not belong to anybody. It
// can still say which blocks a corruption may move a value into, and that has
// to be enough to pass the contract. While the two were one capability this
// shape was inexpressible: such a provider had to claim a harbour it does not
// have, and a manifest reader would have believed it.
func TestAProviderWithNoHarbourCanStillOfferUnassignedTargets(t *testing.T) {
	subject := harbourless{stub{
		id:      "US.aba",
		claimed: []capability.Name{capability.IdentifierUnassignedTarget},
	}}

	if got := provider.Check(subject); len(got) != 0 {
		t.Fatalf("a provider offering only the write side was refused: %v", got)
	}
	if _, ok := interface{}(subject).(provider.RangeAware); ok {
		t.Error("the fixture claims a harbour, so this test no longer describes a scheme without one")
	}
}

// A provider declaring nothing is legal. It brings no capability, so no
// mutator will select it, and that is an answer rather than a failure.
func TestAProviderThatBringsNothingIsLegal(t *testing.T) {
	if got := provider.Check(stub{id: "PL.plain"}); len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

// Every capability that implies an interface has to be one a provider is
// allowed to declare, or Check would refuse it on one rule and demand it on
// another.
func TestEveryImpliedInterfaceBelongsToAProvider(t *testing.T) {
	for _, name := range provider.ImpliedInterfaces() {
		supplier, known := capability.SupplierOf(name)
		if !known {
			t.Errorf("%s implies an interface but is not in the registry", name)
			continue
		}
		if supplier != capability.Provider {
			t.Errorf("%s implies a provider interface but is supplied by %q", name, supplier)
		}
	}
}

// Values are addressed rather than drawn in sequence, so what a field has to
// agree with is looked up by name and a missing binding is answered rather
// than returned as an empty string that reads like a value.
func TestAnAgreementAnswersWhetherAFieldWasBound(t *testing.T) {
	agreed := provider.NewAgreement(map[string]string{"birth_date": "1990-04-11", "gender": "female"})

	if value, ok := agreed.Value("birth_date"); !ok || value != "1990-04-11" {
		t.Errorf("the bound value did not survive: %q %v", value, ok)
	}
	if _, ok := agreed.Value("pesel"); ok {
		t.Error("a field that was never bound is reported as bound")
	}
	if fields := strings.Join(agreed.Fields(), ","); fields != "birth_date,gender" {
		t.Errorf("expected the fields sorted, got %q", fields)
	}

	var none provider.Agreement
	if _, ok := none.Value("anything"); ok {
		t.Error("an empty agreement claims a binding")
	}
	if got := none.Fields(); len(got) != 0 {
		t.Errorf("an empty agreement names fields: %v", got)
	}
}

// The request carries the address of a value rather than a stream, which is
// what lets a provider be called in any order and still answer the same way.
func TestARequestCarriesTheAddressOfTheValue(t *testing.T) {
	source := seed.New("release-2026-08")
	at := seed.Coordinate{Entity: "customer", Row: 7, Field: "pesel", Purpose: "value"}

	first := provider.Request{Source: source, At: at}
	second := provider.Request{Source: source, At: at}

	if first.Source.Uint64(first.At) != second.Source.Uint64(second.At) {
		t.Fatal("the same address produced two different draws")
	}
}
