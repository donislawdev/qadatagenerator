// Package corrupt is the use case behind the second pillar: damage that
// declares what a test should expect of it.
//
// # One ordered damage, exactly one entry
//
// Everything here exists to make that true. A policy orders damage in a place,
// and whatever happens next, an entry comes back: applied, with the values, or
// not applied, with a reason. There is no path through this package that
// returns nothing, and that is deliberate rather than tidy. A dataset where
// twenty seven of forty ordered defects happened looks exactly like a complete
// one and will be trusted as one, so the failure this code has to make
// impossible is not damage going wrong but damage going missing.
//
// # Where the refusals come from
//
// Five of them, and only the last belongs to a mutator:
//
//   - the catalog does not describe this damage at all
//   - this build has no implementation for it
//   - the run cannot do what the entry requires
//   - the entry does not cover this kind of field
//   - the mutator itself declined, for a reason only it could know
//
// The middle three are answered from data rather than from code. The entry
// declares what it requires and what it applies to, so those questions are
// asked once here rather than once per mutator, and a mutator that never has to
// ask them cannot answer them inconsistently.
package corrupt

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// Mutators is where behaviour comes from.
//
// It is a port rather than the adapter itself, because a use case may not know
// where its implementations live. It also lets a test hand in damage that
// behaves in a stated way, which is the only way to exercise what this package
// does with each answer.
type Mutators interface {
	// Lookup returns the mutator for an identifier and whether this build
	// has one.
	Lookup(id catalog.ID) (mutator.Mutator, bool)
}

// Order is one damage a policy asked for, in one place.
type Order struct {
	// Mutator is the catalog identifier of the damage.
	Mutator catalog.ID
	// Entity, Row and Field say where. Row is one based, as it is in the
	// manifest, because a person reading the manifest counts from one.
	Entity string
	Row    int
	Field  string
	// ID is what this entry will be called in the manifest, so that a test
	// can refer to the case.
	ID string
}

// Engine turns ordered damage into damage done and said.
type Engine struct {
	catalog  catalog.Catalog
	mutators Mutators
	// can is what this run is able to do, gathered from the writer, the
	// schema, the providers and the corpus taking part in it.
	can capability.Set
}

// New assembles an engine.
func New(described catalog.Catalog, implementations Mutators, can capability.Set) Engine {
	return Engine{catalog: described, mutators: implementations, can: can}
}

// Apply carries out one ordered damage and returns the entry describing it.
//
// The value it returns is the damaged one when the damage happened, and the
// original otherwise, so a caller that writes the result back is correct in
// both cases without asking which it got.
func (e Engine) Apply(order Order, field recipe.FieldInfo, value string, at seed.Coordinate) (string, manifest.Defect) {
	entry, described := e.catalog.Lookup(order.Mutator)
	if !described {
		return value, undescribed(order)
	}

	if refusal, why := e.refuse(entry, field); refusal {
		return value, refused(entry, order, why)
	}

	damage, _ := e.mutators.Lookup(order.Mutator)
	outcome := damage.Damage(mutator.Target{
		Value:  value,
		Field:  field,
		At:     at,
		Params: parametersOf(entry),
	})
	if !outcome.Applied() {
		return value, refused(entry, order, outcome.Reason())
	}
	return outcome.Value(), applied(entry, order, value, outcome.Value())
}

// refuse asks the three questions that are answered from the entry rather than
// from the mutator, and returns the sentence for the manifest.
func (e Engine) refuse(entry catalog.Entry, field recipe.FieldInfo) (bool, string) {
	if _, implemented := e.mutators.Lookup(entry.ID); !implemented {
		return true, "this build carries no implementation of this damage, only its description"
	}
	if missing := e.can.Missing(entry.Requires); len(missing) > 0 {
		return true, fmt.Sprintf("this run cannot %s, which this damage needs", list(missing))
	}
	if !entry.CoversFieldType(field.Type) {
		return true, fmt.Sprintf(
			"the field is %s and this damage applies to %s",
			describeType(field.Type), strings.Join(entry.AppliesTo.FieldTypes, " or "))
	}
	return false, ""
}

// undescribed is the entry for damage the catalog does not know.
//
// It is still an entry. An identifier nobody describes should have been refused
// when the policy was read, so reaching here means something upstream let it
// through, and answering with silence would hide that instead of showing it.
func undescribed(order Order) manifest.Defect {
	return manifest.Defect{
		ID:            order.ID,
		Location:      locationOf(order),
		Mutator:       order.Mutator,
		Layer:         catalog.Layer(layerPrefix(order.Mutator)),
		Applied:       false,
		Expectation:   catalog.Rejection,
		Confidence:    catalog.Soft,
		Rationale:     "Damage this build cannot describe was ordered.",
		FailureReason: "no entry in the corruption catalog carries this identifier",
	}
}

// applied is the entry for damage that happened.
func applied(e catalog.Entry, order Order, before, after string) manifest.Defect {
	entry := manifest.Defect{
		ID:             order.ID,
		Location:       locationOf(order),
		Mutator:        e.ID,
		Layer:          e.Layer,
		Applied:        true,
		ValueAfter:     after,
		HasValueAfter:  true,
		Expectation:    e.Expectation,
		Confidence:     e.Confidence,
		Rationale:      e.Summary,
		ValueBefore:    before,
		HasValueBefore: e.Reversible,
	}
	if !e.Reversible {
		// The original is not carried for damage that cannot be undone from
		// the manifest, because carrying it would offer a way back that does
		// not exist.
		entry.ValueBefore = ""
	}
	return entry
}

// refused is the entry for damage that did not happen, with the reason.
func refused(e catalog.Entry, order Order, why string) manifest.Defect {
	return manifest.Defect{
		ID:            order.ID,
		Location:      locationOf(order),
		Mutator:       e.ID,
		Layer:         e.Layer,
		Applied:       false,
		Expectation:   e.Expectation,
		Confidence:    e.Confidence,
		Rationale:     e.Summary,
		FailureReason: why,
	}
}

func locationOf(order Order) manifest.Location {
	return manifest.Location{Entity: order.Entity, Row: order.Row, Field: order.Field}
}

// parametersOf takes the defaults the catalog declares.
//
// A policy that overrides one of them puts its value on top, and that belongs
// to whoever reads the policy rather than here.
func parametersOf(entry catalog.Entry) mutator.Parameters {
	params := make(mutator.Parameters, len(entry.Parameters))
	for _, declared := range entry.Parameters {
		params[declared.Name] = declared.Default
	}
	return params
}

// layerPrefix reads the layer out of an identifier, for the one entry built
// without a catalog entry behind it.
func layerPrefix(id catalog.ID) string {
	written := string(id)
	if cut := strings.IndexByte(written, '.'); cut > 0 {
		return written[:cut]
	}
	return written
}

// describeType says what a field is, in words, including when it says nothing.
func describeType(fieldType string) string {
	if strings.TrimSpace(fieldType) == "" {
		return "of no declared type"
	}
	return fieldType
}

// list joins capability names for a sentence a person reads.
func list(names []capability.Name) string {
	written := make([]string, 0, len(names))
	for _, name := range names {
		written = append(written, string(name))
	}
	return strings.Join(written, " or ")
}
