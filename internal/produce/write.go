package produce

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/format"
	"github.com/donislawdev/qadatagenerator/internal/adapter/store"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// Written is what a run left on a disk.
type Written struct {
	// Files are the names the run gave its files, in the order the writer asked
	// for them.
	//
	// 🔴 Collected by watching the writer rather than worked out from the
	// recipe. Which files a form produces is the writer's decision, and one
	// form writes a file per table while another writes a single file for the
	// whole dataset. A caller listing them from the plan would be right about
	// today's forms and quietly wrong about the next one.
	Files []string
	// Rows is how many records the whole run wrote.
	Rows int
}

// Write produces the dataset a recipe describes and leaves it in a directory.
//
// # What this shares with the preview beside it, and why that matters
//
// Everything except where the bytes go. The same plan, the same engine, the
// same writer chosen the same way. That is not tidiness: it is what makes the
// rows somebody looked at before pressing a button the rows they get after
// pressing it. Two assemblies would be two answers, and the second one would
// only be found by somebody comparing a screen with a file.
//
// # Why nothing appears under its real name until the whole run has succeeded
//
// The set of files renames at the end, so a run that fails or is stopped leaves
// nothing that looks like a finished dataset. A half written folder is worse
// than no folder, because it is the kind of thing that gets committed to
// somebody's test repository.
func Write(rec recipe.Recipe, values generate.Values, into string, replace bool,
	stop generate.Halt) (Written, error) {
	plan, err := generate.Prepare(rec, values)
	if err != nil {
		return Written{}, err
	}

	set := store.New(into, replace)
	files := &named{set: set}
	writer, err := format.For(files, rec)
	if err != nil {
		return Written{}, err
	}
	if err := generate.Run(plan, rec, values, writer, stop); err != nil {
		set.Discard()
		return Written{}, err
	}
	if err := set.Commit(); err != nil {
		return Written{}, err
	}
	return Written{Files: files.names, Rows: plan.TotalRows()}, nil
}

// named is the set of files, remembering what was asked of it.
type named struct {
	set   *store.Set
	names []string
}

func (n *named) Create(name string) (format.WriteCloser, error) {
	file, err := n.set.Create(name)
	if err != nil {
		return nil, err
	}
	n.names = append(n.names, name)
	return file, nil
}
