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
// Six of them, and only the last belongs to a mutator:
//
//   - the catalog does not describe this damage at all
//   - this build has no implementation for it
//   - the run cannot do what the entry requires
//   - the entry does not cover this kind of field
//   - the entry does not cover this numbering scheme
//   - the mutator itself declined, for a reason only it could know
//
// The middle four are answered from data rather than from code. The entry
// declares what it requires and what it applies to, so those questions are
// asked once here rather than once per mutator, and a mutator that never has to
// ask them cannot answer them inconsistently.
package corrupt

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
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

// Providers is where numbering schemes come from.
//
// A second port beside the mutators, and for the same reason: a use case may not
// know where implementations live. It answers by the identifier the catalog and
// the schema both use, so nothing here has to know that a scheme is a Go type in
// an adapter.
type Providers interface {
	// Of returns the provider behind a scheme identifier and whether this
	// build has one. A build without it is not a fault: the catalog names
	// schemes nobody has implemented yet, and a mutator handed nothing
	// refuses in a sentence rather than guessing.
	Of(scheme string) (provider.Provider, bool)
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
	// Instead holds the other columns this damage covers in the same entity,
	// in schema order. They are tried, in an order drawn for this row, when
	// the damage finds the value in Field unfit (ApplyInRow says why).
	Instead []string
	// ID is what this entry will be called in the manifest, so that a test
	// can refer to the case.
	ID string
	// Partner is the other row a set level defect spans, one based, or zero
	// when the damage involves one record.
	//
	// Set integrity is never about a single row. A duplicated key involves two
	// of them and naming one is useless to whoever has to go and look, which is
	// why the manifest refuses an L5 entry that names only one place. The
	// engine cannot work the partner out, because it sees one value at a time,
	// so whoever chooses where damage lands chooses this too.
	Partner int
}

// Engine turns ordered damage into damage done and said.
type Engine struct {
	catalog   catalog.Catalog
	mutators  Mutators
	providers Providers
	// can is what this run is able to do, gathered from the writer, the
	// schema, the providers and the corpus taking part in it.
	can capability.Set
	// source is the run's randomness, handed on to a mutator that has to
	// choose between several pieces of damage that are all correct.
	//
	// The engine holds it for the same reason the selector does: which rows
	// are damaged and which of several swaps is made are both draws about this
	// run, and both have to give the same answer twice for one seed. A mutator
	// cannot hold one, because it is a value in a map built once and shared by
	// every run in the process.
	source seed.Source
}

// New assembles an engine.
func New(described catalog.Catalog, implementations Mutators, schemes Providers,
	can capability.Set, source seed.Source) Engine {
	return Engine{catalog: described, mutators: implementations, providers: schemes,
		can: can, source: source}
}

// Apply carries out one ordered damage and returns the entry describing it.
//
// The value it returns is the damaged one when the damage happened, and the
// original otherwise, so a caller that writes the result back is correct in
// both cases without asking which it got.
func (e Engine) Apply(order Order, field recipe.FieldInfo, value string, at seed.Coordinate) (string, manifest.Defect) {
	damaged, entry, _ := e.attempt(order, field, value, at)
	return damaged, entry
}

// attempt is Apply that also says whether the damage declined for what the
// value itself shows, which is the one refusal another column might not repeat.
func (e Engine) attempt(order Order, field recipe.FieldInfo, value string,
	at seed.Coordinate) (string, manifest.Defect, bool) {
	entry, described := e.catalog.Lookup(order.Mutator)
	if !described {
		return value, undescribed(order), false
	}

	// 🔴 Asked for once, here, and handed to the refusal below rather than
	// asked for again after it.
	//
	// It used to be looked up twice, and the second one dropped the answer to
	// whether there was anything to find. That was safe only because the
	// refusal had asked the same question first, under the identifier of the
	// entry rather than of the order, and those two are the same string only
	// because the catalog holds every entry under its own identifier. That is a
	// true invariant of a different package, relied on here without being
	// named. One lookup removes the reliance and takes a second search out of
	// the path walked once per ordered defect per row.
	damage, implemented := e.mutators.Lookup(order.Mutator)
	if kind, why := e.refuse(entry, field, implemented); kind != "" {
		return value, refused(entry, order, kind, why), false
	}

	outcome := damage.Damage(mutator.Target{
		Value:    value,
		Field:    field,
		At:       at,
		Source:   e.source,
		Params:   parametersOf(entry),
		Provider: e.schemeOf(field),
	})
	if !outcome.Applied() {
		return value, refused(entry, order, manifest.DeclinedByMutator, outcome.Reason()), outcome.ValueUnfit()
	}
	return outcome.Value(), applied(entry, order, value, outcome.Value()), false
}

