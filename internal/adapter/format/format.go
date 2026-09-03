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

// keys remembers what identifies each row of each table, so that a child can
// name its parent.
type keys struct {
	byEntity map[string][]string
}

func newKeys() keys { return keys{byEntity: make(map[string][]string)} }

// record stores the identifier of one row.
func (k keys) record(entity, key string) {
	k.byEntity[entity] = append(k.byEntity[entity], key)
}

// of returns the identifier of one row of a table.
func (k keys) of(entity string, row int) string {
	rows := k.byEntity[entity]
	if row < 0 || row >= len(rows) {
		return ""
	}
	return rows[row]
}

// identify returns what names one row: the primary key of the entity when it
// has one, and the row number otherwise.
func identify(entity recipe.Entity, planned dataset.Table, values []string, row int) string {
	if len(entity.PrimaryKey) == 0 {
		return strconv.Itoa(row)
	}
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
