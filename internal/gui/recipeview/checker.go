package recipeview

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/mutators"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipeyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/validate"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
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
	return func(name string, data []byte) validate.Report {
		// The failure is dropped rather than shown, and that is not a swallow.
		// It carries the kind so that a command line can turn it into an exit
		// code, which a window has no use for, and everything a person needs to
		// read is in the report already: a recipe that cannot be used is a
		// report with problems in it.
		report, _ := validate.Run(decodeRecipe, validate.Source{Name: name, Data: data},
			values.New("", ""), mutators.Coverage{Described: described})
		return report
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
