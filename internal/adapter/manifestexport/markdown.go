package manifestexport

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// Markdown writes the entries of a run as a table for a bug report.
//
// # Why this one drops what the CSV keeps
//
// The three forms differ by who opens them, and that decides what belongs in
// each. The spreadsheet form is a review of the whole run, so it carries every
// ordered entry and filters. This one is pasted into a report about damage
// that happened, and fifty three lines saying nothing happened would bury the
// seven that did. So the table holds what was applied and a sentence under it
// carries the rest, which is the shape the Gherkin form already proved: the
// count of what is missing is stated rather than left as a gap.
type Markdown struct {
	to      *bufio.Writer
	err     error
	written int
	skipped int
	widths  [4]int
}

// NewMarkdown starts the document and writes what is known before the first
// entry.
func NewMarkdown(to io.Writer, described catalog.Catalog, rows int) *Markdown {
	m := &Markdown{to: bufio.NewWriter(to)}
	m.widths[0] = widthOf(len(strconv.Itoa(max(rows, 1))), gherkinColumns[0])
	m.widths[1] = len(gherkinColumns[1])
	m.widths[2] = widthOf(widestIdentifier(described), gherkinColumns[2])
	m.widths[3] = widthOf(widestExpectation(), gherkinColumns[3])

	m.write("Expectation manifest. One row per defect that was applied.\n\n")
	m.writeRow([4]string{gherkinColumns[0], gherkinColumns[1], gherkinColumns[2], gherkinColumns[3]})
	// The rule under the heading is what makes these lines a table rather than
	// four lines of text, and it is the same class of missing keyword that the
	// Gherkin form was shipped without. A renderer given the rows alone shows
	// them as a paragraph full of pipes.
	var rule [4]string
	for at := range rule {
		rule[at] = strings.Repeat("-", max(m.widths[at], 3))
	}
	m.writeRow(rule)
	return m
}

// Add takes one entry and writes it when it is one the report is about.
func (m *Markdown) Add(defect manifest.Defect) error {
	if m.err != nil {
		return m.err
	}
	if !defect.Applied {
		m.skipped++
		return nil
	}
	m.written++
	m.writeRow([4]string{
		strconv.Itoa(defect.Location.Row),
		escapeCell(defect.Location.Field),
		escapeCell(string(defect.Mutator)),
		escapeCell(string(defect.Expectation)),
	})
	return m.err
}

// Finish closes the document with what the table left out.
func (m *Markdown) Finish() error {
	if m.err != nil {
		return m.err
	}
	if m.written == 0 {
		m.write("\nNo damage was applied in this run, so this table has no rows.\n")
		m.write(fmt.Sprintf("%d were ordered and none of them happened. "+
			"The canonical manifest beside this file carries the reason for each.\n", m.skipped))
		return flushed(m.to, &m.err, FileFor(FormMarkdown))
	}
	m.write(fmt.Sprintf("\n%d applied, %d ordered and not applied. "+
		"The canonical manifest beside this file carries the reason for each.\n",
		m.written, m.skipped))
	return flushed(m.to, &m.err, FileFor(FormMarkdown))
}

// Written is how many rows the table carries.
func (m *Markdown) Written() int { return m.written }

// writeRow writes one line of the table, padding the bounded columns.
func (m *Markdown) writeRow(cells [4]string) {
	var line strings.Builder
	line.WriteString("|")
	for at, cell := range cells {
		line.WriteString(" ")
		line.WriteString(cell)
		for pad := m.widths[at] - runeCount(cell); pad > 0; pad-- {
			line.WriteString(" ")
		}
		line.WriteString(" |")
	}
	line.WriteString("\n")
	m.write(line.String())
}

func (m *Markdown) write(text string) {
	if m.err != nil {
		return
	}
	if _, err := m.to.WriteString(text); err != nil {
		m.err = fault.Wrap(fault.IO, err, "cannot write "+FileFor(FormMarkdown))
	}
}
