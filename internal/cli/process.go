package cli

import (
	"os"
)

// FromProcess builds the environment of a real invocation.
//
// It is the only place in the package that touches the process it runs in.
// Everything below Run receives what it needs, which is what lets the command
// line be tested without a terminal and without a subprocess.
func FromProcess(args []string, stdout, stderr *os.File) Environment {
	return Environment{
		Args:             args,
		Stdout:           stdout,
		Stderr:           stderr,
		StderrIsTerminal: isTerminal(stderr),
		NoColorRequested: os.Getenv("NO_COLOR") != "",
	}
}

// isTerminal reports whether f is attached to a terminal.
//
// The check asks the file itself rather than the environment. Looking for
// variables such as CI is a list of systems that is never complete, and some
// of those systems pretend to be a terminal anyway, which is why --no-color
// and NO_COLOR exist next to this.
//
// A character device is the portable signal here: a console handle on Windows
// and a tty on the rest. A pipe, a file and a closed handle are all not one.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
