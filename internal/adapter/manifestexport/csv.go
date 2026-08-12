package manifestexport

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// csvColumns are the headings, in the order a reader meets them.
//
// # Why this form carries what the Gherkin table leaves out
//
// The Gherkin form drops an entry that was not applied, and the reason is
// particular to it: a row there is an instruction to a test, and damage that
// never happened is nothing a test can send. This form is read by a person
// reviewing what a run did, and for them the opposite holds. A table showing
// only what succeeded reads as the whole account of the run, and when seven of
// sixty ordered defects were applied that reading is wrong in the direction
// that matters. So every ordered entry has a line, and three columns say
// whether it happened and why not.
var csvColumns = []string{
	"row", "field", "mutator", "layer", "expectation", "confidence",
	"applied", "failure_kind", "failure_reason",
}

// CSV writes every ordered entry as a row a spreadsheet can open.
//
// # What is deliberately not here
//
// The values before and after the damage are in the canonical manifest and not
// in this file. They are the one part of an entry that can be an attack
// payload rather than a description of one: the catalog has a module whose
// whole purpose is to produce hostile values, and this form exists to be
// opened in a spreadsheet. Writing a payload we generated into the file most
// likely to be double clicked would be handing it the one target it wants.
type CSV struct {
	to      *csv.Writer
	err     error
	written int
}

// NewCSV starts the file with its heading row.
func NewCSV(to io.Writer) *CSV {
	made := &CSV{to: csv.NewWriter(to)}
	made.writeRow(csvColumns)
	return made
}

// Add takes one entry, applied or not.
func (c *CSV) Add(defect manifest.Defect) {
	applied := "no"
	if defect.Applied {
		applied = "yes"
	}
	c.written++
	c.writeRow([]string{
		strconv.Itoa(defect.Location.Row),
		defect.Location.Field,
		string(defect.Mutator),
		string(defect.Layer),
		string(defect.Expectation),
		string(defect.Confidence),
		applied,
		string(defect.FailureKind),
		defect.FailureReason,
	})
}

// Finish empties what the writer is holding.
//
// A comment line saying what this file did to its own values would be read as
// a record by every parser that opens it, so what happened to them is said in
// the specification and in the threat model instead of in the file.
func (c *CSV) Finish() error {
	if c.err != nil {
		return c.err
	}
	c.to.Flush()
	if err := c.to.Error(); err != nil {
		return fault.Wrap(fault.IO, err, "cannot write "+FileFor(FormCSV))
	}
	return nil
}

// Written is how many rows the file carries.
func (c *CSV) Written() int { return c.written }

func (c *CSV) writeRow(cells []string) {
	if c.err != nil {
		return
	}
	safe := make([]string, len(cells))
	for at, cell := range cells {
		safe[at] = neutralise(cell)
	}
	if err := c.to.Write(safe); err != nil {
		c.err = fault.Wrap(fault.IO, err, "cannot write "+FileFor(FormCSV))
	}
}

// neutralise stops a spreadsheet from running a value instead of showing it.
//
// # Why a report needs this when the data beside it does not
//
// A value opening with one of these characters is a formula to a spreadsheet
// rather than text, and the value here can be a column name out of somebody
// else's file. The catalog names this class outright as the one injection that
// targets the file an application exports, which is what this file is. The
// corrupted data beside it is left exactly as produced, because being hostile
// is what that file is for. This one is our report about it and a report that
// runs code when it is opened is our defect.
//
// 🔴 The limit is known and is not covered up: a leading apostrophe is
// commonly dropped when a spreadsheet saves the file and it is opened again,
// so this holds for the first pass and can stop holding on the second. That is
// written down in the threat model rather than left for somebody to find, and
// it is a reason to read the canonical manifest when the answer matters.
func neutralise(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	}
	return value
}
