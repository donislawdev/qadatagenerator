package produce

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// planOf is the plan a run of this recipe follows, with what the run will not
// do written into it.
//
// # Why every caller goes through this rather than through the use case
//
// The use case plans rows and knows nothing about the forms a dataset is
// written in, so a setting that means nothing in the form asked for cannot be
// noticed there. It was noticed on the command line alone, which meant the
// preview on the recipe screen, the record left beside the data and the plan
// printed by a dry run could each say something different about one recipe.
// One place making the plan is what keeps the four of them one answer.
func planOf(rec recipe.Recipe, values generate.Values) (generate.Plan, error) {
	plan, err := generate.Prepare(rec, values)
	if err != nil {
		return generate.Plan{}, err
	}
	noteUnreadSettings(rec, &plan)
	return plan, nil
}

// noteUnreadSettings records the settings this run carries and does not read.
//
// 🔴 file_per_entity decides the shape of the object form and means nothing to
// the comma separated one, which writes a file per table whatever it says. The
// setting did nothing and said nothing about doing nothing, which is the
// silence this product exists to find.
//
// 🔴 It is said only when the recipe WROTE the setting. Until 2026-09-24 it was
// said whenever the setting was false, and false is also what a recipe that
// never mentions it carries, so every comma separated recipe was told about a
// line nobody had written (B-301). A note about nothing is noise that teaches a
// reader to skip the list the real notes stand in.
func noteUnreadSettings(rec recipe.Recipe, plan *generate.Plan) {
	if !rec.Output.FilePerEntityGiven || rec.Output.FilePerEntity {
		return
	}
	if rec.Output.Format != "" && rec.Output.Format != format.NameCSV {
		return
	}
	plan.Notes = append(plan.Notes, generate.Note{
		Where: "output.file_per_entity",
		What: "the " + format.NameCSV + " form writes one file per table whatever this says, so " +
			"it changes nothing here. The " + format.NameJSON + " form is the one that reads it",
	})
}
