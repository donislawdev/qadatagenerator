// Package dataset is the shape of a set of records, independent of where it
// came from and of where it is going.
//
// # Why this is not part of the use case that fills it
//
// The shapes here started inside the use case that generates data, and the
// writers took them from there. That was accurate while a dataset could only
// come from a recipe: one table was called Planned because it had been planned,
// and the count of rows and the mapping onto parent rows were things the plan
// had worked out.
//
// The second pillar reads somebody else's file. Such a dataset was never
// planned by anything, so describing it with a type called Planned would be a
// name that lies, and two of that type's members would mean nothing in it.
// Moving the shape here leaves the generator, the corruption engine and the
// writers all depending on the shape rather than on each other, which is the
// dependency rule applied at the one seam where the second pillar would
// otherwise have to import the first.
//
// # What is deliberately not here
//
// Nothing about types, constraints or meaning. A table knows its columns by
// name and its rows as text, and that is all. What a column contains is a
// question with two different answers depending on whether a recipe said so or
// whether it was worked out by looking, and neither of them belongs in the
// shape both use.
//
// A sample is here for the same reason the table is, and it is the same kind of
// thing: what looking at a column showed, before anybody decided what it means.
// Whoever reads a file produces one and whoever works out a schema consumes
// one, and having it here is what stops those two from depending on each other.
package dataset

import "sort"

// Table is one table of a dataset.
type Table struct {
	// Entity is what the table is called. The file it goes to is derived from
	// it by whoever writes, not here.
	Entity string
	// Columns are the field names, in order.
	Columns []string
	// Rows is how many records this table holds.
	Rows int
	// Parent is the entity this one belongs to, empty when it belongs to
	// nobody.
	Parent string
	// Ownership maps each row of this table onto the row of its parent. It is
	// empty when there is no parent, and it is what lets a nested writer put a
	// record where it belongs without holding the whole dataset to work it out.
	Ownership Ownership
}

// Ownership says which row of the parent each row of a table belongs to.
//
// # Why where the children of each parent end, rather than a parent for each child
//
// The children of one parent are planned one after another, so the whole
// mapping is how many children each parent has, added up as it goes. A parent
// for each child held one machine word for every row of the table before a byte
// of it was written, and the preview on the recipe screen paid that at every key
// press, for ten rows it then showed. Found by an outside review on 2026-10-22,
// decided in DEC-266. What this holds is one word for every row of the parent.
type Ownership struct {
	// ends[p] is how many rows belong to parents 0 to p, so the rows of parent
	// p are the ones from ends[p-1] up to but not including ends[p].
	ends []int
	// rows is how many of those rows the table holds. It is fewer than the last
	// end when the table was cut to its first rows.
	rows int
}

// OwnershipOf builds the mapping from how many children each parent has, in
// the order of the parents.
//
// The counts are added up where they stand rather than copied, because a copy
// is the second word per parent this type exists to avoid. Whoever passes them
// hands them over and does not read them again.
func OwnershipOf(shares []int) Ownership {
	total := 0
	for at, share := range shares {
		total += share
		shares[at] = total
	}
	return Ownership{ends: shares, rows: total}
}

// Rows is how many rows the mapping covers.
func (o Ownership) Rows() int { return o.rows }

// ParentOf returns the row of the parent that a row belongs to, and NoParent for
// a row the table does not hold.
func (o Ownership) ParentOf(row int) int {
	if row < 0 || row >= o.rows {
		return NoParent
	}
	return sort.SearchInts(o.ends, row+1)
}

// Cut keeps the first rows and gives each of them the parent it had.
//
// Only the count changes, so a preview of ten rows copies nothing of a mapping
// planned for millions, and row i of the preview belongs where row i of the run
// does because it is the same lookup.
func (o Ownership) Cut(rows int) Ownership {
	if rows < o.rows {
		o.rows = max(rows, 0)
	}
	return o
}

// Record is one row.
type Record struct {
	// Values are the fields, in the order of the table's columns.
	Values []string
	// Absent marks the cells that hold no value at all, in the order of
	// Values, and is nil when every cell holds one.
	//
	// # Why the empty text is not enough
	//
	// A share of empty values leaves a cell with nothing in it, and so does a
	// field nothing in this build fills. Damage that empties a value, and a
	// list that offers the empty text, leave a value that happens to be empty.
	// Both have the empty text in Values. A comma separated file cannot tell
	// them apart and does not try, but a database can, and the difference is
	// the one between NULL and an empty string: a column that refuses the
	// first may accept the second. Whoever reads a file somebody else wrote
	// knows nothing of the kind, and leaves this nil.
	Absent []bool
	// Parent is the row index of the parent record, or NoParent.
	//
	// It travels with the row rather than being looked up later, because the
	// nested form of the output needs it and a writer that had to ask for it
	// would need the whole dataset in memory to answer.
	Parent int
}

// AbsentAt reports whether the cell in the given column holds no value at all,
// rather than a value that may be the empty text.
func (r Record) AbsentAt(column int) bool {
	return column >= 0 && column < len(r.Absent) && r.Absent[column]
}

// NoParent is the parent index of a record that belongs to nobody.
const NoParent = -1

// Sample is what one column of a read dataset held, as text.
//
// # Why the counts are separate from the values
//
// Values is capped, because running a validator over every cell of a file
// somebody else chose the size of is a cost with no ceiling. The counts are
// not capped, because they are cheap and because the difference between them
// is what lets a report say "the first 1000 of 40000 values" rather than
// leaving the reader to assume every value was looked at.
type Sample struct {
	// Field is the column this describes.
	Field string
	// Values are the non-empty values that were kept, in the order they
	// appeared.
	Values []string
	// Rows is the record each kept value came from, counting the first record
	// after the header as one. It is the same length as Values.
	//
	// 🔴 Without it two samples cannot be read side by side, and reading them
	// side by side while believing otherwise is worse than not reading them at
	// all. Empty cells are dropped rather than kept, so the third value of one
	// column and the third of another come from the same record only when
	// neither column has a gap above them. Anything comparing two columns has
	// to line them up by record, and this is what lets it.
	Rows []int
	// NonEmpty and Empty count the whole column rather than the sample.
	//
	// An empty cell is counted apart rather than kept, because it is evidence
	// of nothing: a column of national identifiers with gaps in it is still a
	// column of national identifiers, and a blank neither confirms that nor
	// denies it.
	NonEmpty int
	Empty    int
}
