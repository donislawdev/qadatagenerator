package corrupt_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// stated is damage that behaves exactly as a test says, which is the only way
// to exercise what this package does with each answer a mutator can give.
type stated struct {
	id      catalog.ID
	outcome mutator.Outcome
}

func (s stated) ID() catalog.ID                        { return s.id }
func (s stated) Damage(mutator.Target) mutator.Outcome { return s.outcome }

type registry map[catalog.ID]mutator.Mutator

func (r registry) Lookup(id catalog.ID) (mutator.Mutator, bool) {
	found, ok := r[id]
	return found, ok
}

const damageID catalog.ID = "L2.type.string_in_numeric"

func entry() catalog.Entry {
	return catalog.Entry{
		ID:      damageID,
		Layer:   catalog.FieldShape,
		Summary: "Puts text where the schema declares a number.",
		AppliesTo: catalog.AppliesTo{
			FieldTypes: []string{"integer", "decimal"},
			Formats:    []string{catalog.Any},
			Providers:  []string{catalog.Any},
		},
		Expectation: catalog.Rejection,
		Confidence:  catalog.Hard,
		Reversible:  true,
		Requires:    []capability.Name{capability.SchemaFieldType},
		Example:     catalog.Example{Before: "1250", After: "n/a"},
	}
}

func described(t *testing.T, entries ...catalog.Entry) catalog.Catalog {
	t.Helper()
	built, problems := catalog.New("0.1", "draft", entries)
	if len(problems) > 0 {
		t.Fatalf("the fixture catalog was refused: %v", problems)
	}
	return built
}

func order() corrupt.Order {
	return corrupt.Order{ID: "d1", Mutator: damageID, Entity: "invoice", Row: 4, Field: "amount"}
}

func numeric() recipe.FieldInfo { return recipe.FieldInfo{Name: "amount", Type: "integer"} }

func able() capability.Set { return capability.NewSet(capability.SchemaFieldType) }

func TestDamageThatHappenedIsRecordedWithBothValues(t *testing.T) {
	engine := corrupt.New(described(t, entry()),
		registry{damageID: stated{damageID, mutator.Done("n/a")}}, able())

	value, defect := engine.Apply(order(), numeric(), "1250", seed.Coordinate{})
	if value != "n/a" {
		t.Errorf("the value came back as %q", value)
	}
	if !defect.Applied {
		t.Fatalf("the entry says nothing happened: %s", defect.FailureReason)
	}
	if defect.ValueBefore != "1250" || defect.ValueAfter != "n/a" {
		t.Errorf("the entry carries %q -> %q", defect.ValueBefore, defect.ValueAfter)
	}
	if defect.Expectation != catalog.Rejection || defect.Confidence != catalog.Hard {
		t.Errorf("the verdict was not copied from the entry: %v %v", defect.Expectation, defect.Confidence)
	}
	if defect.Rationale != entry().Summary {
		t.Errorf("the rationale is %q", defect.Rationale)
	}
	if defect.Location.Entity != "invoice" || defect.Location.Row != 4 || defect.Location.Field != "amount" {
		t.Errorf("the entry points at %+v", defect.Location)
	}
}

// Damage that cannot be undone from the manifest does not carry the original.
// Carrying it would offer a way back that does not exist.
func TestIrreversibleDamageDoesNotCarryTheOriginal(t *testing.T) {
	oneWay := entry()
	oneWay.Reversible = false
	engine := corrupt.New(described(t, oneWay),
		registry{damageID: stated{damageID, mutator.Done("n/a")}}, able())

	_, defect := engine.Apply(order(), numeric(), "1250", seed.Coordinate{})
	if defect.HasValueBefore || defect.ValueBefore != "" {
		t.Errorf("the original came back as %q", defect.ValueBefore)
	}
	if !defect.HasValueAfter {
		t.Error("the damaged value is missing, and it is what a test compares against")
	}
}

