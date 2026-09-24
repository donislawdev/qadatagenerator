package recipeview

import (
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/produce"
)

// Writer fills the port above the way the button on this screen needs it.
//
// # Why the recipe is read again here
//
// The check that runs on every key press already read it, and this reads it
// once more, on the press of a button rather than on a key. The alternative is
// a screen holding a half decoded recipe between the two moments, which is a
// second place for the recipe to live and a second thing to keep in step with
// the field somebody types into. One reading per press costs nothing next to
// the run it starts.
//
// # What makes this the same run as the preview and as the command line
//
// Everything below it. The plan, the engine, the writer, the policy and the
// manifest are the ones internal/produce assembles for qadg generate too, and
// the only differences are where the settings of the run come from. That is
// what lets the rows on screen be a promise about the file rather than an
// illustration beside it, and a recipe with a policy give the same damage here
// as in a terminal (DEC-252).
func Writer() Write {
	return func(recipe []byte, settings produce.Settings,
		stop generate.Halt) (produce.Written, error) {
		decoded := decodeRecipe(recipe)
		prepared, err := produce.Prepare(decoded.Recipe,
			values.New(decoded.Recipe.Locale, decoded.Recipe.Seed))
		if err != nil {
			return produce.Written{}, err
		}
		return prepared.Write(settings, stop, time.Now())
	}
}
