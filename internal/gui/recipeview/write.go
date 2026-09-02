package recipeview

import (
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
// # What makes this the same run as the preview
//
// Everything below it. The plan, the engine, the writer and the way the writer
// is chosen are the ones internal/produce uses for the sample, and the only
// difference is that the bytes reach a disk. That is what lets the rows on
// screen be a promise about the file rather than an illustration beside it.
func Writer() Write {
	return func(recipe []byte, into string, replace bool,
		stop generate.Halt) (produce.Written, error) {
		decoded := decodeRecipe(recipe)
		return produce.Write(decoded.Recipe,
			values.New(decoded.Recipe.Locale, decoded.Recipe.Seed), into, replace, stop)
	}
}
