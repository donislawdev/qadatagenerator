// Package manifestexport renders the entries of an expectation manifest into
// the forms a test can use directly.
//
// # Why this is a peer of the JSON writer rather than a reader of it
//
// The canonical manifest is JSON and it is written as the entries happen. An
// export that read that file back would be a second reader of our own format,
// and a disagreement between the writer and the reader would be a new class of
// defect in a project whose whole subject is a report that must not lie. So an
// export is another sink for the same entries, fed at the same moment, and the
// two cannot drift because neither one is the other's input.
//
// # Everything here streams, and the reason is not memory
//
// Every column an export needs is known when its entry is built, so nothing
// waits for entries that have not arrived. The one thing that would need them
// all is aligning a column to its widest value, and that is where a question
// about appearance turns into one about safety: three of the four columns are
// bounded, and the fourth is a column name out of somebody else's file, which
// has no limit at all. Padding every row to the width of one hostile name
// multiplies it by the number of rows. The widths here therefore come from the
// bounded sets only, and the name is written at its natural width. Recorded as
// DEC-165 with what was rejected.
package manifestexport

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// GherkinName is the file the Gherkin form is written to.
//
// The extension is the one a Gherkin tool recognises, and the stem matches the
// canonical manifest beside it so that the two are obviously one run's output.
const GherkinName = "expectation-manifest.feature"

// The column headings, in the order the specification maps them.
var gherkinColumns = [4]string{"row", "field", "mutator", "expectation"}

// Gherkin writes the entries of a run as an Examples table.
type Gherkin struct {
	to      io.Writer
	err     error
	written int
	skipped int
	// widths are the padding targets of the bounded columns, by the index of
	// gherkinColumns. The name column has no target and is not in here.
	widths [4]int
}

// NewGherkin starts the table and writes what is known before the first entry.
//
// The catalog and the number of rows are taken rather than discovered, because
// they are what makes the bounded columns bounded: the widest identifier this
// build can ever write is in the catalog, and the widest row number is the
// count of rows the run was given.
func NewGherkin(to io.Writer, described catalog.Catalog, rows int) *Gherkin {
	g := &Gherkin{to: to}
	g.widths[0] = widthOf(len(strconv.Itoa(max(rows, 1))), gherkinColumns[0])
	g.widths[1] = len(gherkinColumns[1])
	g.widths[2] = widthOf(widestIdentifier(described), gherkinColumns[2])
	g.widths[3] = widthOf(widestExpectation(), gherkinColumns[3])

	// 🔴 The header says what the table does NOT carry. A table listing only
	// the damage that happened reads as a list of everything that was ordered,
	// and the difference is the whole product: an entry that was not applied is
	// not a defect a test can send, so it has no row here and its absence has
	// to be a sentence rather than a gap.
	g.write("# Expectation manifest, Gherkin form. One row per defect that was applied.\n")
	g.write("# Damage that was ordered and not applied is not here, because there is nothing\n")
	g.write("# in the dataset for a test to send. The canonical manifest beside this file\n")
	g.write("# carries those entries with the reason for each.\n")
	// 🔴 The keyword belongs in the file, and leaving it out was found by
	// running this rather than by reading it. Without it these lines are a
	// table, and the whole point of this form is that it is pasted under a
	// scenario outline, where a table with no Examples above it is a parse
	// error. It sits at the left margin so that whoever pastes it indents the
	// block to wherever their scenario lives.
	g.write("Examples:\n")
	g.writeRow([4]string{gherkinColumns[0], gherkinColumns[1], gherkinColumns[2], gherkinColumns[3]})
	return g
}

// Add takes one entry and writes it when it is one a test can use.
func (g *Gherkin) Add(defect manifest.Defect) {
	if g.err != nil {
		return
	}
	if !defect.Applied {
		g.skipped++
		return
	}
	g.written++
	g.writeRow([4]string{
		strconv.Itoa(defect.Location.Row),
		escapeCell(defect.Location.Field),
		escapeCell(string(defect.Mutator)),
		escapeCell(string(defect.Expectation)),
	})
}

