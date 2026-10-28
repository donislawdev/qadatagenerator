// Package manifestjson writes an expectation manifest as the JSON document the
// published specification describes.
//
// # Why the entries go to disk as they happen
//
// A policy damaging five percent of tens of millions of rows produces hundreds
// of thousands of entries, and holding them to the end loses every one of them
// in the failure where they matter most. The model in the domain is a value and
// says what a manifest is. This says how it is written down, and it writes an
// entry as soon as there is one.
//
// # Why through a buffer, and what that costs
//
// As soon as there is one means into a buffer of a few kilobytes, which goes to
// the file when it fills and when the document closes. Until 2026-10-28 every
// entry was a write of its own, one call to the system per defect (found by an
// outside review on 2026-10-22). What the buffer costs is when a failing disk
// is noticed: up to one buffer of entries later, and the run still ends there
// rather than at its close.
//
// # Why the envelope comes last in the file
//
// The envelope carries how many rows were mutated and how much of the ordered
// damage was applied, and neither is known until the last entry. Putting it
// first would mean holding every entry in memory, which is exactly what
// streaming exists to avoid. The order of members inside a JSON object carries
// no meaning for any parser, so the cost is that a person reading the file with
// their eyes finds the summary at the bottom.
package manifestjson

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Name is the file an expectation manifest is written to.
//
// It says what it is rather than being called manifest.json, which is a name a
// dozen other tools already use and which would be ambiguous in somebody's
// working directory. It matches the name of the specification that describes it.
const Name = "expectation-manifest.json"

// Writer turns entries into the document, one at a time.
type Writer struct {
	to      *bufio.Writer
	written int
	err     error
	// named is every entity the entries stand in, held to the list in the
	// envelope once the envelope arrives.
	named map[string]bool
}

// New starts a manifest document and writes everything that is known before the
// first entry.
func New(to io.Writer, specVersion string) *Writer {
	w := &Writer{to: bufio.NewWriter(to), named: map[string]bool{}}
	w.write("{\n")
	w.write(fmt.Sprintf("  %s: %s,\n", quoted("spec_version"), mustEncode(specVersion)))
	w.write(fmt.Sprintf("  %s: [", quoted("defects")))
	return w
}

// Add records one defect.
//
// # Why the entry is held against the model here
//
// A manifest built as a value is checked as a whole, and by then a streaming
// writer has already put its entries on a disk. So the check moves to the one
// moment where refusing still means something: before the bytes are written. An
// entry the model would refuse is a promise nobody can keep, and it must not
// reach whoever reads the file as one.
//
// The answer is the first failure this document met, and every later call
// repeats it, so a caller that went on would still be told.
func (w *Writer) Add(defect manifest.Defect) error {
	if w.err != nil {
		return w.err
	}
	if problems := defect.Problems(); len(problems) > 0 {
		w.err = fault.New(fault.Runtime,
			"this run built a manifest entry that cannot be recorded: "+problems[0])
		return w.err
	}
	separator := "\n    "
	if w.written > 0 {
		separator = ",\n    "
	}
	encoded, err := json.Marshal(defectOf(defect))
	if err != nil {
		w.err = fault.Wrap(fault.Runtime, err, "cannot write a manifest entry")
		return w.err
	}
	w.write(separator + string(encoded))
	w.written++
	defect.NameEntities(w.named)
	return w.err
}

// Finish closes the document with the envelope.
//
// The counts come from the caller rather than from what this saw, because the
// manifest value in the domain is what decides them and two sources for one
// number is how a summary comes to disagree with the list under it.
func (w *Writer) Finish(run manifest.Run, requested, applied int) error {
	if w.err != nil {
		return w.err
	}
	// The list of entities is what a response record about several tables is
	// held to, so an envelope without it, or one leaving out a table an entry
	// stands in, is a manifest verify could only read as the sum of its tables.
	// The entries are on the disk already, so this is the last moment a
	// refusal still keeps the document from being read as a promise.
	if len(run.Entities) == 0 {
		return fault.New(fault.Runtime, "this run built a manifest envelope that lists no entity")
	}
	if unlisted := run.Unlisted(w.named); len(unlisted) > 0 {
		return fault.New(fault.Runtime, "this run built a manifest envelope that cannot be "+
			"recorded: "+unlisted[0])
	}
	if w.written > 0 {
		w.write("\n  ")
	}
	w.write("],\n")
	envelope, err := json.MarshalIndent(runOf(run, requested, applied), "  ", "  ")
	if err != nil {
		return fault.Wrap(fault.Runtime, err, "cannot write the manifest envelope")
	}
	w.write(fmt.Sprintf("  %s: %s\n}\n", quoted("run"), string(envelope)))
	w.flush()
	return w.err
}

// Count is how many entries have been written.
func (w *Writer) Count() int { return w.written }

func (w *Writer) write(text string) {
	if w.err != nil {
		return
	}
	if _, err := w.to.WriteString(text); err != nil {
		w.err = fault.Wrap(fault.IO, err, "cannot write the manifest")
	}
}

// flush sends what the buffer holds to the file. The set of files commits
// after this, so a failure here is the last one that still keeps a short
// document from being given its real name.
func (w *Writer) flush() {
	if w.err != nil {
		return
	}
	if err := w.to.Flush(); err != nil {
		w.err = fault.Wrap(fault.IO, err, "cannot write the manifest")
	}
}

func quoted(name string) string { return mustEncode(name) }

// mustEncode encodes a string that cannot fail to encode, so that the callers
// above stay readable. A Go string always encodes.
func mustEncode(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}
