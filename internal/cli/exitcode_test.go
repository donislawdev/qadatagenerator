package cli

import (
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// The numbers below are quoted from the contract on purpose. Writing them out
// a second time is the point: if somebody changes the table, this test asks
// whether the change was meant, because every one of these numbers already
// lives in somebody else's pipeline.
func TestTableMatchesTheContract(t *testing.T) {
	expected := []struct {
		code int
		name string
	}{
		{0, "OK"},
		{1, "RUNTIME"},
		{2, "USAGE"},
		{3, "CONFIG"},
		{4, "EXPECTATION"},
		{5, "DATA"},
		{6, "IO"},
		{130, "INTERRUPTED"},
		{143, "TERMINATED"},
	}
	if len(exitTable) != len(expected) {
		t.Fatalf("the table has %d rows and the contract has %d", len(exitTable), len(expected))
	}
	for i, want := range expected {
		if int(exitTable[i].code) != want.code || exitTable[i].name != want.name {
			t.Errorf("row %d is %d %s, expected %d %s", i, exitTable[i].code, exitTable[i].name, want.code, want.name)
		}
		if exitTable[i].meaning == "" {
			t.Errorf("row %d has no meaning, and the help text prints it", i)
		}
	}
}

func TestEveryKindHasACode(t *testing.T) {
	kinds := []fault.Kind{
		fault.Runtime, fault.Usage, fault.Config, fault.Expectation,
		fault.Data, fault.IO, fault.Interrupted, fault.Terminated,
	}
	used := make(map[exitCode]fault.Kind, len(kinds))
	for _, kind := range kinds {
		code, known := codeFor(kind)
		if !known {
			t.Errorf("kind %v has no exit code, so a run that fails this way would end with a guess", kind)
			continue
		}
		if other, taken := used[code]; taken {
			t.Errorf("kinds %v and %v both map to %d", other, kind, code)
		}
		used[code] = kind
	}
}

func TestUnknownKindHasNoCodeOfItsOwn(t *testing.T) {
	code, known := codeFor(fault.Unknown)
	if known {
		t.Fatal("a failure that never set its kind must not look like it has a code")
	}
	if code != codeRuntime {
		t.Fatalf("the fallback is the code for a tool failure, got %d", code)
	}
}

func TestRenderedTableCarriesEveryRow(t *testing.T) {
	rendered := renderExitCodes()
	for _, entry := range exitTable {
		if !strings.Contains(rendered, entry.name) {
			t.Errorf("the help text drops %s", entry.name)
		}
		if !strings.Contains(rendered, entry.meaning) {
			t.Errorf("the help text drops the meaning of %s", entry.name)
		}
	}
}