// This is the whole of what this package has to guarantee. Whatever went wrong,
// an entry comes back saying so, and the data is left as it was.
func TestEveryWayDamageCanFailStillProducesAnEntry(t *testing.T) {
	cases := []struct {
		what   string
		engine corrupt.Engine
		says   string
	}{
		{"the catalog does not describe it",
			corrupt.New(described(t, entry()), registry{}, able()), "no entry in the corruption catalog"},
		{"this build has no implementation",
			corrupt.New(described(t, entry()), registry{}, able()), "carries no implementation"},
		{"the run cannot do what it needs",
			corrupt.New(described(t, entry()),
				registry{damageID: stated{damageID, mutator.Done("n/a")}}, capability.NewSet()),
			"this run cannot schema.field-type"},
		{"the mutator declined",
			corrupt.New(described(t, entry()),
				registry{damageID: stated{damageID, mutator.Skipped("the value is already n/a")}}, able()),
			"the value is already n/a"},
	}

	for _, one := range cases {
		t.Run(one.what, func(t *testing.T) {
			asked := order()
			if strings.HasPrefix(one.what, "the catalog") {
				asked.Mutator = "L9.invented.by_nobody"
			}
			value, defect := one.engine.Apply(asked, numeric(), "1250", seed.Coordinate{})

			if value != "1250" {
				t.Errorf("the data was changed to %q by damage that did not happen", value)
			}
			if defect.Applied {
				t.Fatal("the entry claims the damage happened")
			}
			if defect.FailureReason == "" {
				t.Fatal("nothing happened and nothing says why, which is the silence this forbids")
			}
			if !strings.Contains(defect.FailureReason, one.says) {
				t.Errorf("the reason does not say %q: %s", one.says, defect.FailureReason)
			}
			if defect.ID == "" || defect.Mutator == "" {
				t.Error("the entry cannot be referred to or attributed")
			}
		})
	}
}

// The entry declares the field types it covers, so a mutator meant for numbers
// aimed at a name is refused here rather than inside every mutator.
func TestDamageAimedAtAFieldItDoesNotCoverIsRefusedWithBothSides(t *testing.T) {
	engine := corrupt.New(described(t, entry()),
		registry{damageID: stated{damageID, mutator.Done("n/a")}}, able())

	_, defect := engine.Apply(order(), recipe.FieldInfo{Name: "surname", Type: "string"}, "Nowak", seed.Coordinate{})
	if defect.Applied {
		t.Fatal("text was written over a name by damage meant for numbers")
	}
	for _, want := range []string{"the field is string", "integer or decimal"} {
		if !strings.Contains(defect.FailureReason, want) {
			t.Errorf("the reason does not say %q: %s", want, defect.FailureReason)
		}
	}
}

// identifierEntry is damage meant for two numbering schemes and not for others,
// which is how the eight entries naming schemes are written in the catalog.
func identifierEntry() catalog.Entry {
	scheme := entry()
	scheme.AppliesTo.FieldTypes = []string{"identifier"}
	scheme.AppliesTo.Providers = []string{"PL.pesel", "PL.nip"}
	return scheme
}

// 🔴 The entry declares the schemes it covers, and being an identifier is not
// enough to be one of them.
//
// Every national identifier in a dataset is a field of type identifier, and they
// disagree about what they carry: the social security number has no check digit
// at all. Damage aimed at a check digit that landed there would change a digit,
// leave a value the issuer still calls well formed, and write a hard expectation
// of rejection beside it. Nothing would reject it, so the test reading that
// manifest would report a failure that is ours rather than theirs.
func TestDamageAimedAtASchemeItDoesNotCoverIsRefusedWithBothSides(t *testing.T) {
	engine := corrupt.New(described(t, identifierEntry()),
		registry{damageID: stated{damageID, mutator.Done("666123457")}}, able())

	_, defect := engine.Apply(order(),
		recipe.FieldInfo{Name: "ssn", Type: "identifier", Provider: "US.ssn"},
		"666123456", seed.Coordinate{})
	if defect.Applied {
		t.Fatal("a check digit was broken in a scheme that has none")
	}
	for _, want := range []string{"the field holds US.ssn values", "PL.pesel or PL.nip"} {
		if !strings.Contains(defect.FailureReason, want) {
			t.Errorf("the reason does not say %q: %s", want, defect.FailureReason)
		}
	}
}

