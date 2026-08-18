// Package damage assembles the pieces that break records on their way to a
// file, so that both modes which do it and both surfaces which offer them are
// looking at one assembly.
//
// # Why this is not in either surface
//
// Until this package existed, the whole assembly lived in the command line
// adapter. That was correct while the command line was the only thing calling
// it, and it stopped being correct the moment a screen had to do the same work.
// The layer test forbids the window to import the command line, for the good
// reason that two surfaces running each other is how one of them becomes the
// product and the other a copy of it.
//
// The choice was therefore between a second assembly in the window and one
// assembly neither surface owns. A second one would drift, and the drift would
// be silent: both would keep passing their own tests while ordering damage
// differently, which is the trap docs/12 names as the most expensive product
// mistake available here.
//
// # What is here and what is deliberately not
//
// Here: which files a run writes, how the manifest and its export are opened
// as peers, how the engine and the selector are wired to the writer, and what
// a finished run counted.
//
// Not here: a single sentence meant for a person. What a run did comes out of
// this package as numbers, and each surface says it in its own words, because
// the command line speaks English always while the window speaks whatever it
// was translated into. A shared sentence would force one of those two to be
// wrong.
package damage

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/corrupted"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/mutator"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// Providers is where a numbering scheme is looked up, declared here so that
// neither surface has to know which registry answers.
type Providers interface {
	ProviderNamed(name string) (provider.Provider, bool)
}

// Ask is everything a damaging run needs before it starts.
//
// It is one value rather than ten parameters because both modes hand over the
// same set and a function taking them one by one would be over the limit this
// project puts on how many things one function may be given.
type Ask struct {
	// Files is the set the run writes into, so that a run which stops leaves
	// nothing behind rather than a manifest describing records nobody has.
	Files *store.Set
	// Next is the writer the records reach once the damage is done to them.
	Next generate.Writer
	// Described is the catalog of damage this build carries.
	Described catalog.Catalog
	// Schema is what the run knows about the shape of the data, whether that
	// came from a recipe or from looking at somebody else's file.
	Schema recipe.SchemaView
	// Policy is what damage was ordered.
	Policy recipe.Policy
	// Seed is what makes the choice of damage repeatable.
	Seed string
	// Can is what this run is able to do, gathered from every supplier.
	Can capability.Set
	// Schemes is the registry the damage of a numbered column goes through.
	Schemes Providers
	// Form is the extra shape the manifest is exported in, empty for none.
	Form string
	// WidestRows sizes the row column of an export and nothing else.
	WidestRows int
}

// Counts is what a finished run did, as numbers rather than as prose.
//
// # Why every category and not only the applied ones
//
// A count of what was applied is true and leaves a false impression of the
// scope. A run where fifty nine of sixty ordered defects went nowhere looks,
// from the applied count alone, like a run that barely did anything wrong.
// Whoever displays this is expected to say all of it.
type Counts struct {
	// Requested is how many defects the policy ordered.
	Requested int
	// Applied is how many of them reached the data.
	Applied int
	// ByKind breaks the shortfall down, because a build whose catalog is ahead
	// of its engine and a recipe somebody can fix look identical in one number.
	ByKind []string
	// Unpartnered is how many defects about the integrity of a set were never
	// ordered, because a table of one row has no second record to span.
	Unpartnered int
}

// Missing is how many ordered defects never reached the data.
func (c Counts) Missing() int { return c.Requested - c.Applied }

// Run is one damaging run in progress: what it writes through, what it records
// and what it has counted so far.
type Run struct {
	writer   *corrupted.Writer
	entries  *manifestjson.Writer
	selector *corrupt.Selector
	// exported is the extra form the manifest was asked for, or nothing.
	//
	// It is held so that it can be closed. The canonical manifest and an export
	// are peers while entries arrive, and they stay peers at the end: each one
	// has a last line of its own, and an export left unclosed would be a file
	// that stops mid table without saying so.
	exported manifestexport.Export
}

// Files names the manifest files one run writes.
//
// 🔴 One function rather than a phrase repeated at each surface, because a run
// says what it wrote in several places and a second form added to only some of
// them would leave the rest quietly announcing the wrong set of files.
func Files(form string) []string {
	if name := manifestexport.FileFor(form); name != "" {
		return []string{manifestjson.Name, name}
	}
	return []string{manifestjson.Name}
}

