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
//
// # Why the directory is recorded as moved
//
// The field on this screen does what --out does in a terminal, and the note
// beside the data says so the way it says it for the flag (DEC-255). It is
// recorded exactly when the line this screen shows for a terminal carries
// --out, which is whenever the field holds anything, so the note from the
// window is the note that line would leave.
func Writer() Write {
	return func(recipe []byte, settings produce.Settings,
		stop generate.Halt) (produce.Written, error) {
		built := decodeRecipe(recipe).Recipe
		if settings.Into != "" {
			settings.Overrides = append(settings.Overrides, produce.MoveDir(&built, settings.Into))
		}
		prepared, err := produce.Prepare(built, values.New(built.Locale, built.Seed))
		if err != nil {
			return produce.Written{}, err
		}
		return prepared.Write(settings, stop, time.Now())
	}
}