// Finish closes the table with what it left out.
func (g *Gherkin) Finish() error {
	if g.err != nil {
		return g.err
	}
	if g.written == 0 {
		// An empty table is not valid as an Examples block and, more to the
		// point, it would read as a run that found nothing wrong. Saying it
		// outright costs one line and removes the ambiguity.
		g.write("# No damage was applied in this run, so this table has no rows.\n")
	}
	g.write(fmt.Sprintf("# %d applied, %d ordered and not applied.\n", g.written, g.skipped))
	return g.err
}

// Written is how many rows the table carries.
func (g *Gherkin) Written() int { return g.written }

// writeRow writes one line of the table, padding the bounded columns.
func (g *Gherkin) writeRow(cells [4]string) {
	var line strings.Builder
	line.WriteString("|")
	for at, cell := range cells {
		line.WriteString(" ")
		line.WriteString(cell)
		// The name column is padded to its heading and no further, because its
		// content has no upper bound and padding to the widest value seen would
		// multiply one hostile name across every row.
		for pad := g.widths[at] - runeCount(cell); pad > 0; pad-- {
			line.WriteString(" ")
		}
		line.WriteString(" |")
	}
	line.WriteString("\n")
	g.write(line.String())
}

func (g *Gherkin) write(text string) {
	if g.err != nil {
		return
	}
	if _, err := io.WriteString(g.to, text); err != nil {
		g.err = fault.Wrap(fault.IO, err, "cannot write "+GherkinName)
	}
}

// escapeCell makes a value safe inside a table cell.
//
// A pipe would end the cell and a newline would end the row, so both are
// written the way the Gherkin grammar spells them. The value is not shortened:
// this file is data for a test rather than a message for a person, and a
// truncated column name names a column that does not exist.
func escapeCell(value string) string {
	replaced := strings.NewReplacer(
		`\`, `\\`,
		`|`, `\|`,
		"\n", `\n`,
		"\r", `\r`,
	)
	return replaced.Replace(value)
}

// runeCount is how wide a cell is on a screen, near enough for padding.
//
// Characters rather than bytes, because a Polish name padded by its byte count
// comes out short. It is not grapheme clusters, which this build cannot count,
// and the difference only shows on emoji, where the padding is cosmetic anyway.
func runeCount(value string) int {
	return len([]rune(value))
}

// widthOf is the padding target of a bounded column: the wider of its heading
// and its widest possible value.
func widthOf(widest int, heading string) int {
	return max(widest, len(heading))
}

// widestIdentifier is the longest mutator identifier this build could write,
// which is bounded because the catalog is built into the binary.
func widestIdentifier(described catalog.Catalog) int {
	widest := 0
	for _, id := range described.IDs() {
		widest = max(widest, runeCount(string(id)))
	}
	return widest
}

// widestExpectation is the longest of the closed set of expectations.
func widestExpectation() int {
	widest := 0
	for _, written := range []catalog.Expectation{catalog.Rejection, catalog.Acceptance, catalog.Warning} {
		widest = max(widest, runeCount(string(written)))
	}
	return widest
}

// FormGherkin is what the flag is given to ask for the Gherkin table.
const FormGherkin = "gherkin"

// Forms are the shapes this build can export a manifest in, in the order a
// message lists them.
//
// The list is here rather than in the surface that reads the flag, so that the
// refusal and the renderer cannot disagree about what exists. The canonical
// JSON is deliberately absent: it is always written and is not something to
// ask for.
func Forms() []string { return []string{FormGherkin} }

// Usable reports whether a form is one this build writes.
//
// The empty string is usable and means no export, which is the default. That is
// the one answer a caller has to be able to give without naming anything.
func Usable(form string) bool {
	if form == "" {
		return true
	}
	for _, known := range Forms() {
		if form == known {
			return true
		}
	}
	return false
}
