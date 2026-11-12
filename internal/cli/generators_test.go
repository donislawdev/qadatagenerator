package cli

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
)

// namesInTheTable pulls the first word of every row of the table help prints,
// which is the generator that row is about.
//
// It reads the block between the heading and the blank line under it rather
// than looking for names it already knows, because a test asking whether each
// expected name is there cannot notice a name nobody expected.
func namesInTheTable(t *testing.T, printed string) []string {
	t.Helper()
	var found []string
	started := false
	for _, line := range strings.Split(printed, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == generatorsColumnName {
			started = true
			continue
		}
		if !started {
			continue
		}
		if len(fields) == 0 {
			break
		}
		found = append(found, fields[0])
	}
	return found
}

// 🔴 Until 2026-09-03 the only way to learn which generators this build carries
// was to name one it does not have and read the refusal. This is that
// vocabulary asked for rather than stumbled into, and it is held against the
// register in both directions: a generator missing from the list is a value
// nobody can use, and a name in the list that the register does not carry is a
// promise this build cannot keep.
func TestHelpNamesEveryGeneratorThisBuildHasAndNothingElse(t *testing.T) {
	got := run(commandHelp, subjectGenerators)
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	listed := namesInTheTable(t, got.data)
	carried := values.New("", "guard").Names()
	if len(listed) != len(carried) {
		t.Fatalf("the register carries %d generators and the table has %d rows: %v",
			len(carried), len(listed), listed)
	}
	for at, name := range carried {
		if listed[at] != name {
			t.Errorf("row %d names %q and the register has %q there", at, listed[at], name)
		}
	}
}

// A scheme of no one country that covers some countries only says which in its
// row, because the column of locales beside it says yes everywhere and an
// account number asked for in en-US is still Polish (DEC-281). Held against the
// register rather than against a name, so the next such scheme is covered too.
func TestHelpNamesTheCountriesOfASchemeThatCoversSomeOnly(t *testing.T) {
	got := run(commandHelp, subjectGenerators)
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	source := values.New("", "guard")
	scoped := 0
	for _, name := range source.Names() {
		countries := source.CountriesOf(name)
		if len(countries) == 0 {
			continue
		}
		scoped++
		scheme, _ := source.ProviderOf(name)
		said := "(" + scheme + ", " + strings.Join(countries, ", ") + generatorsOnly + ")"
		if !strings.Contains(got.data, said) {
			t.Errorf("%s produces for %v alone and its row does not say %q", name, countries, said)
		}
	}
	if scoped == 0 {
		t.Fatal("no generator covers some countries only, so this guard passed without " +
			"looking at anything. It has to be rewritten rather than removed")
	}
}

// 🔴 A column saying no without saying why would send somebody to look for a
// mistake in their own recipe. Measured on this build: three pairs cannot be
// produced, and each of them is a list this build does not carry rather than
// anything the reader did.
func TestHelpSaysWhereAGeneratorCannotBeProducedAndWhy(t *testing.T) {
	got := run(commandHelp, subjectGenerators)
	if got.code != 0 {
		t.Fatalf("the run ended with %d and said %q", got.code, got.log)
	}

	source := values.New("", "guard")
	refused := 0
	for _, name := range source.Names() {
		for _, locale := range values.Locales() {
			fine, why := source.CanProduce(name, locale)
			if fine {
				continue
			}
			refused++
			if !strings.Contains(got.data, name+" in "+locale+": "+why) {
				t.Errorf("%s cannot be produced in %s and the list does not say why: %q",
					name, locale, why)
			}
		}
	}
	if refused == 0 {
		t.Fatal("no generator is refused in any locale this build carries, so this guard " +
			"passed without looking at anything. It has to be rewritten rather than removed")
	}
}
