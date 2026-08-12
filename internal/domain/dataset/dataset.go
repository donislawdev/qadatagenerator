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
	// ParentOf maps each row of this table onto the row of its parent. It is
	// empty when there is no parent, and it is what lets a nested writer put a
	// record where it belongs without holding the whole dataset to work it out.
	ParentOf []int
}

// Record is one row.
type Record struct {
	// Values are the fields, in the order of the table's columns.
	Values []string
	// Parent is the row index of the parent record, or NoParent.
	//
	// It travels with the row rather than being looked up later, because the
	// nested form of the output needs it and a writer that had to ask for it
	// would need the whole dataset in memory to answer.
	Parent int
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
