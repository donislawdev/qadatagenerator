package produce

import (
	"encoding/json"

	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

// RunNoteName is the file that travels beside the data.
//
// 🔴 Deliberately not called a manifest. That word belongs to the public
// contract of the second pillar, and a name taken in passing is a name that
// stays for years.
const RunNoteName = "run-note.json"

// RunNote is what travels beside the data.
//
// 🔴 It is not a manifest. The expectation manifest is the public contract of
// the second pillar and has its own specification. This says what a run made,
// what it did not do, and what the values in the files are, which is a smaller
// thing with a different reader.
//
// # Why it is written here rather than by the command line
//
// It was, until 2026-09-24, and the window therefore left a folder of records
// that said nothing about itself (B-209). The last field is the reason that
// mattered: a folder of national identifiers has to say, from the folder alone,
// that the values in it describe nobody, and a README does not travel with it.
type RunNote struct {
	Tool          string      `json:"tool_version"`
	Catalog       string      `json:"catalog_version"`
	Manifest      string      `json:"manifest_version"`
	RecipeVersion string      `json:"recipe_schema_version"`
	Seed          string      `json:"seed"`
	Locale        string      `json:"locale"`
	Tag           string      `json:"tag,omitempty"`
	Tables        []NoteTable `json:"tables"`
	NotDone       []NoteEntry `json:"not_done"`
	Overrides     []Override  `json:"overrides"`
	SyntheticData string      `json:"synthetic_data"`
}

// NoteTable is one table a run made.
type NoteTable struct {
	Entity  string   `json:"entity"`
	Rows    int      `json:"rows"`
	Columns []string `json:"columns"`
	Parent  string   `json:"parent,omitempty"`
}

// NoteEntry is one thing a run did not do, and where.
type NoteEntry struct {
	Where string `json:"where"`
	What  string `json:"what"`
}

// Override is one setting of a recipe that something outside it moved.
//
// It carries both sides rather than only the winner, because two runs of one
// recipe giving different data with nothing saying why is the surprise this
// record exists to remove.
type Override struct {
	Setting    string `json:"setting"`
	FromRecipe string `json:"from_recipe"`
	// FromFlag is the value that came from outside the recipe: a flag on the
	// command line, or a control in the window (DEC-255). The name stayed when
	// the window began to write here, because the note is a file on somebody's
	// disk and a second name would split one record of one setting in two.
	FromFlag string `json:"from_flag"`
}

// settingDir is the name the note gives the directory a run writes to.
const settingDir = "output.dir"

// MoveDir points a recipe at a directory given from outside it, and returns the
// record of the move.
//
// # Why one function for both surfaces
//
// The command line moves the directory with --out and the window with its
// field, and the note has to say the same of both (DEC-255). Until 2026-10-13
// only the command line said it, and two runs of one recipe left two notes
// that differed in nothing but this record (B-311). Kept in one place, the
// name of the setting and what counts as a move cannot drift between them.
//
// A directory equal to the recipe's own is still a move, because it was given
// from outside: the recipe could change under it and the run would not follow.
func MoveDir(rec *recipe.Recipe, into string) Override {
	moved := Override{Setting: settingDir, FromRecipe: rec.Output.Dir, FromFlag: into}
	rec.Output.Dir = into
	return moved
}

// Tables is the tables of a plan as the note and the machine readable plan
// carry them.
func Tables(plan generate.Plan) []NoteTable {
	tables := make([]NoteTable, 0, len(plan.Entities))
	for _, planned := range plan.Entities {
		tables = append(tables, NoteTable{
			Entity: planned.Entity, Rows: planned.Rows,
			Columns: planned.Columns, Parent: planned.Parent,
		})
	}
	return tables
}

// NotDone is what a run will not do, as the note and the machine readable plan
// carry it: what the plan says, and after it what the reader of the recipe
// carried without reading.
//
// 🔴 The second half since 2026-10-26. The record beside the data said nothing
// was left undone for a recipe whose date of birth asked for a range this
// build does not read, while validate warned about exactly that. Found by an
// outside review on 2026-10-22.
func NotDone(plan generate.Plan, unchecked []recipe.Unchecked) []NoteEntry {
	entries := make([]NoteEntry, 0, len(plan.Notes)+len(unchecked))
	for _, skipped := range plan.Notes {
		entries = append(entries, NoteEntry{skipped.Where, skipped.What})
	}
	for _, skipped := range unchecked {
		entries = append(entries, NoteEntry{skipped.Where, skipped.What + " - " + skipped.Reason})
	}
	return entries
}

// Overridden is the list of overrides as a document carries it: empty rather
// than absent when nothing was moved.
//
// A run with nothing overridden writes an empty list rather than a null,
// because somebody reading these files reads them all the same way and a type
// that changes with the flags is a trap for them.
func Overridden(applied []Override) []Override {
	if applied == nil {
		return make([]Override, 0)
	}
	return applied
}

// writeRunNote puts the record of the run beside the data.
//
// It goes into a file rather than only onto a screen or a log, because a log
// stays in the pipeline and the data travels. Somebody holding a folder of
// records has to be able to learn from the folder itself that the values in it
// describe nobody.
func writeRunNote(files *store.Set, rec recipe.Recipe, plan generate.Plan,
	applied []Override) error {
	note := RunNote{
		Tool: version.Tool, Catalog: version.Catalog,
		Manifest: version.Manifest, RecipeVersion: version.Recipe,
		Seed: rec.Seed, Locale: rec.Locale, Tag: rec.Tag,
		Tables:        Tables(plan),
		NotDone:       NotDone(plan, rec.Unchecked),
		Overrides:     Overridden(applied),
		SyntheticData: license.SyntheticDataNotice,
	}

	encoded, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		return fault.Wrap(fault.Runtime, err, "cannot write the run note")
	}
	file, err := files.Create(RunNoteName)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(encoded, '\n')); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the run note")
	}
	return nil
}