// Cell is one column of the row being damaged, as the engine needs it.
type Cell struct {
	Field recipe.FieldInfo
	Value string
	// At is the address the damage draws from in this column.
	At seed.Coordinate
}

// Change is a damaged value to be written back to the row.
type Change struct {
	Field string
	Value string
}

// ApplyInRow carries out one order against a row. It returns the entry, and the
// change to write back with true, or false when nothing is to be written.
//
// The cell function answers for a column by name, and says false when the row
// has no such column.
//
// # Why an unfit value moves the order on to another column
//
// An order names the column drawn for it and the other columns its damage
// covers. A mutator that finds the value unfit, such as a value with no letter
// for damage made of letters, has read that off the value alone, so another
// covered column of the same row may well take the damage. Leaving the order
// refused while a column that could carry it sits beside it would make the
// manifest report the draw rather than the data.
//
// Nothing else moves on. A decline that came from trying the damage, such as a
// swap of digits the scheme still accepts, is a finding about this row, and
// carrying the damage on until some column notices it would erase exactly that.
// So the search stops at the first column that does not answer unfit, whatever
// it answers, and the entry is that answer. Every other refusal is about the
// damage or the run and would read the same in every column anyway.
//
// # Why the next column is drawn rather than taken in turn
//
// Each next column is drawn from those not yet tried, so the column that takes
// the damage is equally likely to be any of those that would. Walking on from
// the first would not be: a column standing right after a run of columns the
// damage declines would collect every row they turn away. It is drawn only
// after a decline, so an order that lands at once costs the one draw the
// selector made.
//
// When every column finds its value unfit, the entry is the refusal from the
// column drawn first, because that is where the order was aimed.
func (e Engine) ApplyInRow(order Order, cell func(field string) (Cell, bool)) (manifest.Defect, Change, bool) {
	found, addressed := cell(order.Field)
	first := e.applyAt(order, order.Field, found, addressed)
	if !first.unfit {
		return first.entry, first.change, first.wrote
	}
	remaining := slices.Clone(order.Instead)
	tried := false
	for round := 1; len(remaining) > 0; round++ {
		drawn := int(e.source.IntN(seed.Coordinate{
			Entity: order.Entity, Row: int64(order.Row), Field: string(order.Mutator),
			Purpose: "corrupt:field:" + strconv.Itoa(round),
		}, int64(len(remaining))))
		next := remaining[drawn]
		remaining = slices.Delete(remaining, drawn, drawn+1)
		found, addressed := cell(next)
		if !addressed {
			continue
		}
		tried = true
		if attempt := e.applyAt(order, next, found, addressed); !attempt.unfit {
			return attempt.entry, attempt.change, attempt.wrote
		}
	}
	refusal := first.entry
	if tried {
		// The entry names one column, and without this a reader would ask why
		// the damage was not put in another one that looks as if it could.
		refusal.FailureReason += ". " + otherColumnsTried
	}
	return refusal, Change{}, false
}

// otherColumnsTried closes the reason of a refusal that every covered column of
// the row gave.
const otherColumnsTried = "The other columns of this row that this damage covers were tried " +
	"as well, and none of them could carry it"

// attemptAt is one try of an order on one column.
type attemptAt struct {
	entry  manifest.Defect
	change Change
	wrote  bool
	// unfit says the damage declined for what the value itself shows.
	unfit bool
}

// applyAt carries out an order on one named column.
//
// Damage aimed at no column of this row still gets an answer from the engine,
// and the answer is an entry with a reason. Nothing is written back then, since
// there is no cell to write it to.
func (e Engine) applyAt(order Order, field string, found Cell, addressed bool) attemptAt {
	aimed := order
	aimed.Field, aimed.Instead = field, nil
	value, entry, unfit := e.attempt(aimed, found.Field, found.Value, found.At)
	if !addressed || !entry.Applied {
		return attemptAt{entry: entry, unfit: unfit}
	}
	return attemptAt{entry: entry, change: Change{Field: field, Value: value}, wrote: true}
}

