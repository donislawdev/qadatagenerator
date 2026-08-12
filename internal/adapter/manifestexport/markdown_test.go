package manifestexport_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
)

// 🔴 The rule under the heading is what makes these lines a table, and it is
// the same class of omission the Gherkin form shipped with.
//
// That one was missing the Examples keyword, so the file was a correct table
// and a useless block. This one without the row of dashes renders as a
// paragraph full of pipes: every renderer accepts it, nothing complains, and
// what arrives in the bug report is unreadable. A test reading the cells cannot
// see a line that is not there, so this looks for the line itself and for where
// it sits.
func TestTheTableCarriesTheRuleThatMakesItRender(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewMarkdown(&out, described(t), 1)
	table.Add(applied(1, "pesel", "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	heading, rule := -1, -1
	for at, line := range lines {
		switch {
		case strings.HasPrefix(line, "| row "):
			heading = at
		case strings.HasPrefix(line, "| ---"):
			rule = at
		}
	}
	switch {
	case heading < 0:
		t.Fatalf("the document has no heading row:\n%s", out.String())
	case rule < 0:
		t.Fatalf("the document has no rule, so it renders as a paragraph:\n%s", out.String())
	case rule != heading+1:
		t.Errorf("the rule is at line %d and the heading at %d, so they are not a table",
			rule, heading)
	}
	// A rule with the wrong number of cells is a table with the wrong number of
	// columns, which renders as badly as no rule at all.
	if got := strings.Count(lines[rule], "|"); got != 5 {
		t.Errorf("the rule has %d separators and the table has four columns", got)
	}
}

// Only damage that happened is in the table, and the rest is counted in words.
//
// This form is pasted into a report about damage that happened, so fifty three
// lines saying nothing happened would bury the seven that did. The count is
// what keeps that from being a silence.
func TestTheReportShowsWhatHappenedAndCountsTheRest(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewMarkdown(&out, described(t), 60)
	table.Add(applied(10, "pesel", "L3.checksum.off_by_one"))
	table.Add(refused(11, "pesel", "L3.checksum.off_by_one"))
	table.Add(refused(12, "ssn", "L3.range.unassigned"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	written := out.String()
	if got := table.Written(); got != 1 {
		t.Errorf("the table says it wrote %d rows, want 1", got)
	}
	if strings.Contains(written, "L3.range.unassigned") {
		t.Error("damage that did not happen has a row in the report")
	}
	if !strings.Contains(written, "1 applied, 2 ordered and not applied.") {
		t.Errorf("the document does not count what it left out:\n%s", written)
	}
}

// A run that damaged nothing says so rather than leaving an empty table.
//
// An empty table under a heading reads as a run that found nothing wrong, which
// is a different statement from a run whose whole order went unimplemented.
func TestARunThatDamagedNothingSaysSoInWords(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewMarkdown(&out, described(t), 60)
	table.Add(refused(11, "pesel", "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	written := out.String()
	if !strings.Contains(written, "No damage was applied") {
		t.Errorf("an empty report does not say it is empty:\n%s", written)
	}
	if !strings.Contains(written, "1 were ordered") {
		t.Errorf("an empty report does not say how much was ordered:\n%s", written)
	}
}

// A pipe in a column name does not add a column to the row.
func TestAPipeInAColumnNameDoesNotAddAColumn(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewMarkdown(&out, described(t), 1)
	table.Add(applied(1, "od|do", "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	for _, row := range strings.Split(out.String(), "\n") {
		if !strings.HasPrefix(row, "| 1 ") {
			continue
		}
		if got := strings.Count(row, "|") - strings.Count(row, `\|`); got != 5 {
			t.Errorf("the row has %d cell separators, want 5: %s", got, row)
		}
	}
}