// A column of the right type following no published scheme is the ordinary case
// in the take and corrupt mode, where nothing has said what the values obey. The
// refusal has to read as a sentence about the data rather than naming an empty
// string.
func TestAFieldFollowingNoSchemeIsRefusedInWords(t *testing.T) {
	engine := corrupt.New(described(t, identifierEntry()),
		registry{damageID: stated{damageID, mutator.Done("x")}}, able())

	_, defect := engine.Apply(order(),
		recipe.FieldInfo{Name: "reference", Type: "identifier"}, "1234563218", seed.Coordinate{})
	if defect.Applied {
		t.Fatal("damage meant for two schemes was applied to a column following neither")
	}
	if !strings.Contains(defect.FailureReason, "values of no numbering scheme") {
		t.Errorf("the reason reads badly for a column with no scheme: %s", defect.FailureReason)
	}
}

// A field with no declared type is what the take and corrupt mode will hand in
// before anything has worked one out, so the refusal has to read sensibly
// rather than naming an empty string.
func TestAFieldWithNoDeclaredTypeIsRefusedInWords(t *testing.T) {
	engine := corrupt.New(described(t, entry()),
		registry{damageID: stated{damageID, mutator.Done("n/a")}}, able())

	_, defect := engine.Apply(order(), recipe.FieldInfo{Name: "amount"}, "1250", seed.Coordinate{})
	if !strings.Contains(defect.FailureReason, "of no declared type") {
		t.Errorf("the reason reads badly for an unknown type: %s", defect.FailureReason)
	}
}

// The engine and the manifest are held together here. An entry this package can
// produce and the manifest cannot record would be damage that happened and
// could not be written down.
func TestNoEntryThisEngineProducesIsOneTheManifestWouldRefuse(t *testing.T) {
	engines := map[string]corrupt.Engine{
		"applied": corrupt.New(described(t, entry()),
			registry{damageID: stated{damageID, mutator.Done("n/a")}}, able()),
		"declined": corrupt.New(described(t, entry()),
			registry{damageID: stated{damageID, mutator.Skipped("nothing to do here")}}, able()),
		"unimplemented": corrupt.New(described(t, entry()), registry{}, able()),
	}

	for what, engine := range engines {
		t.Run(what, func(t *testing.T) {
			_, defect := engine.Apply(order(), numeric(), "1250", seed.Coordinate{})
			envelope := manifest.Run{
				GeneratedAt: "2026-08-05T10:14:22Z", Seed: "s", Locale: "pl-PL",
				CatalogVersion: "0.1", RowsTotal: 10, RowsMutated: 1,
			}
			if _, problems := manifest.New("0.1", envelope, []manifest.Defect{defect}); len(problems) > 0 {
				t.Errorf("the manifest refuses an entry this engine produced: %v", problems)
			}
		})
	}
}

// Damage the catalog does not describe still reaches the manifest with a layer,
// because the manifest refuses an entry without one and the identifier carries
// it in front.
func TestDamageNobodyDescribedStillCarriesAReadableLayer(t *testing.T) {
	engine := corrupt.New(described(t, entry()), registry{}, able())
	asked := order()
	asked.Mutator = "L4.invented.by_nobody"

	_, defect := engine.Apply(asked, numeric(), "1250", seed.Coordinate{})
	if defect.Layer != catalog.Business {
		t.Errorf("the layer came back as %q", defect.Layer)
	}
}

// The catalog owns the defaults. A mutator handed none would have to invent
// one, which is this tool deciding how much damage to do.
func TestTheCatalogsDefaultsReachTheMutator(t *testing.T) {
	withKnob := entry()
	withKnob.Parameters = []catalog.Parameter{
		{Name: "value", Default: "n/a", Description: "The text written in place of the number."},
	}
	var seen mutator.Parameters
	engine := corrupt.New(described(t, withKnob),
		registry{damageID: watching{damageID, &seen}}, able())

	engine.Apply(order(), numeric(), "1250", seed.Coordinate{})
	if written, ok := seen.Text("value"); !ok || written != "n/a" {
		t.Errorf("the mutator was handed %v", seen)
	}
}

// watching records what it was handed, so that a test can ask what reached it.
type watching struct {
	id   catalog.ID
	seen *mutator.Parameters
}

func (w watching) ID() catalog.ID { return w.id }

func (w watching) Damage(target mutator.Target) mutator.Outcome {
	*w.seen = target.Params
	return mutator.Done("n/a")
}