// schemeOf returns the provider behind a column, or nothing.
//
// Nothing is answered rather than refused, because a mutator that does not need
// a scheme is the ordinary case and most of the catalog is that. The one that
// needs it says so by asking, and refuses in its own words when the answer is
// empty.
func (e Engine) schemeOf(field recipe.FieldInfo) provider.Provider {
	if e.providers == nil || field.Scheme == "" {
		return nil
	}
	found, has := e.providers.Of(field.Scheme)
	if !has {
		return nil
	}
	return found
}

// refuse asks the four questions answered before any damage runs, and returns
// the kind and the sentence for the manifest.
//
// Three of them are answered from the entry. The first is answered by the
// caller, which has already asked the register for the mutator and hands the
// answer in rather than having this ask the same question again. That is not
// tidiness: the mutator that runs is then the one whose existence was checked,
// rather than a second one found under a second identifier that happens to be
// the same string.
//
// An empty kind means nothing here refuses, which is the one answer that lets
// the caller go on to the mutator.
func (e Engine) refuse(entry catalog.Entry, field recipe.FieldInfo,
	implemented bool) (manifest.FailureKind, string) {
	if !implemented {
		return manifest.Unimplemented,
			"this build carries no implementation of this damage, only its description"
	}
	if missing := e.can.Missing(entry.Requires); len(missing) > 0 {
		return manifest.RunLacksCapability,
			fmt.Sprintf("this run cannot %s, which this damage needs", list(missing))
	}
	// 🔴 The same question asked of the column, because for a handful of
	// capabilities the run is the wrong thing to ask.
	//
	// A run carries schema.max-length when ANY column declares a limit. Damage
	// that goes past a limit, landing on a column that has none, would then
	// pass a check that is true of the recipe and false of the place it landed.
	// Measured on the walkthrough: damage that empties a value went to a column
	// that is not required, under a run that carries the capability because
	// another column is.
	if missing := columnCannot(entry.Requires, field); len(missing) > 0 {
		return manifest.RunLacksCapability, fmt.Sprintf(
			"the column %s does not supply %s, which this damage needs of the column it "+
				"lands on rather than of the run", field.Name, list(missing))
	}
	if !entry.CoversFieldType(field.Type) {
		return manifest.FieldTypeNotCovered, fmt.Sprintf(
			"the field is %s and this damage applies to %s",
			describeType(field.Type), strings.Join(entry.AppliesTo.FieldTypes, " or "))
	}
	if !entry.CoversProvider(field.Scheme) {
		return manifest.SchemeNotCovered, fmt.Sprintf(
			"the field holds %s and this damage applies to %s",
			describeProvider(field.Scheme), strings.Join(entry.AppliesTo.Providers, " or "))
	}
	return "", ""
}

// columnCannot returns the field scoped capabilities this column does not
// supply, out of what an entry requires.
//
// Which of them are a column's business comes from the registry rather than
// from a list here. A list here would be a second copy of a classification the
// model already holds, and the two would part company the first time a
// capability was added.
func columnCannot(required []capability.Name, field recipe.FieldInfo) []capability.Name {
	wanted := capability.FieldScoped(required)
	if len(wanted) == 0 {
		return nil
	}
	return capability.NewSet(recipe.CapabilitiesOfField(field)...).Missing(wanted)
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
		FailureKind:   manifest.Undescribed,
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

// refused is the entry for damage that did not happen, with the kind and the
// reason.
func refused(e catalog.Entry, order Order, kind manifest.FailureKind, why string) manifest.Defect {
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
		FailureKind:   kind,
	}
}

func locationOf(order Order) manifest.Location {
	where := manifest.Location{Entity: order.Entity, Row: order.Row, Field: order.Field}
	if order.Partner > 0 {
		where.Related = []manifest.Location{{Entity: order.Entity, Row: order.Partner}}
	}
	return where
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

// describeProvider says which numbering scheme a field follows, in words.
//
// A column following none is the ordinary case rather than a fault, so the
// sentence has to read as a statement about the data. Whoever reads it is
// deciding whether their recipe asked for the wrong thing, and "the field holds
// values of no numbering scheme" tells them that, while an empty name in the
// middle of a sentence tells them nothing.
func describeProvider(id string) string {
	if strings.TrimSpace(id) == "" {
		return "values of no numbering scheme"
	}
	return id + " values"
}

// list joins capability names for a sentence a person reads.
func list(names []capability.Name) string {
	written := make([]string, 0, len(names))
	for _, name := range names {
		written = append(written, string(name))
	}
	return strings.Join(written, " or ")
}
