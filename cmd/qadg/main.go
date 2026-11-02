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

// The signal handler is not here either, and that is the same decision as the
// rest of this file. A run that is stopped ends through the ordinary path with
// the kind of failure it had, so the codes 130 and 143 come out of the same
// table as every other one. How the process then ends, including sending the
// signal again on Linux and macOS, is Environment.Exit, where a test reaches it.
func main() {
	env := cli.FromProcess(os.Args[1:], os.Stdout, os.Stderr)
	env.Exit(cli.Run(env))
}
