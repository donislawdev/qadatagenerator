package manifestexport_test

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestexport"
	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
	"github.com/donislawdev/qadatagenerator/internal/domain/manifest"
)

func described(t *testing.T) catalog.Catalog {
	t.Helper()
	loaded, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("cannot read the catalog: %v", err)
	}
	return loaded
}

// applied is an entry a test can send, and refused is one it cannot.
func applied(row int, field string, id catalog.ID) manifest.Defect {
	return manifest.Defect{
		ID:            "case",
		Location:      manifest.Location{Entity: "customer", Row: row, Field: field},
		Mutator:       id,
		Layer:         catalog.Layer("L3"),
		Applied:       true,
		ValueAfter:    "x",
		HasValueAfter: true,
		Expectation:   catalog.Rejection,
		Confidence:    catalog.Hard,
		Rationale:     "the check digit no longer agrees",
	}
}

func refused(row int, field string, id catalog.ID) manifest.Defect {
	return manifest.Defect{
		ID:            "case",
		Location:      manifest.Location{Entity: "customer", Row: row, Field: field},
		Mutator:       id,
		Layer:         catalog.Layer("L3"),
		Applied:       false,
		Expectation:   catalog.Rejection,
		Confidence:    catalog.Hard,
		Rationale:     "the check digit no longer agrees",
		FailureReason: "this build carries no implementation of this damage",
		FailureKind:   manifest.Unimplemented,
	}
}

// 🔴 Damage that did not happen has no row, and the file says so in words.
//
// This is the rule the specification states and the one the whole form rests
// on: a row here is an instruction to a test, and there is nothing in the
// dataset for a test to send when the damage was never applied. A table that
// carried them would tell somebody to assert against a value that is not there.
// The count in the last line is what keeps the omission from being a gap.
func TestOnlyAppliedDamageBecomesARowAndTheRestIsCounted(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewGherkin(&out, described(t), 60)
	table.Add(applied(10, "pesel", "L3.checksum.off_by_one"))
	table.Add(refused(11, "pesel", "L3.checksum.off_by_one"))
	table.Add(refused(12, "ssn", "L3.range.unassigned"))
	if err := table.Finish(); err != nil {
		t.Fatalf("the table did not close: %v", err)
	}

	written := out.String()
	if got := table.Written(); got != 1 {
		t.Errorf("the table says it wrote %d rows, want 1", got)
	}
	if strings.Count(written, "\n| ") != 2 {
		t.Errorf("expected a heading and one row, got:\n%s", written)
	}
	if !strings.Contains(written, "# 1 applied, 2 ordered and not applied.") {
		t.Errorf("the file does not count what it left out:\n%s", written)
	}
}

// The keyword has to be in the file, because the form exists to be pasted under
// a scenario outline and a table with no Examples above it is a parse error.
//
// 🔴 Found by running the verb rather than by reading the code, which is why it
// has a test of its own rather than living inside another assertion.
func TestTheTableCarriesTheKeywordThatMakesItPasteable(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewGherkin(&out, described(t), 1)
	table.Add(applied(1, "pesel", "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	written := out.String()
	keyword := strings.Index(written, "Examples:")
	heading := strings.Index(written, "| row ")
	switch {
	case keyword < 0:
		t.Fatalf("the file has no Examples keyword:\n%s", written)
	case heading < 0:
		t.Fatalf("the file has no heading row:\n%s", written)
	case keyword > heading:
		t.Errorf("the keyword comes after the table, so the table is not under it:\n%s", written)
	}
}

// A pipe or a newline in a value ends the cell or the row, so both are written
// the way the grammar spells them.
//
// The value here is a column name, which comes from somebody else's file and is
// therefore whatever they wrote. A file that produced a broken table would be
// this tool handing a test something it cannot parse.
func TestAValueCarryingATableCharacterDoesNotBreakTheTable(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewGherkin(&out, described(t), 1)
	table.Add(applied(1, "od|do", "L3.checksum.off_by_one"))
	table.Add(applied(2, "two\nlines", "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	written := out.String()
	for _, row := range strings.Split(strings.TrimSpace(written), "\n") {
		if strings.HasPrefix(row, "#") || strings.HasPrefix(row, "Examples") {
			continue
		}
		// Four columns means five pipes, and an unescaped one in a value would
		// make six.
		if got := strings.Count(row, "|") - strings.Count(row, `\|`); got != 5 {
			t.Errorf("the row has %d cell separators, want 5: %s", got, row)
		}
	}
	if strings.Contains(written, "two\nlines") {
		t.Error("a newline in a value ended the row instead of being written as an escape")
	}
}

// 🔴 The column of names is NOT padded to its widest value, and this is the
// test for a decision rather than for a behaviour.
//
// Three of the four columns are bounded, so padding them is free. A column name
// comes out of somebody else's file and has no limit, so padding every row to
// the widest one seen multiplies one hostile name by the number of rows. The
// cost of that choice is that rows are uneven, which is what this asserts.
func TestOneEnormousColumnNameDoesNotPadEveryOtherRow(t *testing.T) {
	var out strings.Builder
	table := manifestexport.NewGherkin(&out, described(t), 2)
	table.Add(applied(1, "id", "L3.checksum.off_by_one"))
	table.Add(applied(2, strings.Repeat("k", 5000), "L3.checksum.off_by_one"))
	if err := table.Finish(); err != nil {
		t.Fatal(err)
	}

	for _, row := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(row, "| 1 ") && len(row) > 200 {
			t.Errorf("the short row is %d bytes, so it was padded to the long name", len(row))
		}
	}
}

// The forms this build writes are one list, and the empty string means none.
func TestOnlyTheFormsThisBuildWritesAreUsable(t *testing.T) {
	for _, form := range append([]string{""}, manifestexport.Forms()...) {
		if !manifestexport.Usable(form) {
			t.Errorf("%q is a form this build lists and it was refused", form)
		}
	}
	// The canonical JSON is deliberately not a form to ask for, and the rest are
	// near misses of forms that do exist. Matching is exact rather than folded,
	// because a flag that quietly accepts a different spelling than it documents
	// is one more thing whose behaviour has to be remembered rather than read.
	for _, form := range []string{"gherkin ", "GHERKIN", "json", "Markdown", "csv,gherkin", "feature"} {
		if manifestexport.Usable(form) {
			t.Errorf("%q is not a form this build writes and it was accepted", form)
		}
	}
}

// Every form this build lists writes to a file of its own, and nothing else
// does.
//
// 🔴 The registry is one table read by the flag that refuses an unknown form,
// by the code that opens the file and by the sentences telling a person what
// was written. A form present in the list and absent from the table would be
// accepted by the flag and then written nowhere, so the two directions are
// asserted rather than the list alone.
func TestEveryFormHasItsOwnFileAndNothingElseDoes(t *testing.T) {
	seen := map[string]string{}
	for _, form := range manifestexport.Forms() {
		name := manifestexport.FileFor(form)
		if name == "" {
			t.Errorf("%q is listed as a form and is written to no file", form)
			continue
		}
		if other, taken := seen[name]; taken {
			t.Errorf("%q and %q are both written to %s, so one overwrites the other",
				form, other, name)
		}
		seen[name] = form
		if made := manifestexport.New(form, &strings.Builder{}, described(t), 1); made == nil {
			t.Errorf("%q is listed as a form and nothing starts it", form)
		}
	}
	if name := manifestexport.FileFor(""); name != "" {
		t.Errorf("asking for no export named the file %q", name)
	}
	if made := manifestexport.New("", &strings.Builder{}, described(t), 1); made != nil {
		t.Error("asking for no export started one anyway")
	}
}
