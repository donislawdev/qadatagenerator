package format

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// For returns the writer a recipe asks for.
//
// # Why the choice lives here rather than at each surface
//
// It lived in the command line, because that was where the only caller was, and
// the window cannot import the command line. A second spelling of this switch
// in the window would be two answers to the question of what this build can
// write, and the day one of them gained a format the other would go on refusing
// it while both looked right.
//
// # Why it refuses at all, when two other places refuse the same thing first
//
// The reader of recipes refuses a format it does not know, and the command line
// refuses one that arrived on a flag rather than in the file, which is the
// repair of 2026-08-26: until then a dry run returned before any writer was
// made, so `--dry-run --format parquet` printed how many rows it would write
// and ended with nought while the same command without the flag ended with
// three. This is the third and last of those refusals, and it is here for the
// reason the collision beside it is: a function that trusts every caller to
// have checked is a function that stops being right the day it gains one.
// # Why a halt comes in when only one of the two forms takes it
//
// The object form does all of its writing after the last record has arrived, so
// it is the one writer that has to be told what would stop a run. Handing it in
// here rather than at each surface keeps the two callers from having to know
// which form needs it, which is the same reason the switch itself is here.
//
// # Why the source of values comes in
//
// The SQL form creates tables when a recipe asks it to, typed from what the
// values of each field are, and only the source of values knows that.
func For(files Files, rec recipe.Recipe, values generate.Values, stop generate.Halt) (generate.Writer, error) {
	switch rec.Output.Format {
	case "", NameCSV:
		return NewCSV(files, rec)
	case NameJSON:
		return NewJSON(files, rec, stop)
	case NameSQL:
		return NewSQL(files, rec, values)
	}
	return nil, fault.New(fault.Config, "this build writes "+strings.Join(dataset.Forms(), ", ")+
		", and this run asks for "+rec.Output.Format)
}

// Fits asks a recipe every question the writer of its form asks before it
// writes, and says nothing about the questions of other forms.
//
// # Why one door for them
//
// 🔴 A dry run returns before any writer is made, so a refusal that lived only
// in a writer reached a real run and never a dry one: on 2026-11-25 a recipe
// whose tree would give one object two members of one name ended a dry run with
// nought and the same command without --dry-run with three. Surfaces that stop
// short of writing ask here, and so does every writer, so a question added to a
// form is asked by all of them at once.
func Fits(rec recipe.Recipe, values generate.Values) error {
	if err := NestingCollision(rec); err != nil {
		return err
	}
	return SQLFits(rec, values)
}
