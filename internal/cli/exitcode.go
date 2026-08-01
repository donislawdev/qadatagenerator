package cli

import (
	"fmt"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// exitCode is the number a pipeline reads. The numbers are a public contract:
// somebody else's CI branches on them, so adding or moving one breaks builds
// that nobody here can see.
type exitCode int

const (
	codeOK          exitCode = 0
	codeRuntime     exitCode = 1
	codeUsage       exitCode = 2
	codeConfig      exitCode = 3
	codeExpectation exitCode = 4
	codeData        exitCode = 5
	codeIO          exitCode = 6
	codeInterrupted exitCode = 130
	codeTerminated  exitCode = 143
)

type exitEntry struct {
	code    exitCode
	name    string
	meaning string
	// kind is the failure this code reports. The success row carries
	// fault.Unknown, because no failure maps to it.
	kind fault.Kind
}

// exitTable is the single source for the exit codes. The help text prints it,
// the mapping below reads it, and nothing else holds a copy. A second copy is
// how a table and the behaviour beside it drift apart.
var exitTable = []exitEntry{
	{codeOK, "OK", "the run succeeded", fault.Unknown},
	{codeRuntime, "RUNTIME", "a tool failure that belongs nowhere else", fault.Runtime},
	{codeUsage, "USAGE", "wrong invocation syntax or an unknown flag", fault.Usage},
	{codeConfig, "CONFIG", "the recipe or the corruption policy is not valid", fault.Config},
	{codeExpectation, "EXPECTATION", "data was produced, but the application under test did not behave as the manifest says", fault.Expectation},
	{codeData, "DATA", "the recipe is valid, but it cannot be satisfied", fault.Data},
	{codeIO, "IO", "a read or a write failed", fault.IO},
	{codeInterrupted, "INTERRUPTED", "stopped by the user", fault.Interrupted},
	{codeTerminated, "TERMINATED", "stopped by the system", fault.Terminated},
}

// codeFor turns a failure kind into the number the run ends with.
//
// The second result is false when the kind has no code, which happens only for
// fault.Unknown, meaning a failure arrived without ever setting its kind. That
// is a defect in this program and the caller has to say so rather than pick a
// number.
func codeFor(kind fault.Kind) (exitCode, bool) {
	if kind == fault.Unknown {
		return codeRuntime, false
	}
	for _, entry := range exitTable {
		if entry.kind == kind {
			return entry.code, true
		}
	}
	return codeRuntime, false
}

// renderExitCodes formats the table for the help text. The help is the only
// place a user can learn the codes without reading the source.
func renderExitCodes() string {
	var b strings.Builder
	for _, entry := range exitTable {
		fmt.Fprintf(&b, "  %3d  %-12s %s\n", entry.code, entry.name, entry.meaning)
	}
	return strings.TrimRight(b.String(), "\n")
}
