package produce

import (
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// The policy of a recipe, carried out on the way to the disk.
//
// # Why this is here and not in the command line any more
//
// It lived in the command line for as long as the command line was the only
// thing that wrote a recipe's dataset (B-181), and that was recorded as a
// choice with a condition for coming back to it: the first screen that writes
// one. The recipe screen was that screen, and for three weeks it wrote the
// same recipe clean while the command line wrote it damaged beside a manifest,
// with not a word on either side (B-300, DEC-252). The window cannot import the
// command line, so the assembly moved to where both already look.

// withDamage wraps a writer so that the policy of a recipe is carried out on
// the way to disk, and starts the manifest beside the data.
//
// # Why the manifest joins the same set of files
//
// It goes through the same store as the data, so the rename that gives every
// file its real name covers it too. A run stopped part way therefore leaves
// neither a dataset nor a manifest, rather than a manifest describing records
// nobody will find.
func withDamage(rec recipe.Recipe, source values.Source, writer generate.Writer,
	files *store.Set, exportForm string) (generate.Writer, *damage.Run, error) {
	if !rec.Policy.Asks() {
		return writer, nil, nil
	}
	described, err := catalogyaml.Load()
	if err != nil {
		return nil, nil, err
	}
	schema := values.SchemaOf(rec, source)
	run, err := damage.Start(damage.Ask{
		Files: files, Next: writer, Described: described, Schema: schema,
		Policy: rec.Policy, Seed: rec.Seed, Can: canDo(rec, source, schema),
		Schemes: source, Form: exportForm, WidestRows: widestCount(rec),
	})
	if err != nil {
		return nil, nil, err
	}
	return run.Writer(), run, nil
}

// widestCount is the largest number of rows any one entity will have.
//
// It sizes the row column of an export and nothing else. A recipe can describe
// several entities and the manifest numbers rows within each, so the widest of
// them is the widest number that can appear.
func widestCount(rec recipe.Recipe) int {
	widest := 1
	for _, entity := range rec.Entities {
		widest = max(widest, entity.Count)
	}
	return widest
}

// canDo gathers what this run is able to do, from every supplier taking part.
//
// # Why five suppliers rather than one
//
// DEC-89 corrected the belief that a provider declares what a mutator needs.
// Most of the capabilities come from somewhere else: the writer owns the bytes
// and the file shape, the schema owns what the data declares about itself, the
// word lists own the kind of text they hold, and the engine owns what it can do
// with no help. Reading them off a provider would leave ten mutators of L0 and
// L1 permanently unapplied for a reason nobody chose.
func canDo(rec recipe.Recipe, source values.Source, schema recipe.SchemaView) capability.Set {
	names := format.CapabilitiesOf(rec.Output.Format)
	names = append(names, recipe.CapabilitiesOf(schema)...)
	names = append(names, values.CapabilitiesOf(rec, source)...)
	// The engine needs nothing external for this one, so it is the only
	// capability that is present in every run.
	names = append(names, capability.UnicodeNormalization)
	return capability.NewSet(names...)
}

// closeManifest writes the envelope of a manifest produced beside generated
// data.
//
// The counts come from what the run actually did rather than from what it meant
// to do. Holding the envelope against the model happens where it is written,
// which is the one place both modes pass through.
func closeManifest(run *damage.Run, rec recipe.Recipe, plan generate.Plan,
	at time.Time) error {
	envelope := manifest.Run{
		GeneratedAt:    at.UTC().Format(time.RFC3339),
		Seed:           rec.Seed,
		Locale:         rec.Locale,
		CatalogVersion: version.Catalog,
		Recipe:         rec.Name,
		Tag:            rec.Tag,
		RowsTotal:      plan.TotalRows(),
		RowsMutated:    run.RowsMutated(),
	}
	return run.Finish(envelope)
}
