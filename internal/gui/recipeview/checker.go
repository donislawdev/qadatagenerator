package recipeview

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/damage"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// Checker fills the port above the way qadg validate fills it.
//
// # Why the real one lives here rather than at each composition root
//
// There are two of those: the binary that opens the window, and the list of
// states that the guard and the probe assemble screens from. Written at both,
// this would be two spellings of one wiring, and the day one of them gained a
// registry the other did not is the day the window started answering a
// different question from the command line about the same recipe.
//
// It is a default rather than a replacement for the port. The port stays,
// because a test that had to load the real catalog to check how a problem is
// drawn would be a test about the catalog.
//
// # What makes the two surfaces agree
//
// Not this function. The agreement comes from both handing the same four things
// to the same use case: this decoder, this generator registry, this catalog and
// validate.Run. The command line does it in internal/cli/validate.go and the
// values are the same values.
func Checker(described catalog.Catalog) Check {
	return func(name string, data []byte, stale generate.Halt) Answer {
		// Read once and handed to both. The use case takes a decoder rather
		// than a document, so a closure over what was already read keeps it
		// from knowing that recipes are YAML while costing one reading instead
		// of two. A recipe may be 256 KB and the note on the reader says a
		// document that size can cost hundreds of megabytes to parse, which is
		// a price to pay once per key press rather than twice.
		decoded := decodeRecipe(data)
		read := func([]byte) validate.Decoded { return decoded }

		// The failure is dropped rather than shown, and that is not a swallow.
		// It carries the kind so that a command line can turn it into an exit
		// code, which a window has no use for, and everything a person needs to
		// read is in the report already: a recipe that cannot be used is a
		// report with problems in it.
		report, _ := validate.Run(read, validate.Source{Name: name, Data: data},
			values.New("", ""), mutators.Coverage{Described: described})
		answer := Answer{Report: report}
		if !report.Usable() {
			// A recipe that cannot be used describes nothing, so there are no
			// rows to show. Producing them anyway would mean running a plan
			// over a document the tool has just refused.
			return answer
		}

		// 🔴 The values behind the preview carry the recipe's own locale and
		// seed, unlike the registry above, which is asked only which names
		// exist. A sample drawn from an empty locale would be a sample of a run
		// nobody asked for, and the whole worth of the preview is that it is
		// the run.
		sample, err := produce.Preview(decoded.Recipe, values.New(decoded.Recipe.Locale,
			decoded.Recipe.Seed), produce.DefaultRows, stale)
		if err != nil {
			// A recipe validate approved and the plan refuses. That gap is
			// smaller than it was, because one of its cases was closed on
			// 2026-09-02, and treating what is left as impossible is how the
			// next one would arrive drawn as an approval. A check stopped
			// because the recipe was typed again ends here too, and its
			// answer is dropped by the queue rather than drawn.
			answer.Refused = err
			return answer
		}
		answer.Sample = sample
		// The policy is read from the recipe the run will read, and the plan
		// of it is the one the command line prints on a dry run. It costs a
		// walk over a catalog already in memory, which is nothing next to the
		// sample above.
		if policy := decoded.Recipe.Policy; policy.Asks() {
			answer.Damaged = true
			answer.Planned = damage.PlanOf(policy.Layers, described)
		}
		return answer
	}
}

// decodeRecipe is the same conversion the command line makes, and holds for the
// same reason: both structures carry the same three things, and writing it out
// keeps the use case from knowing that recipes are YAML.
//
// It cannot drift from the other one silently. A change to either structure
// stops this line compiling, which is the compiler doing a guard's work.
func decodeRecipe(data []byte) validate.Decoded {
	return validate.Decoded(recipeyaml.Decode(data))
}
