// Package format turns records into the bytes of a file.
//
// # What every writer here has in common
//
// The bytes are fixed by us and never by the machine. Line endings are always a
// single newline, numbers and dates arrive already written, and the order of
// columns is the order of the recipe. A file whose bytes depended on the
// settings of whichever machine ran the tool would break the one promise the
// whole project rests on.
//
// # How a child says which parent it belongs to
//
// A child record carries the row index of its parent, and a writer turns that
// into something a reader can follow: a column in the flat forms and nesting in
// the tree form. The value is the primary key of the parent when the recipe
// declares one, and the row number otherwise, because a relation nobody can
// follow is a relation that is not there.
package format

import (
	"bufio"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/domain/dataset"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Files is where a writer puts the bytes of one table.
//
// It is an interface rather than the store itself so that a writer can be
// tested without a disk, which is the same reason the use case takes ports.
type Files interface {
	Create(name string) (WriteCloser, error)
}

// WriteCloser is the least a writer needs from a file.
type WriteCloser interface {
	Write(p []byte) (int, error)
	Close() error
}

// finish empties the buffer of a file into it and closes it.
//
// # Why a table is closed when it ends rather than with the rest of the run
//
// 🔴 Every file used to stay open until the set of files was committed, so a
// recipe of many small tables held a handle for each of them at once, and a
// system that allows a process a few hundred open files would stop such a run
// part way. Found by an outside review on 2026-10-22. Closing the file of a
// table at its end holds one at a time. The store still closes whatever is left
// open at the end and answers for a close that failed here, so a writer that
// forgot this would hold handles again and lose nothing.
func finish(buffered *bufio.Writer, file WriteCloser) error {
	if err := buffered.Flush(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	if err := file.Close(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write the dataset")
	}
	return nil
}

// keys remembers what identifies the rows of the tables somebody belongs to,
// so that a child can name its parent.
//
// # Why only those tables, and why only their declared keys
//
// 🔴 The identifier of every row of every table used to be kept until the end
// of the run, a table nobody belongs to included, so the comma separated form,
// which streams everything else, grew in memory with its rows: an outside
// review measured the peak of the whole process on 2026-10-22 at 199 MiB for
// two million rows of one table and 28 for fifty thousand. A hundred thousand
// rows of one column left 3.5 MB of keys in use, measured beside the guard in
// closing_test.go. A table without a primary key is named by its row
// number, and a number can be worked out when it is asked for, so what is kept
// now is the key of each row of a table that is somebody's parent and declares
// one. The key is still taken from the record as it is written, after any
// damage, which is what it always was.
type keys struct {
	// parents are the tables somebody belongs to.
	parents map[string]bool
	// kept holds the key of each row of a parent table that declares one.
	kept map[string][]string
	// counted is how many rows each parent table has, so that a row outside
	// one is named by nothing, as it was when every key was kept.
	counted map[string]int
}

func newKeys(rec recipe.Recipe) keys {
	parents := make(map[string]bool)
	for _, entity := range rec.Entities {
		if len(entity.Parents) > 0 {
			parents[entity.Parents[0].Entity] = true
		}
	}
	return keys{parents: parents, kept: make(map[string][]string), counted: make(map[string]int)}
}

// record notes one row of a table, and keeps its key when a child will ask
// for it.
func (k keys) record(entity recipe.Entity, planned dataset.Table, values []string) {
	if !k.parents[entity.Name] {
		return
	}
	k.counted[entity.Name]++
	if len(entity.PrimaryKey) > 0 {
		k.kept[entity.Name] = append(k.kept[entity.Name], identify(entity, planned, values))
	}
}

// holds reports whether a parent table has the row a child names, which a
// writer that has a word for nothing asks before it asks for the key.
func (k keys) holds(entity string, row int) bool {
	return row >= 0 && row < k.counted[entity]
}

// of returns what names one row of a parent table: its primary key when the
// table declares one, and its row number otherwise.
func (k keys) of(entity string, row int) string {
	if row < 0 || row >= k.counted[entity] {
		return ""
	}
	if kept, declared := k.kept[entity]; declared {
		return kept[row]
	}
	return strconv.Itoa(row)
}

// identify returns the primary key of one row, its parts joined in the order
// the recipe lists them.
func identify(entity recipe.Entity, planned dataset.Table, values []string) string {
	identifier := ""
	for _, part := range entity.PrimaryKey {
		for column, name := range planned.Columns {
			if name != part || column >= len(values) {
				continue
			}
			if identifier != "" {
				identifier += "-"
			}
			identifier += values[column]
		}
	}
	return identifier
}

// parentColumn is the name of the column a child carries to point at its
// parent.
//
// It is the name of the parent entity rather than something with a suffix,
// because a suffix is a convention this project would be inventing and every
// reader would have to learn. A collision with a field of the same name is
// refused rather than resolved, since resolving it would put two different
// things in one column.
func parentColumn(entity recipe.Entity, planned dataset.Table) (string, error) {
	if planned.Parent == "" {
		return "", nil
	}
	// 🔴 The question is asked of the recipe rather than answered here, so that
	// validate can ask the same one. It could not before, because the rule lived
	// in this file and a use case may not import an adapter, and the cost of
	// that was B-205: this refusal used to arrive after validate had approved
	// the same recipe, and it told the reader to go and run validate.
	//
	// It stays here although the plan now refuses such a recipe before any
	// writer is built, so nothing reaches this line through generate any more.
	// A writer that trusts its caller to have checked is a writer that writes a
	// file with two things in one column the day it gains a second caller. The
	// test beside it calls this directly, so the guard is reachable even when
	// the path is not.
	if _, collides := entity.ParentColumnCollision(); collides {
		return "", fault.New(fault.Config, "entities."+entity.Name+" has a field called "+
			planned.Parent+" and also belongs to an entity of that name, so the relation and "+
			"the field would share a column").
			DoInstead("rename the field")
	}
	return planned.Parent, nil
}
