// Command qadg produces test data together with a manifest that says how the
// system under test should react to it.
//
// This file stays this short on purpose. Everything worth testing lives in
// internal/cli, which returns an exit code instead of ending the process, so
// every path through the command line can be exercised without starting one.
package main

import (
	"os"

	"github.com/donislawdev/qadatagenerator/internal/cli"
)

// There is no signal handler yet, and that is deliberate rather than
// forgotten. The codes 130 and 143 are already in the table, and a shell
// reports them on its own for a process that does not catch the signal. A
// handler earns its place with the first command that writes a file, because
// then there is a temporary file to remove before leaving.
func main() {
	os.Exit(cli.Run(cli.FromProcess(os.Args[1:], os.Stdout, os.Stderr)))
}
