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
func planOf(rec recipe.Recipe, values generate.Values, stop generate.Halt) (generate.Plan, error) {
	plan, err := generate.Prepare(rec, values, stop)
	if err != nil {
		return generate.Plan{}, err
	}
	noteUnreadSettings(rec, &plan)
	return plan, nil
}

// noteUnreadSettings records the settings this run carries and does not read.
//
// Each form reads settings the others have no use for: the object form and the
// SQL form a file per table, the comma separated one a delimiter, the SQL form
// a database. A setting that does nothing and says nothing about doing nothing
// is the silence this product exists to find, so each is named when the form
// asked for ignores it.
//
// 🔴 A note is given only for a setting the recipe WROTE. false is also what a
// recipe that never mentions file_per_entity carries, and a note given for it
// told every comma separated recipe about a line nobody had written (B-301). A
// note about nothing is noise that teaches a reader to skip the list the real
// notes stand in.
func noteUnreadSettings(rec recipe.Recipe, plan *generate.Plan) {
	form := rec.Output.Format
	if form == "" {
		form = format.NameCSV
	}
	// file_per_entity: true asks the comma separated form for what it does
	// anyway, so only false is a setting it ignores.
	if form == format.NameCSV && rec.Output.FilePerEntityGiven && !rec.Output.FilePerEntity {
		plan.Notes = append(plan.Notes, generate.Note{
			Where: "output.file_per_entity",
			What: "the " + format.NameCSV + " form writes one file per table whatever this says, so " +
				"it changes nothing here. The " + format.NameJSON + " and " + format.NameSQL +
				" forms are the ones that read it",
		})
	}
	if form != format.NameCSV && rec.Output.Delimiter != "" {
		plan.Notes = append(plan.Notes, generate.Note{
			Where: "output.delimiter",
			What: "the " + form + " form has no delimiter, so it changes nothing here. The " +
				format.NameCSV + " form is the one that reads it",
		})
	}
	if form != format.NameSQL && rec.Output.Dialect != "" {
		plan.Notes = append(plan.Notes, generate.Note{
			Where: "output.dialect",
			What: "the " + form + " form is not written for a database, so this changes nothing " +
				"here. The " + format.NameSQL + " form is the one that reads it",
		})
	}
	// false is what a recipe that never mentions create_tables carries, and it
	// asks for nothing, so only true is a setting the other forms ignore.
	if form != format.NameSQL && rec.Output.CreateTables {
		plan.Notes = append(plan.Notes, generate.Note{
			Where: "output.create_tables",
			What: "the " + form + " form creates no tables, so this changes nothing here. The " +
				format.NameSQL + " form is the one that reads it",
		})
	}
}
