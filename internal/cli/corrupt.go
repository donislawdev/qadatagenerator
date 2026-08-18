package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/capability"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// listOf writes names the way a sentence does.
//
// Two files joined by "and" read fine and three joined by two of them do not,
// which is what a second export form would have produced at every surface that
// announces a run.
func listOf(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
	}
}

// summaryOf is what a person is told about the damage, on the log channel.
//
// # Why every category is named rather than only the worst
//
// A sentence listing what was applied and nothing else is true and leaves a
// false impression of the scope. A run where fifty nine of sixty ordered
// defects were not applied looks, from a count of applied ones alone, like a
// run that did almost nothing wrong.
//
// The counts arrive as numbers and the sentences are made here, because this
// channel is English always while the other surface is translated. One shared
// sentence would have to be wrong on one of them.
func summaryOf(counts damage.Counts, form string) []string {
	lines := []string{fmt.Sprintf("%s ordered, %d applied, manifest in %s",
		counted(counts.Requested, "defect"), counts.Applied,
		listOf(damage.Files(form)))}
	if missing := counts.Missing(); missing > 0 {
		lines = append(lines, fmt.Sprintf(
			"%s could not be applied, and the manifest carries the reason for each one",
			counted(missing, "defect")))
		// 🔴 The breakdown, because the bare count reads like a shortfall and
		// most of it usually is not one. A build whose catalog is ahead of its
		// engine leaves most of what was ordered unimplemented, which is nobody
		// fault and nothing to act on, while a handful of entries saying the
		// run lacks something is a recipe somebody can fix. Those two look
		// identical in a single number.
		lines = append(lines, counts.ByKind...)
	}
	if counts.Unpartnered > 0 {
		lines = append(lines, fmt.Sprintf(
			"%s about the integrity of the set were not ordered at all, because a table of "+
				"one row has no second record for such a defect to span",
			counted(counts.Unpartnered, "defect")))
	}
	return lines
}

// plannedDamage says what a policy will do, without doing any of it.
//
// # Why a dry run has to answer this at all
//
// A dry run is the first phase without the second, so it has to say everything
// the run would do rather than everything it would write. A recipe carrying a
// policy that went unmentioned would let somebody check their recipe, see a
// clean answer and learn only afterwards that most of the damage they asked for
// has no implementation behind it.
//
// It states what is certain rather than estimating. How many rows a share of
// them comes to is a draw, and a number here that the real run then missed
// would be worse than no number.
func plannedDamage(built recipe.Recipe) []string {
	if !built.Policy.Asks() {
		return nil
	}
	described, err := catalogyaml.Load()
	if err != nil {
		// The catalog is our own data inside our own binary, so a failure to
		// read it is not something a recipe can cause. The real run reports it,
		// and a dry run saying nothing here is better than one that invents a
		// number.
		return nil
	}
	ordered, implemented := 0, 0
	for _, layer := range built.Policy.Layers {
		for _, entry := range described.InLayer(catalog.Layer(layer)) {
			ordered++
			if _, has := mutators.Lookup(entry.ID); has {
				implemented++
			}
		}
	}
	return []string{fmt.Sprintf(
		"the policy draws on %s described in %s, and this build implements %d of them. "+
			"A real run writes one manifest entry per ordered defect, applied or not",
		kindsOfDamage(ordered), strings.Join(built.Policy.Layers, ", "), implemented)}
}

// kindsOfDamage counts damage in words.
//
// Written out rather than handed to the general pluraliser, which appends to
// the end of what it is given and would say "kind of damages". The same four
// lines exist in the use case behind validate, and they are four lines rather
// than a shared helper because this text is read by a person deciding whether
// their policy does what they asked, and prose that stumbles there costs more
// than the duplication.
func kindsOfDamage(count int) string {
	if count == 1 {
		return "one kind of damage"
	}
	return strconv.Itoa(count) + " kinds of damage"
}

// withDamage wraps a writer so that the policy of a recipe is carried out on
// the way to disk, and starts the manifest beside the data.
//
// # Why the manifest joins the same set of files
//
// It goes through the same store as the data, so the rename that gives every
// file its real name covers it too. A run stopped part way therefore leaves
// neither a dataset nor a manifest, rather than a manifest describing records
// nobody will find.
func withDamage(built recipe.Recipe, source values.Source, writer generate.Writer,
	files *store.Set, exportForm string) (generate.Writer, *damage.Run, error) {
	if !built.Policy.Asks() {
		return writer, nil, nil
	}
	described, err := catalogyaml.Load()
	if err != nil {
		return nil, nil, err
	}
	schema := values.SchemaOf(built, source)
	run, err := damage.Start(damage.Ask{
		Files: files, Next: writer, Described: described, Schema: schema,
		Policy: built.Policy, Seed: built.Seed, Can: canDo(built, source, schema),
		Schemes: source, Form: exportForm, WidestRows: widestCount(built),
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
func widestCount(built recipe.Recipe) int {
	widest := 1
	for _, entity := range built.Entities {
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
func canDo(built recipe.Recipe, source values.Source, schema recipe.SchemaView) capability.Set {
	names := format.CapabilitiesOf(built.Output.Format)
	names = append(names, recipe.CapabilitiesOf(schema)...)
	names = append(names, values.CapabilitiesOf(built, source)...)
	// The engine needs nothing external for this one, so it is the only
	// capability that is present in every run.
	names = append(names, capability.UnicodeNormalization)
	return capability.NewSet(names...)
}

// closeRecipeManifest writes the envelope of a manifest produced beside
// generated data.
//
// The counts come from what the run actually did rather than from what it meant
// to do. Holding the envelope against the model happens where it is written,
// which is the one place both modes pass through.
func closeRecipeManifest(run *damage.Run, built recipe.Recipe, plan generate.Plan,
	at time.Time) error {
	envelope := manifest.Run{
		GeneratedAt:    at.UTC().Format(time.RFC3339),
		Seed:           built.Seed,
		Locale:         built.Locale,
		CatalogVersion: version.Catalog,
		Recipe:         built.Name,
		Tag:            built.Tag,
		RowsTotal:      plan.TotalRows(),
		RowsMutated:    run.RowsMutated(),
	}
	return run.Finish(envelope)
}