// Start wires the engine, the selector and the manifest to the writer.
//
// # Why the manifest joins the same set of files
//
// It goes through the same store as the data, so the rename that gives every
// file its real name covers it too. A run stopped part way therefore leaves
// neither a dataset nor a manifest, rather than a manifest describing records
// nobody will find.
func Start(ask Ask) (*Run, error) {
	selector, err := corrupt.NewSelector(ask.Policy, ask.Described, ask.Schema, seed.New(ask.Seed))
	if err != nil {
		return nil, err
	}
	sinks, err := openManifest(ask)
	if err != nil {
		return nil, err
	}
	engine := corrupt.New(ask.Described, mutatorRegistry{}, schemeRegistry{ask.Schemes}, ask.Can)
	wrapped := corrupted.New(ask.Next, engine, &selector, sinks.all, ask.Schema)
	return &Run{
		writer: wrapped, entries: sinks.canonical, selector: &selector,
		exported: sinks.exported,
	}, nil
}

// sinks are the documents an arriving entry is written to.
//
// The canonical manifest is held by name as well as in the list, so that
// finishing it does not depend on remembering which position it took.
type sinks struct {
	all       corrupted.Tee
	canonical *manifestjson.Writer
	exported  manifestexport.Export
}

// openManifest opens the canonical manifest and, when one was asked for, the
// export beside it.
//
// The export is a peer of the canonical manifest rather than a rendering of it
// afterwards, so it is opened here and fed the same entries at the same moment.
// Nothing reads what the run just wrote.
func openManifest(ask Ask) (sinks, error) {
	file, err := ask.Files.Create(manifestjson.Name)
	if err != nil {
		return sinks{}, err
	}
	canonical := manifestjson.New(file, version.Manifest)
	opened := sinks{all: corrupted.Tee{canonical}, canonical: canonical}
	name := manifestexport.FileFor(ask.Form)
	if name == "" {
		return opened, nil
	}
	second, err := ask.Files.Create(name)
	if err != nil {
		return sinks{}, err
	}
	opened.exported = manifestexport.New(ask.Form, second, ask.Described, ask.WidestRows)
	opened.all = append(opened.all, opened.exported)
	return opened, nil
}

// Writer is what the records are handed to.
func (r *Run) Writer() *corrupted.Writer { return r.writer }

// Counts is what this run has done so far.
func (r *Run) Counts() Counts {
	return Counts{
		Requested: r.writer.Requested(), Applied: r.writer.Applied(),
		ByKind: r.writer.ByKind(), Unpartnered: r.selector.Unpartnered(),
	}
}

// RowsMutated is how many records carry damage, asked separately because the
// envelope of the manifest needs it before the counts are shown to anybody.
func (r *Run) RowsMutated() int { return r.writer.RowsMutated() }

// Finish closes the document with its envelope, and the export after it.
//
// The counts come from what the run actually did rather than from what it meant
// to do, and the envelope is held against the model before anything is written:
// a manifest the domain would refuse must not reach a disk, because whoever
// reads it will read it as a promise.
func (r *Run) Finish(envelope manifest.Run) error {
	if problems := envelope.Problems(); len(problems) > 0 {
		return fault.New(fault.Runtime, "the manifest envelope is not usable: "+problems[0])
	}
	if err := r.entries.Finish(envelope, r.writer.Requested(), r.writer.Applied()); err != nil {
		return err
	}
	// The canonical manifest first, so that a failure to close the export is
	// reported after the document everything else is compared against. Either
	// failure ends the run, and a run that ends leaves nothing behind, so
	// neither file is ever shipped half written.
	return r.closeExport()
}

// closeExport writes the last line of the export, if there is one.
func (r *Run) closeExport() error {
	if r.exported == nil {
		return nil
	}
	return r.exported.Finish()
}

// mutatorRegistry adapts the package of implementations to the port the engine
// takes. The two differ by nothing but a package boundary, and converting here
// keeps the use case from importing an adapter.
type mutatorRegistry struct{}

func (mutatorRegistry) Lookup(id catalog.ID) (mutator.Mutator, bool) {
	return mutators.Lookup(id)
}

// schemeRegistry adapts the registry of generators to the port the engine takes
// for numbering schemes.
//
// It is the same registry the values come from, which is the point: a column
// filled by the national identifier generator is damaged by the national
// identifier provider, and there is no second list where the two could
// disagree.
type schemeRegistry struct{ source Providers }

func (r schemeRegistry) Of(scheme string) (provider.Provider, bool) {
	return r.source.ProviderNamed(scheme)
}
