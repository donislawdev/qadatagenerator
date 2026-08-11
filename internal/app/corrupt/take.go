package corrupt

import "github.com/donislawdev/qadatagenerator/internal/domain/dataset"

// Source is a dataset that already exists.
//
// It is the one port here that has no counterpart in the pillar that generates
// data, and the difference is the whole of the second pillar: a generator is
// told how many rows to make, while this is told nothing and has to be asked.
//
// The count is answered before the first record rather than discovered at the
// end, because the manifest envelope carries a number of rows and a defect
// about the integrity of a set needs a second row to span. Whoever implements
// this therefore has to look at the whole thing once before handing over the
// first record, and that cost is stated in DEC-155 rather than hidden here.
type Source interface {
	// Table is what was read: the entity, its columns and how many records it
	// holds.
	Table() dataset.Table
	// Next returns the next record, and false when there are none left.
	//
	// Closing whatever was opened belongs to whoever opened it. A walk that
	// ends early is the ordinary case here, because a run can be stopped
	// between two rows, and a port that closed itself only at the end would
	// leave a handle open in exactly that case.
	Next() (dataset.Record, bool, error)
}

// Writer is where the bytes go, and Halt is how this learns it has to stop.
//
// # Why these are declared again rather than taken from the other pillar
//
// Both exist under the same names in the use case that generates data, and
// importing them from there would be shorter. A use case may not import another
// use case, and that rule is not bookkeeping: it is what makes each pillar
// usable without the other, and it is the same rule that put the decorator
// joining them in an adapter rather than in either one.
//
// What is duplicated is the declaration and not the behaviour. Go matches
// interfaces by shape, so the one writer that damages records on their way to a
// file satisfies both of these without a line of adapting code, and there is no
// second implementation for the two to drift apart in.
type Writer interface {
	// Begin announces a table and its columns.
	Begin(table dataset.Table) error
	// Write takes one record.
	Write(record dataset.Record) error
	// End closes a table.
	End(table dataset.Table) error
	// Finish is called once, after the table.
	Finish() error
}

// Halt is how a run learns that something outside it wants it to stop.
type Halt interface {
	// Stopped returns the failure that ends the run, or nil to carry on.
	Stopped() error
}

// NeverStops is a Halt for a caller with nothing that could interrupt it.
type NeverStops struct{}

// Stopped always answers that the run may carry on.
func (NeverStops) Stopped() error { return nil }

// Run walks somebody else's dataset and hands every record to the writer.
//
// # Why this is so short, and why it is still a use case
//
// Nothing about damage appears here. The writer it is given is the one that
// damages records on their way to a file, so this loop is the same loop whether
// anything is being broken or not. That is deliberate: the arrangement means
// the second pillar reads a file the same way in both modes, and a bug in the
// walk cannot be one that only shows when a policy is present.
//
// What is here rather than in the adapter above it is the decision about where
// a run may be stopped, and what a stopped run leaves. Both are behaviour of
// the product rather than a translation of flags.
//
// # Where it can be stopped
//
// Between rows, and nowhere else, which is the same answer the generating
// pillar gives and for the same reason. A table left without its End is not a
// problem: what a stopped run wrote never gets its real name, so an unfinished
// file is thrown away rather than shipped as a complete one.
func Run(from Source, to Writer, stop Halt) error {
	table := from.Table()
	if err := to.Begin(table); err != nil {
		return err
	}
	for {
		if ended := stop.Stopped(); ended != nil {
			return ended
		}
		record, more, err := from.Next()
		if err != nil {
			return err
		}
		if !more {
			break
		}
		if err := to.Write(record); err != nil {
			return err
		}
	}
	if err := to.End(table); err != nil {
		return err
	}
	return to.Finish()
}
